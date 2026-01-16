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
	_ resource.Resource = &certManagerExternalCAADCSResource{}
)

func NewCertManagerExternalCAADCSResource() resource.Resource {
	return &certManagerExternalCAADCSResource{}
}

type certManagerExternalCAADCSResource struct {
	client *infisical.Client
}

type certManagerExternalCAADCSResourceModel struct {
	ProjectSlug           types.String `tfsdk:"project_slug"`
	Id                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Status                types.String `tfsdk:"status"`
	AzureAdcsConnectionId types.String `tfsdk:"azure_adcs_connection_id"`
}

func (r *certManagerExternalCAADCSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_external_ca_adcs"
}

func (r *certManagerExternalCAADCSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage external ADCS (Microsoft Active Directory Certificate Services) certificate authorities in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"project_slug": schema.StringAttribute{
				Description: "The slug of the cert-manager project",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the ADCS CA",
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
			"azure_adcs_connection_id": schema.StringAttribute{
				Description: "The ID of the Azure ADCS app connection for certificate issuance",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the ADCS CA",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *certManagerExternalCAADCSResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerExternalCAADCSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create ADCS CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerExternalCAADCSResourceModel
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

	newCA, err := r.client.CreateADCSCA(infisical.CreateADCSCARequest{
		ProjectId: project.ID,
		Name:      plan.Name.ValueString(),
		Status:    status,
		Configuration: infisical.CertificateAuthorityConfiguration{
			AzureAdcsConnectionId: plan.AzureAdcsConnectionId.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ADCS CA",
			"Couldn't create ADCS CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(newCA.Id)
	plan.Name = types.StringValue(newCA.Name)
	plan.Status = types.StringValue(newCA.Status)

	if newCA.Configuration.AzureAdcsConnectionId != "" {
		plan.AzureAdcsConnectionId = types.StringValue(newCA.Configuration.AzureAdcsConnectionId)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCAADCSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read ADCS CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerExternalCAADCSResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ca, err := r.client.GetADCSCA(infisical.GetCARequest{
		CAId: state.Id.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading ADCS CA",
			"Couldn't read ADCS CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(ca.Name)
	state.Status = types.StringValue(ca.Status)

	if ca.Configuration.AzureAdcsConnectionId != "" {
		state.AzureAdcsConnectionId = types.StringValue(ca.Configuration.AzureAdcsConnectionId)
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCAADCSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update ADCS CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerExternalCAADCSResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerExternalCAADCSResourceModel
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

	updatedCA, err := r.client.UpdateADCSCA(infisical.UpdateADCSCARequest{
		ProjectId: project.ID,
		CAId:      plan.Id.ValueString(),
		Name:      plan.Name.ValueString(),
		Status:    plan.Status.ValueString(),
		Configuration: infisical.CertificateAuthorityConfiguration{
			AzureAdcsConnectionId: plan.AzureAdcsConnectionId.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating ADCS CA",
			"Couldn't update ADCS CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(updatedCA.Id)
	plan.Name = types.StringValue(updatedCA.Name)
	plan.Status = types.StringValue(updatedCA.Status)

	if updatedCA.Configuration.AzureAdcsConnectionId != "" {
		plan.AzureAdcsConnectionId = types.StringValue(updatedCA.Configuration.AzureAdcsConnectionId)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certManagerExternalCAADCSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete ADCS CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerExternalCAADCSResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteADCSCA(infisical.DeleteCARequest{
		CAId: state.Id.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ADCS CA",
			"Couldn't delete ADCS CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerExternalCAADCSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
