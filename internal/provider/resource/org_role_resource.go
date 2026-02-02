package resource

import (
	"context"
	"encoding/json"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &orgRoleResource{}
)

// NewOrgRoleResource is a helper function to simplify the provider implementation.
func NewOrgRoleResource() resource.Resource {
	return &orgRoleResource{}
}

// orgRoleResource is the resource implementation.
type orgRoleResource struct {
	client *infisical.Client
}

type OrgRolePermissionEntry struct {
	Action     types.Set    `tfsdk:"action"`
	Subject    types.String `tfsdk:"subject"`
	Inverted   types.Bool   `tfsdk:"inverted"`
	Conditions types.String `tfsdk:"conditions"`
}

// orgRoleResourceModel describes the data source data model.
type orgRoleResourceModel struct {
	Name        types.String             `tfsdk:"name"`
	Description types.String             `tfsdk:"description"`
	Slug        types.String             `tfsdk:"slug"`
	ID          types.String             `tfsdk:"id"`
	Permissions []OrgRolePermissionEntry `tfsdk:"permissions"`
}

// Metadata returns the resource type name.
func (r *orgRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_role"
}

// Schema defines the schema for the resource.
func (r *orgRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create custom organization roles & save to Infisical. Only Machine Identity authentication is supported for this data source.",
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
			"id": schema.StringAttribute{
				Description:   "The ID of the role",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"permissions": schema.ListNestedAttribute{
				Required:    true,
				Description: "The permissions assigned to the organization role. Refer to the documentation here https://infisical.com/docs/internals/permissions for its usage.",
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
	}
}

// Configure adds the provider configured client to the resource.
func (r *orgRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *orgRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create organization role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan orgRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Permissions == nil {
		resp.Diagnostics.AddError(
			"Error creating organization role",
			"The permission property is required.",
		)
		return
	}

	permissions := make([]map[string]any, len(plan.Permissions))
	for i, perm := range plan.Permissions {
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

		// parse as object
		if !perm.Conditions.IsNull() {
			var conditionsMap map[string]interface{}
			if err := json.Unmarshal([]byte(perm.Conditions.ValueString()), &conditionsMap); err != nil {
				resp.Diagnostics.AddError(
					"Error creating organization role",
					"Error parsing conditions property: "+err.Error(),
				)
				return
			}

			permMap["conditions"] = conditionsMap
		}

		permissions[i] = permMap
	}

	newOrgRole, err := r.client.CreateOrgRole(infisical.CreateOrgRoleRequest{
		Slug:        plan.Slug.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Permissions: permissions,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating organization role",
			"Couldn't save organization role to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newOrgRole.Role.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *orgRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read organization role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state orgRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgRole, err := r.client.GetOrgRoleById(infisical.GetOrgRoleByIdRequest{
		RoleId: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading organization role",
			"Couldn't read organization role from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Description = types.StringValue(orgRole.Role.Description)
	state.ID = types.StringValue(orgRole.Role.ID)
	state.Name = types.StringValue(orgRole.Role.Name)
	state.Slug = types.StringValue(orgRole.Role.Slug)

	permissions := make([]OrgRolePermissionEntry, len(orgRole.Role.Permissions))
	for i, permMap := range orgRole.Role.Permissions {
		entry := OrgRolePermissionEntry{}

		if actionRaw, ok := permMap["action"].([]interface{}); ok {
			actions := make([]string, len(actionRaw))
			for j, v := range actionRaw {
				if strValue, ok := v.(string); ok {
					actions[j] = strValue
				} else {
					resp.Diagnostics.AddError(
						"Invalid Action Type",
						fmt.Sprintf("Expected string type for action at index %d, got %T", j, v),
					)
					return
				}
			}

			entry.Action, diags = types.SetValueFrom(ctx, types.StringType, actions)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		if subject, ok := permMap["subject"].(string); ok {
			entry.Subject = types.StringValue(subject)
		}

		if inverted, ok := permMap["inverted"].(bool); ok {
			entry.Inverted = types.BoolValue(inverted)
		} else {
			entry.Inverted = types.BoolValue(false)
		}

		if conditions, ok := permMap["conditions"].(map[string]any); ok {
			conditionsBytes, err := json.Marshal(conditions)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error reading organization role",
					"Couldn't parse conditions property, unexpected error: "+err.Error(),
				)
				return
			}

			entry.Conditions = types.StringValue(string(conditionsBytes))
		}

		permissions[i] = entry
	}

	state.Permissions = permissions

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *orgRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update organization role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan orgRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state orgRoleResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Permissions == nil {
		resp.Diagnostics.AddError(
			"Error updating organization role",
			"The permission property is required.",
		)
		return
	}

	permissions := make([]map[string]any, len(plan.Permissions))
	for i, perm := range plan.Permissions {
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

		if !perm.Conditions.IsNull() {
			var conditionsMap map[string]interface{}
			if err := json.Unmarshal([]byte(perm.Conditions.ValueString()), &conditionsMap); err != nil {
				resp.Diagnostics.AddError(
					"Error updating organization role",
					"Error parsing conditions property: "+err.Error(),
				)
				return
			}

			permMap["conditions"] = conditionsMap
		}

		permissions[i] = permMap
	}

	_, err := r.client.UpdateOrgRole(infisical.UpdateOrgRoleRequest{
		RoleId:      state.ID.ValueString(),
		Slug:        plan.Slug.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Permissions: permissions,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating organization role",
			"Couldn't update organization role from Infisical, unexpected error: "+err.Error(),
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
func (r *orgRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete organization role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state orgRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteOrgRole(infisical.DeleteOrgRoleRequest{
		RoleId: state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting organization role",
			"Couldn't delete organization role from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *orgRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import organization role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	orgRole, err := r.client.GetOrgRoleById(infisical.GetOrgRoleByIdRequest{
		RoleId: req.ID,
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Organization role not found",
				"The organization role with the given ID was not found",
			)
		} else {
			resp.Diagnostics.AddError(
				"Error fetching organization role",
				"Couldn't fetch organization role from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), orgRole.Role.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), orgRole.Role.Slug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), orgRole.Role.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), orgRole.Role.Description)...)

	permissions := make([]OrgRolePermissionEntry, len(orgRole.Role.Permissions))
	for i, permMap := range orgRole.Role.Permissions {
		entry := OrgRolePermissionEntry{}

		if actionRaw, ok := permMap["action"].([]interface{}); ok {
			actions := make([]string, len(actionRaw))
			for j, v := range actionRaw {
				if strValue, ok := v.(string); ok {
					actions[j] = strValue
				}
			}

			actionSet, diags := types.SetValueFrom(ctx, types.StringType, actions)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			entry.Action = actionSet
		}

		if subject, ok := permMap["subject"].(string); ok {
			entry.Subject = types.StringValue(subject)
		}

		if inverted, ok := permMap["inverted"].(bool); ok {
			entry.Inverted = types.BoolValue(inverted)
		} else {
			entry.Inverted = types.BoolValue(false)
		}

		if conditions, ok := permMap["conditions"].(map[string]any); ok {
			conditionsBytes, err := json.Marshal(conditions)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error importing organization role",
					"Couldn't parse conditions property, unexpected error: "+err.Error(),
				)
				return
			}
			entry.Conditions = types.StringValue(string(conditionsBytes))
		}

		permissions[i] = entry
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permissions"), permissions)...)
}
