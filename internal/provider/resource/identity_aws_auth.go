package resource

import (
	"context"
	"fmt"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityAwsAuthResource is a helper function to simplify the provider implementation.
func NewIdentityAwsAuthResource() resource.Resource {
	return &IdentityAwsAuthResource{}
}

// IdentityAwsAuthResource is the resource implementation.
type IdentityAwsAuthResource struct {
	client *infisical.Client
}

// IdentityAwsAuthResourceSourceModel describes the data source data model.
type IdentityAwsAuthResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	IdentityID              types.String `tfsdk:"identity_id"`
	StsEndpoint             types.String `tfsdk:"sts_endpoint"`
	AllowedAccountIDs       types.List   `tfsdk:"allowed_account_ids"`
	AllowedPrincipalArns    types.List   `tfsdk:"allowed_principal_arns"`
	AccessTokenTrustedIps   types.List   `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL          types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL       types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit types.Int64  `tfsdk:"access_token_num_uses_limit"`
}

type IdentityAwsAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

// Metadata returns the resource type name.
func (r *IdentityAwsAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_aws_auth"
}

// Schema defines the schema for the resource.
func (r *IdentityAwsAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity aws auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the aws auth",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"sts_endpoint": schema.StringAttribute{
				Description:         "The endpoint URL for the AWS STS API. This value should be adjusted based on the AWS region you are operating in",
				MarkdownDescription: " The endpoint URL for the AWS STS API. This value should be adjusted based on the AWS region you are operating in (e.g. `https://sts.us-east-1.amazonaws.com/`); refer to the list of regional STS endpoints [here](https://docs.aws.amazon.com/general/latest/gr/sts.html).",
				Optional:            true,
				Computed:            true,
			},
			"allowed_account_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of trusted AWS account IDs that are allowed to authenticate with Infisical.",
				Optional:    true,
				Computed:    true,
			},
			"allowed_principal_arns": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of trusted IAM principal ARNs that are allowed to authenticate with Infisical. The values should take one of three forms: `arn:aws:iam::123456789012:user/MyUserName`, `arn:aws:iam::123456789012:role/MyRoleName`, or `arn:aws:iam::123456789012:*`. Using a wildcard in this case allows any IAM principal in the account `123456789012` to authenticate with Infisical under the identity",
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
func (r *IdentityAwsAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func updateAwsAuthTerraformStateFromApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityAwsAuthResourceModel, newIdentityAwsAuth *infisicalclient.IdentityAwsAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityAwsAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityAwsAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityAwsAuth.AccessTokenNumUsesLimit)

	planAccessTokenTrustedIps := make([]IdentityAwsAuthResourceTrustedIps, len(newIdentityAwsAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityAwsAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityAwsAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityAwsAuthResourceTrustedIps{IpAddress: types.StringValue(
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

	plan.AllowedPrincipalArns, diags = types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(newIdentityAwsAuth.AllowedPrincipalArns, ","))
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AllowedAccountIDs, diags = types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(newIdentityAwsAuth.AllowedAccountIDS, ","))
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AccessTokenTrustedIps = stateAccessTokenTrustedIps
	plan.StsEndpoint = types.StringValue(newIdentityAwsAuth.StsEndpoint)
}

// Create creates the resource and sets the initial Terraform state.
func (r *IdentityAwsAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to create identity aws auth",
			"Only Machine IdentityAwsAuth authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityAwsAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)

	allowedPrincipalArns := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedPrincipalArns)
	allowedAccoundIds := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedAccountIDs)

	newIdentityAwsAuth, err := r.client.CreateIdentityAwsAuth(infisical.CreateIdentityAwsAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		StsEndpoint:             plan.StsEndpoint.ValueString(),
		AllowedAccountIDS:       strings.Join(allowedAccoundIds, ","),
		AllowedPrincipalArns:    strings.Join(allowedPrincipalArns, ","),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity aws auth",
			"Couldn't save tag to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityAwsAuth.ID)
	updateAwsAuthTerraformStateFromApi(ctx, resp.Diagnostics, &plan, &newIdentityAwsAuth)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityAwsAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to read identity aws auth role",
			"Only Machine IdentityAwsAuth authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityAwsAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityAwsAuth, err := r.client.GetIdentityAwsAuth(infisical.GetIdentityAwsAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity aws auth",
				"Couldn't read identity aws auth from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateAwsAuthTerraformStateFromApi(ctx, resp.Diagnostics, &state, &identityAwsAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityAwsAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to update identity aws auth",
			"Only Machine IdentityAwsAuth authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityAwsAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityAwsAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	allowedPrincipalArns := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedPrincipalArns)
	allowedAccoundIds := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedAccountIDs)

	updatedIdentityAwsAuth, err := r.client.UpdateIdentityAwsAuth(infisical.UpdateIdentityAwsAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		StsEndpoint:             plan.StsEndpoint.ValueString(),
		AllowedAccountIDS:       strings.Join(allowedAccoundIds, ","),
		AllowedPrincipalArns:    strings.Join(allowedPrincipalArns, ","),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity aws auth",
			"Couldn't update identity aws auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	updateAwsAuthTerraformStateFromApi(ctx, resp.Diagnostics, &plan, &updatedIdentityAwsAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityAwsAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to delete identity aws auth",
			"Only Machine IdentityAwsAuth authentication is supported for this operation",
		)
		return
	}

	var state IdentityAwsAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityAwsAuth(infisical.RevokeIdentityAwsAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity aws auth",
			"Couldn't delete identity aws auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
