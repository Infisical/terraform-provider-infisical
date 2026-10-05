package resource

import (
	"context"
	"fmt"
	"regexp"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/google/uuid"
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

// secretPathPattern is the shape the API documents for a rule's secret path: an absolute path,
// optionally with glob segments. Checking it here turns a 400 at apply into a plan-time error.
var secretPathPattern = regexp.MustCompile(`^/`)

// secretValidationRuleResourceTypeNames maps each rule type to the resource that manages it. The
// API's own slugs are kebab-case, so printing one in an error would name a resource type that does
// not exist.
var secretValidationRuleResourceTypeNames = map[infisical.SecretValidationRuleType]string{
	infisical.SecretValidationRuleTypeStaticSecrets:   "infisical_secret_validation_rule_static_secrets",
	infisical.SecretValidationRuleTypeDynamicSecrets:  "infisical_secret_validation_rule_dynamic_secrets",
	infisical.SecretValidationRuleTypeSecretRotations: "infisical_secret_validation_rule_secret_rotations",
}

// SecretValidationRuleBaseResource is the shared implementation behind every secret validation rule
// type. Each type supplies its own constraints schema and the two closures that translate between
// the Terraform model and the API's payload.
type SecretValidationRuleBaseResource struct {
	Type             infisical.SecretValidationRuleType // rule-type segment of the API path, e.g. "static-secrets"
	ResourceTypeName string                             // appended to the provider name, e.g. "_secret_validation_rule_static_secrets"
	RuleName         string                             // rule type as it appears in docs and errors, e.g. "Static Secrets"
	client           *infisical.Client

	ConstraintsAttributes   map[string]schema.Attribute
	ReadConstraintsFromPlan func(ctx context.Context, plan SecretValidationRuleBaseResourceModel, isCreate bool) (map[string]any, diag.Diagnostics)
	ReadConstraintsFromApi  func(ctx context.Context, rule infisical.SecretValidationRule) (types.Object, diag.Diagnostics)
}

type SecretValidationRuleBaseResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ProjectID   types.String `tfsdk:"project_id"`
	Environment types.String `tfsdk:"environment"`
	SecretPath  types.String `tfsdk:"secret_path"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	Constraints types.Object `tfsdk:"constraints"`
}

// Metadata returns the resource type name.
func (r *SecretValidationRuleBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

// ImportState imports an existing secret validation rule by its ID. Read then populates every
// attribute, including the project the rule belongs to.
func (r *SecretValidationRuleBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.Parse(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected the %s secret validation rule ID to be a valid UUID, got: %s", r.RuleName, req.ID),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SecretValidationRuleBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage Secret Validation Rules for %s", r.RuleName),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the secret validation rule.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret validation rule.",
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 64)},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "An optional description of the secret validation rule.",
				// Not LengthAtMost: the API stores an empty description as no description, so an
				// empty string would read back as null and diff on every plan.
				Validators: []validator.String{stringvalidator.LengthBetween(1, 500)},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Infisical project to create the secret validation rule in.",
				Validators:    []validator.String{stringvalidator.LengthBetween(1, 36)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment": schema.StringAttribute{
				Optional:    true,
				Description: "The slug of the environment to scope this rule to. Omit to enforce the rule in every environment of the project.",
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 64)},
			},
			"secret_path": schema.StringAttribute{
				Required:    true,
				Description: "The secret path to scope this rule to. Supports glob patterns such as `/apps/**`.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1024),
					stringvalidator.RegexMatches(secretPathPattern, "must start with a forward slash, for example `/` or `/apps/**`"),
				},
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the rule is enforced. An inactive rule is kept but ignored. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"constraints": schema.SingleNestedAttribute{
				Required:    true,
				Description: fmt.Sprintf("The constraints this %s rule enforces.", r.RuleName),
				Attributes:  r.ConstraintsAttributes,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *SecretValidationRuleBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *SecretValidationRuleBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan SecretValidationRuleBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	constraints, diags := r.ReadConstraintsFromPlan(ctx, plan, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.CreateSecretValidationRule(infisical.CreateSecretValidationRuleRequest{
		Type:        r.Type,
		Name:        plan.Name.ValueString(),
		ProjectID:   plan.ProjectID.ValueString(),
		Description: stringPointer(plan.Description),
		Environment: stringPointer(plan.Environment),
		SecretPath:  plan.SecretPath.ValueString(),
		IsActive:    plan.IsActive.ValueBool(),
		Constraints: constraints,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret validation rule",
			"Couldn't create secret validation rule, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(rule.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *SecretValidationRuleBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state SecretValidationRuleBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetSecretValidationRuleById(infisical.GetSecretValidationRuleByIdRequest{
		Type: r.Type,
		ID:   state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading secret validation rule",
			"Couldn't read secret validation rule, unexpected error: "+err.Error(),
		)
		return
	}

	// The read is type-scoped, so a mismatch should 404 first. If the API ever routes on the ID
	// alone, this stops an import from binding a rule of one type to the resource for another and
	// then updating it through the wrong endpoint.
	if rule.Type != r.Type {
		resp.Diagnostics.AddError(
			"Secret validation rule type mismatch",
			fmt.Sprintf(
				"Secret validation rule %s is a %s rule, which is managed by %s rather than by this resource.",
				rule.ID, rule.Type, secretValidationRuleResourceTypeNames[rule.Type],
			),
		)
		return
	}

	state.Name = types.StringValue(rule.Name)
	state.ProjectID = types.StringValue(rule.ProjectID)
	state.SecretPath = types.StringValue(rule.SecretPath)

	// The pointer already separates a JSON null from an empty string, but this API family has a
	// habit of handing back "" for a description that was never set.
	if rule.Description != nil && *rule.Description != "" {
		state.Description = types.StringValue(*rule.Description)
	} else if !state.Description.IsNull() {
		state.Description = types.StringNull()
	}

	if rule.Environment != nil {
		state.Environment = types.StringValue(rule.Environment.Slug)
	} else {
		state.Environment = types.StringNull()
	}

	// isActive sits outside the response's required set, so an absent value means the default.
	if rule.IsActive != nil {
		state.IsActive = types.BoolValue(*rule.IsActive)
	} else {
		state.IsActive = types.BoolValue(true)
	}

	state.Constraints, diags = r.ReadConstraintsFromApi(ctx, rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *SecretValidationRuleBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan SecretValidationRuleBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SecretValidationRuleBaseResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	constraints, diags := r.ReadConstraintsFromPlan(ctx, plan, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateSecretValidationRule(infisical.UpdateSecretValidationRuleRequest{
		Type:        r.Type,
		ID:          state.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: stringPointer(plan.Description),
		Environment: stringPointer(plan.Environment),
		SecretPath:  plan.SecretPath.ValueString(),
		IsActive:    plan.IsActive.ValueBool(),
		Constraints: constraints,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret validation rule",
			"Couldn't update secret validation rule, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *SecretValidationRuleBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret validation rule",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state SecretValidationRuleBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretValidationRule(infisical.DeleteSecretValidationRuleRequest{
		Type: r.Type,
		ID:   state.ID.ValueString(),
	})

	// A rule that is already gone is the outcome the delete wanted.
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError(
			"Error deleting secret validation rule",
			"Couldn't delete secret validation rule from Infisical, unexpected error: "+err.Error(),
		)
	}
}
