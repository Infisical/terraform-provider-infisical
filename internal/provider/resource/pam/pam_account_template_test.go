package resource

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func pamObject(t *testing.T, attrTypes map[string]attr.Type, values map[string]attr.Value) types.Object {
	t.Helper()
	object, diags := types.ObjectValue(attrTypes, values)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return object
}

func pamStringList(values ...string) types.List {
	elements := make([]attr.Value, len(values))
	for i, value := range values {
		elements[i] = types.StringValue(value)
	}
	return types.ListValueMust(types.StringType, elements)
}

// pamTemplateConfig mimics a configuration: unset attributes are null.
func pamTemplateConfig(accountType string, policies, settings types.Object) pamAccountTemplateResourceModel {
	return pamAccountTemplateResourceModel{
		ID:            types.StringUnknown(),
		Name:          types.StringValue("default-" + accountType),
		Description:   types.StringNull(),
		AccountType:   types.StringValue(accountType),
		GatewayID:     types.StringValue("33333333-3333-3333-3333-333333333333"),
		GatewayPoolID: types.StringNull(),
		Policies:      policies,
		Settings:      settings,
	}
}

func pamTemplateSettings(t *testing.T, values map[string]attr.Value) types.Object {
	t.Helper()
	all := map[string]attr.Value{
		"recording":             types.ObjectNull(pamTemplateRecordingAttrTypes),
		"session_log_masking":   types.ObjectNull(pamTemplateSessionLogMaskingAttrTypes),
		"heartbeat":             types.ObjectNull(pamTemplateScheduleAttrTypes),
		"rotation":              types.ObjectNull(pamTemplateScheduleAttrTypes),
		"password_requirements": types.ObjectNull(pamTemplatePasswordRequirementsAttrTypes),
	}
	for name, value := range values {
		all[name] = value
	}
	return pamObject(t, pamTemplateSettingsAttrTypes, all)
}

func pamTemplatePolicies(t *testing.T, values map[string]attr.Value) types.Object {
	t.Helper()
	all := map[string]attr.Value{
		"requires_approval":    types.BoolNull(),
		"allow_auto_approval":  types.BoolNull(),
		"allow_break_glass":    types.BoolNull(),
		"require_mfa":          types.BoolNull(),
		"require_reason":       types.BoolNull(),
		"max_session_duration": types.Int64Null(),
		"command_blocking":     types.ListNull(types.StringType),
	}
	for name, value := range values {
		all[name] = value
	}
	return pamObject(t, pamTemplatePoliciesAttrTypes, all)
}

// planPamTemplate runs ValidateConfig and ModifyPlan.
func planPamTemplate(t *testing.T, r *pamAccountTemplateResource, config pamAccountTemplateResourceModel) (tfsdk.Plan, diag.Diagnostics) {
	t.Helper()
	ctx := context.Background()
	plan := tfsdk.Plan{Schema: pamResourceSchema(t, r)}
	if diags := plan.Set(ctx, &config); diags.HasError() {
		t.Fatal(diags)
	}
	tfConfig := tfsdk.Config(plan)

	var validateResp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfConfig}, &validateResp)
	if validateResp.Diagnostics.HasError() {
		return plan, validateResp.Diagnostics
	}

	modifyResp := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: tfConfig, Plan: plan}, &modifyResp)
	return modifyResp.Plan, modifyResp.Diagnostics
}

func mustPlanPamTemplate(t *testing.T, r *pamAccountTemplateResource, config pamAccountTemplateResourceModel) tfsdk.Plan {
	t.Helper()
	plan, diags := planPamTemplate(t, r, config)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func pamTemplateModel(t *testing.T, state tfsdk.State) pamAccountTemplateResourceModel {
	t.Helper()
	var model pamAccountTemplateResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

// Omitted settings plan as the API's defaults, including the type-dependent built-in masking.
func TestPamAccountTemplatePlansDefaultSettings(t *testing.T) {
	for accountType, wantMasking := range map[string]bool{"ssh": true, "postgres": true, "windows-ad": false, "aws-iam": false} {
		t.Run(accountType, func(t *testing.T) {
			r := &pamAccountTemplateResource{}
			plan := mustPlanPamTemplate(t, r, pamTemplateConfig(accountType, types.ObjectNull(pamTemplatePoliciesAttrTypes), types.ObjectNull(pamTemplateSettingsAttrTypes)))
			settings := pamTemplateModel(t, tfsdk.State(plan)).Settings.Attributes()

			recording := objectAttributes(t, settings["recording"])
			if !boolAttribute(t, recording, "enabled") || stringAttribute(t, recording, "storage_backend") != "postgres" {
				t.Errorf("expected recording to default to on, in postgres, got %v", recording)
			}
			masking := objectAttributes(t, settings["session_log_masking"])
			if got := boolAttribute(t, masking, "built_in_detection"); got != wantMasking {
				t.Errorf("expected built-in masking to default to %v, got %v", wantMasking, got)
			}
			heartbeat := objectAttributes(t, settings["heartbeat"])
			if !boolAttribute(t, heartbeat, "enabled") || int64Attribute(t, heartbeat, "interval_seconds") != 86400 {
				t.Errorf("expected a daily heartbeat by default, got %v", heartbeat)
			}
			if !settings["rotation"].IsNull() || !settings["password_requirements"].IsNull() {
				t.Error("expected rotation settings to stay unset")
			}
		})
	}
}

func TestPamAccountTemplateRoundTrip(t *testing.T) {
	backend := newFakePamBackend(t)
	r := &pamAccountTemplateResource{}
	configurePamResource(t, r, newPamTestClient(t, backend))

	policies := pamTemplatePolicies(t, map[string]attr.Value{
		"requires_approval":    types.BoolValue(true),
		"allow_break_glass":    types.BoolValue(true),
		"require_mfa":          types.BoolValue(false),
		"max_session_duration": types.Int64Value(3600),
		"command_blocking":     pamStringList("^rm -rf /", `^shutdown\b`),
	})
	settings := pamTemplateSettings(t, map[string]attr.Value{
		"recording": pamObject(t, pamTemplateRecordingAttrTypes, map[string]attr.Value{
			"enabled":         types.BoolNull(),
			"storage_backend": types.StringValue("aws-s3"),
			"connection_id":   types.StringValue("44444444-4444-4444-4444-444444444444"),
			"s3": pamObject(t, pamTemplateRecordingS3AttrTypes, map[string]attr.Value{
				"bucket":     types.StringValue("acme-pam-recordings"),
				"region":     types.StringValue("us-east-1"),
				"key_prefix": types.StringNull(),
			}),
		}),
		"session_log_masking": pamObject(t, pamTemplateSessionLogMaskingAttrTypes, map[string]attr.Value{
			"patterns":           pamStringList(`password=\S+`, `token=\S+`),
			"built_in_detection": types.BoolNull(),
		}),
		"heartbeat": pamObject(t, pamTemplateScheduleAttrTypes, map[string]attr.Value{
			"enabled":          types.BoolValue(false),
			"interval_seconds": types.Int64Null(),
		}),
	})

	plan := mustPlanPamTemplate(t, r, pamTemplateConfig("ssh", policies, settings))
	state := createPamResource(t, r, plan)
	created := pamTemplateModel(t, state)

	sent := backend.requestsTo("POST", "/api/v1/pam/account-templates")[0]
	if sent["recordingConnectionId"] != "44444444-4444-4444-4444-444444444444" {
		t.Errorf("expected the recording connection beside the settings, got %v", sent["recordingConnectionId"])
	}
	sentSettings := jsonObject(t, sent["settings"])
	if _, ok := sentSettings["recordingConnectionId"]; ok {
		t.Error("expected the recording connection to stay out of the settings")
	}
	if sentSettings["sessionLogMaskingPatterns"] != "password=\\S+\ntoken=\\S+" {
		t.Errorf("expected newline-joined masking patterns, got %q", sentSettings["sessionLogMaskingPatterns"])
	}
	if sentSettings["sessionLogMaskingBuiltInDetection"] != true || sentSettings["recordingEnabled"] != true {
		t.Errorf("expected the planned defaults to be sent, got %v", sentSettings)
	}
	sentPolicies := jsonObject(t, sent["policies"])
	if sentPolicies["command-blocking"] != "^rm -rf /\n^shutdown\\b" || sentPolicies["require-mfa"] != false || sentPolicies["max-session-duration"] != float64(3600) {
		t.Errorf("unexpected policies sent: %v", sentPolicies)
	}
	if _, ok := sentPolicies["allow-auto-approval"]; ok {
		t.Error("expected an unset policy to be left out")
	}

	refreshed := readPamResource(t, r, state)
	if !refreshed.Raw.Equal(state.Raw) {
		t.Errorf("expected no drift after a refresh.\ngot:  %v\nwant: %v", refreshed.Raw, state.Raw)
	}

	// Policies taken out of the configuration are cleared, and the settings are sent whole.
	updateConfig := pamTemplateConfig("ssh", types.ObjectNull(pamTemplatePoliciesAttrTypes), settings)
	updateConfig.ID = created.ID
	state = updatePamResource(t, r, mustPlanPamTemplate(t, r, updateConfig), refreshed)
	patch := backend.requestsTo("PATCH", "/api/v1/pam/account-templates/"+created.ID.ValueString())[0]
	if policies, ok := patch["policies"].(map[string]any); !ok || len(policies) != 0 {
		t.Errorf("expected the update to clear the policies, got %v", patch["policies"])
	}
	if _, ok := patch["settings"].(map[string]any)["heartbeat"]; !ok {
		t.Error("expected the update to send the whole settings")
	}
	if refreshed := readPamResource(t, r, state); !refreshed.Raw.Equal(state.Raw) {
		t.Errorf("expected no drift after the update.\ngot:  %v\nwant: %v", refreshed.Raw, state.Raw)
	}

	deletePamResource(t, r, state)
	if len(backend.templates) != 0 {
		t.Error("expected the template to be deleted")
	}
}

func TestPamAccountTemplateRotationSettings(t *testing.T) {
	backend := newFakePamBackend(t)
	r := &pamAccountTemplateResource{}
	configurePamResource(t, r, newPamTestClient(t, backend))

	settings := pamTemplateSettings(t, map[string]attr.Value{
		"rotation": pamObject(t, pamTemplateScheduleAttrTypes, map[string]attr.Value{
			"enabled":          types.BoolValue(true),
			"interval_seconds": types.Int64Value(604800),
		}),
		"password_requirements": pamObject(t, pamTemplatePasswordRequirementsAttrTypes, map[string]attr.Value{
			"length": types.Int64Value(32),
			"required": pamObject(t, pamTemplatePasswordRequiredAttrTypes, map[string]attr.Value{
				"digits":    types.Int64Value(2),
				"lowercase": types.Int64Null(),
				"uppercase": types.Int64Value(2),
				"symbols":   types.Int64Null(),
			}),
			"allowed_symbols": types.StringValue("-_."),
		}),
	})

	state := createPamResource(t, r, mustPlanPamTemplate(t, r, pamTemplateConfig("windows-ad", types.ObjectNull(pamTemplatePoliciesAttrTypes), settings)))

	sent := jsonObject(t, backend.requestsTo("POST", "/api/v1/pam/account-templates")[0]["settings"])
	required := jsonObject(t, jsonObject(t, sent["passwordRequirements"])["required"])
	if required["lowercase"] != float64(0) || required["digits"] != float64(2) {
		t.Errorf("expected unset character counts to be sent as 0, got %v", required)
	}
	if jsonObject(t, sent["rotation"])["intervalSeconds"] != float64(604800) {
		t.Errorf("unexpected rotation sent: %v", sent["rotation"])
	}
	if sent["sessionLogMaskingBuiltInDetection"] != false {
		t.Error("expected built-in masking to default to off for windows-ad")
	}

	if refreshed := readPamResource(t, r, state); !refreshed.Raw.Equal(state.Raw) {
		t.Errorf("expected no drift after a refresh.\ngot:  %v\nwant: %v", refreshed.Raw, state.Raw)
	}
}

// A template made in the dashboard imports without drift against a config that omits settings.
func TestPamAccountTemplateImportMatchesDefaults(t *testing.T) {
	backend := newFakePamBackend(t)
	backend.templates["t1"] = map[string]any{
		"id":   "t1",
		"name": "default-ssh",
		"type": "ssh",
		"settings": map[string]any{
			"heartbeat":                         map[string]any{"enabled": true, "intervalSeconds": 86400},
			"sessionLogMaskingBuiltInDetection": true,
		},
		"policies":  map[string]any{},
		"gatewayId": "33333333-3333-3333-3333-333333333333",
	}
	r := &pamAccountTemplateResource{}
	configurePamResource(t, r, newPamTestClient(t, backend))

	imported := pamTemplateModel(t, importPamResource(t, r, pamResourceSchema(t, r), "t1"))
	if !imported.Policies.IsNull() {
		t.Errorf("expected empty policies to import as null, got %v", imported.Policies)
	}

	config := pamTemplateConfig("ssh", types.ObjectNull(pamTemplatePoliciesAttrTypes), types.ObjectNull(pamTemplateSettingsAttrTypes))
	planned := pamTemplateModel(t, tfsdk.State(mustPlanPamTemplate(t, r, config)))
	if !planned.Settings.Equal(imported.Settings) {
		t.Errorf("expected the imported settings to match the defaults.\nimported: %v\nplanned:  %v", imported.Settings, planned.Settings)
	}
}

func TestPamAccountTemplateValidateConfig(t *testing.T) {
	tests := map[string]struct {
		accountType string
		policies    func(t *testing.T) types.Object
		settings    func(t *testing.T) types.Object
		wantError   string
	}{
		"command blocking on a type without it": {
			accountType: "postgres",
			policies: func(t *testing.T) types.Object {
				return pamTemplatePolicies(t, map[string]attr.Value{"command_blocking": pamStringList("^DROP")})
			},
			wantError: "Command blocking is only supported",
		},
		"session duration outside the aws-iam range": {
			accountType: "aws-iam",
			policies: func(t *testing.T) types.Object {
				return pamTemplatePolicies(t, map[string]attr.Value{"max_session_duration": types.Int64Value(86400)})
			},
			wantError: "900 to 3600",
		},
		"rotation on a type that can't rotate": {
			accountType: "ssh",
			settings: func(t *testing.T) types.Object {
				return pamTemplateSettings(t, map[string]attr.Value{"rotation": pamObject(t, pamTemplateScheduleAttrTypes, map[string]attr.Value{
					"enabled": types.BoolValue(true), "interval_seconds": types.Int64Null(),
				})})
			},
			wantError: "Credential rotation is only supported",
		},
		"s3 recording without a bucket": {
			accountType: "ssh",
			settings: func(t *testing.T) types.Object {
				return pamTemplateSettings(t, map[string]attr.Value{"recording": pamObject(t, pamTemplateRecordingAttrTypes, map[string]attr.Value{
					"enabled":         types.BoolNull(),
					"storage_backend": types.StringValue("aws-s3"),
					"connection_id":   types.StringValue("44444444-4444-4444-4444-444444444444"),
					"s3":              types.ObjectNull(pamTemplateRecordingS3AttrTypes),
				})})
			},
			wantError: "needs both `connection_id` and `s3`",
		},
		"required characters longer than the password": {
			accountType: "postgres",
			settings: func(t *testing.T) types.Object {
				return pamTemplateSettings(t, map[string]attr.Value{"password_requirements": pamObject(t, pamTemplatePasswordRequirementsAttrTypes, map[string]attr.Value{
					"length": types.Int64Value(4),
					"required": pamObject(t, pamTemplatePasswordRequiredAttrTypes, map[string]attr.Value{
						"digits": types.Int64Value(3), "lowercase": types.Int64Value(3), "uppercase": types.Int64Null(), "symbols": types.Int64Null(),
					}),
					"allowed_symbols": types.StringNull(),
				})})
			},
			wantError: "can't add up to more than",
		},
		"no required characters": {
			accountType: "postgres",
			settings: func(t *testing.T) types.Object {
				return pamTemplateSettings(t, map[string]attr.Value{"password_requirements": pamObject(t, pamTemplatePasswordRequirementsAttrTypes, map[string]attr.Value{
					"length": types.Int64Value(16),
					"required": pamObject(t, pamTemplatePasswordRequiredAttrTypes, map[string]attr.Value{
						"digits": types.Int64Null(), "lowercase": types.Int64Null(), "uppercase": types.Int64Null(), "symbols": types.Int64Null(),
					}),
					"allowed_symbols": types.StringNull(),
				})})
			},
			wantError: "At least one kind of character",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			policies, settings := types.ObjectNull(pamTemplatePoliciesAttrTypes), types.ObjectNull(pamTemplateSettingsAttrTypes)
			if test.policies != nil {
				policies = test.policies(t)
			}
			if test.settings != nil {
				settings = test.settings(t)
			}

			_, diags := planPamTemplate(t, &pamAccountTemplateResource{}, pamTemplateConfig(test.accountType, policies, settings))
			if !diags.HasError() {
				t.Fatalf("expected an error containing %q", test.wantError)
			}
			if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, test.wantError) {
				t.Errorf("expected an error containing %q, got %q", test.wantError, detail)
			}
		})
	}
}
