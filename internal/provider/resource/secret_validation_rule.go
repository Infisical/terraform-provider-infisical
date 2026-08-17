package resource

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	SECRET_VALIDATION_RULE_TYPE_STATIC_SECRETS   = "static-secrets"
	SECRET_VALIDATION_RULE_TYPE_DYNAMIC_SECRETS  = "dynamic-secrets"
	SECRET_VALIDATION_RULE_TYPE_SECRET_ROTATIONS = "secret-rotations"

	SECRET_VALIDATION_CONSTRAINT_TYPE_MIN_LENGTH          = "min-length"
	SECRET_VALIDATION_CONSTRAINT_TYPE_MAX_LENGTH          = "max-length"
	SECRET_VALIDATION_CONSTRAINT_TYPE_REGEX_PATTERN       = "regex-pattern"
	SECRET_VALIDATION_CONSTRAINT_TYPE_REQUIRED_PREFIX     = "required-prefix"
	SECRET_VALIDATION_CONSTRAINT_TYPE_REQUIRED_SUFFIX     = "required-suffix"
	SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE = "prevent-value-reuse"

	SECRET_VALIDATION_CONSTRAINT_TARGET_KEY      = "key"
	SECRET_VALIDATION_CONSTRAINT_TARGET_VALUE    = "value"
	SECRET_VALIDATION_CONSTRAINT_TARGET_PASSWORD = "password"

	// SECRET_VALIDATION_MAX_PREVENT_VALUE_REUSE_VERSIONS mirrors Infisical's current cap. It
	// drives a warning rather than an error so raising it server-side doesn't require a
	// provider release.
	SECRET_VALIDATION_MAX_PREVENT_VALUE_REUSE_VERSIONS = 25
)

var (
	SUPPORTED_SECRET_VALIDATION_RULE_TYPES = []string{
		SECRET_VALIDATION_RULE_TYPE_STATIC_SECRETS,
		SECRET_VALIDATION_RULE_TYPE_DYNAMIC_SECRETS,
		SECRET_VALIDATION_RULE_TYPE_SECRET_ROTATIONS,
	}

	SUPPORTED_SECRET_VALIDATION_CONSTRAINT_TYPES = []string{
		SECRET_VALIDATION_CONSTRAINT_TYPE_MIN_LENGTH,
		SECRET_VALIDATION_CONSTRAINT_TYPE_MAX_LENGTH,
		SECRET_VALIDATION_CONSTRAINT_TYPE_REGEX_PATTERN,
		SECRET_VALIDATION_CONSTRAINT_TYPE_REQUIRED_PREFIX,
		SECRET_VALIDATION_CONSTRAINT_TYPE_REQUIRED_SUFFIX,
		SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE,
	}

	SUPPORTED_SECRET_VALIDATION_CONSTRAINT_TARGETS = []string{
		SECRET_VALIDATION_CONSTRAINT_TARGET_KEY,
		SECRET_VALIDATION_CONSTRAINT_TARGET_VALUE,
		SECRET_VALIDATION_CONSTRAINT_TARGET_PASSWORD,
	}
)

var (
	_ resource.Resource                   = &secretValidationRuleResource{}
	_ resource.ResourceWithConfigure      = &secretValidationRuleResource{}
	_ resource.ResourceWithValidateConfig = &secretValidationRuleResource{}
	_ resource.ResourceWithImportState    = &secretValidationRuleResource{}
)

// NewSecretValidationRuleResource is a helper function to simplify the provider implementation.
func NewSecretValidationRuleResource() resource.Resource {
	return &secretValidationRuleResource{}
}

// secretValidationRuleResource is the resource implementation.
type secretValidationRuleResource struct {
	client *infisical.Client
}

type secretValidationRuleConstraintModel struct {
	Type      types.String `tfsdk:"type"`
	AppliesTo types.String `tfsdk:"applies_to"`
	Value     types.String `tfsdk:"value"`
}

type secretValidationRuleConfigModel struct {
	Type        types.String                          `tfsdk:"type"`
	Providers   types.Set                             `tfsdk:"providers"`
	Constraints []secretValidationRuleConstraintModel `tfsdk:"constraints"`
}

// secretValidationRuleResourceModel describes the resource data model.
type secretValidationRuleResourceModel struct {
	ID              types.String                     `tfsdk:"id"`
	ProjectID       types.String                     `tfsdk:"project_id"`
	Name            types.String                     `tfsdk:"name"`
	Description     types.String                     `tfsdk:"description"`
	EnvironmentSlug types.String                     `tfsdk:"environment_slug"`
	SecretPath      types.String                     `tfsdk:"secret_path"`
	IsActive        types.Bool                       `tfsdk:"is_active"`
	Rule            *secretValidationRuleConfigModel `tfsdk:"rule"`
}

// Metadata returns the resource type name.
func (r *secretValidationRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_validation_rule"
}

// Schema defines the schema for the resource.
func (r *secretValidationRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage secret validation rules in an Infisical project. Validation rules enforce constraints on static secret keys and values, and on the credentials generated by dynamic secrets and secret rotations. Infisical rejects rules whose scope and constraints overlap an existing rule of the same type, so use `depends_on` between rules that share an environment and secret path to keep applies deterministic. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the secret validation rule",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project the secret validation rule belongs to",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the secret validation rule",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
			},
			"description": schema.StringAttribute{
				Description: "An optional description of the secret validation rule. Infisical trims surrounding whitespace, so avoid whitespace-only values.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 500),
				},
			},
			"environment_slug": schema.StringAttribute{
				Description: "The slug of the environment to scope the rule to. When omitted, the rule applies to all environments in the project.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path the rule applies to. Supports glob patterns, e.g. \"/\", \"/prod\", \"/prod/**\".",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the rule is enforced. Defaults to true. Set to false to keep the rule but stop enforcing it.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"rule": schema.SingleNestedAttribute{
				Description: "The rule configuration. The type field selects which kind of secret the constraints apply to. Infisical replaces the rule configuration as a whole on update.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The kind of secret the rule applies to. Possible values: " + strings.Join(SUPPORTED_SECRET_VALIDATION_RULE_TYPES, ", "),
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(SUPPORTED_SECRET_VALIDATION_RULE_TYPES...),
						},
					},
					"providers": schema.SetAttribute{
						Description: "The dynamic secret or secret rotation providers the rule applies to, e.g. \"sql-database\" or \"postgres-credentials\". Required when type is `dynamic-secrets` or `secret-rotations`",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.Set{
							setvalidator.SizeAtLeast(1),
							setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
						},
					},
					"constraints": schema.SetNestedAttribute{
						Description: "The constraints enforced by this rule. At least one constraint is required. Order is not significant.",
						Required:    true,
						Validators: []validator.Set{
							setvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "The constraint type. Possible values: " + strings.Join(SUPPORTED_SECRET_VALIDATION_CONSTRAINT_TYPES, ", "),
									Required:    true,
									Validators: []validator.String{
										stringvalidator.OneOf(SUPPORTED_SECRET_VALIDATION_CONSTRAINT_TYPES...),
									},
								},
								"applies_to": schema.StringAttribute{
									Description: "What the constraint applies to. Possible values: " + strings.Join(SUPPORTED_SECRET_VALIDATION_CONSTRAINT_TARGETS, ", ") + ". Static secret rules support `key` and `value`, while dynamic secret and secret rotation rules support `password` only.",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.OneOf(SUPPORTED_SECRET_VALIDATION_CONSTRAINT_TARGETS...),
									},
								},
								"value": schema.StringAttribute{
									Description: "The constraint value, always expressed as a string. For `min-length` and `max-length` this is a number, for `regex-pattern` a regular expression, and for `prevent-value-reuse` the number of previous versions to check.",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.LengthAtLeast(1),
									},
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
func (r *secretValidationRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// ValidateConfig mirrors the cross-field rules Infisical enforces server-side so that
// invalid combinations surface at plan time instead of as an error during apply.
func (r *secretValidationRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var rule *secretValidationRuleConfigModel
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("rule"), &rule)...)
	if resp.Diagnostics.HasError() || rule == nil || rule.Type.IsNull() || rule.Type.IsUnknown() {
		return
	}

	rulePath := path.Root("rule")
	ruleType := rule.Type.ValueString()
	isGeneratedCredential := ruleType == SECRET_VALIDATION_RULE_TYPE_DYNAMIC_SECRETS ||
		ruleType == SECRET_VALIDATION_RULE_TYPE_SECRET_ROTATIONS

	// An unrecognized rule type is already reported by the schema validator, and the checks
	// below are all type-specific, so stop here rather than pile on misleading errors.
	if !isGeneratedCredential && ruleType != SECRET_VALIDATION_RULE_TYPE_STATIC_SECRETS {
		return
	}

	// Providers are required for generated-credential rules and rejected for static ones.
	// An empty set is rejected outright rather than dropped from the request, which would
	// otherwise come back as null and produce a permanent diff.
	if isGeneratedCredential {
		if rule.Providers.IsNull() {
			resp.Diagnostics.AddAttributeError(
				rulePath.AtName("providers"),
				"Missing providers",
				fmt.Sprintf("providers must contain at least one provider when the rule type is %q.", ruleType),
			)
		}
	} else if !rule.Providers.IsNull() {
		resp.Diagnostics.AddAttributeError(
			rulePath.AtName("providers"),
			"Unsupported providers",
			fmt.Sprintf("providers cannot be set when the rule type is %q.", SECRET_VALIDATION_RULE_TYPE_STATIC_SECRETS),
		)
	}

	// Constraints are a set, so there is no stable index to point a diagnostic at. Errors are
	// reported against the whole attribute and name the offending constraint instead.
	constraintsPath := rulePath.AtName("constraints")

	for _, constraint := range rule.Constraints {
		if constraint.Type.IsNull() || constraint.Type.IsUnknown() ||
			constraint.AppliesTo.IsNull() || constraint.AppliesTo.IsUnknown() {
			continue
		}

		constraintType := constraint.Type.ValueString()
		appliesTo := constraint.AppliesTo.ValueString()
		constraintDescription := fmt.Sprintf("The constraint with type %q and applies_to %q", constraintType, appliesTo)

		if isGeneratedCredential {
			if appliesTo != SECRET_VALIDATION_CONSTRAINT_TARGET_PASSWORD {
				resp.Diagnostics.AddAttributeError(
					constraintsPath,
					"Unsupported constraint target",
					fmt.Sprintf("%s is invalid: %q rules only support constraints that apply to %q.", constraintDescription, ruleType, SECRET_VALIDATION_CONSTRAINT_TARGET_PASSWORD),
				)
			}

			if constraintType == SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE {
				resp.Diagnostics.AddAttributeError(
					constraintsPath,
					"Unsupported constraint type",
					fmt.Sprintf("%s is invalid: the %q constraint is only supported for %q rules.", constraintDescription, SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE, SECRET_VALIDATION_RULE_TYPE_STATIC_SECRETS),
				)
			}
		} else if appliesTo == SECRET_VALIDATION_CONSTRAINT_TARGET_PASSWORD {
			resp.Diagnostics.AddAttributeError(
				constraintsPath,
				"Unsupported constraint target",
				fmt.Sprintf("%s is invalid: %q rules only support constraints that apply to %q or %q.", constraintDescription, SECRET_VALIDATION_RULE_TYPE_STATIC_SECRETS, SECRET_VALIDATION_CONSTRAINT_TARGET_KEY, SECRET_VALIDATION_CONSTRAINT_TARGET_VALUE),
			)
		} else if constraintType == SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE &&
			appliesTo != SECRET_VALIDATION_CONSTRAINT_TARGET_VALUE {
			resp.Diagnostics.AddAttributeError(
				constraintsPath,
				"Unsupported constraint target",
				fmt.Sprintf("%s is invalid: the %q constraint can only apply to %q.", constraintDescription, SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE, SECRET_VALIDATION_CONSTRAINT_TARGET_VALUE),
			)
		}

		if constraint.Value.IsNull() || constraint.Value.IsUnknown() {
			continue
		}
		value := strings.TrimSpace(constraint.Value.ValueString())

		// The remaining checks are about the format of the value, which Infisical stores as
		// a string but interprets as a number for these constraint types.
		switch constraintType {
		case SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE:
			versions, err := strconv.Atoi(value)
			if err != nil || versions < 1 {
				resp.Diagnostics.AddAttributeError(
					constraintsPath,
					"Invalid constraint value",
					fmt.Sprintf("The %q constraint value must be a positive integer.", SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE),
				)
			} else if versions > SECRET_VALIDATION_MAX_PREVENT_VALUE_REUSE_VERSIONS {
				// Infisical owns this cap and may raise it, so warn rather than hard-block a
				// value that a newer backend would accept.
				resp.Diagnostics.AddAttributeWarning(
					constraintsPath,
					"Constraint value may exceed the supported maximum",
					fmt.Sprintf("Infisical currently caps the %q constraint at %d previous versions, so %q is likely to be rejected when the rule is applied.", SECRET_VALIDATION_CONSTRAINT_TYPE_PREVENT_VALUE_REUSE, SECRET_VALIDATION_MAX_PREVENT_VALUE_REUSE_VERSIONS, constraint.Value.ValueString()),
				)
			}
		case SECRET_VALIDATION_CONSTRAINT_TYPE_MIN_LENGTH, SECRET_VALIDATION_CONSTRAINT_TYPE_MAX_LENGTH:
			// Infisical accepts any string here but interprets it as a number when the rule
			// is enforced, so a non-numeric value is a warning rather than an error.
			if length, err := strconv.Atoi(value); err != nil || length < 0 {
				resp.Diagnostics.AddAttributeWarning(
					constraintsPath,
					"Constraint value is not a number",
					fmt.Sprintf("The %q constraint value is interpreted as a number when the rule is enforced. %q is not a non-negative integer, which leads to unpredictable enforcement.", constraintType, constraint.Value.ValueString()),
				)
			}
		}
	}
}

// buildSecretValidationRuleConfig converts the planned rule block into the API payload.
func buildSecretValidationRuleConfig(ctx context.Context, plan *secretValidationRuleResourceModel, diagnostics *diag.Diagnostics) infisical.SecretValidationRuleConfig {
	config := infisical.SecretValidationRuleConfig{
		Type:        plan.Rule.Type.ValueString(),
		Constraints: make([]infisical.SecretValidationRuleConstraint, len(plan.Rule.Constraints)),
	}

	if !plan.Rule.Providers.IsNull() && !plan.Rule.Providers.IsUnknown() {
		diagnostics.Append(plan.Rule.Providers.ElementsAs(ctx, &config.Providers, false)...)
	}

	for i, constraint := range plan.Rule.Constraints {
		config.Constraints[i] = infisical.SecretValidationRuleConstraint{
			Type:      constraint.Type.ValueString(),
			AppliesTo: constraint.AppliesTo.ValueString(),
			Value:     constraint.Value.ValueString(),
		}
	}

	return config
}

// buildUpdateSecretValidationRuleRequest describes the full desired state of a rule. Every
// PATCH must go through this helper: description and environmentSlug are cleared when nil,
// so a partially populated body would silently wipe them.
func buildUpdateSecretValidationRuleRequest(plan *secretValidationRuleResourceModel, ruleID string, config infisical.SecretValidationRuleConfig) infisical.UpdateSecretValidationRuleRequest {
	isActive := plan.IsActive.ValueBool()

	return infisical.UpdateSecretValidationRuleRequest{
		ProjectID:       plan.ProjectID.ValueString(),
		RuleID:          ruleID,
		Name:            plan.Name.ValueString(),
		Description:     infisicaltf.OptionalStringPointer(plan.Description),
		EnvironmentSlug: infisicaltf.OptionalStringPointer(plan.EnvironmentSlug),
		SecretPath:      plan.SecretPath.ValueString(),
		Rule:            &config,
		IsActive:        &isActive,
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *secretValidationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan secretValidationRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config := buildSecretValidationRuleConfig(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.CreateSecretValidationRule(infisical.CreateSecretValidationRuleRequest{
		ProjectID:       plan.ProjectID.ValueString(),
		Name:            plan.Name.ValueString(),
		Description:     infisicaltf.OptionalStringPointer(plan.Description),
		EnvironmentSlug: infisicaltf.OptionalStringPointer(plan.EnvironmentSlug),
		SecretPath:      plan.SecretPath.ValueString(),
		Rule:            config,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret validation rule",
			"Couldn't create secret validation rule in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(rule.ID)

	// Infisical only accepts isActive on update, so a rule that should start out disabled
	// needs a follow-up request.
	if !plan.IsActive.ValueBool() {
		updateRequest := buildUpdateSecretValidationRuleRequest(&plan, rule.ID, config)

		if _, err := r.client.UpdateSecretValidationRule(updateRequest); err != nil {
			// Fall through to the state write below so the created rule isn't orphaned
			// outside of Terraform state.
			plan.IsActive = types.BoolValue(rule.IsActive)
			resp.Diagnostics.AddError(
				"Error disabling secret validation rule",
				"The secret validation rule was created but couldn't be disabled, unexpected error: "+err.Error(),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *secretValidationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state secretValidationRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetSecretValidationRuleById(infisical.GetSecretValidationRuleByIdRequest{
		ProjectID: state.ProjectID.ValueString(),
		RuleID:    state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading secret validation rule",
			"Couldn't read secret validation rule from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(rule.ID)
	state.ProjectID = types.StringValue(rule.ProjectID)
	state.Name = types.StringValue(infisicaltf.PreserveStringIfTrimmedEqual(rule.Name, state.Name))
	state.SecretPath = types.StringValue(infisicaltf.PreserveStringIfTrimmedEqual(rule.SecretPath, state.SecretPath))
	state.IsActive = types.BoolValue(rule.IsActive)

	// Infisical trims the description and stores an empty one as null, so both forms map
	// back to a null value to avoid a permanent diff.
	if rule.Description != nil && *rule.Description != "" {
		state.Description = types.StringValue(infisicaltf.PreserveStringIfTrimmedEqual(*rule.Description, state.Description))
	} else {
		state.Description = types.StringNull()
	}

	// Rules are created with an environment slug but read back with an environment ID.
	if rule.EnvID != nil && *rule.EnvID != "" {
		environment, err := r.client.GetProjectEnvironmentByID(infisical.GetProjectEnvironmentByIDRequest{ID: *rule.EnvID})
		if err != nil {
			if err != infisical.ErrNotFound {
				resp.Diagnostics.AddError(
					"Error reading secret validation rule environment",
					"Couldn't resolve the environment scoped to the secret validation rule, unexpected error: "+err.Error(),
				)
				return
			}

			// Falling back to null here would read as "applies to all environments", which
			// is a materially different rule, so keep whatever is already in state.
			resp.Diagnostics.AddWarning(
				"Secret validation rule environment not found",
				fmt.Sprintf("The environment (%s) scoped to secret validation rule %s no longer exists. Keeping the environment_slug currently in state.", *rule.EnvID, rule.ID),
			)
		} else {
			state.EnvironmentSlug = types.StringValue(environment.Environment.Slug)
		}
	} else {
		state.EnvironmentSlug = types.StringNull()
	}

	ruleConfig := &secretValidationRuleConfigModel{
		Type:        types.StringValue(rule.Type),
		Providers:   types.SetNull(types.StringType),
		Constraints: make([]secretValidationRuleConstraintModel, len(rule.Constraints)),
	}

	if len(rule.Providers) > 0 {
		providers, diags := types.SetValueFrom(ctx, types.StringType, rule.Providers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ruleConfig.Providers = providers
	}

	for i, constraint := range rule.Constraints {
		ruleConfig.Constraints[i] = secretValidationRuleConstraintModel{
			Type:      types.StringValue(constraint.Type),
			AppliesTo: types.StringValue(constraint.AppliesTo),
			Value:     types.StringValue(constraint.Value),
		}
	}

	state.Rule = ruleConfig

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *secretValidationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan secretValidationRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state secretValidationRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	config := buildSecretValidationRuleConfig(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := buildUpdateSecretValidationRuleRequest(&plan, state.ID.ValueString(), config)

	if _, err := r.client.UpdateSecretValidationRule(updateRequest); err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret validation rule",
			"Couldn't update secret validation rule in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *secretValidationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state secretValidationRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretValidationRule(infisical.DeleteSecretValidationRuleRequest{
		ProjectID: state.ProjectID.ValueString(),
		RuleID:    state.ID.ValueString(),
	})
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError(
			"Error deleting secret validation rule",
			"Couldn't delete secret validation rule from Infisical, unexpected error: "+err.Error(),
		)
	}
}

// ImportState imports an existing secret validation rule into Terraform state.
// The import ID format is: <project_id>,<rule_id>.
func (r *secretValidationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	parts := strings.SplitN(req.ID, ",", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format <project_id>,<rule_id>",
		)
		return
	}

	projectID := strings.TrimSpace(parts[0])
	ruleID := strings.TrimSpace(parts[1])

	rule, err := r.client.GetSecretValidationRuleById(infisical.GetSecretValidationRuleByIdRequest{
		ProjectID: projectID,
		RuleID:    ruleID,
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Secret validation rule not found",
				fmt.Sprintf("No secret validation rule with ID %q was found in project %q", ruleID, projectID),
			)
		} else {
			resp.Diagnostics.AddError(
				"Error fetching secret validation rule",
				"Couldn't fetch secret validation rule from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	// Read runs immediately after import and populates the remaining attributes.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), rule.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), rule.ProjectID)...)
}
