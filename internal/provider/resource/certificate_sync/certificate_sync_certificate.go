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

const deprecationMessage = "Use the certificate_filters.certificate_ids attribute on the certificate sync resource instead. " +
	"To migrate without detaching certificates, add the IDs to certificate_filters and drop this resource from state with a removed block that sets destroy = false."

// CertificateSyncCertificateResource is deprecated; it matches by certificate order to survive renewals.
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
			"The attachment follows the certificate across renewals. Deprecated: use `certificate_filters.certificate_ids` on the certificate sync resource instead.",
		DeprecationMessage: deprecationMessage,
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

// findAssociation returns the association holding the current certificate in the same certificate
// order as certificateID, or an empty string when the order is not attached.
func (r *CertificateSyncCertificateResource) findAssociation(certificateSyncID, certificateID string) (string, error) {
	orderID, err := newCertificateOrderResolver(r.client, nil).orderOf(certificateID)
	if err != nil {
		if err == infisical.ErrNotFound {
			return "", nil
		}
		return "", err
	}

	current, err := newLinkedCertificates(r.client, certificateSyncID).byOrder(orderID)
	if err != nil || current == nil {
		return "", err
	}
	return current.ID, nil
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
				"Couldn't add certificate %q to certificate sync %q. Revoked or expired certificates cannot be synced.\n\nOriginal error: %s",
				plan.CertificateID.ValueString(), plan.CertificateSyncID.ValueString(), err.Error(),
			),
		)
		return
	}

	// The add response only lists newly linked certificates, so an order that was already attached
	// comes back empty and is resolved through a lookup instead.
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

	// Warn rather than error: erroring during refresh would break `terraform plan` and `terraform destroy`.
	if associationID == "" {
		resp.Diagnostics.AddWarning(
			"Certificate is no longer attached to the certificate sync",
			fmt.Sprintf(
				"Certificate %q is no longer attached to certificate sync %q. Terraform will attach it again.",
				state.CertificateID.ValueString(), state.CertificateSyncID.ValueString(),
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
