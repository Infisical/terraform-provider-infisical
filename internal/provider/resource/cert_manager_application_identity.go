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
	_ resource.Resource = &certManagerApplicationIdentityResource{}
)

func NewCertManagerApplicationIdentityResource() resource.Resource {
	return &certManagerApplicationIdentityResource{}
}

type certManagerApplicationIdentityResource struct {
	client *infisical.Client
}

type certManagerApplicationIdentityResourceModel struct {
	Id            types.String `tfsdk:"id"`
	MembershipId  types.String `tfsdk:"membership_id"`
	ApplicationId types.String `tfsdk:"application_id"`
	IdentityId    types.String `tfsdk:"identity_id"`
	Role          types.String `tfsdk:"role"`
	CustomRoleId  types.String `tfsdk:"custom_role_id"`
}

func (r *certManagerApplicationIdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_application_identity"
}

func (r *certManagerApplicationIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage identity memberships for a PKI application in Infisical. Only Machine Identity authentication is supported for this resource. Import: `terraform import <addr> <applicationId>:<identityId>`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the identity membership",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"membership_id": schema.StringAttribute{
				Description: "The ID of the identity membership",
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
			"identity_id": schema.StringAttribute{
				Description: "The ID of the identity to add",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "The role to assign to the identity (admin, member, viewer, or a custom role slug)",
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

func (r *certManagerApplicationIdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerApplicationIdentityResource) findMembershipByIdentityId(applicationId, identityId string) (*infisical.PkiApplicationMember, error) {
	members, err := r.client.ListPkiApplicationIdentityMembers(infisical.ListPkiApplicationIdentityMembersRequest{
		ApplicationId: applicationId,
	})
	if err != nil {
		return nil, err
	}

	for i := range members.Memberships {
		m := &members.Memberships[i]
		if m.ActorIdentityId != nil && *m.ActorIdentityId == identityId {
			return m, nil
		}
	}

	return nil, nil
}

func (r *certManagerApplicationIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create PKI application identity membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationIdentityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	added, err := r.client.AddPkiApplicationIdentityMember(infisical.AddPkiApplicationIdentityMemberRequest{
		ApplicationId: plan.ApplicationId.ValueString(),
		IdentityId:    plan.IdentityId.ValueString(),
		Role:          plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error adding identity to PKI application", err.Error())
		return
	}

	plan.Id = types.StringValue(added.Membership.MembershipId)
	plan.MembershipId = types.StringValue(added.Membership.MembershipId)
	if added.Membership.ActorIdentityId != nil {
		plan.IdentityId = types.StringValue(*added.Membership.ActorIdentityId)
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

func (r *certManagerApplicationIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read PKI application identity membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationIdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.findMembershipByIdentityId(state.ApplicationId.ValueString(), state.IdentityId.ValueString())
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PKI application identity membership", err.Error())
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

func (r *certManagerApplicationIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update PKI application identity membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationIdentityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerApplicationIdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateResp, err := r.client.UpdatePkiApplicationIdentityMemberRole(infisical.UpdatePkiApplicationIdentityMemberRoleRequest{
		ApplicationId: plan.ApplicationId.ValueString(),
		IdentityId:    plan.IdentityId.ValueString(),
		Role:          plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating PKI application identity membership", err.Error())
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

func (r *certManagerApplicationIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete PKI application identity membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationIdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RemovePkiApplicationIdentityMember(infisical.RemovePkiApplicationIdentityMemberRequest{
		ApplicationId: state.ApplicationId.ValueString(),
		IdentityId:    state.IdentityId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error removing identity from PKI application", err.Error())
		return
	}
}

func (r *certManagerApplicationIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier in the format <applicationId>:<identityId>, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identity_id"), parts[1])...)
}
