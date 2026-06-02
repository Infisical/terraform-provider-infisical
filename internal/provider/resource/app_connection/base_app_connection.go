package resource

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AppConnectionBaseResource is the resource implementation.
type AppConnectionBaseResource struct {
	App                              infisical.AppConnectionApp // used for identifying secret sync route
	ResourceTypeName                 string                     // terraform resource name suffix
	AppConnectionName                string                     // complete descriptive name of the app connection
	SupportsGateway                  bool                       // when true, exposes gateway_id on the resource
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
	GatewayId       types.String `tfsdk:"gateway_id"`
}

// gatewayIdForRequest returns the gateway_id value to send to the API.
// It returns a non-nil pointer only when the resource supports gateways and a
// value is set. When the gateway has been removed from the configuration it
// returns nil, which serializes to an explicit `null` so the API resets the
// connection back to the Internet Gateway (omitting the field would leave the
// existing gateway untouched).
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
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s App Connection", r.AppConnectionName),
		Attributes: map[string]schema.Attribute{
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
			"gateway_id": schema.StringAttribute{
				Optional:    true,
				Description: "The Gateway ID to use for the app connection. If not specified, the Internet Gateway will be used.",
			},
		},
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
	var plan AppConnectionBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
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
		appConnection, apiErr = r.client.CreateAppConnection(infisical.CreateAppConnectionRequest{
			App:         r.App,
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Method:      plan.Method.ValueString(),
			Credentials: credentialsMap,
			ProjectId:   plan.ProjectId.ValueString(),
			GatewayId:   r.gatewayIdForRequest(plan.GatewayId),
		})
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

	diags = resp.State.Set(ctx, plan)
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
	var state AppConnectionBaseResourceModel
	diags := req.State.Get(ctx, &state)
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

		diags = resp.State.Set(ctx, state)
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

	if r.SupportsGateway {
		if appConnection.GatewayId != nil {
			state.GatewayId = types.StringValue(*appConnection.GatewayId)
		} else {
			state.GatewayId = types.StringNull()
		}
	}

	state.Method = types.StringValue(appConnection.Method)
	state.Name = types.StringValue(appConnection.Name)

	diags = resp.State.Set(ctx, state)
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
	var plan AppConnectionBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AppConnectionBaseResourceModel
	diags = req.State.Get(ctx, &state)
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
		appConnection, apiErr = r.client.UpdateAppConnection(infisical.UpdateAppConnectionRequest{
			ID:          state.ID.ValueString(),
			App:         r.App,
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Method:      plan.Method.ValueString(),
			Credentials: credentialsMap,
			ProjectId:   plan.ProjectId.ValueString(),
			GatewayId:   r.gatewayIdForRequest(plan.GatewayId),
		})
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

	diags = resp.State.Set(ctx, plan)
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

	var state AppConnectionBaseResourceModel
	diags := req.State.Get(ctx, &state)
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
