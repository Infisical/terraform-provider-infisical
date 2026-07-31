package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

const (
	identityAuthenticationResourceType   = "identity.authentication"
	identityAuthenticationExpiryEventKey = "identity.authentication.expiry"
)

const (
	minAlertBeforeDays = 1
	maxAlertBeforeDays = 90
)

var (
	_ resource.Resource                = &alertIdentityAuthenticationExpiryResource{}
	_ resource.ResourceWithConfigure   = &alertIdentityAuthenticationExpiryResource{}
	_ resource.ResourceWithImportState = &alertIdentityAuthenticationExpiryResource{}
)

func NewAlertIdentityAuthenticationExpiryResource() resource.Resource {
	return &alertIdentityAuthenticationExpiryResource{}
}

type alertIdentityAuthenticationExpiryResource struct {
	client *infisical.Client
}

type alertIdentityAuthenticationExpiryResourceModel struct {
	ID              types.String                 `tfsdk:"id"`
	IdentityID      types.String                 `tfsdk:"identity_id"`
	ProjectID       types.String                 `tfsdk:"project_id"`
	Name            types.String                 `tfsdk:"name"`
	Description     types.String                 `tfsdk:"description"`
	Enabled         types.Bool                   `tfsdk:"enabled"`
	AlertBeforeDays types.Int64                  `tfsdk:"alert_before_days"`
	DailyReminder   types.Bool                   `tfsdk:"daily_reminder"`
	Channels        map[string]alertChannelModel `tfsdk:"channels"`
}

type identityAuthenticationExpiryCondition struct {
	AlertBefore   string `json:"alertBefore"`
	DailyReminder bool   `json:"dailyReminder"`
}

func (r *alertIdentityAuthenticationExpiryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_identity_authentication_expiry"
}

func (r *alertIdentityAuthenticationExpiryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage alerts that notify you before a machine identity's authentication credentials expire. A machine identity can only have one authentication expiry alert.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the alert.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identity_id": schema.StringAttribute{
				Description: "The ID of the machine identity to watch.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project the machine identity belongs to. Required for project level identities, and must be omitted for organization level ones.",
				Optional:    true,
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
			"alert_before_days": schema.Int64Attribute{
				Description: fmt.Sprintf("How many days before an authentication credential expires the alert fires. Must be between %d and %d.", minAlertBeforeDays, maxAlertBeforeDays),
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(minAlertBeforeDays, maxAlertBeforeDays),
				},
			},
			"daily_reminder": schema.BoolAttribute{
				Description: "Whether to keep notifying once a day until the credential expires, instead of notifying once. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"channels": alertChannelsSchema(),
		},
	}
}

func (r *alertIdentityAuthenticationExpiryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *alertIdentityAuthenticationExpiryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertIdentityAuthenticationExpiryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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
		ResourceType: identityAuthenticationResourceType,
		ResourceID:   plan.IdentityID.ValueString(),
		EventType:    identityAuthenticationExpiryEventKey,
		Condition:    conditionFromPlan(plan),
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
			"Couldn't create machine identity authentication expiry alert in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(alert.Alert.ID)

	channelsWithIDs, diags := alertChannelIDsFromAPI(plan.Channels, alert.Alert.Channels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Channels = channelsWithIDs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertIdentityAuthenticationExpiryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertIdentityAuthenticationExpiryResourceModel
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

	if alert.Alert.ResourceType != identityAuthenticationResourceType || alert.Alert.EventType != identityAuthenticationExpiryEventKey {
		resp.Diagnostics.AddError(
			"Unexpected alert type",
			fmt.Sprintf("Alert with ID %s watches %s for %s events, so it cannot be managed as a machine identity authentication expiry alert.", state.ID.ValueString(), alert.Alert.ResourceType, alert.Alert.EventType),
		)
		return
	}

	resp.Diagnostics.Append(r.setStateFromAlert(ctx, &state, alert.Alert)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertIdentityAuthenticationExpiryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertIdentityAuthenticationExpiryResourceModel
	var state alertIdentityAuthenticationExpiryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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
		Condition: conditionFromPlan(plan),
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
			"Couldn't update machine identity authentication expiry alert in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	channelsWithIDs, diags := alertChannelIDsFromAPI(plan.Channels, alert.Alert.Channels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Channels = channelsWithIDs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertIdentityAuthenticationExpiryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertIdentityAuthenticationExpiryResourceModel
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
			"Couldn't delete machine identity authentication expiry alert from Infisical, unexpected error: "+err.Error(),
		)
	}
}

func (r *alertIdentityAuthenticationExpiryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *alertIdentityAuthenticationExpiryResource) setStateFromAlert(ctx context.Context, state *alertIdentityAuthenticationExpiryResourceModel, alert infisical.Alert) diag.Diagnostics {
	var diags diag.Diagnostics

	state.ID = types.StringValue(alert.ID)
	state.Name = types.StringValue(alert.Name)
	state.Enabled = types.BoolValue(alert.Enabled)

	if alert.ResourceID != nil {
		state.IdentityID = types.StringValue(*alert.ResourceID)
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

	condition, conditionDiags := parseCondition(alert.Condition)
	diags.Append(conditionDiags...)
	if diags.HasError() {
		return diags
	}

	alertBeforeDays, daysDiags := parseAlertBeforeDays(condition.AlertBefore)
	diags.Append(daysDiags...)
	if diags.HasError() {
		return diags
	}

	state.AlertBeforeDays = types.Int64Value(alertBeforeDays)
	state.DailyReminder = types.BoolValue(condition.DailyReminder)

	channels, channelDiags := alertChannelsFromAPI(ctx, alert.Channels, state.Channels)
	diags.Append(channelDiags...)
	if diags.HasError() {
		return diags
	}
	state.Channels = channels

	return diags
}

func conditionFromPlan(plan alertIdentityAuthenticationExpiryResourceModel) identityAuthenticationExpiryCondition {
	return identityAuthenticationExpiryCondition{
		AlertBefore:   fmt.Sprintf("%dd", plan.AlertBeforeDays.ValueInt64()),
		DailyReminder: plan.DailyReminder.ValueBool(),
	}
}

func parseCondition(raw json.RawMessage) (identityAuthenticationExpiryCondition, diag.Diagnostics) {
	var diags diag.Diagnostics
	var condition identityAuthenticationExpiryCondition

	if len(raw) == 0 || string(raw) == "null" {
		diags.AddError(
			"Missing alert condition",
			"Infisical returned an alert without a condition, so the number of days it alerts before expiry is unknown.",
		)
		return condition, diags
	}

	if err := json.Unmarshal(raw, &condition); err != nil {
		diags.AddError(
			"Error reading alert condition",
			"Couldn't parse the condition returned by Infisical, unexpected error: "+err.Error(),
		)
	}

	return condition, diags
}

func parseAlertBeforeDays(alertBefore string) (int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	days, err := strconv.ParseInt(strings.TrimSuffix(alertBefore, "d"), 10, 64)
	if err != nil || !strings.HasSuffix(alertBefore, "d") {
		diags.AddError(
			"Error reading alert condition",
			fmt.Sprintf("Infisical returned an alert that fires %q before expiry, which is not a whole number of days such as \"30d\".", alertBefore),
		)
		return 0, diags
	}

	return days, diags
}
