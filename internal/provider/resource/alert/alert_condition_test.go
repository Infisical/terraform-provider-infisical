package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAlertEventTable(t *testing.T) {
	short := map[string]bool{}
	for _, event := range alertEvents {
		if _, ok := alertResourceTypes[event.resourceType]; !ok {
			t.Errorf("event %q is on resource type %q, which has no entry in alertResourceTypes", event.eventType, event.resourceType)
		}
		if event.shortEventType() == event.eventType {
			t.Errorf("event %q does not start with its resource type %q", event.eventType, event.resourceType)
		}
		key := event.resourceType + "/" + event.shortEventType()
		if short[key] {
			t.Errorf("two events of %q share the short form %q", event.resourceType, event.shortEventType())
		}
		short[key] = true
	}
}

func TestResolveAlertEvent(t *testing.T) {
	cases := []struct {
		resourceType, eventType, want string
	}{
		{alertResourceTypeCertManagerApplication, "certificate.expiry", alertEventTypeCertManagerApplicationCertExpiry},
		{alertResourceTypeCertManagerApplication, alertEventTypeCertManagerApplicationCertExpiry, alertEventTypeCertManagerApplicationCertExpiry},
		{alertResourceTypeCertManager, "certificate.issuance", alertEventTypeCertManagerCertIssuance},
		{alertResourceTypeCertManagerSigner, "certificate.expiry", alertEventTypeCertManagerSignerCertExpiry},
		{alertResourceTypeIdentityAuthentication, "auth-method-changed", alertEventTypeIdentityAuthenticationAuthMethodChanged},
		{alertResourceTypeIdentityAuthentication, "expiry", alertEventTypeIdentityAuthenticationExpiry},
	}
	for _, c := range cases {
		event, ok := resolveAlertEvent(c.resourceType, c.eventType)
		if !ok || event.eventType != c.want {
			t.Errorf("resolveAlertEvent(%q, %q) = %q, %v, want %q", c.resourceType, c.eventType, event.eventType, ok, c.want)
		}
	}

	for _, c := range []struct{ resourceType, eventType string }{
		{alertResourceTypeCertManagerApplication, alertEventTypeCertManagerCertExpiry},
		{alertResourceTypeCertManagerSigner, "certificate.issuance"},
		{alertResourceTypeIdentityAuthentication, "certificate.expiry"},
	} {
		if event, ok := resolveAlertEvent(c.resourceType, c.eventType); ok {
			t.Errorf("resolveAlertEvent(%q, %q) = %q, want no event", c.resourceType, c.eventType, event.eventType)
		}
	}
}

func TestAlertEventTypeSemanticEquality(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		a, b string
		want bool
	}{
		{"certificate.expiry", alertEventTypeCertManagerApplicationCertExpiry, true},
		{alertEventTypeCertManagerApplicationCertExpiry, "certificate.expiry", true},
		{"expiry", alertEventTypeIdentityAuthenticationExpiry, true},
		{"certificate.expiry", "certificate.expiry", true},
		{"certificate.expiry", "certificate.issuance", false},
		{alertEventTypeCertManagerApplicationCertExpiry, alertEventTypeCertManagerCertExpiry, false},
		{"expiry", "auth-method-changed", false},
	}
	for _, c := range cases {
		equal, diags := newAlertEventTypeValue(c.a).StringSemanticEquals(ctx, newAlertEventTypeValue(c.b))
		if diags.HasError() || equal != c.want {
			t.Errorf("%q vs %q: equal = %v (%v), want %v", c.a, c.b, equal, diags, c.want)
		}
	}
}

func stringSetOf(values ...string) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elements)
}

func emptyCondition() *alertConditionModel {
	return &alertConditionModel{
		AlertBeforeDays: types.Int64Null(),
		DailyReminder:   types.BoolNull(),
		ApplicationIDs:  types.SetNull(types.StringType),
		ProfileIDs:      types.SetNull(types.StringType),
		Sources:         types.SetNull(types.StringType),
		SignerIDs:       types.SetNull(types.StringType),
	}
}

func TestAlertConditionForAPI(t *testing.T) {
	expiryFiltered := emptyCondition()
	expiryFiltered.AlertBeforeDays = types.Int64Value(45)
	expiryFiltered.ApplicationIDs = stringSetOf("app-2", "app-1")
	expiryFiltered.Sources = stringSetOf("issued")

	issuanceFiltered := emptyCondition()
	issuanceFiltered.ProfileIDs = stringSetOf("profile-1")

	identityExpiry := emptyCondition()
	identityExpiry.AlertBeforeDays = types.Int64Value(30)
	identityExpiry.DailyReminder = types.BoolValue(true)

	signerExpiry := emptyCondition()
	signerExpiry.AlertBeforeDays = types.Int64Value(30)
	signerExpiry.SignerIDs = stringSetOf("signer-2", "signer-1")

	cases := map[string]struct {
		resourceType, eventType string
		condition               *alertConditionModel
		expiry                  *expiryConditionModel
		want                    string
	}{
		"expiry with filters": {alertResourceTypeCertManager, "certificate.expiry", expiryFiltered, nil,
			`{"alertBefore":"45d","dailyReminder":false,"applicationIds":["app-1","app-2"],"sources":["issued"]}`},
		"event with filters":    {alertResourceTypeCertManager, "certificate.issuance", issuanceFiltered, nil, `{"profileIds":["profile-1"]}`},
		"event without filters": {alertResourceTypeCertManagerApplication, "certificate.revocation", nil, nil, `null`},
		"identity expiry":       {alertResourceTypeIdentityAuthentication, "expiry", identityExpiry, nil, `{"alertBefore":"30d","dailyReminder":true}`},
		"signer expiry with signer_ids": {alertResourceTypeCertManagerSigner, "certificate.expiry", signerExpiry, nil,
			`{"alertBefore":"30d","dailyReminder":false,"signerIds":["signer-1","signer-2"]}`},
		"deprecated expiry block": {alertResourceTypeIdentityAuthentication, "", nil,
			&expiryConditionModel{AlertBeforeDays: types.Int64Value(7), DailyReminder: types.BoolValue(false)}, `{"alertBefore":"7d","dailyReminder":false}`},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			model := alertResourceModel{
				ResourceType: types.StringValue(c.resourceType),
				EventType:    newAlertEventTypeValue(c.eventType),
				Condition:    c.condition,
				Expiry:       c.expiry,
			}
			event, ok := alertEventFromModel(model)
			if !ok {
				t.Fatalf("alertEventFromModel() found no event")
			}
			raw, _ := json.Marshal(alertConditionForAPI(event, model))
			if string(raw) != c.want {
				t.Errorf("condition = %s, want %s", raw, c.want)
			}
		})
	}
}

func TestSetAlertConditionFromAPI(t *testing.T) {
	event := func(eventType string) alertEvent {
		e, ok := alertEventForEventType(eventType)
		if !ok {
			t.Fatalf("no event %q", eventType)
		}
		return e
	}

	t.Run("expiry saved in the UI as weeks", func(t *testing.T) {
		var state alertResourceModel
		diags := setAlertConditionFromAPI(&state, event(alertEventTypeCertManagerApplicationCertExpiry), json.RawMessage(`{"alertBefore":"2w","dailyReminder":true}`))
		if diags.HasError() || state.Condition == nil || state.Condition.AlertBeforeDays.ValueInt64() != 14 || !state.Condition.DailyReminder.ValueBool() {
			t.Errorf("condition = %+v (%v), want 14 days with a daily reminder", state.Condition, diags)
		}
	})

	t.Run("daily_reminder false reads as left out", func(t *testing.T) {
		var state alertResourceModel
		setAlertConditionFromAPI(&state, event(alertEventTypeIdentityAuthenticationExpiry), json.RawMessage(`{"alertBefore":"30d","dailyReminder":false}`))
		if state.Condition == nil || !state.Condition.DailyReminder.IsNull() {
			t.Errorf("daily_reminder = %v, want null", state.Condition)
		}
	})

	t.Run("daily_reminder false stays false when configured", func(t *testing.T) {
		state := alertResourceModel{Condition: &alertConditionModel{DailyReminder: types.BoolValue(false)}}
		setAlertConditionFromAPI(&state, event(alertEventTypeIdentityAuthenticationExpiry), json.RawMessage(`{"alertBefore":"30d","dailyReminder":false}`))
		if state.Condition == nil || state.Condition.DailyReminder.IsNull() || state.Condition.DailyReminder.ValueBool() {
			t.Errorf("daily_reminder = %v, want false", state.Condition)
		}
	})

	t.Run("filters", func(t *testing.T) {
		var state alertResourceModel
		setAlertConditionFromAPI(&state, event(alertEventTypeCertManagerCertIssuance), json.RawMessage(`{"sources":["imported"]}`))
		if state.Condition == nil || len(state.Condition.Sources.Elements()) != 1 || !state.Condition.ApplicationIDs.IsNull() || !state.Condition.AlertBeforeDays.IsNull() {
			t.Errorf("condition = %+v, want only the sources filter", state.Condition)
		}
	})

	t.Run("signer_ids", func(t *testing.T) {
		var state alertResourceModel
		setAlertConditionFromAPI(&state, event(alertEventTypeCertManagerSignerCertExpiry), json.RawMessage(`{"alertBefore":"30d","signerIds":["signer-1","signer-2"]}`))
		if state.Condition == nil || !state.Condition.SignerIDs.Equal(stringSetOf("signer-2", "signer-1")) || !state.Condition.ApplicationIDs.IsNull() {
			t.Errorf("condition = %+v, want the signer_ids filter", state.Condition)
		}
	})

	for _, raw := range []string{`null`, `{}`, ``} {
		t.Run("event without settings "+raw, func(t *testing.T) {
			state := alertResourceModel{Condition: emptyCondition()}
			setAlertConditionFromAPI(&state, event(alertEventTypeIdentityAuthenticationAuthMethodChanged), json.RawMessage(raw))
			if state.Condition != nil {
				t.Errorf("condition = %+v, want none", state.Condition)
			}
		})
	}

	t.Run("deprecated expiry block stays in use", func(t *testing.T) {
		state := alertResourceModel{Expiry: &expiryConditionModel{}, EventType: newAlertEventTypeValue(alertEventTypeIdentityAuthenticationExpiry)}
		setAlertConditionFromAPI(&state, event(alertEventTypeIdentityAuthenticationExpiry), json.RawMessage(`{"alertBefore":"1m"}`))
		if state.Expiry == nil || state.Expiry.AlertBeforeDays.ValueInt64() != 30 || state.Expiry.DailyReminder.ValueBool() || state.Condition != nil {
			t.Errorf("expiry = %+v, condition = %+v, want the expiry block with 30 days and no condition", state.Expiry, state.Condition)
		}
	})

	t.Run("expiry without a duration", func(t *testing.T) {
		var state alertResourceModel
		if diags := setAlertConditionFromAPI(&state, event(alertEventTypeCertManagerSignerCertExpiry), json.RawMessage(`null`)); !diags.HasError() {
			t.Error("want an error for an expiry alert without alertBefore")
		}
	})
}

func TestParseAlertBeforeDays(t *testing.T) {
	cases := map[string]struct {
		want      int64
		wantError bool
	}{
		"1d": {want: 1}, "30d": {want: 30}, "2w": {want: 14}, "1m": {want: 30}, "1y": {want: 365},
		"30": {wantError: true}, "2h": {wantError: true}, "-1d": {wantError: true}, "": {wantError: true}, "d": {wantError: true},
	}
	for alertBefore, c := range cases {
		got, diags := parseAlertBeforeDays(alertBefore)
		if diags.HasError() != c.wantError || (!c.wantError && got != c.want) {
			t.Errorf("parseAlertBeforeDays(%q) = %d (error %v), want %d (error %v)", alertBefore, got, diags.HasError(), c.want, c.wantError)
		}
	}
}

func TestDescribeAlertBeforeDaysLimits(t *testing.T) {
	want := "90 for identity.authentication; 365 for cert-manager, cert-manager.application, cert-manager.signer"
	if got := describeAlertBeforeDaysLimits(); got != want {
		t.Errorf("describeAlertBeforeDaysLimits() = %q, want %q", got, want)
	}
}

func TestValidateAlertEvent(t *testing.T) {
	ctx := context.Background()
	s := alertSchema(t)

	withDays := func(days int64) *alertConditionModel {
		c := emptyCondition()
		c.AlertBeforeDays = types.Int64Value(days)
		return c
	}
	withSources := func(c *alertConditionModel) *alertConditionModel {
		c.Sources = stringSetOf("issued")
		return c
	}
	withSigners := func(c *alertConditionModel) *alertConditionModel {
		c.SignerIDs = stringSetOf("signer-1")
		return c
	}
	reminderOnly := emptyCondition()
	reminderOnly.DailyReminder = types.BoolValue(true)
	nullSigner := withDays(30)
	nullSigner.SignerIDs = types.SetValueMust(types.StringType, []attr.Value{types.StringNull()})

	cases := map[string]struct {
		resourceType, eventType string
		condition               *alertConditionModel
		expiry                  *expiryConditionModel
		wantError               bool
	}{
		"application expiry, short form":        {alertResourceTypeCertManagerApplication, "certificate.expiry", withDays(30), nil, false},
		"application expiry, full form":         {alertResourceTypeCertManagerApplication, alertEventTypeCertManagerApplicationCertExpiry, withDays(365), nil, false},
		"application expiry past its limit":     {alertResourceTypeCertManagerApplication, "certificate.expiry", withDays(366), nil, true},
		"identity expiry past its limit":        {alertResourceTypeIdentityAuthentication, "expiry", withDays(91), nil, true},
		"expiry without alert_before_days":      {alertResourceTypeCertManagerSigner, "certificate.expiry", nil, nil, true},
		"expiry with only daily_reminder":       {alertResourceTypeCertManagerSigner, "certificate.expiry", reminderOnly, nil, true},
		"product-wide expiry with filters":      {alertResourceTypeCertManager, "certificate.expiry", withSources(withDays(30)), nil, false},
		"product-wide event with filters":       {alertResourceTypeCertManager, "certificate.renewal", withSources(emptyCondition()), nil, false},
		"product-wide event without condition":  {alertResourceTypeCertManager, "certificate.renewal", nil, nil, false},
		"application event without condition":   {alertResourceTypeCertManagerApplication, "certificate.issuance", nil, nil, false},
		"application event with filters":        {alertResourceTypeCertManagerApplication, "certificate.issuance", withSources(emptyCondition()), nil, true},
		"signer expiry with filters":            {alertResourceTypeCertManagerSigner, "certificate.expiry", withSources(withDays(30)), nil, true},
		"signer expiry with signer_ids":         {alertResourceTypeCertManagerSigner, "certificate.expiry", withSigners(withDays(30)), nil, false},
		"product-wide expiry with signer_ids":   {alertResourceTypeCertManager, "certificate.expiry", withSigners(withDays(30)), nil, true},
		"application expiry with signer_ids":    {alertResourceTypeCertManagerApplication, "certificate.expiry", withSigners(withDays(30)), nil, true},
		"signer expiry with a null signer_ids":  {alertResourceTypeCertManagerSigner, "certificate.expiry", nullSigner, nil, true},
		"event with alert_before_days":          {alertResourceTypeCertManager, "certificate.issuance", withDays(30), nil, true},
		"empty condition":                       {alertResourceTypeCertManager, "certificate.issuance", emptyCondition(), nil, true},
		"auth method changed":                   {alertResourceTypeIdentityAuthentication, "auth-method-changed", nil, nil, false},
		"event of another resource type":        {alertResourceTypeIdentityAuthentication, "certificate.expiry", withDays(30), nil, true},
		"full event of another resource type":   {alertResourceTypeCertManagerApplication, alertEventTypeCertManagerCertIssuance, nil, nil, true},
		"deprecated expiry on identity":         {alertResourceTypeIdentityAuthentication, "", nil, &expiryConditionModel{AlertBeforeDays: types.Int64Value(30), DailyReminder: types.BoolValue(false)}, false},
		"deprecated expiry on an application":   {alertResourceTypeCertManagerApplication, "", nil, &expiryConditionModel{AlertBeforeDays: types.Int64Value(30), DailyReminder: types.BoolValue(false)}, true},
		"deprecated expiry next to a condition": {alertResourceTypeIdentityAuthentication, "", withDays(30), &expiryConditionModel{AlertBeforeDays: types.Int64Value(30), DailyReminder: types.BoolValue(false)}, true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			model := alertModel(t, map[string]alertChannelModel{})
			model.ResourceType = types.StringValue(c.resourceType)
			model.EventType = newAlertEventTypeValue(c.eventType)
			if c.eventType == "" {
				model.EventType = alertEventTypeValue{StringValue: types.StringNull()}
			}
			model.Condition = c.condition
			model.Expiry = c.expiry

			diags := validateAlertEvent(ctx, alertConfig(t, s, model), c.resourceType)
			if diags.HasError() != c.wantError {
				t.Errorf("validateAlertEvent() error = %v, want %v: %v", diags.HasError(), c.wantError, diags)
			}
		})
	}
}

// event_type and the deprecated expiry block are exclusive, and one of them is required.
func TestConfigValidatorsRequireOneEventSource(t *testing.T) {
	ctx := context.Background()
	s := alertSchema(t)

	model := alertModel(t, map[string]alertChannelModel{})
	model.Expiry = &expiryConditionModel{AlertBeforeDays: types.Int64Value(30), DailyReminder: types.BoolValue(false)}
	both := alertConfig(t, s, model)

	model.EventType = alertEventTypeValue{StringValue: types.StringNull()}
	model.Expiry = nil
	neither := alertConfig(t, s, model)

	for name, config := range map[string]tfsdk.Config{"both": both, "neither": neither} {
		var failed bool
		for _, v := range (&alertResource{}).ConfigValidators(ctx) {
			resp := &resource.ValidateConfigResponse{}
			v.ValidateResource(ctx, resource.ValidateConfigRequest{Config: config}, resp)
			failed = failed || resp.Diagnostics.HasError()
		}
		if !failed {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestAlertResourceModelMatchesSchema(t *testing.T) {
	s := alertSchema(t)
	model := alertModel(t, map[string]alertChannelModel{})
	model.Condition = emptyCondition()
	model.Condition.AlertBeforeDays = types.Int64Value(30)

	var roundTripped alertResourceModel
	if diags := alertPlan(t, s, model).Get(context.Background(), &roundTripped); diags.HasError() {
		t.Fatalf("round trip: %v", diags)
	}
	if roundTripped.EventType.ValueString() != "expiry" || roundTripped.Condition.AlertBeforeDays.ValueInt64() != 30 {
		t.Errorf("model did not survive the round trip: %+v", roundTripped)
	}
}

// The Event types table in the docs template is written by hand, so it has to list every event.
func TestAlertDocsTemplateListsEveryEvent(t *testing.T) {
	template, err := os.ReadFile("../../../../templates/resources/alert.md.tmpl")
	if err != nil {
		t.Fatalf("reading the docs template: %v", err)
	}
	for _, event := range alertEvents {
		condition := "None"
		switch {
		case event.firesBeforeExpiry() && len(event.filters) > 0:
			condition = fmt.Sprintf("`alert_before_days` (1 to %d), optional `daily_reminder`, optional filters", event.maxAlertBeforeDays)
		case event.firesBeforeExpiry():
			condition = fmt.Sprintf("`alert_before_days` (1 to %d), optional `daily_reminder`", event.maxAlertBeforeDays)
		case len(event.filters) > 0:
			condition = "Optional filters"
		}
		row := fmt.Sprintf("| `%s` | `%s` | `%s` | %s |", event.resourceType, event.shortEventType(), event.eventType, condition)
		if !strings.Contains(string(template), row) {
			t.Errorf("templates/resources/alert.md.tmpl has no Event types row matching %q", row)
		}
	}

	rows := 0
	for _, line := range strings.Split(string(template), "\n") {
		if strings.HasPrefix(line, "| `") && !strings.Contains(line, "Full event type") {
			rows++
		}
	}
	if rows != len(alertEvents) {
		t.Errorf("the Event types table has %d rows, want one per event (%d)", rows, len(alertEvents))
	}
}

func TestSetStateFromAlertEventType(t *testing.T) {
	resourceID := "identity-1"
	alert := infisical.Alert{
		ID:           "alert-1",
		ResourceType: alertResourceTypeIdentityAuthentication,
		ResourceID:   &resourceID,
		EventType:    alertEventTypeIdentityAuthenticationExpiry,
		Condition:    json.RawMessage(`{"alertBefore":"30d"}`),
	}

	legacy := alertResourceModel{Expiry: &expiryConditionModel{}}
	if diags := (&alertResource{}).setStateFromAlert(context.Background(), &legacy, alert); diags.HasError() {
		t.Fatalf("setStateFromAlert() diagnostics = %v", diags)
	}
	if !legacy.EventType.IsNull() || legacy.Expiry == nil {
		t.Errorf("legacy alert: event_type = %v, expiry = %v, want no event_type and the expiry block", legacy.EventType, legacy.Expiry)
	}

	var imported alertResourceModel
	if diags := (&alertResource{}).setStateFromAlert(context.Background(), &imported, alert); diags.HasError() {
		t.Fatalf("setStateFromAlert() diagnostics = %v", diags)
	}
	if imported.EventType.ValueString() != alertEventTypeIdentityAuthenticationExpiry || imported.Expiry != nil {
		t.Errorf("imported alert: event_type = %v, expiry = %v, want the full event type and no expiry block", imported.EventType, imported.Expiry)
	}
}
