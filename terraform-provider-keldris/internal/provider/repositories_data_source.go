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

// Ensure RepositoriesDataSource satisfies various datasource interfaces.
var _ datasource.DataSource = &RepositoriesDataSource{}

// RepositoriesDataSource defines the data source implementation.
type RepositoriesDataSource struct {
	client *keldris.Client
}

// RepositoryDataModel describes a single repository in the data source.
type RepositoryDataModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

// RepositoriesDataSourceModel describes the data source data model.
type RepositoriesDataSourceModel struct {
	Repositories []RepositoryDataModel `tfsdk:"repositories"`
}

// NewRepositoriesDataSource creates a new repositories data source.
func NewRepositoriesDataSource() datasource.DataSource {
	return &RepositoriesDataSource{}
}

// Metadata returns the data source type name.
func (d *RepositoriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repositories"
}

// Schema returns the data source schema.
func (d *RepositoriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of Keldris backup repositories.",
		MarkdownDescription: `
Fetches the list of Keldris backup repositories in your organization.

## Example Usage

` + "```hcl" + `
data "keldris_repositories" "all" {}

output "repository_count" {
  value = length(data.keldris_repositories.all.repositories)
}

output "s3_repositories" {
  value = [for r in data.keldris_repositories.all.repositories : r.name if r.type == "s3"]
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"repositories": schema.ListNestedAttribute{
				Description: "List of repositories.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the repository.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the repository.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of repository (s3, b2, sftp, local, rest, dropbox).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the data source with the provider client.
func (d *RepositoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *RepositoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RepositoriesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading repositories")

	repos, err := d.client.ListRepositories(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list repositories: %s", err))
		return
	}

	for _, repo := range repos {
		data.Repositories = append(data.Repositories, RepositoryDataModel{
			ID:   types.StringValue(repo.ID),
			Name: types.StringValue(repo.Name),
			Type: types.StringValue(repo.Type),
		})
	}

	tflog.Trace(ctx, "Read repositories", map[string]interface{}{
		"count": len(data.Repositories),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
