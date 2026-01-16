package resource

import (
	"context"
	"fmt"
	"os"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	SUPPORTED_CERT_ISSUE_KEY_ALGORITHMS = []string{
		"RSA_2048", "RSA_3072", "RSA_4096",
		"ECDSA_P256", "ECDSA_P384", "ECDSA_P521",
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
)

var (
	_ resource.Resource = &certManagerCertificateResource{}
)

func NewCertManagerCertificateResource() resource.Resource {
	return &certManagerCertificateResource{}
}

type certManagerCertificateResource struct {
	client *infisical.Client
}

type certManagerCertificateResourceModel struct {
	ProfileId            types.String `tfsdk:"profile_id"`
	CSRPath              types.String `tfsdk:"csr_path"`
	CommonName           types.String `tfsdk:"common_name"`
	AltNames             types.List   `tfsdk:"alt_names"`
	Organization         types.String `tfsdk:"organization"`
	OU                   types.String `tfsdk:"ou"`
	Country              types.String `tfsdk:"country"`
	Province             types.String `tfsdk:"province"`
	Locality             types.String `tfsdk:"locality"`
	KeyAlgorithm         types.String `tfsdk:"key_algorithm"`
	SignatureAlgorithm   types.String `tfsdk:"signature_algorithm"`
	KeyUsages            types.List   `tfsdk:"key_usages"`
	ExtendedKeyUsages    types.List   `tfsdk:"extended_key_usages"`
	TTL                  types.String `tfsdk:"ttl"`
	TimeoutSeconds       types.Int64  `tfsdk:"timeout_seconds"`
	Id                   types.String `tfsdk:"id"`
	CertificateRequestId types.String `tfsdk:"certificate_request_id"`
	Status               types.String `tfsdk:"status"`
	SerialNumber         types.String `tfsdk:"serial_number"`
	NotBefore            types.String `tfsdk:"not_before"`
	NotAfter             types.String `tfsdk:"not_after"`
	Certificate          types.String `tfsdk:"certificate"`
	PrivateKey           types.String `tfsdk:"private_key"`
	CertificateChain     types.String `tfsdk:"certificate_chain"`
}

func (r *certManagerCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_certificate"
}

func (r *certManagerCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Request and manage certificates from Infisical certificate profiles. Supports both CSR-based and direct field requests. The resource will poll until the certificate is issued or a timeout is reached. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"profile_id": schema.StringAttribute{
				Description: "The ID of the certificate profile to use for issuance",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"csr_path": schema.StringAttribute{
				Description: "Path to a Certificate Signing Request (CSR) file in PEM format. If provided, the certificate will be issued based on the CSR",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"common_name": schema.StringAttribute{
				Description: "The common name (CN) for the certificate. Required when not using CSR",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alt_names": schema.ListAttribute{
				Description: "Subject alternative names (SANs) for the certificate",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "The organization (O) for the certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ou": schema.StringAttribute{
				Description: "The organizational unit (OU) for the certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"country": schema.StringAttribute{
				Description: "The country (C) for the certificate (2-letter code)",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"province": schema.StringAttribute{
				Description: "The state/province (ST) for the certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locality": schema.StringAttribute{
				Description: "The locality (L) for the certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_usages": schema.ListAttribute{
				Description: "Key usages for the certificate. Supported: " + strings.Join(SUPPORTED_CERT_ISSUE_KEY_USAGES, ", "),
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_KEY_USAGES...)),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"extended_key_usages": schema.ListAttribute{
				Description: "Extended key usages for the certificate. Supported: " + strings.Join(SUPPORTED_CERT_ISSUE_EXTENDED_KEY_USAGES, ", "),
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_CERT_ISSUE_EXTENDED_KEY_USAGES...)),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.StringAttribute{
				Description: "Time to live for the certificate (e.g., '30d', '90d', '1y')",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "Maximum time to wait for certificate issuance in seconds. Defaults to 3600 (1 hour)",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3600),
			},
			"id": schema.StringAttribute{
				Description: "The ID of the certificate",
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
				Description: "The status of the certificate (pending, issued, failed)",
				Computed:    true,
			},
			"serial_number": schema.StringAttribute{
				Description: "The serial number of the issued certificate",
				Computed:    true,
			},
			"not_before": schema.StringAttribute{
				Description: "The not-before date of the certificate (RFC3339 format)",
				Computed:    true,
			},
			"not_after": schema.StringAttribute{
				Description: "The not-after (expiration) date of the certificate (RFC3339 format)",
				Computed:    true,
			},
			"certificate": schema.StringAttribute{
				Description: "The issued certificate in PEM format",
				Computed:    true,
				Sensitive:   true,
			},
			"private_key": schema.StringAttribute{
				Description: "The private key in PEM format (only available for direct field requests, not CSR-based)",
				Computed:    true,
				Sensitive:   true,
			},
			"certificate_chain": schema.StringAttribute{
				Description: "The certificate chain in PEM format",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
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

	hasCSR := !plan.CSRPath.IsNull() && !plan.CSRPath.IsUnknown() && plan.CSRPath.ValueString() != ""
	hasCommonName := !plan.CommonName.IsNull() && !plan.CommonName.IsUnknown() && plan.CommonName.ValueString() != ""

	if !hasCSR && !hasCommonName {
		resp.Diagnostics.AddError(
			"Missing certificate request data",
			"Either 'csr_path' or 'common_name' must be provided",
		)
		return
	}

	timeoutSeconds := int64(3600)
	if !plan.TimeoutSeconds.IsNull() && !plan.TimeoutSeconds.IsUnknown() {
		timeoutSeconds = plan.TimeoutSeconds.ValueInt64()
	}

	certRequest := infisical.RequestCertificateRequest{
		ProfileId: plan.ProfileId.ValueString(),
	}

	if hasCSR {
		csrBytes, err := readFileContent(plan.CSRPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading CSR file",
				fmt.Sprintf("Failed to read CSR file at %s: %v", plan.CSRPath.ValueString(), err),
			)
			return
		}
		certRequest.CSR = string(csrBytes)
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

	r.pollCertificateRequest(ctx, &plan, certificateRequestId, timeoutSeconds, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCertificateResource) populatePlanFromImmediateResponse(ctx context.Context, plan *certManagerCertificateResourceModel, certResponse infisical.CertificateResponse) {
	plan.Id = types.StringValue(certResponse.CertificateId)
	plan.Status = types.StringValue("issued")

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
		plan.NotBefore = types.StringValue("")
		plan.NotAfter = types.StringValue("")
		return
	}

	cert := certDetails.Certificate
	plan.CommonName = types.StringValue(cert.CommonName)
	plan.KeyAlgorithm = types.StringValue(cert.KeyAlgorithm)
	plan.SignatureAlgorithm = types.StringValue(cert.SignatureAlgorithm)
	plan.AltNames = r.parseAltNames(ctx, cert.AltNames)
	plan.NotBefore = types.StringValue(cert.NotBefore)
	plan.NotAfter = types.StringValue(cert.NotAfter)

	if cert.CertificateChain != "" {
		plan.CertificateChain = types.StringValue(cert.CertificateChain)
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
		return types.ListNull(types.StringType)
	}
	return altNamesList
}

func (r *certManagerCertificateResource) pollCertificateRequest(ctx context.Context, plan *certManagerCertificateResourceModel, certificateRequestId string, timeoutSeconds int64, resp *resource.CreateResponse) {
	timeout := time.Duration(timeoutSeconds) * time.Second
	pollInterval := 5 * time.Second
	startTime := time.Now()

	for {
		if ctx.Err() != nil {
			resp.Diagnostics.AddError("Operation cancelled", ctx.Err().Error())
			return
		}

		if time.Since(startTime) > timeout {
			resp.Diagnostics.AddError(
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
					resp.Diagnostics.AddError("Operation cancelled", ctx.Err().Error())
					return
				case <-time.After(pollInterval):
					continue
				}
			}
			resp.Diagnostics.AddError(
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

		case "failed":
			errorMsg := "Certificate issuance failed"
			if statusResponse.ErrorMessage != nil && *statusResponse.ErrorMessage != "" {
				errorMsg = fmt.Sprintf("Certificate issuance failed: %s", *statusResponse.ErrorMessage)
			}
			resp.Diagnostics.AddError(
				"Certificate issuance failed",
				fmt.Sprintf("%s. Request ID: %s", errorMsg, certificateRequestId),
			)
			return

		case "pending":
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError("Operation cancelled", ctx.Err().Error())
				return
			case <-time.After(pollInterval):
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
				resp.Diagnostics.AddError("Operation cancelled", ctx.Err().Error())
				return
			case <-time.After(pollInterval):
				continue
			}
		}
	}
}

func (r *certManagerCertificateResource) handleIssuedCertificate(ctx context.Context, plan *certManagerCertificateResourceModel, statusResponse *infisical.GetCertificateRequestStatusResponse) {
	plan.Status = types.StringValue("issued")

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
		CertificateId: state.Id.ValueString(),
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

	state.Status = types.StringValue(cert.Status)
	state.CommonName = types.StringValue(cert.CommonName)

	state.AltNames = r.parseAltNames(ctx, cert.AltNames)

	state.SerialNumber = types.StringValue(cert.SerialNumber)
	state.NotBefore = types.StringValue(cert.NotBefore)
	state.NotAfter = types.StringValue(cert.NotAfter)
	state.KeyAlgorithm = types.StringValue(cert.KeyAlgorithm)
	state.SignatureAlgorithm = types.StringValue(cert.SignatureAlgorithm)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Certificate updates not supported",
		"Certificates cannot be updated. To change certificate attributes, destroy and recreate the resource.",
	)
}

func (r *certManagerCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Certificates cannot be deleted from Infisical. Only removed from state.
}

func (r *certManagerCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func readFileContent(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
