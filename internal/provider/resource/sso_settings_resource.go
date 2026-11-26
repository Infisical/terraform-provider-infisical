package resource

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &SsoSettingsResource{}
	_ resource.ResourceWithConfigure      = &SsoSettingsResource{}
	_ resource.ResourceWithImportState    = &SsoSettingsResource{}
	_ resource.ResourceWithValidateConfig = &SsoSettingsResource{}
)

func NewSsoSettingsResource() resource.Resource {
	return &SsoSettingsResource{}
}

type SsoSettingsResource struct {
	client *infisical.Client
}

type SsoSettingsResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	EnforceSso            types.Bool   `tfsdk:"enforce_sso"`
	EnforceGoogleSso      types.Bool   `tfsdk:"enforce_google_sso"`
	AdminSsoBypassEnabled types.Bool   `tfsdk:"admin_sso_bypass_enabled"`
	SsoMfaTrust           types.String `tfsdk:"sso_mfa_trust"`
}

func (r *SsoSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_settings"
}

func (r *SsoSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage the SSO enforcement settings of the organization. An organization has one set of these settings, and they belong to the organization of the machine identity. " +
			"Destroying this resource sets every setting back to its default. " +
			"Only Machine Identity authentication is supported for this resource, and only for a root organization: sub-organizations cannot manage SSO.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the organization.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enforce_sso": schema.BoolAttribute{
				Description: "Whether members must log in through the organization's SAML or OIDC provider. " +
					"Infisical accepts `true` only after someone has logged in through the active SAML or OIDC configuration, so a new configuration needs a login and a second apply. " +
					"A change to the connection settings of the OIDC configuration (for example `client_id` in `infisical_sso_oidc`) turns enforcement off in Infisical, and the next plan shows the change. " +
					"Cannot be `true` together with `enforce_google_sso`. Defaults to `false`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"enforce_google_sso": schema.BoolAttribute{
				Description: "Whether members must log in with Google. Infisical accepts `true` only after someone has logged in with Google, and only while no SAML, OIDC or LDAP configuration is active. " +
					"Cannot be `true` together with `enforce_sso`. Defaults to `false`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"admin_sso_bypass_enabled": schema.BoolAttribute{
				Description: "Whether organization admins can still log in with a password at `/login/admin` while SSO is enforced. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"sso_mfa_trust": schema.StringAttribute{
				Description: "Whether MFA done at the identity provider satisfies the organization's MFA requirement. One of `none` or `signal`. Has no effect unless MFA is enforced. Defaults to `none`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(string(infisical.SsoMfaTrustNone)),
				Validators:  []validator.String{stringvalidator.OneOf(string(infisical.SsoMfaTrustNone), string(infisical.SsoMfaTrustSignal))},
			},
		},
	}
}

func (r *SsoSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig copies the API rule that SAML/OIDC enforcement and Google enforcement cannot both be on.
func (r *SsoSettingsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var enforceSso, enforceGoogleSso types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("enforce_sso"), &enforceSso)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("enforce_google_sso"), &enforceGoogleSso)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if enforceSso.ValueBool() && enforceGoogleSso.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("enforce_google_sso"),
			"Invalid SSO settings",
			"enforce_sso and enforce_google_sso cannot both be true.",
		)
	}
}

func ssoSettingsModelFromAPI(settings infisical.OrgSsoSettings) SsoSettingsResourceModel {
	return SsoSettingsResourceModel{
		ID:                    types.StringValue(settings.OrgID),
		EnforceSso:            types.BoolValue(settings.AuthEnforced),
		EnforceGoogleSso:      types.BoolValue(settings.GoogleSsoAuthEnforced),
		AdminSsoBypassEnabled: types.BoolValue(settings.BypassOrgAuthEnabled),
		SsoMfaTrust:           types.StringValue(string(settings.SsoMfaTrust)),
	}
}

func ssoSettingsRequestFromPlan(plan SsoSettingsResourceModel) infisical.UpdateOrgSsoSettingsRequest {
	return infisical.UpdateOrgSsoSettingsRequest{
		AuthEnforced:          plan.EnforceSso.ValueBool(),
		GoogleSsoAuthEnforced: plan.EnforceGoogleSso.ValueBool(),
		BypassOrgAuthEnabled:  plan.AdminSsoBypassEnabled.ValueBool(),
		SsoMfaTrust:           infisical.SsoMfaTrust(plan.SsoMfaTrust.ValueString()),
	}
}

// Create writes all four settings. The settings always exist for an organization, so there is nothing to create.
func (r *SsoSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create SSO settings",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan SsoSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.UpdateOrgSsoSettings(ssoSettingsRequestFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating SSO settings",
			"Couldn't create SSO settings, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ssoSettingsModelFromAPI(settings))...)
}

func (r *SsoSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read SSO settings",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state SsoSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	settings, err := r.client.GetOrgSsoSettings()
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading SSO settings",
			"Couldn't read SSO settings from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ssoSettingsModelFromAPI(settings))...)
}

func (r *SsoSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update SSO settings",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan SsoSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.UpdateOrgSsoSettings(ssoSettingsRequestFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating SSO settings",
			"Couldn't update SSO settings, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ssoSettingsModelFromAPI(settings))...)
}

// Delete sets every setting back to its default, because the settings belong to the organization and cannot be removed.
func (r *SsoSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete SSO settings",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	_, err := r.client.UpdateOrgSsoSettings(infisical.UpdateOrgSsoSettingsRequest{SsoMfaTrust: infisical.SsoMfaTrustNone})
	if err != nil {
		if err == infisical.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting SSO settings",
			"Couldn't delete SSO settings, unexpected error: "+err.Error(),
		)
	}
}

func (r *SsoSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import SSO settings",
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

	settings, err := r.client.GetOrgSsoSettings()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing SSO settings",
			"Couldn't read SSO settings from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	// The API reads the organization from the token, so the ID only confirms which organization that is.
	if settings.OrgID != req.ID {
		resp.Diagnostics.AddError(
			"Error importing SSO settings",
			fmt.Sprintf("The machine identity belongs to organization %s, not %s. Use the provider credentials of a machine identity in the organization to import.", settings.OrgID, req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ssoSettingsModelFromAPI(settings))...)
}
