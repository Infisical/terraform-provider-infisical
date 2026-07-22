package resource

import (
	"context"
	"fmt"
	"sort"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &ProjectScopedIdentityResource{}
	_ resource.ResourceWithImportState = &ProjectScopedIdentityResource{}
)

// NewProjectScopedIdentityResource is a helper function to simplify the provider implementation.
func NewProjectScopedIdentityResource() resource.Resource {
	return &ProjectScopedIdentityResource{}
}

// ProjectScopedIdentityResource is the resource implementation.
type ProjectScopedIdentityResource struct {
	client *infisical.Client
}

// ProjectScopedIdentityResourceModel describes the resource data model.
type ProjectScopedIdentityResourceModel struct {
	ID                  types.String                `tfsdk:"id"`
	ProjectID           types.String                `tfsdk:"project_id"`
	Name                types.String                `tfsdk:"name"`
	HasDeleteProtection types.Bool                  `tfsdk:"has_delete_protection"`
	AuthMethods         types.List                  `tfsdk:"auth_methods"`
	Metadata            []MetaEntry                 `tfsdk:"metadata"`
	Roles               []ProjectScopedIdentityRole `tfsdk:"roles"`
}

// ProjectScopedIdentityRole describes a role of a Project Scoped Identity
type ProjectScopedIdentityRole struct {
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
func (r *ProjectScopedIdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_scoped_identity"
}

// Schema defines the schema for the resource.
func (r *ProjectScopedIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage a machine identity scoped to a single project in Infisical, including its project role(s). Project-scoped identities are bound to one project and cannot be assigned to other projects. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the project-scoped identity.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the project that owns this identity.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the identity.",
				Required:    true,
			},
			"has_delete_protection": schema.BoolAttribute{
				Description: "Whether the identity has delete protection enabled. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"auth_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "The authentication methods configured for the identity.",
				Computed:    true,
			},
			"metadata": schema.SetNestedAttribute{
				Description: "The metadata associated with this identity.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The key of the metadata entry.",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the metadata entry.",
							Required:    true,
						},
					},
				},
			},
			"roles": schema.ListNestedAttribute{
				Required:    true,
				Description: "The roles assigned to the project-scoped identity. At least one permanent (non-temporary) role is required.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the project identity role.",
							Computed:    true,
						},
						"role_slug": schema.StringAttribute{
							Description: "The slug of the role. To assign a custom role, set this to the custom role's slug.",
							Required:    true,
						},
						"custom_role_id": schema.StringAttribute{
							// Computed-only: this endpoint identifies custom roles by slug (returned in
							// role_slug), not id, so custom_role_id is informational and cannot be set.
							Description: "The id of the custom role. Read-only; reference custom roles via role_slug.",
							Computed:    true,
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

// orderAPIScopedIdentityRolesByPlan reorders API-returned roles to match the ordering specified in the Terraform plan.
// The API may return roles in a non-deterministic order, but Terraform requires list elements to maintain
// the same positional order as the plan to avoid "inconsistent result after apply" errors.
func orderAPIScopedIdentityRolesByPlan(planRoles []ProjectScopedIdentityRole, apiRoles []ProjectScopedIdentityRole) []ProjectScopedIdentityRole {
	apiRoleMap := make(map[string]ProjectScopedIdentityRole, len(apiRoles))
	for _, role := range apiRoles {
		apiRoleMap[role.RoleSlug.ValueString()] = role
	}

	ordered := make([]ProjectScopedIdentityRole, 0, len(apiRoles))
	matched := make(map[string]bool)

	// First, add roles in the order they appear in the plan (deduplicating by slug)
	for _, planRole := range planRoles {
		slug := planRole.RoleSlug.ValueString()
		if apiRole, ok := apiRoleMap[slug]; ok && !matched[slug] {
			ordered = append(ordered, apiRole)
			matched[slug] = true
		}
	}

	// Then, append any remaining API roles not present in the plan (sorted for determinism)
	var remaining []ProjectScopedIdentityRole
	for _, apiRole := range apiRoles {
		if !matched[apiRole.RoleSlug.ValueString()] {
			remaining = append(remaining, apiRole)
		}
	}
	sort.Slice(remaining, func(i, j int) bool {
		return remaining[i].RoleSlug.ValueString() < remaining[j].RoleSlug.ValueString()
	})
	ordered = append(ordered, remaining...)

	return ordered
}

// buildProjectIdentityRequestRoles converts the roles declared in a Terraform plan into the
// API request representation. It applies the defaults used across the provider for temporary
// roles (relative mode, 1h range) and validates the role list: at least one permanent role is
// required, role slugs must be unique, temporary-only fields may not be set on permanent roles,
// and temporary_access_start_time must be in canonical RFC3339 UTC form.
func buildProjectScopedIdentityRequestRoles(planRoles []ProjectScopedIdentityRole) ([]infisical.CreateProjectIdentityRequestRoles, error) {
	var roles []infisical.CreateProjectIdentityRequestRoles
	var hasAtleastOnePermanentRole bool
	seenSlugs := make(map[string]bool, len(planRoles))
	for _, el := range planRoles {
		slug := el.RoleSlug.ValueString()
		if seenSlugs[slug] {
			return nil, fmt.Errorf("duplicate role_slug %q: each role may only appear once", slug)
		}
		seenSlugs[slug] = true

		isTemporary := el.IsTemporary.ValueBool()
		temporaryMode := el.TemporaryMode.ValueString()
		temporaryRange := el.TemporaryRange.ValueString()
		temporaryAccesStartTime := time.Now().UTC()

		if !isTemporary {
			hasAtleastOnePermanentRole = true
			if temporaryMode != "" || temporaryRange != "" || el.TemporaryAccesStartTime.ValueString() != "" {
				return nil, fmt.Errorf("role %q is permanent (is_temporary = false) but sets temporary_mode/temporary_range/temporary_access_start_time; set is_temporary = true or remove those fields", slug)
			}
		}

		if el.TemporaryAccesStartTime.ValueString() != "" {
			raw := el.TemporaryAccesStartTime.ValueString()
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return nil, fmt.Errorf("must provide valid ISO timestamp for field temporary_access_start_time %q, role %q", raw, slug)
			}
			// The value is stored back from the API in canonical UTC RFC3339 (…Z). Require that
			// form on input so the planned value matches the applied value.
			if canonical := parsed.UTC().Format(time.RFC3339); canonical != raw {
				return nil, fmt.Errorf("temporary_access_start_time %q for role %q must be in canonical RFC3339 UTC format (e.g. %q)", raw, slug, canonical)
			}
			temporaryAccesStartTime = parsed
		}

		// default values
		if isTemporary && temporaryMode == "" {
			temporaryMode = TEMPORARY_MODE_RELATIVE
		}
		if isTemporary && temporaryRange == "" {
			temporaryRange = "1h"
		}

		roles = append(roles, infisical.CreateProjectIdentityRequestRoles{
			Role:                     slug,
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
	}

	if !hasAtleastOnePermanentRole {
		return nil, fmt.Errorf("must have atleast one permanent role")
	}

	return roles, nil
}

// mapAPIRolesToIdentityModel converts API project member roles into the Terraform model
// representation, resolving custom role slugs and nulling temporary-only fields for permanent roles.
func mapAPIRolesToScopedIdentityModel(apiRoles []infisical.ProjectMemberRole) []ProjectScopedIdentityRole {
	result := make([]ProjectScopedIdentityRole, 0, len(apiRoles))
	for _, el := range apiRoles {
		val := ProjectScopedIdentityRole{
			ID:                      types.StringValue(el.ID),
			RoleSlug:                types.StringValue(el.Role),
			TemporaryAccessEndTime:  types.StringValue(el.TemporaryAccessEndTime.Format(time.RFC3339)),
			TemporaryRange:          types.StringValue(el.TemporaryRange),
			TemporaryMode:           types.StringValue(el.TemporaryMode),
			CustomRoleID:            types.StringValue(el.CustomRoleId),
			IsTemporary:             types.BoolValue(el.IsTemporary),
			TemporaryAccesStartTime: types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339)),
		}
		// For a custom role the API returns role = "custom" and the real slug in
		// customRoleSlug (v1 omits customRoleId entirely, v2 includes it).
		if el.CustomRoleSlug != "" {
			val.RoleSlug = types.StringValue(el.CustomRoleSlug)
		}
		if !el.IsTemporary {
			val.TemporaryMode = types.StringNull()
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccesStartTime = types.StringNull()
			val.TemporaryAccessEndTime = types.StringNull()
		}
		result = append(result, val)
	}
	return result
}

// Configure adds the provider configured client to the resource.
func (r *ProjectScopedIdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *ProjectScopedIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project-scoped identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan ProjectScopedIdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := buildProjectScopedIdentityRequestRoles(plan.Roles)
	if err != nil {
		resp.Diagnostics.AddError("Error assigning role to identity", err.Error())
		return
	}

	// The identity and its roles are created in a single atomic call: if a role is
	// invalid the backend rejects the request without creating an orphan identity.
	identity, err := r.client.CreateProjectScopedIdentity(infisical.CreateProjectScopedIdentityRequest{
		ProjectID:           plan.ProjectID.ValueString(),
		Name:                plan.Name.ValueString(),
		HasDeleteProtection: plan.HasDeleteProtection.ValueBool(),
		Metadata:            buildMetadataFromPlan(plan.Metadata),
		Roles:               roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project-scoped identity",
			"Couldn't create project-scoped identity in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	// The create response only returns the identity, so read the membership back to
	// populate the computed role fields (ids, custom role slug, temporary end time).
	membership, err := r.client.GetProjectScopedIdentityMembership(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  plan.ProjectID.ValueString(),
		IdentityID: identity.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project-scoped identity roles",
			"Couldn't read the project-scoped identity roles from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(identity.ID)
	plan.HasDeleteProtection = types.BoolValue(identity.HasDeleteProtection)
	plan.AuthMethods = buildAuthMethodsList(identity.AuthMethods)
	if plan.Metadata != nil {
		plan.Metadata = metadataFromAPI(identity.Metadata)
	}
	plan.Roles = orderAPIScopedIdentityRolesByPlan(plan.Roles, mapAPIRolesToScopedIdentityModel(membership.Membership.Roles))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ProjectScopedIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project-scoped identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state ProjectScopedIdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" || state.ProjectID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	identity, err := r.client.GetProjectScopedIdentityByID(infisical.GetProjectScopedIdentityRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading project-scoped identity",
			"Couldn't read project-scoped identity from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	membership, err := r.client.GetProjectScopedIdentityMembership(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading project-scoped identity roles",
			"Couldn't read the project-scoped identity roles from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(identity.Name)
	state.HasDeleteProtection = types.BoolValue(identity.HasDeleteProtection)
	state.AuthMethods = buildAuthMethodsList(identity.AuthMethods)
	if state.Metadata != nil {
		state.Metadata = metadataFromAPI(identity.Metadata)
	}
	state.Roles = orderAPIScopedIdentityRolesByPlan(state.Roles, mapAPIRolesToScopedIdentityModel(membership.Membership.Roles))

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ProjectScopedIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project-scoped identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan ProjectScopedIdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectScopedIdentityResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := buildProjectScopedIdentityRequestRoles(plan.Roles)
	if err != nil {
		resp.Diagnostics.AddError("Error assigning role to identity", err.Error())
		return
	}

	identity, err := r.client.UpdateProjectScopedIdentity(infisical.UpdateProjectScopedIdentityRequest{
		ProjectID:           state.ProjectID.ValueString(),
		IdentityID:          state.ID.ValueString(),
		Name:                plan.Name.ValueString(),
		HasDeleteProtection: plan.HasDeleteProtection.ValueBool(),
		Metadata:            buildMetadataFromPlan(plan.Metadata),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project-scoped identity",
			"Couldn't update project-scoped identity in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	if err := r.client.UpdateProjectScopedIdentityRoles(infisical.UpdateProjectScopedIdentityRolesRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.ID.ValueString(),
		Roles:      roles,
	}); err != nil {
		plan.ID = state.ID
		plan.HasDeleteProtection = types.BoolValue(identity.HasDeleteProtection)
		plan.AuthMethods = buildAuthMethodsList(identity.AuthMethods)
		if plan.Metadata != nil {
			plan.Metadata = metadataFromAPI(identity.Metadata)
		}
		if current, readErr := r.client.GetProjectScopedIdentityMembership(infisical.GetProjectIdentityByIDRequest{
			ProjectID:  state.ProjectID.ValueString(),
			IdentityID: state.ID.ValueString(),
		}); readErr == nil {
			plan.Roles = orderAPIScopedIdentityRolesByPlan(state.Roles, mapAPIRolesToScopedIdentityModel(current.Membership.Roles))
		} else {
			plan.Roles = state.Roles
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError(
			"Error assigning roles to project-scoped identity",
			"Couldn't update the project-scoped identity roles in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	membership, err := r.client.GetProjectScopedIdentityMembership(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project-scoped identity roles",
			"Couldn't read the project-scoped identity roles from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID
	plan.HasDeleteProtection = types.BoolValue(identity.HasDeleteProtection)
	plan.AuthMethods = buildAuthMethodsList(identity.AuthMethods)
	if plan.Metadata != nil {
		plan.Metadata = metadataFromAPI(identity.Metadata)
	}
	plan.Roles = orderAPIScopedIdentityRolesByPlan(plan.Roles, mapAPIRolesToScopedIdentityModel(membership.Membership.Roles))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ProjectScopedIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project-scoped identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state ProjectScopedIdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectScopedIdentity(infisical.DeleteProjectScopedIdentityRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting project-scoped identity",
			"Couldn't delete project-scoped identity from Infisical, unexpected error: "+err.Error(),
		)
	}
}

// ImportState restores state from a <project_id>,<identity_id> import ID.
func (r *ProjectScopedIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import project-scoped identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format <project_id>,<identity_id>",
		)
		return
	}

	projectID := parts[0]
	identityID := parts[1]

	identity, err := r.client.GetProjectScopedIdentityByID(infisical.GetProjectScopedIdentityRequest{
		ProjectID:  projectID,
		IdentityID: identityID,
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Error importing project-scoped identity",
				fmt.Sprintf("No project-scoped identity found with project_id=%s and id=%s", projectID, identityID),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error importing project-scoped identity",
			"Couldn't read project-scoped identity from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	membership, err := r.client.GetProjectScopedIdentityMembership(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  projectID,
		IdentityID: identityID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing project-scoped identity",
			"Couldn't read the project-scoped identity roles from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	// Leave metadata null (rather than an empty list) when the identity has none, so an
	// imported resource whose config omits the metadata block does not show spurious drift.
	var metadata []MetaEntry
	if len(identity.Metadata) > 0 {
		metadata = metadataFromAPI(identity.Metadata)
	}

	// Sort roles by slug for a deterministic order on import (there is no plan to align
	// against), matching the behaviour of the infisical_project_identity resource.
	roles := mapAPIRolesToScopedIdentityModel(membership.Membership.Roles)
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].RoleSlug.ValueString() < roles[j].RoleSlug.ValueString()
	})

	state := ProjectScopedIdentityResourceModel{
		ID:                  types.StringValue(identity.ID),
		ProjectID:           types.StringValue(identity.ProjectID),
		Name:                types.StringValue(identity.Name),
		HasDeleteProtection: types.BoolValue(identity.HasDeleteProtection),
		AuthMethods:         buildAuthMethodsList(identity.AuthMethods),
		Metadata:            metadata,
		Roles:               roles,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// buildMetadataFromPlan converts the plan metadata slice to CreateMetaEntry slice.
func buildMetadataFromPlan(entries []MetaEntry) []infisical.CreateMetaEntry {
	result := []infisical.CreateMetaEntry{}
	for _, e := range entries {
		result = append(result, infisical.CreateMetaEntry{
			Key:   e.Key.ValueString(),
			Value: e.Value.ValueString(),
		})
	}
	return result
}

// buildAuthMethodsList converts a string slice to a Terraform list value.
func buildAuthMethodsList(methods []string) types.List {
	if len(methods) == 0 {
		return types.ListNull(types.StringType)
	}
	elements := make([]attr.Value, len(methods))
	for i, m := range methods {
		elements[i] = types.StringValue(m)
	}
	return types.ListValueMust(types.StringType, elements)
}

// metadataFromAPI converts API metadata into the Terraform model representation, returning an
// empty (non-nil) slice when there is no metadata to keep plan/state consistent.
func metadataFromAPI(apiMetadata []infisical.MetaEntry) []MetaEntry {
	converted := make([]MetaEntry, len(apiMetadata))
	for i, m := range apiMetadata {
		converted[i] = MetaEntry{
			Key:   types.StringValue(m.Key),
			Value: types.StringValue(m.Value),
		}
	}
	return converted
}
