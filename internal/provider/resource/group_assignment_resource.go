package resource

import (
	"context"
	"errors"
	"fmt"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                     = &groupAssignmentResource{}
	_ resource.ResourceWithImportState      = &groupAssignmentResource{}
	_ resource.ResourceWithConfigValidators = &groupAssignmentResource{}
)

// NewGroupAssignmentResource is a helper function to simplify the provider implementation.
func NewGroupAssignmentResource() resource.Resource {
	return &groupAssignmentResource{}
}

// groupAssignmentResource links a root-organization group into a sub-organization.
type groupAssignmentResource struct {
	client *infisical.Client
}

// groupAssignmentResourceModel describes the resource data model.
type groupAssignmentResourceModel struct {
	ID         types.String               `tfsdk:"id"`
	GroupID    types.String               `tfsdk:"group_id"`
	GroupSlug  types.String               `tfsdk:"group_slug"`
	GroupName  types.String               `tfsdk:"group_name"`
	GroupOrgID types.String               `tfsdk:"group_org_id"`
	Roles      []groupAssignmentRoleModel `tfsdk:"roles"`
}

type groupAssignmentRoleModel struct {
	RoleSlug                 types.String `tfsdk:"role_slug"`
	IsTemporary              types.Bool   `tfsdk:"is_temporary"`
	TemporaryRange           types.String `tfsdk:"temporary_range"`
	TemporaryAccessStartTime types.String `tfsdk:"temporary_access_start_time"`
}

// Metadata returns the resource type name.
func (r *groupAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_assignment"
}

// Schema defines the schema for the resource.
func (r *groupAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Link a group from the root (parent) organization into a sub-organization and assign it organization roles there. " +
			"The provider must be scoped to the target sub-organization through `auth.organization_slug`: Infisical links the group into whichever organization the session is scoped to. " +
			"Before linking, the group is validated against the groups the sub-organization can link from its parent. " +
			"Destroying this resource unlinks the group from the sub-organization; the group itself is left untouched in the root organization. " +
			"Requires an Infisical Enterprise plan. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the group's membership in the sub-organization.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": schema.StringAttribute{
				Description:   "The ID of the root-organization group to link. Exactly one of `group_id` or `group_slug` must be set. Changing this forces a new resource.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"group_slug": schema.StringAttribute{
				Description:   "The slug of the root-organization group to link. Exactly one of `group_id` or `group_slug` must be set. Changing this forces a new resource.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"group_name": schema.StringAttribute{
				Description:   "The name of the linked group.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_org_id": schema.StringAttribute{
				Description:   "The ID of the organization that owns the group (the root organization).",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"roles": schema.SetNestedAttribute{
				Description: "The organization roles assigned to the group within the sub-organization.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_slug": schema.StringAttribute{
							Description: "The slug of the organization role, e.g. `admin`, `member`, `no-access`, or the slug of a custom organization role defined in the sub-organization.",
							Required:    true,
						},
						"is_temporary": schema.BoolAttribute{
							Description: "Whether the role is temporary. When true, `temporary_access_start_time` is required. Defaults to `false`.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"temporary_range": schema.StringAttribute{
							Description: "TTL of a temporary role, e.g. 1m, 1h, 1d. Defaults to 1h for temporary roles.",
							Optional:    true,
						},
						"temporary_access_start_time": schema.StringAttribute{
							Description: "ISO time at which temporary access begins, in the format YYYY-MM-DDTHH:MM:SSZ, e.g. 2024-09-19T12:43:13Z. Required for temporary roles.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *groupAssignmentResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("group_id"),
			path.MatchRoot("group_slug"),
		),
	}
}

// Configure adds the provider configured client to the resource.
func (r *groupAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildGroupAssignmentRoles converts the configured roles into the API's roles array.
func buildGroupAssignmentRoles(roles []groupAssignmentRoleModel) ([]infisical.OrgGroupMembershipRoleRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	requestRoles := make([]infisical.OrgGroupMembershipRoleRequest, 0, len(roles))

	for _, role := range roles {
		requestRole := infisical.OrgGroupMembershipRoleRequest{
			Role:        role.RoleSlug.ValueString(),
			IsTemporary: role.IsTemporary.ValueBool(),
		}

		if requestRole.IsTemporary {
			if role.TemporaryAccessStartTime.ValueString() == "" {
				diags.AddError(
					"Field temporary_access_start_time is required for temporary roles",
					fmt.Sprintf("Must provide a valid ISO timestamp (YYYY-MM-DDTHH:MM:SSZ) for field temporary_access_start_time, role %s", role.RoleSlug.ValueString()),
				)
				continue
			}

			startTime, err := time.Parse(time.RFC3339, role.TemporaryAccessStartTime.ValueString())
			if err != nil {
				diags.AddError(
					"Error parsing field temporary_access_start_time",
					fmt.Sprintf("Must provide a valid ISO timestamp for field temporary_access_start_time %s, role %s", role.TemporaryAccessStartTime.ValueString(), role.RoleSlug.ValueString()),
				)
				continue
			}

			requestRole.TemporaryMode = TEMPORARY_MODE_RELATIVE
			requestRole.TemporaryRange = role.TemporaryRange.ValueString()
			if requestRole.TemporaryRange == "" {
				requestRole.TemporaryRange = TEMPORARY_RANGE_DEFAULT
			}
			requestRole.TemporaryAccessStartTime = &startTime
		}

		requestRoles = append(requestRoles, requestRole)
	}

	return requestRoles, diags
}

// groupAssignmentRolesFromAPI maps the API roles back into the model. Values the API fills in
// with defaults are kept null when the prior state left them unset, to avoid perpetual drift.
func groupAssignmentRolesFromAPI(apiRoles []infisical.OrgGroupMembershipRole, priorRoles []groupAssignmentRoleModel) []groupAssignmentRoleModel {
	priorBySlug := make(map[string]groupAssignmentRoleModel, len(priorRoles))
	for _, role := range priorRoles {
		priorBySlug[role.RoleSlug.ValueString()] = role
	}

	roles := make([]groupAssignmentRoleModel, 0, len(apiRoles))
	for _, apiRole := range apiRoles {
		roleSlug := apiRole.Role
		if apiRole.Role == "custom" && apiRole.CustomRoleSlug != nil && *apiRole.CustomRoleSlug != "" {
			roleSlug = *apiRole.CustomRoleSlug
		}

		role := groupAssignmentRoleModel{
			RoleSlug:                 types.StringValue(roleSlug),
			IsTemporary:              types.BoolValue(apiRole.IsTemporary),
			TemporaryRange:           types.StringNull(),
			TemporaryAccessStartTime: types.StringNull(),
		}

		if apiRole.IsTemporary {
			if apiRole.TemporaryRange != nil {
				role.TemporaryRange = types.StringValue(*apiRole.TemporaryRange)
			}
			if apiRole.TemporaryAccessStartTime != nil {
				role.TemporaryAccessStartTime = types.StringValue(apiRole.TemporaryAccessStartTime.UTC().Format(time.RFC3339))
			}

			prior, ok := priorBySlug[roleSlug]
			if ok && prior.TemporaryRange.IsNull() && role.TemporaryRange.ValueString() == TEMPORARY_RANGE_DEFAULT {
				role.TemporaryRange = types.StringNull()
			}
		}

		roles = append(roles, role)
	}

	return roles
}

func setGroupAssignmentComputed(model *groupAssignmentResourceModel, membership infisical.OrgGroupMembership) {
	model.ID = types.StringValue(membership.ID)
	model.GroupID = types.StringValue(membership.GroupID)
	model.GroupSlug = types.StringValue(membership.Group.Slug)
	model.GroupName = types.StringValue(membership.Group.Name)
	model.GroupOrgID = types.StringValue(membership.Group.OrgID)
}

// Create links the group into the sub-organization.
func (r *groupAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create group assignment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan groupAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, diags := buildGroupAssignmentRoles(plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := ""
	if !plan.GroupID.IsUnknown() && !plan.GroupID.IsNull() {
		groupID = plan.GroupID.ValueString()
	}
	groupSlug := ""
	if !plan.GroupSlug.IsUnknown() && !plan.GroupSlug.IsNull() {
		groupSlug = plan.GroupSlug.ValueString()
	}

	groupRef := groupID
	if groupRef == "" {
		groupRef = groupSlug
	}

	// Validate the group against what the sub-organization can link from its parent.
	availableGroup, err := r.client.GetAvailableGroup(groupID, groupSlug)
	if err != nil {
		if !errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Error listing groups available to the sub-organization",
				"Couldn't list the parent-organization groups available for linking. Ensure the provider is scoped to the target sub-organization through auth.organization_slug and that the machine identity may manage groups there.\n\n"+
					"Infisical Client Error: "+err.Error(),
			)
			return
		}

		// Not available: distinguish an already-linked group from one that cannot be linked at all.
		var existing infisical.OrgGroupMembership
		var lookupErr error
		if groupID != "" {
			existing, lookupErr = r.client.GetOrgGroupMembership(groupID)
		} else {
			existing, lookupErr = r.client.GetOrgGroupMembershipBySlug(groupSlug)
		}
		if lookupErr == nil {
			resp.Diagnostics.AddError(
				"Group is already linked to the sub-organization",
				fmt.Sprintf("Group %s is already linked to this organization. To manage it with Terraform, import it: terraform import <resource address> %s", groupRef, existing.GroupID),
			)
			return
		}

		resp.Diagnostics.AddError(
			"Group is not available to the sub-organization",
			fmt.Sprintf("Group %s is not among the parent-organization groups this organization can link. Check that the group exists in the root organization and that the provider is scoped to a sub-organization (auth.organization_slug), not to the root organization that owns the group.", groupRef),
		)
		return
	}

	membership, err := r.client.CreateOrgGroupMembership(infisical.CreateOrgGroupMembershipRequest{
		GroupID: availableGroup.ID,
		Roles:   roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error linking group to sub-organization",
			"Couldn't link group to the sub-organization in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	// Fall back to the validated group for any fields the create response leaves out.
	if membership.GroupID == "" {
		membership.GroupID = availableGroup.ID
	}
	if membership.Group.Slug == "" {
		membership.Group.Slug = availableGroup.Slug
	}
	if membership.Group.Name == "" {
		membership.Group.Name = availableGroup.Name
	}

	setGroupAssignmentComputed(&plan, membership)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *groupAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read group assignment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state groupAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	membership, err := r.client.GetOrgGroupMembership(state.GroupID.ValueString())
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading group assignment",
			"Couldn't read the group's sub-organization membership from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	setGroupAssignmentComputed(&state, membership)
	state.Roles = groupAssignmentRolesFromAPI(membership.Roles, state.Roles)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update replaces the roles assigned to the linked group.
func (r *groupAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update group assignment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan groupAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state groupAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, diags := buildGroupAssignmentRoles(plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateOrgGroupMembership(infisical.UpdateOrgGroupMembershipRequest{
		GroupID: state.GroupID.ValueString(),
		Roles:   roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating group assignment",
			"Couldn't update the group's roles in the sub-organization, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete unlinks the group from the sub-organization.
func (r *groupAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete group assignment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state groupAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgGroupMembership(state.GroupID.ValueString())
	if err != nil {
		_, getErr := r.client.GetOrgGroupMembership(state.GroupID.ValueString())
		if errors.Is(getErr, infisical.ErrNotFound) {
			return
		}

		errMsg := "Couldn't unlink group from the sub-organization, unexpected error: " + err.Error()
		if getErr != nil {
			errMsg += "\nAdditionally, verifying whether the group is still linked failed: " + getErr.Error()
		}
		resp.Diagnostics.AddError("Error deleting group assignment", errMsg)
	}
}

// ImportState imports an existing link by the ID of the linked group.
func (r *groupAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import group assignment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("group_id"), req, resp)
}
