package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	SUPPORTED_SUBJECT_TYPES   = []string{"common_name", "organization", "country"}
	SUPPORTED_SAN_TYPES       = []string{"dns_name", "ip_address", "email", "uri"}
	SUPPORTED_CERT_KEY_USAGES = []string{
		"digital_signature", "key_encipherment", "non_repudiation",
		"data_encipherment", "key_agreement", "key_cert_sign",
		"crl_sign", "encipher_only", "decipher_only",
	}
	SUPPORTED_CERT_EXT_KEY_USAGES = []string{
		"client_auth", "server_auth", "code_signing",
		"email_protection", "ocsp_signing", "time_stamping",
	}
	SUPPORTED_SIGNATURE_ALGORITHMS = []string{
		"SHA256-RSA", "SHA512-RSA", "SHA384-ECDSA",
		"SHA384-RSA", "SHA256-ECDSA", "SHA512-ECDSA",
	}
	SUPPORTED_KEY_ALGORITHMS = []string{
		"RSA-2048", "RSA-3072", "RSA-4096",
		"ECDSA-P256", "ECDSA-P521", "ECDSA-P384",
	}
)

var (
	_ resource.Resource = &certManagerCertificatePolicyResource{}
)

func NewCertManagerCertificatePolicyResource() resource.Resource {
	return &certManagerCertificatePolicyResource{}
}

type certManagerCertificatePolicyResource struct {
	client *infisical.Client
}

type certManagerCertificatePolicySubjectModel struct {
	Type     types.String `tfsdk:"type"`
	Allowed  types.List   `tfsdk:"allowed"`
	Required types.List   `tfsdk:"required"`
	Denied   types.List   `tfsdk:"denied"`
}

type certManagerCertificatePolicySanModel struct {
	Type     types.String `tfsdk:"type"`
	Allowed  types.List   `tfsdk:"allowed"`
	Required types.List   `tfsdk:"required"`
	Denied   types.List   `tfsdk:"denied"`
}

type certManagerCertificatePolicyKeyUsagesModel struct {
	Allowed  types.List `tfsdk:"allowed"`
	Required types.List `tfsdk:"required"`
	Denied   types.List `tfsdk:"denied"`
}

type certManagerCertificatePolicyExtendedKeyUsagesModel struct {
	Allowed  types.List `tfsdk:"allowed"`
	Required types.List `tfsdk:"required"`
	Denied   types.List `tfsdk:"denied"`
}

type certManagerCertificatePolicyAlgorithmsModel struct {
	Signature    types.List `tfsdk:"signature"`
	KeyAlgorithm types.List `tfsdk:"key_algorithm"`
}

type certManagerCertificatePolicyValidityModel struct {
	Max types.String `tfsdk:"max"`
}

type certManagerCertificatePolicyResourceModel struct {
	ProjectSlug       types.String                                         `tfsdk:"project_slug"`
	Id                types.String                                         `tfsdk:"id"`
	Name              types.String                                         `tfsdk:"name"`
	Description       types.String                                         `tfsdk:"description"`
	Subject           []certManagerCertificatePolicySubjectModel           `tfsdk:"subject"`
	Sans              []certManagerCertificatePolicySanModel               `tfsdk:"sans"`
	KeyUsages         *certManagerCertificatePolicyKeyUsagesModel          `tfsdk:"key_usages"`
	ExtendedKeyUsages *certManagerCertificatePolicyExtendedKeyUsagesModel `tfsdk:"extended_key_usages"`
	Algorithms        *certManagerCertificatePolicyAlgorithmsModel         `tfsdk:"algorithms"`
	Validity          *certManagerCertificatePolicyValidityModel           `tfsdk:"validity"`
}

func (r *certManagerCertificatePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_certificate_policy"
}

func (r *certManagerCertificatePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage certificate policies in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"project_slug": schema.StringAttribute{
				Description: "The slug of the cert-manager project",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The ID of the certificate policy",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the certificate policy",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the certificate policy",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"subject": schema.ListNestedBlock{
				Description: "Subject attribute policies for the certificate policy",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The subject attribute type. Possible values: " + strings.Join(SUPPORTED_SUBJECT_TYPES, ", "),
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(SUPPORTED_SUBJECT_TYPES...),
							},
						},
						"allowed": schema.ListAttribute{
							Description: "List of allowed values for this subject attribute",
							Optional:    true,
							ElementType: types.StringType,
						},
						"required": schema.ListAttribute{
							Description: "List of required values for this subject attribute",
							Optional:    true,
							ElementType: types.StringType,
						},
						"denied": schema.ListAttribute{
							Description: "List of denied values for this subject attribute",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"sans": schema.ListNestedBlock{
				Description: "Subject alternative name (SAN) policies for the certificate policy",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The SAN type. Possible values: " + strings.Join(SUPPORTED_SAN_TYPES, ", "),
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(SUPPORTED_SAN_TYPES...),
							},
						},
						"allowed": schema.ListAttribute{
							Description: "List of allowed values for this SAN type",
							Optional:    true,
							ElementType: types.StringType,
						},
						"required": schema.ListAttribute{
							Description: "List of required values for this SAN type",
							Optional:    true,
							ElementType: types.StringType,
						},
						"denied": schema.ListAttribute{
							Description: "List of denied values for this SAN type",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"key_usages": schema.SingleNestedBlock{
				Description: "Key usage policies for the certificate policy",
				Attributes: map[string]schema.Attribute{
					"allowed": schema.ListAttribute{
						Description: "List of allowed key usages. Possible values: " + strings.Join(SUPPORTED_CERT_KEY_USAGES, ", "),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_KEY_USAGES...)),
						},
					},
					"required": schema.ListAttribute{
						Description: "List of required key usages. Possible values: " + strings.Join(SUPPORTED_CERT_KEY_USAGES, ", "),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_KEY_USAGES...)),
						},
					},
					"denied": schema.ListAttribute{
						Description: "List of denied key usages. Possible values: " + strings.Join(SUPPORTED_CERT_KEY_USAGES, ", "),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_KEY_USAGES...)),
						},
					},
				},
			},
			"extended_key_usages": schema.SingleNestedBlock{
				Description: "Extended key usage policies for the certificate policy",
				Attributes: map[string]schema.Attribute{
					"allowed": schema.ListAttribute{
						Description: "List of allowed extended key usages. Possible values: " + strings.Join(SUPPORTED_CERT_EXT_KEY_USAGES, ", "),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_EXT_KEY_USAGES...)),
						},
					},
					"required": schema.ListAttribute{
						Description: "List of required extended key usages. Possible values: " + strings.Join(SUPPORTED_CERT_EXT_KEY_USAGES, ", "),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_EXT_KEY_USAGES...)),
						},
					},
					"denied": schema.ListAttribute{
						Description: "List of denied extended key usages. Possible values: " + strings.Join(SUPPORTED_CERT_EXT_KEY_USAGES, ", "),
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_EXT_KEY_USAGES...)),
						},
					},
				},
			},
			"algorithms": schema.SingleNestedBlock{
				Description: "Algorithm constraints for the certificate policy. At least one signature algorithm and one key algorithm must be specified.",
				Attributes: map[string]schema.Attribute{
					"signature": schema.ListAttribute{
						Description: "List of allowed signature algorithms (at least one required). Supported values: " + strings.Join(SUPPORTED_SIGNATURE_ALGORITHMS, ", "),
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_SIGNATURE_ALGORITHMS...)),
						},
					},
					"key_algorithm": schema.ListAttribute{
						Description: "List of allowed key algorithms (at least one required). Supported values: " + strings.Join(SUPPORTED_KEY_ALGORITHMS, ", "),
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_KEY_ALGORITHMS...)),
						},
					},
				},
			},
			"validity": schema.SingleNestedBlock{
				Description: "Validity constraints for the certificate policy",
				Attributes: map[string]schema.Attribute{
					"max": schema.StringAttribute{
						Description: "Maximum validity period (e.g., '90d', '2y', '6m')",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *certManagerCertificatePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerCertificatePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create certificate policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerCertificatePolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.GetProject(infisical.GetProjectRequest{
		Slug: plan.ProjectSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error finding project by slug", err.Error())
		return
	}

	createPolicyRequest := infisical.CreateCertificatePolicyRequest{
		ProjectId:   project.ID,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if len(plan.Subject) > 0 {
		createPolicyRequest.Subject = make([]infisical.CertificatePolicySubject, len(plan.Subject))
		for i, subj := range plan.Subject {
			createPolicyRequest.Subject[i] = infisical.CertificatePolicySubject{
				Type: subj.Type.ValueString(),
			}

			if !subj.Allowed.IsNull() {
				allowed := make([]string, 0, len(subj.Allowed.Elements()))
				resp.Diagnostics.Append(subj.Allowed.ElementsAs(ctx, &allowed, false)...)
				createPolicyRequest.Subject[i].Allowed = allowed
			}

			if !subj.Required.IsNull() {
				required := make([]string, 0, len(subj.Required.Elements()))
				resp.Diagnostics.Append(subj.Required.ElementsAs(ctx, &required, false)...)
				createPolicyRequest.Subject[i].Required = required
			}

			if !subj.Denied.IsNull() {
				denied := make([]string, 0, len(subj.Denied.Elements()))
				resp.Diagnostics.Append(subj.Denied.ElementsAs(ctx, &denied, false)...)
				createPolicyRequest.Subject[i].Denied = denied
			}
		}
	}

	if len(plan.Sans) > 0 {
		createPolicyRequest.Sans = make([]infisical.CertificatePolicySAN, len(plan.Sans))
		for i, san := range plan.Sans {
			createPolicyRequest.Sans[i] = infisical.CertificatePolicySAN{
				Type: san.Type.ValueString(),
			}

			if !san.Allowed.IsNull() {
				allowed := make([]string, 0, len(san.Allowed.Elements()))
				resp.Diagnostics.Append(san.Allowed.ElementsAs(ctx, &allowed, false)...)
				createPolicyRequest.Sans[i].Allowed = allowed
			}

			if !san.Required.IsNull() {
				required := make([]string, 0, len(san.Required.Elements()))
				resp.Diagnostics.Append(san.Required.ElementsAs(ctx, &required, false)...)
				createPolicyRequest.Sans[i].Required = required
			}

			if !san.Denied.IsNull() {
				denied := make([]string, 0, len(san.Denied.Elements()))
				resp.Diagnostics.Append(san.Denied.ElementsAs(ctx, &denied, false)...)
				createPolicyRequest.Sans[i].Denied = denied
			}
		}
	}

	if plan.KeyUsages != nil {
		createPolicyRequest.KeyUsages = &infisical.CertificatePolicyKeyUsages{}

		if !plan.KeyUsages.Allowed.IsNull() {
			allowed := make([]string, 0, len(plan.KeyUsages.Allowed.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Allowed.ElementsAs(ctx, &allowed, false)...)
			createPolicyRequest.KeyUsages.Allowed = allowed
		}

		if !plan.KeyUsages.Required.IsNull() {
			required := make([]string, 0, len(plan.KeyUsages.Required.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Required.ElementsAs(ctx, &required, false)...)
			createPolicyRequest.KeyUsages.Required = required
		}

		if !plan.KeyUsages.Denied.IsNull() {
			denied := make([]string, 0, len(plan.KeyUsages.Denied.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Denied.ElementsAs(ctx, &denied, false)...)
			createPolicyRequest.KeyUsages.Denied = denied
		}
	}

	if plan.ExtendedKeyUsages != nil {
		createPolicyRequest.ExtendedKeyUsages = &infisical.CertificatePolicyExtendedKeyUsages{}

		if !plan.ExtendedKeyUsages.Allowed.IsNull() {
			allowed := make([]string, 0, len(plan.ExtendedKeyUsages.Allowed.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Allowed.ElementsAs(ctx, &allowed, false)...)
			createPolicyRequest.ExtendedKeyUsages.Allowed = allowed
		}

		if !plan.ExtendedKeyUsages.Required.IsNull() {
			required := make([]string, 0, len(plan.ExtendedKeyUsages.Required.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Required.ElementsAs(ctx, &required, false)...)
			createPolicyRequest.ExtendedKeyUsages.Required = required
		}

		if !plan.ExtendedKeyUsages.Denied.IsNull() {
			denied := make([]string, 0, len(plan.ExtendedKeyUsages.Denied.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Denied.ElementsAs(ctx, &denied, false)...)
			createPolicyRequest.ExtendedKeyUsages.Denied = denied
		}
	}

	if plan.Algorithms != nil {
		createPolicyRequest.Algorithms = &infisical.CertificatePolicyAlgorithms{}

		if !plan.Algorithms.Signature.IsNull() {
			signature := make([]string, 0, len(plan.Algorithms.Signature.Elements()))
			resp.Diagnostics.Append(plan.Algorithms.Signature.ElementsAs(ctx, &signature, false)...)
			createPolicyRequest.Algorithms.Signature = signature
		}

		if !plan.Algorithms.KeyAlgorithm.IsNull() {
			keyAlgorithm := make([]string, 0, len(plan.Algorithms.KeyAlgorithm.Elements()))
			resp.Diagnostics.Append(plan.Algorithms.KeyAlgorithm.ElementsAs(ctx, &keyAlgorithm, false)...)
			createPolicyRequest.Algorithms.KeyAlgorithm = keyAlgorithm
		}
	}

	if plan.Validity != nil && !plan.Validity.Max.IsNull() {
		createPolicyRequest.Validity = &infisical.CertificatePolicyValidity{
			Max: plan.Validity.Max.ValueString(),
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.CreateCertificatePolicy(createPolicyRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating certificate policy", err.Error())
		return
	}

	plan.Id = types.StringValue(policy.CertificatePolicy.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificatePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read certificate policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var currentState certManagerCertificatePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerCertificatePolicyResourceModel

	policy, err := r.client.GetCertificatePolicy(infisical.GetCertificatePolicyRequest{
		PolicyId: currentState.Id.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading certificate policy", err.Error())
		return
	}

	state.Id = types.StringValue(policy.CertificatePolicy.Id)
	state.ProjectSlug = currentState.ProjectSlug
	state.Name = types.StringValue(policy.CertificatePolicy.Name)
	if policy.CertificatePolicy.Description != "" {
		state.Description = types.StringValue(policy.CertificatePolicy.Description)
	} else {
		state.Description = types.StringNull()
	}

	if len(policy.CertificatePolicy.Subject) > 0 {
		state.Subject = make([]certManagerCertificatePolicySubjectModel, len(policy.CertificatePolicy.Subject))
		for i, subj := range policy.CertificatePolicy.Subject {
			state.Subject[i] = certManagerCertificatePolicySubjectModel{
				Type: types.StringValue(subj.Type),
			}

			if len(subj.Allowed) > 0 {
				allowedList, diags := types.ListValueFrom(ctx, types.StringType, subj.Allowed)
				resp.Diagnostics.Append(diags...)
				state.Subject[i].Allowed = allowedList
			} else {
				state.Subject[i].Allowed = types.ListNull(types.StringType)
			}

			if len(subj.Required) > 0 {
				requiredList, diags := types.ListValueFrom(ctx, types.StringType, subj.Required)
				resp.Diagnostics.Append(diags...)
				state.Subject[i].Required = requiredList
			} else {
				if len(currentState.Subject) > i && !currentState.Subject[i].Required.IsNull() {
					state.Subject[i].Required = currentState.Subject[i].Required
				} else {
					state.Subject[i].Required = types.ListNull(types.StringType)
				}
			}

			if len(subj.Denied) > 0 {
				deniedList, diags := types.ListValueFrom(ctx, types.StringType, subj.Denied)
				resp.Diagnostics.Append(diags...)
				state.Subject[i].Denied = deniedList
			} else {
				state.Subject[i].Denied = types.ListNull(types.StringType)
			}
		}
	}

	if len(policy.CertificatePolicy.Sans) > 0 {
		state.Sans = make([]certManagerCertificatePolicySanModel, len(policy.CertificatePolicy.Sans))
		for i, san := range policy.CertificatePolicy.Sans {
			state.Sans[i] = certManagerCertificatePolicySanModel{
				Type: types.StringValue(san.Type),
			}

			if len(san.Allowed) > 0 {
				allowedList, diags := types.ListValueFrom(ctx, types.StringType, san.Allowed)
				resp.Diagnostics.Append(diags...)
				state.Sans[i].Allowed = allowedList
			} else {
				state.Sans[i].Allowed = types.ListNull(types.StringType)
			}

			if len(san.Required) > 0 {
				requiredList, diags := types.ListValueFrom(ctx, types.StringType, san.Required)
				resp.Diagnostics.Append(diags...)
				state.Sans[i].Required = requiredList
			} else {
				state.Sans[i].Required = types.ListNull(types.StringType)
			}

			if len(san.Denied) > 0 {
				deniedList, diags := types.ListValueFrom(ctx, types.StringType, san.Denied)
				resp.Diagnostics.Append(diags...)
				state.Sans[i].Denied = deniedList
			} else {
				state.Sans[i].Denied = types.ListNull(types.StringType)
			}
		}
	}

	if currentState.KeyUsages != nil {
		state.KeyUsages = &certManagerCertificatePolicyKeyUsagesModel{}

		if policy.CertificatePolicy.KeyUsages != nil {
			if len(policy.CertificatePolicy.KeyUsages.Allowed) > 0 {
				allowedList, diags := types.ListValueFrom(ctx, types.StringType, policy.CertificatePolicy.KeyUsages.Allowed)
				resp.Diagnostics.Append(diags...)
				state.KeyUsages.Allowed = allowedList
			} else {
				state.KeyUsages.Allowed = types.ListNull(types.StringType)
			}

			if len(policy.CertificatePolicy.KeyUsages.Required) > 0 {
				requiredList, diags := types.ListValueFrom(ctx, types.StringType, policy.CertificatePolicy.KeyUsages.Required)
				resp.Diagnostics.Append(diags...)
				state.KeyUsages.Required = requiredList
			} else {
				state.KeyUsages.Required = types.ListNull(types.StringType)
			}

			if len(policy.CertificatePolicy.KeyUsages.Denied) > 0 {
				deniedList, diags := types.ListValueFrom(ctx, types.StringType, policy.CertificatePolicy.KeyUsages.Denied)
				resp.Diagnostics.Append(diags...)
				state.KeyUsages.Denied = deniedList
			} else {
				state.KeyUsages.Denied = types.ListNull(types.StringType)
			}
		} else {
			state.KeyUsages.Allowed = types.ListNull(types.StringType)
			state.KeyUsages.Required = types.ListNull(types.StringType)
			state.KeyUsages.Denied = types.ListNull(types.StringType)
		}
	}

	if currentState.ExtendedKeyUsages != nil {
		state.ExtendedKeyUsages = &certManagerCertificatePolicyExtendedKeyUsagesModel{}

		if policy.CertificatePolicy.ExtendedKeyUsages != nil {
			if len(policy.CertificatePolicy.ExtendedKeyUsages.Allowed) > 0 {
				allowedList, diags := types.ListValueFrom(ctx, types.StringType, policy.CertificatePolicy.ExtendedKeyUsages.Allowed)
				resp.Diagnostics.Append(diags...)
				state.ExtendedKeyUsages.Allowed = allowedList
			} else {
				state.ExtendedKeyUsages.Allowed = types.ListNull(types.StringType)
			}

			if len(policy.CertificatePolicy.ExtendedKeyUsages.Required) > 0 {
				requiredList, diags := types.ListValueFrom(ctx, types.StringType, policy.CertificatePolicy.ExtendedKeyUsages.Required)
				resp.Diagnostics.Append(diags...)
				state.ExtendedKeyUsages.Required = requiredList
			} else {
				state.ExtendedKeyUsages.Required = types.ListNull(types.StringType)
			}

			if len(policy.CertificatePolicy.ExtendedKeyUsages.Denied) > 0 {
				deniedList, diags := types.ListValueFrom(ctx, types.StringType, policy.CertificatePolicy.ExtendedKeyUsages.Denied)
				resp.Diagnostics.Append(diags...)
				state.ExtendedKeyUsages.Denied = deniedList
			} else {
				state.ExtendedKeyUsages.Denied = types.ListNull(types.StringType)
			}
		} else {
			state.ExtendedKeyUsages.Allowed = types.ListNull(types.StringType)
			state.ExtendedKeyUsages.Required = types.ListNull(types.StringType)
			state.ExtendedKeyUsages.Denied = types.ListNull(types.StringType)
		}
	}

	if policy.CertificatePolicy.Algorithms != nil {
		state.Algorithms = &certManagerCertificatePolicyAlgorithmsModel{}

		if len(policy.CertificatePolicy.Algorithms.Signature) > 0 {
			signatureList, diags := types.ListValueFrom(ctx, types.StringType, policy.CertificatePolicy.Algorithms.Signature)
			resp.Diagnostics.Append(diags...)
			state.Algorithms.Signature = signatureList
		} else {
			state.Algorithms.Signature = types.ListNull(types.StringType)
		}

		if len(policy.CertificatePolicy.Algorithms.KeyAlgorithm) > 0 {
			keyAlgorithmList, diags := types.ListValueFrom(ctx, types.StringType, policy.CertificatePolicy.Algorithms.KeyAlgorithm)
			resp.Diagnostics.Append(diags...)
			state.Algorithms.KeyAlgorithm = keyAlgorithmList
		} else {
			state.Algorithms.KeyAlgorithm = types.ListNull(types.StringType)
		}
	}

	if policy.CertificatePolicy.Validity != nil && policy.CertificatePolicy.Validity.Max != "" {
		state.Validity = &certManagerCertificatePolicyValidityModel{
			Max: types.StringValue(policy.CertificatePolicy.Validity.Max),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerCertificatePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update certificate policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerCertificatePolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatePolicyRequest := infisical.UpdateCertificatePolicyRequest{
		PolicyId:    plan.Id.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if len(plan.Subject) > 0 {
		updatePolicyRequest.Subject = make([]infisical.CertificatePolicySubject, len(plan.Subject))
		for i, subj := range plan.Subject {
			updatePolicyRequest.Subject[i] = infisical.CertificatePolicySubject{
				Type: subj.Type.ValueString(),
			}

			if !subj.Allowed.IsNull() {
				allowed := make([]string, 0, len(subj.Allowed.Elements()))
				resp.Diagnostics.Append(subj.Allowed.ElementsAs(ctx, &allowed, false)...)
				updatePolicyRequest.Subject[i].Allowed = allowed
			}

			if !subj.Required.IsNull() {
				required := make([]string, 0, len(subj.Required.Elements()))
				resp.Diagnostics.Append(subj.Required.ElementsAs(ctx, &required, false)...)
				updatePolicyRequest.Subject[i].Required = required
			}

			if !subj.Denied.IsNull() {
				denied := make([]string, 0, len(subj.Denied.Elements()))
				resp.Diagnostics.Append(subj.Denied.ElementsAs(ctx, &denied, false)...)
				updatePolicyRequest.Subject[i].Denied = denied
			}
		}
	}

	if len(plan.Sans) > 0 {
		updatePolicyRequest.Sans = make([]infisical.CertificatePolicySAN, len(plan.Sans))
		for i, san := range plan.Sans {
			updatePolicyRequest.Sans[i] = infisical.CertificatePolicySAN{
				Type: san.Type.ValueString(),
			}

			if !san.Allowed.IsNull() {
				allowed := make([]string, 0, len(san.Allowed.Elements()))
				resp.Diagnostics.Append(san.Allowed.ElementsAs(ctx, &allowed, false)...)
				updatePolicyRequest.Sans[i].Allowed = allowed
			}

			if !san.Required.IsNull() {
				required := make([]string, 0, len(san.Required.Elements()))
				resp.Diagnostics.Append(san.Required.ElementsAs(ctx, &required, false)...)
				updatePolicyRequest.Sans[i].Required = required
			}

			if !san.Denied.IsNull() {
				denied := make([]string, 0, len(san.Denied.Elements()))
				resp.Diagnostics.Append(san.Denied.ElementsAs(ctx, &denied, false)...)
				updatePolicyRequest.Sans[i].Denied = denied
			}
		}
	}

	if plan.KeyUsages != nil {
		updatePolicyRequest.KeyUsages = &infisical.CertificatePolicyKeyUsages{}

		if !plan.KeyUsages.Allowed.IsNull() {
			allowed := make([]string, 0, len(plan.KeyUsages.Allowed.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Allowed.ElementsAs(ctx, &allowed, false)...)
			updatePolicyRequest.KeyUsages.Allowed = allowed
		}

		if !plan.KeyUsages.Required.IsNull() {
			required := make([]string, 0, len(plan.KeyUsages.Required.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Required.ElementsAs(ctx, &required, false)...)
			updatePolicyRequest.KeyUsages.Required = required
		}

		if !plan.KeyUsages.Denied.IsNull() {
			denied := make([]string, 0, len(plan.KeyUsages.Denied.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Denied.ElementsAs(ctx, &denied, false)...)
			updatePolicyRequest.KeyUsages.Denied = denied
		}
	}

	if plan.ExtendedKeyUsages != nil {
		updatePolicyRequest.ExtendedKeyUsages = &infisical.CertificatePolicyExtendedKeyUsages{}

		if !plan.ExtendedKeyUsages.Allowed.IsNull() {
			allowed := make([]string, 0, len(plan.ExtendedKeyUsages.Allowed.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Allowed.ElementsAs(ctx, &allowed, false)...)
			updatePolicyRequest.ExtendedKeyUsages.Allowed = allowed
		}

		if !plan.ExtendedKeyUsages.Required.IsNull() {
			required := make([]string, 0, len(plan.ExtendedKeyUsages.Required.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Required.ElementsAs(ctx, &required, false)...)
			updatePolicyRequest.ExtendedKeyUsages.Required = required
		}

		if !plan.ExtendedKeyUsages.Denied.IsNull() {
			denied := make([]string, 0, len(plan.ExtendedKeyUsages.Denied.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Denied.ElementsAs(ctx, &denied, false)...)
			updatePolicyRequest.ExtendedKeyUsages.Denied = denied
		}
	}

	if plan.Algorithms != nil {
		updatePolicyRequest.Algorithms = &infisical.CertificatePolicyAlgorithms{}

		if !plan.Algorithms.Signature.IsNull() {
			signature := make([]string, 0, len(plan.Algorithms.Signature.Elements()))
			resp.Diagnostics.Append(plan.Algorithms.Signature.ElementsAs(ctx, &signature, false)...)
			updatePolicyRequest.Algorithms.Signature = signature
		}

		if !plan.Algorithms.KeyAlgorithm.IsNull() {
			keyAlgorithm := make([]string, 0, len(plan.Algorithms.KeyAlgorithm.Elements()))
			resp.Diagnostics.Append(plan.Algorithms.KeyAlgorithm.ElementsAs(ctx, &keyAlgorithm, false)...)
			updatePolicyRequest.Algorithms.KeyAlgorithm = keyAlgorithm
		}
	}

	if plan.Validity != nil && !plan.Validity.Max.IsNull() {
		updatePolicyRequest.Validity = &infisical.CertificatePolicyValidity{
			Max: plan.Validity.Max.ValueString(),
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateCertificatePolicy(updatePolicyRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating certificate policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificatePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete certificate policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerCertificatePolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteCertificatePolicy(infisical.DeleteCertificatePolicyRequest{
		PolicyId: state.Id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting certificate policy", err.Error())
		return
	}
}

func (r *certManagerCertificatePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
