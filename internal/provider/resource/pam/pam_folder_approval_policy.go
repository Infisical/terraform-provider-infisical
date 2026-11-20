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
	_ resource.Resource                   = &pamFolderApprovalPolicyResource{}
	_ resource.ResourceWithConfigure      = &pamFolderApprovalPolicyResource{}
	_ resource.ResourceWithImportState    = &pamFolderApprovalPolicyResource{}
	_ resource.ResourceWithValidateConfig = &pamFolderApprovalPolicyResource{}
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
		Description: "Create and manage the approval policy of a PAM folder. Accounts in the folder whose template sets `requires_approval` need an approved request before a session launches or credentials are revealed. A folder has at most one policy. Auto-approval rules aren't managed by this resource: updates keep the rules added in Infisical, and destroying the policy deletes them. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the folder, which identifies the policy since a folder has at most one.",
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
				Description:  "The users and groups that can break glass: grant themselves access without an approver, on accounts whose template sets `allow_break_glass`. Each must be a member of the folder, and there can be at most 20. Defaults to none.",
				Optional:     true,
				Computed:     true,
				Default:      setdefault.StaticValue(types.SetValueMust(pamApproverType, []attr.Value{})),
				Validators:   []validator.Set{setvalidator.SizeAtMost(20)},
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

func (r *pamFolderApprovalPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	accessDuration := path.Root("constraints").AtName("access_duration")
	var minimum, maximum types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, accessDuration.AtName("min"), &minimum)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, accessDuration.AtName("max"), &maximum)...)
	if resp.Diagnostics.HasError() || minimum.IsUnknown() || maximum.IsUnknown() {
		return
	}

	minimumValue, maximumValue := pamDefaultMinAccessDuration, pamDefaultMaxAccessDuration
	if !minimum.IsNull() {
		minimumValue = minimum.ValueString()
	}
	if !maximum.IsNull() {
		maximumValue = maximum.ValueString()
	}
	resp.Diagnostics.Append(checkPamAccessDurationOrder(minimumValue, maximumValue)...)
}

// checkPamAccessDurationOrder rejects a minimum longer than the maximum. The API checks each bound
// alone, and an inverted range rejects every access request in the folder.
func checkPamAccessDurationOrder(minimum, maximum string) diag.Diagnostics {
	var diags diag.Diagnostics
	parsedMinimum, minimumOK := validators.ParseMsDuration(minimum)
	parsedMaximum, maximumOK := validators.ParseMsDuration(maximum)
	if minimumOK && maximumOK && parsedMinimum > parsedMaximum {
		diags.AddAttributeError(path.Root("constraints").AtName("access_duration").AtName("min"), "Invalid access duration",
			fmt.Sprintf("The minimum access duration %q is longer than the maximum %q, so no request could ever be approved.", minimum, maximum))
	}
	return diags
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

// pamApproversFromAPI drops duplicates, which the API stores and a Terraform set can't hold.
func pamApproversFromAPI(ctx context.Context, approvers []infisical.ApprovalPolicyApprover) (types.Set, diag.Diagnostics) {
	models := make([]pamApproverModel, 0, len(approvers))
	seen := map[infisical.ApprovalPolicyApprover]bool{}
	for _, approver := range approvers {
		if seen[approver] {
			continue
		}
		seen[approver] = true
		models = append(models, pamApproverModel{Type: types.StringValue(approver.Type), ID: types.StringValue(approver.ID)})
	}
	return types.SetValueFrom(ctx, pamApproverType, models)
}

// isPamSingleApprovalStep reports whether steps is the one step this resource writes, which needs one
// approval and notifies its approvers.
func isPamSingleApprovalStep(steps []infisical.ApprovalPolicyStep) bool {
	return len(steps) == 1 &&
		steps[0].RequiredApprovals == 1 &&
		steps[0].NotifyApprovers != nil &&
		*steps[0].NotifyApprovers
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
	// Checked again here, since ignore_changes can pair a configured bound with a stored one.
	diags.Append(checkPamAccessDurationOrder(constraints.AccessDuration.Min.ValueString(), constraints.AccessDuration.Max.ValueString())...)

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
		ID:       stringOrNull(policy.ScopeID),
		FolderID: stringOrNull(policy.ScopeID),
		Name:     types.StringValue(policy.Name),
	}

	// A policy of any other shape reads as having no approvers, so the plan shows the next apply rewriting it.
	var approvers []infisical.ApprovalPolicyApprover
	if isPamSingleApprovalStep(policy.Steps) {
		approvers = policy.Steps[0].Approvers
	}
	var approverDiags, bypasserDiags diag.Diagnostics
	model.Approvers, approverDiags = pamApproversFromAPI(ctx, approvers)
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

// findFolderPolicy returns the folder's approval policy, or nil when it has none. It returns ErrNotFound
// when the folder is gone.
func (r *pamFolderApprovalPolicyResource) findFolderPolicy(folderID string) (*infisical.PamAccessApprovalPolicy, error) {
	policies, err := r.client.ListPamAccessApprovalPolicies(infisical.ListPamAccessApprovalPoliciesRequest{FolderID: folderID})
	if err != nil || len(policies) == 0 {
		return nil, err
	}
	return &policies[0], nil
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

	existing, err := r.findFolderPolicy(plan.FolderID.ValueString())
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError("Error creating PAM folder approval policy", "Couldn't check the folder for an approval policy, unexpected error: "+err.Error())
		return
	}
	if existing != nil {
		resp.Diagnostics.AddError("Error creating PAM folder approval policy",
			fmt.Sprintf("Folder %s already has an approval policy, and a folder can only have one. Import it with its folder ID to manage it with Terraform.", plan.FolderID.ValueString()))
		return
	}

	_, err = r.client.CreatePamAccessApprovalPolicy(infisical.CreatePamAccessApprovalPolicyRequest{
		Scope:       pamApprovalPolicyScopeFolder,
		ScopeID:     plan.FolderID.ValueString(),
		Name:        plan.Name.ValueString(),
		Constraints: constraints,
		Steps:       steps,
		Bypassers:   bypassers,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating PAM folder approval policy", "Couldn't create PAM folder approval policy, unexpected error: "+err.Error())
		return
	}

	plan.ID = plan.FolderID
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

	policy, err := r.findFolderPolicy(state.ID.ValueString())
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError("Error reading PAM folder approval policy", "Couldn't read PAM folder approval policy, unexpected error: "+err.Error())
		return
	}
	if policy == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if !isPamSingleApprovalStep(policy.Steps) {
		resp.Diagnostics.AddWarning(
			"PAM folder approval policy changed outside of Terraform",
			fmt.Sprintf("The approval policy of folder %s doesn't have the single step this resource manages, where one approval from any approver grants access and every approver is notified, so the next apply rewrites it that way.", state.ID.ValueString()),
		)
	}

	refreshed, diags := pamApprovalPolicyFromAPI(ctx, *policy)
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

	policy, err := r.findFolderPolicy(state.ID.ValueString())
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError("Error updating PAM folder approval policy", "Couldn't read PAM folder approval policy, unexpected error: "+err.Error())
		return
	}
	if policy == nil {
		resp.Diagnostics.AddError("Error updating PAM folder approval policy",
			fmt.Sprintf("Folder %s no longer has an approval policy, so there is nothing to update. Apply again with refresh enabled and Terraform will create it.", state.ID.ValueString()))
		return
	}

	_, err = r.client.UpdatePamAccessApprovalPolicy(infisical.UpdatePamAccessApprovalPolicyRequest{
		ID:          policy.ID,
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

	policy, err := r.findFolderPolicy(state.ID.ValueString())
	if err == nil && policy != nil {
		err = r.client.DeletePamAccessApprovalPolicy(infisical.DeletePamAccessApprovalPolicyRequest{ID: policy.ID})
	}
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError("Error deleting PAM folder approval policy", "Couldn't delete PAM folder approval policy, unexpected error: "+err.Error())
	}
}

func (r *pamFolderApprovalPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.ParseUUID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected the ID of the PAM folder whose approval policy to import, such as 3f091b42-1661-41a1-b647-98c6a45af047, got: %q", req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
