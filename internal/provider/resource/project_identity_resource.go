package resource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	_ resource.Resource                = &ProjectIdentityResource{}
	_ resource.ResourceWithImportState = &ProjectIdentityResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectIdentityResource() resource.Resource {
	return &ProjectIdentityResource{}
}

// ProjectIdentityResource is the resource implementation.
type ProjectIdentityResource struct {
	client *infisical.Client
}

// projectResourceSourceModel describes the data source data model.
type ProjectIdentityResourceModel struct {
	ProjectID     types.String          `tfsdk:"project_id"`
	IdentityID    types.String          `tfsdk:"identity_id"`
	Identity      types.Object          `tfsdk:"identity"`
	Roles         []ProjectIdentityRole `tfsdk:"roles"`
	MembershipId  types.String          `tfsdk:"membership_id"`
	AdoptExisting types.Bool            `tfsdk:"adopt_existing"`
}

type ProjectIdentityDetails struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	AuthMethods types.List   `tfsdk:"auth_methods"`
}

type ProjectIdentityRole struct {
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
func (r *ProjectIdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_identity"
}

// Schema defines the schema for the resource.
func (r *ProjectIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assign an existing organization-level machine identity to a project & save to Infisical. This resource does not create a new identity; it manages the project membership and role(s) of an identity that already exists at the organization level. To create an identity that lives inside a single project, use the infisical_project_scoped_identity resource instead. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The id of the project",
				Required:    true,
			},
			"identity_id": schema.StringAttribute{
				Description: "The id of the identity.",
				Required:    true,
			},
			"membership_id": schema.StringAttribute{
				Description:   "The membership Id of the project identity",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"adopt_existing": schema.BoolAttribute{
				Description:   "When true, if the identity is already a member of the project (e.g. auto-added by Infisical when the project was created by this identity), the existing membership is adopted and its roles are updated to match the desired state instead of returning an error. Defaults to false.",
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"identity": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The identity details of the project identity",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description:   "The ID of the identity",
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"name": schema.StringAttribute{
						Description: "The name of the identity",
						Computed:    true,
					},
					"auth_methods": schema.ListAttribute{
						ElementType: types.StringType,
						Description: "The auth methods for the identity",
						Computed:    true,
					},
				},
			},
			"roles": schema.ListNestedAttribute{
				Required:    true,
				Description: "The roles assigned to the project identity",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the project identity role.",
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

// orderAPIIdentityRolesByPlan reorders API-returned roles to match the ordering specified in the Terraform plan.
// The API may return roles in a non-deterministic order, but Terraform requires list elements to maintain
// the same positional order as the plan to avoid "inconsistent result after apply" errors.
func orderAPIIdentityRolesByPlan(planRoles []ProjectIdentityRole, apiRoles []ProjectIdentityRole) []ProjectIdentityRole {
	apiRoleMap := make(map[string]ProjectIdentityRole, len(apiRoles))
	for _, role := range apiRoles {
		apiRoleMap[role.RoleSlug.ValueString()] = role
	}

	ordered := make([]ProjectIdentityRole, 0, len(apiRoles))
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
	var remaining []ProjectIdentityRole
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
func buildProjectIdentityRequestRoles(planRoles []ProjectIdentityRole) ([]infisical.CreateProjectIdentityRequestRoles, error) {
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
func mapAPIRolesToIdentityModel(apiRoles []infisical.ProjectMemberRole) []ProjectIdentityRole {
	result := make([]ProjectIdentityRole, 0, len(apiRoles))
	for _, el := range apiRoles {
		val := ProjectIdentityRole{
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
func (r *ProjectIdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ProjectIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan ProjectIdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := buildProjectIdentityRequestRoles(plan.Roles)
	if err != nil {
		resp.Diagnostics.AddError("Error assigning role to identity", err.Error())
		return
	}

	alreadyMember := false
	var existing infisical.GetProjectIdentityByIDResponse
	if plan.AdoptExisting.ValueBool() {
		var getErr error
		existing, getErr = r.client.GetProjectIdentityByID(infisical.GetProjectIdentityByIDRequest{
			ProjectID:  plan.ProjectID.ValueString(),
			IdentityID: plan.IdentityID.ValueString(),
		})
		switch {
		case getErr == nil:
			alreadyMember = true
		case errors.Is(getErr, infisical.ErrNotFound):
			alreadyMember = false
		default:
			resp.Diagnostics.AddError(
				"Error checking project identity membership",
				"Couldn't check existing membership, unexpected error: "+getErr.Error(),
			)
			return
		}
	}

	if alreadyMember {
		prevSlugs := make([]string, 0, len(existing.Membership.Roles))
		for _, role := range existing.Membership.Roles {
			slug := role.Role
			if role.CustomRoleSlug != "" {
				slug = role.CustomRoleSlug
			}
			prevSlugs = append(prevSlugs, slug)
		}
		previousRoles := strings.Join(prevSlugs, ", ")
		if previousRoles == "" {
			previousRoles = "(none)"
		}
		resp.Diagnostics.AddWarning(
			"Adopted existing project identity membership",
			fmt.Sprintf("Identity %s was already a member of project %s (membership %s); the existing membership will be adopted into Terraform state and its roles updated to match the configuration (previous roles: %s).",
				plan.IdentityID.ValueString(), plan.ProjectID.ValueString(), existing.Membership.ID, previousRoles),
		)
		updateRoles := make([]infisical.UpdateProjectIdentityRequestRoles, len(roles))
		for i, role := range roles {
			updateRoles[i] = infisical.UpdateProjectIdentityRequestRoles(role)
		}
		_, err = r.client.UpdateProjectIdentity(infisical.UpdateProjectIdentityRequest{
			ProjectID:  plan.ProjectID.ValueString(),
			IdentityID: plan.IdentityID.ValueString(),
			Roles:      updateRoles,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating existing project identity membership",
				"Couldn't update roles for existing membership, unexpected error: "+err.Error(),
			)
			return
		}
	} else {
		_, err = r.client.CreateProjectIdentity(infisical.CreateProjectIdentityRequest{
			ProjectID:  plan.ProjectID.ValueString(),
			IdentityID: plan.IdentityID.ValueString(),
			Roles:      roles,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error attaching identity to project",
				"Couldn't create project identity to Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	projectIdentityDetails, err := r.client.GetProjectIdentityByID(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  plan.ProjectID.ValueString(),
		IdentityID: plan.IdentityID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching identity",
			"Couldn't find identity in project, unexpected error: "+err.Error(),
		)
		return
	}

	apiRoles := mapAPIRolesToIdentityModel(projectIdentityDetails.Membership.Roles)
	plan.Roles = orderAPIIdentityRolesByPlan(plan.Roles, apiRoles)
	plan.MembershipId = types.StringValue(projectIdentityDetails.Membership.ID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	identityDetails := ProjectIdentityDetails{
		ID:   types.StringValue(projectIdentityDetails.Membership.Identity.Id),
		Name: types.StringValue(projectIdentityDetails.Membership.Identity.Name),
	}
	elements := make([]attr.Value, len(projectIdentityDetails.Membership.Identity.AuthMethods))
	for i, method := range projectIdentityDetails.Membership.Identity.AuthMethods {
		elements[i] = types.StringValue(method)
	}
	identityDetails.AuthMethods = types.ListValueMust(types.StringType, elements)

	diags = resp.State.SetAttribute(ctx, path.Root("identity"), identityDetails)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ProjectIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state ProjectIdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectID.String() == "" || state.IdentityID.String() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	projectIdentityDetails, err := r.client.GetProjectIdentityByID(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error fetching identity",
			"Couldn't find identity in project, unexpected error: "+err.Error(),
		)
		return
	}

	apiRoles := mapAPIRolesToIdentityModel(projectIdentityDetails.Membership.Roles)
	state.Roles = orderAPIIdentityRolesByPlan(state.Roles, apiRoles)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	identityDetails := ProjectIdentityDetails{
		ID:   types.StringValue(projectIdentityDetails.Membership.Identity.Id),
		Name: types.StringValue(projectIdentityDetails.Membership.Identity.Name),
	}
	elements := make([]attr.Value, len(projectIdentityDetails.Membership.Identity.AuthMethods))
	for i, method := range projectIdentityDetails.Membership.Identity.AuthMethods {
		elements[i] = types.StringValue(method)
	}
	identityDetails.AuthMethods = types.ListValueMust(types.StringType, elements)

	diags = resp.State.SetAttribute(ctx, path.Root("identity"), identityDetails)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ProjectIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan ProjectIdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectIdentityResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.IdentityID != plan.IdentityID {
		resp.Diagnostics.AddError(
			"Unable to update project identity",
			fmt.Sprintf("Cannot change identity id, previous identity: %s, new identity id: %s", state.IdentityID, plan.IdentityID),
		)
		return
	}

	requestRoles, err := buildProjectIdentityRequestRoles(plan.Roles)
	if err != nil {
		resp.Diagnostics.AddError("Error assigning role to identity", err.Error())
		return
	}

	roles := make([]infisical.UpdateProjectIdentityRequestRoles, len(requestRoles))
	for i, role := range requestRoles {
		roles[i] = infisical.UpdateProjectIdentityRequestRoles(role)
	}

	_, err = r.client.UpdateProjectIdentity(infisical.UpdateProjectIdentityRequest{
		ProjectID:  plan.ProjectID.ValueString(),
		IdentityID: plan.IdentityID.ValueString(),
		Roles:      roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning roles to identity",
			"Couldn't update role , unexpected error: "+err.Error(),
		)
		return
	}

	projectIdentityDetails, err := r.client.GetProjectIdentityByID(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  plan.ProjectID.ValueString(),
		IdentityID: plan.IdentityID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching identity",
			"Couldn't find identity in project, unexpected error: "+err.Error(),
		)
		return
	}

	apiRoles := mapAPIRolesToIdentityModel(projectIdentityDetails.Membership.Roles)
	plan.Roles = orderAPIIdentityRolesByPlan(plan.Roles, apiRoles)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	identityDetails := ProjectIdentityDetails{
		ID:   types.StringValue(projectIdentityDetails.Membership.Identity.Id),
		Name: types.StringValue(projectIdentityDetails.Membership.Identity.Name),
	}
	elements := make([]attr.Value, len(projectIdentityDetails.Membership.Identity.AuthMethods))
	for i, method := range projectIdentityDetails.Membership.Identity.AuthMethods {
		elements[i] = types.StringValue(method)
	}
	identityDetails.AuthMethods = types.ListValueMust(types.StringType, elements)

	diags = resp.State.SetAttribute(ctx, path.Root("identity"), identityDetails)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ProjectIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state ProjectIdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectIdentity(infisical.DeleteProjectIdentityRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project identity",
			"Couldn't delete project identity from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *ProjectIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectIdentityDetails, err := r.client.GetProjectIdentityByMembershipID(infisical.GetProjectIdentityByMembershipIDRequest{
		MembershipID: req.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error fetching identity", "Couldn't find identity by membership ID, unexpected error: "+err.Error())
		return
	}

	identityDetails := ProjectIdentityDetails{
		ID:   types.StringValue(projectIdentityDetails.Membership.Identity.Id),
		Name: types.StringValue(projectIdentityDetails.Membership.Identity.Name),
	}

	authMethods := make([]attr.Value, len(projectIdentityDetails.Membership.Identity.AuthMethods))
	for i, method := range projectIdentityDetails.Membership.Identity.AuthMethods {
		authMethods[i] = types.StringValue(method)
	}
	identityDetails.AuthMethods = types.ListValueMust(types.StringType, authMethods)

	planRoles := mapAPIRolesToIdentityModel(projectIdentityDetails.Membership.Roles)

	// Sort roles alphabetically by slug for deterministic state after import
	sort.Slice(planRoles, func(i, j int) bool {
		return planRoles[i].RoleSlug.ValueString() < planRoles[j].RoleSlug.ValueString()
	})

	// Create the identity object value
	identityObj, diags := types.ObjectValue(map[string]attr.Type{
		"id":           types.StringType,
		"name":         types.StringType,
		"auth_methods": types.ListType{ElemType: types.StringType},
	}, map[string]attr.Value{
		"id":           identityDetails.ID,
		"name":         identityDetails.Name,
		"auth_methods": identityDetails.AuthMethods,
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := ProjectIdentityResourceModel{
		ProjectID:     types.StringValue(projectIdentityDetails.Membership.Project.ID),
		IdentityID:    types.StringValue(projectIdentityDetails.Membership.Identity.Id),
		MembershipId:  types.StringValue(projectIdentityDetails.Membership.ID),
		Roles:         planRoles,
		Identity:      identityObj,
		AdoptExisting: types.BoolValue(false),
	}

	// Set the state with the imported data
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}
