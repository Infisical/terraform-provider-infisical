package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

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
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"username"`
}

type AccessBypasser struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"username"`
}

type AccessApprovalsRequired struct {
	NumberOfApprovals types.Int64 `tfsdk:"number_of_approvals"`
	StepNumber        types.Int64 `tfsdk:"step_number"`
}

// accessApprovalPolicyResourceModel describes the data source data model.
type accessApprovalPolicyResourceModel struct {
	ID                    types.String              `tfsdk:"id"`
	ProjectID             types.String              `tfsdk:"project_id"`
	Name                  types.String              `tfsdk:"name"`
	EnvironmentSlugs      types.List                `tfsdk:"environment_slugs"`
	EnvironmentSlug       types.String              `tfsdk:"environment_slug"`
	SecretPath            types.String              `tfsdk:"secret_path"`
	Approvers             []AccessApprover          `tfsdk:"approvers"`
	Bypassers             []AccessBypasser          `tfsdk:"bypassers"`
	RequiredApprovals     types.Int64               `tfsdk:"required_approvals"`
	EnforcementLevel      types.String              `tfsdk:"enforcement_level"`
	AllowSelfApproval     types.Bool                `tfsdk:"allow_self_approval"`
	ApprovalsRequired     []AccessApprovalsRequired `tfsdk:"approvals_required"`
	MaxTimePeriod         types.String              `tfsdk:"max_time_period"`
	RequestExpirationTime types.String              `tfsdk:"request_expiration_time"`
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
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{pkg.UnorderedList()},
			},
			"environment_slug": schema.StringAttribute{
				Description: "(DEPRECATED, Use environment_slugs instead) The environment to apply the access approval policy to",
				Optional:    true,
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path to apply the access approval policy to",
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

	var approvers []infisical.CreateAccessApprovalPolicyApprover
	for _, el := range plan.Approvers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username is required for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
			if !el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID is required for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
			if !el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username cannot be used for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
		}

		approvers = append(approvers, infisical.CreateAccessApprovalPolicyApprover{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}
	var bypassers []infisical.CreateAccessApprovalPolicyBypasser
	for _, el := range plan.Bypassers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username is required for user bypassers",
					"Must provide username for user bypassers. By default, this is the email",
				)
				return
			}
			if !el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user bypassers",
					"Must provide username for user bypassers. By default, this is the email",
				)
				return
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID is required for group bypassers",
					"Must provide ID for group bypassers",
				)
				return
			}
			if !el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username cannot be used for group bypassers",
					"Must provide ID for group bypassers",
				)
				return
			}
		}

		bypassers = append(bypassers, infisical.CreateAccessApprovalPolicyBypasser{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}

	var approvalsRequired []infisical.AccessApprovalPolicyApprovalsRequired
	for _, ar := range plan.ApprovalsRequired {
		approvalsRequired = append(approvalsRequired, infisical.AccessApprovalPolicyApprovalsRequired{
			NumberOfApprovals: ar.NumberOfApprovals.ValueInt64(),
			StepNumber:        ar.StepNumber.ValueInt64(),
		})
	}

	var maxTimePeriod *string
	if !plan.MaxTimePeriod.IsNull() && !plan.MaxTimePeriod.IsUnknown() {
		v := plan.MaxTimePeriod.ValueString()
		maxTimePeriod = &v
	}

	var requestExpirationTime *string
	if !plan.RequestExpirationTime.IsNull() && !plan.RequestExpirationTime.IsUnknown() {
		v := plan.RequestExpirationTime.ValueString()
		requestExpirationTime = &v
	}

	if plan.EnvironmentSlugs.IsNull() && plan.EnvironmentSlug.IsNull() {
		resp.Diagnostics.AddError(
			"Error creating access approval policy",
			"Must provide either environment_slugs or environment_slug",
		)
		return
	}

	environments := make([]string, 0)

	if !plan.EnvironmentSlugs.IsNull() {
		envSlugs := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.EnvironmentSlugs)
		if envSlugs != nil {
			environments = append(environments, envSlugs...)
		}
	} else {
		environments = append(environments, plan.EnvironmentSlug.ValueString())
	}

	if plan.EnvironmentSlug.ValueString() != "" && len(environments) > 0 {
		resp.Diagnostics.AddError(
			"Error creating access approval policy",
			"Cannot provide both environment_slugs and environment_slug",
		)
		return
	}

	if !plan.EnvironmentSlug.IsNull() && plan.EnvironmentSlug.ValueString() != "" {
		environments = append(environments, plan.EnvironmentSlug.ValueString())
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
		ApprovalsRequired:     approvalsRequired,
		MaxTimePeriod:         maxTimePeriod,
		RequestExpirationTime: requestExpirationTime,
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

	// Get current state
	var state accessApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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

	state.Name = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.Name)
	state.SecretPath = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.SecretPath)
	state.RequiredApprovals = types.Int64Value(accessApprovalPolicy.AccessApprovalPolicy.RequiredApprovals)
	state.EnforcementLevel = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.EnforcementLevel)
	state.AllowSelfApproval = types.BoolValue(accessApprovalPolicy.AccessApprovalPolicy.AllowedSelfApprovals)

	if accessApprovalPolicy.AccessApprovalPolicy.MaxTimePeriod != nil {
		state.MaxTimePeriod = types.StringValue(*accessApprovalPolicy.AccessApprovalPolicy.MaxTimePeriod)
	} else {
		state.MaxTimePeriod = types.StringNull()
	}

	if accessApprovalPolicy.AccessApprovalPolicy.RequestExpirationTime != nil {
		state.RequestExpirationTime = types.StringValue(*accessApprovalPolicy.AccessApprovalPolicy.RequestExpirationTime)
	} else {
		state.RequestExpirationTime = types.StringNull()
	}

	approvers := make([]AccessApprover, len(accessApprovalPolicy.AccessApprovalPolicy.Approvers))
	for i, el := range accessApprovalPolicy.AccessApprovalPolicy.Approvers {
		if el.Type == "user" {
			approvers[i] = AccessApprover{
				Name: types.StringValue(el.Name),
				Type: types.StringValue(el.Type),
			}
		} else {
			approvers[i] = AccessApprover{
				ID:   types.StringValue(el.ID),
				Type: types.StringValue(el.Type),
			}
		}
	}

	state.Approvers = approvers

	readBypassers := make([]AccessBypasser, len(accessApprovalPolicy.AccessApprovalPolicy.Bypassers))
	for i, el := range accessApprovalPolicy.AccessApprovalPolicy.Bypassers {
		if el.Type == "user" {
			readBypassers[i] = AccessBypasser{
				Name: types.StringValue(el.Name),
				Type: types.StringValue(el.Type),
			}
		} else {
			readBypassers[i] = AccessBypasser{
				ID:   types.StringValue(el.ID),
				Type: types.StringValue(el.Type),
			}
		}
	}

	if len(readBypassers) > 0 {
		state.Bypassers = readBypassers
	} else {
		state.Bypassers = nil
	}

	if len(accessApprovalPolicy.AccessApprovalPolicy.ApprovalsRequired) > 0 {
		readApprovalsRequired := make([]AccessApprovalsRequired, len(accessApprovalPolicy.AccessApprovalPolicy.ApprovalsRequired))
		for i, ar := range accessApprovalPolicy.AccessApprovalPolicy.ApprovalsRequired {
			readApprovalsRequired[i] = AccessApprovalsRequired{
				NumberOfApprovals: types.Int64Value(ar.NumberOfApprovals),
				StepNumber:        types.Int64Value(ar.StepNumber),
			}
		}
		state.ApprovalsRequired = readApprovalsRequired
	} else {
		state.ApprovalsRequired = nil
	}

	if len(accessApprovalPolicy.AccessApprovalPolicy.Environments) > 0 && !state.EnvironmentSlugs.IsNull() {
		// Extract environment slugs from the environment objects
		var environmentSlugs []string
		for _, env := range accessApprovalPolicy.AccessApprovalPolicy.Environments {
			environmentSlugs = append(environmentSlugs, env.Slug)
		}

		// Always set the new environment_slugs field
		environmentSlugsList, diags := types.ListValueFrom(ctx, types.StringType, environmentSlugs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.EnvironmentSlugs = environmentSlugsList

		// For backward compatibility, set environment_slug if there's exactly one environment
		if len(environmentSlugs) == 1 && !state.EnvironmentSlug.IsNull() {
			state.EnvironmentSlug = types.StringValue(environmentSlugs[0])
		}
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

	if state.EnvironmentSlug != plan.EnvironmentSlug && infisicaltf.IsAttrValueEmpty(plan.EnvironmentSlugs) {
		resp.Diagnostics.AddError(
			"Unable to update access approval policy",
			fmt.Sprintf("Cannot change environment, previous environment: %s, new environment: %s", state.EnvironmentSlug, plan.EnvironmentSlug),
		)
		return
	}

	var approvers []infisical.UpdateAccessApprovalPolicyApprover
	for _, el := range plan.Approvers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username is required for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
			if !el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID is required for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
			if !el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username cannot be used for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
		}

		approvers = append(approvers, infisical.UpdateAccessApprovalPolicyApprover{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}

	var updateBypassers []infisical.UpdateAccessApprovalPolicyBypasser
	for _, el := range plan.Bypassers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username is required for user bypassers",
					"Must provide username for user bypassers. By default, this is the email",
				)
				return
			}
			if !el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user bypassers",
					"Must provide username for user bypassers. By default, this is the email",
				)
				return
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID is required for group bypassers",
					"Must provide ID for group bypassers",
				)
				return
			}
			if !el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username cannot be used for group bypassers",
					"Must provide ID for group bypassers",
				)
				return
			}
		}

		updateBypassers = append(updateBypassers, infisical.UpdateAccessApprovalPolicyBypasser{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}

	var updateApprovalsRequired []infisical.AccessApprovalPolicyApprovalsRequired
	for _, ar := range plan.ApprovalsRequired {
		updateApprovalsRequired = append(updateApprovalsRequired, infisical.AccessApprovalPolicyApprovalsRequired{
			NumberOfApprovals: ar.NumberOfApprovals.ValueInt64(),
			StepNumber:        ar.StepNumber.ValueInt64(),
		})
	}

	var updateMaxTimePeriod *string
	if !plan.MaxTimePeriod.IsNull() && !plan.MaxTimePeriod.IsUnknown() {
		v := plan.MaxTimePeriod.ValueString()
		updateMaxTimePeriod = &v
	}

	var updateRequestExpirationTime *string
	if !plan.RequestExpirationTime.IsNull() && !plan.RequestExpirationTime.IsUnknown() {
		v := plan.RequestExpirationTime.ValueString()
		updateRequestExpirationTime = &v
	}

	var environments []string
	if !state.EnvironmentSlugs.IsNull() {
		environments = infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.EnvironmentSlugs)
	} else {
		environments = []string{plan.EnvironmentSlug.ValueString()}
	}

	_, err := r.client.UpdateAccessApprovalPolicy(infisical.UpdateAccessApprovalPolicyRequest{
		ID:                    plan.ID.ValueString(),
		Name:                  plan.Name.ValueString(),
		SecretPath:            plan.SecretPath.ValueString(),
		Approvers:             approvers,
		Bypassers:             updateBypassers,
		RequiredApprovals:     plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:      plan.EnforcementLevel.ValueString(),
		AllowedSelfApprovals:  plan.AllowSelfApproval.ValueBool(),
		ApprovalsRequired:     updateApprovalsRequired,
		MaxTimePeriod:         updateMaxTimePeriod,
		RequestExpirationTime: updateRequestExpirationTime,
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
