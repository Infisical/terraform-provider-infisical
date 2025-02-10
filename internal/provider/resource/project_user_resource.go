package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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

// ProjectUserResource is the resource implementation.
type ProjectUserResource struct {
	client *infisical.Client
}

// projectResourceSourceModel describes the data source data model.
type ProjectUserResourceModel struct {
	ProjectID    types.String      `tfsdk:"project_id"`
	Username     types.String      `tfsdk:"username"`
	User         types.Object      `tfsdk:"user"`
	Roles        []ProjectUserRole `tfsdk:"roles"`
	MembershipId types.String      `tfsdk:"membership_id"`
}

type ProjectUserPersonalDetails struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
}

const TEMPORARY_MODE_RELATIVE = "relative"
const TEMPORARY_RANGE_DEFAULT = "1h"

type ProjectUserRole struct {
	ID                      types.String `tfsdk:"id"`
	RoleSlug                types.String `tfsdk:"role_slug"`
	CustomRoleID            types.String `tfsdk:"custom_role_id"`
	IsTemporary             types.Bool   `tfsdk:"is_temporary"`
	TemporaryMode           types.String `tfsdk:"temporary_mode"`
	TemporaryRange          types.String `tfsdk:"temporary_range"`
	TemporaryAccesStartTime types.String `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime  types.String `tfsdk:"temporary_access_end_time"`
}

// Metadata returns the resource type name.
func (r *ProjectUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_user"
}

// Schema defines the schema for the resource.
func (r *ProjectUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create project users & save to Infisical. Only Machine Identity authentication is supported for this data source",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The id of the project",
				Required:    true,
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
			"user": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The user details of the project user",
				Attributes: map[string]schema.Attribute{
					"email": schema.StringAttribute{
						Description: "The email of the user",
						Computed:    true,
					},
					"first_name": schema.StringAttribute{
						Description: "The first name of the user",
						Computed:    true,
					},
					"last_name": schema.StringAttribute{
						Description: "The last name of the user",
						Computed:    true,
					},
					"id": schema.StringAttribute{
						Description: "The id of the user",
						Computed:    true,
					},
				},
			},
			"roles": schema.ListNestedAttribute{
				Required:    true,
				Description: "The roles assigned to the project user",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the project user role.",
							Computed:    true,
						},
						"role_slug": schema.StringAttribute{
							Description: "The slug of the role",
							Required:    true,
						},
						"custom_role_id": schema.StringAttribute{
							Description: "The id of the custom role slug",
							Computed:    true,
							Optional:    true,
						},
						"is_temporary": schema.BoolAttribute{
							Description: "Flag to indicate the assigned role is temporary or not. When is_temporary is true fields temporary_mode, temporary_range and temporary_access_start_time is required.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"temporary_mode": schema.StringAttribute{
							Description: "Type of temporary access given. Types: relative. Default: relative",
							Optional:    true,
							Computed:    true,
						},
						"temporary_range": schema.StringAttribute{
							Description: "TTL for the temporary time. Eg: 1m, 1h, 1d. Default: 1h",
							Optional:    true,
							Computed:    true,
						},
						"temporary_access_start_time": schema.StringAttribute{
							Description: "ISO time for which temporary access should begin. The current time is used by default.",
							Optional:    true,
							Computed:    true,
						},
						"temporary_access_end_time": schema.StringAttribute{
							Description: "ISO time for which temporary access will end. Computed based on temporary_range and temporary_access_start_time",
							Computed:    true,
							Optional:    true,
						},
					},
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

	var roles []infisical.UpdateProjectUserRequestRoles
	var hasAtleastOnePermanentRole bool
	for _, el := range plan.Roles {
		isTemporary := el.IsTemporary.ValueBool()
		temporaryMode := el.TemporaryMode.ValueString()
		temporaryRange := el.TemporaryRange.ValueString()
		temporaryAccesStartTime := time.Now().UTC()

		if !isTemporary {
			hasAtleastOnePermanentRole = true
		}

		if el.TemporaryAccesStartTime.ValueString() != "" {
			var err error
			temporaryAccesStartTime, err = time.Parse(time.RFC3339, el.TemporaryAccesStartTime.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error parsing field TemporaryAccessStartTime",
					fmt.Sprintf("Must provider valid ISO timestamp for field temporaryAccesStartTime %s, role %s", el.TemporaryAccesStartTime.ValueString(), el.RoleSlug.ValueString()),
				)
				return
			}
		}

		// default values
		if isTemporary && temporaryMode == "" {
			temporaryMode = TEMPORARY_MODE_RELATIVE
		}
		if isTemporary && temporaryRange == "" {
			temporaryRange = "1h"
		}

		roles = append(roles, infisical.UpdateProjectUserRequestRoles{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
	}
	if !hasAtleastOnePermanentRole {
		resp.Diagnostics.AddError("Error assigning role to user", "Must have atleast one permanent role")
		return
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

	planRoles := make([]ProjectUserRole, 0, len(projectUserDetails.Membership.Roles))
	for _, el := range projectUserDetails.Membership.Roles {
		val := ProjectUserRole{
			ID:                      types.StringValue(el.ID),
			RoleSlug:                types.StringValue(el.Role),
			TemporaryAccessEndTime:  types.StringValue(el.TemporaryAccessEndTime.Format(time.RFC3339)),
			TemporaryRange:          types.StringValue(el.TemporaryRange),
			TemporaryMode:           types.StringValue(el.TemporaryMode),
			CustomRoleID:            types.StringValue(el.CustomRoleId),
			IsTemporary:             types.BoolValue(el.IsTemporary),
			TemporaryAccesStartTime: types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339)),
		}

		if el.CustomRoleId != "" {
			val.RoleSlug = types.StringValue(el.CustomRoleSlug)
		}

		if !el.IsTemporary {
			val.TemporaryMode = types.StringNull()
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccesStartTime = types.StringNull()
			val.TemporaryAccessEndTime = types.StringNull()
		}
		planRoles = append(planRoles, val)
	}
	plan.Roles = planRoles
	plan.MembershipId = types.StringValue(projectUserDetails.Membership.ID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userPersonalDetails := ProjectUserPersonalDetails{
		Email:     types.StringValue(projectUserDetails.Membership.User.Email),
		FirstName: types.StringValue(projectUserDetails.Membership.User.FirstName),
		LastName:  types.StringValue(projectUserDetails.Membership.User.LastName),
		ID:        types.StringValue(projectUserDetails.Membership.User.ID),
	}
	diags = resp.State.SetAttribute(ctx, path.Root("user"), userPersonalDetails)
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
			ID:                      types.StringValue(el.ID),
			RoleSlug:                types.StringValue(el.Role),
			TemporaryAccessEndTime:  types.StringValue(el.TemporaryAccessEndTime.Format(time.RFC3339)),
			TemporaryRange:          types.StringValue(el.TemporaryRange),
			TemporaryMode:           types.StringValue(el.TemporaryMode),
			CustomRoleID:            types.StringValue(el.CustomRoleId),
			IsTemporary:             types.BoolValue(el.IsTemporary),
			TemporaryAccesStartTime: types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339)),
		}
		if el.CustomRoleId != "" {
			val.RoleSlug = types.StringValue(el.CustomRoleSlug)
		}
		if !el.IsTemporary {
			val.TemporaryMode = types.StringNull()
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccesStartTime = types.StringNull()
			val.TemporaryAccessEndTime = types.StringNull()
		}
		planRoles = append(planRoles, val)
	}

	state.Roles = planRoles
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userPersonalDetails := ProjectUserPersonalDetails{
		Email:     types.StringValue(projectUserDetails.Membership.User.Email),
		FirstName: types.StringValue(projectUserDetails.Membership.User.FirstName),
		LastName:  types.StringValue(projectUserDetails.Membership.User.LastName),
		ID:        types.StringValue(projectUserDetails.Membership.User.ID),
	}
	diags = resp.State.SetAttribute(ctx, path.Root("user"), userPersonalDetails)
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

	var roles []infisical.UpdateProjectUserRequestRoles
	var hasAtleastOnePermanentRole bool
	for _, el := range plan.Roles {
		isTemporary := el.IsTemporary.ValueBool()
		temporaryMode := el.TemporaryMode.ValueString()
		temporaryRange := el.TemporaryRange.ValueString()
		temporaryAccesStartTime := time.Now().UTC()

		if !isTemporary {
			hasAtleastOnePermanentRole = true
		}

		if el.TemporaryAccesStartTime.ValueString() != "" {
			var err error
			temporaryAccesStartTime, err = time.Parse(time.RFC3339, el.TemporaryAccesStartTime.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error parsing field TemporaryAccessStartTime",
					fmt.Sprintf("Must provider valid ISO timestamp for field temporaryAccesStartTime %s, role %s", el.TemporaryAccesStartTime.ValueString(), el.RoleSlug.ValueString()),
				)
				return
			}
		}

		// default values
		if isTemporary && temporaryMode == "" {
			temporaryMode = TEMPORARY_MODE_RELATIVE
		}
		if isTemporary && temporaryRange == "" {
			temporaryRange = "1h"
		}

		roles = append(roles, infisical.UpdateProjectUserRequestRoles{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
	}

	if !hasAtleastOnePermanentRole {
		resp.Diagnostics.AddError("Error assigning role to user", "Must have atleast one permanent role")
		return
	}

	_, err := r.client.UpdateProjectUser(infisical.UpdateProjectUserRequest{
		ProjectID:    plan.ProjectID.ValueString(),
		MembershipID: plan.MembershipId.ValueString(),
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

	planRoles := make([]ProjectUserRole, 0, len(projectUserDetails.Membership.Roles))
	for _, el := range projectUserDetails.Membership.Roles {
		val := ProjectUserRole{
			ID:                      types.StringValue(el.ID),
			RoleSlug:                types.StringValue(el.Role),
			TemporaryAccessEndTime:  types.StringValue(el.TemporaryAccessEndTime.Format(time.RFC3339)),
			TemporaryRange:          types.StringValue(el.TemporaryRange),
			TemporaryMode:           types.StringValue(el.TemporaryMode),
			CustomRoleID:            types.StringValue(el.CustomRoleId),
			IsTemporary:             types.BoolValue(el.IsTemporary),
			TemporaryAccesStartTime: types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339)),
		}
		if el.CustomRoleId != "" {
			val.RoleSlug = types.StringValue(el.CustomRoleSlug)
		}
		if !el.IsTemporary {
			val.TemporaryMode = types.StringNull()
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccesStartTime = types.StringNull()
			val.TemporaryAccessEndTime = types.StringNull()
		}
		planRoles = append(planRoles, val)
	}
	plan.Roles = planRoles
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userPersonalDetails := ProjectUserPersonalDetails{
		Email:     types.StringValue(projectUserDetails.Membership.User.Email),
		FirstName: types.StringValue(projectUserDetails.Membership.User.FirstName),
		LastName:  types.StringValue(projectUserDetails.Membership.User.LastName),
		ID:        types.StringValue(projectUserDetails.Membership.User.ID),
	}
	diags = resp.State.SetAttribute(ctx, path.Root("user"), userPersonalDetails)
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
