package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_                   resource.Resource = &projectRoleResource{}
	PERMISSION_ACTIONS                    = []string{"create", "edit", "delete", "read"}
	PERMISSION_SUBJECTS                   = []string{"role", "member", "groups", "settings", "integrations", "webhooks", "service-tokens", "environments", "tags", "audit-logs", "ip-allowlist", "workspace", "secrets", "secret-rollback", "secret-approval", "secret-rotation", "identity"}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectRoleResource() resource.Resource {
	return &projectRoleResource{}
}

// projectRoleResource is the resource implementation.
type projectRoleResource struct {
	client *infisical.Client
}

// projectRoleResourceSourceModel describes the data source data model.
type projectRoleResourceModel struct {
	Name        types.String                     `tfsdk:"name"`
	Description types.String                     `tfsdk:"description"`
	Slug        types.String                     `tfsdk:"slug"`
	ProjectSlug types.String                     `tfsdk:"project_slug"`
	ID          types.String                     `tfsdk:"id"`
	Permissions []projectRoleResourcePermissions `tfsdk:"permissions"`
}

type projectRoleResourcePermissions struct {
	Action     types.String                            `tfsdk:"action"`
	Subject    types.String                            `tfsdk:"subject"`
	Conditions *projectRoleResourcePermissionCondition `tfsdk:"conditions"`
}

type projectRoleResourcePermissionCondition struct {
	Environment types.String `tfsdk:"environment"`
	SecretPath  types.String `tfsdk:"secret_path"`
}

// Metadata returns the resource type name.
func (r *projectRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_role"
}

// Schema defines the schema for the resource.
func (r *projectRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create custom project roles & save to Infisical. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug for the new role",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name for the new role",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description for the new role",
				Optional:    true,
			},
			"project_slug": schema.StringAttribute{
				Description: "The slug of the project to create role",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the role",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"permissions": schema.ListNestedAttribute{
				Required:    true,
				Description: "The permissions assigned to the project role",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Description: fmt.Sprintf("Describe what action an entity can take. Enum: %s", strings.Join(PERMISSION_ACTIONS, ",")),
							Required:    true,
						},
						"subject": schema.StringAttribute{
							Description: fmt.Sprintf("Describe what action an entity can take. Enum: %s", strings.Join(PERMISSION_SUBJECTS, ",")),
							Required:    true,
						},
						"conditions": schema.SingleNestedAttribute{
							Optional:    true,
							Description: "The conditions to scope permissions",
							Attributes: map[string]schema.Attribute{
								"environment": schema.StringAttribute{
									Description: "The environment slug this permission should allow.",
									Optional:    true,
								},
								"secret_path": schema.StringAttribute{
									Description: "The secret path this permission should be scoped to",
									Optional:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectRolePermissions := make([]infisicalclient.ProjectRolePermissionRequest, 0, len(plan.Permissions))
	for _, el := range plan.Permissions {
		condition := make(map[string]any)
		if el.Conditions != nil {
			environment := el.Conditions.Environment.ValueString()
			secretPath := el.Conditions.SecretPath.ValueString()
			if environment != "" {
				condition["environment"] = environment
			}
			if secretPath != "" {
				condition["secretPath"] = map[string]string{"$glob": secretPath}
			}
		} else {
			condition = nil
		}

		projectRolePermissions = append(projectRolePermissions, infisicalclient.ProjectRolePermissionRequest{
			Action:     el.Action.ValueString(),
			Subject:    el.Subject.ValueString(),
			Conditions: condition,
		})
	}

	newProjectRole, err := r.client.CreateProjectRole(infisical.CreateProjectRoleRequest{
		ProjectSlug: plan.ProjectSlug.ValueString(),
		Slug:        plan.Slug.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Permissions: projectRolePermissions,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project role",
			"Couldn't save project to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newProjectRole.Role.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	projectRole, err := r.client.GetProjectRoleBySlug(infisical.GetProjectRoleBySlugRequest{
		RoleSlug:    state.Slug.ValueString(),
		ProjectSlug: state.ProjectSlug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project role",
			"Couldn't read project role from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	state.Description = types.StringValue(projectRole.Role.Description)
	state.ID = types.StringValue(projectRole.Role.ID)
	state.Name = types.StringValue(projectRole.Role.Name)

	permissionPlan := make([]projectRoleResourcePermissions, 0, len(projectRole.Role.Permissions))
	for _, el := range projectRole.Role.Permissions {
		action, isValid := el["action"].(string)
		if el["action"] != nil && !isValid {
			action, isValid = el["action"].([]any)[0].(string)
			if !isValid {
				resp.Diagnostics.AddError(
					"Error reading project role",
					"Couldn't read project role from Infiscial, invalid action field in permission",
				)
				return
			}
		}

		subject, isValid := el["subject"].(string)
		if el["subject"] != nil && !isValid {
			subject, isValid = el["subject"].([]any)[0].(string)
			if !isValid {
				resp.Diagnostics.AddError(
					"Error reading project role",
					"Couldn't read project role from Infiscial, invalid subject field in permission",
				)
				return
			}
		}
		var secretPath, environment string
		if el["conditions"] != nil {
			conditions, isValid := el["conditions"].(map[string]any)
			if !isValid {
				resp.Diagnostics.AddError(
					"Error reading project role",
					"Couldn't read project role from Infiscial, invalid conditions field in permission",
				)
				return
			}

			environment, isValid = conditions["environment"].(string)
			if !isValid {
				resp.Diagnostics.AddError(
					"Error reading project role",
					"Couldn't read project role from Infiscial, invalid environment field in permission",
				)
				return
			}

			// secret path parsing.
			if val, isValid := conditions["secretPath"].(map[string]any); isValid {
				secretPath, isValid = val["$glob"].(string)
				if !isValid {
					resp.Diagnostics.AddError(
						"Error reading project role",
						"Couldn't read project role from Infiscial, invalid secret path field in permission",
					)
					return
				}
			}
		}

		permissionPlan = append(permissionPlan, projectRoleResourcePermissions{
			Action:  types.StringValue(action),
			Subject: types.StringValue(subject),
			Conditions: &projectRoleResourcePermissionCondition{
				Environment: types.StringValue(environment),
				SecretPath:  types.StringValue(secretPath),
			},
		})
	}

	state.Permissions = permissionPlan
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectRoleResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectSlug != plan.ProjectSlug {
		resp.Diagnostics.AddError(
			"Unable to update project role",
			"Project slug cannot be updated",
		)
		return
	}

	projectRolePermissions := make([]infisicalclient.ProjectRolePermissionRequest, 0, len(plan.Permissions))
	for _, el := range plan.Permissions {
		condition := make(map[string]any)
		if el.Conditions != nil {
			environment := el.Conditions.Environment.ValueString()
			secretPath := el.Conditions.SecretPath.ValueString()
			if environment != "" {
				condition["environment"] = environment
			}
			if secretPath != "" {
				condition["secretPath"] = map[string]string{"$glob": secretPath}
			}
		} else {
			condition = nil
		}
		projectRolePermissions = append(projectRolePermissions, infisicalclient.ProjectRolePermissionRequest{
			Action:     el.Action.ValueString(),
			Subject:    el.Subject.ValueString(),
			Conditions: condition,
		})
	}

	_, err := r.client.UpdateProjectRole(infisical.UpdateProjectRoleRequest{
		ProjectSlug: plan.ProjectSlug.ValueString(),
		RoleId:      plan.ID.ValueString(),
		Slug:        plan.Slug.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Permissions: projectRolePermissions,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project role",
			"Couldn't update project role from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectRole(infisical.DeleteProjectRoleRequest{
		ProjectSlug: state.ProjectSlug.ValueString(),
		RoleId:      state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project role",
			"Couldn't delete project role from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
