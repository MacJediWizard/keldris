package provider

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/terraform-provider-keldris/internal/keldris"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure ScheduleResource satisfies various resource interfaces.
var _ resource.Resource = &ScheduleResource{}
var _ resource.ResourceWithImportState = &ScheduleResource{}

// ScheduleResource defines the resource implementation.
type ScheduleResource struct {
	client *keldris.Client
}

// RetentionPolicyModel describes the retention policy data model.
type RetentionPolicyModel struct {
	KeepLast    types.Int64 `tfsdk:"keep_last"`
	KeepHourly  types.Int64 `tfsdk:"keep_hourly"`
	KeepDaily   types.Int64 `tfsdk:"keep_daily"`
	KeepWeekly  types.Int64 `tfsdk:"keep_weekly"`
	KeepMonthly types.Int64 `tfsdk:"keep_monthly"`
	KeepYearly  types.Int64 `tfsdk:"keep_yearly"`
}

// BackupWindowModel describes the backup window data model.
type BackupWindowModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

// ScheduleRepositoryModel describes a repository association.
type ScheduleRepositoryModel struct {
	RepositoryID types.String `tfsdk:"repository_id"`
	Priority     types.Int64  `tfsdk:"priority"`
	Enabled      types.Bool   `tfsdk:"enabled"`
}

// ScheduleResourceModel describes the resource data model.
type ScheduleResourceModel struct {
	ID               types.String               `tfsdk:"id"`
	AgentID          types.String               `tfsdk:"agent_id"`
	Name             types.String               `tfsdk:"name"`
	CronExpression   types.String               `tfsdk:"cron_expression"`
	Paths            types.List                 `tfsdk:"paths"`
	Excludes         types.List                 `tfsdk:"excludes"`
	RetentionPolicy  *RetentionPolicyModel      `tfsdk:"retention_policy"`
	BandwidthLimitKB types.Int64                `tfsdk:"bandwidth_limit_kb"`
	BackupWindow     *BackupWindowModel         `tfsdk:"backup_window"`
	ExcludedHours    types.List                 `tfsdk:"excluded_hours"`
	CompressionLevel types.String               `tfsdk:"compression_level"`
	Enabled          types.Bool                 `tfsdk:"enabled"`
	Repositories     []ScheduleRepositoryModel  `tfsdk:"repositories"`
}

// NewScheduleResource creates a new schedule resource.
func NewScheduleResource() resource.Resource {
	return &ScheduleResource{}
}

// Metadata returns the resource type name.
func (r *ScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

// Schema returns the resource schema.
func (r *ScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Keldris backup schedule.",
		MarkdownDescription: `
Manages a Keldris backup schedule.

A schedule defines when and what to back up from an agent to one or more repositories.

## Example Usage

` + "```hcl" + `
resource "keldris_schedule" "daily_backup" {
  name           = "Daily Home Backup"
  agent_id       = keldris_agent.web_server.id
  cron_expression = "0 2 * * *"  # Daily at 2 AM

  paths = [
    "/home",
    "/var/www"
  ]

  excludes = [
    "*.tmp",
    "*.log",
    "node_modules"
  ]

  retention_policy {
    keep_last    = 5
    keep_daily   = 7
    keep_weekly  = 4
    keep_monthly = 6
  }

  repositories {
    repository_id = keldris_repository.s3_backup.id
    priority      = 0
    enabled       = true
  }

  enabled = true
}
` + "```" + `

## Cron Expression

The cron expression follows standard cron format: ` + "`minute hour day month weekday`" + `

Examples:
- ` + "`0 2 * * *`" + ` - Daily at 2:00 AM
- ` + "`0 */6 * * *`" + ` - Every 6 hours
- ` + "`0 0 * * 0`" + ` - Weekly on Sunday at midnight
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the schedule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_id": schema.StringAttribute{
				Description: "The ID of the agent this schedule belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the schedule.",
				Required:    true,
			},
			"cron_expression": schema.StringAttribute{
				Description: "Cron expression defining when backups run.",
				Required:    true,
			},
			"paths": schema.ListAttribute{
				Description: "List of paths to back up.",
				Required:    true,
				ElementType: types.StringType,
			},
			"excludes": schema.ListAttribute{
				Description: "List of patterns to exclude from backup.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"bandwidth_limit_kb": schema.Int64Attribute{
				Description: "Upload bandwidth limit in KB/s.",
				Optional:    true,
			},
			"excluded_hours": schema.ListAttribute{
				Description: "Hours (0-23) when backups should not run.",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"compression_level": schema.StringAttribute{
				Description: "Compression level (off, auto, max).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("off", "auto", "max"),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the schedule is enabled. Default: true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"retention_policy": schema.SingleNestedBlock{
				Description: "Backup retention policy.",
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
				Description: "Time window when backups are allowed.",
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
			"repositories": schema.ListNestedBlock{
				Description: "Repository associations for this schedule.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"repository_id": schema.StringAttribute{
							Description: "The ID of the repository.",
							Required:    true,
						},
						"priority": schema.Int64Attribute{
							Description: "Priority for multi-repository schedules (0 = primary).",
							Optional:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether this repository association is enabled.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
					},
				},
			},
		},
	}
}

// Configure sets up the resource with the provider client.
func (r *ScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating schedule", map[string]interface{}{
		"name":     data.Name.ValueString(),
		"agent_id": data.AgentID.ValueString(),
	})

	// Convert paths list
	var paths []string
	resp.Diagnostics.Append(data.Paths.ElementsAs(ctx, &paths, false)...)
	if resp.Diagnostics.HasError() {
		return
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
	createReq := &keldris.CreateScheduleRequest{
		AgentID:        data.AgentID.ValueString(),
		Name:           data.Name.ValueString(),
		CronExpression: data.CronExpression.ValueString(),
		Paths:          paths,
		Excludes:       excludes,
		ExcludedHours:  excludedHours,
	}

	if !data.BandwidthLimitKB.IsNull() {
		bw := int(data.BandwidthLimitKB.ValueInt64())
		createReq.BandwidthLimitKB = &bw
	}

	if !data.CompressionLevel.IsNull() {
		cl := data.CompressionLevel.ValueString()
		createReq.CompressionLevel = &cl
	}

	enabled := data.Enabled.ValueBool()
	createReq.Enabled = &enabled

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

	// Build repositories
	for _, repo := range data.Repositories {
		createReq.Repositories = append(createReq.Repositories, keldris.ScheduleRepository{
			RepositoryID: repo.RepositoryID.ValueString(),
			Priority:     int(repo.Priority.ValueInt64()),
			Enabled:      repo.Enabled.ValueBool(),
		})
	}

	schedule, err := r.client.CreateSchedule(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create schedule: %s", err))
		return
	}

	data.ID = types.StringValue(schedule.ID)

	tflog.Trace(ctx, "Created schedule", map[string]interface{}{
		"id":   schedule.ID,
		"name": schedule.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *ScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule, err := r.client.GetSchedule(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read schedule: %s", err))
		return
	}

	data.ID = types.StringValue(schedule.ID)
	data.AgentID = types.StringValue(schedule.AgentID)
	data.Name = types.StringValue(schedule.Name)
	data.CronExpression = types.StringValue(schedule.CronExpression)
	data.Enabled = types.BoolValue(schedule.Enabled)

	// Convert paths
	pathsValues := make([]types.String, len(schedule.Paths))
	for i, p := range schedule.Paths {
		pathsValues[i] = types.StringValue(p)
	}
	pathsList, diags := types.ListValueFrom(ctx, types.StringType, schedule.Paths)
	resp.Diagnostics.Append(diags...)
	data.Paths = pathsList

	// Convert excludes
	if len(schedule.Excludes) > 0 {
		excludesList, diags := types.ListValueFrom(ctx, types.StringType, schedule.Excludes)
		resp.Diagnostics.Append(diags...)
		data.Excludes = excludesList
	}

	// Convert retention policy
	if schedule.RetentionPolicy != nil {
		data.RetentionPolicy = &RetentionPolicyModel{
			KeepLast:    types.Int64Value(int64(schedule.RetentionPolicy.KeepLast)),
			KeepHourly:  types.Int64Value(int64(schedule.RetentionPolicy.KeepHourly)),
			KeepDaily:   types.Int64Value(int64(schedule.RetentionPolicy.KeepDaily)),
			KeepWeekly:  types.Int64Value(int64(schedule.RetentionPolicy.KeepWeekly)),
			KeepMonthly: types.Int64Value(int64(schedule.RetentionPolicy.KeepMonthly)),
			KeepYearly:  types.Int64Value(int64(schedule.RetentionPolicy.KeepYearly)),
		}
	}

	// Convert backup window
	if schedule.BackupWindow != nil {
		data.BackupWindow = &BackupWindowModel{
			Start: types.StringValue(schedule.BackupWindow.Start),
			End:   types.StringValue(schedule.BackupWindow.End),
		}
	}

	// Convert bandwidth limit
	if schedule.BandwidthLimitKB != nil {
		data.BandwidthLimitKB = types.Int64Value(int64(*schedule.BandwidthLimitKB))
	}

	// Convert compression level
	if schedule.CompressionLevel != nil {
		data.CompressionLevel = types.StringValue(*schedule.CompressionLevel)
	}

	// Convert repositories
	data.Repositories = nil
	for _, repo := range schedule.Repositories {
		data.Repositories = append(data.Repositories, ScheduleRepositoryModel{
			RepositoryID: types.StringValue(repo.RepositoryID),
			Priority:     types.Int64Value(int64(repo.Priority)),
			Enabled:      types.BoolValue(repo.Enabled),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *ScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating schedule", map[string]interface{}{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})

	// Convert paths list
	var paths []string
	resp.Diagnostics.Append(data.Paths.ElementsAs(ctx, &paths, false)...)
	if resp.Diagnostics.HasError() {
		return
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
	updateReq := &keldris.UpdateScheduleRequest{
		Name:           data.Name.ValueString(),
		CronExpression: data.CronExpression.ValueString(),
		Paths:          paths,
		Excludes:       excludes,
		ExcludedHours:  excludedHours,
	}

	if !data.BandwidthLimitKB.IsNull() {
		bw := int(data.BandwidthLimitKB.ValueInt64())
		updateReq.BandwidthLimitKB = &bw
	}

	if !data.CompressionLevel.IsNull() {
		cl := data.CompressionLevel.ValueString()
		updateReq.CompressionLevel = &cl
	}

	enabled := data.Enabled.ValueBool()
	updateReq.Enabled = &enabled

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

	// Build repositories
	for _, repo := range data.Repositories {
		updateReq.Repositories = append(updateReq.Repositories, keldris.ScheduleRepository{
			RepositoryID: repo.RepositoryID.ValueString(),
			Priority:     int(repo.Priority.ValueInt64()),
			Enabled:      repo.Enabled.ValueBool(),
		})
	}

	_, err := r.client.UpdateSchedule(ctx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update schedule: %s", err))
		return
	}

	tflog.Trace(ctx, "Updated schedule", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *ScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting schedule", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeleteSchedule(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete schedule: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted schedule", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *ScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	schedule, err := r.client.GetSchedule(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import schedule: %s", err))
		return
	}

	// Convert paths
	pathsList, diags := types.ListValueFrom(ctx, types.StringType, schedule.Paths)
	resp.Diagnostics.Append(diags...)

	// Convert excludes
	var excludesList types.List
	if len(schedule.Excludes) > 0 {
		excludesList, diags = types.ListValueFrom(ctx, types.StringType, schedule.Excludes)
		resp.Diagnostics.Append(diags...)
	}

	data := ScheduleResourceModel{
		ID:             types.StringValue(schedule.ID),
		AgentID:        types.StringValue(schedule.AgentID),
		Name:           types.StringValue(schedule.Name),
		CronExpression: types.StringValue(schedule.CronExpression),
		Paths:          pathsList,
		Excludes:       excludesList,
		Enabled:        types.BoolValue(schedule.Enabled),
	}

	// Convert retention policy
	if schedule.RetentionPolicy != nil {
		data.RetentionPolicy = &RetentionPolicyModel{
			KeepLast:    types.Int64Value(int64(schedule.RetentionPolicy.KeepLast)),
			KeepHourly:  types.Int64Value(int64(schedule.RetentionPolicy.KeepHourly)),
			KeepDaily:   types.Int64Value(int64(schedule.RetentionPolicy.KeepDaily)),
			KeepWeekly:  types.Int64Value(int64(schedule.RetentionPolicy.KeepWeekly)),
			KeepMonthly: types.Int64Value(int64(schedule.RetentionPolicy.KeepMonthly)),
			KeepYearly:  types.Int64Value(int64(schedule.RetentionPolicy.KeepYearly)),
		}
	}

	// Convert repositories
	for _, repo := range schedule.Repositories {
		data.Repositories = append(data.Repositories, ScheduleRepositoryModel{
			RepositoryID: types.StringValue(repo.RepositoryID),
			Priority:     types.Int64Value(int64(repo.Priority)),
			Enabled:      types.BoolValue(repo.Enabled),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
