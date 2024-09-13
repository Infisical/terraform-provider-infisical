package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	ProjectSlug  types.String       `tfsdk:"project_slug"`
	GroupSlug    types.String       `tfsdk:"group_slug"`
	Roles        []ProjectGroupRole `tfsdk:"roles"`
	MembershipID types.String       `tfsdk:"membership_id"`
}

type ProjectGroupRole struct {
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
			"project_slug": schema.StringAttribute{
				Description:   "The slug of the project.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_slug": schema.StringAttribute{
				Description: "The slug of the group.",
				Required:    true,
			},
			"membership_id": schema.StringAttribute{
				Description:   "The membership Id of the project group",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"roles": schema.ListNestedAttribute{
				Required:    true,
				Description: "The roles assigned to the project group",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the project group role.",
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

func updateProjectGroupStateByApi(r *ProjectGroupResource, ctx context.Context, diagnose diag.Diagnostics, state *ProjectGroupResourceModel) {
	projectGroupDetails, err := r.client.GetProjectGroupMembership(infisical.GetProjectGroupMembershipRequest{
		ProjectSlug: state.ProjectSlug.ValueString(),
		GroupSlug:   state.GroupSlug.ValueString(),
	})

	if err != nil {
		diagnose.AddError(
			"Error fetching group details",
			"Couldn't find group in project, unexpected error: "+err.Error(),
		)
		return
	}

	planRoles := make([]ProjectGroupRole, 0, len(projectGroupDetails.Membership.Roles))
	for _, el := range projectGroupDetails.Membership.Roles {
		val := ProjectGroupRole{
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
	state.MembershipID = types.StringValue(projectGroupDetails.Membership.ID)
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

	var roles []infisical.CreateProjectGroupRequestRoles
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

		roles = append(roles, infisical.CreateProjectGroupRequestRoles{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
	}

	if !hasAtleastOnePermanentRole {
		resp.Diagnostics.AddError("Error assigning role to group", "Must have atleast one permanent role")
		return
	}

	projectDetail, err := r.client.GetProjectById(infisical.GetProjectByIdRequest{
		ID: plan.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error attaching group to project",
			"Couldn't fetch project details, unexpected error: "+err.Error(),
		)
		return
	}

	_, err = r.client.CreateProjectGroup(infisical.CreateProjectGroupRequest{
		ProjectSlug: projectDetail.Slug,
		GroupSlug:   plan.GroupSlug.ValueString(),
		Roles:       roles,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error attaching group to project",
			"Couldn't create project group to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ProjectSlug = types.StringValue(projectDetail.Slug)
	updateProjectGroupStateByApi(r, ctx, resp.Diagnostics, &plan)
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

	updateProjectGroupStateByApi(r, ctx, resp.Diagnostics, &state)
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

	if plan.GroupSlug != state.GroupSlug {
		resp.Diagnostics.AddError(
			"Unable to update project group",
			fmt.Sprintf("Cannot change group slug, previous group: %s, new group: %s", state.GroupSlug, plan.GroupSlug),
		)
		return
	}

	var roles []infisical.UpdateProjectGroupRequestRoles
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

		roles = append(roles, infisical.UpdateProjectGroupRequestRoles{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
	}

	if !hasAtleastOnePermanentRole {
		resp.Diagnostics.AddError("Error assigning role to group", "Must have atleast one permanent role")
		return
	}

	_, err := r.client.UpdateProjectGroup(infisical.UpdateProjectGroupRequest{
		ProjectSlug: state.ProjectSlug.ValueString(),
		GroupSlug:   plan.GroupSlug.ValueString(),
		Roles:       roles,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning roles to group",
			"Couldn't update role, unexpected error: "+err.Error(),
		)
		return
	}

	updateProjectGroupStateByApi(r, ctx, resp.Diagnostics, &plan)
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
		ProjectSlug: state.ProjectSlug.ValueString(),
		GroupSlug:   state.GroupSlug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project group",
			"Couldn't delete project group from Infiscial, unexpected error: "+err.Error(),
		)
	}
}
