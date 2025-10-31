package resource

import (
	"context"
	"fmt"
	"net/url"
	infisical "terraform-provider-infisical/internal/client"
	"time"

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

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectGroupResource() resource.Resource {
	return &ProjectGroupResource{}
}

// ProjectGroupResource is the resource implementation.
type ProjectGroupResource struct {
	client *infisical.Client
}

// projectResourceSourceModel describes the data source data model.
type ProjectGroupResourceModel struct {
	ProjectID    types.String       `tfsdk:"project_id"`
	GroupID      types.String       `tfsdk:"group_id"`
	GroupName    types.String       `tfsdk:"group_name"`
	Roles        []ProjectGroupRole `tfsdk:"roles"`
	MembershipID types.String       `tfsdk:"membership_id"`
}

type ProjectGroupRole struct {
	RoleSlug                 types.String `tfsdk:"role_slug"`
	IsTemporary              types.Bool   `tfsdk:"is_temporary"`
	TemporaryRange           types.String `tfsdk:"temporary_range"`
	TemporaryAccessStartTime types.String `tfsdk:"temporary_access_start_time"`
}

// Metadata returns the resource type name.
func (r *ProjectGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_group"
}

// Schema defines the schema for the resource.
func (r *ProjectGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create project groups & save to Infisical. Only Machine Identity authentication is supported for this data source",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The id of the project.",
				Required:    true,
			},
			"group_id": schema.StringAttribute{
				Description:   "The id of the group.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_name": schema.StringAttribute{
				Description: "The name of the group.",
				Optional:    true,
			},
			"membership_id": schema.StringAttribute{
				Description:   "The membership Id of the project group",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"roles": schema.SetNestedAttribute{
				Required:    true,
				Description: "The roles assigned to the project group",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_slug": schema.StringAttribute{
							Description: "The slug of the role",
							Required:    true,
						},
						"is_temporary": schema.BoolAttribute{
							Description: "Flag to indicate the assigned role is temporary or not. When is_temporary is true fields temporary_mode, temporary_range and temporary_access_start_time is required.",
							Optional:    true,
						},
						"temporary_range": schema.StringAttribute{
							Description: "TTL for the temporary time. Eg: 1m, 1h, 1d. Default: 1h",
							Optional:    true,
						},
						"temporary_access_start_time": schema.StringAttribute{
							Description: "ISO time for which temporary access should begin. This is in the format YYYY-MM-DDTHH:MM:SSZ e.g. 2024-09-19T12:43:13Z",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ProjectGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ProjectGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan ProjectGroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.GroupID.ValueString() == "" && plan.GroupName.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Unable to create project group",
			"Must provide either group_id or group_name",
		)
		return
	}

	if plan.GroupID.ValueString() != "" && plan.GroupName.ValueString() != "" {
		resp.Diagnostics.AddError(
			"Unable to create project group",
			"Must provide either group_id or group_name, not both",
		)
		return
	}

	var roles []infisical.CreateProjectGroupRequestRoles
	var hasAtleastOnePermanentRole bool
	for _, el := range plan.Roles {
		isTemporary := el.IsTemporary.ValueBool()
		temporaryRange := el.TemporaryRange.ValueString()
		TemporaryAccessStartTime := time.Now().UTC()

		if !isTemporary {
			hasAtleastOnePermanentRole = true
		}

		temporaryMode := ""
		if isTemporary {
			temporaryMode = TEMPORARY_MODE_RELATIVE

			if el.TemporaryAccessStartTime.IsNull() {
				resp.Diagnostics.AddError(
					"Field temporary_access_start_time is required for temporary roles",
					fmt.Sprintf("Must provide valid ISO timestamp (YYYY-MM-DDTHH:MM:SSZ) for field temporary_access_start_time, role %s", el.RoleSlug.ValueString()),
				)
				return
			}
		}

		if isTemporary && temporaryRange == "" {
			temporaryRange = TEMPORARY_RANGE_DEFAULT
		}

		if el.TemporaryAccessStartTime.ValueString() != "" {
			var err error
			TemporaryAccessStartTime, err = time.Parse(time.RFC3339, el.TemporaryAccessStartTime.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error parsing field temporary_access_start_time",
					fmt.Sprintf("Must provider valid ISO timestamp for field temporary_access_start_time %s, role %s", el.TemporaryAccessStartTime.ValueString(), el.RoleSlug.ValueString()),
				)
				return
			}
		}

		roles = append(roles, infisical.CreateProjectGroupRequestRoles{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: TemporaryAccessStartTime,
		})
	}

	if !hasAtleastOnePermanentRole {
		resp.Diagnostics.AddError("Error assigning role to group", "Must have atleast one permanent role")
		return
	}

	request := infisical.CreateProjectGroupRequest{
		ProjectId: plan.ProjectID.ValueString(),
		Roles:     roles,
	}

	if plan.GroupID.ValueString() != "" {
		request.GroupIdOrName = plan.GroupID.ValueString()
	} else {
		request.GroupIdOrName = url.QueryEscape(plan.GroupName.ValueString())
	}

	projectGroupResponse, err := r.client.CreateProjectGroup(request)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error attaching group to project",
			"Couldn't create project group to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.MembershipID = types.StringValue(projectGroupResponse.Membership.ID)
	plan.GroupID = types.StringValue(projectGroupResponse.Membership.GroupID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ProjectGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state ProjectGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectID.ValueString() == "" || state.GroupID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	projectGroupMembership, err := r.client.GetProjectGroupMembership(infisical.GetProjectGroupMembershipRequest{
		ProjectId: state.ProjectID.ValueString(),
		GroupId:   state.GroupID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading project group membership",
				"Couldn't read project group membership from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	stateRoleMap := make(map[string]ProjectGroupRole)
	for _, role := range state.Roles {
		stateRoleMap[role.RoleSlug.ValueString()] = role
	}

	planRoles := make([]ProjectGroupRole, 0, len(projectGroupMembership.Membership.Roles))
	for _, el := range projectGroupMembership.Membership.Roles {
		val := ProjectGroupRole{
			RoleSlug:                 types.StringValue(el.Role),
			TemporaryRange:           types.StringValue(el.TemporaryRange),
			IsTemporary:              types.BoolValue(el.IsTemporary),
			TemporaryAccessStartTime: types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339)),
		}

		if el.Role == "custom" && el.CustomRoleSlug != "" {
			val.RoleSlug = types.StringValue(el.CustomRoleSlug)
		}

		/*
			We do the following because we want to maintain the state when the API returns these properties
			with default values. Without this, there will be unlimited drift because of the optional values.
		*/
		previousRoleState, ok := stateRoleMap[val.RoleSlug.ValueString()]
		if ok {
			if previousRoleState.IsTemporary.ValueBool() && el.IsTemporary {
				if previousRoleState.TemporaryRange.IsNull() && el.TemporaryRange == TEMPORARY_RANGE_DEFAULT {
					val.TemporaryRange = types.StringNull()
				}
			}

			if previousRoleState.IsTemporary.IsNull() && !el.IsTemporary {
				val.IsTemporary = types.BoolNull()
			}
		}

		if !el.IsTemporary {
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccessStartTime = types.StringNull()
		}

		planRoles = append(planRoles, val)
	}

	state.Roles = planRoles
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ProjectGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan ProjectGroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectGroupResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ProjectID != state.ProjectID {
		resp.Diagnostics.AddError(
			"Unable to update project ID",
			fmt.Sprintf("Cannot change project ID, previous project: %s, new project: %s", state.ProjectID, plan.ProjectID),
		)
		return
	}

	if plan.GroupID != state.GroupID {
		resp.Diagnostics.AddError(
			"Unable to update project group",
			fmt.Sprintf("Cannot change group ID, previous group: %s, new group: %s", state.GroupID, plan.GroupID),
		)
		return
	}

	if plan.GroupName != state.GroupName {
		resp.Diagnostics.AddError(
			"Unable to update project group",
			fmt.Sprintf("Cannot change group name, previous group name: %s, new group name: %s", state.GroupName, plan.GroupName),
		)
		return
	}

	var roles []infisical.UpdateProjectGroupRequestRoles
	var hasAtleastOnePermanentRole bool
	for _, el := range plan.Roles {
		isTemporary := el.IsTemporary.ValueBool()
		temporaryRange := el.TemporaryRange.ValueString()
		TemporaryAccessStartTime := time.Now().UTC()

		if !isTemporary {
			hasAtleastOnePermanentRole = true
		}

		temporaryMode := ""
		if isTemporary {
			temporaryMode = TEMPORARY_MODE_RELATIVE

			if el.TemporaryAccessStartTime.IsNull() {
				resp.Diagnostics.AddError(
					"Field temporary_access_start_time is required for temporary roles",
					fmt.Sprintf("Must provide valid ISO timestamp (YYYY-MM-DDTHH:MM:SSZ) for field temporary_access_start_time, role %s", el.RoleSlug.ValueString()),
				)
				return
			}
		}
		if isTemporary && temporaryRange == "" {
			temporaryRange = "1h"
		}

		if el.TemporaryAccessStartTime.ValueString() != "" {
			var err error
			TemporaryAccessStartTime, err = time.Parse(time.RFC3339, el.TemporaryAccessStartTime.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error parsing field temporary_access_start_time",
					fmt.Sprintf("Must provider valid ISO timestamp for field temporary_access_start_time %s, role %s", el.TemporaryAccessStartTime.ValueString(), el.RoleSlug.ValueString()),
				)
				return
			}
		}

		roles = append(roles, infisical.UpdateProjectGroupRequestRoles{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: TemporaryAccessStartTime,
		})
	}

	if !hasAtleastOnePermanentRole {
		resp.Diagnostics.AddError("Error assigning role to group", "Must have atleast one permanent role")
		return
	}

	_, err := r.client.UpdateProjectGroup(infisical.UpdateProjectGroupRequest{
		ProjectId: state.ProjectID.ValueString(),
		GroupId:   state.GroupID.ValueString(),
		Roles:     roles,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning roles to group",
			"Couldn't update role, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ProjectGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state ProjectGroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectGroup(infisical.DeleteProjectGroupRequest{
		ProjectId: state.ProjectID.ValueString(),
		GroupId:   state.GroupID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project group",
			"Couldn't delete project group from Infiscial, unexpected error: "+err.Error(),
		)
	}
}
