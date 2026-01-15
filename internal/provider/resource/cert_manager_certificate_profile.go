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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	SUPPORTED_ENROLLMENT_TYPES = []string{"api", "est", "acme"}
	SUPPORTED_ISSUER_TYPES     = []string{"ca", "self-signed"}
)

var (
	_ resource.Resource = &certManagerCertificateProfileResource{}
)

func NewCertManagerCertificateProfileResource() resource.Resource {
	return &certManagerCertificateProfileResource{}
}

type certManagerCertificateProfileResource struct {
	client *infisical.Client
}

type certManagerCertificateProfileEstConfigModel struct {
	DisableBootstrapCaValidation types.Bool   `tfsdk:"disable_bootstrap_ca_validation"`
	Passphrase                   types.String `tfsdk:"passphrase"`
	CaChain                      types.String `tfsdk:"ca_chain"`
}

type certManagerCertificateProfileApiConfigModel struct {
	AutoRenew       types.Bool  `tfsdk:"auto_renew"`
	RenewBeforeDays types.Int64 `tfsdk:"renew_before_days"`
}

type certManagerCertificateProfileExternalConfigsModel struct {
	Template types.String `tfsdk:"template"`
}

type certManagerCertificateProfileResourceModel struct {
	ProjectSlug           types.String                                       `tfsdk:"project_slug"`
	Id                    types.String                                       `tfsdk:"id"`
	CaId                  types.String                                       `tfsdk:"ca_id"`
	CertificatePolicyId types.String                                       `tfsdk:"certificate_policy_id"`
	Name                  types.String                                       `tfsdk:"name" json:"slug"`
	Description           types.String                                       `tfsdk:"description"`
	EnrollmentType        types.String                                       `tfsdk:"enrollment_type"`
	IssuerType            types.String                                       `tfsdk:"issuer_type"`
	EstConfig             *certManagerCertificateProfileEstConfigModel       `tfsdk:"est_config"`
	ApiConfig             *certManagerCertificateProfileApiConfigModel       `tfsdk:"api_config"`
	ExternalConfigs       *certManagerCertificateProfileExternalConfigsModel `tfsdk:"external_configs"`
}

func (r *certManagerCertificateProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_certificate_profile"
}

func (r *certManagerCertificateProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage certificate profiles in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"project_slug": schema.StringAttribute{
				Description: "The slug of the cert-manager project",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The ID of the certificate profile",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ca_id": schema.StringAttribute{
				Description: "The ID of the certificate authority to use (required unless issuer_type is 'self-signed')",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate_policy_id": schema.StringAttribute{
				Description: "The ID of the certificate policy to use",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The unique name of the certificate profile",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the certificate profile",
				Optional:    true,
			},
			"enrollment_type": schema.StringAttribute{
				Description: "The enrollment type for the profile. Supported values: " + strings.Join(SUPPORTED_ENROLLMENT_TYPES, ", "),
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_ENROLLMENT_TYPES...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"issuer_type": schema.StringAttribute{
				Description: "The issuer type for the profile. Supported values: " + strings.Join(SUPPORTED_ISSUER_TYPES, ", ") + ". Defaults to 'ca'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ca"),
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_ISSUER_TYPES...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"est_config": schema.SingleNestedBlock{
				Description: "EST configuration (required when enrollment_type is 'est')",
				Attributes: map[string]schema.Attribute{
					"disable_bootstrap_ca_validation": schema.BoolAttribute{
						Description: "Whether to disable bootstrap CA validation",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"passphrase": schema.StringAttribute{
						Description: "The passphrase for EST enrollment",
						Optional:    true,
						Sensitive:   true,
					},
					"ca_chain": schema.StringAttribute{
						Description: "The CA certificate chain for EST enrollment",
						Optional:    true,
						Sensitive:   true,
					},
				},
			},
			"api_config": schema.SingleNestedBlock{
				Description: "API configuration (required when enrollment_type is 'api')",
				Attributes: map[string]schema.Attribute{
					"auto_renew": schema.BoolAttribute{
						Description: "Whether to automatically renew certificates",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"renew_before_days": schema.Int64Attribute{
						Description: "Number of days before expiration to renew certificates (1-30)",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(7),
						Validators: []validator.Int64{
							int64validator.Between(1, 30),
						},
					},
				},
			},
			"external_configs": schema.SingleNestedBlock{
				Description: "External configuration for external CA types (e.g., ADCS template name)",
				Attributes: map[string]schema.Attribute{
					"template": schema.StringAttribute{
						Description: "Certificate template name for Azure AD CS",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *certManagerCertificateProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerCertificateProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create certificate profile",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerCertificateProfileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	enrollmentType := plan.EnrollmentType.ValueString()

	switch enrollmentType {
	case "est":
		if plan.EstConfig == nil {
			resp.Diagnostics.AddError("Missing EST configuration", "est_config block is required when enrollment_type is 'est'")
			return
		}
		if plan.EstConfig.Passphrase.IsNull() || plan.EstConfig.Passphrase.ValueString() == "" {
			resp.Diagnostics.AddError("Missing EST passphrase", "est_config.passphrase is required when enrollment_type is 'est'")
			return
		}
	case "api":
		if plan.ApiConfig == nil {
			resp.Diagnostics.AddError("Missing API configuration", "api_config block is required when enrollment_type is 'api'")
			return
		}
	case "acme":
		if plan.IssuerType.ValueString() == "self-signed" {
			resp.Diagnostics.AddError("Invalid issuer type for ACME", "ACME enrollment_type cannot be used with self-signed issuer_type")
			return
		}
	default:
		resp.Diagnostics.AddError("Invalid enrollment type", fmt.Sprintf("enrollment_type must be one of: api, est, acme. Got: %s", enrollmentType))
		return
	}

	if plan.IssuerType.ValueString() == "self-signed" && !plan.CaId.IsNull() {
		resp.Diagnostics.AddError("Invalid CA ID for self-signed", "ca_id should not be specified when issuer_type is 'self-signed'")
		return
	}

	if plan.IssuerType.ValueString() == "ca" && plan.CaId.IsNull() {
		resp.Diagnostics.AddError("Missing CA ID", "ca_id is required when issuer_type is 'ca'")
		return
	}

	project, err := r.client.GetProject(infisical.GetProjectRequest{
		Slug: plan.ProjectSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error finding project by slug", err.Error())
		return
	}

	createProfileRequest := infisical.CreateCertificateProfileRequest{
		ProjectId:             project.ID,
		CertificatePolicyId: plan.CertificatePolicyId.ValueString(),
		Slug:                  plan.Name.ValueString(),
		EnrollmentType:        plan.EnrollmentType.ValueString(),
		Description:           plan.Description.ValueString(),
	}

	if !plan.CaId.IsNull() {
		createProfileRequest.CaId = plan.CaId.ValueString()
	}

	if !plan.IssuerType.IsNull() {
		createProfileRequest.IssuerType = plan.IssuerType.ValueString()
	}

	if plan.EstConfig != nil {
		createProfileRequest.EstConfig = &infisical.CertificateProfileEstConfig{
			DisableBootstrapCaValidation: plan.EstConfig.DisableBootstrapCaValidation.ValueBool(),
			Passphrase:                   plan.EstConfig.Passphrase.ValueString(),
		}
		if !plan.EstConfig.CaChain.IsNull() {
			createProfileRequest.EstConfig.CaChain = plan.EstConfig.CaChain.ValueString()
		}
	}

	if plan.ApiConfig != nil {
		createProfileRequest.ApiConfig = &infisical.CertificateProfileApiConfig{
			AutoRenew: plan.ApiConfig.AutoRenew.ValueBool(),
		}
		if !plan.ApiConfig.RenewBeforeDays.IsNull() {
			createProfileRequest.ApiConfig.RenewBeforeDays = int(plan.ApiConfig.RenewBeforeDays.ValueInt64())
		}
	}

	if plan.ExternalConfigs != nil && !plan.ExternalConfigs.Template.IsNull() {
		createProfileRequest.ExternalConfigs = &infisical.CertificateProfileExternalConfigs{
			Template: plan.ExternalConfigs.Template.ValueString(),
		}
	}

	profile, err := r.client.CreateCertificateProfile(createProfileRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating certificate profile", err.Error())
		return
	}

	plan.Id = types.StringValue(profile.CertificateProfile.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificateProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read certificate profile",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var currentState certManagerCertificateProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerCertificateProfileResourceModel

	profile, err := r.client.GetCertificateProfile(infisical.GetCertificateProfileRequest{
		ProfileId:      currentState.Id.ValueString(),
		IncludeConfigs: true,
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading certificate profile", err.Error())
		return
	}

	state.Id = types.StringValue(profile.CertificateProfile.Id)
	state.ProjectSlug = currentState.ProjectSlug
	state.Name = types.StringValue(profile.CertificateProfile.Slug)
	state.Description = types.StringValue(profile.CertificateProfile.Description)
	state.EnrollmentType = types.StringValue(profile.CertificateProfile.EnrollmentType)
	state.IssuerType = types.StringValue(profile.CertificateProfile.IssuerType)
	state.CertificatePolicyId = types.StringValue(profile.CertificateProfile.CertificatePolicyId)

	if profile.CertificateProfile.CaId != "" {
		state.CaId = types.StringValue(profile.CertificateProfile.CaId)
	}

	if profile.CertificateProfile.EstConfig != nil {
		state.EstConfig = &certManagerCertificateProfileEstConfigModel{
			DisableBootstrapCaValidation: types.BoolValue(profile.CertificateProfile.EstConfig.DisableBootstrapCaValidation),
		}

		if currentState.EstConfig != nil && !currentState.EstConfig.Passphrase.IsNull() {
			state.EstConfig.Passphrase = currentState.EstConfig.Passphrase
		} else {
			state.EstConfig.Passphrase = types.StringNull()
		}

		if profile.CertificateProfile.EstConfig.CaChain != "" {
			state.EstConfig.CaChain = types.StringValue(profile.CertificateProfile.EstConfig.CaChain)
		} else {
			state.EstConfig.CaChain = types.StringNull()
		}
	} else {
		state.EstConfig = nil
	}

	if currentState.ApiConfig != nil {
		if profile.CertificateProfile.ApiConfig != nil {
			state.ApiConfig = &certManagerCertificateProfileApiConfigModel{
				AutoRenew:       types.BoolValue(profile.CertificateProfile.ApiConfig.AutoRenew),
				RenewBeforeDays: types.Int64Value(int64(profile.CertificateProfile.ApiConfig.RenewBeforeDays)),
			}
		} else {
			state.ApiConfig = &certManagerCertificateProfileApiConfigModel{
				AutoRenew:       types.BoolNull(),
				RenewBeforeDays: types.Int64Null(),
			}
		}
	}

	if profile.CertificateProfile.ExternalConfigs != nil {
		state.ExternalConfigs = &certManagerCertificateProfileExternalConfigsModel{
			Template: types.StringValue(profile.CertificateProfile.ExternalConfigs.Template),
		}
	} else {
		state.ExternalConfigs = nil
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerCertificateProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update certificate profile",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerCertificateProfileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateProfileRequest := infisical.UpdateCertificateProfileRequest{
		ProfileId:   plan.Id.ValueString(),
		Slug:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if plan.EstConfig != nil {
		updateProfileRequest.EstConfig = &infisical.CertificateProfileEstConfig{
			DisableBootstrapCaValidation: plan.EstConfig.DisableBootstrapCaValidation.ValueBool(),
			Passphrase:                   plan.EstConfig.Passphrase.ValueString(),
		}
		if !plan.EstConfig.CaChain.IsNull() {
			updateProfileRequest.EstConfig.CaChain = plan.EstConfig.CaChain.ValueString()
		}
	}

	if plan.ApiConfig != nil {
		updateProfileRequest.ApiConfig = &infisical.CertificateProfileApiConfig{
			AutoRenew: plan.ApiConfig.AutoRenew.ValueBool(),
		}
		if !plan.ApiConfig.RenewBeforeDays.IsNull() {
			updateProfileRequest.ApiConfig.RenewBeforeDays = int(plan.ApiConfig.RenewBeforeDays.ValueInt64())
		}
	}

	if plan.ExternalConfigs != nil && !plan.ExternalConfigs.Template.IsNull() {
		updateProfileRequest.ExternalConfigs = &infisical.CertificateProfileExternalConfigs{
			Template: plan.ExternalConfigs.Template.ValueString(),
		}
	}

	_, err := r.client.UpdateCertificateProfile(updateProfileRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating certificate profile", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificateProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete certificate profile",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerCertificateProfileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteCertificateProfile(infisical.DeleteCertificateProfileRequest{
		ProfileId: state.Id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting certificate profile", err.Error())
		return
	}
}

func (r *certManagerCertificateProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
