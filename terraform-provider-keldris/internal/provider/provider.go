// Package provider implements the Keldris Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/MacJediWizard/terraform-provider-keldris/internal/keldris"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure KeldrisProvider satisfies various provider interfaces.
var _ provider.Provider = &KeldrisProvider{}

// KeldrisProvider defines the provider implementation.
type KeldrisProvider struct {
	version string
}

// KeldrisProviderModel describes the provider data model.
type KeldrisProviderModel struct {
	URL    types.String `tfsdk:"url"`
	APIKey types.String `tfsdk:"api_key"`
}

// New creates a new provider factory.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KeldrisProvider{
			version: version,
		}
	}
}

// Metadata returns the provider metadata.
func (p *KeldrisProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "keldris"
	resp.Version = p.version
}

// Schema returns the provider schema.
func (p *KeldrisProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Keldris provider allows you to manage backup infrastructure as code. " +
			"Configure agents, repositories, schedules, and policies for your Keldris backup management system.",
		MarkdownDescription: `
The Keldris provider allows you to manage backup infrastructure as code.

## Configuration

Configure the provider with your Keldris server URL and API key:

` + "```hcl" + `
provider "keldris" {
  url     = "https://keldris.example.com"
  api_key = var.keldris_api_key
}
` + "```" + `

You can also use environment variables:

- ` + "`KELDRIS_URL`" + ` - The Keldris server URL
- ` + "`KELDRIS_API_KEY`" + ` - Your API key for authentication
`,
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "The URL of the Keldris server. Can also be set via KELDRIS_URL environment variable.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "The API key for authenticating with the Keldris server. Can also be set via KELDRIS_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure configures the provider.
func (p *KeldrisProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Keldris provider")

	var config KeldrisProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check for unknown values
	if config.URL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Unknown Keldris URL",
			"The provider cannot create the Keldris API client as there is an unknown configuration value for the Keldris URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the KELDRIS_URL environment variable.",
		)
	}

	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Keldris API Key",
			"The provider cannot create the Keldris API client as there is an unknown configuration value for the Keldris API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the KELDRIS_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Get configuration from environment variables or config
	url := os.Getenv("KELDRIS_URL")
	if !config.URL.IsNull() {
		url = config.URL.ValueString()
	}

	apiKey := os.Getenv("KELDRIS_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	// Validate required configuration
	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Missing Keldris URL",
			"The provider cannot create the Keldris API client as there is a missing or empty value for the Keldris URL. "+
				"Set the url value in the provider configuration or use the KELDRIS_URL environment variable.",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Keldris API Key",
			"The provider cannot create the Keldris API client as there is a missing or empty value for the Keldris API key. "+
				"Set the api_key value in the provider configuration or use the KELDRIS_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Keldris client", map[string]interface{}{
		"keldris_url": url,
	})

	// Create the Keldris client
	client := keldris.NewClient(url, apiKey)

	// Make the client available to resources and data sources
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured Keldris provider", map[string]interface{}{
		"keldris_url": url,
	})
}

// Resources returns the provider resources.
func (p *KeldrisProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAgentResource,
		NewRepositoryResource,
		NewScheduleResource,
		NewPolicyResource,
	}
}

// DataSources returns the provider data sources.
func (p *KeldrisProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAgentsDataSource,
		NewRepositoriesDataSource,
	}
}
