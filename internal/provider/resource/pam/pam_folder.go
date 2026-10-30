package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

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
	_ resource.Resource                = &pamFolderResource{}
	_ resource.ResourceWithConfigure   = &pamFolderResource{}
	_ resource.ResourceWithImportState = &pamFolderResource{}
)

func NewPamFolderResource() resource.Resource {
	return &pamFolderResource{}
}

type pamFolderResource struct {
	client *infisical.Client
}

type pamFolderResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *pamFolderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pam_folder"
}

func (r *pamFolderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage folders in Infisical PAM. The machine identity that creates a folder becomes its admin. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the folder.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the folder, unique in the organization. " + pamNameDescription,
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					infisicaltf.SlugRegexValidator,
				},
			},
			"description": schema.StringAttribute{
				Description: "An optional description of the folder.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
		},
	}
}

func (r *pamFolderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configurePamClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *pamFolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to create PAM folder") {
		return
	}

	var plan pamFolderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	folder, err := r.client.CreatePamFolder(infisical.CreatePamFolderRequest{
		Name:        plan.Name.ValueString(),
		Description: optionalString(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating PAM folder", "Couldn't create PAM folder, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(folder.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pamFolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to read PAM folder") {
		return
	}

	var state pamFolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	folder, err := r.client.GetPamFolder(infisical.GetPamFolderRequest{ID: state.ID.ValueString()})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PAM folder", "Couldn't read PAM folder, unexpected error: "+err.Error())
		return
	}

	state.Name = types.StringValue(folder.Name)
	state.Description = stringOrNull(folder.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *pamFolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to update PAM folder") {
		return
	}

	var plan, state pamFolderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdatePamFolder(infisical.UpdatePamFolderRequest{
		ID:          state.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: optionalString(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating PAM folder", "Couldn't update PAM folder, unexpected error: "+err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pamFolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to delete PAM folder") {
		return
	}

	var state pamFolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePamFolder(infisical.DeletePamFolderRequest{ID: state.ID.ValueString()})
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError("Error deleting PAM folder", "Couldn't delete PAM folder, unexpected error: "+err.Error())
	}
}

func (r *pamFolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
