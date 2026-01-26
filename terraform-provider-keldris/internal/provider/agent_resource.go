package provider

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/terraform-provider-keldris/internal/keldris"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure AgentResource satisfies various resource interfaces.
var _ resource.Resource = &AgentResource{}
var _ resource.ResourceWithImportState = &AgentResource{}

// AgentResource defines the resource implementation.
type AgentResource struct {
	client *keldris.Client
}

// AgentResourceModel describes the resource data model.
type AgentResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Hostname types.String `tfsdk:"hostname"`
	Status   types.String `tfsdk:"status"`
	APIKey   types.String `tfsdk:"api_key"`
}

// NewAgentResource creates a new agent resource.
func NewAgentResource() resource.Resource {
	return &AgentResource{}
}

// Metadata returns the resource type name.
func (r *AgentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

// Schema returns the resource schema.
func (r *AgentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Keldris backup agent.",
		MarkdownDescription: `
Manages a Keldris backup agent.

An agent represents a host that runs backup operations. When you create an agent,
an API key is generated that the agent uses for authentication.

## Example Usage

` + "```hcl" + `
resource "keldris_agent" "web_server" {
  hostname = "web-server-01"
}

output "agent_api_key" {
  value     = keldris_agent.web_server.api_key
  sensitive = true
}
` + "```" + `

~> **Important:** The API key is only available at creation time and cannot be retrieved later.
Store it securely, such as in a secrets manager.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the agent.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Description: "The hostname of the agent.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the agent (pending, active, offline, disabled).",
				Computed:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "The API key for the agent. Only available at creation time.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure sets up the resource with the provider client.
func (r *AgentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*keldris.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *keldris.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource.
func (r *AgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating agent", map[string]interface{}{
		"hostname": data.Hostname.ValueString(),
	})

	agent, err := r.client.CreateAgent(ctx, &keldris.CreateAgentRequest{
		Hostname: data.Hostname.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create agent: %s", err))
		return
	}

	data.ID = types.StringValue(agent.ID)
	data.Hostname = types.StringValue(agent.Hostname)
	data.APIKey = types.StringValue(agent.APIKey)
	data.Status = types.StringValue("pending")

	tflog.Trace(ctx, "Created agent", map[string]interface{}{
		"id":       agent.ID,
		"hostname": agent.Hostname,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *AgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	agent, err := r.client.GetAgent(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent: %s", err))
		return
	}

	data.ID = types.StringValue(agent.ID)
	data.Hostname = types.StringValue(agent.Hostname)
	data.Status = types.StringValue(agent.Status)
	// API key is not returned on read, keep existing value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *AgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Agents don't support updates other than hostname which requires replacement
	// Just re-read the current state
	agent, err := r.client.GetAgent(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent: %s", err))
		return
	}

	data.Status = types.StringValue(agent.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *AgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting agent", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeleteAgent(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete agent: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted agent", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *AgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	agent, err := r.client.GetAgent(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import agent: %s", err))
		return
	}

	data := AgentResourceModel{
		ID:       types.StringValue(agent.ID),
		Hostname: types.StringValue(agent.Hostname),
		Status:   types.StringValue(agent.Status),
		APIKey:   types.StringNull(), // API key not available on import
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
