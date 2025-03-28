package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"

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
	ProjectID    types.String `tfsdk:"project_id"`
	GroupID      types.String `tfsdk:"group_id"`
	GroupName    types.String `tfsdk:"group_name"`
	Roles        types.String `tfsdk:"roles"`
	MembershipID types.String `tfsdk:"membership_id"`
}

type ProjectGroupRole struct {
	RoleSlug string `json:"role_slug"`
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
			"roles": schema.StringAttribute{
				Description: "JSON array of role assignments for this group. Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					pkg.UnorderedJsonEquivalentModifier{},
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

	var parsedRoles []ProjectGroupRole
	err := json.Unmarshal([]byte(plan.Roles.ValueString()), &parsedRoles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing roles JSON",
			fmt.Sprintf("Failed to parse roles JSON: %s", err.Error()),
		)
		return
	}

	var roles []infisical.CreateProjectGroupRequestRoles
	for _, el := range parsedRoles {
		if el.RoleSlug == "" {
			resp.Diagnostics.AddError(
				"Error parsing roles JSON",
				"Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
			)
			return
		}
		roles = append(roles, infisical.CreateProjectGroupRequestRoles{
			Role: el.RoleSlug,
		})
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
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading project group membership",
				"Couldn't read project group membership from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	planRoles := make([]ProjectGroupRole, 0, len(projectGroupMembership.Membership.Roles))
	for _, el := range projectGroupMembership.Membership.Roles {
		val := ProjectGroupRole{
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
	state.MembershipID = types.StringValue(projectGroupMembership.Membership.ID)
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

	var parsedRoles []ProjectGroupRole
	err := json.Unmarshal([]byte(plan.Roles.ValueString()), &parsedRoles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing roles JSON",
			fmt.Sprintf("Failed to parse roles JSON: %s", err.Error()),
		)
		return
	}

	var roles []infisical.UpdateProjectGroupRequestRoles
	for _, el := range parsedRoles {
		if el.RoleSlug == "" {
			resp.Diagnostics.AddError(
				"Error parsing roles JSON",
				"Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
			)
			return
		}
		roles = append(roles, infisical.UpdateProjectGroupRequestRoles{
			Role: el.RoleSlug,
		})
	}

	_, err = r.client.UpdateProjectGroup(infisical.UpdateProjectGroupRequest{
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
