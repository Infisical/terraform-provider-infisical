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

var (
	SUPPORTED_DNS_PROVIDERS = []string{"route53", "cloudflare", "dns-made-easy"}
)

var (
	_ resource.Resource = &certManagerExternalCAACMEResource{}
)

func NewCertManagerExternalCAACMEResource() resource.Resource {
	return &certManagerExternalCAACMEResource{}
}

type certManagerExternalCAACMEResource struct {
	client *infisical.Client
}

type certManagerExternalCAACMEResourceModel struct {
	ProjectSlug        types.String `tfsdk:"project_slug"`
	Id                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Status             types.String `tfsdk:"status"`
	DNSAppConnectionId types.String `tfsdk:"dns_app_connection_id"`
	DNSProvider        types.String `tfsdk:"dns_provider"`
	DNSHostedZoneId    types.String `tfsdk:"dns_hosted_zone_id"`
	DirectoryUrl       types.String `tfsdk:"directory_url"`
	AccountEmail       types.String `tfsdk:"account_email"`
	EABKid             types.String `tfsdk:"eab_kid"`
	EABHmacKey         types.String `tfsdk:"eab_hmac_key"`
}

func (r *certManagerExternalCAACMEResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_external_ca_acme"
}

func (r *certManagerExternalCAACMEResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage external ACME certificate authorities in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"project_slug": schema.StringAttribute{
				Description: "The slug of the cert-manager project",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the ACME CA",
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
			"dns_app_connection_id": schema.StringAttribute{
				Description: "The ID of the DNS app connection for ACME challenge validation",
				Required:    true,
			},
			"dns_provider": schema.StringAttribute{
				Description: "The DNS provider for ACME challenge validation. Supported values: " + strings.Join(SUPPORTED_DNS_PROVIDERS, ", "),
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_DNS_PROVIDERS...),
				},
			},
			"dns_hosted_zone_id": schema.StringAttribute{
				Description: "The hosted zone ID for DNS-01 challenge validation",
				Optional:    true,
			},
			"directory_url": schema.StringAttribute{
				Description: "The ACME directory URL",
				Required:    true,
			},
			"account_email": schema.StringAttribute{
				Description: "The email address for ACME account registration",
				Required:    true,
			},
			"eab_kid": schema.StringAttribute{
				Description: "External Account Binding (EAB) Key ID (optional)",
				Optional:    true,
			},
			"eab_hmac_key": schema.StringAttribute{
				Description: "External Account Binding (EAB) HMAC key (optional)",
				Optional:    true,
				Sensitive:   true,
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the ACME CA",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *certManagerExternalCAACMEResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerExternalCAACMEResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create ACME CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerExternalCAACMEResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.GetProject(infisical.GetProjectRequest{
		Slug: plan.ProjectSlug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	status := "active"
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		status = plan.Status.ValueString()
	}

	dnsProviderConfig := make(map[string]interface{})
	dnsProviderConfig["provider"] = plan.DNSProvider.ValueString()
	if !plan.DNSHostedZoneId.IsNull() && !plan.DNSHostedZoneId.IsUnknown() {
		dnsProviderConfig["hostedZoneId"] = plan.DNSHostedZoneId.ValueString()
	}

	configuration := infisical.CertificateAuthorityConfiguration{
		DNSAppConnectionId: plan.DNSAppConnectionId.ValueString(),
		DNSProviderConfig:  dnsProviderConfig,
		DirectoryUrl:       plan.DirectoryUrl.ValueString(),
		AccountEmail:       plan.AccountEmail.ValueString(),
	}

	if !plan.EABKid.IsNull() && !plan.EABKid.IsUnknown() {
		configuration.EABKid = plan.EABKid.ValueString()
	}

	if !plan.EABHmacKey.IsNull() && !plan.EABHmacKey.IsUnknown() {
		configuration.EABHmacKey = plan.EABHmacKey.ValueString()
	}

	newCA, err := r.client.CreateACMECA(infisical.CreateACMECARequest{
		ProjectId:     project.ID,
		Name:          plan.Name.ValueString(),
		Status:        status,
		Configuration: configuration,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ACME CA",
			"Couldn't create ACME CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(newCA.Id)
	plan.Status = types.StringValue(newCA.Status)

	if newCA.Configuration.DNSAppConnectionId != "" {
		plan.DNSAppConnectionId = types.StringValue(newCA.Configuration.DNSAppConnectionId)
	}
	if newCA.Configuration.DirectoryUrl != "" {
		plan.DirectoryUrl = types.StringValue(newCA.Configuration.DirectoryUrl)
	}
	if newCA.Configuration.AccountEmail != "" {
		plan.AccountEmail = types.StringValue(newCA.Configuration.AccountEmail)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCAACMEResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read ACME CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerExternalCAACMEResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}


	ca, err := r.client.GetACMECA(infisical.GetCARequest{
		CAId: state.Id.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading ACME CA",
			"Couldn't read ACME CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(ca.Name)
	state.Status = types.StringValue(ca.Status)

	if ca.Configuration.DNSAppConnectionId != "" {
		state.DNSAppConnectionId = types.StringValue(ca.Configuration.DNSAppConnectionId)
	}
	if ca.Configuration.DirectoryUrl != "" {
		state.DirectoryUrl = types.StringValue(ca.Configuration.DirectoryUrl)
	}
	if ca.Configuration.AccountEmail != "" {
		state.AccountEmail = types.StringValue(ca.Configuration.AccountEmail)
	}

	if ca.Configuration.DNSProviderConfig != nil {
		if provider, ok := ca.Configuration.DNSProviderConfig["provider"].(string); ok && provider != "" {
			state.DNSProvider = types.StringValue(provider)
		}
		if hostedZoneId, ok := ca.Configuration.DNSProviderConfig["hostedZoneId"].(string); ok && hostedZoneId != "" {
			state.DNSHostedZoneId = types.StringValue(hostedZoneId)
		}
	}

	if ca.Configuration.EABKid != "" {
		state.EABKid = types.StringValue(ca.Configuration.EABKid)
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCAACMEResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update ACME CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerExternalCAACMEResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerExternalCAACMEResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.GetProject(infisical.GetProjectRequest{
		Slug: plan.ProjectSlug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	dnsProviderConfig := make(map[string]interface{})
	dnsProviderConfig["provider"] = plan.DNSProvider.ValueString()
	if !plan.DNSHostedZoneId.IsNull() && !plan.DNSHostedZoneId.IsUnknown() {
		dnsProviderConfig["hostedZoneId"] = plan.DNSHostedZoneId.ValueString()
	}

	configuration := infisical.CertificateAuthorityConfiguration{
		DNSAppConnectionId: plan.DNSAppConnectionId.ValueString(),
		DNSProviderConfig:  dnsProviderConfig,
		DirectoryUrl:       plan.DirectoryUrl.ValueString(),
		AccountEmail:       plan.AccountEmail.ValueString(),
	}

	if !plan.EABKid.IsNull() && !plan.EABKid.IsUnknown() {
		configuration.EABKid = plan.EABKid.ValueString()
	}

	if !plan.EABHmacKey.IsNull() && !plan.EABHmacKey.IsUnknown() {
		configuration.EABHmacKey = plan.EABHmacKey.ValueString()
	}

	_, err = r.client.UpdateACMECA(infisical.UpdateACMECARequest{
		ProjectId:     project.ID,
		CAId:          plan.Id.ValueString(),
		Name:          plan.Name.ValueString(),
		Status:        plan.Status.ValueString(),
		Configuration: configuration,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating ACME CA",
			"Couldn't update ACME CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCAACMEResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete ACME CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerExternalCAACMEResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}


	_, err := r.client.DeleteACMECA(infisical.DeleteCARequest{
		CAId: state.Id.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ACME CA",
			"Couldn't delete ACME CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerExternalCAACMEResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'project_slug:ca_id'",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
