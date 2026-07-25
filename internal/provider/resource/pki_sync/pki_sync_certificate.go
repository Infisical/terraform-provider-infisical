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

// pkiSyncCertificatesPageSize is the max page size accepted by the list certificates endpoint.
const pkiSyncCertificatesPageSize = 100

// PkiSyncCertificateResource links a single certificate to a PKI sync. Each association is
// managed independently so plans stay drift-free regardless of how many certificates a sync holds.
type PkiSyncCertificateResource struct {
	client *infisical.Client
}

func NewPkiSyncCertificateResource() resource.Resource {
	return &PkiSyncCertificateResource{}
}

type PkiSyncCertificateResourceModel struct {
	ID            types.String `tfsdk:"id"`
	PkiSyncID     types.String `tfsdk:"pki_sync_id"`
	CertificateID types.String `tfsdk:"certificate_id"`
}

func (r *PkiSyncCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pki_sync_certificate"
}

// ImportState imports an association using the composite ID "<pki_sync_id>:<certificate_id>".
// Read then resolves the association's own ID.
func (r *PkiSyncCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<pki_sync_id>:<certificate_id>\", got: "+req.ID,
		)
		return
	}

	if _, err := uuid.Parse(parts[0]); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected the PKI sync ID to be a valid UUID, got: "+parts[0])
		return
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected the certificate ID to be a valid UUID, got: "+parts[1])
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pki_sync_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate_id"), parts[1])...)
}

func (r *PkiSyncCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Associate a certificate with a PKI sync so it is synced to the destination. The certificate must belong to the same application as the PKI sync.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the certificate association.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"pki_sync_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the PKI sync to associate the certificate with.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"certificate_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the certificate to associate with the PKI sync.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *PkiSyncCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *PkiSyncCertificateResource) findAssociation(pkiSyncID, certificateID string) (string, error) {
	offset := 0
	for {
		page, err := r.client.ListPkiSyncCertificates(infisical.ListPkiSyncCertificatesRequest{
			PkiSyncID: pkiSyncID,
			Offset:    offset,
			Limit:     pkiSyncCertificatesPageSize,
		})
		if err != nil {
			return "", err
		}

		for _, cert := range page.Certificates {
			if cert.CertificateID == certificateID {
				return cert.ID, nil
			}
		}

		// Advance by the actual page size so a short (non-full) page mid-run never skips entries.
		offset += len(page.Certificates)
		if len(page.Certificates) == 0 || offset >= page.TotalCount {
			return "", nil
		}
	}
}

func (r *PkiSyncCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to add certificate to PKI sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan PkiSyncCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	added, err := r.client.AddPkiSyncCertificates(infisical.AddPkiSyncCertificatesRequest{
		PkiSyncID:      plan.PkiSyncID.ValueString(),
		CertificateIDs: []string{plan.CertificateID.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding certificate to PKI sync",
			"Couldn't add certificate to PKI sync, unexpected error: "+err.Error(),
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
		associationID, err = r.findAssociation(plan.PkiSyncID.ValueString(), plan.CertificateID.ValueString())
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
			"Error adding certificate to PKI sync",
			"The certificate was added but its association could not be found.",
		)
		return
	}

	plan.ID = types.StringValue(associationID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *PkiSyncCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read PKI sync certificate",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state PkiSyncCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	associationID, err := r.findAssociation(state.PkiSyncID.ValueString(), state.CertificateID.ValueString())
	if err != nil {
		if err == infisical.ErrNotFound {
			// The parent sync no longer exists.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading PKI sync certificate",
			"Couldn't read PKI sync certificate association, unexpected error: "+err.Error(),
		)
		return
	}

	if associationID == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(associationID)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PkiSyncCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PkiSyncCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *PkiSyncCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to remove certificate from PKI sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state PkiSyncCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemovePkiSyncCertificates(infisical.RemovePkiSyncCertificatesRequest{
		PkiSyncID:      state.PkiSyncID.ValueString(),
		CertificateIDs: []string{state.CertificateID.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing certificate from PKI sync",
			"Couldn't remove certificate from PKI sync, unexpected error: "+err.Error(),
		)
	}
}
