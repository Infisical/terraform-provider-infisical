package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	Subject    types.String `tfsdk:"subject"`
	Action     types.Set    `tfsdk:"action"`
	Conditions types.String `tfsdk:"conditions"`
	Inverted   *types.Bool  `tfsdk:"inverted"`
}

type EnvironmentJSON struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Position int64  `json:"position"`
}

type PermissionJSON struct {
	Subject    string                 `json:"subject"`
	Action     []string               `json:"action"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
	Inverted   *bool                  `json:"inverted,omitempty"`
}

type RoleJSON struct {
	Name        string           `json:"name"`
	Slug        string           `json:"slug"`
	Permissions []PermissionJSON `json:"permissions,omitempty"`
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
	Roles        types.String `tfsdk:"roles"`
	Environments types.String `tfsdk:"environments"`
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
			"roles": schema.StringAttribute{
				Description: "The roles for the project template",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					pkg.UnorderedJsonEquivalentModifier{},
				},
				Validators: []validator.String{
					infisicaltf.JsonStringValidator,
				},
			},
			"environments": schema.StringAttribute{
				Description: "The environments for the project template as a JSON string",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					pkg.UnorderedJsonEquivalentModifier{},
				},
				Validators: []validator.String{
					infisicaltf.JsonStringValidator,
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

	var environments []infisical.Environment
	if !plan.Environments.IsNull() && plan.Environments.ValueString() != "" {
		var envJSON []EnvironmentJSON
		if err := json.Unmarshal([]byte(plan.Environments.ValueString()), &envJSON); err != nil {
			resp.Diagnostics.AddError(
				"Invalid environments JSON",
				"Could not parse environments JSON: "+err.Error(),
			)
			return
		}

		for _, e := range envJSON {
			environments = append(environments, infisical.Environment{
				Name:     e.Name,
				Slug:     e.Slug,
				Position: e.Position,
			})
		}
	}

	// Parse roles from JSON string
	var roles []infisical.Role
	if !plan.Roles.IsNull() && plan.Roles.ValueString() != "" {
		var rolesJSON []RoleJSON
		if err := json.Unmarshal([]byte(plan.Roles.ValueString()), &rolesJSON); err != nil {
			resp.Diagnostics.AddError(
				"Invalid roles JSON",
				"Could not parse roles JSON: "+err.Error(),
			)
			return
		}

		for _, r := range rolesJSON {
			role := infisical.Role{
				Name: r.Name,
				Slug: r.Slug,
			}

			for _, p := range r.Permissions {
				perm := infisical.Permission{
					Subject:    p.Subject,
					Action:     p.Action,
					Conditions: p.Conditions,
				}
				if p.Inverted != nil {
					perm.Inverted = *p.Inverted
				}
				role.Permissions = append(role.Permissions, perm)
			}

			roles = append(roles, role)
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

	// map the Infisical project template to the resource model
	plan.ID = types.StringValue(res.ID)
	plan.Name = types.StringValue(res.Name)
	plan.Description = types.StringValue(res.Description)
	plan.Type = types.StringValue(res.Type)

	if plan.Roles.IsNull() {
		plan.Roles, diags = marshalRolesToJSON(res.Roles)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if plan.Environments.IsNull() {
		plan.Environments, diags = marshalEnvironmentsToJSON(res.Environments)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
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
	var state ProjectTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	// fetch the project template from Infisical
	template, err := r.client.GetProjectTemplateById(state.ID.ValueString())

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading project template",
			fmt.Sprintf("Could not read project template with ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringValue(template.ID)
	state.Name = types.StringValue(template.Name)
	state.Description = types.StringValue(template.Description)
	state.Type = types.StringValue(template.Type)

	envJSON, err := json.Marshal(template.Environments)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling environments", err.Error())
		return
	}
	currentEnv := state.Environments.ValueString()
	if !areJSONEquivalent(currentEnv, string(envJSON)) {
		state.Environments = types.StringValue(string(envJSON))
	}

	// filter out default roles (admin, member, etc)
	var customRoles []infisical.Role
	for _, role := range template.Roles {
		if !isDefaultRole(role.Slug) {
			customRoles = append(customRoles, role)
		}
	}

	rolesJSON, err := json.Marshal(customRoles)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling roles", err.Error())
		return
	}

	currentRoles := state.Roles.ValueString()

	// normalize both to remove default values before comparing
	normalizedCurrent := normalizeJSON(currentRoles)
	normalizedAPI := normalizeJSON(string(rolesJSON))

	if !areJSONEquivalent(normalizedCurrent, normalizedAPI) {
		// actually different, update the state
		state.Roles = types.StringValue(string(rolesJSON))
	}

	diags = resp.State.Set(ctx, state)
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

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// parse environments from JSON string
	var environments []infisical.Environment
	if !plan.Environments.IsNull() && plan.Environments.ValueString() != "" {
		var envJSON []EnvironmentJSON
		if err := json.Unmarshal([]byte(plan.Environments.ValueString()), &envJSON); err != nil {
			resp.Diagnostics.AddError(
				"Invalid environments JSON",
				"Could not parse environments JSON: "+err.Error(),
			)
			return
		}

		for _, e := range envJSON {
			environments = append(environments, infisical.Environment{
				Name:     e.Name,
				Slug:     e.Slug,
				Position: e.Position,
			})
		}
	}

	// parse roles from JSON string
	var roles []infisical.Role
	if !plan.Roles.IsNull() && plan.Roles.ValueString() != "" {
		var rolesJSON []RoleJSON
		if err := json.Unmarshal([]byte(plan.Roles.ValueString()), &rolesJSON); err != nil {
			resp.Diagnostics.AddError(
				"Invalid roles JSON",
				"Could not parse roles JSON: "+err.Error(),
			)
			return
		}

		for _, r := range rolesJSON {
			role := infisical.Role{
				Name: r.Name,
				Slug: r.Slug,
			}

			for _, p := range r.Permissions {
				perm := infisical.Permission{
					Subject:    p.Subject,
					Action:     p.Action,
					Conditions: p.Conditions,
				}
				if p.Inverted != nil {
					perm.Inverted = *p.Inverted
				}
				role.Permissions = append(role.Permissions, perm)
			}

			roles = append(roles, role)
		}
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
}

func isDefaultRole(slug string) bool {
	return slug == "admin" || slug == "member" || slug == "viewer" || slug == "no-access"
}

func marshalRolesToJSON(roles []infisical.Role) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(roles) == 0 {
		return types.StringNull(), diags
	}

	var rolesJSON []RoleJSON
	for _, role := range roles {
		if isDefaultRole(role.Slug) {
			continue
		}

		roleJSON := RoleJSON{
			Name: role.Name,
			Slug: role.Slug,
		}

		for _, p := range role.Permissions {
			permJSON := PermissionJSON{
				Subject:    p.Subject,
				Action:     p.Action,
				Conditions: p.Conditions,
			}

			// set inverted if true
			if p.Inverted {
				inverted := true
				permJSON.Inverted = &inverted
			}

			roleJSON.Permissions = append(roleJSON.Permissions, permJSON)
		}

		rolesJSON = append(rolesJSON, roleJSON)
	}

	jsonBytes, err := json.Marshal(rolesJSON)
	if err != nil {
		diags.AddError(
			"Error marshaling roles",
			"Could not marshal roles to JSON: "+err.Error(),
		)
		return types.StringNull(), diags
	}

	return types.StringValue(string(jsonBytes)), diags
}

func marshalEnvironmentsToJSON(environments []infisical.Environment) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(environments) == 0 {
		return types.StringNull(), diags
	}

	var envJSON []EnvironmentJSON
	for _, env := range environments {
		envJSON = append(envJSON, EnvironmentJSON{
			Name:     env.Name,
			Slug:     env.Slug,
			Position: env.Position,
		})
	}

	jsonBytes, err := json.Marshal(envJSON)
	if err != nil {
		diags.AddError(
			"Error marshaling environments",
			"Could not marshal environments to JSON: "+err.Error(),
		)
		return types.StringNull(), diags
	}

	return types.StringValue(string(jsonBytes)), diags
}

func areJSONEquivalent(json1, json2 string) bool {
	var obj1, obj2 interface{}

	if err := json.Unmarshal([]byte(json1), &obj1); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(json2), &obj2); err != nil {
		return false
	}

	return reflect.DeepEqual(obj1, obj2)
}

func normalizeJSON(jsonStr string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	normalized := removeDefaults(data)
	result, _ := json.Marshal(normalized)
	return string(result)
}

func removeDefaults(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			if key == "inverted" {
				if boolVal, ok := val.(bool); ok && !boolVal {
					continue // skip inverted when false
				}
			}
			result[key] = removeDefaults(val)
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = removeDefaults(item)
		}
		return result

	default:
		return data
	}
}
