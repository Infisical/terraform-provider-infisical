package resource

import (
	"context"
	"errors"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                     = &subOrganizationGroupResource{}
	_ resource.ResourceWithImportState      = &subOrganizationGroupResource{}
	_ resource.ResourceWithConfigValidators = &subOrganizationGroupResource{}
	_ resource.ResourceWithValidateConfig   = &subOrganizationGroupResource{}
)

func NewSubOrganizationGroupResource() resource.Resource {
	return &subOrganizationGroupResource{}
}

type subOrganizationGroupResource struct {
	client *infisical.Client
}

type subOrganizationGroupResourceModel struct {
	ID        types.String               `tfsdk:"id"`
	GroupID   types.String               `tfsdk:"group_id"`
	GroupSlug types.String               `tfsdk:"group_slug"`
	GroupName types.String               `tfsdk:"group_name"`
	Roles     []subOrganizationGroupRole `tfsdk:"roles"`
}

type subOrganizationGroupRole struct {
	RoleSlug                 types.String `tfsdk:"role_slug"`
	IsTemporary              types.Bool   `tfsdk:"is_temporary"`
	TemporaryRange           types.String `tfsdk:"temporary_range"`
	TemporaryAccessStartTime types.String `tfsdk:"temporary_access_start_time"`
}

func (r *subOrganizationGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sub_organization_group"
}

func (r *subOrganizationGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Link a group from the root organization into a sub-organization and assign it organization roles there. " +
			"The group is linked into the organization the provider is scoped to, so the provider must be scoped to the target sub-organization through `auth.organization_slug`. " +
			"The machine identity needs the `Link Group` permission on sub-organizations in the root organization, and permission to manage groups in the sub-organization. " +
			"Destroying this resource unlinks the group from the sub-organization; the group itself is left untouched in the root organization. " +
			"Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the group's membership in the sub-organization.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": schema.StringAttribute{
				Description:   "The ID of the root-organization group to link. Exactly one of `group_id` or `group_slug` must be set.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"group_slug": schema.StringAttribute{
				Description:   "The slug of the root-organization group to link. Exactly one of `group_id` or `group_slug` must be set.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"group_name": schema.StringAttribute{
				Description:   "The name of the linked group.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"roles": schema.SetNestedAttribute{
				Description: "The organization roles assigned to the group within the sub-organization.",
				Required:    true,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_slug": schema.StringAttribute{
							Description: "The slug of the organization role, e.g. `admin`, `member`, `no-access`, or the slug of a custom role of the sub-organization.",
							Required:    true,
						},
						"is_temporary": schema.BoolAttribute{
							Description: "Flag to indicate the assigned role is temporary or not. When is_temporary is true, temporary_access_start_time is required.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
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

func (r *subOrganizationGroupResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("group_id"),
			path.MatchRoot("group_slug"),
		),
	}
}

// Catches role mistakes at plan time. Temporary fields on a permanent role never reach the API,
// so state would keep them while every refresh clears them and the plan never settles.
func (r *subOrganizationGroupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var roles types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("roles"), &roles)...)
	if resp.Diagnostics.HasError() || roles.IsNull() || roles.IsUnknown() {
		return
	}

	for _, element := range roles.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsUnknown() || object.IsNull() {
			continue
		}

		var role subOrganizationGroupRole
		resp.Diagnostics.Append(object.As(ctx, &role, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		if role.IsTemporary.IsUnknown() {
			continue
		}

		rolePath := path.Root("roles").AtSetValue(element)
		if role.IsTemporary.ValueBool() {
			if role.TemporaryAccessStartTime.IsNull() {
				resp.Diagnostics.AddAttributeError(
					rolePath,
					"Field temporary_access_start_time is required for temporary roles",
					fmt.Sprintf("Must provide valid ISO timestamp (YYYY-MM-DDTHH:MM:SSZ) for field temporary_access_start_time, role %s", role.RoleSlug.ValueString()),
				)
			}
			continue
		}

		if isKnownAndSet(role.TemporaryRange) || isKnownAndSet(role.TemporaryAccessStartTime) {
			resp.Diagnostics.AddAttributeError(rolePath, permanentRoleTemporaryFieldsSummary, permanentRoleTemporaryFieldsDetail(role))
		}
	}
}

func isKnownAndSet(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown()
}

const permanentRoleTemporaryFieldsSummary = "Temporary fields set on a permanent role"

func permanentRoleTemporaryFieldsDetail(role subOrganizationGroupRole) string {
	return fmt.Sprintf("Role %s isn't temporary, so temporary_range and temporary_access_start_time do nothing. Set is_temporary = true or remove them.", role.RoleSlug.ValueString())
}

func (r *subOrganizationGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildSubOrganizationGroupRoles(roles []subOrganizationGroupRole) ([]infisical.OrgGroupMembershipRoleRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	requestRoles := make([]infisical.OrgGroupMembershipRoleRequest, 0, len(roles))

	for _, role := range roles {
		requestRole := infisical.OrgGroupMembershipRoleRequest{
			Role:        role.RoleSlug.ValueString(),
			IsTemporary: role.IsTemporary.ValueBool(),
		}

		if !requestRole.IsTemporary && (!role.TemporaryRange.IsNull() || !role.TemporaryAccessStartTime.IsNull()) {
			diags.AddError(permanentRoleTemporaryFieldsSummary, permanentRoleTemporaryFieldsDetail(role))
			continue
		}

		if requestRole.IsTemporary {
			if role.TemporaryAccessStartTime.ValueString() == "" {
				diags.AddError(
					"Field temporary_access_start_time is required for temporary roles",
					fmt.Sprintf("Must provide valid ISO timestamp (YYYY-MM-DDTHH:MM:SSZ) for field temporary_access_start_time, role %s", role.RoleSlug.ValueString()),
				)
				continue
			}

			startTime, err := time.Parse(time.RFC3339, role.TemporaryAccessStartTime.ValueString())
			if err != nil {
				diags.AddError(
					"Error parsing field temporary_access_start_time",
					fmt.Sprintf("Must provide valid ISO timestamp for field temporary_access_start_time %s, role %s", role.TemporaryAccessStartTime.ValueString(), role.RoleSlug.ValueString()),
				)
				continue
			}

			requestRole.TemporaryMode = TEMPORARY_MODE_RELATIVE
			requestRole.TemporaryRange = role.TemporaryRange.ValueString()
			if requestRole.TemporaryRange == "" {
				requestRole.TemporaryRange = TEMPORARY_RANGE_DEFAULT
			}
			// The API only takes UTC, an offset like +02:00 gets a 422.
			startTime = startTime.UTC()
			requestRole.TemporaryAccessStartTime = &startTime
		}

		requestRoles = append(requestRoles, requestRole)
	}

	return requestRoles, diags
}

// Keeps the user's form for values the API fills in or reformats (default range, start time
// offset), otherwise every plan shows drift.
func subOrganizationGroupRolesFromAPI(apiRoles []infisical.OrgGroupMembershipRole, priorRoles []subOrganizationGroupRole) []subOrganizationGroupRole {
	priorBySlug := make(map[string]subOrganizationGroupRole, len(priorRoles))
	for _, role := range priorRoles {
		priorBySlug[role.RoleSlug.ValueString()] = role
	}

	roles := make([]subOrganizationGroupRole, 0, len(apiRoles))
	for _, apiRole := range apiRoles {
		roleSlug := apiRole.Role
		if apiRole.CustomRoleSlug != nil && *apiRole.CustomRoleSlug != "" {
			roleSlug = *apiRole.CustomRoleSlug
		}

		role := subOrganizationGroupRole{
			RoleSlug:                 types.StringValue(roleSlug),
			IsTemporary:              types.BoolValue(apiRole.IsTemporary),
			TemporaryRange:           types.StringNull(),
			TemporaryAccessStartTime: types.StringNull(),
		}

		if apiRole.IsTemporary {
			prior, hasPrior := priorBySlug[roleSlug]

			if apiRole.TemporaryRange != nil {
				role.TemporaryRange = types.StringValue(*apiRole.TemporaryRange)
				if hasPrior && prior.TemporaryRange.IsNull() && *apiRole.TemporaryRange == TEMPORARY_RANGE_DEFAULT {
					role.TemporaryRange = types.StringNull()
				}
			}

			if apiRole.TemporaryAccessStartTime != nil {
				role.TemporaryAccessStartTime = types.StringValue(apiRole.TemporaryAccessStartTime.UTC().Format(time.RFC3339))
				if hasPrior {
					priorStart, err := time.Parse(time.RFC3339, prior.TemporaryAccessStartTime.ValueString())
					if err == nil && priorStart.Equal(*apiRole.TemporaryAccessStartTime) {
						role.TemporaryAccessStartTime = prior.TemporaryAccessStartTime
					}
				}
			}
		}

		roles = append(roles, role)
	}

	return roles
}

func setSubOrganizationGroupComputed(model *subOrganizationGroupResourceModel, membership infisical.OrgGroupMembership) {
	model.ID = types.StringValue(membership.ID)
	model.GroupID = types.StringValue(membership.GroupID)
	model.GroupSlug = types.StringValue(membership.Group.Slug)
	model.GroupName = types.StringValue(membership.Group.Name)
}

// True when the group is owned by the session's own org (e.g. provider scoped to root). That's not
// a link, so this resource shouldn't touch it. A failed lookup is returned instead of guessed,
// since the API would happily delete a native membership on destroy.
func (r *subOrganizationGroupResource) isNativeMembership(membership infisical.OrgGroupMembership) (bool, error) {
	sessionOrgID, err := r.client.GetSessionOrganizationID()
	if err != nil {
		return false, err
	}
	return membership.Group.OrgID == sessionOrgID, nil
}

func nativeGroupError(diags *diag.Diagnostics, groupRef string) {
	diags.AddError(
		"Group belongs to the organization the provider is scoped to",
		fmt.Sprintf("Group %s is owned by the organization the provider is scoped to, so there is nothing to link. Groups can only be linked from the root organization into a sub-organization: set auth.organization_slug to the slug of the target sub-organization.", groupRef),
	)
}

// The group is either native to the session's org or was linked outside Terraform. Only the
// second one gets the import hint.
func (r *subOrganizationGroupResource) existingMembershipError(diags *diag.Diagnostics, groupRef string, membership infisical.OrgGroupMembership) {
	native, err := r.isNativeMembership(membership)
	if err != nil {
		diags.AddError(
			"Group already has a membership in the organization",
			fmt.Sprintf("Group %s already has a membership in the organization the provider is scoped to, but couldn't tell whether it was linked or belongs to that organization: %s", groupRef, err.Error()),
		)
		return
	}
	if native {
		nativeGroupError(diags, groupRef)
		return
	}

	diags.AddError(
		"Group is already linked to the sub-organization",
		fmt.Sprintf("Group %s is already linked to the organization the provider is scoped to. To manage it with Terraform, import it: terraform import <resource address> %s", groupRef, membership.GroupID),
	)
}

func (r *subOrganizationGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create sub-organization group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan subOrganizationGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, diags := buildSubOrganizationGroupRoles(plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := plan.GroupID.ValueString()
	groupRef := groupID

	if plan.GroupID.IsUnknown() || plan.GroupID.IsNull() {
		groupSlug := plan.GroupSlug.ValueString()
		groupRef = groupSlug

		availableGroup, err := r.client.GetAvailableGroupBySlug(groupSlug)
		if err != nil {
			if !errors.Is(err, infisical.ErrNotFound) {
				resp.Diagnostics.AddError(
					"Error listing groups available to the sub-organization",
					"Couldn't list the root-organization groups available for linking, unexpected error: "+err.Error(),
				)
				return
			}

			if existing, lookupErr := r.client.GetOrgGroupMembershipBySlug(groupSlug); lookupErr == nil {
				r.existingMembershipError(&resp.Diagnostics, groupRef, existing)
				return
			}

			resp.Diagnostics.AddError(
				"Group not found",
				fmt.Sprintf("No root-organization group with slug %s is available to link. Check that the group exists in the root organization and that the provider is scoped to a sub-organization through auth.organization_slug.", groupSlug),
			)
			return
		}

		groupID = availableGroup.ID
	}

	membership, err := r.client.CreateOrgGroupMembership(infisical.CreateOrgGroupMembershipRequest{
		GroupID: groupID,
		Roles:   roles,
	})
	if err != nil {
		if existing, lookupErr := r.client.GetOrgGroupMembership(groupID); lookupErr == nil {
			r.existingMembershipError(&resp.Diagnostics, groupRef, existing)
			return
		}

		resp.Diagnostics.AddError(
			"Error linking group to sub-organization",
			"Couldn't link the group to the sub-organization, unexpected error: "+err.Error(),
		)
		return
	}

	setSubOrganizationGroupComputed(&plan, membership)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *subOrganizationGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read sub-organization group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state subOrganizationGroupResourceModel
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
			"Error reading sub-organization group",
			"Couldn't read the group's sub-organization membership, unexpected error: "+err.Error(),
		)
		return
	}

	setSubOrganizationGroupComputed(&state, membership)
	state.Roles = subOrganizationGroupRolesFromAPI(membership.Roles, state.Roles)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *subOrganizationGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update sub-organization group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan subOrganizationGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state subOrganizationGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, diags := buildSubOrganizationGroupRoles(plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	membership, err := r.client.UpdateOrgGroupMembership(infisical.UpdateOrgGroupMembershipRequest{
		GroupID: state.GroupID.ValueString(),
		Roles:   roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating sub-organization group",
			"Couldn't update the group's roles in the sub-organization, unexpected error: "+err.Error(),
		)
		return
	}

	setSubOrganizationGroupComputed(&plan, membership)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *subOrganizationGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete sub-organization group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state subOrganizationGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgGroupMembership(state.GroupID.ValueString())
	if err == nil || errors.Is(err, infisical.ErrNotFound) {
		return
	}

	// Unlinked groups come back as a 400, not a 404, so double check the link is really gone.
	if _, getErr := r.client.GetOrgGroupMembership(state.GroupID.ValueString()); errors.Is(getErr, infisical.ErrNotFound) {
		return
	}

	resp.Diagnostics.AddError(
		"Error deleting sub-organization group",
		"Couldn't unlink the group from the sub-organization, unexpected error: "+err.Error(),
	)
}

// Takes either the group ID or its slug.
func (r *subOrganizationGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import sub-organization group",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var membership infisical.OrgGroupMembership
	var err error
	if _, parseErr := uuid.Parse(req.ID); parseErr == nil {
		membership, err = r.client.GetOrgGroupMembership(req.ID)
	} else {
		membership, err = r.client.GetOrgGroupMembershipBySlug(req.ID)
	}

	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Group not found",
				fmt.Sprintf("No group %s is linked to the organization the provider is scoped to.", req.ID),
			)
			return
		}

		resp.Diagnostics.AddError(
			"Error importing sub-organization group",
			"Couldn't read the group's sub-organization membership, unexpected error: "+err.Error(),
		)
		return
	}

	native, err := r.isNativeMembership(membership)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing sub-organization group",
			"Couldn't verify the group is linked from the root organization, unexpected error: "+err.Error(),
		)
		return
	}
	if native {
		nativeGroupError(&resp.Diagnostics, req.ID)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), membership.GroupID)...)
}
