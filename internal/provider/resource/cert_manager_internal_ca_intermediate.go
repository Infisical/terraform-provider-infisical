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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &certManagerInternalCAIntermediateResource{}
)

func NewCertManagerInternalCAIntermediateResource() resource.Resource {
	return &certManagerInternalCAIntermediateResource{}
}

type certManagerInternalCAIntermediateResource struct {
	client *infisical.Client
}

type certManagerInternalCAIntermediateResourceModel struct {
	ProjectSlug   types.String `tfsdk:"project_slug"`
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ParentCAId    types.String `tfsdk:"parent_ca_id"`
	FriendlyName  types.String `tfsdk:"friendly_name"`
	CommonName    types.String `tfsdk:"common_name"`
	Organization  types.String `tfsdk:"organization"`
	OU            types.String `tfsdk:"ou"`
	Country       types.String `tfsdk:"country"`
	Province      types.String `tfsdk:"province"`
	Locality      types.String `tfsdk:"locality"`
	KeyAlgorithm  types.String `tfsdk:"key_algorithm"`
	MaxPathLength types.Int64  `tfsdk:"max_path_length"`
	Status        types.String `tfsdk:"status"`
}

func (r *certManagerInternalCAIntermediateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_internal_ca_intermediate"
}

func (r *certManagerInternalCAIntermediateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage internal intermediate certificate authorities in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"project_slug": schema.StringAttribute{
				Description: "The slug of the cert-manager project",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the intermediate CA",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_ca_id": schema.StringAttribute{
				Description: "The ID of the parent CA (root or intermediate CA)",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"friendly_name": schema.StringAttribute{
				Description: "The friendly display name of the intermediate CA",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"common_name": schema.StringAttribute{
				Description: "The common name (CN) of the intermediate CA certificate",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "The organization (O) of the intermediate CA certificate",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ou": schema.StringAttribute{
				Description: "The organizational unit (OU) of the intermediate CA certificate",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"country": schema.StringAttribute{
				Description: "The country (C) of the intermediate CA certificate",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 2),
				},
			},
			"province": schema.StringAttribute{
				Description: "The state/province (ST) of the intermediate CA certificate",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locality": schema.StringAttribute{
				Description: "The locality (L) of the intermediate CA certificate",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_algorithm": schema.StringAttribute{
				Description: "The key algorithm for the intermediate CA. Supported values: " + strings.Join(SUPPORTED_ROOT_CA_KEY_ALGORITHMS, ", "),
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_ROOT_CA_KEY_ALGORITHMS...),
				},
			},
			"max_path_length": schema.Int64Attribute{
				Description: "The maximum path length for certificate chains issued by this CA",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
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
			"id": schema.StringAttribute{
				Description:   "The ID of the intermediate CA",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *certManagerInternalCAIntermediateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerInternalCAIntermediateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create intermediate CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerInternalCAIntermediateResourceModel
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

	keyAlgorithm := "RSA_2048"
	if !plan.KeyAlgorithm.IsNull() && !plan.KeyAlgorithm.IsUnknown() {
		keyAlgorithm = plan.KeyAlgorithm.ValueString()
	}

	status := "active"
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		status = plan.Status.ValueString()
	}

	var maxPathLength *int
	if !plan.MaxPathLength.IsNull() && !plan.MaxPathLength.IsUnknown() {
		val := int(plan.MaxPathLength.ValueInt64())
		maxPathLength = &val
	}

	newCA, err := r.client.CreateInternalCA(infisical.CreateInternalCARequest{
		ProjectId: project.ID,
		Name:      plan.Name.ValueString(),
		Status:    status,
		Configuration: infisical.CertificateAuthorityConfiguration{
			Type:          "intermediate",
			FriendlyName:  plan.FriendlyName.ValueString(),
			CommonName:    plan.CommonName.ValueString(),
			Organization:  plan.Organization.ValueString(),
			OU:            plan.OU.ValueString(),
			Country:       plan.Country.ValueString(),
			Province:      plan.Province.ValueString(),
			Locality:      plan.Locality.ValueString(),
			KeyAlgorithm:  keyAlgorithm,
			MaxPathLength: maxPathLength,
			ParentCaId:    plan.ParentCAId.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating intermediate CA",
			"Couldn't create intermediate CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(newCA.Id)
	plan.KeyAlgorithm = types.StringValue(keyAlgorithm)
	plan.Status = types.StringValue(newCA.Status)
	if maxPathLength != nil {
		plan.MaxPathLength = types.Int64Value(int64(*maxPathLength))
	} else {
		plan.MaxPathLength = types.Int64Null()
	}

	if newCA.Configuration.FriendlyName != "" {
		plan.FriendlyName = types.StringValue(newCA.Configuration.FriendlyName)
	}
	if newCA.Configuration.CommonName != "" {
		plan.CommonName = types.StringValue(newCA.Configuration.CommonName)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerInternalCAIntermediateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read intermediate CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerInternalCAIntermediateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.GetProject(infisical.GetProjectRequest{
		Slug: state.ProjectSlug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	ca, err := r.client.GetCA(infisical.GetCARequest{
		ProjectId: project.ID,
		CAId:      state.Id.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading intermediate CA",
			"Couldn't read intermediate CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(ca.Name)
	status := ca.Status
	if status == "pending-certificate" {
		status = "active"
	}
	state.Status = types.StringValue(status)

	if ca.Configuration.FriendlyName != "" {
		state.FriendlyName = types.StringValue(ca.Configuration.FriendlyName)
	}
	if ca.Configuration.CommonName != "" {
		state.CommonName = types.StringValue(ca.Configuration.CommonName)
	}
	if ca.Configuration.Organization != "" {
		state.Organization = types.StringValue(ca.Configuration.Organization)
	}
	if ca.Configuration.OU != "" {
		state.OU = types.StringValue(ca.Configuration.OU)
	}
	if ca.Configuration.Country != "" {
		state.Country = types.StringValue(ca.Configuration.Country)
	}
	if ca.Configuration.Province != "" {
		state.Province = types.StringValue(ca.Configuration.Province)
	}
	if ca.Configuration.Locality != "" {
		state.Locality = types.StringValue(ca.Configuration.Locality)
	}
	if ca.Configuration.KeyAlgorithm != "" {
		state.KeyAlgorithm = types.StringValue(ca.Configuration.KeyAlgorithm)
	}
	if ca.Configuration.ParentCaId != "" {
		state.ParentCAId = types.StringValue(ca.Configuration.ParentCaId)
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerInternalCAIntermediateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update intermediate CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerInternalCAIntermediateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerInternalCAIntermediateResourceModel
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

	var maxPathLength *int
	if !plan.MaxPathLength.IsNull() && !plan.MaxPathLength.IsUnknown() {
		val := int(plan.MaxPathLength.ValueInt64())
		maxPathLength = &val
	}

	_, err = r.client.UpdateInternalCA(infisical.UpdateInternalCARequest{
		ProjectId: project.ID,
		CAId:      plan.Id.ValueString(),
		Name:      plan.Name.ValueString(),
		Status:    plan.Status.ValueString(),
		Configuration: infisical.CertificateAuthorityConfiguration{
			Type:          "intermediate",
			FriendlyName:  plan.FriendlyName.ValueString(),
			CommonName:    plan.CommonName.ValueString(),
			Organization:  plan.Organization.ValueString(),
			OU:            plan.OU.ValueString(),
			Country:       plan.Country.ValueString(),
			Province:      plan.Province.ValueString(),
			Locality:      plan.Locality.ValueString(),
			KeyAlgorithm:  plan.KeyAlgorithm.ValueString(),
			MaxPathLength: maxPathLength,
			ParentCaId:    plan.ParentCAId.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating intermediate CA",
			"Couldn't update intermediate CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerInternalCAIntermediateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete intermediate CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerInternalCAIntermediateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.GetProject(infisical.GetProjectRequest{
		Slug: state.ProjectSlug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	_, err = r.client.DeleteInternalCA(infisical.DeleteCARequest{
		ProjectId: project.ID,
		CAId:      state.Id.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting intermediate CA",
			"Couldn't delete intermediate CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerInternalCAIntermediateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
