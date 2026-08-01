package resource

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func alertSchemaAttributes(t *testing.T) map[string]bool {
	t.Helper()

	resp := &resource.SchemaResponse{}
	(&alertResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", resp.Diagnostics)
	}

	names := make(map[string]bool, len(resp.Schema.Attributes))
	for name := range resp.Schema.Attributes {
		names[name] = true
	}
	return names
}

// Every event's condition block has to exist in the schema under exactly the name the table gives,
// because the table is what maps a block back to the event type sent to the API.
func TestAlertEventsHaveConditionBlocks(t *testing.T) {
	attributes := alertSchemaAttributes(t)

	for _, event := range alertEvents {
		if !attributes[event.block] {
			t.Errorf("event %q declares condition block %q, which the schema does not have", event.eventType, event.block)
		}
		if event.resourceType == "" || event.eventType == "" {
			t.Errorf("event %+v is missing a resource type or event type", event)
		}
	}
}

// The resource model is converted to and from the schema on every operation, so a tfsdk tag that
// does not match the schema is a runtime error on first use. Converting a populated model catches
// every mismatch at once, across the alert, its condition block and its channels.
func TestAlertResourceModelMatchesSchema(t *testing.T) {
	resp := &resource.SchemaResponse{}
	(&alertResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", resp.Diagnostics)
	}

	model := alertResourceModel{
		ID:           types.StringValue("alert-1"),
		ResourceType: types.StringValue(alertResourceTypeIdentityAuthentication),
		ResourceID:   types.StringValue("identity-1"),
		ProjectID:    types.StringNull(),
		Name:         types.StringValue("Credentials expiring"),
		Description:  types.StringNull(),
		Enabled:      types.BoolValue(true),
		AuthenticationExpiry: &authenticationExpiryConditionModel{
			AlertBeforeDays: types.Int64Value(30),
			DailyReminder:   types.BoolValue(false),
		},
		Channels: map[string]alertChannelModel{
			"Security team": {
				ID:      types.StringValue("channel-1"),
				Enabled: types.BoolValue(true),
				Email: emailChannel(t, alertChannelRecipientModel{
					Type: types.StringValue(AlertRecipientTypeUser),
					ID:   types.StringValue("user-1"),
				}),
			},
			"On-call": {
				ID:        types.StringValue("channel-2"),
				Enabled:   types.BoolValue(true),
				PagerDuty: &alertPagerDutyChannelModel{IntegrationKey: types.StringValue("00000000000000000000000000000000")},
			},
		},
	}

	var value attr.Value
	if diags := tfsdk.ValueFrom(context.Background(), model, resp.Schema.Type(), &value); diags.HasError() {
		t.Fatalf("converting the model to its schema type: %v", diags)
	}

	var roundTripped alertResourceModel
	if diags := tfsdk.ValueAs(context.Background(), value, &roundTripped); diags.HasError() {
		t.Fatalf("converting the schema type back to the model: %v", diags)
	}

	if roundTripped.AuthenticationExpiry == nil || roundTripped.AuthenticationExpiry.AlertBeforeDays.ValueInt64() != 30 {
		t.Errorf("condition did not survive the round trip: %+v", roundTripped.AuthenticationExpiry)
	}
	if len(roundTripped.Channels) != 2 {
		t.Errorf("channels did not survive the round trip: %+v", roundTripped.Channels)
	}
}

func TestSupportedAlertResourceTypes(t *testing.T) {
	types := supportedAlertResourceTypes()
	if len(types) != 1 || types[0] != alertResourceTypeIdentityAuthentication {
		t.Errorf("supportedAlertResourceTypes() = %v, want only %q", types, alertResourceTypeIdentityAuthentication)
	}
}

func TestAlertConditionForAPI(t *testing.T) {
	plan := alertResourceModel{
		ResourceType: types.StringValue(alertResourceTypeIdentityAuthentication),
		AuthenticationExpiry: &authenticationExpiryConditionModel{
			AlertBeforeDays: types.Int64Value(30),
			DailyReminder:   types.BoolValue(true),
		},
	}

	event, condition, diags := alertConditionForAPI(plan)
	if diags.HasError() {
		t.Fatalf("alertConditionForAPI() diagnostics = %v", diags)
	}

	if event.eventType != alertEventTypeIdentityAuthenticationExpiry {
		t.Errorf("event type = %q, want %q", event.eventType, alertEventTypeIdentityAuthenticationExpiry)
	}
	if event.resourceType != alertResourceTypeIdentityAuthentication {
		t.Errorf("resource type = %q, want %q", event.resourceType, alertResourceTypeIdentityAuthentication)
	}

	got, ok := condition.(authenticationExpiryCondition)
	if !ok {
		t.Fatalf("condition = %T, want an authenticationExpiryCondition", condition)
	}
	if got.AlertBefore != "30d" {
		t.Errorf("alertBefore = %q, want 30d", got.AlertBefore)
	}
	if !got.DailyReminder {
		t.Error("dailyReminder = false, want true")
	}
}

func TestAlertConditionForAPIWithoutABlock(t *testing.T) {
	_, _, diags := alertConditionForAPI(alertResourceModel{})
	if !diags.HasError() {
		t.Error("alertConditionForAPI() without a condition block: no error, want one")
	}
}

func TestSetAlertConditionFromAPI(t *testing.T) {
	event, ok := alertEventForEventType(alertEventTypeIdentityAuthenticationExpiry)
	if !ok {
		t.Fatal("no event registered for the identity authentication expiry event type")
	}

	var state alertResourceModel
	diags := setAlertConditionFromAPI(&state, event, json.RawMessage(`{"alertBefore":"7d","dailyReminder":true}`))
	if diags.HasError() {
		t.Fatalf("setAlertConditionFromAPI() diagnostics = %v", diags)
	}

	if state.AuthenticationExpiry == nil {
		t.Fatal("authentication_expiry block is nil, want it populated")
	}
	if got := state.AuthenticationExpiry.AlertBeforeDays.ValueInt64(); got != 7 {
		t.Errorf("alert_before_days = %d, want 7", got)
	}
	if !state.AuthenticationExpiry.DailyReminder.ValueBool() {
		t.Error("daily_reminder = false, want true")
	}
}

// A condition the API stores without a dailyReminder is the same as one that opted out of it.
func TestSetAlertConditionFromAPIDefaultsDailyReminder(t *testing.T) {
	event, _ := alertEventForEventType(alertEventTypeIdentityAuthenticationExpiry)

	var state alertResourceModel
	diags := setAlertConditionFromAPI(&state, event, json.RawMessage(`{"alertBefore":"30d"}`))
	if diags.HasError() {
		t.Fatalf("setAlertConditionFromAPI() diagnostics = %v", diags)
	}

	if state.AuthenticationExpiry.DailyReminder.ValueBool() {
		t.Error("daily_reminder = true, want false")
	}
}

func TestSetAlertConditionFromAPIRejectsUnreadableConditions(t *testing.T) {
	event, _ := alertEventForEventType(alertEventTypeIdentityAuthenticationExpiry)

	cases := map[string]string{
		"no condition":         ``,
		"null condition":       `null`,
		"not a day count":      `{"alertBefore":"3 months"}`,
		"a unit we cannot map": `{"alertBefore":"2w"}`,
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			var state alertResourceModel
			diags := setAlertConditionFromAPI(&state, event, json.RawMessage(raw))
			if !diags.HasError() {
				t.Errorf("setAlertConditionFromAPI(%q): no error, want one", raw)
			}
			if state.AuthenticationExpiry != nil {
				t.Error("authentication_expiry block was populated despite the error")
			}
		})
	}
}

func TestParseAlertBeforeDays(t *testing.T) {
	cases := map[string]struct {
		want      int64
		wantError bool
	}{
		"1d":  {want: 1},
		"30d": {want: 30},
		"90d": {want: 90},
		"30":  {wantError: true},
		"1m":  {wantError: true},
		"":    {wantError: true},
		"d":   {wantError: true},
	}

	for alertBefore, c := range cases {
		t.Run(alertBefore, func(t *testing.T) {
			got, diags := parseAlertBeforeDays(alertBefore)
			if diags.HasError() != c.wantError {
				t.Fatalf("parseAlertBeforeDays(%q) error = %v, want error = %v", alertBefore, diags.HasError(), c.wantError)
			}
			if !c.wantError && got != c.want {
				t.Errorf("parseAlertBeforeDays(%q) = %d, want %d", alertBefore, got, c.want)
			}
		})
	}
}
