package resource

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &subOrganizationResource{}
	_ resource.ResourceWithImportState = &subOrganizationResource{}
)

// NewSubOrganizationResource is a helper function to simplify the provider implementation.
func NewSubOrganizationResource() resource.Resource {
	return &subOrganizationResource{}
}

// subOrganizationResource is the resource implementation.
type subOrganizationResource struct {
	client *infisical.Client
}

// subOrganizationResourceModel describes the resource data model.
type subOrganizationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	ParentOrgID types.String `tfsdk:"parent_org_id"`
}

// Metadata returns the resource type name.
func (r *subOrganizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sub_organization"
}

// Schema defines the schema for the resource.
func (r *subOrganizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage sub-organizations in Infisical. Requires an Infisical Enterprise plan, and only Machine Identity authentication scoped to the root organization is supported. The authenticating identity needs the `sub-organization` create permission and must not be scoped to a sub-organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the sub-organization.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the sub-organization.",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description:   "The slug of the sub-organization. If omitted, Infisical generates one from the name. Changing this updates the sub-organization.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					infisicaltf.SlugRegexValidator,
				},
			},
			"parent_org_id": schema.StringAttribute{
				Description:   "The ID of the parent (root) organization.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *subOrganizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *subOrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create sub-organization",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan subOrganizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := infisical.CreateSubOrganizationRequest{
		Name: plan.Name.ValueString(),
	}
	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		createRequest.Slug = plan.Slug.ValueString()
	}

	newSubOrg, err := r.client.CreateSubOrganization(createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating sub-organization",
			"Couldn't create sub-organization in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newSubOrg.Organization.ID)
	plan.Name = types.StringValue(newSubOrg.Organization.Name)
	plan.Slug = types.StringValue(newSubOrg.Organization.Slug)
	plan.ParentOrgID = types.StringValue(newSubOrg.Organization.ParentOrgID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *subOrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read sub-organization",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state subOrganizationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subOrg, err := r.client.GetSubOrganizationById(state.ID.ValueString())
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading sub-organization",
			"Couldn't read sub-organization from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(subOrg.ID)
	state.Name = types.StringValue(subOrg.Name)
	state.Slug = types.StringValue(subOrg.Slug)
	state.ParentOrgID = types.StringValue(subOrg.ParentOrgID)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *subOrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update sub-organization",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan subOrganizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state subOrganizationResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedSubOrg, err := r.client.UpdateSubOrganization(infisical.UpdateSubOrganizationRequest{
		SubOrgID: state.ID.ValueString(),
		Name:     plan.Name.ValueString(),
		Slug:     plan.Slug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating sub-organization",
			"Couldn't update sub-organization in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(updatedSubOrg.Organization.ID)
	plan.Name = types.StringValue(updatedSubOrg.Organization.Name)
	plan.Slug = types.StringValue(updatedSubOrg.Organization.Slug)
	plan.ParentOrgID = types.StringValue(updatedSubOrg.Organization.ParentOrgID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *subOrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete sub-organization",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state subOrganizationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSubOrganization(infisical.DeleteSubOrganizationRequest{
		SubOrgID: state.ID.ValueString(),
	})
	if err != nil {
		_, getErr := r.client.GetSubOrganizationById(state.ID.ValueString())
		if getErr == infisical.ErrNotFound {
			return
		}

		errMsg := "Couldn't delete sub-organization from Infisical, unexpected error: " + err.Error()
		if getErr != nil {
			errMsg += "\nAdditionally, verifying whether the sub-organization still exists failed: " + getErr.Error()
		}
		resp.Diagnostics.AddError("Error deleting sub-organization", errMsg)
		return
	}
}

func (r *subOrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import sub-organization",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
