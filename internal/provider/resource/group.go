package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &ProjectGroupResource{}
)

// NewGroupResource is a helper function to simplify the provider implementation.
func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

// GroupResource is the resource implementation.
type GroupResource struct {
	client *infisical.Client
}

// groupResourceSourceModel describes the resource data model.
type GroupResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Slug types.String `tfsdk:"slug"`
	Role types.String `tfsdk:"role"`
}

// Metadata returns the resource type name.
func (r *GroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the schema for the resource.
func (r *GroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create groups & save to Infisical. Only Machine Identity authentication is supported for this resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The id of the group.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the group.",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the group.",
				Required:    true,
			},
			"role": schema.StringAttribute{
				Description: "The role of the group.",
				Required:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *GroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan GroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := infisical.CreateGroupRequest{
		Name: plan.Name.ValueString(),
		Slug: plan.Slug.ValueString(),
		Role: plan.Role.ValueString(),
	}

	groupResponse, err := r.client.CreateGroup(request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating group",
			"Couldn't create group in Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(groupResponse.ID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupResponse, err := r.client.GetGroupById(infisical.GetGroupByIdRequest{ID: state.ID.ValueString()})
	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading group",
			"Couldn't read group in Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(groupResponse.Name)
	state.Slug = types.StringValue(groupResponse.Slug)
	if groupResponse.CustomRoleSlug != "" {
		state.Role = types.StringValue(groupResponse.CustomRoleSlug)
	} else {
		state.Role = types.StringValue(groupResponse.Role)
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan GroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := infisical.UpdateGroupRequest{
		ID:   plan.ID.ValueString(),
		Name: plan.Name.ValueString(),
		Slug: plan.Slug.ValueString(),
		Role: plan.Role.ValueString(),
	}

	_, err := r.client.UpdateGroup(request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating group",
			"Couldn't update group in Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteGroup(infisical.DeleteGroupRequest{ID: state.ID.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group",
			"Couldn't delete group in Infiscial, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	groupResponse, err := r.client.GetGroupById(infisical.GetGroupByIdRequest{ID: req.ID})
	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.Diagnostics.AddError(
				"Error importing group",
				fmt.Sprintf("No group found with ID: %s", req.ID),
			)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error importing group",
				"Couldn't read group from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	var state GroupResourceModel
	state.ID = types.StringValue(req.ID)
	state.Name = types.StringValue(groupResponse.Name)
	state.Slug = types.StringValue(groupResponse.Slug)
	if groupResponse.CustomRoleSlug != "" {
		state.Role = types.StringValue(groupResponse.CustomRoleSlug)
	} else {
		state.Role = types.StringValue(groupResponse.Role)
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
