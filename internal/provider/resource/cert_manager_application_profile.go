package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &certManagerApplicationProfileResource{}
)

func NewCertManagerApplicationProfileResource() resource.Resource {
	return &certManagerApplicationProfileResource{}
}

type certManagerApplicationProfileResource struct {
	client *infisical.Client
}

type certManagerApplicationProfileResourceModel struct {
	Id            types.String                             `tfsdk:"id"`
	ApplicationId types.String                             `tfsdk:"application_id"`
	ProfileId     types.String                             `tfsdk:"profile_id"`
	ApiConfig     *certManagerApplicationProfileApiConfig  `tfsdk:"api_config"`
	EstConfig     *certManagerApplicationProfileEstConfig  `tfsdk:"est_config"`
	AcmeConfig    *certManagerApplicationProfileAcmeConfig `tfsdk:"acme_config"`
	ScepConfig    *certManagerApplicationProfileScepConfig `tfsdk:"scep_config"`
}

type certManagerApplicationProfileApiConfig struct {
	AutoRenew       types.Bool  `tfsdk:"auto_renew"`
	RenewBeforeDays types.Int64 `tfsdk:"renew_before_days"`
}

type certManagerApplicationProfileEstConfig struct {
	Passphrase                   types.String `tfsdk:"passphrase"`
	DisableBootstrapCaValidation types.Bool   `tfsdk:"disable_bootstrap_ca_validation"`
	CaChain                      types.String `tfsdk:"ca_chain"`
	EstEndpointUrl               types.String `tfsdk:"est_endpoint_url"`
}

type certManagerApplicationProfileAcmeConfig struct {
	SkipDnsOwnershipVerification types.Bool   `tfsdk:"skip_dns_ownership_verification"`
	SkipEabBinding               types.Bool   `tfsdk:"skip_eab_binding"`
	DirectoryUrl                 types.String `tfsdk:"directory_url"`
	EabKid                       types.String `tfsdk:"eab_kid"`
	EabSecret                    types.String `tfsdk:"eab_secret"`
}

type certManagerApplicationProfileScepConfig struct {
	ChallengeType                 types.String `tfsdk:"challenge_type"`
	ChallengePassword             types.String `tfsdk:"challenge_password"`
	IncludeCaCertInResponse       types.Bool   `tfsdk:"include_ca_cert_in_response"`
	AllowCertBasedRenewal         types.Bool   `tfsdk:"allow_cert_based_renewal"`
	DynamicChallengeExpiryMinutes types.Int64  `tfsdk:"dynamic_challenge_expiry_minutes"`
	DynamicChallengeMaxPending    types.Int64  `tfsdk:"dynamic_challenge_max_pending"`
	ScepEndpointUrl               types.String `tfsdk:"scep_endpoint_url"`
	ChallengeEndpointUrl          types.String `tfsdk:"challenge_endpoint_url"`
	RaCertificatePem              types.String `tfsdk:"ra_certificate_pem"`
	RaCertExpiresAt               types.String `tfsdk:"ra_cert_expires_at"`
}

func (r *certManagerApplicationProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_application_profile"
}

func (r *certManagerApplicationProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attach a certificate profile to a Certificate Manager application and configure its enrollment methods. Each enrollment block (api_config, est_config, acme_config, scep_config) is optional; add, edit, or remove a block to update the matching enrollment on the (application, profile) pair. Only Machine Identity authentication is supported. Import: `terraform import <addr> <applicationId>:<profileId>`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite identifier in the format <applicationId>:<profileId>",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "The ID of the Certificate Manager application",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"profile_id": schema.StringAttribute{
				Description: "The ID of the certificate profile to attach",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_config": schema.SingleNestedAttribute{
				Description: "Enable the API enrollment method on the (application, profile) pair. Omit the block to disable API enrollment.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"auto_renew": schema.BoolAttribute{
						Description: "Whether to automatically renew certificates. Defaults to false when omitted.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"renew_before_days": schema.Int64Attribute{
						Description: "Number of days before expiration to renew (1-30). Defaults to 7 when omitted.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(1, 30)},
					},
				},
			},
			"est_config": schema.SingleNestedAttribute{
				Description: "Enable the EST enrollment method on the (application, profile) pair. Omit the block to disable EST enrollment.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"passphrase": schema.StringAttribute{
						Description: "EST passphrase used to authorize certificate requests. Sensitive: stored only on the client side; the backend stores a hash.",
						Required:    true,
						Sensitive:   true,
					},
					"disable_bootstrap_ca_validation": schema.BoolAttribute{
						Description: "Whether to disable bootstrap CA validation. Defaults to false.",
						Optional:    true,
						Computed:    true,
					},
					"ca_chain": schema.StringAttribute{
						Description: "PEM-encoded CA chain used for bootstrap CA validation (only honored when disable_bootstrap_ca_validation is false).",
						Optional:    true,
					},
					"est_endpoint_url": schema.StringAttribute{
						Description: "The EST endpoint URL clients should use.",
						Computed:    true,
					},
				},
			},
			"acme_config": schema.SingleNestedAttribute{
				Description: "Enable the ACME enrollment method on the (application, profile) pair. Omit the block to disable ACME enrollment.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"skip_dns_ownership_verification": schema.BoolAttribute{
						Description: "Skip DNS ownership verification. Defaults to false.",
						Optional:    true,
						Computed:    true,
					},
					"skip_eab_binding": schema.BoolAttribute{
						Description: "Skip External Account Binding. Defaults to false. Cannot be set to true at the same time as skip_dns_ownership_verification.",
						Optional:    true,
						Computed:    true,
					},
					"directory_url": schema.StringAttribute{
						Description: "The ACME directory URL clients should use.",
						Computed:    true,
					},
					"eab_kid": schema.StringAttribute{
						Description: "External Account Binding key identifier. Populated on create and on import; routine refreshes don't re-fetch it. Rotated only by the explicit rotate endpoint, never by Terraform.",
						Computed:    true,
						Sensitive:   true,
					},
					"eab_secret": schema.StringAttribute{
						Description: "External Account Binding shared secret. Populated on create and on import; routine refreshes don't re-fetch it. Rotated only by the explicit rotate endpoint, never by Terraform.",
						Computed:    true,
						Sensitive:   true,
					},
				},
			},
			"scep_config": schema.SingleNestedAttribute{
				Description: "Enable the SCEP enrollment method on the (application, profile) pair. Omit the block to disable SCEP enrollment.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"challenge_type": schema.StringAttribute{
						Description: "SCEP challenge type. Supported values: static, dynamic. Defaults to static.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.String{stringvalidator.OneOf("static", "dynamic")},
					},
					"challenge_password": schema.StringAttribute{
						Description: "Static-mode SCEP challenge password (min 8 chars). Required when challenge_type is static.",
						Optional:    true,
						Sensitive:   true,
					},
					"include_ca_cert_in_response": schema.BoolAttribute{
						Description: "Include the issuing CA certificate in SCEP responses. Defaults to true.",
						Optional:    true,
						Computed:    true,
					},
					"allow_cert_based_renewal": schema.BoolAttribute{
						Description: "Allow certificate-based renewal. Defaults to true.",
						Optional:    true,
						Computed:    true,
					},
					"dynamic_challenge_expiry_minutes": schema.Int64Attribute{
						Description: "Expiry of a dynamic challenge in minutes (1-1440). Only used when challenge_type is dynamic.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(1, 1440)},
					},
					"dynamic_challenge_max_pending": schema.Int64Attribute{
						Description: "Maximum pending dynamic challenges (1-1000). Only used when challenge_type is dynamic.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(1, 1000)},
					},
					"scep_endpoint_url": schema.StringAttribute{
						Description: "The SCEP endpoint URL clients should use.",
						Computed:    true,
					},
					"challenge_endpoint_url": schema.StringAttribute{
						Description: "The SCEP dynamic challenge endpoint URL (only set when challenge_type is dynamic).",
						Computed:    true,
					},
					"ra_certificate_pem": schema.StringAttribute{
						Description: "The PEM-encoded RA certificate used by the SCEP service.",
						Computed:    true,
					},
					"ra_cert_expires_at": schema.StringAttribute{
						Description: "ISO-8601 timestamp when the RA certificate expires.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *certManagerApplicationProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *certManagerApplicationProfileResource) applyEnrollment(
	prior, plan *certManagerApplicationProfileResourceModel,
	diags interface{ AddError(string, string) },
) {
	appId := plan.ApplicationId.ValueString()
	profileId := plan.ProfileId.ValueString()

	switch {
	case plan.ApiConfig == nil && prior != nil && prior.ApiConfig != nil:
		if _, err := r.client.ClearPkiApplicationApiEnrollment(infisical.ClearPkiApplicationApiEnrollmentRequest{
			ApplicationId: appId, ProfileId: profileId,
		}); err != nil {
			diags.AddError("Error disabling API enrollment", err.Error())
			return
		}
	case plan.ApiConfig != nil:
		setReq := infisical.SetPkiApplicationApiEnrollmentRequest{
			ApplicationId: appId, ProfileId: profileId,
			AutoRenew: plan.ApiConfig.AutoRenew.ValueBool(),
		}
		if !plan.ApiConfig.RenewBeforeDays.IsNull() && !plan.ApiConfig.RenewBeforeDays.IsUnknown() {
			v := int(plan.ApiConfig.RenewBeforeDays.ValueInt64())
			setReq.RenewBeforeDays = &v
		}
		if _, err := r.client.SetPkiApplicationApiEnrollment(setReq); err != nil {
			diags.AddError("Error setting API enrollment", err.Error())
			return
		}
	}

	switch {
	case plan.EstConfig == nil && prior != nil && prior.EstConfig != nil:
		if _, err := r.client.ClearPkiApplicationEstEnrollment(infisical.ClearPkiApplicationEstEnrollmentRequest{
			ApplicationId: appId, ProfileId: profileId,
		}); err != nil {
			diags.AddError("Error disabling EST enrollment", err.Error())
			return
		}
	case plan.EstConfig != nil:
		setReq := infisical.SetPkiApplicationEstEnrollmentRequest{
			ApplicationId: appId, ProfileId: profileId,
			Passphrase: plan.EstConfig.Passphrase.ValueString(),
		}
		if !plan.EstConfig.DisableBootstrapCaValidation.IsNull() && !plan.EstConfig.DisableBootstrapCaValidation.IsUnknown() {
			v := plan.EstConfig.DisableBootstrapCaValidation.ValueBool()
			setReq.DisableBootstrapCaValidation = &v
		}
		if !plan.EstConfig.CaChain.IsNull() && !plan.EstConfig.CaChain.IsUnknown() {
			v := plan.EstConfig.CaChain.ValueString()
			setReq.CaChain = &v
		}
		if _, err := r.client.SetPkiApplicationEstEnrollment(setReq); err != nil {
			diags.AddError("Error setting EST enrollment", err.Error())
			return
		}
	}

	switch {
	case plan.AcmeConfig == nil && prior != nil && prior.AcmeConfig != nil:
		if _, err := r.client.ClearPkiApplicationAcmeEnrollment(infisical.ClearPkiApplicationAcmeEnrollmentRequest{
			ApplicationId: appId, ProfileId: profileId,
		}); err != nil {
			diags.AddError("Error disabling ACME enrollment", err.Error())
			return
		}
	case plan.AcmeConfig != nil:
		setReq := infisical.SetPkiApplicationAcmeEnrollmentRequest{
			ApplicationId: appId, ProfileId: profileId,
		}
		if !plan.AcmeConfig.SkipDnsOwnershipVerification.IsNull() && !plan.AcmeConfig.SkipDnsOwnershipVerification.IsUnknown() {
			v := plan.AcmeConfig.SkipDnsOwnershipVerification.ValueBool()
			setReq.SkipDnsOwnershipVerification = &v
		}
		if !plan.AcmeConfig.SkipEabBinding.IsNull() && !plan.AcmeConfig.SkipEabBinding.IsUnknown() {
			v := plan.AcmeConfig.SkipEabBinding.ValueBool()
			setReq.SkipEabBinding = &v
		}
		if _, err := r.client.SetPkiApplicationAcmeEnrollment(setReq); err != nil {
			diags.AddError("Error setting ACME enrollment", err.Error())
			return
		}
	}

	switch {
	case plan.ScepConfig == nil && prior != nil && prior.ScepConfig != nil:
		if _, err := r.client.ClearPkiApplicationScepEnrollment(infisical.ClearPkiApplicationScepEnrollmentRequest{
			ApplicationId: appId, ProfileId: profileId,
		}); err != nil {
			diags.AddError("Error disabling SCEP enrollment", err.Error())
			return
		}
	case plan.ScepConfig != nil:
		setReq := infisical.SetPkiApplicationScepEnrollmentRequest{
			ApplicationId: appId, ProfileId: profileId,
			ChallengeType: plan.ScepConfig.ChallengeType.ValueString(),
		}
		if !plan.ScepConfig.ChallengePassword.IsNull() && !plan.ScepConfig.ChallengePassword.IsUnknown() {
			setReq.ChallengePassword = plan.ScepConfig.ChallengePassword.ValueString()
		}
		if !plan.ScepConfig.IncludeCaCertInResponse.IsNull() && !plan.ScepConfig.IncludeCaCertInResponse.IsUnknown() {
			v := plan.ScepConfig.IncludeCaCertInResponse.ValueBool()
			setReq.IncludeCaCertInResponse = &v
		}
		if !plan.ScepConfig.AllowCertBasedRenewal.IsNull() && !plan.ScepConfig.AllowCertBasedRenewal.IsUnknown() {
			v := plan.ScepConfig.AllowCertBasedRenewal.ValueBool()
			setReq.AllowCertBasedRenewal = &v
		}
		if !plan.ScepConfig.DynamicChallengeExpiryMinutes.IsNull() && !plan.ScepConfig.DynamicChallengeExpiryMinutes.IsUnknown() {
			v := int(plan.ScepConfig.DynamicChallengeExpiryMinutes.ValueInt64())
			setReq.DynamicChallengeExpiryMinutes = &v
		}
		if !plan.ScepConfig.DynamicChallengeMaxPending.IsNull() && !plan.ScepConfig.DynamicChallengeMaxPending.IsUnknown() {
			v := int(plan.ScepConfig.DynamicChallengeMaxPending.ValueInt64())
			setReq.DynamicChallengeMaxPending = &v
		}
		if _, err := r.client.SetPkiApplicationScepEnrollment(setReq); err != nil {
			diags.AddError("Error setting SCEP enrollment", err.Error())
			return
		}
	}
}

func (r *certManagerApplicationProfileResource) refresh(model *certManagerApplicationProfileResourceModel) (bool, error) {
	enrollment, err := r.client.GetPkiApplicationEnrollment(infisical.GetPkiApplicationEnrollmentRequest{
		ApplicationId: model.ApplicationId.ValueString(),
		ProfileId:     model.ProfileId.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			return false, nil
		}
		return false, err
	}

	if enrollment.Api != nil {
		if model.ApiConfig == nil {
			model.ApiConfig = &certManagerApplicationProfileApiConfig{}
		}
		model.ApiConfig.AutoRenew = types.BoolValue(enrollment.Api.AutoRenew)
		if enrollment.Api.RenewBeforeDays != nil {
			model.ApiConfig.RenewBeforeDays = types.Int64Value(int64(*enrollment.Api.RenewBeforeDays))
		} else {
			model.ApiConfig.RenewBeforeDays = types.Int64Null()
		}
	} else {
		model.ApiConfig = nil
	}

	if enrollment.Est != nil {
		if model.EstConfig == nil {
			model.EstConfig = &certManagerApplicationProfileEstConfig{}
		}
		model.EstConfig.DisableBootstrapCaValidation = types.BoolValue(enrollment.Est.DisableBootstrapCaValidation)
		model.EstConfig.EstEndpointUrl = types.StringValue(enrollment.Est.EstEndpointUrl)
		if model.EstConfig.Passphrase.IsUnknown() {
			model.EstConfig.Passphrase = types.StringNull()
		}
		if model.EstConfig.CaChain.IsUnknown() {
			model.EstConfig.CaChain = types.StringNull()
		}
	} else {
		model.EstConfig = nil
	}

	if enrollment.Acme != nil {
		if model.AcmeConfig == nil {
			model.AcmeConfig = &certManagerApplicationProfileAcmeConfig{}
		}
		model.AcmeConfig.SkipDnsOwnershipVerification = types.BoolValue(enrollment.Acme.SkipDnsOwnershipVerification)
		model.AcmeConfig.SkipEabBinding = types.BoolValue(enrollment.Acme.SkipEabBinding)
		model.AcmeConfig.DirectoryUrl = types.StringValue(enrollment.Acme.DirectoryUrl)
		needsReveal := model.AcmeConfig.EabKid.IsNull() || model.AcmeConfig.EabKid.IsUnknown() ||
			model.AcmeConfig.EabSecret.IsNull() || model.AcmeConfig.EabSecret.IsUnknown()
		if needsReveal {
			reveal, err := r.client.RevealPkiApplicationAcmeEabSecret(infisical.RevealPkiApplicationAcmeEabSecretRequest{
				ApplicationId: model.ApplicationId.ValueString(),
				ProfileId:     model.ProfileId.ValueString(),
			})
			if err != nil {
				return false, err
			}
			model.AcmeConfig.EabKid = types.StringValue(reveal.EabKid)
			model.AcmeConfig.EabSecret = types.StringValue(reveal.EabSecret)
		}
	} else {
		model.AcmeConfig = nil
	}

	if enrollment.Scep != nil {
		if model.ScepConfig == nil {
			model.ScepConfig = &certManagerApplicationProfileScepConfig{}
		}
		model.ScepConfig.ChallengeType = types.StringValue(enrollment.Scep.ChallengeType)
		model.ScepConfig.IncludeCaCertInResponse = types.BoolValue(enrollment.Scep.IncludeCaCertInResponse)
		model.ScepConfig.AllowCertBasedRenewal = types.BoolValue(enrollment.Scep.AllowCertBasedRenewal)
		if enrollment.Scep.DynamicChallengeExpiryMinutes != nil {
			model.ScepConfig.DynamicChallengeExpiryMinutes = types.Int64Value(int64(*enrollment.Scep.DynamicChallengeExpiryMinutes))
		} else {
			model.ScepConfig.DynamicChallengeExpiryMinutes = types.Int64Null()
		}
		if enrollment.Scep.DynamicChallengeMaxPending != nil {
			model.ScepConfig.DynamicChallengeMaxPending = types.Int64Value(int64(*enrollment.Scep.DynamicChallengeMaxPending))
		} else {
			model.ScepConfig.DynamicChallengeMaxPending = types.Int64Null()
		}
		model.ScepConfig.ScepEndpointUrl = types.StringValue(enrollment.Scep.ScepEndpointUrl)
		if enrollment.Scep.ChallengeEndpointUrl != nil {
			model.ScepConfig.ChallengeEndpointUrl = types.StringValue(*enrollment.Scep.ChallengeEndpointUrl)
		} else {
			model.ScepConfig.ChallengeEndpointUrl = types.StringNull()
		}
		model.ScepConfig.RaCertificatePem = types.StringValue(enrollment.Scep.RaCertificatePem)
		model.ScepConfig.RaCertExpiresAt = types.StringValue(enrollment.Scep.RaCertExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
		if model.ScepConfig.ChallengePassword.IsUnknown() {
			model.ScepConfig.ChallengePassword = types.StringNull()
		}
	} else {
		model.ScepConfig = nil
	}

	return true, nil
}

func (r *certManagerApplicationProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to attach Certificate Manager application profile",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.AttachPkiApplicationProfiles(infisical.AttachPkiApplicationProfilesRequest{
		ApplicationId: plan.ApplicationId.ValueString(),
		ProfileIds:    []string{plan.ProfileId.ValueString()},
	}); err != nil {
		resp.Diagnostics.AddError("Error attaching profile to application", err.Error())
		return
	}

	r.applyEnrollment(nil, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.refresh(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error refreshing application profile after create", err.Error())
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Profile detached unexpectedly", "the profile attachment was not found after create")
		return
	}

	plan.Id = types.StringValue(fmt.Sprintf("%s:%s", plan.ApplicationId.ValueString(), plan.ProfileId.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerApplicationProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read Certificate Manager application profile",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.refresh(&state)
	if err != nil {
		resp.Diagnostics.AddError("Error reading application profile", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Id = types.StringValue(fmt.Sprintf("%s:%s", state.ApplicationId.ValueString(), state.ProfileId.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerApplicationProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update Certificate Manager application profile",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan, prior certManagerApplicationProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyEnrollment(&prior, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.refresh(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error refreshing application profile after update", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	plan.Id = types.StringValue(fmt.Sprintf("%s:%s", plan.ApplicationId.ValueString(), plan.ProfileId.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerApplicationProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to detach Certificate Manager application profile",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.DetachPkiApplicationProfile(infisical.DetachPkiApplicationProfileRequest{
		ApplicationId: state.ApplicationId.ValueString(),
		ProfileId:     state.ProfileId.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error detaching profile from application", err.Error())
		return
	}
}

func (r *certManagerApplicationProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier in the format <applicationId>:<profileId>, got: %s", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
