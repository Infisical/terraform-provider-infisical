package resource

import (
	"context"
	"fmt"
	"strconv"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityKubernetesAuthResource is a helper function to simplify the provider implementation.
func NewIdentityKubernetesAuthResource() resource.Resource {
	return &IdentityKubernetesAuthResource{}
}

// IdentityKubernetesAuthResource is the resource implementation.
type IdentityKubernetesAuthResource struct {
	client *infisical.Client
}

// IdentityKubernetesAuthResourceSourceModel describes the data source data model.
type IdentityKubernetesAuthResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	IdentityID                 types.String `tfsdk:"identity_id"`
	KubernetesHost             types.String `tfsdk:"kubernetes_host"`
	CaCertificate              types.String `tfsdk:"kubernetes_ca_certificate"`
	TokenReviewerJWT           types.String `tfsdk:"token_reviewer_jwt"`
	AllowedServiceAccountNames types.String `tfsdk:"allowed_service_account_names"`
	AllowedAudience            types.String `tfsdk:"allowed_audience"`
	AllowedNamespaces          types.String `tfsdk:"allowed_namespaces"`
	AccessTokenTTL             types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL          types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit    types.Int64  `tfsdk:"access_token_num_uses_limit"`
	AccessTokenTrustedIps      types.List   `tfsdk:"access_token_trusted_ips"`
}

type IdentityKubernetesAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

// Metadata returns the resource type name.
func (r *IdentityKubernetesAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_kubernetes_auth"
}

// Schema defines the schema for the resource.
func (r *IdentityKubernetesAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity kubernetes auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the kubernetes auth",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"kubernetes_host": schema.StringAttribute{
				Description: "The host string, host:port pair, or URL to the base of the Kubernetes API server. This can usually be obtained by running `kubectl cluster-info`.",
				Required:    true,
			},
			"token_reviewer_jwt": schema.StringAttribute{
				Description:         "A long-lived service account JWT token for Infisical to access the TokenReview API to validate other service account JWT tokens submitted by applications/pods. This is the JWT token obtained from step 1.5.",
				MarkdownDescription: "A long-lived service account JWT token for Infisical to access the [TokenReview API](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-review-v1/) to validate other service account JWT tokens submitted by applications/pods. This is the JWT token obtained from step 1.5.",
				Required:            true,
			},
			"kubernetes_ca_certificate": schema.StringAttribute{
				Description:         "The PEM-encoded CA cert for the Kubernetes API server. This is used by the TLS client for secure communication with the Kubernetes API server.",
				MarkdownDescription: "The PEM-encoded CA cert for the Kubernetes API server. This is used by the TLS client for secure communication with the Kubernetes API server.",
				Optional:            true,
				Computed:            true,
			},
			"allowed_service_account_names": schema.StringAttribute{
				Description: "A comma-separated list of trusted service account names that are allowed to authenticate with Infisical.",
				Optional:    true,
				Computed:    true,
			},
			"allowed_audience": schema.StringAttribute{
				Description: "An optional audience claim that the service account JWT token must have to authenticate with Infisical.",
				Optional:    true,
				Computed:    true,
			},
			"allowed_namespaces": schema.StringAttribute{
				Description: "A comma-separated list of trusted namespaces that service accounts must belong to authenticate with Infisical.",
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
							Required: true,
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
func (r *IdentityKubernetesAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func updateKubernetesAuthStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityKubernetesAuthResourceModel, newIdentityKubernetesAuth *infisicalclient.IdentityKubernetesAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityKubernetesAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityKubernetesAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityKubernetesAuth.AccessTokenNumUsesLimit)
	plan.AllowedAudience = types.StringValue(newIdentityKubernetesAuth.AllowedAudience)
	plan.AllowedServiceAccountNames = types.StringValue(newIdentityKubernetesAuth.AllowedServiceAccountNames)
	plan.AllowedNamespaces = types.StringValue(newIdentityKubernetesAuth.AllowedNamespaces)
	plan.CaCertificate = types.StringValue(newIdentityKubernetesAuth.CACERT)

	planAccessTokenTrustedIps := make([]IdentityKubernetesAuthResourceTrustedIps, len(newIdentityKubernetesAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityKubernetesAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityKubernetesAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityKubernetesAuthResourceTrustedIps{IpAddress: types.StringValue(
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

	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AccessTokenTrustedIps = stateAccessTokenTrustedIps
}

// Create creates the resource and sets the initial Terraform state.
func (r *IdentityKubernetesAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to create identity kubernetes auth",
			"Only Machine IdentityKubernetesAuth authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityKubernetesAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	newIdentityKubernetesAuth, err := r.client.CreateIdentityKubernetesAuth(infisical.CreateIdentityKubernetesAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		KubernetesHost:          plan.KubernetesHost.ValueString(),
		CACERT:                  plan.CaCertificate.ValueString(),
		TokenReviewerJwt:        plan.TokenReviewerJWT.ValueString(),
		AllowedNamespaces:       plan.AllowedNamespaces.ValueString(),
		AllowedNames:            plan.AllowedServiceAccountNames.ValueString(),
		AllowedAudience:         plan.AllowedAudience.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity kubernetes auth",
			"Couldn't save kubernetes auth to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityKubernetesAuth.ID)
	updateKubernetesAuthStateByApi(ctx, resp.Diagnostics, &plan, &newIdentityKubernetesAuth)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityKubernetesAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to read identity kubernetes auth role",
			"Only Machine IdentityKubernetesAuth authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityKubernetesAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityKubernetesAuth, err := r.client.GetIdentityKubernetesAuth(infisical.GetIdentityKubernetesAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity kubernetes auth",
				"Couldn't read identity kubernetes auth from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateKubernetesAuthStateByApi(ctx, resp.Diagnostics, &state, &identityKubernetesAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityKubernetesAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to update identity kubernetes auth",
			"Only Machine IdentityKubernetesAuth authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityKubernetesAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityKubernetesAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	updatedIdentityKubernetesAuth, err := r.client.UpdateIdentityKubernetesAuth(infisical.UpdateIdentityKubernetesAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		KubernetesHost:          plan.KubernetesHost.ValueString(),
		CACERT:                  plan.CaCertificate.ValueString(),
		TokenReviewerJwt:        plan.TokenReviewerJWT.ValueString(),
		AllowedNamespaces:       plan.AllowedNamespaces.ValueString(),
		AllowedNames:            plan.AllowedServiceAccountNames.ValueString(),
		AllowedAudience:         plan.AllowedAudience.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity kubernetes auth",
			"Couldn't update identity kubernetes auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	updateKubernetesAuthStateByApi(ctx, resp.Diagnostics, &plan, &updatedIdentityKubernetesAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityKubernetesAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to delete identity kubernetes auth",
			"Only Machine IdentityKubernetesAuth authentication is supported for this operation",
		)
		return
	}

	var state IdentityKubernetesAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityKubernetesAuth(infisical.RevokeIdentityKubernetesAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity kubernetes auth",
			"Couldn't delete identity kubernetes auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
