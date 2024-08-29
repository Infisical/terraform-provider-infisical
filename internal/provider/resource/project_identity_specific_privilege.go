package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_                                      resource.Resource = &projectIdentitySpecificPrivilegeResourceResource{}
	SPECIFIC_PRIVILEGE_PERMISSION_ACTIONS                    = []string{"create", "edit", "delete", "read"}
	SPECIFIC_PRIVILEGE_PERMISSION_SUBJECTS                   = []string{"secrets"}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectIdentitySpecificPrivilegeResource() resource.Resource {
	return &projectIdentitySpecificPrivilegeResourceResource{}
}

// projectIdentitySpecificPrivilegeResourceResource is the resource implementation.
type projectIdentitySpecificPrivilegeResourceResource struct {
	client *infisical.Client
}

// projectIdentitySpecificPrivilegeResourceResourceSourceModel describes the data source data model.
type projectIdentitySpecificPrivilegeResourceResourceModel struct {
	Slug                    types.String                                                `tfsdk:"slug"`
	ProjectSlug             types.String                                                `tfsdk:"project_slug"`
	IdentityID              types.String                                                `tfsdk:"identity_id"`
	ID                      types.String                                                `tfsdk:"id"`
	Permission              projectIdentitySpecificPrivilegeResourceResourcePermissions `tfsdk:"permission"`
	IsTemporary             types.Bool                                                  `tfsdk:"is_temporary"`
	TemporaryMode           types.String                                                `tfsdk:"temporary_mode"`
	TemporaryRange          types.String                                                `tfsdk:"temporary_range"`
	TemporaryAccesStartTime types.String                                                `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime  types.String                                                `tfsdk:"temporary_access_end_time"`
}

type projectIdentitySpecificPrivilegeResourceResourcePermissions struct {
	Actions    types.List                                                          `tfsdk:"actions"`
	Subject    types.String                                                        `tfsdk:"subject"`
	Conditions projectIdentitySpecificPrivilegeResourceResourcePermissionCondition `tfsdk:"conditions"`
}

type projectIdentitySpecificPrivilegeResourceResourcePermissionCondition struct {
	Environment types.String `tfsdk:"environment"`
	SecretPath  types.String `tfsdk:"secret_path"`
}

// Metadata returns the resource type name.
func (r *projectIdentitySpecificPrivilegeResourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_identity_specific_privilege"
}

// Schema defines the schema for the resource.
func (r *projectIdentitySpecificPrivilegeResourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create additional privileges for identities & save to Infisical. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"identity_id": schema.StringAttribute{
				Description: "The identity id to create identity specific privilege",
				Required:    true,
			},
			"project_slug": schema.StringAttribute{
				Description: "The slug of the project to create identity specific privilege",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug for the new privilege",
				Optional:    true,
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the privilege",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_temporary": schema.BoolAttribute{
				Description: "Flag to indicate the assigned specific privilege is temporary or not. When is_temporary is true fields temporary_mode, temporary_range and temporary_access_start_time is required.",
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
			"permission": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The permissions assigned to the project identity specific privilege",
				Attributes: map[string]schema.Attribute{
					"actions": schema.ListAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: fmt.Sprintf("Describe what action an entity can take. Enum: %s", strings.Join(PERMISSION_ACTIONS, ",")),
					},
					"subject": schema.StringAttribute{
						Description: fmt.Sprintf("Describe what action an entity can take. Enum: %s", strings.Join(PERMISSION_SUBJECTS, ",")),
						Required:    true,
					},
					"conditions": schema.SingleNestedAttribute{
						Required:    true,
						Description: "The conditions to scope permissions",
						Attributes: map[string]schema.Attribute{
							"environment": schema.StringAttribute{
								Description: "The environment slug this permission should allow.",
								Required:    true,
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
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectIdentitySpecificPrivilegeResourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectIdentitySpecificPrivilegeResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project identity specific privilege",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectIdentitySpecificPrivilegeResourceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planPermissionActions := make([]types.String, 0, len(plan.Permission.Actions.Elements()))
	diags = plan.Permission.Actions.ElementsAs(ctx, &planPermissionActions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	condition := make(map[string]any)
	environment := plan.Permission.Conditions.Environment.ValueString()
	secretPath := plan.Permission.Conditions.SecretPath.ValueString()
	condition["environment"] = environment
	if secretPath != "" {
		condition["secretPath"] = map[string]string{"$glob": secretPath}
	}

	actions := make([]string, 0, len(planPermissionActions))
	for _, action := range planPermissionActions {
		actions = append(actions, action.ValueString())
	}
	privilegePermission := infisicalclient.ProjectSpecificPrivilegePermissionRequest{
		Actions:    actions,
		Subject:    plan.Permission.Subject.ValueString(),
		Conditions: condition,
	}

	if plan.IsTemporary.ValueBool() {
		temporaryMode := plan.TemporaryMode.ValueString()
		temporaryRange := plan.TemporaryRange.ValueString()
		temporaryAccesStartTime := time.Now().UTC()

		if plan.TemporaryAccesStartTime.ValueString() != "" {
			var err error
			temporaryAccesStartTime, err = time.Parse(time.RFC3339, plan.TemporaryAccesStartTime.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error parsing field TemporaryAccessStartTime",
					fmt.Sprintf("Must provider valid ISO timestamp for field temporaryAccesStartTime %s", plan.TemporaryAccesStartTime.ValueString()),
				)
				return
			}
		}

		// default values
		if temporaryMode == "" {
			temporaryMode = TEMPORARY_MODE_RELATIVE
		}
		if temporaryRange == "" {
			temporaryRange = "1h"
		}

		newProjectRole, err := r.client.CreateTemporaryProjectIdentitySpecificPrivilege(infisical.CreateTemporaryProjectIdentitySpecificPrivilegeRequest{
			ProjectSlug:              plan.ProjectSlug.ValueString(),
			Slug:                     plan.Slug.ValueString(),
			IdentityId:               plan.IdentityID.ValueString(),
			Permissions:              privilegePermission,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating project identity specific privilege",
				"Couldn't save project identity specific privilege to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		plan.ID = types.StringValue(newProjectRole.Privilege.ID)
		plan.TemporaryAccessEndTime = types.StringValue(newProjectRole.Privilege.TemporaryAccessEndTime.Format(time.RFC3339))
		plan.TemporaryAccesStartTime = types.StringValue(newProjectRole.Privilege.TemporaryAccessStartTime.Format(time.RFC3339))
		plan.Slug = types.StringValue(newProjectRole.Privilege.Slug)
		plan.TemporaryRange = types.StringValue(newProjectRole.Privilege.TemporaryRange)
		plan.TemporaryMode = types.StringValue(newProjectRole.Privilege.TemporaryMode)
	} else {
		newProjectRole, err := r.client.CreatePermanentProjectIdentitySpecificPrivilege(infisical.CreatePermanentProjectIdentitySpecificPrivilegeRequest{
			ProjectSlug: plan.ProjectSlug.ValueString(),
			Slug:        plan.Slug.ValueString(),
			IdentityId:  plan.IdentityID.ValueString(),
			Permissions: privilegePermission,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating project identity specific privilege",
				"Couldn't save project identity specific privilege to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		plan.ID = types.StringValue(newProjectRole.Privilege.ID)
		plan.Slug = types.StringValue(newProjectRole.Privilege.Slug)
		plan.TemporaryAccessEndTime = types.StringNull()
		plan.TemporaryAccesStartTime = types.StringNull()
		plan.TemporaryRange = types.StringNull()
		plan.TemporaryMode = types.StringNull()
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectIdentitySpecificPrivilegeResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project identity specific privilege",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectIdentitySpecificPrivilegeResourceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	projectIdentitySpecificPrivilegeResource, err := r.client.GetProjectIdentitySpecificPrivilegeBySlug(infisical.GetProjectIdentitySpecificPrivilegeRequest{
		PrivilegeSlug: state.Slug.ValueString(),
		ProjectSlug:   state.ProjectSlug.ValueString(),
		IdentityID:    state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project identity specific privilege",
			"Couldn't read project identity specific privilege from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.ID)
	if projectIdentitySpecificPrivilegeResource.Privilege.IsTemporary {
		state.TemporaryAccessEndTime = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.TemporaryAccessEndTime.Format(time.RFC3339))
		state.TemporaryAccesStartTime = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.TemporaryAccessStartTime.Format(time.RFC3339))
		state.TemporaryRange = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.TemporaryRange)
		state.TemporaryMode = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.TemporaryMode)
	} else {
		state.TemporaryAccessEndTime = types.StringNull()
		state.TemporaryAccesStartTime = types.StringNull()
		state.TemporaryRange = types.StringNull()
		state.TemporaryMode = types.StringNull()
	}

	planPermissionActions := make([]attr.Value, 0, len(projectIdentitySpecificPrivilegeResource.Privilege.Permissions))
	var planPermissionSubject, planPermissionEnvironment, planPermissionSecretPath types.String
	for _, el := range projectIdentitySpecificPrivilegeResource.Privilege.Permissions {
		action, isValid := el["action"].(string)
		if el["action"] != nil && !isValid {
			action, isValid = el["action"].([]any)[0].(string)
			if !isValid {
				resp.Diagnostics.AddError(
					"Error reading project identity specific privilege",
					"Couldn't read project identity specific privilege from Infiscial, invalid action field in permission",
				)
				return
			}
		}

		subject, isValid := el["subject"].(string)
		if el["subject"] != nil && !isValid {
			subject, isValid = el["subject"].([]any)[0].(string)
			if !isValid {
				resp.Diagnostics.AddError(
					"Error reading project identity specific privilege",
					"Couldn't read project identity specific privilege from Infiscial, invalid subject field in permission",
				)
				return
			}
		}

		conditions, isValid := el["conditions"].(map[string]any)
		if !isValid {
			resp.Diagnostics.AddError(
				"Error reading project identity specific privilege",
				"Couldn't read project identity specific privilege from Infiscial, invalid conditions field in permission",
			)
			return
		}

		planPermissionActions = append(planPermissionActions, types.StringValue(action))
		environment, isValid := conditions["environment"].(string)
		if !isValid {
			resp.Diagnostics.AddError(
				"Error reading project identity specific privilege",
				"Couldn't read project identity specific privilege from Infiscial, invalid environment field in permission",
			)
			return
		}
		planPermissionEnvironment = types.StringValue(environment)

		planPermissionSubject = types.StringValue(subject)
		if val, isValid := conditions["secretPath"].(map[string]any); isValid {
			secretPath, isValid := val["$glob"].(string)
			if !isValid {
				resp.Diagnostics.AddError(
					"Error reading project identity specific privilege",
					"Couldn't read project identity specific privilege from Infiscial, invalid secret path field in permission",
				)
				return
			}
			planPermissionSecretPath = types.StringValue(secretPath)
		}
	}

	stateAction, diags := basetypes.NewListValue(types.StringType, planPermissionActions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Permission = projectIdentitySpecificPrivilegeResourceResourcePermissions{
		Actions: stateAction,
		Subject: planPermissionSubject,
		Conditions: projectIdentitySpecificPrivilegeResourceResourcePermissionCondition{
			Environment: planPermissionEnvironment,
			SecretPath:  planPermissionSecretPath,
		},
	}
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectIdentitySpecificPrivilegeResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project identity specific privilege",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectIdentitySpecificPrivilegeResourceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectIdentitySpecificPrivilegeResourceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectSlug != plan.ProjectSlug {
		resp.Diagnostics.AddError(
			"Unable to update project identity specific privilege",
			"Project slug cannot be updated",
		)
		return
	}

	planPermissionActions := make([]types.String, 0, len(plan.Permission.Actions.Elements()))
	diags = plan.Permission.Actions.ElementsAs(ctx, &planPermissionActions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	condition := make(map[string]any)
	environment := plan.Permission.Conditions.Environment.ValueString()
	secretPath := plan.Permission.Conditions.SecretPath.ValueString()
	condition["environment"] = environment
	if secretPath != "" {
		condition["secretPath"] = map[string]string{"$glob": secretPath}
	}

	actions := make([]string, 0, len(planPermissionActions))
	for _, action := range planPermissionActions {
		actions = append(actions, action.ValueString())
	}
	privilegePermission := infisicalclient.ProjectSpecificPrivilegePermissionRequest{
		Actions:    actions,
		Subject:    plan.Permission.Subject.ValueString(),
		Conditions: condition,
	}
	isTemporary := plan.IsTemporary.ValueBool()
	temporaryMode := plan.TemporaryMode.ValueString()
	temporaryRange := plan.TemporaryRange.ValueString()
	temporaryAccesStartTime := time.Now().UTC()

	if plan.TemporaryAccesStartTime.ValueString() != "" {
		var err error
		temporaryAccesStartTime, err = time.Parse(time.RFC3339, plan.TemporaryAccesStartTime.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing field TemporaryAccessStartTime",
				fmt.Sprintf("Must provider valid ISO timestamp for field temporaryAccesStartTime %s", plan.TemporaryAccesStartTime.ValueString()),
			)
			return
		}
	}

	if isTemporary && temporaryMode == "" {
		temporaryMode = TEMPORARY_MODE_RELATIVE
	}
	if isTemporary && temporaryRange == "" {
		temporaryRange = "1h"
	}

	updatedSpecificPrivilege, err := r.client.UpdateProjectIdentitySpecificPrivilege(infisical.UpdateProjectIdentitySpecificPrivilegeRequest{
		ProjectSlug:   plan.ProjectSlug.ValueString(),
		PrivilegeSlug: state.Slug.ValueString(),
		IdentityId:    plan.IdentityID.ValueString(),
		Details: infisical.UpdateProjectIdentitySpecificPrivilegeDataRequest{
			Slug:                     plan.Slug.ValueString(),
			Permissions:              privilegePermission,
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project identity specific privilege",
			"Couldn't update project identity specific privilege from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Slug = types.StringValue(updatedSpecificPrivilege.Privilege.Slug)
	if updatedSpecificPrivilege.Privilege.IsTemporary {
		plan.TemporaryAccessEndTime = types.StringValue(updatedSpecificPrivilege.Privilege.TemporaryAccessEndTime.Format(time.RFC3339))
		plan.TemporaryAccesStartTime = types.StringValue(updatedSpecificPrivilege.Privilege.TemporaryAccessStartTime.Format(time.RFC3339))
		plan.TemporaryRange = types.StringValue(updatedSpecificPrivilege.Privilege.TemporaryRange)
		plan.TemporaryMode = types.StringValue(updatedSpecificPrivilege.Privilege.TemporaryMode)
	} else {
		plan.TemporaryAccessEndTime = types.StringNull()
		plan.TemporaryAccesStartTime = types.StringNull()
		plan.TemporaryRange = types.StringNull()
		plan.TemporaryMode = types.StringNull()
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectIdentitySpecificPrivilegeResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project identity specific privilege",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectIdentitySpecificPrivilegeResourceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectIdentitySpecificPrivilege(infisical.DeleteProjectIdentitySpecificPrivilegeRequest{
		ProjectSlug:   state.ProjectSlug.ValueString(),
		IdentityId:    state.IdentityID.ValueString(),
		PrivilegeSlug: state.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project identity specific privilege",
			"Couldn't delete project identity specific privilege from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
