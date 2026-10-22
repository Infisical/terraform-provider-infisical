package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	infisicalerrors "terraform-provider-infisical/internal/errors"
)

var (
	certificateRevocationReasonTable = []struct{ name, api string }{
		{"unspecified", "UNSPECIFIED"},
		{"key_compromise", "KEY_COMPROMISE"},
		{"ca_compromise", "CA_COMPROMISE"},
		{"affiliation_changed", "AFFILIATION_CHANGED"},
		{"superseded", "SUPERSEDED"},
		{"cessation_of_operation", "CESSATION_OF_OPERATION"},
		{"certificate_hold", "CERTIFICATE_HOLD"},
		{"privilege_withdrawn", "PRIVILEGE_WITHDRAWN"},
		{"aa_compromise", "A_A_COMPROMISE"},
	}
	SUPPORTED_CERT_REVOCATION_REASONS, certificateRevocationReasons = buildCertificateRevocationReasons()
)

var nowFunc = time.Now

func buildCertificateRevocationReasons() ([]string, map[string]string) {
	names := make([]string, 0, len(certificateRevocationReasonTable))
	apiValues := make(map[string]string, len(certificateRevocationReasonTable))
	for _, reason := range certificateRevocationReasonTable {
		names = append(names, reason.name)
		apiValues[reason.name] = reason.api
	}
	return names, apiValues
}

const (
	certificateStatusActive      = "active"
	certificateStatusRevoked     = "revoked"
	certificateStatusRenewed     = "renewed"
	certificateIssuedFromCSRKey  = "issued_from_csr"
	certificatePendingRenewalKey = "pending_renewal_request_id"
	certificateCaTypeInternal    = "internal"

	certificateRenewalKeySourceCSR   = "csr"
	certificateRenewalKeySourceReuse = "reuse"
)

func (r *certManagerCertificateResource) followLatestRenewal(ctx context.Context, state *certManagerCertificateResourceModel, latestId string, diagnostics *diag.Diagnostics) (infisical.Certificate, bool) {
	latestResponse, err := r.client.GetCertificate(infisical.GetCertificateRequest{CertificateId: latestId})
	if err == infisical.ErrNotFound {
		tflog.Warn(ctx, "Latest renewal of the certificate is no longer available, keeping the current certificate", map[string]interface{}{
			"certificate_id": state.Id.ValueString(),
			"renewal_id":     latestId,
		})
		return infisical.Certificate{}, false
	}
	if err != nil {
		diagnostics.AddError("Error reading renewed certificate", fmt.Sprintf("Failed to read certificate %s, which renewed %s: %v", latestId, state.Id.ValueString(), err))
		return infisical.Certificate{}, false
	}

	tflog.Info(ctx, "Certificate was renewed outside Terraform, following the latest renewal", map[string]interface{}{
		"previous_certificate_id": state.Id.ValueString(),
		"certificate_id":          latestId,
	})

	if !r.populateCertificateMaterial(state, latestId, diagnostics) {
		return infisical.Certificate{}, false
	}
	state.Id = types.StringValue(latestId)

	return latestResponse.Certificate, true
}

func (r *certManagerCertificateResource) populateCertificateMaterial(state *certManagerCertificateResourceModel, certificateId string, diagnostics *diag.Diagnostics) bool {
	privateKey := ""
	bundle, err := r.client.GetCertificateBundle(infisical.GetCertificateRequest{CertificateId: certificateId})
	var apiErr *infisicalerrors.APIError
	if err != nil && errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
		var body infisical.GetCertificateBodyResponse
		body, err = r.client.GetCertificateBody(infisical.GetCertificateRequest{CertificateId: certificateId})
		bundle = infisical.GetCertificateBundleResponse{Certificate: body.Certificate, CertificateChain: body.CertificateChain, SerialNumber: body.SerialNumber}
		if err == nil {
			diagnostics.AddWarning(
				"Private key of certificate not available",
				fmt.Sprintf("This identity is not allowed to read the private key of certificate %s, so private_key is empty.", certificateId),
			)
		}
	} else if err == nil {
		privateKey = bundle.PrivateKey
	}
	if err != nil {
		diagnostics.AddError("Error reading certificate", fmt.Sprintf("Failed to read the certificate body of %s: %v", certificateId, err))
		return false
	}

	state.Certificate = types.StringValue(bundle.Certificate)
	state.CertificateChain = types.StringValue(bundle.CertificateChain)
	state.PrivateKey = types.StringValue(privateKey)
	return true
}

type privateStateReader interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

func pendingRenewalRequestId(ctx context.Context, private privateStateReader) string {
	value, _ := private.GetKey(ctx, certificatePendingRenewalKey)
	var requestId string
	if len(value) == 0 || json.Unmarshal(value, &requestId) != nil {
		return ""
	}
	return requestId
}

func (r *certManagerCertificateResource) resolvePendingRenewal(ctx context.Context, cert infisical.Certificate, requestId string, resp *resource.ReadResponse) {
	if cert.RenewedByCertificateId == "" {
		status, err := r.client.GetCertificateRequestStatus(infisical.GetCertificateRequestStatusRequest{RequestId: requestId})
		if err != nil || !strings.EqualFold(status.Status, "failed") {
			return
		}
		message := "no reason given"
		if status.ErrorMessage != nil && *status.ErrorMessage != "" {
			message = *status.ErrorMessage
		}
		resp.Diagnostics.AddWarning(
			"Certificate renewal failed",
			fmt.Sprintf("The renewal of certificate %s requested on a previous apply failed: %s. The next plan requests a new renewal if the certificate is still due.", cert.Id, message),
		)
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, certificatePendingRenewalKey, nil)...)
}

func applyInPlaceChanges(state *certManagerCertificateResourceModel, plan certManagerCertificateResourceModel) {
	state.CSR = plan.CSR
	state.TimeoutSeconds = plan.TimeoutSeconds
	state.ManualRenewDaysBeforeExpiry = plan.ManualRenewDaysBeforeExpiry
	state.RevokeOnDestroy = plan.RevokeOnDestroy
	state.RevocationReason = plan.RevocationReason
	state.DeleteOnDestroy = plan.DeleteOnDestroy
}

func certificateIssuanceFailed(diagnostics diag.Diagnostics) bool {
	for _, d := range diagnostics.Errors() {
		if d.Summary() == "Certificate issuance failed" {
			return true
		}
	}
	return false
}

func normalizeCSR(csr string) string {
	return strings.TrimSpace(strings.ReplaceAll(csr, "\r\n", "\n"))
}

func hasCSRValue(csr types.String) bool {
	return !csr.IsNull() && !csr.IsUnknown() && csr.ValueString() != ""
}

func certificateIssuanceModeChanged(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.State.Raw.IsNull() {
		return
	}
	issuedFromCSR, diags := req.Private.GetKey(ctx, certificateIssuedFromCSRKey)
	resp.Diagnostics.Append(diags...)
	stateHasCSR := hasCSRValue(req.StateValue) || string(issuedFromCSR) == "true"
	if req.PlanValue.IsUnknown() {
		resp.RequiresReplace = !stateHasCSR
		return
	}
	resp.RequiresReplace = stateHasCSR != hasCSRValue(req.PlanValue)
}

func certificateCSRChanged(plan, state certManagerCertificateResourceModel) bool {
	if !hasCSRValue(state.CSR) {
		return false
	}
	if plan.CSR.IsUnknown() {
		return true
	}
	return hasCSRValue(plan.CSR) && normalizeCSR(plan.CSR.ValueString()) != normalizeCSR(state.CSR.ValueString())
}

func parseCertificateTTLDays(ttl string) (float64, bool) {
	ttl = strings.TrimSpace(ttl)
	if len(ttl) < 2 {
		return 0, false
	}
	var value float64
	if _, err := fmt.Sscanf(ttl[:len(ttl)-1], "%g", &value); err != nil || value <= 0 {
		return 0, false
	}
	switch ttl[len(ttl)-1] {
	case 's':
		return value / (24 * 60 * 60), true
	case 'm':
		return value / (24 * 60), true
	case 'h':
		return value / 24, true
	case 'd':
		return value, true
	case 'w':
		return value * 7, true
	case 'y':
		return value * 365, true
	}
	return 0, false
}

func certificateLifetimeDays(plan, state certManagerCertificateResourceModel) (float64, bool) {
	ttlChanged := !plan.TTL.IsUnknown() && !plan.TTL.Equal(state.TTL)
	if ttlChanged && !plan.TTL.IsNull() {
		if days, ok := parseCertificateTTLDays(plan.TTL.ValueString()); ok {
			return days, true
		}
	}
	notBefore, err := time.Parse(time.RFC3339, state.NotBefore.ValueString())
	if err != nil {
		return 0, false
	}
	notAfter, err := time.Parse(time.RFC3339, state.NotAfter.ValueString())
	if err != nil {
		return 0, false
	}
	return notAfter.Sub(notBefore).Hours() / 24, true
}

func certificateRenewalDue(plan, state certManagerCertificateResourceModel) bool {
	if plan.ManualRenewDaysBeforeExpiry.IsNull() || plan.ManualRenewDaysBeforeExpiry.IsUnknown() {
		return false
	}
	notAfter, err := time.Parse(time.RFC3339, state.NotAfter.ValueString())
	if err != nil {
		return false
	}
	renewAt := notAfter.Add(-time.Duration(plan.ManualRenewDaysBeforeExpiry.ValueInt64()) * 24 * time.Hour)
	return !nowFunc().Before(renewAt)
}

func certificateExpired(state certManagerCertificateResourceModel) bool {
	notAfter, err := time.Parse(time.RFC3339, state.NotAfter.ValueString())
	return err == nil && !nowFunc().Before(notAfter)
}

func validateRenewalWindowOnCreate(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan certManagerCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.ManualRenewDaysBeforeExpiry.IsNull() || plan.ManualRenewDaysBeforeExpiry.IsUnknown() || plan.TTL.IsNull() || plan.TTL.IsUnknown() {
		return
	}
	lifetimeDays, ok := parseCertificateTTLDays(plan.TTL.ValueString())
	if ok && float64(plan.ManualRenewDaysBeforeExpiry.ValueInt64()) >= lifetimeDays {
		resp.Diagnostics.AddAttributeError(
			path.Root("manual_renew_days_before_expiry"),
			"Renewal window too large",
			fmt.Sprintf("manual_renew_days_before_expiry (%d) must be smaller than the certificate lifetime (%.0f days), otherwise every apply renews the certificate again.", plan.ManualRenewDaysBeforeExpiry.ValueInt64(), lifetimeDays),
		)
	}
}

func (r *certManagerCertificateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if req.State.Raw.IsNull() {
		validateRenewalWindowOnCreate(ctx, req, resp)
		return
	}

	var plan, state, config certManagerCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.CSR.IsNull() && hasCSRValue(state.CSR) {
		plan.CSR = types.StringNull()
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("csr"))
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
		return
	}

	csrChanged := certificateCSRChanged(plan, state)
	renewDays := plan.ManualRenewDaysBeforeExpiry
	renewDaysSet := !renewDays.IsNull() && !renewDays.IsUnknown()
	if !csrChanged && !renewDaysSet {
		return
	}

	if renewDaysSet {
		if lifetimeDays, ok := certificateLifetimeDays(plan, state); ok && float64(renewDays.ValueInt64()) >= lifetimeDays {
			resp.Diagnostics.AddAttributeError(
				path.Root("manual_renew_days_before_expiry"),
				"Renewal window too large",
				fmt.Sprintf("manual_renew_days_before_expiry (%d) must be smaller than the certificate lifetime (%.0f days), otherwise every apply renews the certificate again.", renewDays.ValueInt64(), lifetimeDays),
			)
			return
		}
	}

	if pendingRequestId := pendingRenewalRequestId(ctx, req.Private); pendingRequestId != "" {
		if csrChanged {
			resp.Diagnostics.AddAttributeError(
				path.Root("csr"),
				"Certificate renewal still pending",
				fmt.Sprintf("Certificate %s has a renewal from a previous apply that is still pending (request %s). Wait for it to finish before changing the CSR.", state.Id.ValueString(), pendingRequestId),
			)
		} else if certificateRenewalDue(plan, state) {
			resp.Diagnostics.AddWarning(
				"Certificate renewal still pending",
				fmt.Sprintf("Certificate %s has a renewal from a previous apply that is still pending (request %s), so no new renewal is planned. Terraform picks up the renewed certificate on a later refresh.", state.Id.ValueString(), pendingRequestId),
			)
		}
		return
	}

	if state.Status.ValueString() == certificateStatusRevoked || state.Status.ValueString() == certificateStatusRenewed || certificateExpired(state) {
		if csrChanged {
			resp.Diagnostics.AddAttributeError(
				path.Root("csr"),
				"Certificate can no longer be renewed",
				fmt.Sprintf("Certificate %s is revoked, expired, or already renewed by a certificate that is revoked, deleted or not readable by this identity, so it cannot be renewed with the new CSR. Create a new infisical_cert_manager_certificate resource instead.", state.Id.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddWarning(
			"Certificate can no longer be renewed",
			fmt.Sprintf("Certificate %s is revoked, expired, or already renewed by a certificate that is revoked, deleted or not readable by this identity, so manual_renew_days_before_expiry has no effect on this resource. Create a new infisical_cert_manager_certificate resource to get a valid certificate.", state.Id.ValueString()),
		)
		return
	}

	if !csrChanged {
		if !certificateRenewalDue(plan, state) {
			return
		}
		resp.Diagnostics.AddWarning(
			"Certificate will be renewed",
			fmt.Sprintf("Certificate %s expires at %s, which is within manual_renew_days_before_expiry (%d days). It will be renewed in place on apply.", state.Id.ValueString(), state.NotAfter.ValueString(), renewDays.ValueInt64()),
		)
	}

	plan.Id = types.StringUnknown()
	plan.CertificateRequestId = types.StringUnknown()
	plan.Status = types.StringUnknown()
	plan.SerialNumber = types.StringUnknown()
	plan.NotBefore = types.StringUnknown()
	plan.NotAfter = types.StringUnknown()
	plan.Certificate = types.StringUnknown()
	plan.CertificateChain = types.StringUnknown()
	plan.PrivateKey = types.StringUnknown()

	if csrChanged {
		subjectStrings := map[*types.String]types.String{
			&plan.CommonName:         config.CommonName,
			&plan.Organization:       config.Organization,
			&plan.OU:                 config.OU,
			&plan.Country:            config.Country,
			&plan.Province:           config.Province,
			&plan.Locality:           config.Locality,
			&plan.KeyAlgorithm:       config.KeyAlgorithm,
			&plan.SignatureAlgorithm: config.SignatureAlgorithm,
		}
		for planValue, configValue := range subjectStrings {
			if configValue.IsNull() {
				*planValue = types.StringUnknown()
			}
		}
		subjectLists := map[*types.List]types.List{
			&plan.AltNames:          config.AltNames,
			&plan.DomainComponents:  config.DomainComponents,
			&plan.KeyUsages:         config.KeyUsages,
			&plan.ExtendedKeyUsages: config.ExtendedKeyUsages,
		}
		for planValue, configValue := range subjectLists {
			if configValue.IsNull() {
				*planValue = types.ListUnknown(types.StringType)
			}
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}
func (r *certManagerCertificateResource) keepCertificateWhileRenewalPending(ctx context.Context, plan, state certManagerCertificateResourceModel, requestId string, pollDiagnostics diag.Diagnostics, resp *resource.UpdateResponse) {
	applyInPlaceChanges(&state, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Private != nil {
		value, _ := json.Marshal(requestId)
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, certificatePendingRenewalKey, value)...)
	}
	reason := "it did not finish in time"
	if errs := pollDiagnostics.Errors(); len(errs) > 0 {
		reason = errs[0].Detail()
	}
	resp.Diagnostics.AddWarning(
		"Certificate renewal still pending",
		fmt.Sprintf("The renewal of certificate %s was requested (request %s), but %s. The current certificate stays in state, no second renewal is requested, and Terraform picks up the renewed certificate on a later refresh.", state.Id.ValueString(), requestId, reason),
	)
}

func (r *certManagerCertificateResource) certificateHasInternalCa(certificateId string) bool {
	certResponse, err := r.client.GetCertificate(infisical.GetCertificateRequest{CertificateId: certificateId})
	return err == nil && certResponse.Certificate.CaType != nil && *certResponse.Certificate.CaType == certificateCaTypeInternal
}

func (r *certManagerCertificateResource) renewCertificate(ctx context.Context, plan, state certManagerCertificateResourceModel, issuedFromCSR bool, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to renew certificate",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	isCSRCertificate := hasCSRValue(plan.CSR)
	renewRequest := infisical.RenewCertificateRequest{CertificateId: state.Id.ValueString()}
	csrChanged := isCSRCertificate && certificateCSRChanged(plan, state)
	if (isCSRCertificate || issuedFromCSR) && !csrChanged && r.certificateHasInternalCa(state.Id.ValueString()) {
		renewRequest.RenewalKeySource = certificateRenewalKeySourceReuse
	} else if isCSRCertificate {
		renewRequest.RenewalKeySource = certificateRenewalKeySourceCSR
		renewRequest.CSR = normalizeCSR(plan.CSR.ValueString())
	}

	renewResponse, err := r.client.RenewCertificate(renewRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error renewing certificate", err.Error())
		return
	}

	plan.CertificateRequestId = types.StringValue(renewResponse.CertificateRequestId)

	if renewResponse.CertificateId != "" && renewResponse.Certificate != "" {
		r.populatePlanFromImmediateResponse(ctx, &plan, infisical.CertificateResponse{
			CertificateId:    renewResponse.CertificateId,
			Certificate:      renewResponse.Certificate,
			CertificateChain: renewResponse.CertificateChain,
			SerialNumber:     renewResponse.SerialNumber,
			PrivateKey:       renewResponse.PrivateKey,
		})
	} else {
		timeoutSeconds := int64(3600)
		if !plan.TimeoutSeconds.IsNull() && !plan.TimeoutSeconds.IsUnknown() {
			timeoutSeconds = plan.TimeoutSeconds.ValueInt64()
		}
		var pollDiagnostics diag.Diagnostics
		r.pollCertificateRequest(ctx, &plan, renewResponse.CertificateRequestId, timeoutSeconds, &pollDiagnostics)
		if certificateIssuanceFailed(pollDiagnostics) {
			resp.Diagnostics.Append(pollDiagnostics...)
			return
		}
		if pollDiagnostics.HasError() {
			r.keepCertificateWhileRenewalPending(ctx, plan, state, renewResponse.CertificateRequestId, pollDiagnostics, resp)
			return
		}
	}

	var bundleErr error
	if !isCSRCertificate && plan.PrivateKey.ValueString() == "" {
		var bundle infisical.GetCertificateBundleResponse
		bundle, bundleErr = r.client.GetCertificateBundle(infisical.GetCertificateRequest{CertificateId: plan.Id.ValueString()})
		if bundleErr == nil {
			plan.PrivateKey = types.StringValue(bundle.PrivateKey)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)

	var apiErr *infisicalerrors.APIError
	if bundleErr != nil && errors.As(bundleErr, &apiErr) && apiErr.StatusCode == 403 {
		resp.Diagnostics.AddWarning(
			"Private key of certificate not available",
			fmt.Sprintf("Certificate %s was renewed as %s, but this identity is not allowed to read its private key, so private_key is empty.", state.Id.ValueString(), plan.Id.ValueString()),
		)
	} else if bundleErr != nil {
		resp.Diagnostics.AddError(
			"Error reading private key of renewed certificate",
			fmt.Sprintf("Certificate %s was renewed as %s, but its private key could not be read: %v", state.Id.ValueString(), plan.Id.ValueString(), bundleErr),
		)
	}
}
