package resource

import (
	"context"
	"encoding/json"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &ProjectUserResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectUserResource() resource.Resource {
	return &ProjectUserResource{}
}

const TEMPORARY_MODE_RELATIVE = "relative"
const TEMPORARY_RANGE_DEFAULT = "1h"

// ProjectUserResource is the resource implementation.
type ProjectUserResource struct {
	client *infisical.Client
}

// projectResourceSourceModel describes the data source data model.
type ProjectUserResourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	Username     types.String `tfsdk:"username"`
	Roles        types.String `tfsdk:"roles"`
	MembershipId types.String `tfsdk:"membership_id"`
}

type ProjectUserRole struct {
	RoleSlug string `json:"role_slug"`
}

// Metadata returns the resource type name.
func (r *ProjectUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_user"
}

// Schema defines the schema for the resource.
func (r *ProjectUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create project users & save to Infisical. Only Machine Identity authentication is supported for this resource",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description:   "The id of the project",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"username": schema.StringAttribute{
				Description: "The usename of the user. By default its the email",
				Required:    true,
			},
			"membership_id": schema.StringAttribute{
				Description:   "The membershipId of the project user",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"roles": schema.StringAttribute{
				Required:    true,
				Description: "JSON array of role assignments for this user. Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
				PlanModifiers: []planmodifier.String{
					pkg.UnorderedJsonEquivalentModifier{},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ProjectUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *ProjectUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan ProjectUserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var parsedRoles []ProjectUserRole
	err := json.Unmarshal([]byte(plan.Roles.ValueString()), &parsedRoles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing roles JSON",
			fmt.Sprintf("Failed to parse roles JSON: %s", err.Error()),
		)
		return
	}

	var roles []infisical.UpdateProjectUserRequestRoles
	for _, el := range parsedRoles {
		if el.RoleSlug == "" {
			resp.Diagnostics.AddError(
				"Error parsing roles JSON",
				"Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
			)
			return
		}
		roles = append(roles, infisical.UpdateProjectUserRequestRoles{
			Role: el.RoleSlug,
		})
	}

	invitedUser, err := r.client.InviteUsersToProject(infisical.InviteUsersToProjectRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Usernames: []string{plan.Username.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error inviting user",
			"Couldn't create project user to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	if len(invitedUser) == 0 {
		resp.Diagnostics.AddError(
			"Error inviting user",
			"Could not add user to project. No invite was sent, is the user already in the project?",
		)
		return
	}

	_, err = r.client.UpdateProjectUser(infisical.UpdateProjectUserRequest{
		ProjectID:    plan.ProjectID.ValueString(),
		MembershipID: invitedUser[0].ID,
		Roles:        roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning roles to user",
			"Couldn't update role , unexpected error: "+err.Error(),
		)
		return
	}

	projectUserDetails, err := r.client.GetProjectUserByUsername(infisical.GetProjectUserByUserNameRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Username:  plan.Username.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching user",
			"Couldn't find user in project, unexpected error: "+err.Error(),
		)
		return
	}

	plan.MembershipId = types.StringValue(projectUserDetails.Membership.ID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *ProjectUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state ProjectUserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectID.ValueString() == "" || state.Username.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	projectUserDetails, err := r.client.GetProjectUserByUsername(infisical.GetProjectUserByUserNameRequest{
		ProjectID: state.ProjectID.ValueString(),
		Username:  state.Username.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error fetching user",
			"Couldn't find user in project, unexpected error: "+err.Error(),
		)

		return
	}

	planRoles := make([]ProjectUserRole, 0, len(projectUserDetails.Membership.Roles))
	for _, el := range projectUserDetails.Membership.Roles {
		val := ProjectUserRole{
			RoleSlug: el.Role,
		}
		planRoles = append(planRoles, val)
	}

	rolesJSON, err := json.Marshal(planRoles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error serializing roles to JSON",
			fmt.Sprintf("Failed to serialize roles to JSON: %s", err.Error()),
		)
		return
	}

	state.Roles = types.StringValue(string(rolesJSON))
	state.MembershipId = types.StringValue(projectUserDetails.Membership.ID)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ProjectUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan ProjectUserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectUserResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Username != plan.Username {
		resp.Diagnostics.AddError(
			"Unable to update project user",
			fmt.Sprintf("Cannot change username, previous username: %s, new username: %s", state.Username, plan.Username),
		)
		return
	}

	var parsedRoles []ProjectUserRole
	err := json.Unmarshal([]byte(plan.Roles.ValueString()), &parsedRoles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing roles JSON",
			fmt.Sprintf("Failed to parse roles JSON: %s", err.Error()),
		)
		return
	}

	var roles []infisical.UpdateProjectUserRequestRoles
	for _, el := range parsedRoles {
		if el.RoleSlug == "" {
			resp.Diagnostics.AddError(
				"Error parsing roles JSON",
				"Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
			)
			return
		}
		roles = append(roles, infisical.UpdateProjectUserRequestRoles{
			Role: el.RoleSlug,
		})
	}

	_, err = r.client.UpdateProjectUser(infisical.UpdateProjectUserRequest{
		ProjectID:    plan.ProjectID.ValueString(),
		MembershipID: plan.MembershipId.ValueString(),
		Roles:        roles,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning roles to user",
			"Couldn't update role, unexpected error: "+err.Error(),
		)
		return
	}

	projectUserDetails, err := r.client.GetProjectUserByUsername(infisical.GetProjectUserByUserNameRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Username:  plan.Username.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching user",
			"Couldn't find user in project, unexpected error: "+err.Error(),
		)
		return
	}

	plan.MembershipId = types.StringValue(projectUserDetails.Membership.ID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ProjectUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state ProjectUserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectUser(infisical.DeleteProjectUserRequest{
		ProjectID: state.ProjectID.ValueString(),
		Username:  []string{state.Username.ValueString()},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project user",
			"Couldn't delete project user from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
