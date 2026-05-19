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
	_ resource.Resource = &certManagerUserResource{}
)

func NewCertManagerUserResource() resource.Resource {
	return &certManagerUserResource{}
}

type certManagerUserResource struct {
	client *infisical.Client
}

type certManagerUserResourceModel struct {
	Id           types.String            `tfsdk:"id"`
	MembershipId types.String            `tfsdk:"membership_id"`
	Email        types.String            `tfsdk:"email"`
	UserId       types.String            `tfsdk:"user_id"`
	Roles        []CertManagerMemberRole `tfsdk:"roles"`
}

func (r *certManagerUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_user"
}

func (r *certManagerUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage user memberships at the cert manager scope in Infisical. Only Machine Identity authentication is supported for this resource. Import: `terraform import <addr> <email>`.",
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
			"email": schema.StringAttribute{
				Description: "The email of the user",
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
			"roles": certManagerRolesSchema(),
		},
	}
}

func (r *certManagerUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerUserResource) findUserByEmail(email string) (*infisical.CertManagerUserMembership, error) {
	users, err := r.client.ListCertManagerUsers()
	if err != nil {
		return nil, err
	}

	for i := range users.Memberships {
		m := &users.Memberships[i]
		if m.User.Email == email || m.User.Username == email {
			return m, nil
		}
	}

	return nil, nil
}

func (r *certManagerUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create cert manager user membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, hasPermanent, err := certManagerBuildRoleUpdates(plan.Roles, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error parsing roles", err.Error())
		return
	}
	if !hasPermanent {
		resp.Diagnostics.AddError("Error assigning role to user", "Must have at least one permanent role")
		return
	}

	_, err = r.client.InviteCertManagerUsers(infisical.InviteCertManagerUsersRequest{
		Emails: []string{plan.Email.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error inviting cert manager user", err.Error())
		return
	}

	member, err := r.findUserByEmail(plan.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error finding cert manager user after invite", err.Error())
		return
	}
	if member == nil {
		resp.Diagnostics.AddError("Error finding cert manager user after invite", "User not found in cert manager after invite")
		return
	}

	_, err = r.client.UpdateCertManagerUser(infisical.UpdateCertManagerUserRequest{
		UserId: member.UserID,
		Roles:  roles,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error assigning roles to cert manager user", err.Error())
		return
	}

	refreshed, err := r.client.GetCertManagerUser(infisical.GetCertManagerUserRequest{
		UserId: member.UserID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error fetching cert manager user", err.Error())
		return
	}

	plan.Id = types.StringValue(refreshed.Membership.ID)
	plan.MembershipId = types.StringValue(refreshed.Membership.ID)
	plan.UserId = types.StringValue(refreshed.Membership.UserID)
	apiRoles := certManagerRolesFromAPI(refreshed.Membership.Roles)
	plan.Roles = orderCertManagerRolesByPlan(plan.Roles, apiRoles)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read cert manager user membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.UserId.IsNull() || state.UserId.ValueString() == "" {
		member, err := r.findUserByEmail(state.Email.ValueString())
		if err != nil {
			if err == infisical.ErrNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("Error fetching cert manager user", err.Error())
			return
		}
		if member == nil {
			resp.State.RemoveResource(ctx)
			return
		}
		state.UserId = types.StringValue(member.UserID)
	}

	refreshed, err := r.client.GetCertManagerUser(infisical.GetCertManagerUserRequest{
		UserId: state.UserId.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error fetching cert manager user", err.Error())
		return
	}

	state.Id = types.StringValue(refreshed.Membership.ID)
	state.MembershipId = types.StringValue(refreshed.Membership.ID)
	state.UserId = types.StringValue(refreshed.Membership.UserID)
	if state.Email.IsNull() || state.Email.ValueString() == "" {
		state.Email = types.StringValue(refreshed.Membership.User.Email)
	}
	apiRoles := certManagerRolesFromAPI(refreshed.Membership.Roles)
	state.Roles = orderCertManagerRolesByPlan(state.Roles, apiRoles)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update cert manager user membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, hasPermanent, err := certManagerBuildRoleUpdates(plan.Roles, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error parsing roles", err.Error())
		return
	}
	if !hasPermanent {
		resp.Diagnostics.AddError("Error assigning role to user", "Must have at least one permanent role")
		return
	}

	_, err = r.client.UpdateCertManagerUser(infisical.UpdateCertManagerUserRequest{
		UserId: state.UserId.ValueString(),
		Roles:  roles,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating cert manager user roles", err.Error())
		return
	}

	refreshed, err := r.client.GetCertManagerUser(infisical.GetCertManagerUserRequest{
		UserId: state.UserId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error fetching cert manager user", err.Error())
		return
	}

	plan.Id = types.StringValue(refreshed.Membership.ID)
	plan.MembershipId = types.StringValue(refreshed.Membership.ID)
	plan.UserId = types.StringValue(refreshed.Membership.UserID)
	apiRoles := certManagerRolesFromAPI(refreshed.Membership.Roles)
	plan.Roles = orderCertManagerRolesByPlan(plan.Roles, apiRoles)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete cert manager user membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RemoveCertManagerUser(infisical.RemoveCertManagerUserRequest{
		UserId: state.UserId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error removing cert manager user", err.Error())
		return
	}
}

func (r *certManagerUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import cert manager user membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	member, err := r.findUserByEmail(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing cert manager user membership", err.Error())
		return
	}
	if member == nil {
		resp.Diagnostics.AddError(
			"Unable to import cert manager user membership",
			fmt.Sprintf("No cert manager user found matching email %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), member.UserID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), member.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("membership_id"), member.ID)...)
}
