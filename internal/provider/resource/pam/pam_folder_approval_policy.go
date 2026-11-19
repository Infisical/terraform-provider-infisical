package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"
	"terraform-provider-infisical/internal/pkg/validators"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	pamApprovalPolicyScopeFolder = "pam-folder"

	pamApproverTypeUser  = "user"
	pamApproverTypeGroup = "group"

	// Infisical's own folder approval settings use these access duration bounds.
	pamDefaultMinAccessDuration = "30s"
	pamDefaultMaxAccessDuration = "7d"
)

var (
	_ resource.Resource                = &pamFolderApprovalPolicyResource{}
	_ resource.ResourceWithConfigure   = &pamFolderApprovalPolicyResource{}
	_ resource.ResourceWithImportState = &pamFolderApprovalPolicyResource{}
)

var pamApproverType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type": types.StringType,
	"id":   types.StringType,
}}

var pamAccessDurationAttrTypes = map[string]attr.Type{
	"min": types.StringType,
	"max": types.StringType,
}

var pamApprovalConstraintsAttrTypes = map[string]attr.Type{
	"access_duration": types.ObjectType{AttrTypes: pamAccessDurationAttrTypes},
}

func NewPamFolderApprovalPolicyResource() resource.Resource {
	return &pamFolderApprovalPolicyResource{}
}

type pamFolderApprovalPolicyResource struct {
	client *infisical.Client
}

type pamFolderApprovalPolicyModel struct {
	ID          types.String `tfsdk:"id"`
	FolderID    types.String `tfsdk:"folder_id"`
	Name        types.String `tfsdk:"name"`
	Approvers   types.Set    `tfsdk:"approvers"`
	Bypassers   types.Set    `tfsdk:"bypassers"`
	Constraints types.Object `tfsdk:"constraints"`
}

type pamApproverModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
}

type pamApprovalConstraintsModel struct {
	AccessDuration pamAccessDurationModel `tfsdk:"access_duration"`
}

type pamAccessDurationModel struct {
	Min types.String `tfsdk:"min"`
	Max types.String `tfsdk:"max"`
}

func pamApproverObject(kind string) schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Description: fmt.Sprintf("Whether the %s is a `%s` or a `%s`.", kind, pamApproverTypeUser, pamApproverTypeGroup),
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf(pamApproverTypeUser, pamApproverTypeGroup)},
			},
			"id": schema.StringAttribute{
				Description: fmt.Sprintf("The ID of the %s's user or group.", kind),
				Required:    true,
				Validators:  []validator.String{infisicaltf.UuidValidator},
			},
		},
	}
}

func pamAccessDurationAttribute(bound, defaultValue string) schema.StringAttribute {
	return schema.StringAttribute{
		Description: fmt.Sprintf("The %s access duration a request can ask for, such as `1h` or `2d`, from 30 seconds to 7 days. Defaults to `%s`.", bound, defaultValue),
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(defaultValue),
		Validators:  []validator.String{validators.MsDurationBetween(30*time.Second, 7*24*time.Hour)},
	}
}

func (r *pamFolderApprovalPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pam_folder_approval_policy"
}

func (r *pamFolderApprovalPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	defaultAccessDuration := types.ObjectValueMust(pamAccessDurationAttrTypes, map[string]attr.Value{
		"min": types.StringValue(pamDefaultMinAccessDuration),
		"max": types.StringValue(pamDefaultMaxAccessDuration),
	})

	resp.Schema = schema.Schema{
		Description: "Create and manage the approval policy of a PAM folder. Accounts in the folder whose template sets `requires_approval` need an approved request before a session launches or credentials are revealed. A folder has at most one policy. Auto-approval rules aren't managed by this resource, so rules added in Infisical are kept. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the approval policy.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"folder_id": schema.StringAttribute{
				Description:   "The ID of the folder the policy covers. Changing it creates a new policy.",
				Required:      true,
				Validators:    []validator.String{infisicaltf.UuidValidator},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the approval policy.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 128)},
			},
			"approvers": schema.SetNestedAttribute{
				Description:  "The users and groups that can approve a request. One approval from any of them grants it, and each is notified of new requests. Each must be a member of the folder.",
				Required:     true,
				Validators:   []validator.Set{setvalidator.SizeAtLeast(1)},
				NestedObject: pamApproverObject("approver"),
			},
			"bypassers": schema.SetNestedAttribute{
				Description:  "The users and groups that can break glass: grant themselves access without an approver, on accounts whose template sets `allow_break_glass`. Each must be a member of the folder. Defaults to none.",
				Optional:     true,
				Computed:     true,
				Default:      setdefault.StaticValue(types.SetValueMust(pamApproverType, []attr.Value{})),
				Validators:   []validator.Set{setvalidator.SizeAtMost(100)},
				NestedObject: pamApproverObject("bypasser"),
			},
			"constraints": schema.SingleNestedAttribute{
				Description: "Limits on the requests the policy approves.",
				Optional:    true,
				Computed:    true,
				Default: objectdefault.StaticValue(types.ObjectValueMust(pamApprovalConstraintsAttrTypes, map[string]attr.Value{
					"access_duration": defaultAccessDuration,
				})),
				Attributes: map[string]schema.Attribute{
					"access_duration": schema.SingleNestedAttribute{
						Description: "How long the access a request asks for may last.",
						Optional:    true,
						Computed:    true,
						Default:     objectdefault.StaticValue(defaultAccessDuration),
						Attributes: map[string]schema.Attribute{
							"min": pamAccessDurationAttribute("shortest", pamDefaultMinAccessDuration),
							"max": pamAccessDurationAttribute("longest", pamDefaultMaxAccessDuration),
						},
					},
				},
			},
		},
	}
}

func (r *pamFolderApprovalPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configurePamClient(req, resp); client != nil {
		r.client = client
	}
}

func pamApproversToAPI(ctx context.Context, set types.Set) ([]infisical.ApprovalPolicyApprover, diag.Diagnostics) {
	var approvers []pamApproverModel
	diags := set.ElementsAs(ctx, &approvers, false)
	result := make([]infisical.ApprovalPolicyApprover, 0, len(approvers))
	for _, approver := range approvers {
		result = append(result, infisical.ApprovalPolicyApprover{Type: approver.Type.ValueString(), ID: approver.ID.ValueString()})
	}
	return result, diags
}

func pamApproversFromAPI(ctx context.Context, approvers []infisical.ApprovalPolicyApprover) (types.Set, diag.Diagnostics) {
	models := make([]pamApproverModel, 0, len(approvers))
	for _, approver := range approvers {
		models = append(models, pamApproverModel{Type: types.StringValue(approver.Type), ID: types.StringValue(approver.ID)})
	}
	return types.SetValueFrom(ctx, pamApproverType, models)
}

func pamApprovalPolicyToAPI(ctx context.Context, plan pamFolderApprovalPolicyModel) ([]infisical.ApprovalPolicyStep, []infisical.ApprovalPolicyApprover, infisical.PamAccessApprovalPolicyConstraints, diag.Diagnostics) {
	var diags diag.Diagnostics

	approvers, approverDiags := pamApproversToAPI(ctx, plan.Approvers)
	diags.Append(approverDiags...)
	// Infisical's folder approval settings save one step that needs one approval, so the policy keeps that shape.
	notifyApprovers := true
	steps := []infisical.ApprovalPolicyStep{{RequiredApprovals: 1, NotifyApprovers: &notifyApprovers, Approvers: approvers}}

	bypassers, bypasserDiags := pamApproversToAPI(ctx, plan.Bypassers)
	diags.Append(bypasserDiags...)

	var constraints pamApprovalConstraintsModel
	diags.Append(plan.Constraints.As(ctx, &constraints, basetypes.ObjectAsOptions{})...)

	return steps, bypassers, infisical.PamAccessApprovalPolicyConstraints{
		AccessDuration: infisical.PamAccessApprovalPolicyAccessDuration{
			Min: constraints.AccessDuration.Min.ValueString(),
			Max: constraints.AccessDuration.Max.ValueString(),
		},
	}, diags
}

func pamApprovalPolicyFromAPI(ctx context.Context, policy infisical.PamAccessApprovalPolicy) (pamFolderApprovalPolicyModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := pamFolderApprovalPolicyModel{
		ID:       types.StringValue(policy.ID),
		FolderID: stringOrNull(policy.ScopeID),
		Name:     types.StringValue(policy.Name),
	}

	// Read like Infisical's folder approval settings, which show the first step's approvers.
	var firstStepApprovers []infisical.ApprovalPolicyApprover
	if len(policy.Steps) > 0 {
		firstStepApprovers = policy.Steps[0].Approvers
	}
	var approverDiags, bypasserDiags diag.Diagnostics
	model.Approvers, approverDiags = pamApproversFromAPI(ctx, firstStepApprovers)
	diags.Append(approverDiags...)
	model.Bypassers, bypasserDiags = pamApproversFromAPI(ctx, policy.Bypassers)
	diags.Append(bypasserDiags...)

	accessDuration := policy.Constraints.Constraints.AccessDuration
	constraints, constraintDiags := types.ObjectValueFrom(ctx, pamApprovalConstraintsAttrTypes, pamApprovalConstraintsModel{
		AccessDuration: pamAccessDurationModel{Min: types.StringValue(accessDuration.Min), Max: types.StringValue(accessDuration.Max)},
	})
	diags.Append(constraintDiags...)
	model.Constraints = constraints

	return model, diags
}

func (r *pamFolderApprovalPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to create PAM folder approval policy") {
		return
	}

	var plan pamFolderApprovalPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	steps, bypassers, constraints, diags := pamApprovalPolicyToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.CreatePamAccessApprovalPolicy(infisical.CreatePamAccessApprovalPolicyRequest{
		Scope:       pamApprovalPolicyScopeFolder,
		ScopeID:     plan.FolderID.ValueString(),
		Name:        plan.Name.ValueString(),
		Conditions:  []map[string]any{},
		Constraints: constraints,
		Steps:       steps,
		Bypassers:   bypassers,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating PAM folder approval policy", "Couldn't create PAM folder approval policy, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(policy.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pamFolderApprovalPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to read PAM folder approval policy") {
		return
	}

	var state pamFolderApprovalPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	policy, err := r.client.GetPamAccessApprovalPolicy(infisical.GetPamAccessApprovalPolicyRequest{ID: state.ID.ValueString()})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PAM folder approval policy", "Couldn't read PAM folder approval policy, unexpected error: "+err.Error())
		return
	}

	refreshed, diags := pamApprovalPolicyFromAPI(ctx, policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *pamFolderApprovalPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to update PAM folder approval policy") {
		return
	}

	var plan, state pamFolderApprovalPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	steps, bypassers, constraints, diags := pamApprovalPolicyToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdatePamAccessApprovalPolicy(infisical.UpdatePamAccessApprovalPolicyRequest{
		ID:          state.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		Constraints: constraints,
		Steps:       steps,
		Bypassers:   bypassers,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating PAM folder approval policy", "Couldn't update PAM folder approval policy, unexpected error: "+err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pamFolderApprovalPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to delete PAM folder approval policy") {
		return
	}

	var state pamFolderApprovalPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePamAccessApprovalPolicy(infisical.DeletePamAccessApprovalPolicyRequest{ID: state.ID.ValueString()})
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError("Error deleting PAM folder approval policy", "Couldn't delete PAM folder approval policy, unexpected error: "+err.Error())
	}
}

func (r *pamFolderApprovalPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.ParseUUID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected the ID of a PAM folder approval policy, such as 3f091b42-1661-41a1-b647-98c6a45af047, got: %q", req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
