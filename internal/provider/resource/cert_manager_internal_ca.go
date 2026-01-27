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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	CA_TYPE_ROOT         = "root"
	CA_TYPE_INTERMEDIATE = "intermediate"

	DEFAULT_CA_KEY_ALGORITHM = "RSA_2048"
	DEFAULT_CA_STATUS        = "active"
)

var (
	SUPPORTED_CA_TYPES               = []string{CA_TYPE_ROOT, CA_TYPE_INTERMEDIATE}
	SUPPORTED_ROOT_CA_KEY_ALGORITHMS = []string{"RSA_2048", "RSA_3072", "RSA_4096", "EC_prime256v1", "EC_secp384r1", "EC_secp521r1"}
	SUPPORTED_CA_STATUSES            = []string{"active", "disabled"}
)

var (
	_ resource.Resource = &certManagerInternalCAResource{}
)

func NewCertManagerInternalCAResource() resource.Resource {
	return &certManagerInternalCAResource{}
}

type certManagerInternalCAResource struct {
	client *infisical.Client
}

type certManagerInternalCAResourceModel struct {
	ProjectSlug  types.String `tfsdk:"project_slug"`
	Id           types.String `tfsdk:"id"`
	Type         types.String `tfsdk:"type"`
	Name         types.String `tfsdk:"name"`
	CommonName   types.String `tfsdk:"common_name"`
	Organization types.String `tfsdk:"organization"`
	OU           types.String `tfsdk:"ou"`
	Country      types.String `tfsdk:"country"`
	Province     types.String `tfsdk:"province"`
	Locality     types.String `tfsdk:"locality"`
	KeyAlgorithm types.String `tfsdk:"key_algorithm"`
	Status       types.String `tfsdk:"status"`
}

func (r *certManagerInternalCAResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_internal_ca"
}

func (r *certManagerInternalCAResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage internal certificate authorities (root or intermediate) in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"project_slug": schema.StringAttribute{
				Description: "The slug of the cert-manager project",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of the CA. Supported values: " + strings.Join(SUPPORTED_CA_TYPES, ", "),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_CA_TYPES...),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the CA",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"common_name": schema.StringAttribute{
				Description: "The common name (CN) of the CA certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "The organization (O) of the CA certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ou": schema.StringAttribute{
				Description: "The organizational unit (OU) of the CA certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"country": schema.StringAttribute{
				Description: "The country (C) of the CA certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 2),
				},
			},
			"province": schema.StringAttribute{
				Description: "The state/province (ST) of the CA certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locality": schema.StringAttribute{
				Description: "The locality (L) of the CA certificate",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_algorithm": schema.StringAttribute{
				Description: "The key algorithm for the CA. Supported values: " + strings.Join(SUPPORTED_ROOT_CA_KEY_ALGORITHMS, ", ") + ". Defaults to '" + DEFAULT_CA_KEY_ALGORITHM + "'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(DEFAULT_CA_KEY_ALGORITHM),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_ROOT_CA_KEY_ALGORITHMS...),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the CA. Supported values: " + strings.Join(SUPPORTED_CA_STATUSES, ", ") + ". Defaults to '" + DEFAULT_CA_STATUS + "'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(DEFAULT_CA_STATUS),
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_CA_STATUSES...),
				},
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the CA",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *certManagerInternalCAResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerInternalCAResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerInternalCAResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that at least one subject field is provided
	subjectFields := []types.String{
		plan.CommonName, plan.Organization, plan.OU,
		plan.Country, plan.Province, plan.Locality,
	}
	hasSubjectField := false
	for _, field := range subjectFields {
		if !field.IsNull() && !field.IsUnknown() && field.ValueString() != "" {
			hasSubjectField = true
			break
		}
	}
	if !hasSubjectField {
		resp.Diagnostics.AddError(
			"Missing subject fields",
			"At least one of the fields common_name, organization, ou, country, province, or locality must be provided",
		)
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

	newCA, err := r.client.CreateInternalCA(infisical.CreateInternalCARequest{
		ProjectId: project.ID,
		Name:      plan.Name.ValueString(),
		Status:    plan.Status.ValueString(),
		Configuration: infisical.CertificateAuthorityConfiguration{
			Type:         plan.Type.ValueString(),
			CommonName:   plan.CommonName.ValueString(),
			Organization: plan.Organization.ValueString(),
			OU:           plan.OU.ValueString(),
			Country:      plan.Country.ValueString(),
			Province:     plan.Province.ValueString(),
			Locality:     plan.Locality.ValueString(),
			KeyAlgorithm: plan.KeyAlgorithm.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CA",
			"Couldn't create CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(newCA.Id)
	plan.Status = types.StringValue(newCA.Status)

	if newCA.Configuration.CommonName != "" {
		plan.CommonName = types.StringValue(newCA.Configuration.CommonName)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerInternalCAResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerInternalCAResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ca, err := r.client.GetInternalCA(infisical.GetCARequest{
		CAId: state.Id.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading CA",
			"Couldn't read CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(ca.Name)
	status := ca.Status
	if status == "pending-certificate" {
		status = "active"
	}
	state.Status = types.StringValue(status)

	if ca.Configuration.Type != "" {
		state.Type = types.StringValue(ca.Configuration.Type)
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

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerInternalCAResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerInternalCAResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerInternalCAResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that at least one subject field is provided
	subjectFields := []types.String{
		plan.CommonName, plan.Organization, plan.OU,
		plan.Country, plan.Province, plan.Locality,
	}
	hasSubjectField := false
	for _, field := range subjectFields {
		if !field.IsNull() && !field.IsUnknown() && field.ValueString() != "" {
			hasSubjectField = true
			break
		}
	}
	if !hasSubjectField {
		resp.Diagnostics.AddError(
			"Missing subject fields",
			"At least one of the fields common_name, organization, ou, country, province, or locality must be provided",
		)
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

	_, err = r.client.UpdateInternalCA(infisical.UpdateInternalCARequest{
		ProjectId: project.ID,
		CAId:      plan.Id.ValueString(),
		Name:      plan.Name.ValueString(),
		Status:    plan.Status.ValueString(),
		Configuration: infisical.CertificateAuthorityConfiguration{
			Type:         plan.Type.ValueString(),
			CommonName:   plan.CommonName.ValueString(),
			Organization: plan.Organization.ValueString(),
			OU:           plan.OU.ValueString(),
			Country:      plan.Country.ValueString(),
			Province:     plan.Province.ValueString(),
			Locality:     plan.Locality.ValueString(),
			KeyAlgorithm: plan.KeyAlgorithm.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating CA",
			"Couldn't update CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerInternalCAResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerInternalCAResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteInternalCA(infisical.DeleteCARequest{
		CAId: state.Id.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting CA",
			"Couldn't delete CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerInternalCAResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
