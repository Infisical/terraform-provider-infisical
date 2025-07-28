package resource

import (
	"context"
	"encoding/json"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &ProjectTemplateResource{}
)

type ProjectTemplateResource struct {
	client *infisical.Client
}

type PermissionModel struct {
	Subject   types.String `tfsdk:"subject"`
	Action    types.Set    `tfsdk:"action"`
	Condition types.String `tfsdk:"condition"`
	Inverted  types.Bool   `tfsdk:"inverted"`
}

type RoleModel struct {
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	Permissions types.List   `tfsdk:"permissions"`
}

type EnvironmentModel struct {
	Name     types.String `tfsdk:"name"`
	Slug     types.String `tfsdk:"slug"`
	Position types.Int64  `tfsdk:"position"`
}

type ProjectTemplateResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Roles        types.List   `tfsdk:"roles"`
	Environments types.List   `tfsdk:"environments"`
	Type         types.String `tfsdk:"type"`
}

func NewProjectTemplateResource() resource.Resource {
	return &ProjectTemplateResource{}
}

// Metadata returns the resource type name.
func (r *ProjectTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_template"
}

func (r *ProjectTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create project templates & save to Infisical. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the project template",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the project template",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the project template",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the project template. Refer to the documentation here https://infisical.com/docs/api-reference/endpoints/project-templates/create#body-type for the available options",
				Required:    true,
			},
			"roles": schema.ListNestedAttribute{
				Description: "The roles for the project template",
				Computed:    true,
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the role",
							Required:    true,
						},
						"slug": schema.StringAttribute{
							Description: "The slug of the role",
							Required:    true,
						},
						"permissions": schema.ListNestedAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The permissions assigned to the role. Refer to the documentation here https://infisical.com/docs/api-reference/endpoints/project-templates/create#body-roles-permissions for its usage.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"action": schema.SetAttribute{
										ElementType: types.StringType,
										Description: "Describe what actions an entity can take.",
										Required:    true,
									},
									"subject": schema.StringAttribute{
										Description: "Describe the entity the permission pertains to.",
										Required:    true,
									},
									"inverted": schema.BoolAttribute{
										Description: "Whether rule forbids. Set this to true if permission forbids.",
										Optional:    true,
										Default:     booldefault.StaticBool(false),
										Computed:    true,
									},
									"conditions": schema.StringAttribute{
										Optional:    true,
										Description: "When specified, only matching conditions will be allowed to access given resource. Refer to the documentation in https://infisical.com/docs/internals/permissions#conditions for the complete list of supported properties and operators.",
										PlanModifiers: []planmodifier.String{
											pkg.JsonEquivalentModifier{},
										},
										Validators: []validator.String{
											infisicaltf.JsonStringValidator,
										},
									},
								},
							},
						},
					},
				},
			},
			"environments": schema.ListNestedAttribute{
				Optional:    true,
				Description: "The environments for the project template",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the environment",
							Required:    true,
						},
						"slug": schema.StringAttribute{
							Description: "The slug of the environment",
							Required:    true,
						},
						"position": schema.Int64Attribute{
							Description: "The position of the environment",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ProjectTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Read the plan data into the resource model
	var plan ProjectTemplateResourceModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Map the plan data to the Infisical CreateProjectTemplateRequest
	environments := []infisical.Environment{}

	if !plan.Environments.IsNull() || !plan.Environments.IsUnknown() {
		environments, diags = r.unmarshalEnvironments(plan.Environments)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	roles := []infisical.Role{}

	if !plan.Roles.IsNull() || !plan.Roles.IsUnknown() {
		roles, diags = r.unmarshalRoles(plan.Roles)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	res, err := r.client.CreateProjectTemplate(infisical.CreateProjectTemplateRequest{
		Name:         plan.Name.ValueString(),
		Description:  plan.Description.ValueString(),
		Type:         plan.Type.ValueString(),
		Environments: environments,
		Roles:        roles,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project template",
			fmt.Sprintf("Could not create project template: %s", err.Error()),
		)
		return
	}

	// Map the Infisical project template to the resource model
	plan.ID = types.StringValue(res.ID)
	plan.Name = types.StringValue(res.Name)
	plan.Description = types.StringValue(res.Description)
	plan.Type = types.StringValue(res.Type)

	plan.Roles, diags = r.marshalRoles(res.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Environments, diags = r.marshalEnvironments(res.Environments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ProjectTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Read the state data into the resource model
	var plan ProjectTemplateResourceModel
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	// Fetch the project template from Infisical
	template, err := r.client.GetProjectTemplateById(plan.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project template",
			fmt.Sprintf("Could not read project template with ID %s: %s", plan.ID.ValueString(), err.Error()),
		)
		return
	}

	// Map the Infisical project template to the resource model
	plan.ID = types.StringValue(template.ID)
	plan.Name = types.StringValue(template.Name)
	plan.Description = types.StringValue(template.Description)
	plan.Type = types.StringValue(template.Type)

	plan.Environments, diags = r.marshalEnvironments(template.Environments)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	plan.Roles, diags = r.marshalRoles(template.Roles)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

}

func (r *ProjectTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan ProjectTemplateResourceModel
	var state ProjectTemplateResourceModel

	// Step 1: Get new planned config
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 2: Get current state (optional, useful for diffs)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var roles []infisical.Role
	var environments []infisical.Environment

	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		roles, diags = r.unmarshalRoles(plan.Roles)
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Environments.IsNull() && !plan.Environments.IsUnknown() {
		environments, diags = r.unmarshalEnvironments(plan.Environments)
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.UpdateProjectTemplate(infisical.UpdateProjectTemplateRequest{
		ID:           plan.ID.ValueString(),
		Name:         plan.Name.ValueString(),
		Description:  plan.Description.ValueString(),
		Type:         plan.Type.ValueString(),
		Roles:        roles,
		Environments: environments,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project template",
			fmt.Sprintf("Could not update project template with ID %s: %s", plan.ID.ValueString(), err.Error()),
		)
		return
	}

	plan.Name = types.StringValue(apiResp.Name)
	plan.Description = types.StringValue(apiResp.Description)
	plan.Type = types.StringValue(apiResp.Type)

	plan.Roles, diags = r.marshalRoles(apiResp.Roles)
	resp.Diagnostics.Append(diags...)

	plan.Environments, diags = r.marshalEnvironments(apiResp.Environments)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ProjectTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Read the state data into the resource model
	var state ProjectTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the Infisical API to delete the project template
	_, err := r.client.DeleteProjectTemplate(state.ID.ValueString())

	if err != nil {
		if err == infisical.ErrNotFound {
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting project template",
			fmt.Sprintf("Could not delete project template with ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringNull()
}

func (r ProjectTemplateResource) marshalRoles(roles []infisical.Role) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	var tfValues []attr.Value

	// Define the nested types
	permissionType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"subject":    types.StringType,
			"action":     types.SetType{ElemType: types.StringType},
			"conditions": types.StringType,
			"inverted":   types.BoolType,
		},
	}

	roleType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":        types.StringType,
			"slug":        types.StringType,
			"permissions": types.ListType{ElemType: permissionType},
		},
	}

	for _, role := range roles {
		if isDefaultRole(role.Slug) {
			continue
		}

		permissions := []attr.Value{}

		for _, p := range role.Permissions {
			var actionValues []attr.Value
			for _, a := range p.Action {
				actionValues = append(actionValues, types.StringValue(a))
			}

			actions, actionDiags := types.SetValue(types.StringType, actionValues)
			diags.Append(actionDiags...)

			values := map[string]attr.Value{
				"subject":    types.StringValue(p.Subject),
				"action":     actions,
				"conditions": types.StringNull(), // Default to null if no conditions
				"inverted":   types.BoolValue(p.Inverted),
			}

			if p.Conditions != nil {
				encodedConditions, err := json.Marshal(p.Conditions)
				if err != nil {
					diags.AddError(
						"Error marshalling conditions",
						fmt.Sprintf("Could not marshal conditions for permission: %s", err.Error()),
					)
					continue
				}

				values["conditions"] = types.StringValue(string(encodedConditions))
			}

			// Create the permission object
			permObj, permDiags := types.ObjectValue(permissionType.AttrTypes, values)

			diags.Append(permDiags...)

			permissions = append(permissions, permObj)
		}

		// Build the permissions list
		permissionsList, permissionsDiags := types.ListValue(permissionType, permissions)
		diags.Append(permissionsDiags...)

		// Create the role object
		roleObj, roleDiags := types.ObjectValue(roleType.AttrTypes, map[string]attr.Value{
			"name":        types.StringValue(role.Name),
			"slug":        types.StringValue(role.Slug),
			"permissions": permissionsList,
		})
		diags.Append(roleDiags...)

		tfValues = append(tfValues, roleObj)
	}

	// Build the top-level list of roles
	tfList, listDiags := types.ListValue(roleType, tfValues)
	diags.Append(listDiags...)

	return tfList, diags
}

func (r ProjectTemplateResource) unmarshalRoles(tfRoles types.List) ([]infisical.Role, diag.Diagnostics) {
	roles := make([]infisical.Role, len(tfRoles.Elements()))
	var diags diag.Diagnostics

	for index, roleVal := range tfRoles.Elements() {
		roleObj, ok := roleVal.(types.Object)
		if !ok {
			continue
		}

		attrs := roleObj.Attributes()

		role := infisical.Role{}

		if name, ok := attrs["name"].(types.String); ok {
			role.Name = name.ValueString()
		}

		if slug, ok := attrs["slug"].(types.String); ok {
			role.Slug = slug.ValueString()
		}

		if isDefaultRole(role.Slug) {
			continue
		}

		if list, ok := attrs["permissions"].(types.List); ok {
			permissions := []infisical.Permission{}

			for _, permVal := range list.Elements() {

				permission := infisical.Permission{
					Conditions: make(map[string]any),
				}

				permObj, ok := permVal.(types.Object)

				if !ok {
					diags.AddError(
						"Invalid Permission Object",
						"Expected a valid permission object, but got an invalid type.",
					)
					continue
				}

				attrs := permObj.Attributes()

				if subject, ok := attrs["subject"].(types.String); ok {
					if subject.IsNull() || subject.IsUnknown() {
						diags.AddError(
							"Invalid Permission Subject",
							"Expected a valid permission subject string, but got an invalid type.",
						)
						continue
					}

					permission.Subject = subject.ValueString()
				}

				if inverted, ok := attrs["inverted"].(types.Bool); ok {
					permission.Inverted = inverted.ValueBool()
				}

				if actionList, ok := attrs["action"].(types.Set); ok {
					if actionList.IsNull() || actionList.IsUnknown() {
						diags.AddError(
							"Invalid Permission Action",
							"Expected a valid permission action list, but got an invalid type.",
						)
					}

					var actions []string
					for _, act := range actionList.Elements() {
						actionStr, ok := act.(types.String)
						if !ok {
							diags.AddError(
								"Invalid Action Type",
								"Expected a string for action, but got an invalid type.",
							)
							continue
						}
						actions = append(actions, actionStr.ValueString())
					}

					permission.Action = actions
				}

				if condition, ok := attrs["conditions"].(types.String); ok && !condition.IsNull() {
					err := json.Unmarshal([]byte(condition.ValueString()), &permission.Conditions)

					if err != nil {
						diags.AddError(
							"Error unmarshalling conditions",
							fmt.Sprintf("Could not unmarshal conditions for permission: %s", err.Error()),
						)
					}
				}

				permissions = append(permissions, permission)
			}

			role.Permissions = permissions
		}

		roles[index] = role
	}

	return roles, diags
}

func (r ProjectTemplateResource) marshalEnvironments(envs []infisical.Environment) (types.List, diag.Diagnostics) {
	var envValues []attr.Value

	// Define the Terraform type schema for each environment object
	envObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":     types.StringType,
			"slug":     types.StringType,
			"position": types.Int64Type,
		},
	}

	for _, env := range envs {
		obj, _ := types.ObjectValue(envObjectType.AttrTypes, map[string]attr.Value{
			"name":     types.StringValue(env.Name),
			"slug":     types.StringValue(env.Slug),
			"position": types.Int64Value(env.Position),
		})
		envValues = append(envValues, obj)
	}

	return types.ListValue(envObjectType, envValues)
}

func (r ProjectTemplateResource) unmarshalEnvironments(tfList types.List) ([]infisical.Environment, diag.Diagnostics) {
	var envs []infisical.Environment
	var diags diag.Diagnostics

	// Make sure we only process if the list is known and not null
	if tfList.IsNull() || tfList.IsUnknown() {
		return envs, diags
	}

	// Iterate over each element in the Terraform list
	for _, elem := range tfList.Elements() {
		objVal, ok := elem.(types.Object)
		if !ok {
			diags.AddError(
				"Invalid Environment Object",
				"Expected a valid environment object, but got an invalid type.",
			)
			continue
		}

		attrs := objVal.Attributes()

		name, _ := attrs["name"].(types.String)
		slug, _ := attrs["slug"].(types.String)
		position, _ := attrs["position"].(types.Int64)

		env := infisical.Environment{
			Name:     name.ValueString(),
			Slug:     slug.ValueString(),
			Position: position.ValueInt64(),
		}

		envs = append(envs, env)
	}

	return envs, diags
}

func isDefaultRole(slug string) bool {
	return slug == "admin" || slug == "member" || slug == "viewer" || slug == "no-access"
}
