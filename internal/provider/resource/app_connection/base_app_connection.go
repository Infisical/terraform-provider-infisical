package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

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
	client                           *infisical.Client
	AllowedMethods                   []string
	CredentialsAttributes            map[string]schema.Attribute
	ReadCredentialsForCreateFromPlan func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics)
	ReadCredentialsForUpdateFromPlan func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics)
	OverwriteCredentialsFields       func(state *AppConnectionBaseResourceModel) diag.Diagnostics
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

	appConnection, err := r.client.CreateAppConnection(infisical.CreateAppConnectionRequest{
		App:         r.App,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Method:      plan.Method.ValueString(),
		Credentials: credentialsMap,
		ProjectId:   plan.ProjectId.ValueString(),
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

	appConnection, err := r.client.UpdateAppConnection(infisical.UpdateAppConnectionRequest{
		ID:          state.ID.ValueString(),
		App:         r.App,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Method:      plan.Method.ValueString(),
		Credentials: credentialsMap,
		ProjectId:   plan.ProjectId.ValueString(),
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
