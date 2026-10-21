package resource

import (
	"context"
	"errors"
	"fmt"
	"slices"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"
	"terraform-provider-infisical/internal/pkg/validators"
	"time"

	"github.com/hashicorp/go-uuid"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	approvalPolicyScopePkiApplication = "pki-application"

	approverTypeUser  = "user"
	approverTypeGroup = "group"

	maxApprovalPolicyNameLength = 128
	maxApprovalStepNameLength   = 128
	maxRequiredApprovals        = 100
)

var (
	_ resource.Resource                   = &certManagerApplicationApprovalPolicyResource{}
	_ resource.ResourceWithConfigure      = &certManagerApplicationApprovalPolicyResource{}
	_ resource.ResourceWithImportState    = &certManagerApplicationApprovalPolicyResource{}
	_ resource.ResourceWithValidateConfig = &certManagerApplicationApprovalPolicyResource{}
)

var approvalPolicyApproverObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type": types.StringType,
	"id":   types.StringType,
}}

func NewCertManagerApplicationApprovalPolicyResource() resource.Resource {
	return &certManagerApplicationApprovalPolicyResource{}
}

type certManagerApplicationApprovalPolicyResource struct {
	client *infisical.Client
}

type certRequestApprovalPolicyModel struct {
	ID                         types.String                         `tfsdk:"id"`
	Name                       types.String                         `tfsdk:"name"`
	ApplicationID              types.String                         `tfsdk:"application_id"`
	ProfileNames               types.Set                            `tfsdk:"profile_names"`
	BypassForMachineIdentities types.Bool                           `tfsdk:"bypass_for_machine_identities"`
	MaxRequestTTL              types.String                         `tfsdk:"max_request_ttl"`
	Steps                      []certRequestApprovalPolicyStepModel `tfsdk:"steps"`
}

type certRequestApprovalPolicyStepModel struct {
	Name              types.String `tfsdk:"name"`
	RequiredApprovals types.Int64  `tfsdk:"required_approvals"`
	NotifyApprovers   types.Bool   `tfsdk:"notify_approvers"`
	Approvers         types.Set    `tfsdk:"approvers"`
}

type approvalPolicyApproverModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
}

func (r *certManagerApplicationApprovalPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_application_approval_policy"
}

func (r *certManagerApplicationApprovalPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage a Certificate Manager application's approval policy for certificate requests. A certificate requested through the application from one of the policy's certificate profiles waits for the approvals the policy asks for before it is issued.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the approval policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the approval policy.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, maxApprovalPolicyNameLength),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "The ID of the Certificate Manager application the policy covers. Every approver must be a member of the application. Changing it creates a new policy.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"profile_names": schema.SetAttribute{
				Description: "The names of the certificate profiles whose certificate requests need approval.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"bypass_for_machine_identities": schema.BoolAttribute{
				Description: "Whether certificates requested by machine identities skip the approval. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"max_request_ttl": schema.StringAttribute{
				Description: "How long a certificate request can wait for approval before it expires, such as `7d` or `12h`. Must be between 1 hour and 30 days. Leave it out for requests that never expire.",
				Optional:    true,
				Validators: []validator.String{
					validators.MsDurationBetween(time.Hour, 30*24*time.Hour),
				},
			},
			"steps": schema.ListNestedAttribute{
				Description: "The approval steps, in the order a request goes through them. A request moves to the next step once the current one has the approvals it requires.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "An optional name for the step.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, maxApprovalStepNameLength),
							},
						},
						"required_approvals": schema.Int64Attribute{
							Description: fmt.Sprintf("How many of the step's approvers must approve before the request moves on. Must be between 1 and %d.", maxRequiredApprovals),
							Required:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, maxRequiredApprovals),
							},
						},
						"notify_approvers": schema.BoolAttribute{
							Description: "Whether the step's approvers are notified when a request reaches it. Defaults to true.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
						"approvers": schema.SetNestedAttribute{
							Description: "The users and groups that can approve at this step.",
							Required:    true,
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Description: "The type of the approver. Options: " + approverTypeUser + ", " + approverTypeGroup + ".",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.OneOf(approverTypeUser, approverTypeGroup),
										},
									},
									"id": schema.StringAttribute{
										Description: "The ID of the approving user or group.",
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
		},
	}
}

func (r *certManagerApplicationApprovalPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var steps types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("steps"), &steps)...)
	if resp.Diagnostics.HasError() || steps.IsNull() || steps.IsUnknown() {
		return
	}

	for i, element := range steps.Elements() {
		step, ok := element.(types.Object)
		if !ok || step.IsNull() || step.IsUnknown() {
			continue
		}

		requiredApprovals, ok := step.Attributes()["required_approvals"].(types.Int64)
		if !ok || requiredApprovals.IsNull() || requiredApprovals.IsUnknown() {
			continue
		}

		users, countable := countUserApprovers(step.Attributes()["approvers"])
		if countable && requiredApprovals.ValueInt64() > int64(users) {
			resp.Diagnostics.AddAttributeError(
				path.Root("steps").AtListIndex(i).AtName("required_approvals"),
				"Step can never be approved",
				fmt.Sprintf("The step requires %d approvals but only has %d approving users, so no request could ever get through it. Lower required_approvals or add approvers.", requiredApprovals.ValueInt64(), users),
			)
		}
	}
}

func countUserApprovers(value attr.Value) (int, bool) {
	approvers, ok := value.(types.Set)
	if !ok || approvers.IsNull() || approvers.IsUnknown() {
		return 0, false
	}

	users := 0
	for _, element := range approvers.Elements() {
		approver, ok := element.(types.Object)
		if !ok || approver.IsNull() || approver.IsUnknown() {
			return 0, false
		}
		approverType, ok := approver.Attributes()["type"].(types.String)
		if !ok || approverType.IsUnknown() || approverType.ValueString() != approverTypeUser {
			return 0, false
		}
		users++
	}
	return users, true
}

func (r *certManagerApplicationApprovalPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerApplicationApprovalPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan certRequestApprovalPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conditions, steps, diags := certRequestApprovalPolicyToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := infisical.CreateCertRequestApprovalPolicyRequest{
		Scope:                      approvalPolicyScopePkiApplication,
		ScopeID:                    plan.ApplicationID.ValueString(),
		Name:                       plan.Name.ValueString(),
		Conditions:                 conditions,
		Steps:                      steps,
		BypassForMachineIdentities: plan.BypassForMachineIdentities.ValueBool(),
		MaxRequestTTL:              plan.MaxRequestTTL.ValueStringPointer(),
	}

	created, err := r.client.CreateCertRequestApprovalPolicy(createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating certificate request approval policy",
			"Couldn't create certificate request approval policy in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(created.Policy.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *certManagerApplicationApprovalPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state certRequestApprovalPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetCertRequestApprovalPolicy(infisical.GetCertRequestApprovalPolicyRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading certificate request approval policy",
			"Couldn't read certificate request approval policy with ID "+state.ID.ValueString()+" from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	if !isApplicationApprovalPolicy(policy.Policy) {
		resp.Diagnostics.AddError(
			"Unsupported certificate request approval policy",
			"Certificate request approval policy with ID "+state.ID.ValueString()+" is a legacy project level policy, not tied to an application. "+
				"This resource only manages application policies, so drop it from state with `terraform state rm` and manage it in Infisical, "+
				"or recreate it on an application.",
		)
		return
	}

	refreshed, diags := certRequestApprovalPolicyFromAPI(ctx, policy.Policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *certManagerApplicationApprovalPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state certRequestApprovalPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conditions, steps, diags := certRequestApprovalPolicyToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateCertRequestApprovalPolicy(infisical.UpdateCertRequestApprovalPolicyRequest{
		ID:                         state.ID.ValueString(),
		Name:                       plan.Name.ValueString(),
		Conditions:                 conditions,
		Steps:                      steps,
		BypassForMachineIdentities: plan.BypassForMachineIdentities.ValueBool(),
		MaxRequestTTL:              plan.MaxRequestTTL.ValueStringPointer(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Error updating certificate request approval policy",
				"Certificate request approval policy with ID "+state.ID.ValueString()+" no longer exists in Infisical, so there is nothing to update. "+
					"It was most likely deleted outside of Terraform. Apply again with refresh enabled and Terraform will notice it is gone and create it anew.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating certificate request approval policy",
			"Couldn't update certificate request approval policy in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *certManagerApplicationApprovalPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state certRequestApprovalPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCertRequestApprovalPolicy(infisical.DeleteCertRequestApprovalPolicyRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil && !errors.Is(err, infisical.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Error deleting certificate request approval policy",
			"Couldn't delete certificate request approval policy from Infisical, unexpected error: "+err.Error(),
		)
	}
}

func (r *certManagerApplicationApprovalPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.ParseUUID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected the ID of a certificate request approval policy, such as 3f091b42-1661-41a1-b647-98c6a45af047, got: %q", req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func isApplicationApprovalPolicy(policy infisical.CertRequestApprovalPolicy) bool {
	return policy.ScopeType != nil && *policy.ScopeType == approvalPolicyScopePkiApplication && policy.ScopeID != nil
}

func certRequestApprovalPolicyToAPI(ctx context.Context, model certRequestApprovalPolicyModel) ([]infisical.CertRequestApprovalPolicyCondition, []infisical.ApprovalPolicyStep, diag.Diagnostics) {
	var diags diag.Diagnostics

	profileNames := make([]string, 0, len(model.ProfileNames.Elements()))
	diags.Append(model.ProfileNames.ElementsAs(ctx, &profileNames, false)...)
	if diags.HasError() {
		return nil, nil, diags
	}
	slices.Sort(profileNames)

	steps := make([]infisical.ApprovalPolicyStep, 0, len(model.Steps))
	for _, step := range model.Steps {
		var approvers []approvalPolicyApproverModel
		diags.Append(step.Approvers.ElementsAs(ctx, &approvers, false)...)
		if diags.HasError() {
			return nil, nil, diags
		}

		apiApprovers := make([]infisical.ApprovalPolicyApprover, 0, len(approvers))
		for _, approver := range approvers {
			apiApprovers = append(apiApprovers, infisical.ApprovalPolicyApprover{
				Type: approver.Type.ValueString(),
				ID:   approver.ID.ValueString(),
			})
		}

		notifyApprovers := step.NotifyApprovers.ValueBool()
		steps = append(steps, infisical.ApprovalPolicyStep{
			Name:              infisicaltf.OptionalStringPointer(step.Name),
			RequiredApprovals: step.RequiredApprovals.ValueInt64(),
			NotifyApprovers:   &notifyApprovers,
			Approvers:         apiApprovers,
		})
	}

	return []infisical.CertRequestApprovalPolicyCondition{{ProfileNames: profileNames}}, steps, diags
}

func certRequestApprovalPolicyFromAPI(ctx context.Context, policy infisical.CertRequestApprovalPolicy) (certRequestApprovalPolicyModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := certRequestApprovalPolicyModel{
		ID:                         types.StringValue(policy.ID),
		Name:                       types.StringValue(policy.Name),
		ApplicationID:              types.StringPointerValue(policy.ScopeID),
		MaxRequestTTL:              types.StringPointerValue(policy.MaxRequestTTL),
		BypassForMachineIdentities: types.BoolValue(policy.BypassForMachineIdentities != nil && *policy.BypassForMachineIdentities),
	}

	profileNames := []string{}
	for _, condition := range policy.Conditions.Conditions {
		for _, name := range condition.ProfileNames {
			if !slices.Contains(profileNames, name) {
				profileNames = append(profileNames, name)
			}
		}
	}
	profileNameSet, setDiags := types.SetValueFrom(ctx, types.StringType, profileNames)
	diags.Append(setDiags...)
	if diags.HasError() {
		return model, diags
	}
	model.ProfileNames = profileNameSet

	model.Steps = make([]certRequestApprovalPolicyStepModel, 0, len(policy.Steps))
	for _, step := range policy.Steps {
		approvers := make([]approvalPolicyApproverModel, 0, len(step.Approvers))
		for _, approver := range step.Approvers {
			approvers = append(approvers, approvalPolicyApproverModel{
				Type: types.StringValue(approver.Type),
				ID:   types.StringValue(approver.ID),
			})
		}
		approverSet, approverDiags := types.SetValueFrom(ctx, approvalPolicyApproverObjectType, approvers)
		diags.Append(approverDiags...)
		if diags.HasError() {
			return model, diags
		}

		stepModel := certRequestApprovalPolicyStepModel{
			Name:              types.StringNull(),
			RequiredApprovals: types.Int64Value(step.RequiredApprovals),
			// Infisical does not notify on steps saved without notifyApprovers.
			NotifyApprovers: types.BoolValue(step.NotifyApprovers != nil && *step.NotifyApprovers),
			Approvers:       approverSet,
		}
		if step.Name != nil && *step.Name != "" {
			stepModel.Name = types.StringValue(*step.Name)
		}

		model.Steps = append(model.Steps, stepModel)
	}

	return model, diags
}
