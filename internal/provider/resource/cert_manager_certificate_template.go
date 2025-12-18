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
	_ resource.Resource = &certManagerCertificateTemplateResource{}
)

func NewCertManagerCertificateTemplateResource() resource.Resource {
	return &certManagerCertificateTemplateResource{}
}

type certManagerCertificateTemplateResource struct {
	client *infisical.Client
}

type certManagerCertificateTemplateSubjectModel struct {
	Type     types.String `tfsdk:"type"`
	Allowed  types.List   `tfsdk:"allowed"`
	Required types.List   `tfsdk:"required"`
	Denied   types.List   `tfsdk:"denied"`
}

type certManagerCertificateTemplateSanModel struct {
	Type     types.String `tfsdk:"type"`
	Allowed  types.List   `tfsdk:"allowed"`
	Required types.List   `tfsdk:"required"`
	Denied   types.List   `tfsdk:"denied"`
}

type certManagerCertificateTemplateKeyUsagesModel struct {
	Allowed  types.List `tfsdk:"allowed"`
	Required types.List `tfsdk:"required"`
	Denied   types.List `tfsdk:"denied"`
}

type certManagerCertificateTemplateExtendedKeyUsagesModel struct {
	Allowed  types.List `tfsdk:"allowed"`
	Required types.List `tfsdk:"required"`
	Denied   types.List `tfsdk:"denied"`
}

type certManagerCertificateTemplateAlgorithmsModel struct {
	Signature    types.List `tfsdk:"signature"`
	KeyAlgorithm types.List `tfsdk:"key_algorithm"`
}

type certManagerCertificateTemplateValidityModel struct {
	Max types.String `tfsdk:"max"`
}

type certManagerCertificateTemplateResourceModel struct {
	ProjectSlug       types.String                                          `tfsdk:"project_slug"`
	Id                types.String                                          `tfsdk:"id"`
	Name              types.String                                          `tfsdk:"name"`
	Description       types.String                                          `tfsdk:"description"`
	Subject           []certManagerCertificateTemplateSubjectModel          `tfsdk:"subject"`
	Sans              []certManagerCertificateTemplateSanModel              `tfsdk:"sans"`
	KeyUsages         *certManagerCertificateTemplateKeyUsagesModel         `tfsdk:"key_usages"`
	ExtendedKeyUsages *certManagerCertificateTemplateExtendedKeyUsagesModel `tfsdk:"extended_key_usages"`
	Algorithms        *certManagerCertificateTemplateAlgorithmsModel        `tfsdk:"algorithms"`
	Validity          *certManagerCertificateTemplateValidityModel          `tfsdk:"validity"`
}

func (r *certManagerCertificateTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_certificate_template"
}

func (r *certManagerCertificateTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage certificate templates in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"project_slug": schema.StringAttribute{
				Description: "The slug of the cert-manager project",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The ID of the certificate template",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the certificate template",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the certificate template",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"subject": schema.ListNestedBlock{
				Description: "Subject attribute policies for the certificate template",
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
				Description: "Subject alternative name (SAN) policies for the certificate template",
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
				Description: "Key usage policies for the certificate template",
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
				Description: "Extended key usage policies for the certificate template",
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
				Description: "Algorithm constraints for the certificate template. At least one signature algorithm and one key algorithm must be specified.",
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
				Description: "Validity constraints for the certificate template",
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

func (r *certManagerCertificateTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerCertificateTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create certificate template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerCertificateTemplateResourceModel

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

	createTemplateRequest := infisical.CreateCertificateTemplateRequest{
		ProjectId:   project.ID,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if len(plan.Subject) > 0 {
		createTemplateRequest.Subject = make([]infisical.CertificateTemplateSubject, len(plan.Subject))
		for i, subj := range plan.Subject {
			createTemplateRequest.Subject[i] = infisical.CertificateTemplateSubject{
				Type: subj.Type.ValueString(),
			}

			if !subj.Allowed.IsNull() {
				allowed := make([]string, 0, len(subj.Allowed.Elements()))
				resp.Diagnostics.Append(subj.Allowed.ElementsAs(ctx, &allowed, false)...)
				createTemplateRequest.Subject[i].Allowed = allowed
			}

			if !subj.Required.IsNull() {
				required := make([]string, 0, len(subj.Required.Elements()))
				resp.Diagnostics.Append(subj.Required.ElementsAs(ctx, &required, false)...)
				createTemplateRequest.Subject[i].Required = required
			}

			if !subj.Denied.IsNull() {
				denied := make([]string, 0, len(subj.Denied.Elements()))
				resp.Diagnostics.Append(subj.Denied.ElementsAs(ctx, &denied, false)...)
				createTemplateRequest.Subject[i].Denied = denied
			}
		}
	}

	if len(plan.Sans) > 0 {
		createTemplateRequest.Sans = make([]infisical.CertificateTemplateSAN, len(plan.Sans))
		for i, san := range plan.Sans {
			createTemplateRequest.Sans[i] = infisical.CertificateTemplateSAN{
				Type: san.Type.ValueString(),
			}

			if !san.Allowed.IsNull() {
				allowed := make([]string, 0, len(san.Allowed.Elements()))
				resp.Diagnostics.Append(san.Allowed.ElementsAs(ctx, &allowed, false)...)
				createTemplateRequest.Sans[i].Allowed = allowed
			}

			if !san.Required.IsNull() {
				required := make([]string, 0, len(san.Required.Elements()))
				resp.Diagnostics.Append(san.Required.ElementsAs(ctx, &required, false)...)
				createTemplateRequest.Sans[i].Required = required
			}

			if !san.Denied.IsNull() {
				denied := make([]string, 0, len(san.Denied.Elements()))
				resp.Diagnostics.Append(san.Denied.ElementsAs(ctx, &denied, false)...)
				createTemplateRequest.Sans[i].Denied = denied
			}
		}
	}

	if plan.KeyUsages != nil {
		createTemplateRequest.KeyUsages = &infisical.CertificateTemplateKeyUsages{}

		if !plan.KeyUsages.Allowed.IsNull() {
			allowed := make([]string, 0, len(plan.KeyUsages.Allowed.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Allowed.ElementsAs(ctx, &allowed, false)...)
			createTemplateRequest.KeyUsages.Allowed = allowed
		}

		if !plan.KeyUsages.Required.IsNull() {
			required := make([]string, 0, len(plan.KeyUsages.Required.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Required.ElementsAs(ctx, &required, false)...)
			createTemplateRequest.KeyUsages.Required = required
		}

		if !plan.KeyUsages.Denied.IsNull() {
			denied := make([]string, 0, len(plan.KeyUsages.Denied.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Denied.ElementsAs(ctx, &denied, false)...)
			createTemplateRequest.KeyUsages.Denied = denied
		}
	}

	if plan.ExtendedKeyUsages != nil {
		createTemplateRequest.ExtendedKeyUsages = &infisical.CertificateTemplateExtendedKeyUsages{}

		if !plan.ExtendedKeyUsages.Allowed.IsNull() {
			allowed := make([]string, 0, len(plan.ExtendedKeyUsages.Allowed.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Allowed.ElementsAs(ctx, &allowed, false)...)
			createTemplateRequest.ExtendedKeyUsages.Allowed = allowed
		}

		if !plan.ExtendedKeyUsages.Required.IsNull() {
			required := make([]string, 0, len(plan.ExtendedKeyUsages.Required.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Required.ElementsAs(ctx, &required, false)...)
			createTemplateRequest.ExtendedKeyUsages.Required = required
		}

		if !plan.ExtendedKeyUsages.Denied.IsNull() {
			denied := make([]string, 0, len(plan.ExtendedKeyUsages.Denied.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Denied.ElementsAs(ctx, &denied, false)...)
			createTemplateRequest.ExtendedKeyUsages.Denied = denied
		}
	}

	if plan.Algorithms != nil {
		createTemplateRequest.Algorithms = &infisical.CertificateTemplateAlgorithms{}

		if !plan.Algorithms.Signature.IsNull() {
			signature := make([]string, 0, len(plan.Algorithms.Signature.Elements()))
			resp.Diagnostics.Append(plan.Algorithms.Signature.ElementsAs(ctx, &signature, false)...)
			createTemplateRequest.Algorithms.Signature = signature
		}

		if !plan.Algorithms.KeyAlgorithm.IsNull() {
			keyAlgorithm := make([]string, 0, len(plan.Algorithms.KeyAlgorithm.Elements()))
			resp.Diagnostics.Append(plan.Algorithms.KeyAlgorithm.ElementsAs(ctx, &keyAlgorithm, false)...)
			createTemplateRequest.Algorithms.KeyAlgorithm = keyAlgorithm
		}
	}

	if plan.Validity != nil && !plan.Validity.Max.IsNull() {
		createTemplateRequest.Validity = &infisical.CertificateTemplateValidity{
			Max: plan.Validity.Max.ValueString(),
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.CreateCertificateTemplate(createTemplateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating certificate template", err.Error())
		return
	}

	plan.Id = types.StringValue(template.CertificateTemplate.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificateTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read certificate template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var currentState certManagerCertificateTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerCertificateTemplateResourceModel

	template, err := r.client.GetCertificateTemplate(infisical.GetCertificateTemplateRequest{
		TemplateId: currentState.Id.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading certificate template", err.Error())
		return
	}

	state.Id = types.StringValue(template.CertificateTemplate.Id)
	state.Name = types.StringValue(template.CertificateTemplate.Name)
	if template.CertificateTemplate.Description != "" {
		state.Description = types.StringValue(template.CertificateTemplate.Description)
	} else {
		state.Description = types.StringNull()
	}

	if len(template.CertificateTemplate.Subject) > 0 {
		state.Subject = make([]certManagerCertificateTemplateSubjectModel, len(template.CertificateTemplate.Subject))
		for i, subj := range template.CertificateTemplate.Subject {
			state.Subject[i] = certManagerCertificateTemplateSubjectModel{
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
				state.Subject[i].Required = types.ListNull(types.StringType)
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

	if len(template.CertificateTemplate.Sans) > 0 {
		state.Sans = make([]certManagerCertificateTemplateSanModel, len(template.CertificateTemplate.Sans))
		for i, san := range template.CertificateTemplate.Sans {
			state.Sans[i] = certManagerCertificateTemplateSanModel{
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
		state.KeyUsages = &certManagerCertificateTemplateKeyUsagesModel{}

		if template.CertificateTemplate.KeyUsages != nil {
			if len(template.CertificateTemplate.KeyUsages.Allowed) > 0 {
				allowedList, diags := types.ListValueFrom(ctx, types.StringType, template.CertificateTemplate.KeyUsages.Allowed)
				resp.Diagnostics.Append(diags...)
				state.KeyUsages.Allowed = allowedList
			} else {
				state.KeyUsages.Allowed = types.ListNull(types.StringType)
			}

			if len(template.CertificateTemplate.KeyUsages.Required) > 0 {
				requiredList, diags := types.ListValueFrom(ctx, types.StringType, template.CertificateTemplate.KeyUsages.Required)
				resp.Diagnostics.Append(diags...)
				state.KeyUsages.Required = requiredList
			} else {
				state.KeyUsages.Required = types.ListNull(types.StringType)
			}

			if len(template.CertificateTemplate.KeyUsages.Denied) > 0 {
				deniedList, diags := types.ListValueFrom(ctx, types.StringType, template.CertificateTemplate.KeyUsages.Denied)
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
		state.ExtendedKeyUsages = &certManagerCertificateTemplateExtendedKeyUsagesModel{}

		if template.CertificateTemplate.ExtendedKeyUsages != nil {
			if len(template.CertificateTemplate.ExtendedKeyUsages.Allowed) > 0 {
				allowedList, diags := types.ListValueFrom(ctx, types.StringType, template.CertificateTemplate.ExtendedKeyUsages.Allowed)
				resp.Diagnostics.Append(diags...)
				state.ExtendedKeyUsages.Allowed = allowedList
			} else {
				state.ExtendedKeyUsages.Allowed = types.ListNull(types.StringType)
			}

			if len(template.CertificateTemplate.ExtendedKeyUsages.Required) > 0 {
				requiredList, diags := types.ListValueFrom(ctx, types.StringType, template.CertificateTemplate.ExtendedKeyUsages.Required)
				resp.Diagnostics.Append(diags...)
				state.ExtendedKeyUsages.Required = requiredList
			} else {
				state.ExtendedKeyUsages.Required = types.ListNull(types.StringType)
			}

			if len(template.CertificateTemplate.ExtendedKeyUsages.Denied) > 0 {
				deniedList, diags := types.ListValueFrom(ctx, types.StringType, template.CertificateTemplate.ExtendedKeyUsages.Denied)
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

	if template.CertificateTemplate.Algorithms != nil {
		state.Algorithms = &certManagerCertificateTemplateAlgorithmsModel{}

		if len(template.CertificateTemplate.Algorithms.Signature) > 0 {
			signatureList, diags := types.ListValueFrom(ctx, types.StringType, template.CertificateTemplate.Algorithms.Signature)
			resp.Diagnostics.Append(diags...)
			state.Algorithms.Signature = signatureList
		} else {
			state.Algorithms.Signature = types.ListNull(types.StringType)
		}

		if len(template.CertificateTemplate.Algorithms.KeyAlgorithm) > 0 {
			keyAlgorithmList, diags := types.ListValueFrom(ctx, types.StringType, template.CertificateTemplate.Algorithms.KeyAlgorithm)
			resp.Diagnostics.Append(diags...)
			state.Algorithms.KeyAlgorithm = keyAlgorithmList
		} else {
			state.Algorithms.KeyAlgorithm = types.ListNull(types.StringType)
		}
	}

	if template.CertificateTemplate.Validity != nil && template.CertificateTemplate.Validity.Max != "" {
		state.Validity = &certManagerCertificateTemplateValidityModel{
			Max: types.StringValue(template.CertificateTemplate.Validity.Max),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerCertificateTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update certificate template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerCertificateTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTemplateRequest := infisical.UpdateCertificateTemplateRequest{
		TemplateId:  plan.Id.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if len(plan.Subject) > 0 {
		updateTemplateRequest.Subject = make([]infisical.CertificateTemplateSubject, len(plan.Subject))
		for i, subj := range plan.Subject {
			updateTemplateRequest.Subject[i] = infisical.CertificateTemplateSubject{
				Type: subj.Type.ValueString(),
			}

			if !subj.Allowed.IsNull() {
				allowed := make([]string, 0, len(subj.Allowed.Elements()))
				resp.Diagnostics.Append(subj.Allowed.ElementsAs(ctx, &allowed, false)...)
				updateTemplateRequest.Subject[i].Allowed = allowed
			}

			if !subj.Required.IsNull() {
				required := make([]string, 0, len(subj.Required.Elements()))
				resp.Diagnostics.Append(subj.Required.ElementsAs(ctx, &required, false)...)
				updateTemplateRequest.Subject[i].Required = required
			}

			if !subj.Denied.IsNull() {
				denied := make([]string, 0, len(subj.Denied.Elements()))
				resp.Diagnostics.Append(subj.Denied.ElementsAs(ctx, &denied, false)...)
				updateTemplateRequest.Subject[i].Denied = denied
			}
		}
	}

	if len(plan.Sans) > 0 {
		updateTemplateRequest.Sans = make([]infisical.CertificateTemplateSAN, len(plan.Sans))
		for i, san := range plan.Sans {
			updateTemplateRequest.Sans[i] = infisical.CertificateTemplateSAN{
				Type: san.Type.ValueString(),
			}

			if !san.Allowed.IsNull() {
				allowed := make([]string, 0, len(san.Allowed.Elements()))
				resp.Diagnostics.Append(san.Allowed.ElementsAs(ctx, &allowed, false)...)
				updateTemplateRequest.Sans[i].Allowed = allowed
			}

			if !san.Required.IsNull() {
				required := make([]string, 0, len(san.Required.Elements()))
				resp.Diagnostics.Append(san.Required.ElementsAs(ctx, &required, false)...)
				updateTemplateRequest.Sans[i].Required = required
			}

			if !san.Denied.IsNull() {
				denied := make([]string, 0, len(san.Denied.Elements()))
				resp.Diagnostics.Append(san.Denied.ElementsAs(ctx, &denied, false)...)
				updateTemplateRequest.Sans[i].Denied = denied
			}
		}
	}

	if plan.KeyUsages != nil {
		updateTemplateRequest.KeyUsages = &infisical.CertificateTemplateKeyUsages{}

		if !plan.KeyUsages.Allowed.IsNull() {
			allowed := make([]string, 0, len(plan.KeyUsages.Allowed.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Allowed.ElementsAs(ctx, &allowed, false)...)
			updateTemplateRequest.KeyUsages.Allowed = allowed
		}

		if !plan.KeyUsages.Required.IsNull() {
			required := make([]string, 0, len(plan.KeyUsages.Required.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Required.ElementsAs(ctx, &required, false)...)
			updateTemplateRequest.KeyUsages.Required = required
		}

		if !plan.KeyUsages.Denied.IsNull() {
			denied := make([]string, 0, len(plan.KeyUsages.Denied.Elements()))
			resp.Diagnostics.Append(plan.KeyUsages.Denied.ElementsAs(ctx, &denied, false)...)
			updateTemplateRequest.KeyUsages.Denied = denied
		}
	}

	if plan.ExtendedKeyUsages != nil {
		updateTemplateRequest.ExtendedKeyUsages = &infisical.CertificateTemplateExtendedKeyUsages{}

		if !plan.ExtendedKeyUsages.Allowed.IsNull() {
			allowed := make([]string, 0, len(plan.ExtendedKeyUsages.Allowed.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Allowed.ElementsAs(ctx, &allowed, false)...)
			updateTemplateRequest.ExtendedKeyUsages.Allowed = allowed
		}

		if !plan.ExtendedKeyUsages.Required.IsNull() {
			required := make([]string, 0, len(plan.ExtendedKeyUsages.Required.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Required.ElementsAs(ctx, &required, false)...)
			updateTemplateRequest.ExtendedKeyUsages.Required = required
		}

		if !plan.ExtendedKeyUsages.Denied.IsNull() {
			denied := make([]string, 0, len(plan.ExtendedKeyUsages.Denied.Elements()))
			resp.Diagnostics.Append(plan.ExtendedKeyUsages.Denied.ElementsAs(ctx, &denied, false)...)
			updateTemplateRequest.ExtendedKeyUsages.Denied = denied
		}
	}

	if plan.Algorithms != nil {
		updateTemplateRequest.Algorithms = &infisical.CertificateTemplateAlgorithms{}

		if !plan.Algorithms.Signature.IsNull() {
			signature := make([]string, 0, len(plan.Algorithms.Signature.Elements()))
			resp.Diagnostics.Append(plan.Algorithms.Signature.ElementsAs(ctx, &signature, false)...)
			updateTemplateRequest.Algorithms.Signature = signature
		}

		if !plan.Algorithms.KeyAlgorithm.IsNull() {
			keyAlgorithm := make([]string, 0, len(plan.Algorithms.KeyAlgorithm.Elements()))
			resp.Diagnostics.Append(plan.Algorithms.KeyAlgorithm.ElementsAs(ctx, &keyAlgorithm, false)...)
			updateTemplateRequest.Algorithms.KeyAlgorithm = keyAlgorithm
		}
	}

	if plan.Validity != nil && !plan.Validity.Max.IsNull() {
		updateTemplateRequest.Validity = &infisical.CertificateTemplateValidity{
			Max: plan.Validity.Max.ValueString(),
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateCertificateTemplate(updateTemplateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating certificate template", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificateTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete certificate template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerCertificateTemplateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteCertificateTemplate(infisical.DeleteCertificateTemplateRequest{
		TemplateId: state.Id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting certificate template", err.Error())
		return
	}
}

func (r *certManagerCertificateTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
