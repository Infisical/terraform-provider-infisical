package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &certManagerCACertificateResource{}
)

func NewCertManagerCACertificateResource() resource.Resource {
	return &certManagerCACertificateResource{}
}

type certManagerCACertificateResource struct {
	client *infisical.Client
}

type certManagerCACertificateResourceModel struct {
	CaId             types.String `tfsdk:"ca_id"`
	Id               types.String `tfsdk:"id"`
	ParentCaId       types.String `tfsdk:"parent_ca_id"`
	NotBefore        types.String `tfsdk:"not_before"`
	NotAfter         types.String `tfsdk:"not_after"`
	MaxPathLength    types.Int64  `tfsdk:"max_path_length"`
	Certificate      types.String `tfsdk:"certificate"`
	CertificateChain types.String `tfsdk:"certificate_chain"`
	SerialNumber     types.String `tfsdk:"serial_number"`
}

func (r *certManagerCACertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_ca_certificate"
}

func (r *certManagerCACertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage CA certificates in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"ca_id": schema.StringAttribute{
				Description: "The ID of the certificate authority to generate a certificate for",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier for this CA certificate resource",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_ca_id": schema.StringAttribute{
				Description: "The ID of the parent CA (required for intermediate CAs)",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"not_before": schema.StringAttribute{
				Description: "The date and time when the CA becomes valid in RFC3339 format (e.g., '2024-01-01T00:00:00Z')",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"not_after": schema.StringAttribute{
				Description: "The date and time when the CA expires in RFC3339 format (e.g., '2034-01-01T00:00:00Z')",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"max_path_length": schema.Int64Attribute{
				Description: "The maximum number of intermediate CAs that may follow this CA in the certificate chain. Use -1 for no path limit",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(-1),
				Validators: []validator.Int64{
					int64validator.AtLeast(-1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"certificate": schema.StringAttribute{
				Description: "The generated CA certificate in PEM format",
				Computed:    true,
				Sensitive:   true,
			},
			"certificate_chain": schema.StringAttribute{
				Description: "The certificate chain of the CA in PEM format",
				Computed:    true,
				Sensitive:   true,
			},
			"serial_number": schema.StringAttribute{
				Description: "The serial number of the CA certificate",
				Computed:    true,
			},
		},
	}
}

func (r *certManagerCACertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerCACertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create CA certificate",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerCACertificateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	notBefore, err := time.Parse(time.RFC3339, plan.NotBefore.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid not_before format", "not_before must be in RFC3339 format (e.g., '2024-01-01T00:00:00Z')")
		return
	}

	notAfter, err := time.Parse(time.RFC3339, plan.NotAfter.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid not_after format", "not_after must be in RFC3339 format (e.g., '2034-01-01T00:00:00Z')")
		return
	}

	if notAfter.Before(notBefore) {
		resp.Diagnostics.AddError("Invalid date range", "not_after must be after not_before")
		return
	}

	generateRequest := infisical.GenerateCACertificateRequest{
		CaId:      plan.CaId.ValueString(),
		NotBefore: plan.NotBefore.ValueString(),
		NotAfter:  plan.NotAfter.ValueString(),
	}

	if !plan.MaxPathLength.IsNull() {
		maxPathLength := int(plan.MaxPathLength.ValueInt64())
		generateRequest.MaxPathLength = &maxPathLength
	}

	if !plan.ParentCaId.IsNull() && plan.ParentCaId.ValueString() != "" {
		generateRequest.ParentCaId = plan.ParentCaId.ValueString()
	}

	certificate, err := r.client.GenerateCACertificate(generateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error generating CA certificate", err.Error())
		return
	}

	plan.Id = types.StringValue(fmt.Sprintf("%s:%s", plan.CaId.ValueString(), certificate.SerialNumber))
	plan.Certificate = types.StringValue(certificate.Certificate)
	plan.CertificateChain = types.StringValue(certificate.CertificateChain)
	plan.SerialNumber = types.StringValue(certificate.SerialNumber)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerCACertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read CA certificate",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerCACertificateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificate, err := r.client.GetCACertificate(infisical.GetCACertificateRequest{
		CaId: state.CaId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddWarning(
			"CA certificate not found",
			"The CA certificate appears to have been manually deleted from Infisical. Please remove this resource from your Terraform configuration.",
		)
		resp.State.RemoveResource(ctx)
		return
	}

	if certificate.SerialNumber != state.SerialNumber.ValueString() {
		resp.Diagnostics.AddWarning(
			"CA certificate has changed",
			"The CA certificate appears to have been regenerated. The current certificate has a different serial number.",
		)
	}

	state.Certificate = types.StringValue(certificate.Certificate)
	state.CertificateChain = types.StringValue(certificate.CertificateChain)
	state.SerialNumber = types.StringValue(certificate.SerialNumber)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerCACertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"CA certificates cannot be updated. To change certificate properties, you must replace the resource (which will regenerate the certificate).",
	)
}

func (r *certManagerCACertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state certManagerCACertificateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetCACertificate(infisical.GetCACertificateRequest{
		CaId: state.CaId.ValueString(),
	})

	if err != nil {
		return
	}

	resp.Diagnostics.AddError(
		"Cannot delete CA certificate",
		"CA certificates cannot be directly deleted. If you need to remove the certificate, you must first delete the certificate authority",
	)
}

func (r *certManagerCACertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) == 1 {
		caId := parts[0]

		certificate, err := r.client.GetCACertificate(infisical.GetCACertificateRequest{
			CaId: caId,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error importing CA certificate", err.Error())
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ca_id"), caId)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s:%s", caId, certificate.SerialNumber))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate"), certificate.Certificate)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate_chain"), certificate.CertificateChain)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("serial_number"), certificate.SerialNumber)...)

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("not_before"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("not_after"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("max_path_length"), types.Int64Unknown())...)

	} else if len(parts) == 2 {
		caId := parts[0]
		serialNumber := parts[1]

		certificate, err := r.client.GetCACertificate(infisical.GetCACertificateRequest{
			CaId: caId,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error importing CA certificate", err.Error())
			return
		}

		if certificate.SerialNumber != serialNumber {
			resp.Diagnostics.AddError(
				"Certificate serial number mismatch",
				fmt.Sprintf("Expected serial number %s, but current certificate has serial number %s", serialNumber, certificate.SerialNumber),
			)
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ca_id"), caId)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate"), certificate.Certificate)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate_chain"), certificate.CertificateChain)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("serial_number"), certificate.SerialNumber)...)

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("not_before"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("not_after"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("max_path_length"), types.Int64Unknown())...)

	} else {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			"Import ID must be either 'ca_id' or 'ca_id:serial_number'",
		)
	}
}
