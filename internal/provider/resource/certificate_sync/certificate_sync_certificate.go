package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// certificateSyncCertificatesPageSize is the max page size accepted by the list certificates endpoint.
const certificateSyncCertificatesPageSize = 100

// renewalGuidance is shared by the Read warning and the Create error so both explain the same
// thing: Infisical attaches the renewed certificate to the sync and drops the superseded one, and
// it refuses to sync certificates that have been renewed, revoked, or expired. The provider does
// not fetch the certificate to work out which of those applies, so one message covers all three.
const renewalGuidance = "This usually means the certificate was renewed: Infisical associates the renewed certificate with the sync " +
	"and drops the superseded one. Certificates that have been renewed, revoked, or expired cannot be synced. " +
	"Update certificate_id to the renewed certificate before applying. Referencing the resource that performs the renewal " +
	"keeps the ID in step automatically."

// CertificateSyncCertificateResource attaches one certificate to one certificate sync. Modelling a
// single association per resource (rather than a list of certificate IDs on the sync) means attaching
// or detaching one certificate leaves the others untouched in both the plan and the API call.
type CertificateSyncCertificateResource struct {
	client *infisical.Client
}

func NewCertificateSyncCertificateResource() resource.Resource {
	return &CertificateSyncCertificateResource{}
}

type CertificateSyncCertificateResourceModel struct {
	ID                types.String `tfsdk:"id"`
	CertificateSyncID types.String `tfsdk:"certificate_sync_id"`
	CertificateID     types.String `tfsdk:"certificate_id"`
}

func (r *CertificateSyncCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate_sync_certificate"
}

// ImportState imports an association using the composite ID "<certificate_sync_id>:<certificate_id>".
// Read then resolves the association's own ID.
func (r *CertificateSyncCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<certificate_sync_id>:<certificate_id>\", got: "+req.ID,
		)
		return
	}

	if _, err := uuid.Parse(parts[0]); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected the certificate sync ID to be a valid UUID, got: "+parts[0])
		return
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected the certificate ID to be a valid UUID, got: "+parts[1])
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate_sync_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate_id"), parts[1])...)
}

func (r *CertificateSyncCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attach a certificate to a certificate sync so it is synced to the destination. The certificate must belong to the same application as the certificate sync. " +
			"Only a currently valid certificate can be synced: certificates that have been renewed, revoked, or expired are rejected. " +
			"Because Infisical attaches the renewed certificate and drops the superseded one when a certificate is renewed, pinning a literal " +
			"certificate ID will break at the next renewal. Reference the resource that issues or renews the certificate instead so the ID stays in step automatically.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the certificate association.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"certificate_sync_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the certificate sync to associate the certificate with.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"certificate_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the certificate to associate with the certificate sync.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *CertificateSyncCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Source Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// findAssociation pages through the sync's certificates and returns the association ID for the
// given certificate ID, or an empty string when the certificate is not associated.
func (r *CertificateSyncCertificateResource) findAssociation(certificateSyncID, certificateID string) (string, error) {
	offset := 0
	for {
		page, err := r.client.ListCertificateSyncCertificates(infisical.ListCertificateSyncCertificatesRequest{
			CertificateSyncID: certificateSyncID,
			Offset:            offset,
			Limit:             certificateSyncCertificatesPageSize,
		})
		if err != nil {
			return "", err
		}

		for _, cert := range page.Certificates {
			if cert.CertificateID == certificateID {
				return cert.ID, nil
			}
		}

		// Advance by how many items came back rather than by the limit we asked for: if a page
		// returns fewer items than requested, advancing by the limit would skip the difference.
		offset += len(page.Certificates)
		if len(page.Certificates) == 0 || offset >= page.TotalCount {
			return "", nil
		}
	}
}

func (r *CertificateSyncCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to add certificate to certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan CertificateSyncCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	added, err := r.client.AddCertificateSyncCertificates(infisical.AddCertificateSyncCertificatesRequest{
		CertificateSyncID: plan.CertificateSyncID.ValueString(),
		CertificateIDs:    []string{plan.CertificateID.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding certificate to certificate sync",
			fmt.Sprintf(
				"Couldn't add certificate %q to certificate sync %q. %s\n\nOriginal error: %s",
				plan.CertificateID.ValueString(), plan.CertificateSyncID.ValueString(), renewalGuidance, err.Error(),
			),
		)
		return
	}

	// The add response should echo the created association. If it does not, resolve the ID via
	// a lookup so a successful create never persists an empty computed ID.
	associationID := ""
	for _, cert := range added {
		if cert.CertificateID == plan.CertificateID.ValueString() {
			associationID = cert.ID
			break
		}
	}
	if associationID == "" {
		associationID, err = r.findAssociation(plan.CertificateSyncID.ValueString(), plan.CertificateID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error resolving certificate association",
				"Couldn't resolve the certificate association after adding it, unexpected error: "+err.Error(),
			)
			return
		}
	}
	if associationID == "" {
		resp.Diagnostics.AddError(
			"Error adding certificate to certificate sync",
			"The certificate was added but its association could not be found.",
		)
		return
	}

	plan.ID = types.StringValue(associationID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CertificateSyncCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read certificate sync certificate",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state CertificateSyncCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	associationID, err := r.findAssociation(state.CertificateSyncID.ValueString(), state.CertificateID.ValueString())
	if err != nil {
		if err == infisical.ErrNotFound {
			// The whole sync is gone, which is ordinary deletion rather than something the user
			// needs to act on, so drop the association without the warning issued further down.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading certificate sync certificate",
			"Couldn't read certificate sync certificate association, unexpected error: "+err.Error(),
		)
		return
	}

	// The sync still exists but the certificate is no longer attached to it. Warn rather than
	// error: erroring during refresh would break `terraform plan` and `terraform destroy`. The
	// resource is dropped from state so Terraform plans a re-create, which fails loudly if the
	// pinned certificate is no longer syncable.
	if associationID == "" {
		resp.Diagnostics.AddWarning(
			"Certificate is no longer attached to the certificate sync",
			fmt.Sprintf(
				"Certificate %q is no longer attached to certificate sync %q. %s",
				state.CertificateID.ValueString(), state.CertificateSyncID.ValueString(), renewalGuidance,
			),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(associationID)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *CertificateSyncCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CertificateSyncCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CertificateSyncCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to remove certificate from certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state CertificateSyncCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemoveCertificateSyncCertificates(infisical.RemoveCertificateSyncCertificatesRequest{
		CertificateSyncID: state.CertificateSyncID.ValueString(),
		CertificateIDs:    []string{state.CertificateID.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing certificate from certificate sync",
			"Couldn't remove certificate from certificate sync, unexpected error: "+err.Error(),
		)
	}
}
