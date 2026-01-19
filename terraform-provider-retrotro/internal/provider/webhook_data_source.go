package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var _ datasource.DataSource = &webhookDataSource{}

// NewWebhookDataSource is a helper function to simplify the provider implementation.
func NewWebhookDataSource() datasource.DataSource {
	return &webhookDataSource{}
}

// webhookDataSource is the data source implementation.
type webhookDataSource struct {
	client *RetrotroClient
}

// webhookDataSourceModel maps the data source schema data.
type webhookDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	TeamID    types.String `tfsdk:"team_id"`
	Name      types.String `tfsdk:"name"`
	URL       types.String `tfsdk:"url"`
	Events    types.List   `tfsdk:"events"`
	IsEnabled types.Bool   `tfsdk:"enabled"`
}

// Metadata returns the data source type name.
func (d *webhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

// Schema defines the schema for the data source.
func (d *webhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a Retrotro webhook by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the webhook.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Description: "The ID of the team the webhook belongs to.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the webhook.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL the webhook sends events to.",
				Computed:    true,
			},
			"events": schema.ListAttribute{
				Description: "List of events the webhook is subscribed to.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the webhook is enabled.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *webhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*RetrotroClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *RetrotroClient, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *webhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state webhookDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the webhook
	webhook, err := d.client.GetWebhook(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading webhook",
			"Could not read webhook: "+err.Error(),
		)
		return
	}

	if webhook == nil {
		resp.Diagnostics.AddError(
			"Webhook not found",
			fmt.Sprintf("No webhook found with ID: %s", state.ID.ValueString()),
		)
		return
	}

	// Map response to state
	state.Name = types.StringValue(webhook.Name)
	state.URL = types.StringValue(webhook.URL)

	eventsList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Events = eventsList
	state.IsEnabled = types.BoolValue(webhook.IsEnabled)

	// Set state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
