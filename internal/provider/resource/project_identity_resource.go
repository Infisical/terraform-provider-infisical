package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource = &ProjectIdentityResource{}
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
	ProjectID    types.String          `tfsdk:"project_id"`
	IdentityID   types.String          `tfsdk:"identity_id"`
	Identity     types.Object          `tfsdk:"identity"`
	Roles        []ProjectIdentityRole `tfsdk:"roles"`
	MembershipId types.String          `tfsdk:"membership_id"`
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
		Description: "Create project identities & save to Infisical. Only Machine Identity authentication is supported for this data source",
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

	var roles []infisical.CreateProjectIdentityRequestRoles
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

		roles = append(roles, infisical.CreateProjectIdentityRequestRoles{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
	}
	if !hasAtleastOnePermanentRole {
		resp.Diagnostics.AddError("Error assigning role to identity", "Must have atleast one permanent role")
		return
	}

	_, err := r.client.CreateProjectIdentity(infisical.CreateProjectIdentityRequest{
		ProjectID:  plan.ProjectID.ValueString(),
		IdentityID: plan.IdentityID.ValueString(),
		Roles:      roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error attaching identity to project",
			"Couldn't create project identity to Infiscial, unexpected error: "+err.Error(),
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

	planRoles := make([]ProjectIdentityRole, 0, len(projectIdentityDetails.Membership.Roles))
	for _, el := range projectIdentityDetails.Membership.Roles {
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
	if resp.Diagnostics.HasError() {
		return
	}

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
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error fetching identity",
			"Couldn't find identity in project, unexpected error: "+err.Error(),
		)
		return
	}

	planRoles := make([]ProjectIdentityRole, 0, len(projectIdentityDetails.Membership.Roles))
	for _, el := range projectIdentityDetails.Membership.Roles {
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
	if resp.Diagnostics.HasError() {
		return
	}

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

	var roles []infisical.UpdateProjectIdentityRequestRoles
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

		roles = append(roles, infisical.UpdateProjectIdentityRequestRoles{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
	}

	if !hasAtleastOnePermanentRole {
		resp.Diagnostics.AddError("Error assigning role to identity", "Must have atleast one permanent role")
		return
	}

	_, err := r.client.UpdateProjectIdentity(infisical.UpdateProjectIdentityRequest{
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

	planRoles := make([]ProjectIdentityRole, 0, len(projectIdentityDetails.Membership.Roles))
	for _, el := range projectIdentityDetails.Membership.Roles {
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
	if resp.Diagnostics.HasError() {
		return
	}
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
			"Couldn't delete project identity from Infiscial, unexpected error: "+err.Error(),
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

	planRoles := make([]ProjectIdentityRole, 0, len(projectIdentityDetails.Membership.Roles))
	for _, el := range projectIdentityDetails.Membership.Roles {
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
		ProjectID:    types.StringValue(projectIdentityDetails.Membership.Project.ID),
		IdentityID:   types.StringValue(projectIdentityDetails.Membership.Identity.Id),
		MembershipId: types.StringValue(projectIdentityDetails.Membership.ID),
		Roles:        planRoles,
		Identity:     identityObj,
	}

	// Set the state with the imported data
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}
