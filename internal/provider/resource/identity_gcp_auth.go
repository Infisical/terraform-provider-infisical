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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityGcpAuthResource is a helper function to simplify the provider implementation.
func NewIdentityGcpAuthResource() resource.Resource {
	return &IdentityGcpAuthResource{}
}

// IdentityGcpAuthResource is the resource implementation.
type IdentityGcpAuthResource struct {
	client *infisical.Client
}

// IdentityGcpAuthResourceSourceModel describes the data source data model.
type IdentityGcpAuthResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	IdentityID                  types.String `tfsdk:"identity_id"`
	Type                        types.String `tfsdk:"type"`
	AllowedServiceAccountEmails types.String `tfsdk:"allowed_service_account_emails"`
	AllowedProjects             types.String `tfsdk:"allowed_projects"`
	AllowedZones                types.String `tfsdk:"allowed_zones"`
	AccessTokenTrustedIps       types.List   `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL              types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL           types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit     types.Int64  `tfsdk:"access_token_num_uses_limit"`
}

type IdentityGcpAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

// Metadata returns the resource type name.
func (r *IdentityGcpAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_gcp_auth"
}

// Schema defines the schema for the resource.
func (r *IdentityGcpAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity gcp auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the gcp auth",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Description: "The Type of GCP Auth Method to use: Options are gce, iam. Default:gce",
				Optional:    true,
				Default:     stringdefault.StaticString("gce"),
				Computed:    true,
			},
			"allowed_service_account_emails": schema.StringAttribute{
				Description:         " A comma-separated list of trusted service account emails corresponding to the GCE resource(s) allowed to authenticate with Infisical",
				MarkdownDescription: " A comma-separated list of trusted service account emails corresponding to the GCE resource(s) allowed to authenticate with Infisical; this could be something like `test@project.iam.gserviceaccount.com`, `12345-compute@developer.gserviceaccount.com`, etc.",
				Optional:            true,
				Computed:            true,
			},
			"allowed_projects": schema.StringAttribute{
				Description: "A comma-separated list of trusted GCP projects that the GCE instance must belong to authenticate with Infisical. Note that this validation property will only work for GCE instances",
				Optional:    true,
				Computed:    true,
			},
			"allowed_zones": schema.StringAttribute{
				Description: "A comma-separated list of trusted zones that the GCE instances must belong to authenticate with Infisical; this should be the fully-qualified zone name in the format `<region>-<zone>`like `us-central1-a`, `us-west1-b`, etc. Note that this validation property will only work for GCE instances.",
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
func (r *IdentityGcpAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func updateGcpAuthStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityGcpAuthResourceModel, newIdentityGcpAuth *infisicalclient.IdentityGcpAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityGcpAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityGcpAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityGcpAuth.AccessTokenNumUsesLimit)
	plan.AllowedZones = types.StringValue(newIdentityGcpAuth.AllowedZones)
	plan.AllowedProjects = types.StringValue(newIdentityGcpAuth.AllowedProjects)
	plan.AllowedServiceAccountEmails = types.StringValue(newIdentityGcpAuth.AllowedServiceAccounts)

	planAccessTokenTrustedIps := make([]IdentityGcpAuthResourceTrustedIps, len(newIdentityGcpAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityGcpAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityGcpAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityGcpAuthResourceTrustedIps{IpAddress: types.StringValue(
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
func (r *IdentityGcpAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to create identity gcp auth",
			"Only Machine IdentityGcpAuth authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityGcpAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	newIdentityGcpAuth, err := r.client.CreateIdentityGcpAuth(infisical.CreateIdentityGcpAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		Type:                    plan.Type.ValueString(),
		AllowedServiceAccounts:  plan.AllowedServiceAccountEmails.ValueString(),
		AllowedProjects:         plan.AllowedProjects.ValueString(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		AllowedZones:            plan.AllowedZones.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity gcp auth",
			"Couldn't save gcp auth to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityGcpAuth.ID)
	updateGcpAuthStateByApi(ctx, resp.Diagnostics, &plan, &newIdentityGcpAuth)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityGcpAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to read identity gcp auth role",
			"Only Machine IdentityGcpAuth authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityGcpAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityGcpAuth, err := r.client.GetIdentityGcpAuth(infisical.GetIdentityGcpAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity gcp auth",
				"Couldn't read identity gcp auth from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateGcpAuthStateByApi(ctx, resp.Diagnostics, &state, &identityGcpAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityGcpAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to update identity gcp auth",
			"Only Machine IdentityGcpAuth authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityGcpAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityGcpAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	updatedIdentityGcpAuth, err := r.client.UpdateIdentityGcpAuth(infisical.UpdateIdentityGcpAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		Type:                    plan.Type.ValueString(),
		AllowedServiceAccounts:  plan.AllowedServiceAccountEmails.ValueString(),
		AllowedProjects:         plan.AllowedProjects.ValueString(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		AllowedZones:            plan.AllowedZones.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity gcp auth",
			"Couldn't update identity gcp auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	updateGcpAuthStateByApi(ctx, resp.Diagnostics, &plan, &updatedIdentityGcpAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityGcpAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to delete identity gcp auth",
			"Only Machine IdentityGcpAuth authentication is supported for this operation",
		)
		return
	}

	var state IdentityGcpAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityGcpAuth(infisical.RevokeIdentityGcpAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity gcp auth",
			"Couldn't delete identity gcp auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
