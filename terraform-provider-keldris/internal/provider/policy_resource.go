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

// Ensure PolicyResource satisfies various resource interfaces.
var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithImportState = &PolicyResource{}

// PolicyResource defines the resource implementation.
type PolicyResource struct {
	client *keldris.Client
}

// PolicyResourceModel describes the resource data model.
type PolicyResourceModel struct {
	ID               types.String          `tfsdk:"id"`
	Name             types.String          `tfsdk:"name"`
	Description      types.String          `tfsdk:"description"`
	CronExpression   types.String          `tfsdk:"cron_expression"`
	Paths            types.List            `tfsdk:"paths"`
	Excludes         types.List            `tfsdk:"excludes"`
	RetentionPolicy  *RetentionPolicyModel `tfsdk:"retention_policy"`
	BandwidthLimitKB types.Int64           `tfsdk:"bandwidth_limit_kb"`
	BackupWindow     *BackupWindowModel    `tfsdk:"backup_window"`
	ExcludedHours    types.List            `tfsdk:"excluded_hours"`
}

// NewPolicyResource creates a new policy resource.
func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

// Metadata returns the resource type name.
func (r *PolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema returns the resource schema.
func (r *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Keldris backup policy.",
		MarkdownDescription: `
Manages a Keldris backup policy.

A policy is a template that can be applied to multiple agents to create schedules
with consistent settings. This is useful for standardizing backup configurations
across your infrastructure.

## Example Usage

` + "```hcl" + `
resource "keldris_policy" "standard" {
  name        = "Standard Daily Backup"
  description = "Standard backup policy for production servers"

  cron_expression = "0 2 * * *"

  paths = [
    "/home",
    "/var/www",
    "/etc"
  ]

  excludes = [
    "*.tmp",
    "*.log",
    "*.cache",
    "node_modules",
    ".git"
  ]

  retention_policy {
    keep_last    = 5
    keep_daily   = 7
    keep_weekly  = 4
    keep_monthly = 12
    keep_yearly  = 2
  }

  backup_window {
    start = "00:00"
    end   = "06:00"
  }

  bandwidth_limit_kb = 10240  # 10 MB/s
}
` + "```" + `

## Applying Policies

After creating a policy, you can apply it to agents using the Keldris API
or web interface to create schedules based on the policy template.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the policy.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the policy.",
				Optional:    true,
			},
			"cron_expression": schema.StringAttribute{
				Description: "Default cron expression for schedules created from this policy.",
				Optional:    true,
			},
			"paths": schema.ListAttribute{
				Description: "Default paths to back up.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"excludes": schema.ListAttribute{
				Description: "Default patterns to exclude from backup.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"bandwidth_limit_kb": schema.Int64Attribute{
				Description: "Default upload bandwidth limit in KB/s.",
				Optional:    true,
			},
			"excluded_hours": schema.ListAttribute{
				Description: "Default hours (0-23) when backups should not run.",
				Optional:    true,
				ElementType: types.Int64Type,
			},
		},
		Blocks: map[string]schema.Block{
			"retention_policy": schema.SingleNestedBlock{
				Description: "Default backup retention policy.",
				Attributes: map[string]schema.Attribute{
					"keep_last": schema.Int64Attribute{
						Description: "Keep the last N snapshots.",
						Optional:    true,
					},
					"keep_hourly": schema.Int64Attribute{
						Description: "Keep N hourly snapshots.",
						Optional:    true,
					},
					"keep_daily": schema.Int64Attribute{
						Description: "Keep N daily snapshots.",
						Optional:    true,
					},
					"keep_weekly": schema.Int64Attribute{
						Description: "Keep N weekly snapshots.",
						Optional:    true,
					},
					"keep_monthly": schema.Int64Attribute{
						Description: "Keep N monthly snapshots.",
						Optional:    true,
					},
					"keep_yearly": schema.Int64Attribute{
						Description: "Keep N yearly snapshots.",
						Optional:    true,
					},
				},
			},
			"backup_window": schema.SingleNestedBlock{
				Description: "Default time window when backups are allowed.",
				Attributes: map[string]schema.Attribute{
					"start": schema.StringAttribute{
						Description: "Start time in HH:MM format.",
						Optional:    true,
					},
					"end": schema.StringAttribute{
						Description: "End time in HH:MM format.",
						Optional:    true,
					},
				},
			},
		},
	}
}

// Configure sets up the resource with the provider client.
func (r *PolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating policy", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Convert paths list
	var paths []string
	if !data.Paths.IsNull() {
		resp.Diagnostics.Append(data.Paths.ElementsAs(ctx, &paths, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert excludes list
	var excludes []string
	if !data.Excludes.IsNull() {
		resp.Diagnostics.Append(data.Excludes.ElementsAs(ctx, &excludes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert excluded hours
	var excludedHours []int
	if !data.ExcludedHours.IsNull() {
		var hours []int64
		resp.Diagnostics.Append(data.ExcludedHours.ElementsAs(ctx, &hours, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, h := range hours {
			excludedHours = append(excludedHours, int(h))
		}
	}

	// Build request
	createReq := &keldris.CreatePolicyRequest{
		Name:           data.Name.ValueString(),
		Description:    data.Description.ValueString(),
		CronExpression: data.CronExpression.ValueString(),
		Paths:          paths,
		Excludes:       excludes,
		ExcludedHours:  excludedHours,
	}

	if !data.BandwidthLimitKB.IsNull() {
		bw := int(data.BandwidthLimitKB.ValueInt64())
		createReq.BandwidthLimitKB = &bw
	}

	// Build retention policy
	if data.RetentionPolicy != nil {
		createReq.RetentionPolicy = &keldris.RetentionPolicy{
			KeepLast:    int(data.RetentionPolicy.KeepLast.ValueInt64()),
			KeepHourly:  int(data.RetentionPolicy.KeepHourly.ValueInt64()),
			KeepDaily:   int(data.RetentionPolicy.KeepDaily.ValueInt64()),
			KeepWeekly:  int(data.RetentionPolicy.KeepWeekly.ValueInt64()),
			KeepMonthly: int(data.RetentionPolicy.KeepMonthly.ValueInt64()),
			KeepYearly:  int(data.RetentionPolicy.KeepYearly.ValueInt64()),
		}
	}

	// Build backup window
	if data.BackupWindow != nil {
		createReq.BackupWindow = &keldris.BackupWindow{
			Start: data.BackupWindow.Start.ValueString(),
			End:   data.BackupWindow.End.ValueString(),
		}
	}

	policy, err := r.client.CreatePolicy(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy: %s", err))
		return
	}

	data.ID = types.StringValue(policy.ID)

	tflog.Trace(ctx, "Created policy", map[string]interface{}{
		"id":   policy.ID,
		"name": policy.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetPolicy(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy: %s", err))
		return
	}

	data.ID = types.StringValue(policy.ID)
	data.Name = types.StringValue(policy.Name)
	data.Description = types.StringValue(policy.Description)
	data.CronExpression = types.StringValue(policy.CronExpression)

	// Convert paths
	if len(policy.Paths) > 0 {
		pathsList, diags := types.ListValueFrom(ctx, types.StringType, policy.Paths)
		resp.Diagnostics.Append(diags...)
		data.Paths = pathsList
	}

	// Convert excludes
	if len(policy.Excludes) > 0 {
		excludesList, diags := types.ListValueFrom(ctx, types.StringType, policy.Excludes)
		resp.Diagnostics.Append(diags...)
		data.Excludes = excludesList
	}

	// Convert retention policy
	if policy.RetentionPolicy != nil {
		data.RetentionPolicy = &RetentionPolicyModel{
			KeepLast:    types.Int64Value(int64(policy.RetentionPolicy.KeepLast)),
			KeepHourly:  types.Int64Value(int64(policy.RetentionPolicy.KeepHourly)),
			KeepDaily:   types.Int64Value(int64(policy.RetentionPolicy.KeepDaily)),
			KeepWeekly:  types.Int64Value(int64(policy.RetentionPolicy.KeepWeekly)),
			KeepMonthly: types.Int64Value(int64(policy.RetentionPolicy.KeepMonthly)),
			KeepYearly:  types.Int64Value(int64(policy.RetentionPolicy.KeepYearly)),
		}
	}

	// Convert backup window
	if policy.BackupWindow != nil {
		data.BackupWindow = &BackupWindowModel{
			Start: types.StringValue(policy.BackupWindow.Start),
			End:   types.StringValue(policy.BackupWindow.End),
		}
	}

	// Convert bandwidth limit
	if policy.BandwidthLimitKB != nil {
		data.BandwidthLimitKB = types.Int64Value(int64(*policy.BandwidthLimitKB))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating policy", map[string]interface{}{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})

	// Convert paths list
	var paths []string
	if !data.Paths.IsNull() {
		resp.Diagnostics.Append(data.Paths.ElementsAs(ctx, &paths, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert excludes list
	var excludes []string
	if !data.Excludes.IsNull() {
		resp.Diagnostics.Append(data.Excludes.ElementsAs(ctx, &excludes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert excluded hours
	var excludedHours []int
	if !data.ExcludedHours.IsNull() {
		var hours []int64
		resp.Diagnostics.Append(data.ExcludedHours.ElementsAs(ctx, &hours, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, h := range hours {
			excludedHours = append(excludedHours, int(h))
		}
	}

	// Build request
	updateReq := &keldris.UpdatePolicyRequest{
		Name:           data.Name.ValueString(),
		Description:    data.Description.ValueString(),
		CronExpression: data.CronExpression.ValueString(),
		Paths:          paths,
		Excludes:       excludes,
		ExcludedHours:  excludedHours,
	}

	if !data.BandwidthLimitKB.IsNull() {
		bw := int(data.BandwidthLimitKB.ValueInt64())
		updateReq.BandwidthLimitKB = &bw
	}

	// Build retention policy
	if data.RetentionPolicy != nil {
		updateReq.RetentionPolicy = &keldris.RetentionPolicy{
			KeepLast:    int(data.RetentionPolicy.KeepLast.ValueInt64()),
			KeepHourly:  int(data.RetentionPolicy.KeepHourly.ValueInt64()),
			KeepDaily:   int(data.RetentionPolicy.KeepDaily.ValueInt64()),
			KeepWeekly:  int(data.RetentionPolicy.KeepWeekly.ValueInt64()),
			KeepMonthly: int(data.RetentionPolicy.KeepMonthly.ValueInt64()),
			KeepYearly:  int(data.RetentionPolicy.KeepYearly.ValueInt64()),
		}
	}

	// Build backup window
	if data.BackupWindow != nil {
		updateReq.BackupWindow = &keldris.BackupWindow{
			Start: data.BackupWindow.Start.ValueString(),
			End:   data.BackupWindow.End.ValueString(),
		}
	}

	_, err := r.client.UpdatePolicy(ctx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update policy: %s", err))
		return
	}

	tflog.Trace(ctx, "Updated policy", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting policy", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeletePolicy(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete policy: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted policy", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	policy, err := r.client.GetPolicy(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import policy: %s", err))
		return
	}

	data := PolicyResourceModel{
		ID:             types.StringValue(policy.ID),
		Name:           types.StringValue(policy.Name),
		Description:    types.StringValue(policy.Description),
		CronExpression: types.StringValue(policy.CronExpression),
	}

	// Convert paths
	if len(policy.Paths) > 0 {
		pathsList, diags := types.ListValueFrom(ctx, types.StringType, policy.Paths)
		resp.Diagnostics.Append(diags...)
		data.Paths = pathsList
	}

	// Convert excludes
	if len(policy.Excludes) > 0 {
		excludesList, diags := types.ListValueFrom(ctx, types.StringType, policy.Excludes)
		resp.Diagnostics.Append(diags...)
		data.Excludes = excludesList
	}

	// Convert retention policy
	if policy.RetentionPolicy != nil {
		data.RetentionPolicy = &RetentionPolicyModel{
			KeepLast:    types.Int64Value(int64(policy.RetentionPolicy.KeepLast)),
			KeepHourly:  types.Int64Value(int64(policy.RetentionPolicy.KeepHourly)),
			KeepDaily:   types.Int64Value(int64(policy.RetentionPolicy.KeepDaily)),
			KeepWeekly:  types.Int64Value(int64(policy.RetentionPolicy.KeepWeekly)),
			KeepMonthly: types.Int64Value(int64(policy.RetentionPolicy.KeepMonthly)),
			KeepYearly:  types.Int64Value(int64(policy.RetentionPolicy.KeepYearly)),
		}
	}

	// Convert backup window
	if policy.BackupWindow != nil {
		data.BackupWindow = &BackupWindowModel{
			Start: types.StringValue(policy.BackupWindow.Start),
			End:   types.StringValue(policy.BackupWindow.End),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
