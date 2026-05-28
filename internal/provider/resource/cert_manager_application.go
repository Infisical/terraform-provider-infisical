package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &certManagerApplicationResource{}
)

func NewCertManagerApplicationResource() resource.Resource {
	return &certManagerApplicationResource{}
}

type certManagerApplicationResource struct {
	client *infisical.Client
}

type certManagerApplicationResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *certManagerApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_application"
}

func (r *certManagerApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage Certificate Manager applications in Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Certificate Manager application",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Certificate Manager application",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the Certificate Manager application",
				Optional:    true,
			},
		},
	}
}

func (r *certManagerApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create Certificate Manager application",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := infisical.CreatePkiApplicationRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		createRequest.Description = &description
	}

	application, err := r.client.CreatePkiApplication(createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Certificate Manager application",
			"Couldn't create application in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(application.Application.Id)
	plan.Name = types.StringValue(application.Application.Name)
	if application.Application.Description != nil {
		plan.Description = types.StringValue(*application.Application.Description)
	} else {
		plan.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read Certificate Manager application",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	application, err := r.client.GetPkiApplication(infisical.GetPkiApplicationRequest{
		ApplicationId: state.Id.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading Certificate Manager application",
			"Couldn't read application from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Id = types.StringValue(application.Application.Id)
	state.Name = types.StringValue(application.Application.Name)
	if application.Application.Description != nil {
		state.Description = types.StringValue(*application.Application.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update Certificate Manager application",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := infisical.UpdatePkiApplicationRequest{
		ApplicationId: plan.Id.ValueString(),
		Name:          plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		updateRequest.Description = &description
	}

	application, err := r.client.UpdatePkiApplication(updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Certificate Manager application",
			"Couldn't update application in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(application.Application.Id)
	plan.Name = types.StringValue(application.Application.Name)
	if application.Application.Description != nil {
		plan.Description = types.StringValue(*application.Application.Description)
	} else {
		plan.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete Certificate Manager application",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeletePkiApplication(infisical.DeletePkiApplicationRequest{
		ApplicationId: state.Id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Certificate Manager application",
			"Couldn't delete application from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
