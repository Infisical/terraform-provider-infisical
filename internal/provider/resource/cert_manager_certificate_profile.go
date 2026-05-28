package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	SUPPORTED_ISSUER_TYPES = []string{"ca", "self-signed"}
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

type certManagerCertificateProfileDefaultsModel struct {
	CommonName         types.String `tfsdk:"common_name"`
	TtlDays            types.Int64  `tfsdk:"ttl_days"`
	KeyAlgorithm       types.String `tfsdk:"key_algorithm"`
	SignatureAlgorithm types.String `tfsdk:"signature_algorithm"`
	KeyUsages          types.List   `tfsdk:"key_usages"`
	ExtendedKeyUsages  types.List   `tfsdk:"extended_key_usages"`
	Organization       types.String `tfsdk:"organization"`
	OrganizationalUnit types.String `tfsdk:"organizational_unit"`
	Country            types.String `tfsdk:"country"`
	State              types.String `tfsdk:"state"`
	Locality           types.String `tfsdk:"locality"`
}

type certManagerCertificateProfileResourceModel struct {
	Id                  types.String                                `tfsdk:"id"`
	CaId                types.String                                `tfsdk:"ca_id"`
	CertificatePolicyId types.String                                `tfsdk:"certificate_policy_id"`
	Name                types.String                                `tfsdk:"name" json:"slug"`
	Description         types.String                                `tfsdk:"description"`
	IssuerType          types.String                                `tfsdk:"issuer_type"`
	Defaults            *certManagerCertificateProfileDefaultsModel `tfsdk:"defaults"`
}

func (r *certManagerCertificateProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_certificate_profile"
}

func (r *certManagerCertificateProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage certificate profiles in Certificate Manager. Enrollment methods are configured on the application via the `infisical_cert_manager_application_profile` resource. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
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
			},
			"description": schema.StringAttribute{
				Description: "The description of the certificate profile",
				Optional:    true,
			},
			"issuer_type": schema.StringAttribute{
				Description: "The issuer type for the profile. Supported values: " + strings.Join(SUPPORTED_ISSUER_TYPES, ", ") + ". Defaults to 'ca'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ca"),
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_ISSUER_TYPES...),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"defaults": schema.SingleNestedBlock{
				Description: "Default certificate attribute values applied when issuing certificates from this profile",
				Attributes: map[string]schema.Attribute{
					"common_name": schema.StringAttribute{
						Description: "Default common name",
						Optional:    true,
					},
					"ttl_days": schema.Int64Attribute{
						Description: "Default certificate validity in days",
						Optional:    true,
					},
					"key_algorithm": schema.StringAttribute{
						Description: "Default key algorithm. Supported values: " + strings.Join(SUPPORTED_CERT_ISSUE_KEY_ALGORITHMS, ", "),
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_KEY_ALGORITHMS...),
						},
					},
					"signature_algorithm": schema.StringAttribute{
						Description: "Default signature algorithm. Supported values: " + strings.Join(SUPPORTED_CERT_ISSUE_SIGNATURE_ALGORITHMS, ", "),
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_SIGNATURE_ALGORITHMS...),
						},
					},
					"key_usages": schema.ListAttribute{
						Description: "Default key usages. Supported values: " + strings.Join(SUPPORTED_CERT_ISSUE_KEY_USAGES, ", "),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_KEY_USAGES...)),
						},
					},
					"extended_key_usages": schema.ListAttribute{
						Description: "Default extended key usages. Supported values: " + strings.Join(SUPPORTED_CERT_ISSUE_EXTENDED_KEY_USAGES, ", "),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_EXTENDED_KEY_USAGES...)),
						},
					},
					"organization": schema.StringAttribute{
						Description: "Default organization (O)",
						Optional:    true,
					},
					"organizational_unit": schema.StringAttribute{
						Description: "Default organizational unit (OU)",
						Optional:    true,
					},
					"country": schema.StringAttribute{
						Description: "Default country (C)",
						Optional:    true,
					},
					"state": schema.StringAttribute{
						Description: "Default state/province (ST)",
						Optional:    true,
					},
					"locality": schema.StringAttribute{
						Description: "Default locality (L)",
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

func (r *certManagerCertificateProfileResource) buildDefaults(ctx context.Context, plan *certManagerCertificateProfileResourceModel, diags *diag.Diagnostics) *infisical.CertificateProfileDefaults {
	if plan.Defaults == nil {
		return nil
	}

	result := &infisical.CertificateProfileDefaults{}
	d := plan.Defaults

	if !d.CommonName.IsNull() && !d.CommonName.IsUnknown() {
		result.CommonName = d.CommonName.ValueString()
	}
	if !d.TtlDays.IsNull() && !d.TtlDays.IsUnknown() {
		days := int(d.TtlDays.ValueInt64())
		result.TtlDays = &days
	}
	if !d.KeyAlgorithm.IsNull() && !d.KeyAlgorithm.IsUnknown() {
		result.KeyAlgorithm = d.KeyAlgorithm.ValueString()
	}
	if !d.SignatureAlgorithm.IsNull() && !d.SignatureAlgorithm.IsUnknown() {
		result.SignatureAlgorithm = d.SignatureAlgorithm.ValueString()
	}
	if !d.KeyUsages.IsNull() && !d.KeyUsages.IsUnknown() {
		values := make([]string, 0, len(d.KeyUsages.Elements()))
		diags.Append(d.KeyUsages.ElementsAs(ctx, &values, false)...)
		result.KeyUsages = values
	}
	if !d.ExtendedKeyUsages.IsNull() && !d.ExtendedKeyUsages.IsUnknown() {
		values := make([]string, 0, len(d.ExtendedKeyUsages.Elements()))
		diags.Append(d.ExtendedKeyUsages.ElementsAs(ctx, &values, false)...)
		result.ExtendedKeyUsages = values
	}
	if !d.Organization.IsNull() && !d.Organization.IsUnknown() {
		result.Organization = d.Organization.ValueString()
	}
	if !d.OrganizationalUnit.IsNull() && !d.OrganizationalUnit.IsUnknown() {
		result.OrganizationalUnit = d.OrganizationalUnit.ValueString()
	}
	if !d.Country.IsNull() && !d.Country.IsUnknown() {
		result.Country = d.Country.ValueString()
	}
	if !d.State.IsNull() && !d.State.IsUnknown() {
		result.State = d.State.ValueString()
	}
	if !d.Locality.IsNull() && !d.Locality.IsUnknown() {
		result.Locality = d.Locality.ValueString()
	}

	return result
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

	if plan.IssuerType.ValueString() == "self-signed" && !plan.CaId.IsNull() {
		resp.Diagnostics.AddError("Invalid CA ID for self-signed", "ca_id should not be specified when issuer_type is 'self-signed'")
		return
	}

	if plan.IssuerType.ValueString() == "ca" && plan.CaId.IsNull() {
		resp.Diagnostics.AddError("Missing CA ID", "ca_id is required when issuer_type is 'ca'")
		return
	}

	createProfileRequest := infisical.CreateCertificateProfileRequest{
		CertificatePolicyId: plan.CertificatePolicyId.ValueString(),
		Slug:                plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		d := plan.Description.ValueString()
		createProfileRequest.Description = &d
	}

	if !plan.CaId.IsNull() {
		createProfileRequest.CaId = plan.CaId.ValueString()
	}

	if !plan.IssuerType.IsNull() {
		createProfileRequest.IssuerType = plan.IssuerType.ValueString()
	}

	defaults := r.buildDefaults(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	createProfileRequest.Defaults = defaults

	profile, err := r.client.CreateCertificateProfile(createProfileRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating certificate profile", err.Error())
		return
	}

	plan.Id = types.StringValue(profile.CertificateProfile.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificateProfileResource) defaultsToState(ctx context.Context, src *infisical.CertificateProfileDefaults, diags *diag.Diagnostics) *certManagerCertificateProfileDefaultsModel {
	if src == nil {
		return nil
	}

	out := &certManagerCertificateProfileDefaultsModel{
		CommonName:         stringValueOrNull(src.CommonName),
		KeyAlgorithm:       stringValueOrNull(src.KeyAlgorithm),
		SignatureAlgorithm: stringValueOrNull(src.SignatureAlgorithm),
		Organization:       stringValueOrNull(src.Organization),
		OrganizationalUnit: stringValueOrNull(src.OrganizationalUnit),
		Country:            stringValueOrNull(src.Country),
		State:              stringValueOrNull(src.State),
		Locality:           stringValueOrNull(src.Locality),
	}

	if src.TtlDays != nil {
		out.TtlDays = types.Int64Value(int64(*src.TtlDays))
	} else {
		out.TtlDays = types.Int64Null()
	}

	if len(src.KeyUsages) > 0 {
		l, d := types.ListValueFrom(ctx, types.StringType, src.KeyUsages)
		diags.Append(d...)
		out.KeyUsages = l
	} else {
		out.KeyUsages = types.ListNull(types.StringType)
	}

	if len(src.ExtendedKeyUsages) > 0 {
		l, d := types.ListValueFrom(ctx, types.StringType, src.ExtendedKeyUsages)
		diags.Append(d...)
		out.ExtendedKeyUsages = l
	} else {
		out.ExtendedKeyUsages = types.ListNull(types.StringType)
	}

	return out
}

func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
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
		IncludeConfigs: false,
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
	state.Name = types.StringValue(profile.CertificateProfile.Slug)
	if profile.CertificateProfile.Description != nil {
		state.Description = types.StringValue(*profile.CertificateProfile.Description)
	} else {
		state.Description = types.StringNull()
	}
	state.IssuerType = types.StringValue(profile.CertificateProfile.IssuerType)
	state.CertificatePolicyId = types.StringValue(profile.CertificateProfile.CertificatePolicyId)

	if profile.CertificateProfile.CaId != "" {
		state.CaId = types.StringValue(profile.CertificateProfile.CaId)
	}

	state.Defaults = r.defaultsToState(ctx, profile.CertificateProfile.Defaults, &resp.Diagnostics)

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
		ProfileId:  plan.Id.ValueString(),
		Slug:       plan.Name.ValueString(),
		IssuerType: plan.IssuerType.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		d := plan.Description.ValueString()
		updateProfileRequest.Description = &d
	}

	defaults := r.buildDefaults(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updateProfileRequest.Defaults = defaults

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
