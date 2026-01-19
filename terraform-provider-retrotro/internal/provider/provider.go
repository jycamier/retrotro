package provider

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &RetrotroProvider{}

// RetrotroProvider defines the provider implementation.
type RetrotroProvider struct {
	version string
}

// RetrotroProviderModel describes the provider data model.
type RetrotroProviderModel struct {
	APIURL   types.String `tfsdk:"api_url"`
	APIToken types.String `tfsdk:"api_token"`
}

// Metadata returns the provider type name.
func (p *RetrotroProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "retrotro"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *RetrotroProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Retrotro resources like webhooks.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Description: "The base URL for the Retrotro API. Can also be set via the RETROTRO_API_URL environment variable.",
				Optional:    true,
			},
			"api_token": schema.StringAttribute{
				Description: "API token for authentication. Can also be set via the RETROTRO_API_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares a Retrotro API client for data sources and resources.
func (p *RetrotroProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config RetrotroProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values from environment
	apiURL := getEnvOrDefault("RETROTRO_API_URL", "http://localhost:8080")
	apiToken := getEnvOrDefault("RETROTRO_API_TOKEN", "")

	// Override with configuration values if provided
	if !config.APIURL.IsNull() {
		apiURL = config.APIURL.ValueString()
	}
	if !config.APIToken.IsNull() {
		apiToken = config.APIToken.ValueString()
	}

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"The provider requires an API token. Set it via the api_token attribute or RETROTRO_API_TOKEN environment variable.",
		)
		return
	}

	// Create the API client
	client := &RetrotroClient{
		BaseURL:    apiURL,
		Token:      apiToken,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Make the client available to resources and data sources
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented by the provider.
func (p *RetrotroProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWebhookResource,
	}
}

// DataSources defines the data sources implemented by the provider.
func (p *RetrotroProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewTeamDataSource,
		NewWebhookDataSource,
	}
}

// New creates a new provider factory.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RetrotroProvider{
			version: version,
		}
	}
}
