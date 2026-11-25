package resource

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.ResourceWithImportState      = &AppConnectionBaseResource{}
	_ resource.ResourceWithConfigValidators = &AppConnectionBaseResource{}
	_ resource.ResourceWithModifyPlan       = &AppConnectionBaseResource{}
)

// credentialsManagedByInfisical reports whether Infisical changes the connection's credentials
// itself: platform-managed credentials or a credential rotation that is turned on
// (backend/src/services/app-connection/app-connection-service.ts in Infisical core). A paused
// rotation can't be detected: GET by ID doesn't return the rotation, only the list endpoint does.
func credentialsManagedByInfisical(appConnection infisical.AppConnection) bool {
	return appConnection.IsPlatformManagedCredentials || appConnection.IsAutoRotationEnabled
}

// readConnection reads the connection before an update, to find out whether Infisical manages
// its credentials and which non-secret credential fields it holds.
func (r *AppConnectionBaseResource) readConnection(id string) (infisical.AppConnection, error) {
	return r.client.GetAppConnectionById(infisical.GetAppConnectionByIdRequest{App: r.App, ID: id})
}

// credentialsDifferingFromInfisical lists the configured credential fields that differ from the
// non-secret credentials Infisical returns. Each attribute is matched to its API key, the name in
// camelCase unless CredentialsAPIKeys says otherwise. Secret fields aren't returned and are
// skipped. Infisical leaves out a non-secret field that has no value, so an absent string counts
// as empty; an absent bool or number is skipped, since Infisical applies its own default to it.
func (r *AppConnectionBaseResource) credentialsDifferingFromInfisical(planned types.Object, apiCredentials map[string]any) []string {
	var differing []string
	for name, value := range planned.Attributes() {
		if value.IsUnknown() {
			continue
		}
		key, mapped := r.CredentialsAPIKeys[name]
		if !mapped {
			key = snakeToCamel(name)
		}
		apiValue, returned := apiCredentials[key]
		if !returned {
			attribute, known := r.CredentialsAttributes[name]
			if known && attribute.IsSensitive() {
				continue
			}
			if _, isString := value.(basetypes.StringValue); !isString {
				continue
			}
			apiValue = nil
		}
		// A removed setting matches only when Infisical has none either.
		if value.IsNull() {
			if apiValue != nil && apiValue != "" {
				differing = append(differing, "credentials."+name)
			}
			continue
		}
		if !credentialValueMatches(value, apiValue) {
			differing = append(differing, "credentials."+name)
		}
	}
	sort.Strings(differing)
	return differing
}

func credentialValueMatches(value attr.Value, apiValue any) bool {
	switch v := value.(type) {
	case basetypes.StringValue:
		s, ok := apiValue.(string)
		return ok && strings.TrimSpace(s) == strings.TrimSpace(v.ValueString())
	case basetypes.BoolValue:
		b, ok := apiValue.(bool)
		return ok && b == v.ValueBool()
	case basetypes.Int32Value:
		n, ok := apiValue.(float64)
		return ok && int32(n) == v.ValueInt32()
	case basetypes.Int64Value:
		n, ok := apiValue.(float64)
		return ok && int64(n) == v.ValueInt64()
	case basetypes.Float64Value:
		n, ok := apiValue.(float64)
		return ok && n == v.ValueFloat64()
	}
	// A type this comparison doesn't handle never counts as a match.
	return false
}

func snakeToCamel(name string) string {
	parts := strings.Split(name, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func credentialsMismatchDetail(appConnectionName string, appConnection infisical.AppConnection, differing []string) string {
	mismatch := fmt.Sprintf("The configured %s doesn't match what Infisical has. Set it to the value in Infisical.", differing[0])
	if len(differing) > 1 {
		mismatch = fmt.Sprintf("These configured fields don't match what Infisical has: %s. Set them to the values in Infisical.", strings.Join(differing, ", "))
	}
	if appConnection.IsPlatformManagedCredentials {
		return fmt.Sprintf("Infisical manages the credentials of this %s App Connection (platform-managed credentials) and rejects any change to them, so Terraform doesn't send them. %s To use different credentials, create a new connection: run `terraform taint` on this resource, then apply. `terraform apply -replace` stops at this check too.", appConnectionName, mismatch)
	}
	return fmt.Sprintf("Infisical rotates the credentials of this %s App Connection, so after an import or a rotation Terraform records the configured credentials without sending them. %s After that apply, Terraform sends changes to the credentials again.", appConnectionName, mismatch)
}

// AppConnectionBaseResource is the resource implementation.
type AppConnectionBaseResource struct {
	App                   infisical.AppConnectionApp // used for identifying secret sync route
	ResourceTypeName      string                     // terraform resource name suffix
	AppConnectionName     string                     // complete descriptive name of the app connection
	DocsURL               string                     // optional Infisical guide linked from the resource description
	SupportsGateway       bool                       // when true, exposes gateway_id and gateway_pool_id and sends them to the API
	RequiresGateway       bool                       // when true, one of gateway_id or gateway_pool_id is required
	client                *infisical.Client
	AllowedMethods        []string
	CredentialsAttributes map[string]schema.Attribute
	// API keys of credential attributes whose names don't follow snake_case to camelCase, for
	// comparing the configuration with the credentials Infisical returns.
	CredentialsAPIKeys               map[string]string
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

// ModifyPlan handles configured credentials that differ from state on a connection whose
// credentials Infisical manages. Update never sends them, so when state has none (after an
// import, or after Infisical changed them) the plan warns and the apply records the configured
// ones; a change to credentials already in state is refused, because it would never reach
// Infisical.
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

	// A replacement creates a new connection, which takes the configured credentials. A value
	// known only at apply is checked by Update instead.
	replacing, diags := planReplacesConnection(ctx, req)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || replacing || hasUnknownAttribute(planned) {
		return
	}

	// A failed read here is not worth failing the plan over: Update reads the connection again.
	appConnection, err := r.readConnection(id.ValueString())
	if err != nil {
		return
	}
	if r.needsCurrentPassword(appConnection, planned, current) {
		resp.Diagnostics.AddAttributeError(path.Root("credentials"), "Credentials managed by Infisical", rotationNeedsPasswordDetail(r.AppConnectionName))
		return
	}
	if !recordsCredentialsWithoutSending(appConnection, current) {
		return
	}

	if differing := r.credentialsDifferingFromInfisical(planned, appConnection.Credentials); len(differing) > 0 {
		resp.Diagnostics.AddAttributeError(path.Root("credentials"), "Credentials managed by Infisical", credentialsMismatchDetail(r.AppConnectionName, appConnection, differing))
		return
	}
	resp.Diagnostics.AddAttributeWarning(
		path.Root("credentials"),
		"Credentials managed by Infisical",
		fmt.Sprintf("Infisical manages the credentials of this %s App Connection (%s), so Terraform records the configured credentials without sending them.", r.AppConnectionName, credentialsManager(appConnection)),
	)
}

// recordsCredentialsWithoutSending reports whether an update records the configured credentials
// in state without sending them. Infisical rejects any credentials sent to a platform-managed
// connection. With rotation on, it accepts them, but after an import or a rotation, when state
// holds no credentials, the configured ones are likely stale and would replace the ones Infisical
// rotated; once state holds them, a change the user makes is sent.
func recordsCredentialsWithoutSending(appConnection infisical.AppConnection, stateCredentials types.Object) bool {
	if appConnection.IsPlatformManagedCredentials {
		return true
	}
	return appConnection.IsAutoRotationEnabled && credentialsUnset(stateCredentials)
}

// needsCurrentPassword reports whether a credentials change on a connection with rotation on would
// send a password Terraform can't vouch for. The API takes the credentials as one object, so a
// change to a setting such as a certificate sends the configured password too, and Infisical may
// have rotated it since. A change that sets a secret field is the user's own, and is sent.
func (r *AppConnectionBaseResource) needsCurrentPassword(appConnection infisical.AppConnection, planned, current types.Object) bool {
	if !appConnection.IsAutoRotationEnabled || credentialsUnset(current) || planned.Equal(current) {
		return false
	}
	plannedAttributes, currentAttributes := planned.Attributes(), current.Attributes()
	for name, attribute := range r.CredentialsAttributes {
		if attribute.IsSensitive() && !plannedAttributes[name].Equal(currentAttributes[name]) {
			return false
		}
	}
	return true
}

func rotationNeedsPasswordDetail(appConnectionName string) string {
	return fmt.Sprintf("Infisical rotates the credentials of this %s App Connection. Changing them sends the password too, and the configured one may no longer be current. Set the current password in the configuration along with this change.", appConnectionName)
}

func credentialsManager(appConnection infisical.AppConnection) string {
	if appConnection.IsPlatformManagedCredentials {
		return "platform-managed credentials"
	}
	return "credential rotation"
}

// planReplacesConnection reports whether the plan replaces the connection: method and project_id
// use RequiresReplace, and an unknown value there replaces it too. The framework doesn't pass
// attribute-level replacements to ModifyPlan, so they are compared here.
func planReplacesConnection(ctx context.Context, req resource.ModifyPlanRequest) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	for _, attribute := range []string{"method", "project_id"} {
		var planned, current types.String
		diags.Append(req.Plan.GetAttribute(ctx, path.Root(attribute), &planned)...)
		diags.Append(req.State.GetAttribute(ctx, path.Root(attribute), &current)...)
		if diags.HasError() {
			return false, diags
		}
		if planned.IsUnknown() || !planned.Equal(current) {
			return true, diags
		}
	}
	return false, diags
}

func hasUnknownAttribute(object types.Object) bool {
	if object.IsUnknown() {
		return true
	}
	for _, value := range object.Attributes() {
		if value.IsUnknown() {
			return true
		}
	}
	return false
}

// credentialsUnset reports whether state holds no credentials: none after an import, or all
// fields nulled by OverwriteCredentialsFields after the hash changed.
func credentialsUnset(credentials types.Object) bool {
	if credentials.IsNull() || credentials.IsUnknown() {
		return true
	}
	for _, value := range credentials.Attributes() {
		if !value.IsNull() {
			return false
		}
	}
	return true
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

	if state.CredentialsHash.ValueString() != appConnection.CredentialsHash {
		// An import starts with no hash. The API never returns secret credentials, so there is
		// nothing to compare against and no conflict to warn about. When Infisical manages the
		// credentials, it changes them itself, so a new hash is not a conflict either.
		if !state.CredentialsHash.IsNull() && !credentialsManagedByInfisical(appConnection) {
			resp.Diagnostics.AddWarning(
				"App connection credentials conflict",
				fmt.Sprintf("The credentials for the %s App Connection with ID %s have been updated outside of Terraform.", r.AppConnectionName, state.ID.ValueString()),
			)
		}

		// Nulling the credentials makes the next apply take the configured credentials again:
		// Update sends them, or records them without sending when Infisical manages them.
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

	current, err := r.readConnection(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating app connection",
			"Couldn't read app connection before updating it, unexpected error: "+err.Error(),
		)
		return
	}
	// With rotation on and credentials in state, only a change the user made is sent: resending
	// unchanged credentials would replace the ones Infisical rotated.
	skipCredentials := recordsCredentialsWithoutSending(current, state.Credentials) ||
		(current.IsAutoRotationEnabled && plan.Credentials.Equal(state.Credentials))

	// ModifyPlan refuses these when it can; a credential known only at apply reaches here.
	if r.needsCurrentPassword(current, plan.Credentials, state.Credentials) {
		resp.Diagnostics.AddAttributeError(path.Root("credentials"), "Credentials managed by Infisical", rotationNeedsPasswordDetail(r.AppConnectionName))
		return
	}
	if recordsCredentialsWithoutSending(current, state.Credentials) {
		if differing := r.credentialsDifferingFromInfisical(plan.Credentials, current.Credentials); len(differing) > 0 {
			resp.Diagnostics.AddAttributeError(path.Root("credentials"), "Credentials managed by Infisical", credentialsMismatchDetail(r.AppConnectionName, current, differing))
			return
		}
	}

	// A nil map leaves credentials out of the request, so Infisical keeps the ones it has.
	var credentialsMap map[string]any
	if !skipCredentials {
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
