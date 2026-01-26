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

// Ensure RepositoryResource satisfies various resource interfaces.
var _ resource.Resource = &RepositoryResource{}
var _ resource.ResourceWithImportState = &RepositoryResource{}

// RepositoryResource defines the resource implementation.
type RepositoryResource struct {
	client *keldris.Client
}

// RepositoryResourceModel describes the resource data model.
type RepositoryResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	EscrowEnabled types.Bool   `tfsdk:"escrow_enabled"`
	Password      types.String `tfsdk:"password"`

	// S3 config
	S3Bucket          types.String `tfsdk:"s3_bucket"`
	S3Region          types.String `tfsdk:"s3_region"`
	S3Endpoint        types.String `tfsdk:"s3_endpoint"`
	S3AccessKeyID     types.String `tfsdk:"s3_access_key_id"`
	S3SecretAccessKey types.String `tfsdk:"s3_secret_access_key"`
	S3Path            types.String `tfsdk:"s3_path"`

	// Local config
	LocalPath types.String `tfsdk:"local_path"`

	// SFTP config
	SFTPHost       types.String `tfsdk:"sftp_host"`
	SFTPPort       types.Int64  `tfsdk:"sftp_port"`
	SFTPUser       types.String `tfsdk:"sftp_user"`
	SFTPPassword   types.String `tfsdk:"sftp_password"`
	SFTPPrivateKey types.String `tfsdk:"sftp_private_key"`
	SFTPPath       types.String `tfsdk:"sftp_path"`

	// B2 config
	B2AccountID     types.String `tfsdk:"b2_account_id"`
	B2ApplicationKey types.String `tfsdk:"b2_application_key"`
	B2Bucket        types.String `tfsdk:"b2_bucket"`
	B2Path          types.String `tfsdk:"b2_path"`

	// REST config
	RESTURL      types.String `tfsdk:"rest_url"`
	RESTUsername types.String `tfsdk:"rest_username"`
	RESTPassword types.String `tfsdk:"rest_password"`
}

// NewRepositoryResource creates a new repository resource.
func NewRepositoryResource() resource.Resource {
	return &RepositoryResource{}
}

// Metadata returns the resource type name.
func (r *RepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

// Schema returns the resource schema.
func (r *RepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Keldris backup repository.",
		MarkdownDescription: `
Manages a Keldris backup repository.

A repository is a storage destination for backups. Keldris supports multiple
backend types including S3, B2, SFTP, local filesystem, and REST server.

## Example Usage

### S3 Repository

` + "```hcl" + `
resource "keldris_repository" "s3_backup" {
  name           = "production-backups"
  type           = "s3"
  escrow_enabled = true

  s3_bucket            = "my-backup-bucket"
  s3_region            = "us-west-2"
  s3_access_key_id     = var.aws_access_key
  s3_secret_access_key = var.aws_secret_key
  s3_path              = "backups/"
}
` + "```" + `

### Local Repository

` + "```hcl" + `
resource "keldris_repository" "local_backup" {
  name = "local-backups"
  type = "local"

  local_path = "/mnt/backups"
}
` + "```" + `

### SFTP Repository

` + "```hcl" + `
resource "keldris_repository" "sftp_backup" {
  name = "remote-backups"
  type = "sftp"

  sftp_host     = "backup.example.com"
  sftp_port     = 22
  sftp_user     = "backup"
  sftp_password = var.sftp_password
  sftp_path     = "/backups"
}
` + "```" + `

~> **Important:** The repository password is only available at creation time.
Store it securely as it's needed to access the encrypted backups.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the repository.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of repository (s3, b2, sftp, local, rest, dropbox).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("s3", "b2", "sftp", "local", "rest", "dropbox"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"escrow_enabled": schema.BoolAttribute{
				Description: "Enable key escrow for password recovery. Default: false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"password": schema.StringAttribute{
				Description: "The repository encryption password. Only available at creation time.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// S3 config
			"s3_bucket": schema.StringAttribute{
				Description: "S3 bucket name.",
				Optional:    true,
			},
			"s3_region": schema.StringAttribute{
				Description: "S3 region.",
				Optional:    true,
			},
			"s3_endpoint": schema.StringAttribute{
				Description: "S3 endpoint URL for S3-compatible storage.",
				Optional:    true,
			},
			"s3_access_key_id": schema.StringAttribute{
				Description: "S3 access key ID.",
				Optional:    true,
				Sensitive:   true,
			},
			"s3_secret_access_key": schema.StringAttribute{
				Description: "S3 secret access key.",
				Optional:    true,
				Sensitive:   true,
			},
			"s3_path": schema.StringAttribute{
				Description: "Path prefix within the S3 bucket.",
				Optional:    true,
			},

			// Local config
			"local_path": schema.StringAttribute{
				Description: "Local filesystem path for backup storage.",
				Optional:    true,
			},

			// SFTP config
			"sftp_host": schema.StringAttribute{
				Description: "SFTP server hostname.",
				Optional:    true,
			},
			"sftp_port": schema.Int64Attribute{
				Description: "SFTP server port. Default: 22.",
				Optional:    true,
			},
			"sftp_user": schema.StringAttribute{
				Description: "SFTP username.",
				Optional:    true,
			},
			"sftp_password": schema.StringAttribute{
				Description: "SFTP password.",
				Optional:    true,
				Sensitive:   true,
			},
			"sftp_private_key": schema.StringAttribute{
				Description: "SFTP private key (PEM format).",
				Optional:    true,
				Sensitive:   true,
			},
			"sftp_path": schema.StringAttribute{
				Description: "Path on the SFTP server.",
				Optional:    true,
			},

			// B2 config
			"b2_account_id": schema.StringAttribute{
				Description: "Backblaze B2 account ID.",
				Optional:    true,
			},
			"b2_application_key": schema.StringAttribute{
				Description: "Backblaze B2 application key.",
				Optional:    true,
				Sensitive:   true,
			},
			"b2_bucket": schema.StringAttribute{
				Description: "Backblaze B2 bucket name.",
				Optional:    true,
			},
			"b2_path": schema.StringAttribute{
				Description: "Path prefix within the B2 bucket.",
				Optional:    true,
			},

			// REST config
			"rest_url": schema.StringAttribute{
				Description: "REST server URL.",
				Optional:    true,
			},
			"rest_username": schema.StringAttribute{
				Description: "REST server username.",
				Optional:    true,
			},
			"rest_password": schema.StringAttribute{
				Description: "REST server password.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure sets up the resource with the provider client.
func (r *RepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildConfig builds the configuration map from the resource model.
func (r *RepositoryResource) buildConfig(data *RepositoryResourceModel) map[string]interface{} {
	config := make(map[string]interface{})

	switch data.Type.ValueString() {
	case "s3":
		if !data.S3Bucket.IsNull() {
			config["bucket"] = data.S3Bucket.ValueString()
		}
		if !data.S3Region.IsNull() {
			config["region"] = data.S3Region.ValueString()
		}
		if !data.S3Endpoint.IsNull() {
			config["endpoint"] = data.S3Endpoint.ValueString()
		}
		if !data.S3AccessKeyID.IsNull() {
			config["access_key_id"] = data.S3AccessKeyID.ValueString()
		}
		if !data.S3SecretAccessKey.IsNull() {
			config["secret_access_key"] = data.S3SecretAccessKey.ValueString()
		}
		if !data.S3Path.IsNull() {
			config["path"] = data.S3Path.ValueString()
		}

	case "local":
		if !data.LocalPath.IsNull() {
			config["path"] = data.LocalPath.ValueString()
		}

	case "sftp":
		if !data.SFTPHost.IsNull() {
			config["host"] = data.SFTPHost.ValueString()
		}
		if !data.SFTPPort.IsNull() {
			config["port"] = data.SFTPPort.ValueInt64()
		}
		if !data.SFTPUser.IsNull() {
			config["user"] = data.SFTPUser.ValueString()
		}
		if !data.SFTPPassword.IsNull() {
			config["password"] = data.SFTPPassword.ValueString()
		}
		if !data.SFTPPrivateKey.IsNull() {
			config["private_key"] = data.SFTPPrivateKey.ValueString()
		}
		if !data.SFTPPath.IsNull() {
			config["path"] = data.SFTPPath.ValueString()
		}

	case "b2":
		if !data.B2AccountID.IsNull() {
			config["account_id"] = data.B2AccountID.ValueString()
		}
		if !data.B2ApplicationKey.IsNull() {
			config["application_key"] = data.B2ApplicationKey.ValueString()
		}
		if !data.B2Bucket.IsNull() {
			config["bucket"] = data.B2Bucket.ValueString()
		}
		if !data.B2Path.IsNull() {
			config["path"] = data.B2Path.ValueString()
		}

	case "rest":
		if !data.RESTURL.IsNull() {
			config["url"] = data.RESTURL.ValueString()
		}
		if !data.RESTUsername.IsNull() {
			config["username"] = data.RESTUsername.ValueString()
		}
		if !data.RESTPassword.IsNull() {
			config["password"] = data.RESTPassword.ValueString()
		}
	}

	return config
}

// Create creates the resource.
func (r *RepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating repository", map[string]interface{}{
		"name": data.Name.ValueString(),
		"type": data.Type.ValueString(),
	})

	repoResp, err := r.client.CreateRepository(ctx, &keldris.CreateRepositoryRequest{
		Name:          data.Name.ValueString(),
		Type:          data.Type.ValueString(),
		Config:        r.buildConfig(&data),
		EscrowEnabled: data.EscrowEnabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create repository: %s", err))
		return
	}

	data.ID = types.StringValue(repoResp.Repository.ID)
	data.Password = types.StringValue(repoResp.Password)

	tflog.Trace(ctx, "Created repository", map[string]interface{}{
		"id":   repoResp.Repository.ID,
		"name": repoResp.Repository.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *RepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repo, err := r.client.GetRepository(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read repository: %s", err))
		return
	}

	data.ID = types.StringValue(repo.ID)
	data.Name = types.StringValue(repo.Name)
	data.Type = types.StringValue(repo.Type)
	// Password and config are not returned on read, keep existing values

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *RepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating repository", map[string]interface{}{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})

	_, err := r.client.UpdateRepository(ctx, data.ID.ValueString(), &keldris.UpdateRepositoryRequest{
		Name:   data.Name.ValueString(),
		Config: r.buildConfig(&data),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update repository: %s", err))
		return
	}

	tflog.Trace(ctx, "Updated repository", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *RepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting repository", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeleteRepository(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete repository: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted repository", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *RepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	repo, err := r.client.GetRepository(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import repository: %s", err))
		return
	}

	data := RepositoryResourceModel{
		ID:       types.StringValue(repo.ID),
		Name:     types.StringValue(repo.Name),
		Type:     types.StringValue(repo.Type),
		Password: types.StringNull(), // Password not available on import
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
