package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

// NewWebhookResource is a helper function to simplify the provider implementation.
func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

// webhookResource is the resource implementation.
type webhookResource struct {
	client *RetrotroClient
}

// webhookResourceModel maps the resource schema data.
type webhookResourceModel struct {
	ID        types.String `tfsdk:"id"`
	TeamID    types.String `tfsdk:"team_id"`
	Name      types.String `tfsdk:"name"`
	URL       types.String `tfsdk:"url"`
	Secret    types.String `tfsdk:"secret"`
	Events    types.List   `tfsdk:"events"`
	IsEnabled types.Bool   `tfsdk:"enabled"`
}

// Metadata returns the resource type name.
func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

// Schema defines the schema for the resource.
func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Retrotro webhook.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the webhook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Description: "The ID of the team this webhook belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the webhook.",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL to send webhook events to.",
				Required:    true,
			},
			"secret": schema.StringAttribute{
				Description: "Secret for signing webhook payloads (HMAC-SHA256).",
				Optional:    true,
				Sensitive:   true,
			},
			"events": schema.ListAttribute{
				Description: "List of events to subscribe to. Valid values: 'retro.completed', 'action.created'.",
				Required:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the webhook is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*RetrotroClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *RetrotroClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract events from the plan
	var events []string
	diags = plan.Events.ElementsAs(ctx, &events, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the request
	createReq := CreateWebhookRequest{
		Name:      plan.Name.ValueString(),
		URL:       plan.URL.ValueString(),
		Events:    events,
		IsEnabled: plan.IsEnabled.ValueBool(),
	}

	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
		secret := plan.Secret.ValueString()
		createReq.Secret = &secret
	}

	// Create the webhook
	webhook, err := r.client.CreateWebhook(ctx, plan.TeamID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating webhook",
			"Could not create webhook: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(webhook.ID)

	eventsList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Events = eventsList
	plan.IsEnabled = types.BoolValue(webhook.IsEnabled)

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the webhook
	webhook, err := r.client.GetWebhook(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading webhook",
			"Could not read webhook: "+err.Error(),
		)
		return
	}

	if webhook == nil {
		resp.State.RemoveResource(ctx)
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

// Update updates the resource and sets the updated Terraform state on success.
func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract events from the plan
	var events []string
	diags = plan.Events.ElementsAs(ctx, &events, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the request
	name := plan.Name.ValueString()
	url := plan.URL.ValueString()
	enabled := plan.IsEnabled.ValueBool()

	updateReq := UpdateWebhookRequest{
		Name:      &name,
		URL:       &url,
		Events:    events,
		IsEnabled: &enabled,
	}

	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
		secret := plan.Secret.ValueString()
		updateReq.Secret = &secret
	}

	// Update the webhook
	webhook, err := r.client.UpdateWebhook(ctx, plan.TeamID.ValueString(), plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating webhook",
			"Could not update webhook: "+err.Error(),
		)
		return
	}

	// Map response to state
	eventsList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Events = eventsList
	plan.IsEnabled = types.BoolValue(webhook.IsEnabled)

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the webhook
	err := r.client.DeleteWebhook(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting webhook",
			"Could not delete webhook: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
