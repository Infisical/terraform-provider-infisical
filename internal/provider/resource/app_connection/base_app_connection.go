package resource

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithImportState      = &AppConnectionBaseResource{}
	_ resource.ResourceWithConfigValidators = &AppConnectionBaseResource{}
	_ resource.ResourceWithModifyPlan       = &AppConnectionBaseResource{}
)

// credentialsManagedByInfisical reports whether Infisical owns the connection's credentials:
// platform-managed credentials or a credential rotation that is turned on. Infisical rejects an
// update that sends credentials to a platform-managed connection, and replaces credentials it
// rotates (backend/src/services/app-connection/app-connection-service.ts in Infisical core), so
// the provider never sends them for these connections. A paused rotation can't be detected:
// GET by ID doesn't return the rotation, only the list endpoint does.
func credentialsManagedByInfisical(appConnection infisical.AppConnection) bool {
	return appConnection.IsPlatformManagedCredentials || appConnection.IsAutoRotationEnabled
}

// credentialsManagedByInfisicalForID reads the connection to find out whether Infisical manages
// its credentials. The API never returns credentials, so this is the only way to know before an
// update.
func (r *AppConnectionBaseResource) credentialsManagedByInfisicalForID(id string) (bool, error) {
	appConnection, err := r.client.GetAppConnectionById(infisical.GetAppConnectionByIdRequest{App: r.App, ID: id})
	if err != nil {
		return false, err
	}
	return credentialsManagedByInfisical(appConnection), nil
}

// AppConnectionBaseResource is the resource implementation.
type AppConnectionBaseResource struct {
	App                              infisical.AppConnectionApp // used for identifying secret sync route
	ResourceTypeName                 string                     // terraform resource name suffix
	AppConnectionName                string                     // complete descriptive name of the app connection
	DocsURL                          string                     // optional Infisical guide linked from the resource description
	SupportsGateway                  bool                       // when true, exposes gateway_id and gateway_pool_id and sends them to the API
	RequiresGateway                  bool                       // when true, one of gateway_id or gateway_pool_id is required
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
// that includes gateway_id and gateway_pool_id. Connections without gateway support keep using
// the base model directly, leaving their schema untouched.
type appConnectionGatewayResourceModel struct {
	AppConnectionBaseResourceModel
	GatewayId     types.String `tfsdk:"gateway_id"`
	GatewayPoolId types.String `tfsdk:"gateway_pool_id"`
}

// appConnectionGateway is how a connection reaches its host: one gateway, a gateway pool, or
// neither (the Internet Gateway). The API allows at most one of the two.
type appConnectionGateway struct {
	Id     types.String
	PoolId types.String
}

// planStateGetter is satisfied by both tfsdk.Plan and tfsdk.State (value receivers), letting
// readModel handle plan and state reads uniformly.
type planStateGetter interface {
	Get(ctx context.Context, target interface{}) diag.Diagnostics
}

// readModel reads plan/state into the base model. For gateway-enabled connections it also
// returns the configured gateway_id and gateway_pool_id; otherwise both are null.
func (r *AppConnectionBaseResource) readModel(ctx context.Context, src planStateGetter) (AppConnectionBaseResourceModel, appConnectionGateway, diag.Diagnostics) {
	if r.SupportsGateway {
		var m appConnectionGatewayResourceModel
		diags := src.Get(ctx, &m)
		return m.AppConnectionBaseResourceModel, appConnectionGateway{Id: m.GatewayId, PoolId: m.GatewayPoolId}, diags
	}

	var m AppConnectionBaseResourceModel
	diags := src.Get(ctx, &m)
	return m, appConnectionGateway{Id: types.StringNull(), PoolId: types.StringNull()}, diags
}

// stateValue returns the value to write to state, gateway-aware when the connection supports
// gateways so the struct matches the schema (which then includes gateway_id and gateway_pool_id).
func (r *AppConnectionBaseResource) stateValue(model AppConnectionBaseResourceModel, gateway appConnectionGateway) interface{} {
	if r.SupportsGateway {
		return appConnectionGatewayResourceModel{
			AppConnectionBaseResourceModel: model,
			GatewayId:                      gateway.Id,
			GatewayPoolId:                  gateway.PoolId,
		}
	}
	return model
}

// reconcileGateway returns the gateway_id and gateway_pool_id to persist from the API response,
// so a gateway or pool chosen in the dashboard shows up as drift. For connections that don't
// support gateways it leaves the current value untouched.
func (r *AppConnectionBaseResource) reconcileGateway(current appConnectionGateway, appConnection infisical.AppConnection) appConnectionGateway {
	if !r.SupportsGateway {
		return current
	}
	return appConnectionGateway{
		Id:     stringValueOrNull(appConnection.GatewayId),
		PoolId: stringValueOrNull(appConnection.GatewayPoolId),
	}
}

func stringValueOrNull(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// gatewayFieldForRequest returns a gateway_id or gateway_pool_id value to send to the API. Both
// are always sent, and an unset one goes as an explicit `null`. The API clears a field it
// receives as null and leaves an omitted one as it is, and setting a gateway clears the pool
// (and the reverse) only for the field that was left out, so sending both is the only way the
// result matches the configuration.
func gatewayFieldForRequest(value types.String) *string {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	v := value.ValueString()
	return &v
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
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:    []validator.String{stringvalidator.OneOf(r.AllowedMethods...)},
			Required:      true,
			Description:   fmt.Sprintf("The method used to authenticate with %s. Possible values are: %s", r.AppConnectionName, strings.Join(r.AllowedMethods, ", ")),
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
		gatewayDescription := "The Gateway ID to use for the app connection. Conflicts with `gateway_pool_id`. If neither is specified, the Internet Gateway will be used."
		poolDescription := "The ID of the gateway pool to use for the app connection, instead of a single gateway. Conflicts with `gateway_id`. Gateway pools require an Infisical Enterprise plan. Requires Infisical v0.159.29 or later."
		if r.RequiresGateway {
			gatewayDescription = fmt.Sprintf("The Gateway ID to use for the app connection. %s connections can only reach the host through a gateway, so set either `gateway_id` or `gateway_pool_id`.", r.AppConnectionName)
			poolDescription = fmt.Sprintf("The ID of the gateway pool to use for the app connection, instead of a single gateway. %s connections can only reach the host through a gateway, so set either `gateway_id` or `gateway_pool_id`. Gateway pools require an Infisical Enterprise plan. Requires Infisical v0.159.29 or later.", r.AppConnectionName)
		}
		attributes["gateway_id"] = schema.StringAttribute{
			Optional:    true,
			Description: gatewayDescription,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		}
		attributes["gateway_pool_id"] = schema.StringAttribute{
			Optional:    true,
			Description: poolDescription,
			Validators:  []validator.String{infisicaltf.UuidValidator},
		}
	}

	description := fmt.Sprintf("Create and manage %s App Connection", r.AppConnectionName)
	if r.DocsURL != "" {
		description += fmt.Sprintf(". For setup, see the [%s app connection guide](%s).", r.AppConnectionName, r.DocsURL)
	}

	resp.Schema = schema.Schema{
		Description: description,
		Attributes:  attributes,
	}
}

// ConfigValidators makes gateway_id and gateway_pool_id mutually exclusive, as the API does, and
// requires one of them when the connection can only reach its host through a gateway.
func (r *AppConnectionBaseResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	if !r.SupportsGateway {
		return nil
	}

	gatewayPaths := []path.Expression{path.MatchRoot("gateway_id"), path.MatchRoot("gateway_pool_id")}
	if r.RequiresGateway {
		return []resource.ConfigValidator{resourcevalidator.ExactlyOneOf(gatewayPaths...)}
	}
	return []resource.ConfigValidator{resourcevalidator.Conflicting(gatewayPaths...)}
}

// ModifyPlan warns when the configured credentials differ from state on a connection whose
// credentials Infisical manages, because Update won't send them.
func (r *AppConnectionBaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var id types.String
	var planned, current types.Object
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("credentials"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("credentials"), &current)...)
	if resp.Diagnostics.HasError() || planned.Equal(current) || r.client == nil {
		return
	}

	// A failed read here is not worth failing the plan over: Update reads the connection again.
	managed, err := r.credentialsManagedByInfisicalForID(id.ValueString())
	if err != nil || !managed {
		return
	}

	resp.Diagnostics.AddAttributeWarning(
		path.Root("credentials"),
		"Credentials managed by Infisical",
		fmt.Sprintf("Infisical manages the credentials of this %s App Connection (platform-managed credentials or credential rotation), so Terraform won't send the configured credentials. Change them in Infisical instead.", r.AppConnectionName),
	)
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
	plan, gateway, diags := r.readModel(ctx, req.Plan)
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
				GatewayId:                  gatewayFieldForRequest(gateway.Id),
				GatewayPoolId:              gatewayFieldForRequest(gateway.PoolId),
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
	gateway = r.reconcileGateway(gateway, appConnection)

	diags = resp.State.Set(ctx, r.stateValue(plan, gateway))
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
	state, gateway, diags := r.readModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
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

	gateway = r.reconcileGateway(gateway, appConnection)

	// When Infisical manages the credentials, it changes them itself, so a new hash is not a
	// conflict and the configured credentials stay in state. An import has none to keep.
	if credentialsManagedByInfisical(appConnection) && !state.CredentialsHash.IsNull() {
		state.CredentialsHash = types.StringValue(appConnection.CredentialsHash)
	} else if state.CredentialsHash.ValueString() != appConnection.CredentialsHash {
		// An import starts with no hash. The API never returns secret credentials, so there is
		// nothing to compare against and no conflict to warn about.
		if !state.CredentialsHash.IsNull() {
			resp.Diagnostics.AddWarning(
				"App connection credentials conflict",
				fmt.Sprintf("The credentials for the %s App Connection with ID %s have been updated outside of Terraform.", r.AppConnectionName, state.ID.ValueString()),
			)
		}

		// Nulling the credentials makes the next plan send the configured credentials again.
		diags = r.OverwriteCredentialsFields(&state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		state.CredentialsHash = types.StringValue(appConnection.CredentialsHash)
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

	diags = resp.State.Set(ctx, r.stateValue(state, gateway))
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
	plan, gateway, diags := r.readModel(ctx, req.Plan)
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

	managed, err := r.credentialsManagedByInfisicalForID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating app connection",
			"Couldn't read app connection before updating it, unexpected error: "+err.Error(),
		)
		return
	}

	// A nil map leaves credentials out of the request, so Infisical keeps the ones it manages.
	var credentialsMap map[string]any
	if !managed {
		credentialsMap, diags = r.ReadCredentialsForUpdateFromPlan(ctx, plan, state)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	var appConnection infisical.AppConnection
	err = retryAppConnectionOp(ctx, r.IsRetryableError, func() error {
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
				GatewayId:                  gatewayFieldForRequest(gateway.Id),
				GatewayPoolId:              gatewayFieldForRequest(gateway.PoolId),
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
	gateway = r.reconcileGateway(gateway, appConnection)

	diags = resp.State.Set(ctx, r.stateValue(plan, gateway))
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

	// Already gone, which is what the delete wanted.
	if err == infisical.ErrNotFound {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting app connection",
			"Couldn't delete app connection from Infisical, unexpected error: "+err.Error(),
		)
	}
}

// ImportState accepts the app connection ID. Read then fills in everything the API returns. The
// API never returns secret credentials, so the first plan after an import sends the configured
// credentials again.
func (r *AppConnectionBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.ParseUUID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected the ID of an app connection, such as 3f091b42-1661-41a1-b647-98c6a45af047, got: %q", req.ID),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
