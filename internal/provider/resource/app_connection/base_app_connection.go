package resource

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AppConnectionBaseResource is the resource implementation.
type AppConnectionBaseResource struct {
	App                              infisical.AppConnectionApp // used for identifying secret sync route
	ResourceTypeName                 string                     // terraform resource name suffix
	AppConnectionName                string                     // complete descriptive name of the app connection
	SupportsGateway                  bool                       // when true, exposes gateway_id and sends it to the API
	client                           *infisical.Client
	AllowedMethods                   []string
	CredentialsAttributes            map[string]schema.Attribute
	ReadCredentialsForCreateFromPlan func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics)
	ReadCredentialsForUpdateFromPlan func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics)
	OverwriteCredentialsFields       func(state *AppConnectionBaseResourceModel) diag.Diagnostics
	// IsRetryableError, if set, is called on API errors during Create/Update.
	// Returning true causes the operation to be retried with exponential backoff.
	IsRetryableError func(err error) bool
}

const (
	appConnectionMaxRetries    = 3
	appConnectionInitialDelay  = 10 * time.Second
	appConnectionBackoffFactor = 1.5
)

// retryAppConnectionOp calls fn repeatedly with exponential backoff while isRetryable returns true.
// It makes up to maxRetries additional attempts after the first try.
func retryAppConnectionOp(
	ctx context.Context,
	isRetryable func(error) bool,
	fn func() error,
) error {
	delay := appConnectionInitialDelay
	var lastErr error
	for attempt := 0; attempt <= appConnectionMaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if isRetryable == nil || !isRetryable(lastErr) {
			return lastErr
		}

		if attempt == appConnectionMaxRetries {
			break
		}

		sleep := jitter(delay, 0.2)
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(ctx.Err(), lastErr)
		case <-timer.C:
		}

		next := time.Duration(float64(delay) * appConnectionBackoffFactor)
		delay = next
	}

	return lastErr
}

// jitter adds random noise (±fraction of d) to a duration to avoid thundering-herd retries.
func jitter(d time.Duration, fraction float64) time.Duration {
	delta := time.Duration(float64(d) * fraction)
	return d - delta + time.Duration(rand.Int64N(int64(2*delta+1)))
}

type AppConnectionBaseResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Method          types.String `tfsdk:"method"`
	Description     types.String `tfsdk:"description"`
	ProjectId       types.String `tfsdk:"project_id"`
	Credentials     types.Object `tfsdk:"credentials"`
	CredentialsHash types.String `tfsdk:"credentials_hash"`
}

// appConnectionGatewayResourceModel is used for plan/state I/O on connections that support
// gateways. The embedded base model's tfsdk fields are promoted, so this matches the schema
// that includes gateway_id. Connections without gateway support keep using the base model
// directly, leaving their schema untouched.
type appConnectionGatewayResourceModel struct {
	AppConnectionBaseResourceModel
	GatewayId types.String `tfsdk:"gateway_id"`
}

// planStateGetter is satisfied by both tfsdk.Plan and tfsdk.State (value receivers), letting
// readModel handle plan and state reads uniformly.
type planStateGetter interface {
	Get(ctx context.Context, target interface{}) diag.Diagnostics
}

// readModel reads plan/state into the base model. For gateway-enabled connections it also
// returns the configured gateway_id; otherwise that value is null.
func (r *AppConnectionBaseResource) readModel(ctx context.Context, src planStateGetter) (AppConnectionBaseResourceModel, types.String, diag.Diagnostics) {
	if r.SupportsGateway {
		var m appConnectionGatewayResourceModel
		diags := src.Get(ctx, &m)
		return m.AppConnectionBaseResourceModel, m.GatewayId, diags
	}

	var m AppConnectionBaseResourceModel
	diags := src.Get(ctx, &m)
	return m, types.StringNull(), diags
}

// stateValue returns the value to write to state, gateway-aware when the connection supports
// gateways so the struct matches the schema (which then includes gateway_id).
func (r *AppConnectionBaseResource) stateValue(model AppConnectionBaseResourceModel, gatewayId types.String) interface{} {
	if r.SupportsGateway {
		return appConnectionGatewayResourceModel{
			AppConnectionBaseResourceModel: model,
			GatewayId:                      gatewayId,
		}
	}
	return model
}

// reconcileGatewayId returns the gateway_id to persist from the API response. For connections
// that don't support gateways it leaves the current value untouched.
func (r *AppConnectionBaseResource) reconcileGatewayId(current types.String, appConnection infisical.AppConnection) types.String {
	if !r.SupportsGateway {
		return current
	}
	if appConnection.GatewayId != nil {
		return types.StringValue(*appConnection.GatewayId)
	}
	return types.StringNull()
}

// gatewayIdForRequest returns the gateway_id value to send to the API. It returns a non-nil
// pointer only when the resource supports gateways and a value is set. When the gateway has
// been removed from the configuration it returns nil, which serializes to an explicit `null`
// so the API resets the connection back to the Internet Gateway (omitting the field would
// leave the existing gateway untouched).
func (r *AppConnectionBaseResource) gatewayIdForRequest(gatewayId types.String) *string {
	if !r.SupportsGateway {
		return nil
	}
	if gatewayId.IsNull() || gatewayId.IsUnknown() || gatewayId.ValueString() == "" {
		return nil
	}
	value := gatewayId.ValueString()
	return &value
}

// Metadata returns the resource type name.
func (r *AppConnectionBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

func (r *AppConnectionBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   "The ID of the app connection",
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"method": schema.StringAttribute{
			Required:    true,
			Description: fmt.Sprintf("The method used to authenticate with %s. Possible values are: %s", r.AppConnectionName, strings.Join(r.AllowedMethods, ", ")),
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: fmt.Sprintf("The name of the %s App Connection to create. Must be slug-friendly", r.AppConnectionName),
		},
		"description": schema.StringAttribute{
			Optional:    true,
			Description: fmt.Sprintf("An optional description for the %s App Connection.", r.AppConnectionName),
		},
		"project_id": schema.StringAttribute{
			Optional:      true,
			Description:   "The ID of the project to scope the app connection to. If not provided, the app connection will be scoped to the organization.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"credentials": schema.SingleNestedAttribute{
			Required:    true,
			Description: fmt.Sprintf("The credentials for the %s App Connection", r.AppConnectionName),
			Attributes:  r.CredentialsAttributes,
		},
		"credentials_hash": schema.StringAttribute{
			Computed:    true,
			Description: fmt.Sprintf("The hash of the %s App Connection credentials", r.AppConnectionName),
		},
	}

	if r.SupportsGateway {
		attributes["gateway_id"] = schema.StringAttribute{
			Optional:    true,
			Description: "The Gateway ID to use for the app connection. If not specified, the Internet Gateway will be used.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		}
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s App Connection", r.AppConnectionName),
		Attributes:  attributes,
	}
}

// Configure adds the provider configured client to the resource.
func (r *AppConnectionBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *AppConnectionBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create app connection",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	plan, gatewayId, diags := r.readModel(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	methodIsValid := false
	for _, method := range r.AllowedMethods {
		if plan.Method.ValueString() == method {
			methodIsValid = true
			break
		}
	}

	if !methodIsValid {
		resp.Diagnostics.AddError(
			"Unable to create app connection",
			fmt.Sprintf("Invalid value for method field. Allowed values are: %s", strings.Join(r.AllowedMethods, ", ")),
		)
		return
	}

	credentialsMap, diags := r.ReadCredentialsForCreateFromPlan(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var appConnection infisical.AppConnection
	err := retryAppConnectionOp(ctx, r.IsRetryableError, func() error {
		var apiErr error
		baseRequest := infisical.CreateAppConnectionRequest{
			App:         r.App,
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Method:      plan.Method.ValueString(),
			Credentials: credentialsMap,
			ProjectId:   plan.ProjectId.ValueString(),
		}

		if r.SupportsGateway {
			appConnection, apiErr = r.client.CreateAppConnectionWithGateway(infisical.CreateAppConnectionWithGateway{
				CreateAppConnectionRequest: baseRequest,
				GatewayId:                  r.gatewayIdForRequest(gatewayId),
			})
		} else {
			appConnection, apiErr = r.client.CreateAppConnection(baseRequest)
		}
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating app connection",
			"Couldn't create app connection, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(appConnection.Id)
	plan.CredentialsHash = types.StringValue(appConnection.CredentialsHash)
	gatewayId = r.reconcileGatewayId(gatewayId, appConnection)

	diags = resp.State.Set(ctx, r.stateValue(plan, gatewayId))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *AppConnectionBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read app connection",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	state, gatewayId, diags := r.readModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	appConnection, err := r.client.GetAppConnectionById(infisical.GetAppConnectionByIdRequest{
		App: r.App,
		ID:  state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading app connection",
				"Couldn't read app connection, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Reconcile gateway_id from the API so out-of-band changes are detected as drift.
	gatewayId = r.reconcileGatewayId(gatewayId, appConnection)

	if state.CredentialsHash.ValueString() != appConnection.CredentialsHash {
		resp.Diagnostics.AddWarning(
			"App connection credentials conflict",
			fmt.Sprintf("The credentials for the %s App Connection with ID %s have been updated outside of Terraform.", r.AppConnectionName, state.ID.ValueString()),
		)

		// force TF update
		diags = r.OverwriteCredentialsFields(&state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		diags = resp.State.Set(ctx, r.stateValue(state, gatewayId))
		resp.Diagnostics.Append(diags...)
		return
	}

	if !(state.Description.IsNull() && appConnection.Description == "") {
		state.Description = types.StringValue(appConnection.Description)
	}

	if appConnection.ProjectId != nil {
		state.ProjectId = types.StringValue(*appConnection.ProjectId)
	} else {
		state.ProjectId = types.StringNull()
	}

	state.Method = types.StringValue(appConnection.Method)
	state.Name = types.StringValue(appConnection.Name)

	diags = resp.State.Set(ctx, r.stateValue(state, gatewayId))
	resp.Diagnostics.Append(diags...)
}

func (r *AppConnectionBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update app connection",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	plan, gatewayId, diags := r.readModel(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, _, diags := r.readModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	methodIsValid := false
	for _, method := range r.AllowedMethods {
		if plan.Method.ValueString() == method {
			methodIsValid = true
			break
		}
	}

	if !methodIsValid {
		resp.Diagnostics.AddError(
			"Unable to create app connection",
			fmt.Sprintf("Invalid value for method field. Allowed values are: %s", strings.Join(r.AllowedMethods, ", ")),
		)
		return
	}

	credentialsMap, diags := r.ReadCredentialsForUpdateFromPlan(ctx, plan, state)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var appConnection infisical.AppConnection
	err := retryAppConnectionOp(ctx, r.IsRetryableError, func() error {
		var apiErr error
		baseRequest := infisical.UpdateAppConnectionRequest{
			ID:          state.ID.ValueString(),
			App:         r.App,
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Method:      plan.Method.ValueString(),
			Credentials: credentialsMap,
			ProjectId:   plan.ProjectId.ValueString(),
		}

		if r.SupportsGateway {
			appConnection, apiErr = r.client.UpdateAppConnectionWithGateway(infisical.UpdateAppConnectionWithGateway{
				UpdateAppConnectionRequest: baseRequest,
				GatewayId:                  r.gatewayIdForRequest(gatewayId),
			})
		} else {
			appConnection, apiErr = r.client.UpdateAppConnection(baseRequest)
		}
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating app connection",
			"Couldn't update app connection, unexpected error: "+err.Error(),
		)
		return
	}

	plan.CredentialsHash = types.StringValue(appConnection.CredentialsHash)
	gatewayId = r.reconcileGatewayId(gatewayId, appConnection)

	diags = resp.State.Set(ctx, r.stateValue(plan, gatewayId))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AppConnectionBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete app connection",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	state, _, diags := r.readModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteAppConnection(infisical.DeleteAppConnectionRequest{
		App: r.App,
		ID:  state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting app connection",
			"Couldn't delete app connection from Infisical, unexpected error: "+err.Error(),
		)
	}
}
