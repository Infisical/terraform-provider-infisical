package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &AppConnectionGcpResource{}
)

// NewAppConnectionGcpResource is a helper function to simplify the provider implementation.
func NewAppConnectionGcpResource() resource.Resource {
	return &AppConnectionGcpResource{}
}

// AppConnectionGcp is the resource implementation.
type AppConnectionGcpResource struct {
	client *infisical.Client
}

// AppConnectionGcpResourceModel describes the data source data model.
type AppConnectionGcpResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Method              types.String `tfsdk:"method"`
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	CredentialsHash     types.String `tfsdk:"credentials_hash"`
}

// Metadata returns the resource type name.
func (r *AppConnectionGcpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_connection_gcp"
}

// Schema defines the schema for the resource.
func (r *AppConnectionGcpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage GCP App Connection",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the app connection",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"method": schema.StringAttribute{
				Required:    true,
				Description: "The method used to authenticate with GCP. Possible values are: service-account-impersonation",
			},
			"service_account_email": schema.StringAttribute{
				Optional:    true,
				Description: "The service account email to connect with GCP. The service account ID (the part of the email before '@') must be suffixed with the first two sections of your organization ID e.g. service-account-df92581a-0fe9@my-project.iam.gserviceaccount.com. For more details, refer to the documentation here https://infisical.com/docs/integrations/app-connections/gcp#configure-service-account-for-infisical",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the GCP App Connection to create. Must be slug-friendly",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "An optional description for the GCP App Connection.",
			},
			"credentials_hash": schema.StringAttribute{
				Computed:    true,
				Description: "The hash of the GCP App Connection credentials",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *AppConnectionGcpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *AppConnectionGcpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create GCP app connection",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan AppConnectionGcpResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Method.ValueString() != string(infisicalclient.AppConnectionGcpMethodServiceAccountImpersonation) {
		resp.Diagnostics.AddError(
			"Unable to create GCP app connection",
			"Invalid value for method field. Possible values are: service-account-impersonation",
		)
		return
	}

	if plan.ServiceAccountEmail.IsNull() || plan.ServiceAccountEmail.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Unable to create GCP app connection",
			"Service account email field must be defined",
		)
		return
	}

	credentialsMap := map[string]interface{}{}
	credentialsMap["serviceAccountEmail"] = plan.ServiceAccountEmail.ValueString()

	appConnection, err := r.client.CreateAppConnection(infisicalclient.CreateAppConnectionRequest{
		App:         infisicalclient.AppConnectionAppGCP,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Method:      plan.Method.ValueString(),
		Credentials: credentialsMap,
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
func (r *AppConnectionGcpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read GCP app connection",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state AppConnectionGcpResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	appConnection, err := r.client.GetAppConnectionById(infisicalclient.GetAppConnectionByIdRequest{
		App: infisicalclient.AppConnectionAppGCP,
		ID:  state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading GCP app connection",
				"Couldn't read app connection, unexpected error: "+err.Error(),
			)
			return
		}
	}

	if state.CredentialsHash.ValueString() != appConnection.CredentialsHash {
		resp.Diagnostics.AddWarning(
			"App connection credentials conflict",
			fmt.Sprintf("The credentials for the GCP app connection with ID %s have been updated outside of Terraform.", state.ID.ValueString()),
		)

		state.ServiceAccountEmail = types.StringNull()
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		return
	}

	if !(state.Description.IsNull() && appConnection.Description == "") {
		state.Description = types.StringValue(appConnection.Description)
	}

	state.Method = types.StringValue(appConnection.Method)
	state.Name = types.StringValue(appConnection.Name)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *AppConnectionGcpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update GCP app connection",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan AppConnectionGcpResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AppConnectionGcpResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Method.ValueString() != string(infisicalclient.AppConnectionGcpMethodServiceAccountImpersonation) {
		resp.Diagnostics.AddError(
			"Unable to update GCP app connection",
			"Invalid value for method field. Possible values are: service-account-impersonation",
		)
		return
	}

	if plan.ServiceAccountEmail.IsNull() || plan.ServiceAccountEmail.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Unable to update GCP app connection",
			"Service account email field must be defined",
		)
		return
	}

	credentialsMap := map[string]interface{}{}
	if state.ServiceAccountEmail.ValueString() != plan.ServiceAccountEmail.ValueString() {
		credentialsMap["serviceAccountEmail"] = plan.ServiceAccountEmail.ValueString()
	}

	appConnection, err := r.client.UpdateAppConnection(infisicalclient.UpdateAppConnectionRequest{
		ID:          state.ID.ValueString(),
		App:         infisicalclient.AppConnectionAppGCP,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Method:      plan.Method.ValueString(),
		Credentials: credentialsMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating GCP app connection",
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

// Delete deletes the resource and removes the Terraform state on success.
func (r *AppConnectionGcpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete GCP app connection",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state AppConnectionGcpResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteAppConnection(infisical.DeleteAppConnectionRequest{
		App: infisical.AppConnectionAppGCP,
		ID:  state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting GCP app connection",
			"Couldn't delete app connection from Infisical, unexpected error: "+err.Error(),
		)
	}
}
