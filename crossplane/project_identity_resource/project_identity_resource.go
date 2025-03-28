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
	ProjectID    types.String `tfsdk:"project_id"`
	IdentityID   types.String `tfsdk:"identity_id"`
	Roles        types.String `tfsdk:"roles"`
	MembershipId types.String `tfsdk:"membership_id"`
}

type ProjectIdentityDetails struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	AuthMethods []string `json:"auth_methods"`
}

type ProjectIdentityRole struct {
	RoleSlug string `json:"role_slug"`
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
			"roles": schema.StringAttribute{
				Description: "JSON array of role assignments for this identity. Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					pkg.UnorderedJsonEquivalentModifier{},
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

	var parsedRoles []ProjectIdentityRole
	err := json.Unmarshal([]byte(plan.Roles.ValueString()), &parsedRoles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing roles JSON",
			fmt.Sprintf("Failed to parse roles JSON: %s", err.Error()),
		)
		return
	}

	var roles []infisical.CreateProjectIdentityRequestRoles
	for _, el := range parsedRoles {
		if el.RoleSlug == "" {
			resp.Diagnostics.AddError(
				"Error parsing roles JSON",
				"Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
			)
			return
		}
		roles = append(roles, infisical.CreateProjectIdentityRequestRoles{
			Role: el.RoleSlug,
		})
	}

	_, err = r.client.CreateProjectIdentity(infisical.CreateProjectIdentityRequest{
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

	plan.MembershipId = types.StringValue(projectIdentityDetails.Membership.ID)
	diags = resp.State.Set(ctx, plan)
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

	roles := make([]ProjectIdentityRole, 0, len(projectIdentityDetails.Membership.Roles))
	for _, el := range projectIdentityDetails.Membership.Roles {
		if el.CustomRoleId != "" {
			roles = append(roles, ProjectIdentityRole{
				RoleSlug: el.CustomRoleSlug,
			})
		} else {
			roles = append(roles, ProjectIdentityRole{
				RoleSlug: el.Role,
			})
		}
	}

	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error serializing roles to JSON",
			fmt.Sprintf("Failed to serialize roles to JSON: %s", err.Error()),
		)
		return
	}
	state.Roles = types.StringValue(string(rolesJSON))
	state.MembershipId = types.StringValue(projectIdentityDetails.Membership.ID)

	diags = resp.State.Set(ctx, state)
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

	var parsedRoles []ProjectIdentityRole
	err := json.Unmarshal([]byte(plan.Roles.ValueString()), &parsedRoles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing roles JSON",
			fmt.Sprintf("Failed to parse roles JSON: %s", err.Error()),
		)
		return
	}

	for _, el := range parsedRoles {
		if el.RoleSlug == "" {
			resp.Diagnostics.AddError(
				"Error parsing roles JSON",
				"Each role object must include a `role_slug` field. Example: `[{\"role_slug\":\"admin\"},{\"role_slug\":\"member\"}]`.",
			)
			return
		}

		roles = append(roles, infisical.UpdateProjectIdentityRequestRoles{
			Role: el.RoleSlug,
		})
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

	plan.MembershipId = types.StringValue(projectIdentityDetails.Membership.ID)
	diags = resp.State.Set(ctx, plan)
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
