package resource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	infisical "terraform-provider-infisical/internal/client"
	infisicalerrors "terraform-provider-infisical/internal/errors"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultEmailDomainVerificationTimeout     = 10 * time.Minute
	defaultEmailDomainVerificationTimeoutText = "10m"
)

// Each attempt is a write request with a DNS lookup on the server, so retries back off.
var (
	emailDomainVerificationInitialInterval = 5 * time.Second
	emailDomainVerificationMaxInterval     = 30 * time.Second
)

// The error codes that the verify endpoint returns. They decide whether a failed verification is retried.
const (
	emailDomainErrorDNSLookupFailed     = "EMAIL_DOMAIN_DNS_LOOKUP_FAILED"
	emailDomainErrorDNSRecordMismatch   = "EMAIL_DOMAIN_DNS_RECORD_MISMATCH"
	emailDomainErrorVerificationExpired = "EMAIL_DOMAIN_VERIFICATION_EXPIRED"
	emailDomainErrorAlreadyVerified     = "EMAIL_DOMAIN_ALREADY_VERIFIED"
)

var (
	_ resource.Resource                = &EmailDomainVerificationResource{}
	_ resource.ResourceWithConfigure   = &EmailDomainVerificationResource{}
	_ resource.ResourceWithImportState = &EmailDomainVerificationResource{}
)

func NewEmailDomainVerificationResource() resource.Resource {
	return &EmailDomainVerificationResource{}
}

type EmailDomainVerificationResource struct {
	client *infisical.Client
}

type EmailDomainVerificationResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	EmailDomainID types.String   `tfsdk:"email_domain_id"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func (r *EmailDomainVerificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_domain_verification"
}

func (r *EmailDomainVerificationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Verify an email domain that was claimed with `infisical_email_domain`. " +
			"Infisical looks up the DNS record in the `verification_record` of the email domain. " +
			fmt.Sprintf("Create retries while the record is not found yet, because DNS changes take time to propagate, and waits up to the create value in timeouts (default: %s). ", defaultEmailDomainVerificationTimeoutText) +
			"If the domain is no longer verified, the next plan creates this resource again. " +
			"The API cannot remove a verification, so destroying this resource only removes it from the Terraform state. " +
			"Verifying a domain needs a plan that includes SAML, LDAP or OIDC SSO. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the verified email domain. The same value as `email_domain_id`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"email_domain_id": schema.StringAttribute{
				Description:   "The ID of the email domain to verify (the `id` of an `infisical_email_domain`).",
				Required:      true,
				Validators:    []validator.String{infisicaltf.UuidValidator},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: fmt.Sprintf("How long Terraform retries the verification while the DNS TXT record is not found, such as \"5m\" or \"30m\". Defaults to %s.", defaultEmailDomainVerificationTimeoutText),
			}),
		},
	}
}

func (r *EmailDomainVerificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func isEmailDomainDNSNotReady(apiErr *infisicalerrors.APIError) bool {
	return apiErr.ErrorCode == emailDomainErrorDNSLookupFailed || apiErr.ErrorCode == emailDomainErrorDNSRecordMismatch
}

// verifyEmailDomain calls verify until it succeeds, fails for a reason that a retry cannot fix, or ctx ends.
func (r *EmailDomainVerificationResource) verifyEmailDomain(ctx context.Context, emailDomainID string) error {
	var wait time.Duration
	lastMessage := "no attempt finished"

	waitEnded := func() error {
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("stopped verifying the email domain because the operation was cancelled. Last response: %s", lastMessage)
		}
		return fmt.Errorf("timed out waiting for Infisical to find the DNS TXT record. To wait longer, increase timeouts.create. Last response: %s", lastMessage)
	}

	for {
		if wait > 0 {
			select {
			case <-ctx.Done():
				return waitEnded()
			case <-time.After(wait):
			}
		}

		_, err := r.client.VerifyEmailDomainWithContext(ctx, infisical.VerifyEmailDomainRequest{ID: emailDomainID})
		if err == nil {
			return nil
		}
		if errors.Is(err, infisical.ErrNotFound) {
			return fmt.Errorf("email domain %s was not found", emailDomainID)
		}
		if ctx.Err() != nil {
			return waitEnded()
		}

		var apiErr *infisicalerrors.APIError
		if !errors.As(err, &apiErr) {
			return err
		}
		switch {
		// The API answers 409 when this organization already verified the domain, which is the result we want.
		case apiErr.StatusCode == http.StatusConflict:
			return nil
		case isEmailDomainDNSNotReady(apiErr):
			lastMessage = apiErr.ErrorMessage
			wait = nextEmailDomainVerificationWait(wait, emailDomainVerificationInitialInterval, emailDomainVerificationMaxInterval)
		case apiErr.ErrorCode == emailDomainErrorVerificationExpired:
			return fmt.Errorf("%w. The verification code expires 7 days after the email domain is created. Replace the infisical_email_domain (for example with terraform apply -replace) to get a new code, then update the DNS TXT record", err)
		case apiErr.ErrorCode == emailDomainErrorAlreadyVerified:
			return fmt.Errorf("%w. Another organization has already verified this domain, and a domain can be verified by only one organization", err)
		default:
			return err
		}
	}
}

// nextEmailDomainVerificationWait returns the first interval, then doubles the previous wait up to the cap.
func nextEmailDomainVerificationWait(previous, initial, maxInterval time.Duration) time.Duration {
	if previous <= 0 {
		return initial
	}
	return min(previous*2, maxInterval)
}

func (r *EmailDomainVerificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create email domain verification",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan EmailDomainVerificationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, defaultEmailDomainVerificationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	verifyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := r.verifyEmailDomain(verifyCtx, plan.EmailDomainID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error verifying email domain",
			"Couldn't verify email domain, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = plan.EmailDomainID
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *EmailDomainVerificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read email domain verification",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state EmailDomainVerificationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	emailDomain, err := r.client.GetEmailDomainByID(infisical.GetEmailDomainByIDRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading email domain verification",
			"Couldn't read email domain from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	// Without a verified status there is no verification to keep: remove it
	if emailDomain.Status != infisical.EmailDomainStatusVerified {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(emailDomain.ID)
	state.EmailDomainID = types.StringValue(emailDomain.ID)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update only runs when timeouts change, because email_domain_id is ForceNew. Nothing goes to the API.
func (r *EmailDomainVerificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EmailDomainVerificationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *EmailDomainVerificationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// The API cannot remove a verification. Terraform removes the resource from state.
}

func (r *EmailDomainVerificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.ParseUUID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be the ID of the email domain (a UUID)",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email_domain_id"), req.ID)...)
}
