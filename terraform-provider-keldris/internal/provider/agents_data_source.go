package provider

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/terraform-provider-keldris/internal/keldris"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure AgentsDataSource satisfies various datasource interfaces.
var _ datasource.DataSource = &AgentsDataSource{}

// AgentsDataSource defines the data source implementation.
type AgentsDataSource struct {
	client *keldris.Client
}

// AgentDataModel describes a single agent in the data source.
type AgentDataModel struct {
	ID       types.String `tfsdk:"id"`
	Hostname types.String `tfsdk:"hostname"`
	Status   types.String `tfsdk:"status"`
}

// AgentsDataSourceModel describes the data source data model.
type AgentsDataSourceModel struct {
	Agents []AgentDataModel `tfsdk:"agents"`
}

// NewAgentsDataSource creates a new agents data source.
func NewAgentsDataSource() datasource.DataSource {
	return &AgentsDataSource{}
}

// Metadata returns the data source type name.
func (d *AgentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agents"
}

// Schema returns the data source schema.
func (d *AgentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of Keldris backup agents.",
		MarkdownDescription: `
Fetches the list of Keldris backup agents in your organization.

## Example Usage

` + "```hcl" + `
data "keldris_agents" "all" {}

output "agent_count" {
  value = length(data.keldris_agents.all.agents)
}

output "active_agents" {
  value = [for a in data.keldris_agents.all.agents : a.hostname if a.status == "active"]
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"agents": schema.ListNestedAttribute{
				Description: "List of agents.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the agent.",
							Computed:    true,
						},
						"hostname": schema.StringAttribute{
							Description: "The hostname of the agent.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the agent (pending, active, offline, disabled).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the data source with the provider client.
func (d *AgentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*keldris.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *keldris.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read reads the data source.
func (d *AgentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AgentsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading agents")

	agents, err := d.client.ListAgents(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list agents: %s", err))
		return
	}

	for _, agent := range agents {
		data.Agents = append(data.Agents, AgentDataModel{
			ID:       types.StringValue(agent.ID),
			Hostname: types.StringValue(agent.Hostname),
			Status:   types.StringValue(agent.Status),
		})
	}

	tflog.Trace(ctx, "Read agents", map[string]interface{}{
		"count": len(data.Agents),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
