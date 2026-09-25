package datasource

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SecretValidationRulesDataSource{}

func NewSecretValidationRulesDataSource() datasource.DataSource {
	return &SecretValidationRulesDataSource{}
}

// SecretValidationRulesDataSource reads the secret validation rules of a project.
type SecretValidationRulesDataSource struct {
	client *infisical.Client
}

// SecretValidationRulesDataSourceModel describes the data source data model.
type SecretValidationRulesDataSourceModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	Type      types.String `tfsdk:"type"`
	Rules     types.List   `tfsdk:"rules"`
}

var secretValidationRuleEnvironmentAttrTypes = map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
	"slug": types.StringType,
}

var secretValidationRuleStringConstraintsAttrTypes = map[string]attr.Type{
	"min_length":      types.Int64Type,
	"max_length":      types.Int64Type,
	"regex_pattern":   types.StringType,
	"required_prefix": types.StringType,
	"required_suffix": types.StringType,
}

var secretValidationRuleValueConstraintsAttrTypes = func() map[string]attr.Type {
	result := make(map[string]attr.Type, len(secretValidationRuleStringConstraintsAttrTypes)+2)
	maps.Copy(result, secretValidationRuleStringConstraintsAttrTypes)
	result["previous_versions"] = types.Int64Type
	result["unique_within_scope"] = types.BoolType

	return result
}()

// secretValidationRuleConstraintsAttrTypes is the union of every rule type's constraints, because a
// list attribute has one element type. The keys that do not apply to a given rule's type come back
// null.
var secretValidationRuleConstraintsAttrTypes = map[string]attr.Type{
	"key_constraints":      types.ObjectType{AttrTypes: secretValidationRuleStringConstraintsAttrTypes},
	"value_constraints":    types.ObjectType{AttrTypes: secretValidationRuleValueConstraintsAttrTypes},
	"providers":            types.SetType{ElemType: types.StringType},
	"password_constraints": types.ObjectType{AttrTypes: secretValidationRuleStringConstraintsAttrTypes},
}

var secretValidationRuleAttrTypes = map[string]attr.Type{
	"id":          types.StringType,
	"name":        types.StringType,
	"description": types.StringType,
	"project_id":  types.StringType,
	"secret_path": types.StringType,
	"is_active":   types.BoolType,
	"created_at":  types.StringType,
	"updated_at":  types.StringType,
	"type":        types.StringType,
	"environment": types.ObjectType{AttrTypes: secretValidationRuleEnvironmentAttrTypes},
	"constraints": types.ObjectType{AttrTypes: secretValidationRuleConstraintsAttrTypes},
}

func (d *SecretValidationRulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_validation_rules"
}

func stringConstraintsDataSourceAttributes(subject string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"min_length": schema.Int64Attribute{
			Computed:    true,
			Description: fmt.Sprintf("The minimum number of characters the %s must contain.", subject),
		},
		"max_length": schema.Int64Attribute{
			Computed:    true,
			Description: fmt.Sprintf("The maximum number of characters the %s may contain.", subject),
		},
		"regex_pattern": schema.StringAttribute{
			Computed:    true,
			Description: fmt.Sprintf("A regular expression the %s must match.", subject),
		},
		"required_prefix": schema.StringAttribute{
			Computed:    true,
			Description: fmt.Sprintf("A string the %s must start with.", subject),
		},
		"required_suffix": schema.StringAttribute{
			Computed:    true,
			Description: fmt.Sprintf("A string the %s must end with.", subject),
		},
	}
}

func (d *SecretValidationRulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	valueConstraintsAttributes := stringConstraintsDataSourceAttributes("secret value")
	valueConstraintsAttributes["previous_versions"] = schema.Int64Attribute{
		Computed:    true,
		Description: "How many of the secret's own previous versions the new value must differ from. Null when the rule allows a value that repeats a previous version.",
	}
	valueConstraintsAttributes["unique_within_scope"] = schema.BoolAttribute{
		Computed:    true,
		Description: "Whether the rule rejects a value that another secret in the rule's scope already holds. Null when the rule allows a value another secret already holds.",
	}

	resp.Schema = schema.Schema{
		Description: "Fetch the secret validation rules of an Infisical project. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Infisical project to read secret validation rules from.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Return only the rules of this type. Omit to return every rule in the project. Supported: static-secrets, dynamic-secrets, secret-rotations.",
				Validators: []validator.String{stringvalidator.OneOf(
					string(infisical.SecretValidationRuleTypeStaticSecrets),
					string(infisical.SecretValidationRuleTypeDynamicSecrets),
					string(infisical.SecretValidationRuleTypeSecretRotations),
				)},
			},
			"rules": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The secret validation rules of the project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the secret validation rule.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the secret validation rule.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "The description of the secret validation rule.",
						},
						"project_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the project the rule belongs to.",
						},
						"secret_path": schema.StringAttribute{
							Computed:    true,
							Description: "The secret path the rule is scoped to.",
						},
						"is_active": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the rule is enforced.",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "When the rule was created.",
						},
						"updated_at": schema.StringAttribute{
							Computed:    true,
							Description: "When the rule was last updated.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of the rule: static-secrets, dynamic-secrets or secret-rotations.",
						},
						"environment": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "The environment the rule is scoped to. Null when the rule is enforced in every environment of the project.",
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:    true,
									Description: "The ID of the environment.",
								},
								"name": schema.StringAttribute{
									Computed:    true,
									Description: "The name of the environment.",
								},
								"slug": schema.StringAttribute{
									Computed:    true,
									Description: "The slug of the environment.",
								},
							},
						},
						"constraints": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "The constraints the rule enforces. Only the attributes that apply to the rule's type are set; the rest are null.",
							Attributes: map[string]schema.Attribute{
								"key_constraints": schema.SingleNestedAttribute{
									Computed:    true,
									Description: "Constraints enforced on the secret key. Set only on static-secrets rules.",
									Attributes:  stringConstraintsDataSourceAttributes("secret key"),
								},
								"value_constraints": schema.SingleNestedAttribute{
									Computed:    true,
									Description: "Constraints enforced on the secret value. Set only on static-secrets rules.",
									Attributes:  valueConstraintsAttributes,
								},
								"providers": schema.SetAttribute{
									Computed:    true,
									ElementType: types.StringType,
									Description: "The providers the rule applies to. Set only on dynamic-secrets and secret-rotations rules.",
								},
								"password_constraints": schema.SingleNestedAttribute{
									Computed:    true,
									Description: "Constraints the generated password must satisfy. Set only on dynamic-secrets and secret-rotations rules.",
									Attributes:  stringConstraintsDataSourceAttributes("generated password"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *SecretValidationRulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

	d.client = client
}

func (d *SecretValidationRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch secret validation rules",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data SecretValidationRulesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var rules []infisical.SecretValidationRule
	var err error

	if data.Type.IsNull() || data.Type.IsUnknown() {
		rules, err = d.client.ListAllSecretValidationRules(infisical.ListAllSecretValidationRulesRequest{
			ProjectID: data.ProjectID.ValueString(),
		})
	} else {
		rules, err = d.client.ListSecretValidationRules(infisical.ListSecretValidationRulesRequest{
			Type:      infisical.SecretValidationRuleType(data.Type.ValueString()),
			ProjectID: data.ProjectID.ValueString(),
		})
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the secret validation rules",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	// Sort by ID so the list is stable when the API returns rules in a different order.
	slices.SortFunc(rules, func(a, b infisical.SecretValidationRule) int {
		return cmp.Compare(a.ID, b.ID)
	})

	elements := make([]attr.Value, 0, len(rules))
	for _, rule := range rules {
		element, diags := secretValidationRuleObject(ctx, rule)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		elements = append(elements, element)
	}

	ruleList, diags := types.ListValue(types.ObjectType{AttrTypes: secretValidationRuleAttrTypes}, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Rules = ruleList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func secretValidationRuleObject(ctx context.Context, rule infisical.SecretValidationRule) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	environment := types.ObjectNull(secretValidationRuleEnvironmentAttrTypes)
	if rule.Environment != nil {
		value, environmentDiags := types.ObjectValue(secretValidationRuleEnvironmentAttrTypes, map[string]attr.Value{
			"id":   types.StringValue(rule.Environment.ID),
			"name": types.StringValue(rule.Environment.Name),
			"slug": types.StringValue(rule.Environment.Slug),
		})
		diags.Append(environmentDiags...)
		if diags.HasError() {
			return types.ObjectNull(secretValidationRuleAttrTypes), diags
		}

		environment = value
	}

	constraints, constraintDiags := secretValidationRuleConstraintsObject(ctx, rule)
	diags.Append(constraintDiags...)
	if diags.HasError() {
		return types.ObjectNull(secretValidationRuleAttrTypes), diags
	}

	// isActive sits outside the API response's required set, so an absent value means the default.
	isActive := true
	if rule.IsActive != nil {
		isActive = *rule.IsActive
	}

	object, objectDiags := types.ObjectValue(secretValidationRuleAttrTypes, map[string]attr.Value{
		"id":          types.StringValue(rule.ID),
		"name":        types.StringValue(rule.Name),
		"description": secretValidationRuleDescriptionValue(rule.Description),
		"project_id":  types.StringValue(rule.ProjectID),
		"secret_path": types.StringValue(rule.SecretPath),
		"is_active":   types.BoolValue(isActive),
		"created_at":  types.StringValue(rule.CreatedAt),
		"updated_at":  types.StringValue(rule.UpdatedAt),
		"type":        types.StringValue(string(rule.Type)),
		"environment": environment,
		"constraints": constraints,
	})
	diags.Append(objectDiags...)

	return object, diags
}

func secretValidationRuleConstraintsObject(ctx context.Context, rule infisical.SecretValidationRule) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	key, keyDiags := secretValidationRuleStringConstraintsObject(rule.KeyConstraints)
	diags.Append(keyDiags...)

	value := types.ObjectNull(secretValidationRuleValueConstraintsAttrTypes)
	if rule.ValueConstraints != nil {
		values := secretValidationRuleStringConstraintsValues(&rule.ValueConstraints.SecretValidationRuleStringConstraints)
		values["previous_versions"] = secretValidationRuleInt64Value(rule.ValueConstraints.UniqueAcrossLastVersions)
		values["unique_within_scope"] = secretValidationRuleFlagValue(rule.ValueConstraints.UniqueWithinScope)

		valueObject, valueDiags := types.ObjectValue(secretValidationRuleValueConstraintsAttrTypes, values)
		diags.Append(valueDiags...)
		value = valueObject
	}

	providers := types.SetNull(types.StringType)
	if rule.Providers != nil {
		providerSet, providerDiags := types.SetValueFrom(ctx, types.StringType, rule.Providers)
		diags.Append(providerDiags...)
		providers = providerSet
	}

	password, passwordDiags := secretValidationRuleStringConstraintsObject(rule.PasswordConstraints)
	diags.Append(passwordDiags...)

	if diags.HasError() {
		return types.ObjectNull(secretValidationRuleConstraintsAttrTypes), diags
	}

	object, objectDiags := types.ObjectValue(secretValidationRuleConstraintsAttrTypes, map[string]attr.Value{
		"key_constraints":      key,
		"value_constraints":    value,
		"providers":            providers,
		"password_constraints": password,
	})
	diags.Append(objectDiags...)

	return object, diags
}

func secretValidationRuleStringConstraintsObject(constraints *infisical.SecretValidationRuleStringConstraints) (types.Object, diag.Diagnostics) {
	if constraints == nil {
		return types.ObjectNull(secretValidationRuleStringConstraintsAttrTypes), nil
	}

	return types.ObjectValue(secretValidationRuleStringConstraintsAttrTypes, secretValidationRuleStringConstraintsValues(constraints))
}

func secretValidationRuleStringConstraintsValues(constraints *infisical.SecretValidationRuleStringConstraints) map[string]attr.Value {
	return map[string]attr.Value{
		"min_length":      secretValidationRuleInt64Value(constraints.MinLength),
		"max_length":      secretValidationRuleInt64Value(constraints.MaxLength),
		"regex_pattern":   secretValidationRuleStringValue(constraints.RegexPattern),
		"required_prefix": secretValidationRuleStringValue(constraints.RequiredPrefix),
		"required_suffix": secretValidationRuleStringValue(constraints.RequiredSuffix),
	}
}

func secretValidationRuleInt64Value(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*value)
}

func secretValidationRuleFlagValue(value *bool) types.Bool {
	if value == nil || !*value {
		return types.BoolNull()
	}

	return types.BoolValue(true)
}

func secretValidationRuleDescriptionValue(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}

	return types.StringValue(*value)
}

func secretValidationRuleStringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}

	return types.StringValue(*value)
}
