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
	_ resource.Resource = &certManagerApplicationUserResource{}
)

func NewCertManagerApplicationUserResource() resource.Resource {
	return &certManagerApplicationUserResource{}
}

type certManagerApplicationUserResource struct {
	client *infisical.Client
}

type certManagerApplicationUserResourceModel struct {
	Id            types.String `tfsdk:"id"`
	MembershipId  types.String `tfsdk:"membership_id"`
	ApplicationId types.String `tfsdk:"application_id"`
	Email         types.String `tfsdk:"email"`
	UserId        types.String `tfsdk:"user_id"`
	Role          types.String `tfsdk:"role"`
	CustomRoleId  types.String `tfsdk:"custom_role_id"`
}

func (r *certManagerApplicationUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_application_user"
}

func (r *certManagerApplicationUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage user memberships for a Certificate Manager application in Infisical. Only Machine Identity authentication is supported for this resource. Import: `terraform import <addr> <applicationId>:<userId>`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the user membership",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"membership_id": schema.StringAttribute{
				Description: "The ID of the user membership",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "The ID of the Certificate Manager application",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Description: "The email of the user to add",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The ID of the user",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role": schema.StringAttribute{
				Description: "The role to assign to the user (admin, member, viewer, or a custom role slug)",
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

func (r *certManagerApplicationUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerApplicationUserResource) findMembershipByEmail(applicationId, email string) (*infisical.PkiApplicationMember, error) {
	members, err := r.client.ListPkiApplicationUserMembers(infisical.ListPkiApplicationUserMembersRequest{
		ApplicationId: applicationId,
	})
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(email)
	for i := range members.Memberships {
		m := &members.Memberships[i]
		if m.Details != nil {
			if m.Details.Email != nil && strings.ToLower(*m.Details.Email) == target {
				return m, nil
			}
			if m.Details.Username != nil && strings.ToLower(*m.Details.Username) == target {
				return m, nil
			}
		}
	}

	return nil, nil
}

func (r *certManagerApplicationUserResource) findMembershipByUserId(applicationId, userId string) (*infisical.PkiApplicationMember, error) {
	members, err := r.client.ListPkiApplicationUserMembers(infisical.ListPkiApplicationUserMembersRequest{
		ApplicationId: applicationId,
	})
	if err != nil {
		return nil, err
	}

	for i := range members.Memberships {
		m := &members.Memberships[i]
		if m.ActorUserId != nil && *m.ActorUserId == userId {
			return m, nil
		}
	}

	return nil, nil
}

func (r *certManagerApplicationUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to add user to Certificate Manager application",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	addResponse, err := r.client.AddPkiApplicationUserMembers(infisical.AddPkiApplicationUserMembersRequest{
		ApplicationId: plan.ApplicationId.ValueString(),
		Emails:        []string{plan.Email.ValueString()},
		Role:          plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding user to Certificate Manager application",
			"Couldn't add user to application in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	email := plan.Email.ValueString()
	emailLower := strings.ToLower(email)
	for _, unresolved := range addResponse.Unresolved {
		if strings.ToLower(unresolved) == emailLower {
			resp.Diagnostics.AddError(
				"Unable to add user to Certificate Manager application",
				fmt.Sprintf("User %q was not found in the Infisical organization. Verify the email is correct and the user has been invited to the organization.", email),
			)
			return
		}
	}
	for _, skipped := range addResponse.Skipped {
		if strings.ToLower(skipped) == emailLower {
			resp.Diagnostics.AddWarning(
				"User is already a member of the Certificate Manager application",
				fmt.Sprintf("User %q is already attached to this application. The existing membership will be adopted into Terraform state.", email),
			)
		}
	}

	var member *infisical.PkiApplicationMember
	target := emailLower
	for i := range addResponse.Memberships {
		m := &addResponse.Memberships[i]
		if m.Details != nil {
			if m.Details.Email != nil && strings.ToLower(*m.Details.Email) == target {
				member = m
				break
			}
			if m.Details.Username != nil && strings.ToLower(*m.Details.Username) == target {
				member = m
				break
			}
		}
	}
	if member == nil {
		found, err := r.findMembershipByEmail(plan.ApplicationId.ValueString(), plan.Email.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading user membership after create",
				"Couldn't read user membership from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
		if found == nil {
			resp.Diagnostics.AddError(
				"Error reading user membership after create",
				"User membership was not found after creation. The Infisical API did not return the expected membership.",
			)
			return
		}
		member = found
	}

	plan.Id = types.StringValue(member.MembershipId)
	plan.MembershipId = types.StringValue(member.MembershipId)
	if member.ActorUserId != nil {
		plan.UserId = types.StringValue(*member.ActorUserId)
	} else {
		plan.UserId = types.StringNull()
	}
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

func (r *certManagerApplicationUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read Certificate Manager application user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var member *infisical.PkiApplicationMember
	var err error

	if !state.UserId.IsNull() && state.UserId.ValueString() != "" {
		member, err = r.findMembershipByUserId(state.ApplicationId.ValueString(), state.UserId.ValueString())
	} else {
		member, err = r.findMembershipByEmail(state.ApplicationId.ValueString(), state.Email.ValueString())
	}

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading Certificate Manager application user",
			"Couldn't read user membership from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	if member == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Id = types.StringValue(member.MembershipId)
	state.MembershipId = types.StringValue(member.MembershipId)
	if member.ActorUserId != nil {
		state.UserId = types.StringValue(*member.ActorUserId)
	}
	if member.Role == "custom" && member.CustomRoleId != nil && !state.Role.IsNull() && state.Role.ValueString() != "" && state.Role.ValueString() != "custom" {
	} else {
		state.Role = types.StringValue(member.Role)
	}
	if member.CustomRoleId != nil {
		state.CustomRoleId = types.StringValue(*member.CustomRoleId)
	} else {
		state.CustomRoleId = types.StringNull()
	}

	if state.Email.IsNull() || state.Email.ValueString() == "" {
		if member.Details != nil {
			if member.Details.Email != nil {
				state.Email = types.StringValue(*member.Details.Email)
			} else if member.Details.Username != nil {
				state.Email = types.StringValue(*member.Details.Username)
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerApplicationUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update Certificate Manager application user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerApplicationUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerApplicationUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateResp, err := r.client.UpdatePkiApplicationUserMemberRole(infisical.UpdatePkiApplicationUserMemberRoleRequest{
		ApplicationId: plan.ApplicationId.ValueString(),
		UserId:        state.UserId.ValueString(),
		Role:          plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Certificate Manager application user",
			"Couldn't update user membership in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	member := updateResp.Membership
	plan.Id = types.StringValue(member.MembershipId)
	plan.MembershipId = types.StringValue(member.MembershipId)
	if member.ActorUserId != nil {
		plan.UserId = types.StringValue(*member.ActorUserId)
	} else {
		plan.UserId = state.UserId
	}
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

func (r *certManagerApplicationUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to remove user from Certificate Manager application",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerApplicationUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RemovePkiApplicationUserMember(infisical.RemovePkiApplicationUserMemberRequest{
		ApplicationId: state.ApplicationId.ValueString(),
		UserId:        state.UserId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing user from Certificate Manager application",
			"Couldn't remove user from application in Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerApplicationUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier in the format <applicationId>:<userId>, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
}
