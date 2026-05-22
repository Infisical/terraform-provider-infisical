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
	_ resource.Resource = &certManagerGroupResource{}
)

func NewCertManagerGroupResource() resource.Resource {
	return &certManagerGroupResource{}
}

type certManagerGroupResource struct {
	client *infisical.Client
}

type certManagerGroupResourceModel struct {
	Id           types.String `tfsdk:"id"`
	MembershipId types.String `tfsdk:"membership_id"`
	GroupId      types.String `tfsdk:"group_id"`
	Role         types.String `tfsdk:"role"`
}

func (r *certManagerGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_group"
}

func (r *certManagerGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage group memberships in Certificate Manager. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the group membership",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"membership_id": schema.StringAttribute{
				Description: "The ID of the group membership",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "The ID of the group",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "The role to assign to the group (admin, member, or viewer)",
				Required:    true,
			},
		},
	}
}

func (r *certManagerGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create Certificate Manager group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	added, err := r.client.AddCertManagerGroup(infisical.AddCertManagerGroupRequest{
		GroupId: plan.GroupId.ValueString(),
		Roles:   []infisical.CertManagerMembershipRoleUpdate{{Role: plan.Role.ValueString()}},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding group to Certificate Manager",
			"Couldn't add group to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(added.GroupMembership.ID)
	plan.MembershipId = types.StringValue(added.GroupMembership.ID)
	plan.GroupId = types.StringValue(added.GroupMembership.GroupId)
	plan.Role = types.StringValue(firstRole(added.GroupMembership.Roles, plan.Role.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read Certificate Manager group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.client.GetCertManagerGroup(infisical.GetCertManagerGroupRequest{
		GroupId: state.GroupId.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading Certificate Manager group",
			"Couldn't read group membership from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Id = types.StringValue(refreshed.GroupMembership.ID)
	state.MembershipId = types.StringValue(refreshed.GroupMembership.ID)
	state.Role = types.StringValue(firstRole(refreshed.GroupMembership.Roles, state.Role.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update Certificate Manager group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateCertManagerGroup(infisical.UpdateCertManagerGroupRequest{
		GroupId: plan.GroupId.ValueString(),
		Roles:   []infisical.CertManagerMembershipRoleUpdate{{Role: plan.Role.ValueString()}},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Certificate Manager group",
			"Couldn't update group roles in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	refreshed, err := r.client.GetCertManagerGroup(infisical.GetCertManagerGroupRequest{
		GroupId: plan.GroupId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Certificate Manager group",
			"Couldn't read group membership from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(refreshed.GroupMembership.ID)
	plan.MembershipId = types.StringValue(refreshed.GroupMembership.ID)
	plan.Role = types.StringValue(firstRole(refreshed.GroupMembership.Roles, plan.Role.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete Certificate Manager group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RemoveCertManagerGroup(infisical.RemoveCertManagerGroupRequest{
		GroupId: state.GroupId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing Certificate Manager group",
			"Couldn't remove group from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import Certificate Manager group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	refreshed, err := r.client.GetCertManagerGroup(infisical.GetCertManagerGroupRequest{
		GroupId: req.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to import Certificate Manager group",
			fmt.Sprintf("Couldn't find Certificate Manager group for group_id %q: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), refreshed.GroupMembership.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("membership_id"), refreshed.GroupMembership.ID)...)
}
