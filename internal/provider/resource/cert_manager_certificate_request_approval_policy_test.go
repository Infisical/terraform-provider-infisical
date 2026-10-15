package resource

import (
	"context"
	"slices"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func certRequestApprovalPolicySchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &resource.SchemaResponse{}
	(&certManagerCertificateRequestApprovalPolicyResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", resp.Diagnostics)
	}
	return resp.Schema
}

func certRequestApprovalPolicyRaw(t *testing.T, s schema.Schema, model certRequestApprovalPolicyModel) tfsdk.Plan {
	t.Helper()

	var value attr.Value
	if diags := tfsdk.ValueFrom(context.Background(), model, s.Type(), &value); diags.HasError() {
		t.Fatalf("converting the model to its schema type: %v", diags)
	}
	raw, err := value.ToTerraformValue(context.Background())
	if err != nil {
		t.Fatalf("converting the model to a raw value: %v", err)
	}
	return tfsdk.Plan{Raw: raw, Schema: s}
}

func approverSet(t *testing.T, approvers ...approvalPolicyApproverModel) types.Set {
	t.Helper()

	set, diags := types.SetValueFrom(context.Background(), approvalPolicyApproverObjectType, approvers)
	if diags.HasError() {
		t.Fatalf("building the approvers set: %v", diags)
	}
	return set
}

func approver(approverType, id string) approvalPolicyApproverModel {
	return approvalPolicyApproverModel{Type: types.StringValue(approverType), ID: types.StringValue(id)}
}

func certRequestApprovalPolicyModelFor(t *testing.T, applicationID types.String, steps ...certRequestApprovalPolicyStepModel) certRequestApprovalPolicyModel {
	t.Helper()

	profileNames, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"web", "api"})
	if diags.HasError() {
		t.Fatalf("building profile names: %v", diags)
	}

	return certRequestApprovalPolicyModel{
		ID:                         types.StringValue("policy-1"),
		Name:                       types.StringValue("Web approvals"),
		ApplicationID:              applicationID,
		ProfileNames:               profileNames,
		BypassForMachineIdentities: types.BoolValue(false),
		Steps:                      steps,
	}
}

func TestCertRequestApprovalPolicyModelMatchesSchema(t *testing.T) {
	s := certRequestApprovalPolicySchema(t)
	model := certRequestApprovalPolicyModelFor(t, types.StringValue("app-1"), certRequestApprovalPolicyStepModel{
		Name:              types.StringNull(),
		RequiredApprovals: types.Int64Value(1),
		NotifyApprovers:   types.BoolValue(true),
		Approvers:         approverSet(t, approver("user", "user-1")),
	})

	plan := certRequestApprovalPolicyRaw(t, s, model)
	var roundTripped certRequestApprovalPolicyModel
	if diags := plan.Get(context.Background(), &roundTripped); diags.HasError() {
		t.Fatalf("reading the model back: %v", diags)
	}
	if len(roundTripped.Steps) != 1 || roundTripped.ApplicationID.ValueString() != "app-1" {
		t.Errorf("model did not survive the round trip: %+v", roundTripped)
	}
}

func TestCertRequestApprovalPolicyToAPI(t *testing.T) {
	model := certRequestApprovalPolicyModelFor(t, types.StringValue("app-1"),
		certRequestApprovalPolicyStepModel{
			Name:              types.StringValue("security"),
			RequiredApprovals: types.Int64Value(2),
			NotifyApprovers:   types.BoolValue(false),
			Approvers:         approverSet(t, approver("user", "user-1"), approver("group", "group-1")),
		},
		certRequestApprovalPolicyStepModel{
			Name:              types.StringNull(),
			RequiredApprovals: types.Int64Value(1),
			NotifyApprovers:   types.BoolValue(true),
			Approvers:         approverSet(t, approver("user", "user-2")),
		},
	)

	conditions, steps, diags := certRequestApprovalPolicyToAPI(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("certRequestApprovalPolicyToAPI() diagnostics = %v", diags)
	}

	if len(conditions) != 1 || !slices.Equal(conditions[0].ProfileNames, []string{"api", "web"}) {
		t.Errorf("conditions = %+v, want one condition listing api and web in order", conditions)
	}

	if len(steps) != 2 {
		t.Fatalf("steps = %+v, want two in the configured order", steps)
	}
	if steps[0].Name == nil || *steps[0].Name != "security" || steps[0].RequiredApprovals != 2 || len(steps[0].Approvers) != 2 {
		t.Errorf("first step = %+v, want the security step with two approvers", steps[0])
	}
	if steps[0].NotifyApprovers == nil || *steps[0].NotifyApprovers {
		t.Errorf("first step notifyApprovers = %v, want an explicit false", steps[0].NotifyApprovers)
	}
	if steps[1].Name != nil {
		t.Errorf("unnamed step was given name %q", *steps[1].Name)
	}
}

func TestCertRequestApprovalPolicyFromAPI(t *testing.T) {
	scopeType, scopeID, stepName, notify, bypass, ttl := "pki-application", "app-1", "security", true, true, "7d"

	policy := infisical.CertRequestApprovalPolicy{
		ID:                         "policy-1",
		Name:                       "Web approvals",
		BypassForMachineIdentities: &bypass,
		MaxRequestTTL:              &ttl,
		ScopeType:                  &scopeType,
		ScopeID:                    &scopeID,
		Steps: []infisical.ApprovalPolicyStep{
			{Name: &stepName, RequiredApprovals: 1, NotifyApprovers: &notify, Approvers: []infisical.ApprovalPolicyApprover{{Type: "user", ID: "user-1"}}},
			{RequiredApprovals: 1, Approvers: []infisical.ApprovalPolicyApprover{{Type: "group", ID: "group-1"}}},
		},
	}
	policy.Conditions.Conditions = []infisical.CertRequestApprovalPolicyCondition{
		{ProfileNames: []string{"web"}},
		{ProfileNames: []string{"api", "web"}},
	}

	model, diags := certRequestApprovalPolicyFromAPI(context.Background(), policy)
	if diags.HasError() {
		t.Fatalf("certRequestApprovalPolicyFromAPI() diagnostics = %v", diags)
	}

	if model.ApplicationID.ValueString() != "app-1" {
		t.Errorf("application_id = %v, want app-1", model.ApplicationID)
	}
	if !model.BypassForMachineIdentities.ValueBool() {
		t.Error("bypass_for_machine_identities = false, want true")
	}
	if model.MaxRequestTTL.ValueString() != "7d" {
		t.Errorf("max_request_ttl = %v, want 7d as Infisical stored it", model.MaxRequestTTL)
	}

	var profileNames []string
	model.ProfileNames.ElementsAs(context.Background(), &profileNames, false)
	slices.Sort(profileNames)
	if !slices.Equal(profileNames, []string{"api", "web"}) {
		t.Errorf("profile_names = %v, want every profile the conditions name, once", profileNames)
	}

	if len(model.Steps) != 2 {
		t.Fatalf("steps = %+v, want two", model.Steps)
	}
	if model.Steps[0].Name.ValueString() != "security" || !model.Steps[0].NotifyApprovers.ValueBool() {
		t.Errorf("first step = %+v, want the named, notifying step", model.Steps[0])
	}
	if !model.Steps[1].Name.IsNull() {
		t.Errorf("unnamed step name = %v, want null", model.Steps[1].Name)
	}
	if model.Steps[1].NotifyApprovers.ValueBool() {
		t.Error("step without notifyApprovers read as notifying")
	}
}

func TestIsApplicationApprovalPolicy(t *testing.T) {
	scopeType, scopeID := "pki-application", "app-1"

	if !isApplicationApprovalPolicy(infisical.CertRequestApprovalPolicy{ScopeType: &scopeType, ScopeID: &scopeID}) {
		t.Error("an application policy was not recognized")
	}
	if isApplicationApprovalPolicy(infisical.CertRequestApprovalPolicy{}) {
		t.Error("a legacy project level policy was taken for an application policy")
	}
}

func TestCertRequestApprovalPolicyFromAPIDefaults(t *testing.T) {
	scopeType, scopeID := "pki-application", "app-1"
	model, diags := certRequestApprovalPolicyFromAPI(context.Background(), infisical.CertRequestApprovalPolicy{
		ID:        "policy-1",
		Name:      "Web approvals",
		ScopeType: &scopeType,
		ScopeID:   &scopeID,
	})
	if diags.HasError() {
		t.Fatalf("certRequestApprovalPolicyFromAPI() diagnostics = %v", diags)
	}

	if model.BypassForMachineIdentities.ValueBool() {
		t.Error("bypass_for_machine_identities = true, want false when Infisical leaves it out")
	}
	if model.ProfileNames.IsNull() {
		t.Error("profile_names = null, want an empty set so the plan shows what is missing")
	}
	if !model.MaxRequestTTL.IsNull() {
		t.Errorf("max_request_ttl = %v, want null for requests that never expire", model.MaxRequestTTL)
	}
}

func TestCertRequestApprovalPolicyValidateConfig(t *testing.T) {
	s := certRequestApprovalPolicySchema(t)

	cases := map[string]struct {
		step      certRequestApprovalPolicyStepModel
		wantError bool
	}{
		"enough users": {
			step: certRequestApprovalPolicyStepModel{RequiredApprovals: types.Int64Value(2), Approvers: approverSet(t, approver("user", "user-1"), approver("user", "user-2"))},
		},
		"more approvals than users": {
			step:      certRequestApprovalPolicyStepModel{RequiredApprovals: types.Int64Value(3), Approvers: approverSet(t, approver("user", "user-1"), approver("user", "user-2"))},
			wantError: true,
		},
		"group approver": {
			step: certRequestApprovalPolicyStepModel{RequiredApprovals: types.Int64Value(5), Approvers: approverSet(t, approver("user", "user-1"), approver("group", "group-1"))},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			c.step.Name = types.StringNull()
			c.step.NotifyApprovers = types.BoolValue(true)
			plan := certRequestApprovalPolicyRaw(t, s, certRequestApprovalPolicyModelFor(t, types.StringValue("app-1"), c.step))

			resp := &resource.ValidateConfigResponse{}
			(&certManagerCertificateRequestApprovalPolicyResource{}).ValidateConfig(context.Background(), resource.ValidateConfigRequest{
				Config: tfsdk.Config{Raw: plan.Raw, Schema: s},
			}, resp)

			if resp.Diagnostics.HasError() != c.wantError {
				t.Errorf("ValidateConfig() error = %v, want %v: %v", resp.Diagnostics.HasError(), c.wantError, resp.Diagnostics)
			}
		})
	}
}

func TestCertRequestApprovalPolicyValidateConfigWithUnknownValues(t *testing.T) {
	ctx := context.Background()
	s := certRequestApprovalPolicySchema(t)
	listType, ok := s.Attributes["steps"].GetType().(types.ListType)
	if !ok {
		t.Fatalf("steps is %T, want a list", s.Attributes["steps"].GetType())
	}
	stepType, ok := listType.ElemType.(types.ObjectType)
	if !ok {
		t.Fatalf("a step is %T, want an object", listType.ElemType)
	}

	base := certRequestApprovalPolicyModelFor(t, types.StringValue("app-1"), certRequestApprovalPolicyStepModel{
		Name:              types.StringNull(),
		RequiredApprovals: types.Int64Value(1),
		NotifyApprovers:   types.BoolValue(true),
		Approvers:         approverSet(t, approver("user", "user-1")),
	})

	unknownApproverType := types.SetValueMust(approvalPolicyApproverObjectType, []attr.Value{
		types.ObjectValueMust(approvalPolicyApproverObjectType.AttrTypes, map[string]attr.Value{"type": types.StringUnknown(), "id": types.StringValue("user-1")}),
	})

	cases := map[string]attr.Value{
		"unknown steps": types.ListUnknown(stepType),
		"unknown step":  types.ListValueMust(stepType, []attr.Value{types.ObjectUnknown(stepType.AttrTypes)}),
		"unknown approver": types.ListValueMust(stepType, []attr.Value{types.ObjectValueMust(stepType.AttrTypes, map[string]attr.Value{
			"name":               types.StringNull(),
			"required_approvals": types.Int64Value(3),
			"notify_approvers":   types.BoolValue(true),
			"approvers":          types.SetValueMust(approvalPolicyApproverObjectType, []attr.Value{types.ObjectUnknown(approvalPolicyApproverObjectType.AttrTypes)}),
		})}),
		"unknown approver type": types.ListValueMust(stepType, []attr.Value{types.ObjectValueMust(stepType.AttrTypes, map[string]attr.Value{
			"name":               types.StringNull(),
			"required_approvals": types.Int64Value(3),
			"notify_approvers":   types.BoolValue(true),
			"approvers":          unknownApproverType,
		})}),
	}

	for name, steps := range cases {
		t.Run(name, func(t *testing.T) {
			plan := certRequestApprovalPolicyRaw(t, s, base)
			if diags := plan.SetAttribute(ctx, path.Root("steps"), steps); diags.HasError() {
				t.Fatalf("setting steps: %v", diags)
			}

			resp := &resource.ValidateConfigResponse{}
			(&certManagerCertificateRequestApprovalPolicyResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{
				Config: tfsdk.Config{Raw: plan.Raw, Schema: s},
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("ValidateConfig() diagnostics = %v, want none while values are unknown", resp.Diagnostics)
			}
		})
	}
}

func TestCertRequestApprovalPolicyImportStateRejectsNonIDs(t *testing.T) {
	ctx := context.Background()
	s := certRequestApprovalPolicySchema(t)

	for id, wantError := range map[string]bool{
		"3f091b42-1661-41a1-b647-98c6a45af047": false,
		"../../admin":                          true,
		"not-a-uuid":                           true,
		"":                                     true,
	} {
		resp := &resource.ImportStateResponse{State: tfsdk.State{Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil), Schema: s}}
		(&certManagerCertificateRequestApprovalPolicyResource{}).ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
		if resp.Diagnostics.HasError() != wantError {
			t.Errorf("ImportState(%q) error = %v, want %v: %v", id, resp.Diagnostics.HasError(), wantError, resp.Diagnostics)
		}
	}
}

func TestMaxRequestTTLValidator(t *testing.T) {
	cases := map[string]bool{
		"1h":       true,
		"7d":       true,
		"48h":      true,
		"30d":      true,
		"2 days":   true,
		"1.5 Days": true,
		"4w":       true,
		"3600000":  true,
		"59m":      false,
		"31d":      false,
		"5w":       false,
		"1y":       false,
		"abc":      false,
		"7 dayz":   false,
		"":         false,
	}

	for value, wantValid := range cases {
		resp := &validator.StringResponse{}
		maxRequestTTLValidator{}.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("max_request_ttl"),
			ConfigValue: types.StringValue(value),
		}, resp)
		if resp.Diagnostics.HasError() == wantValid {
			t.Errorf("max_request_ttl %q: valid = %v, want %v", value, !resp.Diagnostics.HasError(), wantValid)
		}
	}
}
