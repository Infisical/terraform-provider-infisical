package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &certManagerApplicationGroupResource{}
)

func NewCertManagerApplicationGroupResource() resource.Resource {
	return &certManagerApplicationGroupResource{}
}

type certManagerApplicationGroupResource struct {
	client *infisical.Client
}

type certManagerApplicationGroupResourceModel struct {
	Id            types.String `tfsdk:"id"`
	MembershipId  types.String `tfsdk:"membership_id"`
	ApplicationId types.String `tfsdk:"application_id"`
	GroupId       types.String `tfsdk:"group_id"`
	Role          types.String `tfsdk:"role"`
	CustomRoleId  types.String `tfsdk:"custom_role_id"`
}

func (r *certManagerApplicationGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_application_group"
}

func (r *certManagerApplicationGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage group memberships for a PKI application in Infisical. Only Machine Identity authentication is supported for this resource. Import: `terraform import <addr> <applicationId>:<groupId>`.",
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
			"application_id": schema.StringAttribute{
				Description: "The ID of the PKI application",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "The ID of the group to add",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "The role to assign to the group (admin, member, viewer, or a custom role slug)",
				Required:    true,
			},
			"custom_role_id": schema.StringAttribute{
				Description: "The ID of the custom role, when role is canonicalized to 'custom' by the backend",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *certManagerApplicationGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerApplicationGroupResource) findMembershipByGroupId(applicationId, groupId string) (*infisical.PkiApplicationMember, error) {
	members, err := r.client.ListPkiApplicationGroupMembers(infisical.ListPkiApplicationGroupMembersRequest{
		ApplicationId: applicationId,
	})
	if err != nil {
		return nil, err
	}

	for i := range members.Memberships {
		m := &members.Memberships[i]
		if m.ActorGroupId != nil && *m.ActorGroupId == groupId {
			return m, nil
		}
	}

	return nil, nil
}

func (r *certManagerApplicationGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create PKI application group membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	added, err := r.client.AddPkiApplicationGroupMember(infisical.AddPkiApplicationGroupMemberRequest{
		ApplicationId: plan.ApplicationId.ValueString(),
		GroupId:       plan.GroupId.ValueString(),
		Role:          plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error adding group to PKI application", err.Error())
		return
	}

	plan.Id = types.StringValue(added.Membership.MembershipId)
	plan.MembershipId = types.StringValue(added.Membership.MembershipId)
	if added.Membership.ActorGroupId != nil {
		plan.GroupId = types.StringValue(*added.Membership.ActorGroupId)
	}
	if added.Membership.Role == "custom" && added.Membership.CustomRoleId != nil && !plan.Role.IsNull() && plan.Role.ValueString() != "" && plan.Role.ValueString() != "custom" {
	} else {
		plan.Role = types.StringValue(added.Membership.Role)
	}
	if added.Membership.CustomRoleId != nil {
		plan.CustomRoleId = types.StringValue(*added.Membership.CustomRoleId)
	} else {
		plan.CustomRoleId = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerApplicationGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read PKI application group membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.findMembershipByGroupId(state.ApplicationId.ValueString(), state.GroupId.ValueString())
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PKI application group membership", err.Error())
		return
	}

	if member == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Id = types.StringValue(member.MembershipId)
	state.MembershipId = types.StringValue(member.MembershipId)
	if member.Role == "custom" && member.CustomRoleId != nil && !state.Role.IsNull() && state.Role.ValueString() != "" && state.Role.ValueString() != "custom" {
	} else {
		state.Role = types.StringValue(member.Role)
	}
	if member.CustomRoleId != nil {
		state.CustomRoleId = types.StringValue(*member.CustomRoleId)
	} else {
		state.CustomRoleId = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerApplicationGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update PKI application group membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerApplicationGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateResp, err := r.client.UpdatePkiApplicationGroupMemberRole(infisical.UpdatePkiApplicationGroupMemberRoleRequest{
		ApplicationId: plan.ApplicationId.ValueString(),
		GroupId:       plan.GroupId.ValueString(),
		Role:          plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating PKI application group membership", err.Error())
		return
	}

	member := updateResp.Membership
	plan.Id = types.StringValue(member.MembershipId)
	plan.MembershipId = types.StringValue(member.MembershipId)
	if member.Role == "custom" && member.CustomRoleId != nil && !plan.Role.IsNull() && plan.Role.ValueString() != "" && plan.Role.ValueString() != "custom" {
	} else {
		plan.Role = types.StringValue(member.Role)
	}
	if member.CustomRoleId != nil {
		plan.CustomRoleId = types.StringValue(*member.CustomRoleId)
	} else {
		plan.CustomRoleId = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerApplicationGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete PKI application group membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RemovePkiApplicationGroupMember(infisical.RemovePkiApplicationGroupMemberRequest{
		ApplicationId: state.ApplicationId.ValueString(),
		GroupId:       state.GroupId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error removing group from PKI application", err.Error())
		return
	}
}

func (r *certManagerApplicationGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier in the format <applicationId>:<groupId>, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[1])...)
}
