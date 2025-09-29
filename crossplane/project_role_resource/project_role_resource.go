package resource

import (
	"context"
	"encoding/json"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
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

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectRoleResource() resource.Resource {
	return &projectRoleResource{}
}

// projectRoleResource is the resource implementation.
type projectRoleResource struct {
	client *infisical.Client
}

type ProjectRolePermissionV2Entry struct {
	Action     types.Set    `tfsdk:"action"`
	Subject    types.String `tfsdk:"subject"`
	Inverted   types.Bool   `tfsdk:"inverted"`
	Conditions types.String `tfsdk:"conditions"`
}

type ProjectRolePermissionV2JSON struct {
	Action     []string               `json:"action"`
	Subject    string                 `json:"subject"`
	Inverted   *bool                  `json:"inverted,omitempty"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// projectRoleResourceModel describes the resource model.
type projectRoleResourceModel struct {
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Slug          types.String `tfsdk:"slug"`
	ProjectSlug   types.String `tfsdk:"project_slug"`
	ID            types.String `tfsdk:"id"`
	PermissionsV2 types.String `tfsdk:"permissions"`
}

// Metadata returns the resource type name.
func (r *projectRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_role"
}

// Schema defines the schema for the resource.
func (r *projectRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create custom project roles & save to Infisical. Only Machine Identity authentication is supported for this resource.",
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
			"permissions": schema.StringAttribute{
				Optional:    true,
				Description: "The permissions assigned to the project role. Refer to the documentation here https://infisical.com/docs/internals/permissions for its usage. Legacy permissions (V1) is not supported for this resource.",

				PlanModifiers: []planmodifier.String{
					pkg.UnorderedJsonEquivalentModifier{},
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

	// Permissions V2
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

	var permissionsV2JSON []ProjectRolePermissionV2JSON
	err = json.Unmarshal([]byte(plan.PermissionsV2.ValueString()), &permissionsV2JSON)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project role",
			"Failed to parse permissions JSON: "+err.Error(),
		)
		return
	}

	var permissionsV2 []ProjectRolePermissionV2Entry
	for _, perm := range permissionsV2JSON {
		// Convert action slice to types.Set
		actionElements := make([]attr.Value, len(perm.Action))
		for i, action := range perm.Action {
			actionElements[i] = types.StringValue(action)
		}
		actionSet, _ := types.SetValue(types.StringType, actionElements)

		// Convert conditions map to JSON string - FIX: Handle null case properly
		var conditionsValue types.String
		if perm.Conditions != nil {
			conditionsBytes, _ := json.Marshal(perm.Conditions)
			conditionsValue = types.StringValue(string(conditionsBytes))
		} else {
			conditionsValue = types.StringNull() // Use null instead of empty string
		}

		entry := ProjectRolePermissionV2Entry{
			Action:     actionSet,
			Subject:    types.StringValue(perm.Subject),
			Inverted:   types.BoolValue(perm.Inverted != nil && *perm.Inverted),
			Conditions: conditionsValue, // Now properly null when no conditions
		}

		permissionsV2 = append(permissionsV2, entry)
	}

	permissions := make([]map[string]any, len(permissionsV2))
	for i, perm := range permissionsV2 {
		actionValues := perm.Action.Elements()
		actionStrings := make([]string, 0, len(actionValues))
		for _, v := range actionValues {
			if strVal, ok := v.(types.String); ok {
				actionStrings = append(actionStrings, strVal.ValueString())
			}
		}

		permMap := map[string]any{
			"action":   actionStrings,
			"subject":  perm.Subject.ValueString(),
			"inverted": perm.Inverted.ValueBool(),
		}

		// FIX: Now this check will work correctly
		if !perm.Conditions.IsNull() && perm.Conditions.ValueString() != "" {
			var conditionsMap map[string]interface{}
			if err := json.Unmarshal([]byte(perm.Conditions.ValueString()), &conditionsMap); err != nil {
				resp.Diagnostics.AddError(
					"Error creating project role",
					"Error parsing conditions property: "+err.Error(),
				)
				return
			}

			permMap["conditions"] = conditionsMap
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
			"Couldn't save project role to Infisical, unexpected error: "+err.Error(),
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

	// Permissions V2

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
			"Couldn't read project role from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Description = types.StringValue(projectRole.Role.Description)
	state.ID = types.StringValue(projectRole.Role.ID)
	state.Name = types.StringValue(projectRole.Role.Name)

	permissionsJSON := make([]ProjectRolePermissionV2JSON, len(projectRole.Role.Permissions))
	for i, permMap := range projectRole.Role.Permissions {
		jsonEntry := ProjectRolePermissionV2JSON{}

		if actionRaw, ok := permMap["action"].([]interface{}); ok {
			actions := make([]string, len(actionRaw))
			for i, v := range actionRaw {
				if strValue, ok := v.(string); ok {
					actions[i] = strValue
				}
			}
			jsonEntry.Action = actions
		}

		if subject, ok := permMap["subject"].(string); ok {
			jsonEntry.Subject = subject
		}

		if inverted, ok := permMap["inverted"].(bool); ok {
			if inverted {
				jsonEntry.Inverted = &inverted
			}
			// If false, leave as nil so omitempty will exclude it
		}

		if conditions, ok := permMap["conditions"].(map[string]any); ok {
			jsonEntry.Conditions = conditions
		}

		permissionsJSON[i] = jsonEntry
	}

	permissionV2Bytes, err := json.Marshal(permissionsJSON)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project role",
			"Couldn't parse permissions property, unexpected error: "+err.Error(),
		)
		return
	}

	state.PermissionsV2 = types.StringValue(string(permissionV2Bytes))

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

	var permissionsV2JSON []ProjectRolePermissionV2JSON
	err = json.Unmarshal([]byte(plan.PermissionsV2.ValueString()), &permissionsV2JSON)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project role",
			"Failed to parse permissions JSON: "+err.Error(),
		)
		return
	}

	var permissionsV2 []ProjectRolePermissionV2Entry
	for _, perm := range permissionsV2JSON {
		// Convert action slice to types.Set
		actionElements := make([]attr.Value, len(perm.Action))
		for i, action := range perm.Action {
			actionElements[i] = types.StringValue(action)
		}
		actionSet, _ := types.SetValue(types.StringType, actionElements)

		// Convert conditions map to JSON string - FIX: Handle null case properly
		var conditionsValue types.String
		if perm.Conditions != nil {
			conditionsBytes, _ := json.Marshal(perm.Conditions)
			conditionsValue = types.StringValue(string(conditionsBytes))
		} else {
			conditionsValue = types.StringNull() // Use null instead of empty string
		}

		entry := ProjectRolePermissionV2Entry{
			Action:     actionSet,
			Subject:    types.StringValue(perm.Subject),
			Inverted:   types.BoolValue(perm.Inverted != nil && *perm.Inverted),
			Conditions: conditionsValue, // Now properly null when no conditions
		}

		permissionsV2 = append(permissionsV2, entry)
	}

	permissions := make([]map[string]any, len(permissionsV2))
	for i, perm := range permissionsV2 {
		actionValues := perm.Action.Elements()
		actionStrings := make([]string, 0, len(actionValues))
		for _, v := range actionValues {
			if strVal, ok := v.(types.String); ok {
				actionStrings = append(actionStrings, strVal.ValueString())
			}
		}

		permMap := map[string]any{
			"action":   actionStrings,
			"subject":  perm.Subject.ValueString(),
			"inverted": perm.Inverted.ValueBool(),
		}

		// FIX: Now this check will work correctly
		if !perm.Conditions.IsNull() && perm.Conditions.ValueString() != "" {
			var conditionsMap map[string]interface{}
			if err := json.Unmarshal([]byte(perm.Conditions.ValueString()), &conditionsMap); err != nil {
				resp.Diagnostics.AddError(
					"Error creating project role",
					"Error parsing conditions property: "+err.Error(),
				)
				return
			}

			permMap["conditions"] = conditionsMap
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
			"Couldn't update project role from Infisical, unexpected error: "+err.Error(),
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
			"Couldn't delete project role from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

}
