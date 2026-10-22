package resource

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	infisicalerrors "terraform-provider-infisical/internal/errors"
)

var (
	SUPPORTED_CERT_ISSUE_KEY_ALGORITHMS = []string{
		"RSA_2048", "RSA_3072", "RSA_4096",
		"EC_prime256v1", "EC_secp384r1", "EC_secp521r1",
	}
	SUPPORTED_CERT_ISSUE_SIGNATURE_ALGORITHMS = []string{
		"RSA-SHA256", "RSA-SHA384", "RSA-SHA512",
		"ECDSA-SHA256", "ECDSA-SHA384", "ECDSA-SHA512",
	}
	SUPPORTED_CERT_ISSUE_KEY_USAGES = []string{
		"digital_signature", "key_encipherment", "non_repudiation",
		"data_encipherment", "key_agreement", "key_cert_sign",
		"crl_sign", "encipher_only", "decipher_only",
	}
	SUPPORTED_CERT_ISSUE_EXTENDED_KEY_USAGES = []string{
		"client_auth", "server_auth", "code_signing",
		"email_protection", "ocsp_signing", "time_stamping",
	}
	legacyKeyUsageMap = map[string]string{
		"digitalSignature": "digital_signature",
		"keyEncipherment":  "key_encipherment",
		"nonRepudiation":   "non_repudiation",
		"dataEncipherment": "data_encipherment",
		"keyAgreement":     "key_agreement",
		"keyCertSign":      "key_cert_sign",
		"cRLSign":          "crl_sign",
		"encipherOnly":     "encipher_only",
		"decipherOnly":     "decipher_only",
	}
	legacyExtendedKeyUsageMap = map[string]string{
		"clientAuth":      "client_auth",
		"serverAuth":      "server_auth",
		"codeSigning":     "code_signing",
		"emailProtection": "email_protection",
		"ocspSigning":     "ocsp_signing",
		"timeStamping":    "time_stamping",
	}
)

func normalizeKeyUsage(usage string) string {
	if v, ok := legacyKeyUsageMap[usage]; ok {
		return v
	}
	return usage
}

func normalizeExtendedKeyUsage(usage string) string {
	if v, ok := legacyExtendedKeyUsageMap[usage]; ok {
		return v
	}
	return usage
}

var (
	_ resource.Resource                   = &certManagerCertificateResource{}
	_ resource.ResourceWithModifyPlan     = &certManagerCertificateResource{}
	_ resource.ResourceWithValidateConfig = &certManagerCertificateResource{}
)

func NewCertManagerCertificateResource() resource.Resource {
	return &certManagerCertificateResource{}
}

type certManagerCertificateResource struct {
	client *infisical.Client
}

type certManagerCertificateResourceModel struct {
	ProfileId                   types.String `tfsdk:"profile_id"`
	ApplicationId               types.String `tfsdk:"application_id"`
	CSR                         types.String `tfsdk:"csr"`
	CommonName                  types.String `tfsdk:"common_name"`
	AltNames                    types.List   `tfsdk:"alt_names"`
	Organization                types.String `tfsdk:"organization"`
	OU                          types.String `tfsdk:"ou"`
	Country                     types.String `tfsdk:"country"`
	Province                    types.String `tfsdk:"province"`
	Locality                    types.String `tfsdk:"locality"`
	DomainComponents            types.List   `tfsdk:"domain_components"`
	KeyAlgorithm                types.String `tfsdk:"key_algorithm"`
	SignatureAlgorithm          types.String `tfsdk:"signature_algorithm"`
	KeyUsages                   types.List   `tfsdk:"key_usages"`
	ExtendedKeyUsages           types.List   `tfsdk:"extended_key_usages"`
	TTL                         types.String `tfsdk:"ttl"`
	TimeoutSeconds              types.Int64  `tfsdk:"timeout_seconds"`
	ManualRenewDaysBeforeExpiry types.Int64  `tfsdk:"manual_renew_days_before_expiry"`
	RevokeOnDestroy             types.Bool   `tfsdk:"revoke_on_destroy"`
	RevocationReason            types.String `tfsdk:"revocation_reason_on_destroy"`
	DeleteOnDestroy             types.Bool   `tfsdk:"delete_on_destroy"`
	Id                          types.String `tfsdk:"id"`
	OrderId                     types.String `tfsdk:"order_id"`
	CertificateRequestId        types.String `tfsdk:"certificate_request_id"`
	Status                      types.String `tfsdk:"status"`
	SerialNumber                types.String `tfsdk:"serial_number"`
	NotBefore                   types.String `tfsdk:"not_before"`
	NotAfter                    types.String `tfsdk:"not_after"`
	Certificate                 types.String `tfsdk:"certificate"`
	PrivateKey                  types.String `tfsdk:"private_key"`
	CertificateChain            types.String `tfsdk:"certificate_chain"`
}

func (r *certManagerCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_certificate"
}

func (r *certManagerCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Request and manage certificates from Infisical certificate profiles, either from a CSR or from subject fields. The resource polls until the certificate is issued or `timeout_seconds` is reached. Only Machine Identity authentication is supported.\n\n" +
			"`manual_renew_days_before_expiry` renews the certificate in place on `terraform apply` once it is close to expiry, and changing `csr` renews it in place with the new CSR. Renewal only runs when you apply, so apply regularly or use the profile's auto-renewal instead. The previous certificate stays valid until it expires.\n\n" +
			"`revoke_on_destroy` and `delete_on_destroy` act when the resource is destroyed or replaced, using the values from the last apply. With `revoke_on_destroy`, add `lifecycle { create_before_destroy = true }` so the replacement exists before the old certificate is revoked.",
		Attributes: map[string]schema.Attribute{
			"profile_id": schema.StringAttribute{
				Description: "The ID of the certificate profile to use for issuance. Find it on the profile's page in Certificate Manager, or reference an `infisical_cert_manager_certificate_profile` resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "The ID of the Certificate Manager application to issue this certificate from",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"csr": schema.StringAttribute{
				Description: "Certificate Signing Request (CSR) in PEM format. If provided, the certificate will be issued based on the CSR. Use Terraform's file() function to read from a file (e.g., file(\"./my-certificate.csr\")). Changing the CSR renews the certificate in place with the new CSR. Adding or removing the CSR replaces the certificate.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIf(
						certificateIssuanceModeChanged,
						"Switching between CSR and field-based issuance replaces the certificate.",
						"Switching between CSR and field-based issuance replaces the certificate.",
					),
				},
			},
			"common_name": schema.StringAttribute{
				Description: "The common name (CN) for the certificate. Required when not using CSR",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alt_names": schema.ListAttribute{
				Description: "Subject alternative names (SANs) for the certificate",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "The organization (O) for the certificate",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ou": schema.StringAttribute{
				Description: "The organizational unit (OU) for the certificate",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"country": schema.StringAttribute{
				Description: "The country (C) for the certificate (2-letter code)",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 2),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"province": schema.StringAttribute{
				Description: "The state/province (ST) for the certificate",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locality": schema.StringAttribute{
				Description: "The locality (L) for the certificate",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain_components": schema.ListAttribute{
				Description: "Domain components (DC) for the certificate. Multi-valued; each entry becomes a DC attribute in the subject.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
			},
			"key_algorithm": schema.StringAttribute{
				Description: "The key algorithm for the certificate. Supported: " + strings.Join(SUPPORTED_CERT_ISSUE_KEY_ALGORITHMS, ", "),
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_KEY_ALGORITHMS...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"signature_algorithm": schema.StringAttribute{
				Description: "The signature algorithm for the certificate. Supported: " + strings.Join(SUPPORTED_CERT_ISSUE_SIGNATURE_ALGORITHMS, ", "),
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_SIGNATURE_ALGORITHMS...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_usages": schema.ListAttribute{
				Description: "Key usages for the certificate. Supported: " + strings.Join(SUPPORTED_CERT_ISSUE_KEY_USAGES, ", "),
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_KEY_USAGES...)),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
			},
			"extended_key_usages": schema.ListAttribute{
				Description: "Extended key usages for the certificate. Supported: " + strings.Join(SUPPORTED_CERT_ISSUE_EXTENDED_KEY_USAGES, ", "),
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_EXTENDED_KEY_USAGES...)),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.StringAttribute{
				Description: "Time to live for the certificate (e.g., '30d', '90d', '1y').",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "Maximum time to wait for certificate issuance in seconds. Defaults to 3600 (1 hour)",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3600),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"manual_renew_days_before_expiry": schema.Int64Attribute{
				Description: "Renew the certificate in place on `terraform apply` once it has fewer than this many days left before expiry. Must be at least 1 and smaller than the certificate lifetime. Renewal only happens when an apply runs. Not set by default.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"revoke_on_destroy": schema.BoolAttribute{
				Description: "Revoke the certificate, and every other renewal of it that is still valid, when the resource is destroyed or replaced. Uses the value from the last apply. Certificates from Azure AD CS or Venafi TPP cannot be revoked, so destroying them fails while this is set. Self-signed certificates are skipped with a warning. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"revocation_reason_on_destroy": schema.StringAttribute{
				Description: "The reason sent when the certificate is revoked on destroy, used with `revoke_on_destroy`. Supported: " + strings.Join(SUPPORTED_CERT_REVOCATION_REASONS, ", ") + ". Defaults to unspecified.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("unspecified"),
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_CERT_REVOCATION_REASONS...),
				},
			},
			"delete_on_destroy": schema.BoolAttribute{
				Description: "Delete the certificate, and every other renewal of it, when the resource is destroyed or replaced. Deleting does not revoke it. Set `revoke_on_destroy` as well to revoke it first. Uses the value from the last apply. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"id": schema.StringAttribute{
				Description: "The ID of the certificate. Changes when the certificate is renewed.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"order_id": schema.StringAttribute{
				Description: "The ID of the certificate order. A certificate and all of its renewals share one order, so use it to select this certificate in a certificate sync's `certificate_filters`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate_request_id": schema.StringAttribute{
				Description: "The ID of the certificate request",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the certificate: active or revoked. It is renewed when the certificate was renewed outside Terraform and that renewal is no longer available, and deleted when it was deleted outside Terraform but still has to be revoked because revoke_on_destroy is set. Use not_after to tell whether it has expired.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"serial_number": schema.StringAttribute{
				Description: "The serial number of the issued certificate",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"not_before": schema.StringAttribute{
				Description: "The not-before date of the certificate (RFC3339 format)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"not_after": schema.StringAttribute{
				Description: "The not-after (expiration) date of the certificate (RFC3339 format)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate": schema.StringAttribute{
				Description: "The issued certificate in PEM format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key": schema.StringAttribute{
				Description: "The private key in PEM format (only available for direct field requests, not CSR-based).",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate_chain": schema.StringAttribute{
				Description: "The certificate chain in PEM format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *certManagerCertificateResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config certManagerCertificateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.RevocationReason.IsNull() || config.RevocationReason.IsUnknown() {
		return
	}
	if config.RevokeOnDestroy.ValueBool() || config.RevokeOnDestroy.IsUnknown() {
		return
	}
	resp.Diagnostics.AddAttributeWarning(
		path.Root("revocation_reason_on_destroy"),
		"Revocation reason has no effect",
		"revocation_reason_on_destroy is only used when revoke_on_destroy is true.",
	)
}

func (r *certManagerCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to request certificate",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerCertificateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasCSR := !plan.CSR.IsNull() && !plan.CSR.IsUnknown() && plan.CSR.ValueString() != ""
	hasCommonName := !plan.CommonName.IsNull() && !plan.CommonName.IsUnknown() && plan.CommonName.ValueString() != ""

	if !hasCSR && !hasCommonName {
		resp.Diagnostics.AddError(
			"Missing certificate request data",
			"Either 'csr' or 'common_name' must be provided",
		)
		return
	}

	timeoutSeconds := int64(3600)
	if !plan.TimeoutSeconds.IsNull() && !plan.TimeoutSeconds.IsUnknown() {
		timeoutSeconds = plan.TimeoutSeconds.ValueInt64()
	}

	certRequest := infisical.RequestCertificateRequest{
		ProfileId:     plan.ProfileId.ValueString(),
		ApplicationId: plan.ApplicationId.ValueString(),
	}

	if hasCSR {
		csrContent := strings.ReplaceAll(plan.CSR.ValueString(), "\r\n", "\n")
		csrContent = strings.TrimSpace(csrContent)
		certRequest.CSR = csrContent
	}

	hasAttributes := false
	attributes := infisical.CertificateAttributes{}

	if !plan.CommonName.IsNull() && !plan.CommonName.IsUnknown() {
		attributes.CommonName = plan.CommonName.ValueString()
		hasAttributes = true
	}

	if !plan.Organization.IsNull() && !plan.Organization.IsUnknown() {
		attributes.Organization = plan.Organization.ValueString()
		hasAttributes = true
	}

	if !plan.OU.IsNull() && !plan.OU.IsUnknown() {
		attributes.OU = plan.OU.ValueString()
		hasAttributes = true
	}

	if !plan.Country.IsNull() && !plan.Country.IsUnknown() {
		attributes.Country = plan.Country.ValueString()
		hasAttributes = true
	}

	if !plan.Province.IsNull() && !plan.Province.IsUnknown() {
		attributes.Province = plan.Province.ValueString()
		hasAttributes = true
	}

	if !plan.Locality.IsNull() && !plan.Locality.IsUnknown() {
		attributes.Locality = plan.Locality.ValueString()
		hasAttributes = true
	}

	if !plan.DomainComponents.IsNull() && !plan.DomainComponents.IsUnknown() {
		domainComponents := make([]string, 0, len(plan.DomainComponents.Elements()))
		resp.Diagnostics.Append(plan.DomainComponents.ElementsAs(ctx, &domainComponents, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(domainComponents) > 0 {
			attributes.DomainComponents = domainComponents
			hasAttributes = true
		}
	}

	if !plan.AltNames.IsNull() && !plan.AltNames.IsUnknown() {
		altNamesStr := make([]string, 0)
		resp.Diagnostics.Append(plan.AltNames.ElementsAs(ctx, &altNamesStr, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		altNames := make([]infisical.CertificateAltName, 0, len(altNamesStr))
		for _, altName := range altNamesStr {
			altNames = append(altNames, infisical.CertificateAltName{
				Type:  "dns_name",
				Value: altName,
			})
		}
		attributes.AltNames = altNames
		hasAttributes = true
	}

	if !plan.KeyAlgorithm.IsNull() && !plan.KeyAlgorithm.IsUnknown() {
		attributes.KeyAlgorithm = plan.KeyAlgorithm.ValueString()
		hasAttributes = true
	}

	if !plan.SignatureAlgorithm.IsNull() && !plan.SignatureAlgorithm.IsUnknown() {
		attributes.SignatureAlgorithm = plan.SignatureAlgorithm.ValueString()
		hasAttributes = true
	}

	if !plan.KeyUsages.IsNull() && !plan.KeyUsages.IsUnknown() {
		keyUsages := make([]string, 0)
		resp.Diagnostics.Append(plan.KeyUsages.ElementsAs(ctx, &keyUsages, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		attributes.KeyUsages = keyUsages
		hasAttributes = true
	}

	if !plan.ExtendedKeyUsages.IsNull() && !plan.ExtendedKeyUsages.IsUnknown() {
		extKeyUsages := make([]string, 0)
		resp.Diagnostics.Append(plan.ExtendedKeyUsages.ElementsAs(ctx, &extKeyUsages, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		attributes.ExtendedKeyUsages = extKeyUsages
		hasAttributes = true
	}

	if !plan.TTL.IsNull() && !plan.TTL.IsUnknown() {
		attributes.TTL = plan.TTL.ValueString()
		hasAttributes = true
	}

	if hasAttributes {
		certRequest.Attributes = &attributes
	}

	requestResponse, err := r.client.RequestCertificate(certRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error requesting certificate", err.Error())
		return
	}

	certificateRequestId := requestResponse.CertificateRequestId
	plan.CertificateRequestId = types.StringValue(certificateRequestId)

	if requestResponse.Certificate.CertificateId != "" && requestResponse.Certificate.Certificate != "" {
		r.populatePlanFromImmediateResponse(ctx, &plan, requestResponse.Certificate)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		return
	}

	r.pollCertificateRequest(ctx, &plan, certificateRequestId, timeoutSeconds, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificateResource) populatePlanFromImmediateResponse(ctx context.Context, plan *certManagerCertificateResourceModel, certResponse infisical.CertificateResponse) {
	plan.Id = types.StringValue(certResponse.CertificateId)
	plan.Status = types.StringValue(certificateStatusActive)

	if certResponse.Certificate != "" {
		plan.Certificate = types.StringValue(certResponse.Certificate)
	} else {
		plan.Certificate = types.StringValue("")
	}

	if certResponse.PrivateKey != "" {
		plan.PrivateKey = types.StringValue(certResponse.PrivateKey)
	} else {
		plan.PrivateKey = types.StringValue("")
	}

	if certResponse.CertificateChain != "" {
		plan.CertificateChain = types.StringValue(certResponse.CertificateChain)
	} else {
		plan.CertificateChain = types.StringValue("")
	}

	if certResponse.SerialNumber != "" {
		plan.SerialNumber = types.StringValue(certResponse.SerialNumber)
	} else {
		plan.SerialNumber = types.StringValue("")
	}

	r.populateCertificateDetails(ctx, plan, certResponse.CertificateId)
}

func (r *certManagerCertificateResource) populateCertificateDetails(ctx context.Context, plan *certManagerCertificateResourceModel, certificateId string) {
	certDetails, err := r.client.GetCertificate(infisical.GetCertificateRequest{
		CertificateId: certificateId,
	})
	if err != nil {
		tflog.Warn(ctx, "Failed to fetch certificate details", map[string]interface{}{
			"certificate_id": certificateId,
			"error":          err.Error(),
		})
		plan.NotBefore = types.StringValue("")
		plan.NotAfter = types.StringValue("")
		if plan.OrderId.IsUnknown() {
			plan.OrderId = types.StringNull()
		}
		for _, value := range []*types.String{&plan.CommonName, &plan.Organization, &plan.OU, &plan.Country, &plan.Province, &plan.Locality, &plan.KeyAlgorithm, &plan.SignatureAlgorithm, &plan.CSR, &plan.TTL} {
			if value.IsUnknown() {
				*value = types.StringNull()
			}
		}
		for _, value := range []*types.List{&plan.AltNames, &plan.DomainComponents, &plan.KeyUsages, &plan.ExtendedKeyUsages} {
			if value.IsUnknown() {
				*value = types.ListNull(types.StringType)
			}
		}
		return
	}

	cert := certDetails.Certificate
	plan.Status = types.StringValue(cert.Status)
	plan.OrderId = types.StringValue(cert.OrderId)
	plan.CommonName = types.StringValue(cert.CommonName)
	plan.KeyAlgorithm = types.StringValue(cert.KeyAlgorithm)
	plan.SignatureAlgorithm = types.StringValue(cert.SignatureAlgorithm)
	plan.AltNames = r.parseAltNames(ctx, cert.AltNames)
	plan.NotBefore = types.StringValue(cert.NotBefore)
	plan.NotAfter = types.StringValue(cert.NotAfter)

	if cert.CertificateChain != "" {
		plan.CertificateChain = types.StringValue(cert.CertificateChain)
	}

	if cert.ProfileId != "" {
		plan.ProfileId = types.StringValue(cert.ProfileId)
	}

	if cert.ApplicationId != "" {
		plan.ApplicationId = types.StringValue(cert.ApplicationId)
	}

	if cert.SubjectOrganization != "" {
		plan.Organization = types.StringValue(cert.SubjectOrganization)
	} else if plan.Organization.IsUnknown() {
		plan.Organization = types.StringNull()
	}

	if cert.SubjectOrganizationalUnit != "" {
		plan.OU = types.StringValue(cert.SubjectOrganizationalUnit)
	} else if plan.OU.IsUnknown() {
		plan.OU = types.StringNull()
	}

	if cert.SubjectCountry != "" {
		plan.Country = types.StringValue(cert.SubjectCountry)
	} else if plan.Country.IsUnknown() {
		plan.Country = types.StringNull()
	}

	if cert.SubjectState != "" {
		plan.Province = types.StringValue(cert.SubjectState)
	} else if plan.Province.IsUnknown() {
		plan.Province = types.StringNull()
	}

	if cert.SubjectLocality != "" {
		plan.Locality = types.StringValue(cert.SubjectLocality)
	} else if plan.Locality.IsUnknown() {
		plan.Locality = types.StringNull()
	}

	if cert.SubjectDomainComponents != "" {
		dcs := strings.Split(cert.SubjectDomainComponents, ",")
		for i := range dcs {
			dcs[i] = strings.TrimSpace(dcs[i])
		}
		if list, diags := types.ListValueFrom(ctx, types.StringType, dcs); !diags.HasError() {
			plan.DomainComponents = list
		}
	} else if plan.DomainComponents.IsUnknown() {
		plan.DomainComponents = types.ListNull(types.StringType)
	}

	if len(cert.KeyUsages) > 0 {
		normalized := make([]string, len(cert.KeyUsages))
		for i, u := range cert.KeyUsages {
			normalized[i] = normalizeKeyUsage(u)
		}
		if list, diags := types.ListValueFrom(ctx, types.StringType, normalized); !diags.HasError() {
			plan.KeyUsages = list
		}
	} else if plan.KeyUsages.IsUnknown() {
		plan.KeyUsages = types.ListNull(types.StringType)
	}

	if len(cert.ExtendedKeyUsages) > 0 {
		normalized := make([]string, len(cert.ExtendedKeyUsages))
		for i, u := range cert.ExtendedKeyUsages {
			normalized[i] = normalizeExtendedKeyUsage(u)
		}
		if list, diags := types.ListValueFrom(ctx, types.StringType, normalized); !diags.HasError() {
			plan.ExtendedKeyUsages = list
		}
	} else if plan.ExtendedKeyUsages.IsUnknown() {
		plan.ExtendedKeyUsages = types.ListNull(types.StringType)
	}

	if plan.CSR.IsUnknown() {
		plan.CSR = types.StringNull()
	}
	if plan.TTL.IsUnknown() {
		if nb, err := time.Parse(time.RFC3339, cert.NotBefore); err == nil {
			if na, err := time.Parse(time.RFC3339, cert.NotAfter); err == nil {
				days := int(math.Round(na.Sub(nb).Hours() / 24))
				if days > 0 {
					plan.TTL = types.StringValue(fmt.Sprintf("%dd", days))
				} else {
					plan.TTL = types.StringNull()
				}
			} else {
				plan.TTL = types.StringNull()
			}
		} else {
			plan.TTL = types.StringNull()
		}
	}
}

func (r *certManagerCertificateResource) parseAltNames(ctx context.Context, altNames interface{}) types.List {
	if altNames == nil {
		return types.ListNull(types.StringType)
	}

	var altNamesStrings []string

	switch v := altNames.(type) {
	case []interface{}:
		for _, item := range v {
			switch itemVal := item.(type) {
			case string:
				altNamesStrings = append(altNamesStrings, itemVal)
			case map[string]interface{}:
				if value, ok := itemVal["value"].(string); ok {
					altNamesStrings = append(altNamesStrings, value)
				}
			}
		}
	case []string:
		altNamesStrings = v
	case string:
		if strings.Contains(v, ",") {
			for _, part := range strings.Split(v, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					altNamesStrings = append(altNamesStrings, trimmed)
				}
			}
		} else if v != "" {
			altNamesStrings = []string{v}
		}
	}

	if len(altNamesStrings) == 0 {
		return types.ListNull(types.StringType)
	}

	altNamesList, diags := types.ListValueFrom(ctx, types.StringType, altNamesStrings)
	if diags.HasError() {
		tflog.Warn(ctx, "Failed to parse alt names list", map[string]interface{}{
			"error": diags.Errors(),
		})
		return types.ListNull(types.StringType)
	}
	return altNamesList
}

func (r *certManagerCertificateResource) pollCertificateRequest(ctx context.Context, plan *certManagerCertificateResourceModel, certificateRequestId string, timeoutSeconds int64, diagnostics *diag.Diagnostics) {
	timeout := time.Duration(timeoutSeconds) * time.Second
	startTime := time.Now()

	minInterval := 2 * time.Second
	maxInterval := 30 * time.Second
	currentInterval := minInterval

	for {
		if ctx.Err() != nil {
			diagnostics.AddError("Operation cancelled", ctx.Err().Error())
			return
		}

		if time.Since(startTime) > timeout {
			diagnostics.AddError(
				"Certificate issuance timeout",
				fmt.Sprintf("Certificate issuance did not complete within %d seconds. Request ID: %s", timeoutSeconds, certificateRequestId),
			)
			return
		}

		statusResponse, err := r.client.GetCertificateRequestStatus(infisical.GetCertificateRequestStatusRequest{
			RequestId: certificateRequestId,
		})
		if err != nil {
			if err == infisical.ErrNotFound {
				select {
				case <-ctx.Done():
					diagnostics.AddError("Operation cancelled", ctx.Err().Error())
					return
				case <-time.After(currentInterval):
					currentInterval = r.nextBackoffInterval(currentInterval, maxInterval)
					continue
				}
			}
			diagnostics.AddError(
				"Error checking certificate request status",
				fmt.Sprintf("Failed to check certificate request status: %v. Request ID: %s", err, certificateRequestId),
			)
			return
		}

		status := strings.ToLower(statusResponse.Status)

		switch status {
		case "issued":
			r.handleIssuedCertificate(ctx, plan, &statusResponse)
			return

		case "failed", "rejected":
			errorMsg := "Certificate issuance failed"
			if statusResponse.ErrorMessage != nil && *statusResponse.ErrorMessage != "" {
				errorMsg = fmt.Sprintf("Certificate issuance failed: %s", *statusResponse.ErrorMessage)
			}
			diagnostics.AddError(
				"Certificate issuance failed",
				fmt.Sprintf("%s. Request ID: %s", errorMsg, certificateRequestId),
			)
			return

		case "pending":
			select {
			case <-ctx.Done():
				diagnostics.AddError("Operation cancelled", ctx.Err().Error())
				return
			case <-time.After(currentInterval):
				currentInterval = r.nextBackoffInterval(currentInterval, maxInterval)
				continue
			}

		default:
			if statusResponse.CertificateId != nil && *statusResponse.CertificateId != "" &&
				statusResponse.Certificate != nil && *statusResponse.Certificate != "" {
				r.handleIssuedCertificate(ctx, plan, &statusResponse)
				return
			}
			select {
			case <-ctx.Done():
				diagnostics.AddError("Operation cancelled", ctx.Err().Error())
				return
			case <-time.After(currentInterval):
				currentInterval = r.nextBackoffInterval(currentInterval, maxInterval)
				continue
			}
		}
	}
}

func (r *certManagerCertificateResource) nextBackoffInterval(current, maxInterval time.Duration) time.Duration {
	next := time.Duration(float64(current) * 1.5)
	if next > maxInterval {
		return maxInterval
	}
	return next
}

func (r *certManagerCertificateResource) handleIssuedCertificate(ctx context.Context, plan *certManagerCertificateResourceModel, statusResponse *infisical.GetCertificateRequestStatusResponse) {
	plan.Status = types.StringValue(certificateStatusActive)

	certificateId := ""
	if statusResponse.CertificateId != nil && *statusResponse.CertificateId != "" {
		certificateId = *statusResponse.CertificateId
		plan.Id = types.StringValue(certificateId)
	}

	if statusResponse.Certificate != nil && *statusResponse.Certificate != "" {
		plan.Certificate = types.StringValue(*statusResponse.Certificate)
	} else {
		plan.Certificate = types.StringValue("")
	}

	if statusResponse.PrivateKey != nil && *statusResponse.PrivateKey != "" {
		plan.PrivateKey = types.StringValue(*statusResponse.PrivateKey)
	} else {
		plan.PrivateKey = types.StringValue("")
	}

	if statusResponse.CertificateChain != nil && *statusResponse.CertificateChain != "" {
		plan.CertificateChain = types.StringValue(*statusResponse.CertificateChain)
	} else {
		plan.CertificateChain = types.StringValue("")
	}

	if statusResponse.SerialNumber != nil && *statusResponse.SerialNumber != "" {
		plan.SerialNumber = types.StringValue(*statusResponse.SerialNumber)
	} else {
		plan.SerialNumber = types.StringValue("")
	}

	if certificateId != "" {
		r.populateCertificateDetails(ctx, plan, certificateId)
	}
}

func (r *certManagerCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.read(ctx, req, req.Private, resp)
}

func (r *certManagerCertificateResource) read(ctx context.Context, req resource.ReadRequest, private privateStateReader, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read certificate",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerCertificateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certResponse, err := r.client.GetCertificate(infisical.GetCertificateRequest{
		CertificateId:  state.Id.ValueString(),
		IncludeDeleted: true,
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading certificate", err.Error())
		return
	}

	cert := certResponse.Certificate

	pending, hasPending := readPendingRenewal(ctx, private)
	pendingResolved := hasPending && r.resolvePendingRenewal(ctx, cert, pending, resp)

	followed := false
	latestRenewalId := ""
	if cert.LatestRenewalCertificateId != nil && *cert.LatestRenewalCertificateId != cert.Id {
		latestRenewalId = *cert.LatestRenewalCertificateId
	}

	if latestRenewalId != "" && (cert.RenewedByCertificateId != "" || cert.DeletedAt != "") {
		latestCert, ok := r.followLatestRenewal(ctx, &state, latestRenewalId, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if ok {
			cert = latestCert
			followed = true
		}
	}

	if followed && pendingResolved && pending.CSR != "" && csrMatchesCertificateKey(pending.CSR, state.Certificate.ValueString()) {
		state.CSR = types.StringValue(pending.CSR)
	}

	if cert.DeletedAt != "" {
		if !certificateDeletedButRevocable(state, cert) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddWarning(
			"Certificate deleted outside Terraform",
			fmt.Sprintf("Certificate %s was deleted outside Terraform but is not revoked. Because revoke_on_destroy is set, Terraform replaces it and revokes the deleted certificate.", cert.Id),
		)
		state.Status = types.StringValue(certificateStatusDeleted)
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		return
	}

	missingKey := privateFlagSet(ctx, private, certificateMissingKeyKey)
	if missingKey || state.Certificate.IsNull() || state.Certificate.IsUnknown() || state.Certificate.ValueString() == "" {
		if !r.populateCertificateMaterial(&state, cert.Id, cert.HasPrivateKey, &resp.Diagnostics) {
			return
		}
		if missingKey && resp.Private != nil {
			resp.Diagnostics.Append(resp.Private.SetKey(ctx, certificateMissingKeyKey, nil)...)
		}
	}

	if !hasCSRValue(state.CSR) && cert.HasPrivateKey != nil && !*cert.HasPrivateKey && resp.Private != nil {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, certificateIssuedFromCSRKey, []byte("true"))...)
	}

	state.Status = types.StringValue(cert.Status)
	if cert.RenewedByCertificateId != "" {
		state.Status = types.StringValue(certificateStatusRenewed)
	}
	state.CommonName = types.StringValue(cert.CommonName)

	state.AltNames = r.parseAltNames(ctx, cert.AltNames)

	state.SerialNumber = types.StringValue(cert.SerialNumber)
	state.OrderId = types.StringValue(cert.OrderId)
	state.NotBefore = types.StringValue(cert.NotBefore)
	state.NotAfter = types.StringValue(cert.NotAfter)
	state.KeyAlgorithm = types.StringValue(cert.KeyAlgorithm)
	state.SignatureAlgorithm = types.StringValue(cert.SignatureAlgorithm)

	if state.TTL.IsNull() || state.TTL.IsUnknown() {
		if nb, err := time.Parse(time.RFC3339, cert.NotBefore); err == nil {
			if na, err := time.Parse(time.RFC3339, cert.NotAfter); err == nil {
				diff := na.Sub(nb)
				days := int(math.Round(diff.Hours() / 24))
				if days > 0 {
					state.TTL = types.StringValue(fmt.Sprintf("%dd", days))
				}
			}
		}
	}

	if cert.ProfileId != "" {
		state.ProfileId = types.StringValue(cert.ProfileId)
	}

	if state.TimeoutSeconds.IsNull() || state.TimeoutSeconds.IsUnknown() {
		state.TimeoutSeconds = types.Int64Value(3600)
	}

	if state.RevokeOnDestroy.IsNull() || state.RevokeOnDestroy.IsUnknown() {
		state.RevokeOnDestroy = types.BoolValue(false)
	}

	if state.RevocationReason.IsNull() || state.RevocationReason.IsUnknown() {
		state.RevocationReason = types.StringValue("unspecified")
	}

	if state.DeleteOnDestroy.IsNull() || state.DeleteOnDestroy.IsUnknown() {
		state.DeleteOnDestroy = types.BoolValue(false)
	}

	if cert.ApplicationId != "" {
		state.ApplicationId = types.StringValue(cert.ApplicationId)
	}

	if len(cert.KeyUsages) > 0 {
		normalized := make([]string, len(cert.KeyUsages))
		for i, u := range cert.KeyUsages {
			normalized[i] = normalizeKeyUsage(u)
		}
		keyUsagesList, diags := types.ListValueFrom(ctx, types.StringType, normalized)
		resp.Diagnostics.Append(diags...)
		state.KeyUsages = keyUsagesList
	} else if state.KeyUsages.IsNull() || state.KeyUsages.IsUnknown() || followed {
		state.KeyUsages = types.ListNull(types.StringType)
	}

	if len(cert.ExtendedKeyUsages) > 0 {
		normalized := make([]string, len(cert.ExtendedKeyUsages))
		for i, u := range cert.ExtendedKeyUsages {
			normalized[i] = normalizeExtendedKeyUsage(u)
		}
		extKeyUsagesList, diags := types.ListValueFrom(ctx, types.StringType, normalized)
		resp.Diagnostics.Append(diags...)
		state.ExtendedKeyUsages = extKeyUsagesList
	} else if state.ExtendedKeyUsages.IsNull() || state.ExtendedKeyUsages.IsUnknown() || followed {
		state.ExtendedKeyUsages = types.ListNull(types.StringType)
	}

	if cert.SubjectOrganization != "" {
		state.Organization = types.StringValue(cert.SubjectOrganization)
	} else if state.Organization.IsUnknown() || followed {
		state.Organization = types.StringNull()
	}

	if cert.SubjectOrganizationalUnit != "" {
		state.OU = types.StringValue(cert.SubjectOrganizationalUnit)
	} else if state.OU.IsUnknown() || followed {
		state.OU = types.StringNull()
	}

	if cert.SubjectCountry != "" {
		state.Country = types.StringValue(cert.SubjectCountry)
	} else if state.Country.IsUnknown() || followed {
		state.Country = types.StringNull()
	}

	if cert.SubjectState != "" {
		state.Province = types.StringValue(cert.SubjectState)
	} else if state.Province.IsUnknown() || followed {
		state.Province = types.StringNull()
	}

	if cert.SubjectLocality != "" {
		state.Locality = types.StringValue(cert.SubjectLocality)
	} else if state.Locality.IsUnknown() || followed {
		state.Locality = types.StringNull()
	}

	if cert.SubjectDomainComponents != "" {
		dcs := strings.Split(cert.SubjectDomainComponents, ",")
		for i := range dcs {
			dcs[i] = strings.TrimSpace(dcs[i])
		}
		if list, diags := types.ListValueFrom(ctx, types.StringType, dcs); !diags.HasError() {
			state.DomainComponents = list
		}
	} else if state.DomainComponents.IsNull() || len(state.DomainComponents.Elements()) > 0 {
		state.DomainComponents = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.update(ctx, req, req.Private, resp)
}

func (r *certManagerCertificateResource) update(ctx context.Context, req resource.UpdateRequest, private privateStateReader, resp *resource.UpdateResponse) {
	var plan, state certManagerCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Certificate.IsUnknown() || (!certificateCSRChanged(plan, state) && !certificateRenewalDue(plan, state)) {
		applyInPlaceChanges(&state, plan)
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		return
	}

	pending, resuming := readPendingRenewal(ctx, private)
	r.renewCertificate(ctx, plan, state, privateFlagSet(ctx, private, certificateIssuedFromCSRKey), pending, resuming, resp)
}

func (r *certManagerCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.delete(ctx, req, req.Private, resp)
}

func (r *certManagerCertificateResource) delete(ctx context.Context, req resource.DeleteRequest, private privateStateReader, resp *resource.DeleteResponse) {
	var state certManagerCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.RevokeOnDestroy.ValueBool() && !state.DeleteOnDestroy.ValueBool() {
		return
	}

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to destroy certificate",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	certificateId := state.Id.ValueString()

	if pending, ok := readPendingRenewal(ctx, private); ok {
		if err := r.client.CancelCertificateRequest(infisical.GetCertificateRequestStatusRequest{RequestId: pending.RequestId}); err != nil && err != infisical.ErrNotFound {
			resp.Diagnostics.AddWarning(
				"Pending renewal not cancelled",
				fmt.Sprintf("The pending renewal request %s of certificate %s could not be cancelled: %v. If it is still issued, revoke that certificate in Infisical.", pending.RequestId, certificateId, err),
			)
		}
	}

	certResponse, err := r.client.GetCertificate(infisical.GetCertificateRequest{CertificateId: certificateId, IncludeDeleted: true})
	if err == infisical.ErrNotFound {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading certificate before destroying it", err.Error())
		return
	}

	visited := map[string]bool{certificateId: true}
	older, ok := r.collectRenewalChain(certResponse.Certificate.RenewedFromCertificateId, func(c infisical.Certificate) string { return c.RenewedFromCertificateId }, visited, &resp.Diagnostics)
	if !ok {
		return
	}
	for left, right := 0, len(older)-1; left < right; left, right = left+1, right-1 {
		older[left], older[right] = older[right], older[left]
	}
	newer, ok := r.collectRenewalChain(certResponse.Certificate.RenewedByCertificateId, func(c infisical.Certificate) string { return c.RenewedByCertificateId }, visited, &resp.Diagnostics)
	if !ok {
		return
	}
	generations := append(append(older, certResponse.Certificate), newer...)

	if state.RevokeOnDestroy.ValueBool() {
		for _, generation := range generations {
			expired := notAfterPassed(generation.NotAfter)
			if generation.Id == certificateId {
				expired = certificateExpired(state)
			}
			if expired || generation.Status == certificateStatusRevoked {
				continue
			}
			if generation.CaId == "" {
				resp.Diagnostics.AddWarning(
					"Certificate not revoked",
					fmt.Sprintf("Certificate %s has no issuing certificate authority, for example because it is self-signed, so it cannot be revoked.", generation.Id),
				)
				continue
			}
			err = r.client.RevokeCertificate(infisical.RevokeCertificateRequest{
				CertificateId:    generation.Id,
				RevocationReason: certificateRevocationReasons[state.RevocationReason.ValueString()],
			})
			if err == nil || badRequestSays(err, "is already revoked") {
				continue
			}
			if err == infisical.ErrNotFound {
				_, readErr := r.client.GetCertificate(infisical.GetCertificateRequest{CertificateId: generation.Id, IncludeDeleted: true})
				if readErr == infisical.ErrNotFound {
					continue
				}
				if readErr != nil {
					resp.Diagnostics.AddError("Error revoking certificate", fmt.Sprintf("Revoking certificate %s returned not found, and reading it again failed: %v", generation.Id, readErr))
					return
				}
				resp.Diagnostics.AddError("Error revoking certificate", fmt.Sprintf("Revoking certificate %s returned not found, but the certificate still exists. Check that its certificate authority and app connection still exist.", generation.Id))
				return
			}
			permanent := badRequestSays(err, "not supported by this certificate authority type") || badRequestSays(err, "because it was imported and then deleted")
			if permanent && generation.DeletedAt != "" {
				resp.Diagnostics.AddWarning(
					"Deleted certificate not revoked",
					fmt.Sprintf("Certificate %s was already deleted and cannot be revoked: %v", generation.Id, err),
				)
				continue
			}
			resp.Diagnostics.AddError("Error revoking certificate", fmt.Sprintf("Failed to revoke certificate %s: %v", generation.Id, err))
			return
		}
	}

	if !state.DeleteOnDestroy.ValueBool() {
		return
	}
	for _, generation := range generations {
		if generation.DeletedAt != "" {
			continue
		}
		err = r.client.DeleteCertificate(infisical.DeleteCertificateRequest{CertificateId: generation.Id})
		if err != nil && err != infisical.ErrNotFound && !badRequestSays(err, "was already deleted") {
			resp.Diagnostics.AddError("Error deleting certificate", fmt.Sprintf("Failed to delete certificate %s: %v", generation.Id, err))
			return
		}
	}
}

func badRequestSays(err error, text string) bool {
	var apiErr *infisicalerrors.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 400 && strings.Contains(apiErr.ErrorMessage, text)
}

func (r *certManagerCertificateResource) collectRenewalChain(startId string, next func(infisical.Certificate) string, visited map[string]bool, diagnostics *diag.Diagnostics) ([]infisical.Certificate, bool) {
	chain := []infisical.Certificate{}
	for id := startId; id != ""; {
		if visited[id] {
			diagnostics.AddError("Renewal history loops", fmt.Sprintf("The renewal history of this certificate refers back to certificate %s, so Terraform cannot tell which certificates to clean up.", id))
			return nil, false
		}
		visited[id] = true
		certResponse, err := r.client.GetCertificate(infisical.GetCertificateRequest{CertificateId: id, IncludeDeleted: true})
		if err == infisical.ErrNotFound {
			break
		}
		if err != nil {
			diagnostics.AddError("Error reading renewal history before destroying it", fmt.Sprintf("Failed to read certificate %s: %v", id, err))
			return nil, false
		}
		chain = append(chain, certResponse.Certificate)
		id = next(certResponse.Certificate)
	}
	return chain, true
}

func (r *certManagerCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
