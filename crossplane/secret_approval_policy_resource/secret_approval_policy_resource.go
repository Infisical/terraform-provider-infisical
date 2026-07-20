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

type secretApprovalPolicyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	Name              types.String `tfsdk:"name"`
	EnvironmentSlugs  types.List   `tfsdk:"environment_slugs"`
	SecretPath        types.String `tfsdk:"secret_path"`
	GroupApprovers    types.List   `tfsdk:"group_approvers"`
	UserApprovers     types.List   `tfsdk:"user_approvers"`
	GroupBypassers    types.List   `tfsdk:"group_bypassers"`
	UserBypassers     types.List   `tfsdk:"user_bypassers"`
	RequiredApprovals types.Int64  `tfsdk:"required_approvals"`
	EnforcementLevel  types.String `tfsdk:"enforcement_level"`
	AllowSelfApproval types.Bool   `tfsdk:"allow_self_approval"`
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
			"group_approvers": schema.ListAttribute{
				Description:   "Array of group IDs to assign as approvers",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{pkg.UnorderedList()},
			},
			"user_approvers": schema.ListAttribute{
				Description:   "Array of usernames to assign as approvers",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{pkg.UnorderedList()},
			},
			"group_bypassers": schema.ListAttribute{
				Optional:      true,
				Description:   "Array of group IDs belonging to the groups to assign as bypassers",
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{pkg.UnorderedList()},
			},
			"user_bypassers": schema.ListAttribute{
				Optional:      true,
				Description:   "Array of usernames belonging to the users to assign as bypassers",
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{pkg.UnorderedList()},
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

	approverOutputs := buildSecretApproversFromLists(ctx, resp.Diagnostics, plan.GroupApprovers, plan.UserApprovers)
	var approvers []infisical.CreateSecretApprovalPolicyApprover
	for _, a := range approverOutputs {
		approvers = append(approvers, infisical.CreateSecretApprovalPolicyApprover{
			ID: a.ID, Name: a.Name, Type: a.Type,
		})
	}

	bypasserOutputs := buildSecretBypassersFromLists(ctx, resp.Diagnostics, plan.GroupBypassers, plan.UserBypassers)
	var bypassers []infisical.CreateSecretApprovalPolicyBypasser
	for _, b := range bypasserOutputs {
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

	if policy.DeletedAt != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(policy.Name)
	state.SecretPath = types.StringValue(policy.SecretPath)
	state.RequiredApprovals = types.Int64Value(policy.RequiredApprovals)
	state.EnforcementLevel = types.StringValue(policy.EnforcementLevel)
	state.AllowSelfApproval = types.BoolValue(policy.AllowedSelfApprovals)

	if !state.GroupApprovers.IsNull() {
		groupApproverIDs := make([]string, 0)
		for _, el := range policy.Approvers {
			if el.Type == "group" {
				groupApproverIDs = append(groupApproverIDs, el.ID)
			}
		}
		state.GroupApprovers, diags = types.ListValueFrom(ctx, types.StringType, groupApproverIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !state.UserApprovers.IsNull() {
		userApproverNames := make([]string, 0)
		for _, el := range policy.Approvers {
			if el.Type == "user" {
				userApproverNames = append(userApproverNames, el.Name)
			}
		}
		state.UserApprovers, diags = types.ListValueFrom(ctx, types.StringType, userApproverNames)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	groupBypassers := make([]string, 0)
	userBypassers := make([]string, 0)
	for _, el := range policy.Bypassers {
		if el.Type == "user" {
			userBypassers = append(userBypassers, el.Name)
		} else {
			groupBypassers = append(groupBypassers, el.ID)
		}
	}

	if !state.GroupBypassers.IsNull() {
		state.GroupBypassers, diags = types.ListValueFrom(ctx, types.StringType, groupBypassers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !state.UserBypassers.IsNull() {
		state.UserBypassers, diags = types.ListValueFrom(ctx, types.StringType, userBypassers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

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

	approverOutputs := buildSecretApproversFromLists(ctx, resp.Diagnostics, plan.GroupApprovers, plan.UserApprovers)
	var updateApprovers []infisical.UpdateSecretApprovalPolicyApprover
	for _, a := range approverOutputs {
		updateApprovers = append(updateApprovers, infisical.UpdateSecretApprovalPolicyApprover{
			ID: a.ID, Name: a.Name, Type: a.Type,
		})
	}

	bypasserOutputs := buildSecretBypassersFromLists(ctx, resp.Diagnostics, plan.GroupBypassers, plan.UserBypassers)
	var updateBypassers []infisical.UpdateSecretApprovalPolicyBypasser
	for _, b := range bypasserOutputs {
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

type secretApproverOutput struct {
	Type string
	ID   string
	Name string
}

func buildSecretApproversFromLists(ctx context.Context, diagnostics diag.Diagnostics, groupApproversList, userApproversList types.List) []secretApproverOutput {
	var result []secretApproverOutput

	if userApprovers := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, userApproversList); userApprovers != nil {
		for _, username := range userApprovers {
			result = append(result, secretApproverOutput{Name: username, Type: "user"})
		}
	}

	if groupApprovers := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, groupApproversList); groupApprovers != nil {
		for _, groupId := range groupApprovers {
			result = append(result, secretApproverOutput{ID: groupId, Type: "group"})
		}
	}

	return result
}

type secretBypasserOutput struct {
	Type string
	ID   string
	Name string
}

func buildSecretBypassersFromLists(ctx context.Context, diagnostics diag.Diagnostics, groupBypassersList, userBypassersList types.List) []secretBypasserOutput {
	var result []secretBypasserOutput

	if userBypassers := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, userBypassersList); userBypassers != nil {
		for _, username := range userBypassers {
			result = append(result, secretBypasserOutput{Name: username, Type: "user"})
		}
	}

	if groupBypassers := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, groupBypassersList); groupBypassers != nil {
		for _, groupId := range groupBypassers {
			result = append(result, secretBypasserOutput{ID: groupId, Type: "group"})
		}
	}

	return result
}
