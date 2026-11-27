package resource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	infisical "terraform-provider-infisical/internal/client"
	infisicalerrors "terraform-provider-infisical/internal/errors"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultOidcJwtSignatureAlgorithm = "RS256"
	defaultOidcMfaClaimName          = "amr"
)

var (
	_ resource.Resource                = &SsoOidcResource{}
	_ resource.ResourceWithConfigure   = &SsoOidcResource{}
	_ resource.ResourceWithImportState = &SsoOidcResource{}
)

func NewSsoOidcResource() resource.Resource {
	return &SsoOidcResource{}
}

type SsoOidcResource struct {
	client *infisical.Client
}

type SsoOidcResourceModel struct {
	ID                     types.String        `tfsdk:"id"`
	DiscoveryURL           types.String        `tfsdk:"discovery_url"`
	Custom                 *SsoOidcCustomModel `tfsdk:"custom"`
	ClientID               types.String        `tfsdk:"client_id"`
	ClientSecret           types.String        `tfsdk:"client_secret"`
	ClientSecretWO         types.String        `tfsdk:"client_secret_wo"`
	ClientSecretWOVersion  types.Int64         `tfsdk:"client_secret_wo_version"`
	AllowedEmailDomains    types.Set           `tfsdk:"allowed_email_domains"`
	JwtSignatureAlgorithm  types.String        `tfsdk:"jwt_signature_algorithm"`
	ManageGroupMemberships types.Bool          `tfsdk:"manage_group_memberships"`
	IsActive               types.Bool          `tfsdk:"is_active"`
	MfaClaimName           types.String        `tfsdk:"mfa_claim_name"`
	MfaAmrValues           types.Set           `tfsdk:"mfa_amr_values"`
}

type SsoOidcCustomModel struct {
	Issuer                types.String `tfsdk:"issuer"`
	AuthorizationEndpoint types.String `tfsdk:"authorization_endpoint"`
	TokenEndpoint         types.String `tfsdk:"token_endpoint"`
	JwksUri               types.String `tfsdk:"jwks_uri"`
	UserinfoEndpoint      types.String `tfsdk:"userinfo_endpoint"`
}

func (r *SsoOidcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_oidc"
}

func (r *SsoOidcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	nonEmpty := []validator.String{stringvalidator.LengthAtLeast(1)}

	resp.Schema = schema.Schema{
		Description: "Manage the OIDC SSO configuration of the organization. An organization has at most one OIDC configuration, and it belongs to the organization of the machine identity. " +
			"Set either `discovery_url` or `custom`. " +
			"A change to `discovery_url`, `custom`, `client_id`, the client secret, `allowed_email_domains`, `jwt_signature_algorithm` or `is_active` turns off SSO enforcement and SCIM for the organization in Infisical, and destroying this resource turns them off too. " +
			"A change to `manage_group_memberships`, `mfa_claim_name` or `mfa_amr_values` alone keeps them on. " +
			"Older Infisical versions turn them off on any update that changes an attribute other than `mfa_claim_name` and `mfa_amr_values`. " +
			"Only Machine Identity authentication is supported for this resource, and only for a root organization: sub-organizations cannot manage SSO.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the OIDC configuration.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"discovery_url": schema.StringAttribute{
				Description: "The URL of the OIDC discovery endpoint. Infisical reads the issuer and the endpoints from it. Exactly one of `discovery_url` and `custom` must be set.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(path.MatchRoot("custom")),
				},
			},
			"custom": schema.SingleNestedAttribute{
				Description: "The OIDC endpoints, for an identity provider without a discovery endpoint. Exactly one of `discovery_url` and `custom` must be set.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"issuer": schema.StringAttribute{
						Description: "The issuer of the identity provider.",
						Required:    true,
						Validators:  nonEmpty,
					},
					"authorization_endpoint": schema.StringAttribute{
						Description: "The authorization endpoint of the identity provider.",
						Required:    true,
						Validators:  nonEmpty,
					},
					"token_endpoint": schema.StringAttribute{
						Description: "The token endpoint to use for OIDC token exchange.",
						Required:    true,
						Validators:  nonEmpty,
					},
					"jwks_uri": schema.StringAttribute{
						Description: "The URL of the OIDC JWKS endpoint.",
						Required:    true,
						Validators:  nonEmpty,
					},
					"userinfo_endpoint": schema.StringAttribute{
						Description: "The userinfo endpoint to get user information from the OIDC provider.",
						Required:    true,
						Validators:  nonEmpty,
					},
				},
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID to use for OIDC authentication.",
				Required:    true,
				Validators:  nonEmpty,
			},
			"client_secret": schema.StringAttribute{
				Description: "The client secret to use for OIDC authentication. Exactly one of `client_secret` and `client_secret_wo` must be set.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ExactlyOneOf(path.MatchRoot("client_secret_wo")),
					stringvalidator.ConflictsWith(path.MatchRoot("client_secret_wo_version")),
				},
			},
			"client_secret_wo": schema.StringAttribute{
				Description: "The client secret to use for OIDC authentication, as a write-only value that is not stored in state. Exactly one of `client_secret` and `client_secret_wo` must be set. Requires Terraform version 1.11.0 or higher.",
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.AlsoRequires(path.MatchRoot("client_secret_wo_version")),
				},
			},
			"client_secret_wo_version": schema.Int64Attribute{
				Description: "Used together with `client_secret_wo` to trigger an update. Increment this value when `client_secret_wo` changes.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AlsoRequires(path.MatchRoot("client_secret_wo")),
				},
			},
			"allowed_email_domains": schema.SetAttribute{
				Description: "The email domains that can log in through this configuration, for example `example.com`. When empty, every domain can log in.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(`^[^\s,]+$`), "must not contain whitespace or commas"),
					),
				},
			},
			"jwt_signature_algorithm": schema.StringAttribute{
				Description: "The algorithm to use for JWT signature verification. One of `RS256`, `HS256`, `RS512` or `EdDSA`. Defaults to `RS256`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultOidcJwtSignatureAlgorithm),
				Validators:  []validator.String{stringvalidator.OneOf("RS256", "HS256", "RS512", "EdDSA")},
			},
			"manage_group_memberships": schema.BoolAttribute{
				Description: "Whether Infisical syncs the user's group memberships from the groups claim at each login. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether this OIDC configuration is enabled. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"mfa_claim_name": schema.StringAttribute{
				Description: "The ID token claim that lists the authentication methods the user completed. Defaults to `amr`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultOidcMfaClaimName),
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 128)},
			},
			"mfa_amr_values": schema.SetAttribute{
				Description: "Additional MFA claim values, besides `mfa`, that prove the identity provider completed MFA. At most 20 values.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtMost(20),
					setvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 64)),
				},
			},
		},
	}
}

func (r *SsoOidcResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func oidcClientSecretFromConfig(ctx context.Context, plan SsoOidcResourceModel, config tfsdk.Config) (string, diag.Diagnostics) {
	if !plan.ClientSecret.IsNull() {
		return plan.ClientSecret.ValueString(), nil
	}
	var secret types.String
	diags := config.GetAttribute(ctx, path.Root("client_secret_wo"), &secret)
	return secret.ValueString(), diags
}

func oidcStringsFromSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	values := []string{}
	if set.IsNull() || set.IsUnknown() {
		return values, nil
	}
	diags := set.ElementsAs(ctx, &values, false)
	return values, diags
}

// oidcSetFromAPI keeps an empty list null when the user did not set the attribute.
func oidcSetFromAPI(values []string, prior types.Set) types.Set {
	if len(values) == 0 && prior.IsNull() {
		return types.SetNull(types.StringType)
	}
	elements := make([]types.String, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	set, _ := types.SetValueFrom(context.Background(), types.StringType, elements)
	return set
}

// splitOidcAllowedEmailDomains reads allowedEmailDomains, which the API stores as one string joined with ", ".
func splitOidcAllowedEmailDomains(value string) []string {
	domains := []string{}
	for _, domain := range strings.Split(value, ",") {
		if domain = strings.TrimSpace(domain); domain != "" {
			domains = append(domains, domain)
		}
	}
	return domains
}

// ssoOidcModelFromAPI builds state from a GET response. prior supplies what the API does not decide: whether the
// user keeps the secret in state, and whether an empty list was set or left out.
func ssoOidcModelFromAPI(config infisical.OidcConfig, prior SsoOidcResourceModel) SsoOidcResourceModel {
	model := SsoOidcResourceModel{
		ID:                     types.StringValue(config.ID),
		DiscoveryURL:           types.StringNull(),
		ClientID:               types.StringValue(config.ClientID),
		ClientSecret:           types.StringNull(),
		ClientSecretWO:         types.StringNull(),
		ClientSecretWOVersion:  prior.ClientSecretWOVersion,
		AllowedEmailDomains:    oidcSetFromAPI(splitOidcAllowedEmailDomains(config.AllowedEmailDomains), prior.AllowedEmailDomains),
		JwtSignatureAlgorithm:  types.StringValue(config.JwtSignatureAlgorithm),
		ManageGroupMemberships: types.BoolValue(config.ManageGroupMemberships),
		IsActive:               types.BoolValue(config.IsActive),
		MfaClaimName:           types.StringValue(config.MfaClaimName),
		MfaAmrValues:           oidcSetFromAPI(config.MfaAmrValues, prior.MfaAmrValues),
	}

	if config.ConfigurationType == infisical.OidcConfigurationTypeCustom {
		model.Custom = &SsoOidcCustomModel{
			Issuer:                types.StringValue(config.Issuer),
			AuthorizationEndpoint: types.StringValue(config.AuthorizationEndpoint),
			TokenEndpoint:         types.StringValue(config.TokenEndpoint),
			JwksUri:               types.StringValue(config.JwksUri),
			UserinfoEndpoint:      types.StringValue(config.UserinfoEndpoint),
		}
	} else {
		model.DiscoveryURL = types.StringValue(config.DiscoveryURL)
	}

	// GET returns the decrypted secret. It goes into state only for client_secret, never for client_secret_wo.
	if !prior.ClientSecret.IsNull() {
		model.ClientSecret = types.StringValue(config.ClientSecret)
	}

	return model
}

// oidcDesiredConfig is the configuration that the plan asks for, in the API shape.
func oidcDesiredConfig(ctx context.Context, plan SsoOidcResourceModel, config tfsdk.Config) (infisical.OidcConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	clientSecret, d := oidcClientSecretFromConfig(ctx, plan, config)
	diags.Append(d...)
	allowedEmailDomains, d := oidcStringsFromSet(ctx, plan.AllowedEmailDomains)
	diags.Append(d...)
	mfaAmrValues, d := oidcStringsFromSet(ctx, plan.MfaAmrValues)
	diags.Append(d...)

	desired := infisical.OidcConfig{
		ConfigurationType:      infisical.OidcConfigurationTypeDiscoveryURL,
		DiscoveryURL:           plan.DiscoveryURL.ValueString(),
		ClientID:               plan.ClientID.ValueString(),
		ClientSecret:           clientSecret,
		AllowedEmailDomains:    strings.Join(allowedEmailDomains, ","),
		JwtSignatureAlgorithm:  plan.JwtSignatureAlgorithm.ValueString(),
		ManageGroupMemberships: plan.ManageGroupMemberships.ValueBool(),
		IsActive:               plan.IsActive.ValueBool(),
		MfaClaimName:           plan.MfaClaimName.ValueString(),
		MfaAmrValues:           mfaAmrValues,
	}
	if plan.Custom != nil {
		desired.ConfigurationType = infisical.OidcConfigurationTypeCustom
		desired.Issuer = plan.Custom.Issuer.ValueString()
		desired.AuthorizationEndpoint = plan.Custom.AuthorizationEndpoint.ValueString()
		desired.TokenEndpoint = plan.Custom.TokenEndpoint.ValueString()
		desired.JwksUri = plan.Custom.JwksUri.ValueString()
		desired.UserinfoEndpoint = plan.Custom.UserinfoEndpoint.ValueString()
	}
	return desired, diags
}

func (r *SsoOidcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create OIDC SSO configuration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan SsoOidcResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desired, diags := oidcDesiredConfig(ctx, plan, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := infisical.CreateOidcConfigRequest{
		ConfigurationType:      desired.ConfigurationType,
		DiscoveryURL:           desired.DiscoveryURL,
		Issuer:                 desired.Issuer,
		AuthorizationEndpoint:  desired.AuthorizationEndpoint,
		TokenEndpoint:          desired.TokenEndpoint,
		JwksUri:                desired.JwksUri,
		UserinfoEndpoint:       desired.UserinfoEndpoint,
		ClientID:               desired.ClientID,
		ClientSecret:           desired.ClientSecret,
		AllowedEmailDomains:    desired.AllowedEmailDomains,
		JwtSignatureAlgorithm:  desired.JwtSignatureAlgorithm,
		ManageGroupMemberships: desired.ManageGroupMemberships,
		IsActive:               desired.IsActive,
		MfaClaimName:           desired.MfaClaimName,
		MfaAmrValues:           desired.MfaAmrValues,
	}

	if _, err := r.client.CreateOidcConfig(request); err != nil {
		var apiErr *infisicalerrors.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			resp.Diagnostics.AddError(
				"Error creating OIDC SSO configuration",
				"The organization already has an OIDC configuration. To manage it with Terraform, import it with the organization ID: terraform import <resource address> <organization-id>",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error creating OIDC SSO configuration",
			"Couldn't create OIDC SSO configuration, unexpected error: "+err.Error(),
		)
		return
	}

	// The create response leaves out the client ID and secret, so state comes from GET.
	config, err := r.client.GetOidcConfig()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading OIDC SSO configuration",
			"Created the OIDC SSO configuration, but couldn't read it back, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ssoOidcModelFromAPI(config, plan))...)
}

func (r *SsoOidcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read OIDC SSO configuration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state SsoOidcResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	config, err := r.client.GetOidcConfig()
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading OIDC SSO configuration",
			"Couldn't read OIDC SSO configuration from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	// The API reads the organization from the token, this prevents side-effects due to switching creds in between.
	if config.ID != state.ID.ValueString() {
		resp.Diagnostics.AddError(
			"OIDC SSO configuration does not match state",
			fmt.Sprintf("The provider credentials see OIDC configuration %s in organization %s, but state holds configuration %s. "+
				"Either the credentials belong to a different organization, or the configuration was recreated outside Terraform. "+
				"Use the credentials of the organization that owns this resource, or run terraform state rm and import it again with the organization ID.",
				config.ID, config.OrgID, state.ID.ValueString()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ssoOidcModelFromAPI(config, state))...)
}

func (r *SsoOidcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update OIDC SSO configuration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan SsoOidcResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desired, diags := oidcDesiredConfig(ctx, plan, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateOidcConfig(infisical.UpdateOidcConfigRequest{
		ConfigurationType:      desired.ConfigurationType,
		DiscoveryURL:           desired.DiscoveryURL,
		Issuer:                 desired.Issuer,
		AuthorizationEndpoint:  desired.AuthorizationEndpoint,
		TokenEndpoint:          desired.TokenEndpoint,
		JwksUri:                desired.JwksUri,
		UserinfoEndpoint:       desired.UserinfoEndpoint,
		ClientID:               desired.ClientID,
		ClientSecret:           desired.ClientSecret,
		AllowedEmailDomains:    desired.AllowedEmailDomains,
		JwtSignatureAlgorithm:  desired.JwtSignatureAlgorithm,
		ManageGroupMemberships: desired.ManageGroupMemberships,
		IsActive:               desired.IsActive,
		MfaClaimName:           desired.MfaClaimName,
		MfaAmrValues:           desired.MfaAmrValues,
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Error updating OIDC SSO configuration",
				"The OIDC SSO configuration no longer exists in Infisical. Run terraform apply again to create it.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating OIDC SSO configuration",
			"Couldn't update OIDC SSO configuration, unexpected error: "+err.Error(),
		)
		return
	}

	// The update response leaves out the client ID and secret. After the update they are the desired values.
	updated.ClientID = desired.ClientID
	updated.ClientSecret = desired.ClientSecret

	resp.Diagnostics.Append(resp.State.Set(ctx, ssoOidcModelFromAPI(updated, plan))...)
}

func (r *SsoOidcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete OIDC SSO configuration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	if err := r.client.DeleteOidcConfig(); err != nil {
		if err == infisical.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting OIDC SSO configuration",
			"Couldn't delete OIDC SSO configuration, unexpected error: "+err.Error(),
		)
	}
}

// ImportState takes the organization ID, because an organization has at most one OIDC configuration.
func (r *SsoOidcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import OIDC SSO configuration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	if _, err := uuid.ParseUUID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be the ID of the organization (a UUID)",
		)
		return
	}

	config, err := r.client.GetOidcConfig()
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError("Error importing OIDC SSO configuration", "The organization has no OIDC configuration")
			return
		}
		resp.Diagnostics.AddError(
			"Error importing OIDC SSO configuration",
			"Couldn't read OIDC SSO configuration from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	// The API reads the organization from the token, so the ID only confirms which organization that is.
	if config.OrgID != req.ID {
		resp.Diagnostics.AddError(
			"Error importing OIDC SSO configuration",
			fmt.Sprintf("The machine identity belongs to organization %s, not %s. Use the provider credentials of a machine identity in the organization to import.", config.OrgID, req.ID),
		)
		return
	}

	// client_secret starts as known, so the secret is imported into state. Configurations that use
	// client_secret_wo drop it on the next apply.
	prior := SsoOidcResourceModel{
		ClientSecret:          types.StringValue(""),
		ClientSecretWOVersion: types.Int64Null(),
		AllowedEmailDomains:   types.SetNull(types.StringType),
		MfaAmrValues:          types.SetNull(types.StringType),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, ssoOidcModelFromAPI(config, prior))...)
}
