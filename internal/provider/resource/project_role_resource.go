package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_                   resource.Resource = &projectRoleResource{}
	PERMISSION_ACTIONS                    = []string{"create", "edit", "delete", "read"}
	PERMISSION_SUBJECTS                   = []string{"role", "member", "groups", "settings", "integrations", "webhooks", "service-tokens", "environments", "tags", "audit-logs", "ip-allowlist", "workspace", "secrets", "secret-rollback", "secret-approval", "secret-rotation", "identity", "certificate-authorities", "certificates", "certificate-templates", "kms", "pki-alerts", "pki-collections"}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectRoleResource() resource.Resource {
	return &projectRoleResource{}
}

// projectRoleResource is the resource implementation.
type projectRoleResource struct {
	client *infisical.Client
}

type PermissionV2Entry struct {
	Action     []string                     `tfsdk:"action"`
	Subject    string                       `tfsdk:"subject"`
	Inverted   bool                         `tfsdk:"inverted"`
	Conditions map[string]map[string]string `tfsdk:"conditions"`
}

// projectRoleResourceSourceModel describes the data source data model.
type projectRoleResourceModel struct {
	Name          types.String        `tfsdk:"name"`
	Description   types.String        `tfsdk:"description"`
	Slug          types.String        `tfsdk:"slug"`
	ProjectSlug   types.String        `tfsdk:"project_slug"`
	ID            types.String        `tfsdk:"id"`
	Permissions   types.List          `tfsdk:"permissions"`
	PermissionsV2 []PermissionV2Entry `tfsdk:"permissions_v2"`
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

func validatePermissionV2Array(permissions []map[string]any) error {
	for _, permission := range permissions {
		subject, exists := permission["subject"]
		if !exists {
			return fmt.Errorf("Error parsing permissions_v2: subject property should be defined")
		}

		_, ok := subject.(string)
		if !ok {
			return fmt.Errorf("Error parsing permissions_v2: subject property should be a string")
		}

		action, exists := permission["action"]
		if !exists {
			return fmt.Errorf("Error parsing permissions_v2: action property should be defined")
		}

		_, ok = action.([]interface{})
		if !ok {
			return fmt.Errorf("Error parsing permissions_v2: action property should be an array")
		}

		inverted, exists := permission["inverted"]
		if !exists {
			return fmt.Errorf("Error parsing permissions_v2: inverted property should be defined")
		}

		_, ok = inverted.(bool)
		if !ok {
			return fmt.Errorf("Error parsing permissions_v2: inverted property should be a boolean")
		}

	}

	return nil
}

// Metadata returns the resource type name.
func (r *projectRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_role"
}

var permissionsObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"action":  types.StringType,
		"subject": types.StringType,
		"conditions": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"environment": types.StringType,
				"secret_path": types.StringType,
			},
		},
	},
}

// Schema defines the schema for the resource.
func (r *projectRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create custom project roles & save to Infisical. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug for the new role",
				Required:    true,
				Validators: []validator.String{
					infisicaltf.SlugRegexValidator,
				},
			},
			"name": schema.StringAttribute{
				Description: "The name for the new role",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description for the new role. Defaults to an empty string.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
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
			"permissions_v2": schema.SetNestedAttribute{
				Optional:    true,
				Description: "The permissions assigned to the project role",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.SetAttribute{
							ElementType: types.StringType,
							Description: fmt.Sprintf("Describe what actions an entity can take. Enum: %s", strings.Join(PERMISSION_ACTIONS, ",")),
							Required:    true,
						},
						"subject": schema.StringAttribute{
							Description: fmt.Sprintf("Describe the entity the permission pertains to. Enum: %s", strings.Join(PERMISSION_SUBJECTS, ",")),
							Required:    true,
						},
						"inverted": schema.BoolAttribute{
							Description: "Whether rule forbids. Set this to true if permission forbids.",
							Required:    true,
						},
						"conditions": schema.MapAttribute{
							Optional:    true,
							Description: "When specified, only matching conditions will be allowed to access given resource.",
							ElementType: types.MapType{
								ElemType: types.StringType,
							},
						},
					},
				},
			},
			"permissions": schema.ListNestedAttribute{
				Optional:    true,
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

	if (!plan.Permissions.IsNull() && plan.PermissionsV2 != nil) || (plan.Permissions.IsNull() && plan.PermissionsV2 == nil) {
		resp.Diagnostics.AddError(
			"Error creating project role",
			"Define either the permissions or permissions_v2 property but not both.",
		)
		return
	}

	// Permissions V1
	if !plan.Permissions.IsNull() {
		permissions := []projectRoleResourcePermissions{}
		diags = plan.Permissions.ElementsAs(ctx, &permissions, true)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		projectRolePermissions := make([]infisicalclient.ProjectRolePermissionRequest, 0, len(permissions))

		for _, el := range permissions {
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
	} else {
		project, err := r.client.GetProject(infisical.GetProjectRequest{
			Slug: plan.ProjectSlug.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating project role",
				"Unexpected error: "+err.Error(),
			)
			return
		}

		permissions := make([]map[string]any, len(plan.PermissionsV2))
		for i, perm := range plan.PermissionsV2 {
			permMap := map[string]any{
				"action":   perm.Action,
				"subject":  perm.Subject,
				"inverted": perm.Inverted,
			}

			if perm.Conditions != nil {
				permMap["conditions"] = perm.Conditions
			}

			permissions[i] = permMap
		}

		newProjectRole, err := r.client.CreateProjectRoleV2(infisical.CreateProjectRoleV2Request{
			ProjectId:   project.ID,
			Slug:        plan.Slug.ValueString(),
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Permissions: permissions,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating project role",
				"Couldn't save project to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		plan.ID = types.StringValue(newProjectRole.Role.ID)
	}

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

	// Permissions V1
	if !state.Permissions.IsNull() {
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
				actions, isValid := el["action"].([]any)
				if !isValid {
					resp.Diagnostics.AddError(
						"Error reading project role",
						"Couldn't read project role from Infiscial, invalid action field in permission",
					)
					return
				}

				if len(actions) > 1 {
					resp.Diagnostics.AddWarning(
						"Drift detected",
						"Multiple actions are not supported on 'infisical_project_role', use 'infisical_project_role_v2'.",
					)
					state.Permissions = types.ListNull(permissionsObjectType)
					resp.State.Set(ctx, state)
					return
				}

				action, isValid = actions[0].(string)
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
					if permissionV2Environment, isValid := conditions["environment"].(map[string]any); isValid {
						environment, isValid = permissionV2Environment["$eq"].(string)
						if !isValid {
							resp.Diagnostics.AddWarning(
								"Drift detected",
								"Environment condition provided are not compatible on 'infisical_project_role', use 'infisical_project_role_v2'.",
							)
							state.Permissions = types.ListNull(permissionsObjectType)
							resp.State.Set(ctx, state)
							return
						}
					}
				}

				// secret path parsing.
				if val, isValid := conditions["secretPath"].(map[string]any); isValid {
					secretPath, isValid = val["$glob"].(string)
					if !isValid {
						resp.Diagnostics.AddWarning(
							"Drift detected",
							"Secret path condition provided are not compatible on 'infisical_project_role', use 'infisical_project_role_v2'.",
						)
						state.Permissions = types.ListNull(permissionsObjectType)
						resp.State.Set(ctx, state)
						return
					}
				}
			}

			var conditions *projectRoleResourcePermissionCondition

			if el["conditions"] == nil {
				conditions = nil
			} else {
				conditions = &projectRoleResourcePermissionCondition{
					Environment: types.StringValue(environment),
					SecretPath:  types.StringValue(secretPath),
				}
			}

			permissionPlan = append(permissionPlan, projectRoleResourcePermissions{
				Action:     types.StringValue(action),
				Subject:    types.StringValue(subject),
				Conditions: conditions,
			})
		}

		permissionListValue, diags := types.ListValueFrom(ctx, permissionsObjectType, permissionPlan)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}

		state.Permissions = permissionListValue
	} else {
		project, err := r.client.GetProject(infisical.GetProjectRequest{
			Slug: state.ProjectSlug.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading project role",
				"Unexpected error: "+err.Error(),
			)
			return
		}

		projectRole, err := r.client.GetProjectRoleBySlugV2(infisical.GetProjectRoleBySlugV2Request{
			ProjectId: project.ID,
			RoleSlug:  state.Slug.ValueString(),
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

		permissions := make([]PermissionV2Entry, len(projectRole.Role.Permissions))
		for i, permMap := range projectRole.Role.Permissions {
			entry := PermissionV2Entry{}

			if actionRaw, ok := permMap["action"].([]interface{}); ok {
				actions := make([]string, len(actionRaw))
				for i, v := range actionRaw {
					actions[i] = v.(string)
				}
				entry.Action = actions
			}

			if subject, ok := permMap["subject"].(string); ok {
				entry.Subject = subject
			}

			if inverted, ok := permMap["inverted"].(bool); ok {
				entry.Inverted = inverted
			}

			if conditions, ok := permMap["conditions"].(map[string]any); ok {
				entry.Conditions = make(map[string]map[string]string)

				for field, ops := range conditions {
					if opsMap, ok := ops.(map[string]any); ok {
						entry.Conditions[field] = make(map[string]string)

						for op, value := range opsMap {
							if strValue, ok := value.(string); ok {
								entry.Conditions[field][op] = strValue
							}
						}
					}
				}
			}

			permissions[i] = entry
		}

		state.PermissionsV2 = permissions
	}

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

	if (!plan.Permissions.IsNull() && plan.PermissionsV2 != nil) || (plan.Permissions.IsNull() && plan.PermissionsV2 == nil) {
		resp.Diagnostics.AddError(
			"Error updating project role",
			"Define either the permissions or permissions_v2 property but not both.",
		)
		return
	}

	// Permissions V1
	if !plan.Permissions.IsNull() {
		permissions := []projectRoleResourcePermissions{}
		diags = plan.Permissions.ElementsAs(ctx, &permissions, true)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		projectRolePermissions := make([]infisicalclient.ProjectRolePermissionRequest, 0, len(permissions))
		for _, el := range permissions {
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
	} else {
		project, err := r.client.GetProject(infisical.GetProjectRequest{
			Slug: plan.ProjectSlug.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project role",
				"Unexpected error: "+err.Error(),
			)
			return
		}

		permissions := make([]map[string]any, len(plan.PermissionsV2))
		for i, perm := range plan.PermissionsV2 {
			permMap := map[string]any{
				"action":   perm.Action,
				"subject":  perm.Subject,
				"inverted": perm.Inverted,
			}

			if perm.Conditions != nil {
				permMap["conditions"] = perm.Conditions
			}

			permissions[i] = permMap
		}

		_, err = r.client.UpdateProjectRoleV2(infisical.UpdateProjectRoleV2Request{
			ProjectId:   project.ID,
			RoleId:      plan.ID.ValueString(),
			Slug:        plan.Slug.ValueString(),
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Permissions: permissions,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project role",
				"Couldn't update project role from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
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
