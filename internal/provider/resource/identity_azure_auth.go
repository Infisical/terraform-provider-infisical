package resource

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"
	infisicalstrings "terraform-provider-infisical/internal/pkg/strings"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityAzureAuthResource is a helper function to simplify the provider implementation.
func NewIdentityAzureAuthResource() resource.Resource {
	return &IdentityAzureAuthResource{}
}

// IdentityAzureAuthResource is the resource implementation.
type IdentityAzureAuthResource struct {
	client *infisical.Client
}

// IdentityAzureAuthResourceSourceModel describes the data source data model.
type IdentityAzureAuthResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	IdentityID                 types.String `tfsdk:"identity_id"`
	TenantID                   types.String `tfsdk:"tenant_id"`
	Resource                   types.String `tfsdk:"resource_url"`
	AllowedServicePrincipalIDs types.List   `tfsdk:"allowed_service_principal_ids"`
	AccessTokenTrustedIps      types.List   `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL             types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL          types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit    types.Int64  `tfsdk:"access_token_num_uses_limit"`
}

type IdentityAzureAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

// Metadata returns the resource type name.
func (r *IdentityAzureAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_azure_auth"
}

// Schema defines the schema for the resource.
func (r *IdentityAzureAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity azure auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the azure auth",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tenant_id": schema.StringAttribute{
				Description:         " The tenant ID for the Azure AD organization.",
				MarkdownDescription: " The [tenant ID](https://learn.microsoft.com/en-us/entra/fundamentals/how-to-find-tenant) for the Azure AD organization.",
				Required:            true,
			},
			"resource_url": schema.StringAttribute{
				Description:         "The resource URL for the application registered in Azure AD. The value is expected to match the `aud` claim of the access token JWT later used in the login operation against Infisical. Defaault: https://management.azure.com",
				MarkdownDescription: "The resource URL for the application registered in Azure AD. The value is expected to match the `aud` claim of the access token JWT later used in the login operation against Infisical. See the [resource](https://learn.microsoft.com/en-us/entra/identity/managed-identities-azure-resources/how-to-use-vm-token#get-a-token-using-http) parameter for how the audience is set when requesting a JWT access token from the Azure Instance Metadata Service (IMDS) endpoint. In most cases, this value should be `https://management.azure.com/` which is the default",
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("https://management.azure.com"),
			},
			"allowed_service_principal_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of Azure AD service principal IDs that are allowed to authenticate with Infisical",
				Optional:    true,
				Computed:    true,
			},
			"access_token_trusted_ips": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A list of IPs or CIDR ranges that access tokens can be used from. You can use 0.0.0.0/0, to allow usage from any network address..",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"access_token_ttl": schema.Int64Attribute{
				Description: "The lifetime for an access token in seconds. This value will be referenced at renewal time. Default: 2592000",
				Computed:    true,
				Optional:    true,
			},
			"access_token_max_ttl": schema.Int64Attribute{
				Description: "The maximum lifetime for an access token in seconds. This value will be referenced at renewal time. Default: 2592000",
				Computed:    true,
				Optional:    true,
			},
			"access_token_num_uses_limit": schema.Int64Attribute{
				Description: "The maximum number of times that an access token can be used; a value of 0 implies infinite number of uses. Default:0",
				Computed:    true,
				Optional:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IdentityAzureAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func updateAzureAuthTerraformStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityAzureAuthResourceModel, newIdentityAzureAuth *infisicalclient.IdentityAzureAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityAzureAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityAzureAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityAzureAuth.AccessTokenNumUsesLimit)
	plan.TenantID = types.StringValue(newIdentityAzureAuth.TenantID)
	plan.Resource = types.StringValue(newIdentityAzureAuth.Resource)

	planAccessTokenTrustedIps := make([]IdentityAzureAuthResourceTrustedIps, len(newIdentityAzureAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityAzureAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityAzureAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityAzureAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress,
			)}
		}
	}

	stateAccessTokenTrustedIps, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ip_address": types.StringType,
		},
	}, planAccessTokenTrustedIps)
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AllowedServicePrincipalIDs, diags = types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(newIdentityAzureAuth.AllowedServicePrincipalIDS, ","))
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AccessTokenTrustedIps = stateAccessTokenTrustedIps
}

// Create creates the resource and sets the initial Terraform state.
func (r *IdentityAzureAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity azure auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityAzureAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowedServicePrincipalIds := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedServicePrincipalIDs)
	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	newIdentityAzureAuth, err := r.client.CreateIdentityAzureAuth(infisical.CreateIdentityAzureAuthRequest{
		IdentityID:                 plan.IdentityID.ValueString(),
		AccessTokenTTL:             plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:          plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit:    plan.AccessTokenNumUsesLimit.ValueInt64(),
		TenantID:                   plan.TenantID.ValueString(),
		AllowedServicePrincipalIDS: strings.Join(allowedServicePrincipalIds, ","),
		Resource:                   plan.Resource.ValueString(),
		AccessTokenTrustedIPS:      accessTokenTrustedIps,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity azure auth",
			"Couldn't save azure auth to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityAzureAuth.ID)
	updateAzureAuthTerraformStateByApi(ctx, resp.Diagnostics, &plan, &newIdentityAzureAuth)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityAzureAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read identity azure auth role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityAzureAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityAzureAuth, err := r.client.GetIdentityAzureAuth(infisical.GetIdentityAzureAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity azure auth",
				"Couldn't read identity azure auth from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateAzureAuthTerraformStateByApi(ctx, resp.Diagnostics, &state, &identityAzureAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityAzureAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity azure auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityAzureAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityAzureAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	allowedServicePrincipalIds := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedServicePrincipalIDs)

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	updatedIdentityAzureAuth, err := r.client.UpdateIdentityAzureAuth(infisical.UpdateIdentityAzureAuthRequest{
		IdentityID:                 plan.IdentityID.ValueString(),
		AccessTokenTrustedIPS:      accessTokenTrustedIps,
		AccessTokenTTL:             plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:          plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit:    plan.AccessTokenNumUsesLimit.ValueInt64(),
		TenantID:                   plan.TenantID.ValueString(),
		AllowedServicePrincipalIDS: strings.Join(allowedServicePrincipalIds, ","),
		Resource:                   plan.Resource.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity azure auth",
			"Couldn't update identity azure auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	updateAzureAuthTerraformStateByApi(ctx, resp.Diagnostics, &plan, &updatedIdentityAzureAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityAzureAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity azure auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityAzureAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityAzureAuth(infisical.RevokeIdentityAzureAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity azure auth",
			"Couldn't delete identity azure auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *IdentityAzureAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import identity azure auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	identity, err := r.client.GetIdentity(infisical.GetIdentityRequest{
		IdentityID: req.ID,
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.Diagnostics.AddError(
				"Identity not found",
				"The identity with the given ID was not found",
			)
		} else {
			resp.Diagnostics.AddError(
				"Error importing identity azure auth",
				"Couldn't read identity azure auth from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	if len(identity.Identity.AuthMethods) == 0 {
		resp.Diagnostics.AddError(
			"Identity azure auth not found",
			"The identity with the given ID has no configured auth methods",
		)
		return
	}

	hasAzureAuth := slices.Contains(identity.Identity.AuthMethods, "azure-auth")

	if !hasAzureAuth {
		resp.Diagnostics.AddError(
			"Identity azure auth not found",
			"The identity with the given ID does not have azure auth configured",
		)
		return
	}

	identityAzureAuth, err := r.client.GetIdentityAzureAuth(infisical.GetIdentityAzureAuthRequest{
		IdentityID: req.ID,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing identity azure auth",
			"Couldn't read identity azure auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	var diags diag.Diagnostics
	var state IdentityAzureAuthResourceModel

	state.ID = types.StringValue(identityAzureAuth.ID)
	state.IdentityID = types.StringValue(identityAzureAuth.IdentityID)
	state.Resource = types.StringValue(identityAzureAuth.Resource)
	state.TenantID = types.StringValue(identityAzureAuth.TenantID)
	state.AccessTokenTTL = types.Int64Value(identityAzureAuth.AccessTokenTTL)
	state.AccessTokenMaxTTL = types.Int64Value(identityAzureAuth.AccessTokenMaxTTL)
	state.AccessTokenNumUsesLimit = types.Int64Value(identityAzureAuth.AccessTokenNumUsesLimit)

	accessTokenTrustedIps := make([]IdentityAzureAuthResourceTrustedIps, len(identityAzureAuth.AccessTokenTrustedIPS))
	for i, el := range identityAzureAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			accessTokenTrustedIps[i] = IdentityAzureAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			accessTokenTrustedIps[i] = IdentityAzureAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress,
			)}
		}
	}

	state.AccessTokenTrustedIps, diags = types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ip_address": types.StringType,
		},
	}, accessTokenTrustedIps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.AllowedServicePrincipalIDs, diags = types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(identityAzureAuth.AllowedServicePrincipalIDS, ","))

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}
