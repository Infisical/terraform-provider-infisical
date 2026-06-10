package resource

import (
	"context"
	"fmt"
	"strings"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	WEBHOOK_TYPE_GENERAL         = "general"
	WEBHOOK_TYPE_SLACK           = "slack"
	WEBHOOK_TYPE_MICROSOFT_TEAMS = "microsoft-teams"
	SUPPORTED_WEBHOOK_TYPES      = []string{WEBHOOK_TYPE_GENERAL, WEBHOOK_TYPE_SLACK, WEBHOOK_TYPE_MICROSOFT_TEAMS}

	WEBHOOK_EVENT_SECRET_MODIFIED        = "secrets.modified"
	WEBHOOK_EVENT_SECRET_ROTATION_FAILED = "secrets.rotation-failed"
	SUPPORTED_WEBHOOK_EVENTS             = []string{WEBHOOK_EVENT_SECRET_MODIFIED, WEBHOOK_EVENT_SECRET_ROTATION_FAILED}
)

var (
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithConfigure   = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

type webhookResource struct {
	client *infisical.Client
}

type webhookResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	Environment      types.String `tfsdk:"environment"`
	SecretPath       types.String `tfsdk:"secret_path"`
	WebhookURL       types.String `tfsdk:"webhook_url"`
	WebhookSecretKey types.String `tfsdk:"webhook_secret_key"`
	Type             types.String `tfsdk:"type"`
	EventsFilter     types.Set    `tfsdk:"events_filter"`
	IsDisabled       types.Bool   `tfsdk:"is_disabled"`
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage webhooks in Infisical. Webhooks notify an external URL when secrets change at a given path.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the webhook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project the webhook belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				Description: "The slug of the environment the webhook listens to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path the webhook listens to. Defaults to '/'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"webhook_url": schema.StringAttribute{
				Description: "The URL Infisical sends the event payload to.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"webhook_secret_key": schema.StringAttribute{
				Description: "The secret key used to sign the webhook payload so the receiver can verify it. Write-only: it is never returned by the API, so it cannot be imported.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The webhook type. Options: " + strings.Join(SUPPORTED_WEBHOOK_TYPES, ", ") + ". Defaults to '" + WEBHOOK_TYPE_GENERAL + "'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(WEBHOOK_TYPE_GENERAL),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_WEBHOOK_TYPES...),
				},
			},
			"events_filter": schema.SetAttribute{
				Description: "The events that trigger the webhook. Options: " + strings.Join(SUPPORTED_WEBHOOK_EVENTS, ", ") + ". An empty set means the webhook fires on every supported event.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_WEBHOOK_EVENTS...)),
				},
			},
			"is_disabled": schema.BoolAttribute{
				Description: "Whether the webhook is disabled. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func eventsFilterToSet(ctx context.Context, filters []infisical.WebhookEventFilter) (types.Set, diag.Diagnostics) {
	names := make([]string, 0, len(filters))
	for _, f := range filters {
		names = append(names, f.EventName)
	}
	return types.SetValueFrom(ctx, types.StringType, names)
}

func (r *webhookResource) eventsFilterFromSet(ctx context.Context, set types.Set) ([]infisical.WebhookEventFilter, diag.Diagnostics) {
	var diags diag.Diagnostics
	filters := []infisical.WebhookEventFilter{}
	if set.IsNull() || set.IsUnknown() {
		return filters, diags
	}

	var names []string
	diags.Append(set.ElementsAs(ctx, &names, false)...)
	if diags.HasError() {
		return filters, diags
	}

	for _, name := range names {
		filters = append(filters, infisical.WebhookEventFilter{EventName: name})
	}
	return filters, diags
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eventsFilter, diags := r.eventsFilterFromSet(ctx, plan.EventsFilter)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := infisical.CreateWebhookRequest{
		ProjectID:    plan.ProjectID.ValueString(),
		Environment:  plan.Environment.ValueString(),
		WebhookUrl:   plan.WebhookURL.ValueString(),
		SecretPath:   plan.SecretPath.ValueString(),
		Type:         plan.Type.ValueString(),
		EventsFilter: eventsFilter,
	}

	if !plan.WebhookSecretKey.IsNull() && !plan.WebhookSecretKey.IsUnknown() {
		createRequest.WebhookSecretKey = plan.WebhookSecretKey.ValueString()
	}

	webhook, err := r.client.CreateWebhook(createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating webhook",
			"Couldn't create webhook in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(webhook.Webhook.ID)
	plan.SecretPath = types.StringValue(webhook.Webhook.SecretPath)
	plan.Type = types.StringValue(webhook.Webhook.Type)
	plan.Environment = types.StringValue(webhook.Webhook.Environment.Slug)

	filterSet, diags := eventsFilterToSet(ctx, webhook.Webhook.EventsFilter)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.EventsFilter = filterSet

	// Create always provisions the webhook enabled; honor a requested disabled state with a follow-up update.
	if !plan.IsDisabled.IsNull() && !plan.IsDisabled.IsUnknown() && plan.IsDisabled.ValueBool() != webhook.Webhook.IsDisabled {
		isDisabled := plan.IsDisabled.ValueBool()
		updated, err := r.client.UpdateWebhook(infisical.UpdateWebhookRequest{
			ID:         webhook.Webhook.ID,
			IsDisabled: &isDisabled,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error setting webhook disabled state",
				"Couldn't update webhook disabled state in Infisical, unexpected error: "+err.Error(),
			)
			return
		}
		plan.IsDisabled = types.BoolValue(updated.Webhook.IsDisabled)
	} else {
		plan.IsDisabled = types.BoolValue(webhook.Webhook.IsDisabled)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.client.GetWebhookByID(infisical.GetWebhookByIDRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading webhook",
			"Couldn't read webhook with ID "+state.ID.ValueString()+" from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.ProjectID = types.StringValue(webhook.Webhook.ProjectID)
	state.Environment = types.StringValue(webhook.Webhook.Environment.Slug)
	state.SecretPath = types.StringValue(webhook.Webhook.SecretPath)
	state.Type = types.StringValue(webhook.Webhook.Type)
	state.WebhookURL = types.StringValue(webhook.Webhook.URL)
	state.IsDisabled = types.BoolValue(webhook.Webhook.IsDisabled)

	filterSet, diags := eventsFilterToSet(ctx, webhook.Webhook.EventsFilter)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.EventsFilter = filterSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	var state webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := infisical.UpdateWebhookRequest{
		ID: state.ID.ValueString(),
	}

	if !plan.IsDisabled.Equal(state.IsDisabled) {
		isDisabled := plan.IsDisabled.ValueBool()
		updateRequest.IsDisabled = &isDisabled
	}

	if !plan.EventsFilter.Equal(state.EventsFilter) {
		eventsFilter, diags := r.eventsFilterFromSet(ctx, plan.EventsFilter)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateRequest.EventsFilter = &eventsFilter
	}

	// Only is_disabled and events_filter are mutable; every other attribute forces replacement.
	if updateRequest.IsDisabled == nil && updateRequest.EventsFilter == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	updated, err := r.client.UpdateWebhook(updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating webhook",
			"Couldn't update webhook in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(updated.Webhook.ID)
	plan.IsDisabled = types.BoolValue(updated.Webhook.IsDisabled)

	filterSet, diags := eventsFilterToSet(ctx, updated.Webhook.EventsFilter)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.EventsFilter = filterSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteWebhook(infisical.DeleteWebhookRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting webhook",
			"Couldn't delete webhook from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
