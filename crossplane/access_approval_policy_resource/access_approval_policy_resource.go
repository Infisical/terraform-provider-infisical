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

// NewAccessApprovalPolicyResource is a helper function to simplify the provider implementation.
func NewAccessApprovalPolicyResource() resource.Resource {
	return &accessApprovalPolicyResource{}
}

// accessApprovalPolicyResource is the resource implementation.
type accessApprovalPolicyResource struct {
	client *infisical.Client
}

type AccessApprover struct {
	Type     types.String `tfsdk:"type"`
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"username"`
	Sequence types.Int64  `tfsdk:"step"`
}

type AccessApprovalPolicyApprovalsRequired struct {
	NumberOfApprovals types.Int64 `tfsdk:"number_of_approvals"`
	StepNumber        types.Int64 `tfsdk:"step_number"`
}

// accessApprovalPolicyResourceModel describes the data source data model.
type accessApprovalPolicyResourceModel struct {
	ID                    types.String                            `tfsdk:"id"`
	ProjectID             types.String                            `tfsdk:"project_id"`
	Name                  types.String                            `tfsdk:"name"`
	EnvironmentSlugs      types.List                              `tfsdk:"environment_slugs"`
	SecretPath            types.String                            `tfsdk:"secret_path"`
	Approvers             []AccessApprover                        `tfsdk:"approvers"`
	GroupApprovers        types.List                              `tfsdk:"group_approvers"`
	UserApprovers         types.List                              `tfsdk:"user_approvers"`
	GroupBypassers        types.List                              `tfsdk:"group_bypassers"`
	UserBypassers         types.List                              `tfsdk:"user_bypassers"`
	RequiredApprovals     types.Int64                             `tfsdk:"required_approvals"`
	EnforcementLevel      types.String                            `tfsdk:"enforcement_level"`
	AllowSelfApproval     types.Bool                              `tfsdk:"allow_self_approval"`
	ApprovalsRequired     []AccessApprovalPolicyApprovalsRequired `tfsdk:"approvals_required"`
	MaxTimePeriod         types.String                            `tfsdk:"max_time_period"`
	RequestExpirationTime types.String                            `tfsdk:"request_expiration_time"`
}

// Metadata returns the resource type name.
func (r *accessApprovalPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_approval_policy"
}

// Schema defines the schema for the resource.
func (r *accessApprovalPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create access approval policy for your projects",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the access approval policy",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project to add the access approval policy",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description:   "The name of the access approval policy",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_slugs": schema.ListAttribute{
				Description:   "The environments to apply the access approval policy to",
				Required:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{pkg.UnorderedList()},
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path to apply the access approval policy to",
				Required:    true,
			},
			"approvers": schema.ListNestedAttribute{
				Optional:    true,
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
						"step": schema.Int64Attribute{
							Description: "The step number of the approver",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"group_approvers": schema.ListAttribute{
				Description:   "(DEPRECATED, use approvers instead) Array of group IDs to assign as approvers. Uses step 1 by default.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{pkg.UnorderedList()},
			},
			"user_approvers": schema.ListAttribute{
				Description:   "(DEPRECATED, use approvers instead) Array of usernames to assign as approvers. Uses step 1 by default.",
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
				Description: "Whether to allow approvers to approve their own requests",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"approvals_required": schema.ListNestedAttribute{
				Optional:    true,
				Description: "The number of approvals required per step for multi-step approval policies",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"number_of_approvals": schema.Int64Attribute{
							Description: "The number of approvals required for this step",
							Required:    true,
						},
						"step_number": schema.Int64Attribute{
							Description: "The step number this approval count applies to",
							Required:    true,
						},
					},
				},
			},
			"max_time_period": schema.StringAttribute{
				Description: "The maximum time period for the access approval, specified as a duration string (e.g. '1h', '30m', '2d'). Use 'permanent' or leave empty for no limit.",
				Optional:    true,
			},
			"request_expiration_time": schema.StringAttribute{
				Description: "The time after which the access request expires, specified as a duration string (e.g. '1h', '3d', '72h'). Must be between 1 minute and 1 year. Use 'never' or leave empty for no expiration.",
				Optional:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *accessApprovalPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *accessApprovalPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create access approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan accessApprovalPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectDetail, err := r.client.GetProjectById(infisical.GetProjectByIdRequest{
		ID: plan.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating access approval policy",
			"Couldn't fetch project details, unexpected error: "+err.Error(),
		)
		return
	}

	allApprovers, ok := mergeDeprecatedApprovers(ctx, &resp.Diagnostics, plan.Approvers, plan.GroupApprovers, plan.UserApprovers)
	if !ok {
		return
	}

	validatedApprovers, ok := validateAndMapApproversFromPlan(allApprovers, &resp.Diagnostics)
	if !ok {
		return
	}

	var approvers []infisical.CreateAccessApprovalPolicyApprover
	for _, a := range validatedApprovers {
		approvers = append(approvers, infisical.CreateAccessApprovalPolicyApprover{
			ID: a.ID, Name: a.Name, Type: a.Type, Sequence: a.Sequence,
		})
	}

	bypasserOutputs := buildBypassersFromLists(ctx, resp.Diagnostics, plan.GroupBypassers, plan.UserBypassers)
	var bypassers []infisical.CreateAccessApprovalPolicyBypasser
	for _, b := range bypasserOutputs {
		bypassers = append(bypassers, infisical.CreateAccessApprovalPolicyBypasser{
			ID: b.ID, Name: b.Name, Type: b.Type,
		})
	}

	environments := make([]string, 0)
	envSlugs := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.EnvironmentSlugs)
	if envSlugs != nil {
		environments = append(environments, envSlugs...)
	}

	accessApprovalPolicy, err := r.client.CreateAccessApprovalPolicy(infisical.CreateAccessApprovalPolicyRequest{
		Name:                  plan.Name.ValueString(),
		ProjectSlug:           projectDetail.Slug,
		Environments:          environments,
		SecretPath:            plan.SecretPath.ValueString(),
		Approvers:             approvers,
		Bypassers:             bypassers,
		RequiredApprovals:     plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:      plan.EnforcementLevel.ValueString(),
		AllowedSelfApprovals:  plan.AllowSelfApproval.ValueBool(),
		ApprovalsRequired:     mapApprovalsRequiredFromPlan(plan.ApprovalsRequired),
		MaxTimePeriod:         infisicaltf.OptionalStringPointer(plan.MaxTimePeriod),
		RequestExpirationTime: infisicaltf.OptionalStringPointer(plan.RequestExpirationTime),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating access approval policy",
			"Couldn't save access approval policy, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.ID)
	plan.Name = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.Name)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *accessApprovalPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read access approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state accessApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	accessApprovalPolicy, err := r.client.GetAccessApprovalPolicyByID(infisical.GetAccessApprovalPolicyByIDRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error fetching access approval policy from your project",
				"Couldn't read access approval policy from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	policy := accessApprovalPolicy.AccessApprovalPolicy

	if policy.DeletedAt != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(policy.Name)
	state.SecretPath = types.StringValue(policy.SecretPath)
	state.RequiredApprovals = types.Int64Value(policy.RequiredApprovals)
	state.EnforcementLevel = types.StringValue(policy.EnforcementLevel)
	state.AllowSelfApproval = types.BoolValue(policy.AllowedSelfApprovals)
	state.MaxTimePeriod = infisicaltf.StringPointerToTypesString(policy.MaxTimePeriod)
	state.RequestExpirationTime = infisicaltf.StringPointerToTypesString(policy.RequestExpirationTime)
	state.Approvers = mapApproversFromAPI(policy.Approvers)
	state.ApprovalsRequired = mapApprovalsRequiredFromAPI(policy.Approvers)

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

	if len(groupBypassers) > 0 {
		state.GroupBypassers, diags = types.ListValueFrom(ctx, types.StringType, groupBypassers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if len(userBypassers) > 0 {
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

// Update updates the resource and sets the updated Terraform state on success.
func (r *accessApprovalPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update access approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan accessApprovalPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state accessApprovalPolicyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectID != plan.ProjectID {
		resp.Diagnostics.AddError(
			"Unable to update access approval policy",
			fmt.Sprintf("Cannot change project ID, previous project ID: %s, new project ID: %s", state.ProjectID, plan.ProjectID),
		)
		return
	}

	if infisicaltf.IsAttrValueEmpty(plan.EnvironmentSlugs) {
		resp.Diagnostics.AddError(
			"Unable to update access approval policy",
			fmt.Sprintf("Cannot change environment to empty list. previous environment: %s, new environment: %s", state.EnvironmentSlugs, plan.EnvironmentSlugs),
		)
		return
	}

	allApprovers, ok := mergeDeprecatedApprovers(ctx, &resp.Diagnostics, plan.Approvers, plan.GroupApprovers, plan.UserApprovers)
	if !ok {
		return
	}

	validatedApprovers, ok := validateAndMapApproversFromPlan(allApprovers, &resp.Diagnostics)
	if !ok {
		return
	}

	var approvers []infisical.UpdateAccessApprovalPolicyApprover
	for _, a := range validatedApprovers {
		approvers = append(approvers, infisical.UpdateAccessApprovalPolicyApprover{
			ID: a.ID, Name: a.Name, Type: a.Type, Sequence: a.Sequence,
		})
	}

	bypasserOutputs := buildBypassersFromLists(ctx, resp.Diagnostics, plan.GroupBypassers, plan.UserBypassers)
	var updateBypassers []infisical.UpdateAccessApprovalPolicyBypasser
	for _, b := range bypasserOutputs {
		updateBypassers = append(updateBypassers, infisical.UpdateAccessApprovalPolicyBypasser{
			ID: b.ID, Name: b.Name, Type: b.Type,
		})
	}

	environments := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.EnvironmentSlugs)

	_, err := r.client.UpdateAccessApprovalPolicy(infisical.UpdateAccessApprovalPolicyRequest{
		ID:                    plan.ID.ValueString(),
		Name:                  plan.Name.ValueString(),
		SecretPath:            plan.SecretPath.ValueString(),
		Approvers:             approvers,
		Bypassers:             updateBypassers,
		RequiredApprovals:     plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:      plan.EnforcementLevel.ValueString(),
		AllowedSelfApprovals:  plan.AllowSelfApproval.ValueBool(),
		ApprovalsRequired:     mapApprovalsRequiredFromPlan(plan.ApprovalsRequired),
		MaxTimePeriod:         infisicaltf.OptionalStringPointer(plan.MaxTimePeriod),
		RequestExpirationTime: infisicaltf.OptionalStringPointer(plan.RequestExpirationTime),
		Environments:          environments,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating access approval policy",
			"Couldn't update access approval policy, unexpected error: "+err.Error(),
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
func (r *accessApprovalPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete access approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state accessApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteAccessApprovalPolicy(infisical.DeleteAccessApprovalPolicyRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting access approval policy",
			"Couldn't delete access approval policy, unexpected error: "+err.Error(),
		)
		return
	}
}

func validateAndMapApproversFromPlan(planApprovers []AccessApprover, diagnostics *diag.Diagnostics) ([]infisicaltf.AccessApproverOutput, bool) {
	inputs := make([]infisicaltf.AccessApproverInput, len(planApprovers))
	for i, el := range planApprovers {
		inputs[i] = infisicaltf.AccessApproverInput{Type: el.Type, ID: el.ID, Name: el.Name, Sequence: el.Sequence}
	}
	return infisicaltf.ValidateAndMapApprovers(inputs, diagnostics)
}

type bypasserOutput struct {
	Type string
	ID   string
	Name string
}

func buildBypassersFromLists(ctx context.Context, diagnostics diag.Diagnostics, groupBypassersList, userBypassersList types.List) []bypasserOutput {
	var result []bypasserOutput

	if userBypassers := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, userBypassersList); userBypassers != nil {
		for _, username := range userBypassers {
			result = append(result, bypasserOutput{Name: username, Type: "user"})
		}
	}

	if groupBypassers := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, groupBypassersList); groupBypassers != nil {
		for _, groupId := range groupBypassers {
			result = append(result, bypasserOutput{ID: groupId, Type: "group"})
		}
	}

	return result
}

func mapApprovalsRequiredFromPlan(planApprovalsRequired []AccessApprovalPolicyApprovalsRequired) []infisical.AccessApprovalPolicyApprovalsRequired {
	var result []infisical.AccessApprovalPolicyApprovalsRequired
	for _, ar := range planApprovalsRequired {
		result = append(result, infisical.AccessApprovalPolicyApprovalsRequired{
			NumberOfApprovals: ar.NumberOfApprovals.ValueInt64(),
			StepNumber:        ar.StepNumber.ValueInt64(),
		})
	}
	return result
}

func mapApproversFromAPI(apiApprovers []infisical.AccessApprovalPolicyApprover) []AccessApprover {
	approvers := make([]AccessApprover, len(apiApprovers))
	for i, el := range apiApprovers {
		if el.Type == "user" {
			approvers[i] = AccessApprover{
				Name:     types.StringValue(el.Name),
				Type:     types.StringValue(el.Type),
				Sequence: types.Int64Value(el.Sequence),
			}
		} else {
			approvers[i] = AccessApprover{
				ID:       types.StringValue(el.ID),
				Type:     types.StringValue(el.Type),
				Sequence: types.Int64Value(el.Sequence),
			}
		}
	}
	return approvers
}

func mergeDeprecatedApprovers(ctx context.Context, diagnostics *diag.Diagnostics, planApprovers []AccessApprover, groupApprovers, userApprovers types.List) ([]AccessApprover, bool) {
	hasNewFormat := len(planApprovers) > 0
	hasOldFormat := !groupApprovers.IsNull() || !userApprovers.IsNull()

	if hasNewFormat && hasOldFormat {
		diagnostics.AddError(
			"Conflicting approver configuration",
			"Cannot use both 'approvers' and the deprecated 'group_approvers'/'user_approvers' fields. Use 'approvers' for new configurations.",
		)
		return nil, false
	}

	if hasNewFormat {
		return planApprovers, true
	}

	result := make([]AccessApprover, 0)

	if groupIDs := infisicaltf.StringListToGoStringSlice(ctx, *diagnostics, groupApprovers); groupIDs != nil {
		for _, id := range groupIDs {
			result = append(result, AccessApprover{
				Type:     types.StringValue("group"),
				ID:       types.StringValue(id),
				Sequence: types.Int64Value(1),
			})
		}
	}

	if usernames := infisicaltf.StringListToGoStringSlice(ctx, *diagnostics, userApprovers); usernames != nil {
		for _, username := range usernames {
			result = append(result, AccessApprover{
				Type:     types.StringValue("user"),
				Name:     types.StringValue(username),
				Sequence: types.Int64Value(1),
			})
		}
	}

	return result, true
}

func mapApprovalsRequiredFromAPI(apiApprovers []infisical.AccessApprovalPolicyApprover) []AccessApprovalPolicyApprovalsRequired {
	approvalsRequiredMap := make(map[int64]int64)
	for _, approver := range apiApprovers {
		if approver.ApprovalsRequired > 0 {
			approvalsRequiredMap[approver.Sequence] = approver.ApprovalsRequired
		}
	}
	if len(approvalsRequiredMap) == 0 {
		return nil
	}
	result := make([]AccessApprovalPolicyApprovalsRequired, 0, len(approvalsRequiredMap))
	for stepNumber, numberOfApprovals := range approvalsRequiredMap {
		result = append(result, AccessApprovalPolicyApprovalsRequired{
			NumberOfApprovals: types.Int64Value(numberOfApprovals),
			StepNumber:        types.Int64Value(stepNumber),
		})
	}
	return result
}
