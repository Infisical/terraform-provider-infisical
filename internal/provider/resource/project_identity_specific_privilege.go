package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

type IdentityPermissionV2Entry struct {
	Action     types.Set    `tfsdk:"action"`
	Subject    types.String `tfsdk:"subject"`
	Inverted   types.Bool   `tfsdk:"inverted"`
	Conditions types.String `tfsdk:"conditions"`
}

// projectIdentitySpecificPrivilegeResourceResourceSourceModel describes the data source data model.
type projectIdentitySpecificPrivilegeResourceResourceModel struct {
	Slug                    types.String                                                 `tfsdk:"slug"`
	ProjectSlug             types.String                                                 `tfsdk:"project_slug"`
	IdentityID              types.String                                                 `tfsdk:"identity_id"`
	ID                      types.String                                                 `tfsdk:"id"`
	Permission              *projectIdentitySpecificPrivilegeResourceResourcePermissions `tfsdk:"permission"`
	PermissionsV2           []IdentityPermissionV2Entry                                  `tfsdk:"permissions_v2"`
	IsTemporary             types.Bool                                                   `tfsdk:"is_temporary"`
	TemporaryMode           types.String                                                 `tfsdk:"temporary_mode"`
	TemporaryRange          types.String                                                 `tfsdk:"temporary_range"`
	TemporaryAccesStartTime types.String                                                 `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime  types.String                                                 `tfsdk:"temporary_access_end_time"`
}

type projectIdentitySpecificPrivilegeResourceResourcePermissions struct {
	Actions    types.List                                                           `tfsdk:"actions"`
	Subject    types.String                                                         `tfsdk:"subject"`
	Conditions *projectIdentitySpecificPrivilegeResourceResourcePermissionCondition `tfsdk:"conditions"`
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
				Description:   "The slug for the new privilege",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					infisicaltf.SlugRegexValidator,
				},
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
				Description:   "Type of temporary access given. Types: relative. Default: relative",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"temporary_range": schema.StringAttribute{
				Description:   "TTL for the temporary time. Eg: 1m, 1h, 1d. Default: 1h",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"temporary_access_start_time": schema.StringAttribute{
				Description:   "ISO time for which temporary access should begin. The current time is used by default.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"temporary_access_end_time": schema.StringAttribute{
				Description:   "ISO time for which temporary access will end. Computed based on temporary_range and temporary_access_start_time",
				Computed:      true,
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"permission": schema.SingleNestedAttribute{
				Optional:           true,
				Description:        "(DEPRECATED, USE permissions_v2) The permissions assigned to the project identity specific privilege",
				DeprecationMessage: "Use permissions_v2 instead as it allows you to be more granular with access control",
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
			"permissions_v2": schema.SetNestedAttribute{
				Optional:    true,
				Description: "The permissions assigned to the project identity specific privilege. Refer to the documentation here https://infisical.com/docs/internals/permissions for its usage.",
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
							Required:    true,
						},
						"conditions": schema.StringAttribute{
							Optional:    true,
							Description: "When specified, only matching conditions will be allowed to access given resource.",
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

	if (plan.Permission != nil && plan.PermissionsV2 != nil) || (plan.Permission == nil && plan.PermissionsV2 == nil) {
		resp.Diagnostics.AddError(
			"Error creating project identity specific privilege",
			"Define either the permission or permissions_v2 property but not both.",
		)
		return
	}

	if plan.Permission != nil {
		// Permission V1
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
			plan.TemporaryAccessEndTime = types.StringValue("")
			plan.TemporaryAccesStartTime = types.StringValue("")
			plan.TemporaryRange = types.StringValue("")
			plan.TemporaryMode = types.StringValue("")
		}
	} else {
		// Permission V2
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
						"Error creating project identity specific privilege",
						"Error parsing conditions property: "+err.Error(),
					)
					return
				}

				permMap["conditions"] = conditionsMap
			}

			permissions[i] = permMap
		}

		isTemporary := plan.IsTemporary.ValueBool()
		temporaryMode := plan.TemporaryMode.ValueString()
		temporaryRange := plan.TemporaryRange.ValueString()
		temporaryAccesStartTime := time.Now().UTC()

		if isTemporary && plan.TemporaryAccesStartTime.ValueString() != "" {
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

		newProjectRole, err := r.client.CreateProjectIdentitySpecificPrivilegeV2(infisical.CreateProjectIdentitySpecificPrivilegeV2Request{
			ProjectId:   project.ID,
			Slug:        plan.Slug.ValueString(),
			IdentityId:  plan.IdentityID.ValueString(),
			Permissions: permissions,
			Type: infisical.CreateProjectIdentitySpecificPrivilegeV2Type{
				IsTemporary:              isTemporary,
				TemporaryMode:            temporaryMode,
				TemporaryRange:           temporaryRange,
				TemporaryAccessStartTime: temporaryAccesStartTime,
			},
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

		if isTemporary {
			plan.TemporaryAccessEndTime = types.StringValue(newProjectRole.Privilege.TemporaryAccessEndTime.Format(time.RFC3339))
			plan.TemporaryAccesStartTime = types.StringValue(newProjectRole.Privilege.TemporaryAccessStartTime.Format(time.RFC3339))
			plan.TemporaryRange = types.StringValue(newProjectRole.Privilege.TemporaryRange)
			plan.TemporaryMode = types.StringValue(newProjectRole.Privilege.TemporaryMode)
		} else {
			plan.TemporaryAccessEndTime = types.StringValue("")
			plan.TemporaryAccesStartTime = types.StringValue("")
			plan.TemporaryRange = types.StringValue("")
			plan.TemporaryMode = types.StringValue("")
		}
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

	if state.Permission != nil {
		// Permission V1
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
			state.TemporaryAccessEndTime = types.StringValue("")
			state.TemporaryAccesStartTime = types.StringValue("")
			state.TemporaryRange = types.StringValue("")
			state.TemporaryMode = types.StringValue("")
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

		state.Permission = &projectIdentitySpecificPrivilegeResourceResourcePermissions{
			Actions: stateAction,
			Subject: planPermissionSubject,
			Conditions: &projectIdentitySpecificPrivilegeResourceResourcePermissionCondition{
				Environment: planPermissionEnvironment,
				SecretPath:  planPermissionSecretPath,
			},
		}
	} else {
		// Permission V2
		projectIdentitySpecificPrivilegeResource, err := r.client.GetProjectIdentitySpecificPrivilegeV2(infisical.GetProjectIdentitySpecificPrivilegeV2Request{
			ID: state.ID.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading project identity specific privilege",
				"Couldn't read project identity specific privilege from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		if projectIdentitySpecificPrivilegeResource.Privilege.IsTemporary {
			state.TemporaryAccessEndTime = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.TemporaryAccessEndTime.Format(time.RFC3339))
			state.TemporaryAccesStartTime = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.TemporaryAccessStartTime.Format(time.RFC3339))
			state.TemporaryRange = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.TemporaryRange)
			state.TemporaryMode = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.TemporaryMode)
		} else {
			state.TemporaryAccessEndTime = types.StringValue("")
			state.TemporaryAccesStartTime = types.StringValue("")
			state.TemporaryRange = types.StringValue("")
			state.TemporaryMode = types.StringValue("")
		}

		state.Slug = types.StringValue(projectIdentitySpecificPrivilegeResource.Privilege.Slug)

		permissions := make([]IdentityPermissionV2Entry, len(projectIdentitySpecificPrivilegeResource.Privilege.Permissions))
		for i, permMap := range projectIdentitySpecificPrivilegeResource.Privilege.Permissions {
			entry := IdentityPermissionV2Entry{}

			if actionRaw, ok := permMap["action"].([]interface{}); ok {
				actions := make([]string, len(actionRaw))
				for i, v := range actionRaw {
					if strValue, ok := v.(string); ok {
						actions[i] = strValue
					} else {
						resp.Diagnostics.AddError(
							"Invalid Action Type",
							fmt.Sprintf("Expected string type for action at index %d, got %T", i, v),
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
			}

			if conditions, ok := permMap["conditions"].(map[string]any); ok {
				conditionsBytes, err := json.Marshal(conditions)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error reading identity specific privilege",
						"Couldn't parse conditions property, unexpected error: "+err.Error(),
					)
					return
				}

				entry.Conditions = types.StringValue(string(conditionsBytes))
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

	if (plan.Permission != nil && plan.PermissionsV2 != nil) || (plan.Permission == nil && plan.PermissionsV2 == nil) {
		resp.Diagnostics.AddError(
			"Error updating project identity specific privilege",
			"Define either the permission or permissions_v2 property but not both.",
		)
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

	if plan.Permission != nil {
		// Permission V1
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
			plan.TemporaryAccessEndTime = types.StringValue("")
			plan.TemporaryAccesStartTime = types.StringValue("")
			plan.TemporaryRange = types.StringValue("")
			plan.TemporaryMode = types.StringValue("")
		}
	} else {
		// Permission V2
		permissions := make([]map[string]any, len(plan.PermissionsV2))
		for i, perm := range plan.PermissionsV2 {
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
						"Error updating project role",
						"Error parsing conditions property: "+err.Error(),
					)
					return
				}

				permMap["conditions"] = conditionsMap
			}

			permissions[i] = permMap
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

		updatedSpecificPrivilege, err := r.client.UpdateProjectIdentitySpecificPrivilegeV2(infisical.UpdateProjectIdentitySpecificPrivilegeV2Request{
			ID:          plan.ID.ValueString(),
			Slug:        plan.Slug.ValueString(),
			Permissions: permissions,
			Type: infisical.UpdateProjectIdentitySpecificPrivilegeV2Type{
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
			plan.TemporaryAccessEndTime = types.StringValue("")
			plan.TemporaryAccesStartTime = types.StringValue("")
			plan.TemporaryRange = types.StringValue("")
			plan.TemporaryMode = types.StringValue("")
		}
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
