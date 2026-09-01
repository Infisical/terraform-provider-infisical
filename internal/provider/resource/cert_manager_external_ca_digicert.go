package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const digiCertCaPurposeSsl = "ssl"

var (
	_ resource.Resource = &certManagerExternalCADigiCertResource{}

	SUPPORTED_DIGICERT_CA_PURPOSES = []string{digiCertCaPurposeSsl, "code_signing"}
)

func NewCertManagerExternalCADigiCertResource() resource.Resource {
	return &certManagerExternalCADigiCertResource{}
}

type certManagerExternalCADigiCertResource struct {
	client *infisical.Client
}

type certManagerExternalCADigiCertVerifiedContactModel struct {
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Email     types.String `tfsdk:"email"`
	JobTitle  types.String `tfsdk:"job_title"`
	Telephone types.String `tfsdk:"telephone"`
}

type certManagerExternalCADigiCertResourceModel struct {
	Id              types.String                                       `tfsdk:"id"`
	Name            types.String                                       `tfsdk:"name"`
	Status          types.String                                       `tfsdk:"status"`
	AppConnectionId types.String                                       `tfsdk:"app_connection_id"`
	OrganizationId  types.Int64                                        `tfsdk:"organization_id"`
	ProductNameId   types.String                                       `tfsdk:"product_name_id"`
	Purpose         types.String                                       `tfsdk:"purpose"`
	VerifiedContact *certManagerExternalCADigiCertVerifiedContactModel `tfsdk:"verified_contact"`
}

func (r *certManagerExternalCADigiCertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_external_ca_digicert"
}

func (r *certManagerExternalCADigiCertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage external DigiCert (CertCentral) certificate authorities in Certificate Manager. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the DigiCert CA",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the CA. Supported values: " + strings.Join(SUPPORTED_CA_STATUSES, ", ") + ". Defaults to 'active'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_CA_STATUSES...),
				},
			},
			"app_connection_id": schema.StringAttribute{
				Description: "The ID of the DigiCert app connection used for certificate issuance",
				Required:    true,
			},
			"organization_id": schema.Int64Attribute{
				Description: "The CertCentral organization ID that will be listed on issued certificates",
				Required:    true,
			},
			"product_name_id": schema.StringAttribute{
				Description: "The DigiCert product name ID used for issuance (e.g. ssl_plus, code_signing, code_signing_ev)",
				Required:    true,
			},
			"purpose": schema.StringAttribute{
				Description: "Whether this CA issues SSL/TLS or code signing certificates. Supported values: " + strings.Join(SUPPORTED_DIGICERT_CA_PURPOSES, ", ") + ". Defaults to 'ssl'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_DIGICERT_CA_PURPOSES...),
				},
			},
			"verified_contact": schema.SingleNestedAttribute{
				Description: "Contact info for the user who approves first-time code signing orders for the organization. Required when purpose is 'code_signing'.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"first_name": schema.StringAttribute{
						Description: "The first name of the verified contact",
						Required:    true,
					},
					"last_name": schema.StringAttribute{
						Description: "The last name of the verified contact",
						Required:    true,
					},
					"email": schema.StringAttribute{
						Description: "The email address of the verified contact",
						Required:    true,
					},
					"job_title": schema.StringAttribute{
						Description: "The job title of the verified contact",
						Required:    true,
					},
					"telephone": schema.StringAttribute{
						Description: "The telephone number of the verified contact",
						Required:    true,
					},
				},
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the DigiCert CA",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *certManagerExternalCADigiCertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// buildConfiguration maps the plan onto the API's DigiCert CA configuration payload.
func (r *certManagerExternalCADigiCertResource) buildConfiguration(plan certManagerExternalCADigiCertResourceModel) infisical.CertificateAuthorityConfiguration {
	organizationId := int(plan.OrganizationId.ValueInt64())

	configuration := infisical.CertificateAuthorityConfiguration{
		AppConnectionId: plan.AppConnectionId.ValueString(),
		DigiCertOrgId:   &organizationId,
		ProductNameId:   plan.ProductNameId.ValueString(),
	}

	if !plan.Purpose.IsNull() && !plan.Purpose.IsUnknown() {
		configuration.Purpose = plan.Purpose.ValueString()
	}

	if plan.VerifiedContact != nil {
		configuration.VerifiedContact = &infisical.DigiCertCAVerifiedContact{
			FirstName: plan.VerifiedContact.FirstName.ValueString(),
			LastName:  plan.VerifiedContact.LastName.ValueString(),
			Email:     plan.VerifiedContact.Email.ValueString(),
			JobTitle:  plan.VerifiedContact.JobTitle.ValueString(),
			Telephone: plan.VerifiedContact.Telephone.ValueString(),
		}
	}

	return configuration
}

// keepIfOnlyWhitespaceDiffers returns the value already held in state when it
// differs from the server's value by surrounding whitespace alone. The API trims
// these fields, so mirroring the trimmed value back unconditionally would leave a
// permanent diff on a config that is already applied. A difference beyond
// whitespace is real drift and the server value wins.
func keepIfOnlyWhitespaceDiffers(current types.String, serverValue string) types.String {
	if !current.IsNull() && !current.IsUnknown() && strings.TrimSpace(current.ValueString()) == serverValue {
		return current
	}
	return types.StringValue(serverValue)
}

// applyServerOwnedFields copies back only the values the server owns after a write.
//
// Config-owned attributes are deliberately left untouched here. The API trims and
// normalizes strings (every verifiedContact field and the CA name go through a
// zod .trim()), and writing the normalized value back over what the practitioner
// configured fails Terraform's "Provider produced inconsistent result after apply"
// check for attributes that are not Computed. Read reconciles those instead, where
// a difference is reported as drift rather than an error.
func (r *certManagerExternalCADigiCertResource) applyServerOwnedFields(model *certManagerExternalCADigiCertResourceModel, ca infisical.CertificateAuthority) {
	model.Id = types.StringValue(ca.Id)
	model.Status = types.StringValue(ca.Status)

	// purpose is Optional+Computed, so it must never be left unknown after apply.
	purpose := ca.Configuration.Purpose
	if purpose == "" {
		purpose = digiCertCaPurposeSsl
	}
	model.Purpose = types.StringValue(purpose)
}

// applyCAToState mirrors the full API response onto the model. Only Read uses this:
// there, a value differing from config is legitimate drift and Terraform surfaces it
// as a diff.
func (r *certManagerExternalCADigiCertResource) applyCAToState(model *certManagerExternalCADigiCertResourceModel, ca infisical.CertificateAuthority) {
	r.applyServerOwnedFields(model, ca)

	model.Name = keepIfOnlyWhitespaceDiffers(model.Name, ca.Name)

	if ca.Configuration.AppConnectionId != "" {
		model.AppConnectionId = types.StringValue(ca.Configuration.AppConnectionId)
	}

	if ca.Configuration.DigiCertOrgId != nil {
		model.OrganizationId = types.Int64Value(int64(*ca.Configuration.DigiCertOrgId))
	}

	if ca.Configuration.ProductNameId != "" {
		model.ProductNameId = types.StringValue(ca.Configuration.ProductNameId)
	}

	if contact := ca.Configuration.VerifiedContact; contact != nil {
		current := model.VerifiedContact
		if current == nil {
			current = &certManagerExternalCADigiCertVerifiedContactModel{}
		}

		model.VerifiedContact = &certManagerExternalCADigiCertVerifiedContactModel{
			FirstName: keepIfOnlyWhitespaceDiffers(current.FirstName, contact.FirstName),
			LastName:  keepIfOnlyWhitespaceDiffers(current.LastName, contact.LastName),
			Email:     keepIfOnlyWhitespaceDiffers(current.Email, contact.Email),
			JobTitle:  keepIfOnlyWhitespaceDiffers(current.JobTitle, contact.JobTitle),
			Telephone: keepIfOnlyWhitespaceDiffers(current.Telephone, contact.Telephone),
		}
	} else {
		model.VerifiedContact = nil
	}
}

func (r *certManagerExternalCADigiCertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create DigiCert CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerExternalCADigiCertResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	status := "active"
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		status = plan.Status.ValueString()
	}

	newCA, err := r.client.CreateDigiCertCA(infisical.CreateDigiCertCARequest{
		Name:          plan.Name.ValueString(),
		Status:        status,
		Configuration: r.buildConfiguration(plan),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating DigiCert CA",
			"Couldn't create DigiCert CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	r.applyServerOwnedFields(&plan, newCA.CertificateAuthority)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCADigiCertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read DigiCert CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerExternalCADigiCertResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ca, err := r.client.GetDigiCertCA(infisical.GetCARequest{
		CAId: state.Id.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading DigiCert CA",
			"Couldn't read DigiCert CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	r.applyCAToState(&state, ca.CertificateAuthority)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCADigiCertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update DigiCert CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerExternalCADigiCertResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerExternalCADigiCertResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedCA, err := r.client.UpdateDigiCertCA(infisical.UpdateDigiCertCARequest{
		CAId:          state.Id.ValueString(),
		Name:          plan.Name.ValueString(),
		Status:        plan.Status.ValueString(),
		Configuration: r.buildConfiguration(plan),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating DigiCert CA",
			"Couldn't update DigiCert CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	r.applyServerOwnedFields(&plan, updatedCA.CertificateAuthority)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCADigiCertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete DigiCert CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerExternalCADigiCertResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteDigiCertCA(infisical.DeleteCARequest{
		CAId: state.Id.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting DigiCert CA",
			"Couldn't delete DigiCert CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerExternalCADigiCertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if strings.Contains(id, ":") {
		parts := strings.Split(id, ":")
		id = parts[len(parts)-1]
	}
	if id == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected <caId> or <projectSlug>:<caId>, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
