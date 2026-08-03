package resource

import (
	"context"
	"fmt"
	"strings"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = &alertResource{}
	_ resource.ResourceWithConfigure        = &alertResource{}
	_ resource.ResourceWithImportState      = &alertResource{}
	_ resource.ResourceWithConfigValidators = &alertResource{}
	_ resource.ResourceWithValidateConfig   = &alertResource{}
	_ resource.ResourceWithModifyPlan       = &alertResource{}
)

func NewAlertResource() resource.Resource {
	return &alertResource{}
}

type alertResource struct {
	client *infisical.Client
}

type alertResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceID   types.String `tfsdk:"resource_id"`
	ProjectID    types.String `tfsdk:"project_id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Enabled      types.Bool   `tfsdk:"enabled"`

	AuthenticationExpiry *authenticationExpiryConditionModel `tfsdk:"authentication_expiry"`

	Channels map[string]alertChannelModel `tfsdk:"channels"`
}

func (r *alertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert"
}

func (r *alertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage alerts that notify you about a resource in Infisical. An alert watches one resource for one event, and a resource can only have one alert per event within a scope.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the alert.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type": schema.StringAttribute{
				Description: "The type of the resource to watch. Options: " + strings.Join(supportedAlertResourceTypes(), ", ") + ".",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(supportedAlertResourceTypes()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: "The ID of the resource to watch. For the " + alertResourceTypeIdentityAuthentication + " resource type this is a machine identity's ID.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project the resource belongs to. Required for project level resources, and must be omitted for organization level ones.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the alert.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"description": schema.StringAttribute{
				Description: "An optional description of the alert.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1000),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the alert is evaluated. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			alertBlockAuthenticationExpiry: authenticationExpiryConditionSchema(),
			"channels":                     alertChannelsSchema(),
		},
	}
}

func (r *alertResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(alertConditionBlockPaths()...),
	}
}

func (r *alertResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var resourceType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("resource_type"), &resourceType)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if resourceType.IsNull() || resourceType.IsUnknown() {
		return
	}

	accepted := alertBlocksForResourceType(resourceType.ValueString())
	if len(accepted) == 0 {
		return
	}

	for _, event := range alertEvents {
		var block types.Object
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(event.block), &block)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if block.IsNull() || event.resourceType == resourceType.ValueString() {
			continue
		}

		resp.Diagnostics.AddAttributeError(
			path.Root(event.block),
			"Unexpected alert condition",
			fmt.Sprintf(
				"A %s block describes an alert on the %s resource type, but this alert watches %s. Use one of: %s.",
				event.block, event.resourceType, resourceType.ValueString(), strings.Join(accepted, ", "),
			),
		)
	}
}

func (r *alertResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	plannedEvent, plannedEventKnown, plannedDiags := alertEventFromBlocks(ctx, req.Plan)
	storedEvent, storedEventKnown, storedDiags := alertEventFromBlocks(ctx, req.State)
	resp.Diagnostics.Append(plannedDiags...)
	resp.Diagnostics.Append(storedDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plannedEventKnown && storedEventKnown && plannedEvent.eventType != storedEvent.eventType {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root(plannedEvent.block))
	}

	channelsPath := path.Root("channels")

	var planned, stored types.Map
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, channelsPath, &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, channelsPath, &stored)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planned.IsNull() || planned.IsUnknown() || stored.IsNull() || stored.IsUnknown() {
		return
	}

	storedChannels := stored.Elements()
	for name, element := range planned.Elements() {
		channel, ok := element.(types.Object)
		if !ok || channel.IsNull() || channel.IsUnknown() {
			continue
		}
		if id, ok := channel.Attributes()["id"]; ok && id != nil && id.IsUnknown() {
			continue
		}

		storedChannel, ok := storedChannels[name].(types.Object)
		if !ok {
			continue
		}

		if channelKeepsStoredID(channel, storedChannel) {
			continue
		}

		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, channelsPath.AtMapKey(name).AtName("id"), types.StringUnknown())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *alertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *alertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	event, condition, diags := alertConditionForAPI(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	channels, diags := alertChannelsToAPIInput(ctx, plan.Channels, nil)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := infisical.CreateAlertRequest{
		Name:         plan.Name.ValueString(),
		ResourceType: plan.ResourceType.ValueString(),
		ResourceID:   plan.ResourceID.ValueString(),
		EventType:    event.eventType,
		Condition:    condition,
		Enabled:      plan.Enabled.ValueBool(),
		Channels:     channels,
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		createRequest.Description = &description
	}

	if !plan.ProjectID.IsNull() && !plan.ProjectID.IsUnknown() {
		projectID := plan.ProjectID.ValueString()
		createRequest.ProjectID = &projectID
	}

	alert, err := r.client.CreateAlert(createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating alert",
			"Couldn't create alert in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(alert.Alert.ID)

	channelsWithIDs, channelDiags := alertChannelIDsFromAPI(plan.Channels, alert.Alert.Channels)
	plan.Channels = channelsWithIDs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(channelDiags...)
}

func (r *alertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alert, err := r.client.GetAlertByID(infisical.GetAlertByIDRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading alert",
			"Couldn't read alert with ID "+state.ID.ValueString()+" from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.setStateFromAlert(ctx, &state, alert.Alert)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertResourceModel
	var state alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, condition, diags := alertConditionForAPI(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	channels, diags := alertChannelsToAPIInput(ctx, plan.Channels, state.Channels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := infisical.UpdateAlertRequest{
		ID:        state.ID.ValueString(),
		Name:      plan.Name.ValueString(),
		Condition: condition,
		Enabled:   plan.Enabled.ValueBool(),
		Channels:  channels,
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		updateRequest.Description = &description
	}

	alert, err := r.client.UpdateAlert(updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating alert",
			"Couldn't update alert in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	channelsWithIDs, channelDiags := alertChannelIDsFromAPI(plan.Channels, alert.Alert.Channels)
	plan.Channels = channelsWithIDs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(channelDiags...)
}

func (r *alertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteAlert(infisical.DeleteAlertRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError(
			"Error deleting alert",
			"Couldn't delete alert from Infisical, unexpected error: "+err.Error(),
		)
	}
}

func (r *alertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *alertResource) setStateFromAlert(ctx context.Context, state *alertResourceModel, alert infisical.Alert) diag.Diagnostics {
	var diags diag.Diagnostics

	event, ok := alertEventForEventType(alert.EventType)
	if !ok || event.resourceType != alert.ResourceType {
		diags.AddError(
			"Unsupported alert",
			fmt.Sprintf(
				"Alert with ID %s watches %s for %s events, which this provider does not know how to manage. Please upgrade the provider.",
				alert.ID, alert.ResourceType, alert.EventType,
			),
		)
		return diags
	}

	state.ID = types.StringValue(alert.ID)
	state.Name = types.StringValue(alert.Name)
	state.Enabled = types.BoolValue(alert.Enabled)
	state.ResourceType = types.StringValue(alert.ResourceType)

	if alert.ResourceID != nil {
		state.ResourceID = types.StringValue(*alert.ResourceID)
	} else {
		state.ResourceID = types.StringNull()
	}

	if alert.ProjectID != nil {
		state.ProjectID = types.StringValue(*alert.ProjectID)
	} else {
		state.ProjectID = types.StringNull()
	}

	if alert.Description != nil {
		state.Description = types.StringValue(*alert.Description)
	} else {
		state.Description = types.StringNull()
	}

	diags.Append(setAlertConditionFromAPI(state, event, alert.Condition)...)
	if diags.HasError() {
		return diags
	}

	channels, channelDiags := alertChannelsFromAPI(ctx, alert.Channels, state.Channels)
	diags.Append(channelDiags...)
	if diags.HasError() {
		return diags
	}
	state.Channels = channels

	return diags
}
