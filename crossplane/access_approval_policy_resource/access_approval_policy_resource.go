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

// accessApprovalPolicyResourceModel describes the data source data model.
type accessApprovalPolicyResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ProjectID             types.String `tfsdk:"project_id"`
	Name                  types.String `tfsdk:"name"`
	EnvironmentSlugs      types.List   `tfsdk:"environment_slugs"`
	SecretPath            types.String `tfsdk:"secret_path"`
	GroupApprovers        types.List   `tfsdk:"group_approvers"`
	UserApprovers         types.List   `tfsdk:"user_approvers"`
	GroupBypassers        types.List   `tfsdk:"group_bypassers"`
	UserBypassers         types.List   `tfsdk:"user_bypassers"`
	RequiredApprovals     types.Int64  `tfsdk:"required_approvals"`
	EnforcementLevel      types.String `tfsdk:"enforcement_level"`
	AllowSelfApproval     types.Bool   `tfsdk:"allow_self_approval"`
	MaxTimePeriod         types.String `tfsdk:"max_time_period"`
	RequestExpirationTime types.String `tfsdk:"request_expiration_time"`
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
				Description: "Whether to allow approvers to approve their own requests",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"max_time_period": schema.StringAttribute{
				Description: "The maximum time period for the access approval, specified as a duration string (e.g. '1h', '30m', '2d'). If omitted, the default behavior is 'permanent'.",
				Optional:    true,
			},
			"request_expiration_time": schema.StringAttribute{
				Description: "The time after which the access request expires, specified as a duration string (e.g. '1h', '3d', '72h'). Must be between 1 minute and 1 year. If omitted, the default behavior is 'never'.",
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

	approvers := buildApproversFromLists(ctx, resp.Diagnostics, plan.GroupApprovers, plan.UserApprovers)
	var createApprovers []infisical.CreateAccessApprovalPolicyApprover
	for _, a := range approvers {
		createApprovers = append(createApprovers, infisical.CreateAccessApprovalPolicyApprover{
			ID: a.ID, Name: a.Name, Type: a.Type,
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
		Approvers:             createApprovers,
		Bypassers:             bypassers,
		RequiredApprovals:     plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:      plan.EnforcementLevel.ValueString(),
		AllowedSelfApprovals:  plan.AllowSelfApproval.ValueBool(),
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
	for _, approver := range policy.Approvers {
		// The number of required approvers per step is returned in the API inside
		// each approver.
		if approver.Sequence == 1 {
			state.RequiredApprovals = types.Int64Value(approver.ApprovalsRequired)
			break
		}
	}
	state.EnforcementLevel = types.StringValue(policy.EnforcementLevel)
	state.AllowSelfApproval = types.BoolValue(policy.AllowedSelfApprovals)
	state.MaxTimePeriod = infisicaltf.StringPointerToTypesString(policy.MaxTimePeriod)
	state.RequestExpirationTime = infisicaltf.StringPointerToTypesString(policy.RequestExpirationTime)

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

	approverOutputs := buildApproversFromLists(ctx, resp.Diagnostics, plan.GroupApprovers, plan.UserApprovers)
	var approvers []infisical.UpdateAccessApprovalPolicyApprover
	for _, a := range approverOutputs {
		approvers = append(approvers, infisical.UpdateAccessApprovalPolicyApprover{
			ID: a.ID, Name: a.Name, Type: a.Type,
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

type approverOutput struct {
	Type string
	ID   string
	Name string
}

func buildApproversFromLists(ctx context.Context, diagnostics diag.Diagnostics, groupApproversList, userApproversList types.List) []approverOutput {
	var result []approverOutput

	if userApprovers := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, userApproversList); userApprovers != nil {
		for _, username := range userApprovers {
			result = append(result, approverOutput{Name: username, Type: "user"})
		}
	}

	if groupApprovers := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, groupApproversList); groupApprovers != nil {
		for _, groupId := range groupApprovers {
			result = append(result, approverOutput{ID: groupId, Type: "group"})
		}
	}

	return result
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
