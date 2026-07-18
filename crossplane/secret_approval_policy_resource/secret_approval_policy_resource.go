package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSecretApprovalPolicyResource() resource.Resource {
	return &secretApprovalPolicyResource{}
}

type secretApprovalPolicyResource struct {
	client *infisical.Client
}

type SecretApprover struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"username"`
}

type SecretBypasser struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"username"`
}

type secretApprovalPolicyResourceModel struct {
	ID                types.String     `tfsdk:"id"`
	ProjectID         types.String     `tfsdk:"project_id"`
	Name              types.String     `tfsdk:"name"`
	EnvironmentSlugs  types.List       `tfsdk:"environment_slugs"`
	SecretPath        types.String     `tfsdk:"secret_path"`
	Approvers         []SecretApprover `tfsdk:"approvers"`
	Bypassers         []SecretBypasser `tfsdk:"bypassers"`
	RequiredApprovals types.Int64      `tfsdk:"required_approvals"`
	EnforcementLevel  types.String     `tfsdk:"enforcement_level"`
	AllowSelfApproval types.Bool       `tfsdk:"allow_self_approval"`
}

func (r *secretApprovalPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_approval_policy"
}

func (r *secretApprovalPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create secret approval policy for your projects",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the secret approval policy",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project to add the secret approval policy",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description:   "The name of the secret approval policy",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_slugs": schema.ListAttribute{
				Description:   "The environments to apply the secret approval policy to",
				Required:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{pkg.UnorderedList()},
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path to apply the secret approval policy to",
				Required:    true,
			},
			"approvers": schema.SetNestedAttribute{
				Required:    true,
				Description: "The required approvers",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The type of approver. Either group or user",
							Required:    true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the approver",
							Optional:    true,
						},
						"username": schema.StringAttribute{
							Description: "The username of the approver. By default, this is the email",
							Optional:    true,
						},
					},
				},
			},
			"bypassers": schema.SetNestedAttribute{
				Optional:    true,
				Description: "The bypassers who can bypass the approval policy",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The type of bypasser. Either group or user",
							Required:    true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the bypasser",
							Optional:    true,
						},
						"username": schema.StringAttribute{
							Description: "The username of the bypasser. By default, this is the email",
							Optional:    true,
						},
					},
				},
			},
			"required_approvals": schema.Int64Attribute{
				Description: "The number of required approvers",
				Required:    true,
			},
			"enforcement_level": schema.StringAttribute{
				Description: "The enforcement level of the policy. This can either be hard or soft",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("hard"),
			},
			"allow_self_approval": schema.BoolAttribute{
				Description: "Whether to allow the approvers to approve their own changes",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *secretApprovalPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *secretApprovalPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secret approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan secretApprovalPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	approvers, ok := validateAndMapApprovers(plan.Approvers, &resp.Diagnostics)
	if !ok {
		return
	}

	bypasserInputs := make([]infisicaltf.BypasserInput, len(plan.Bypassers))
	for i, el := range plan.Bypassers {
		bypasserInputs[i] = infisicaltf.BypasserInput{Type: el.Type, ID: el.ID, Name: el.Name}
	}
	validatedBypassers, bypassersOk := infisicaltf.ValidateAndMapBypassers(bypasserInputs, &resp.Diagnostics)
	if !bypassersOk {
		return
	}
	var bypassers []infisical.CreateSecretApprovalPolicyBypasser
	for _, b := range validatedBypassers {
		bypassers = append(bypassers, infisical.CreateSecretApprovalPolicyBypasser{
			ID: b.ID, Name: b.Name, Type: b.Type,
		})
	}

	environments := make([]string, 0)
	envSlugs := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.EnvironmentSlugs)
	if envSlugs != nil {
		environments = append(environments, envSlugs...)
	}

	secretApprovalPolicy, err := r.client.CreateSecretApprovalPolicy(infisical.CreateSecretApprovalPolicyRequest{
		Name:                 plan.Name.ValueString(),
		ProjectID:            plan.ProjectID.ValueString(),
		Environments:         environments,
		SecretPath:           plan.SecretPath.ValueString(),
		Approvers:            approvers,
		Bypassers:            bypassers,
		RequiredApprovals:    plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:     plan.EnforcementLevel.ValueString(),
		AllowedSelfApprovals: plan.AllowSelfApproval.ValueBool(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret approval policy",
			"Couldn't save secret approval policy, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(secretApprovalPolicy.SecretApprovalPolicy.ID)
	plan.Name = types.StringValue(secretApprovalPolicy.SecretApprovalPolicy.Name)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *secretApprovalPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read secret approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state secretApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	secretApprovalPolicy, err := r.client.GetSecretApprovalPolicyByID(infisical.GetSecretApprovalPolicyByIDRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error fetching secret approval policy from your project",
				"Couldn't read secret approval policy from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	policy := secretApprovalPolicy.SecretApprovalPolicy

	state.Name = types.StringValue(policy.Name)
	state.SecretPath = types.StringValue(policy.SecretPath)
	state.RequiredApprovals = types.Int64Value(policy.RequiredApprovals)
	state.EnforcementLevel = types.StringValue(policy.EnforcementLevel)
	state.AllowSelfApproval = types.BoolValue(policy.AllowedSelfApprovals)

	approvers := make([]SecretApprover, len(policy.Approvers))
	for i, el := range policy.Approvers {
		if el.Type == "user" {
			approvers[i] = SecretApprover{
				Name: types.StringValue(el.Name),
				Type: types.StringValue(el.Type),
			}
		} else {
			approvers[i] = SecretApprover{
				ID:   types.StringValue(el.ID),
				Type: types.StringValue(el.Type),
			}
		}
	}
	state.Approvers = approvers
	state.Bypassers = mapSecretBypassersFromAPI(policy.Bypassers)

	if len(policy.Environments) > 0 {
		var environmentSlugs []string
		for _, env := range policy.Environments {
			environmentSlugs = append(environmentSlugs, env.Slug)
		}

		environmentSlugsList, diags := types.ListValueFrom(ctx, types.StringType, environmentSlugs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.EnvironmentSlugs = environmentSlugsList
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *secretApprovalPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan secretApprovalPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state secretApprovalPolicyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectID != plan.ProjectID {
		resp.Diagnostics.AddError(
			"Unable to update secret approval policy",
			fmt.Sprintf("Cannot change project ID, previous project ID: %s, new project ID: %s", state.ProjectID, plan.ProjectID),
		)
		return
	}

	if infisicaltf.IsAttrValueEmpty(plan.EnvironmentSlugs) {
		resp.Diagnostics.AddError(
			"Unable to update secret approval policy",
			fmt.Sprintf("Cannot change environment to empty list. previous environment: %s, new environment: %s", state.EnvironmentSlugs, plan.EnvironmentSlugs),
		)
		return
	}

	approvers, ok := validateAndMapApprovers(plan.Approvers, &resp.Diagnostics)
	if !ok {
		return
	}

	var updateApprovers []infisical.UpdateSecretApprovalPolicyApprover
	for _, a := range approvers {
		updateApprovers = append(updateApprovers, infisical.UpdateSecretApprovalPolicyApprover{
			ID:   a.ID,
			Name: a.Name,
			Type: a.Type,
		})
	}

	bypasserInputs := make([]infisicaltf.BypasserInput, len(plan.Bypassers))
	for i, el := range plan.Bypassers {
		bypasserInputs[i] = infisicaltf.BypasserInput{Type: el.Type, ID: el.ID, Name: el.Name}
	}
	validatedBypassers, bypassersOk := infisicaltf.ValidateAndMapBypassers(bypasserInputs, &resp.Diagnostics)
	if !bypassersOk {
		return
	}
	var updateBypassers []infisical.UpdateSecretApprovalPolicyBypasser
	for _, b := range validatedBypassers {
		updateBypassers = append(updateBypassers, infisical.UpdateSecretApprovalPolicyBypasser{
			ID: b.ID, Name: b.Name, Type: b.Type,
		})
	}

	environments := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.EnvironmentSlugs)

	_, err := r.client.UpdateSecretApprovalPolicy(infisical.UpdateSecretApprovalPolicyRequest{
		ID:                   plan.ID.ValueString(),
		Name:                 plan.Name.ValueString(),
		SecretPath:           plan.SecretPath.ValueString(),
		Approvers:            updateApprovers,
		Bypassers:            updateBypassers,
		RequiredApprovals:    plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:     plan.EnforcementLevel.ValueString(),
		AllowedSelfApprovals: plan.AllowSelfApproval.ValueBool(),
		Environments:         environments,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret approval policy",
			"Couldn't update secret approval policy, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *secretApprovalPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state secretApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretApprovalPolicy(infisical.DeleteSecretApprovalPolicyRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting secret approval policy",
			"Couldn't delete secret approval policy, unexpected error: "+err.Error(),
		)
		return
	}
}

func validateAndMapApprovers(planApprovers []SecretApprover, diagnostics *diag.Diagnostics) ([]infisical.CreateSecretApprovalPolicyApprover, bool) {
	var approvers []infisical.CreateSecretApprovalPolicyApprover
	for _, el := range planApprovers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				diagnostics.AddError(
					"Field username is required for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return nil, false
			}
			if !el.ID.IsNull() {
				diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return nil, false
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				diagnostics.AddError(
					"Field ID is required for group approvers",
					"Must provide ID for group approvers",
				)
				return nil, false
			}
			if !el.Name.IsNull() {
				diagnostics.AddError(
					"Field name cannot be used for group approvers",
					"Must provide ID for group approvers",
				)
				return nil, false
			}
		}

		approvers = append(approvers, infisical.CreateSecretApprovalPolicyApprover{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}
	return approvers, true
}

func mapSecretBypassersFromAPI(apiBypassers []infisical.SecretApprovalPolicyBypasser) []SecretBypasser {
	if len(apiBypassers) == 0 {
		return nil
	}
	bypassers := make([]SecretBypasser, len(apiBypassers))
	for i, el := range apiBypassers {
		if el.Type == "user" {
			bypassers[i] = SecretBypasser{
				Name: types.StringValue(el.Name),
				Type: types.StringValue(el.Type),
			}
		} else {
			bypassers[i] = SecretBypasser{
				ID:   types.StringValue(el.ID),
				Type: types.StringValue(el.Type),
			}
		}
	}
	return bypassers
}
