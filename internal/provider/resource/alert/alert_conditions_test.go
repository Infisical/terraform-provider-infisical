package resource

import (
	"context"
	"encoding/json"
	"slices"
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
		Expiry: &expiryConditionModel{
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

	if roundTripped.Expiry == nil || roundTripped.Expiry.AlertBeforeDays.ValueInt64() != 30 {
		t.Errorf("condition did not survive the round trip: %+v", roundTripped.Expiry)
	}
	if len(roundTripped.Channels) != 2 {
		t.Errorf("channels did not survive the round trip: %+v", roundTripped.Channels)
	}
}

func TestSupportedAlertResourceTypes(t *testing.T) {
	types := supportedAlertResourceTypes()
	want := []string{alertResourceTypeCertManager, alertResourceTypeCertManagerApplication, alertResourceTypeCertManagerSigner, alertResourceTypeIdentityAuthentication}
	if !slices.Equal(types, want) {
		t.Errorf("supportedAlertResourceTypes() = %v, want %v", types, want)
	}
}

// An event is keyed on its resource type and its condition block together, so every resource type
// that expires can share one expiry block instead of each declaring its own copy of the same fields.
func TestAlertEventFor(t *testing.T) {
	event, ok := alertEventFor(alertResourceTypeIdentityAuthentication, alertBlockExpiry)
	if !ok {
		t.Fatalf("alertEventFor(%q, %q) found no event", alertResourceTypeIdentityAuthentication, alertBlockExpiry)
	}
	if event.eventType != alertEventTypeIdentityAuthenticationExpiry {
		t.Errorf("event type = %q, want %q", event.eventType, alertEventTypeIdentityAuthenticationExpiry)
	}

	if _, ok := alertEventFor("certificate", alertBlockExpiry); ok {
		t.Error("alertEventFor() resolved an expiry event for a resource type that has none")
	}
	if _, ok := alertEventFor(alertResourceTypeIdentityAuthentication, "rotation_failed"); ok {
		t.Error("alertEventFor() resolved an event for a condition block that does not exist")
	}
}

// The schema has one attribute per condition block, so a block that several events share has to be
// listed once. Listing it twice would have ExactlyOneOf compare the block against itself.
func TestAlertConditionBlockNamesAreUnique(t *testing.T) {
	blocks := alertConditionBlockNames()

	seen := make(map[string]bool, len(blocks))
	for _, block := range blocks {
		if seen[block] {
			t.Errorf("alertConditionBlockNames() lists %q more than once: %v", block, blocks)
		}
		seen[block] = true
	}

	for _, event := range alertEvents {
		if !seen[event.block] {
			t.Errorf("alertConditionBlockNames() = %v, missing %q", blocks, event.block)
		}
	}
}

func TestAlertBlocksForResourceType(t *testing.T) {
	blocks := alertBlocksForResourceType(alertResourceTypeIdentityAuthentication)
	if len(blocks) != 1 || blocks[0] != alertBlockExpiry {
		t.Errorf("alertBlocksForResourceType(%q) = %v, want only %q", alertResourceTypeIdentityAuthentication, blocks, alertBlockExpiry)
	}

	if got := alertBlocksForResourceType("certificate"); len(got) != 0 {
		t.Errorf("alertBlocksForResourceType() for a resource type with no events = %v, want none", got)
	}
}

// The expiry block's description names the resource types it applies to, so the list has to come from
// the event table rather than being written out by hand.
func TestAlertResourceTypesForBlock(t *testing.T) {
	resourceTypes := alertResourceTypesForBlock(alertBlockExpiry)
	want := []string{alertResourceTypeCertManager, alertResourceTypeCertManagerApplication, alertResourceTypeCertManagerSigner, alertResourceTypeIdentityAuthentication}
	if !slices.Equal(resourceTypes, want) {
		t.Errorf("alertResourceTypesForBlock(%q) = %v, want %v", alertBlockExpiry, resourceTypes, want)
	}

	if got := alertResourceTypesForBlock(alertBlockIssuance); !slices.Equal(got, []string{alertResourceTypeCertManager, alertResourceTypeCertManagerApplication}) {
		t.Errorf("alertResourceTypesForBlock(%q) = %v, want the application and Certificate Manager scopes", alertBlockIssuance, got)
	}

	if got := alertResourceTypesForBlock("rotation_failed"); len(got) != 0 {
		t.Errorf("alertResourceTypesForBlock() for an unknown block = %v, want none", got)
	}
}

func TestAlertConditionForAPI(t *testing.T) {
	plan := alertResourceModel{
		ResourceType: types.StringValue(alertResourceTypeIdentityAuthentication),
		Expiry: &expiryConditionModel{
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

	got, ok := condition.(expiryCondition)
	if !ok {
		t.Fatalf("condition = %T, want an expiryCondition", condition)
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

// The block alone does not name an event type, so a condition on a resource type that has no matching
// event has nothing to be sent under and cannot fall back to another resource type's event.
func TestAlertConditionForAPIRejectsBlockOnWrongResourceType(t *testing.T) {
	plan := alertResourceModel{
		ResourceType: types.StringValue("certificate"),
		Expiry: &expiryConditionModel{
			AlertBeforeDays: types.Int64Value(30),
			DailyReminder:   types.BoolValue(false),
		},
	}

	if _, _, diags := alertConditionForAPI(plan); !diags.HasError() {
		t.Error("alertConditionForAPI() with a condition the resource type has no event for: no error, want one")
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

	if state.Expiry == nil {
		t.Fatal("expiry block is nil, want it populated")
	}
	if got := state.Expiry.AlertBeforeDays.ValueInt64(); got != 7 {
		t.Errorf("alert_before_days = %d, want 7", got)
	}
	if !state.Expiry.DailyReminder.ValueBool() {
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

	if state.Expiry.DailyReminder.ValueBool() {
		t.Error("daily_reminder = true, want false")
	}
}

func TestSetAlertConditionFromAPIRejectsUnreadableConditions(t *testing.T) {
	event, _ := alertEventForEventType(alertEventTypeIdentityAuthenticationExpiry)

	cases := map[string]string{
		"no condition":         ``,
		"null condition":       `null`,
		"not a day count":      `{"alertBefore":"3 months"}`,
		"a unit we cannot map": `{"alertBefore":"2h"}`,
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			var state alertResourceModel
			diags := setAlertConditionFromAPI(&state, event, json.RawMessage(raw))
			if !diags.HasError() {
				t.Errorf("setAlertConditionFromAPI(%q): no error, want one", raw)
			}
			if state.Expiry != nil {
				t.Error("expiry block was populated despite the error")
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
		"2w":  {want: 14},
		"1m":  {want: 30},
		"1y":  {want: 365},
		"30":  {wantError: true},
		"2h":  {wantError: true},
		"-1d": {wantError: true},
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

func TestAlertConditionForAPIWithAnEventBlock(t *testing.T) {
	cases := map[string]struct {
		plan alertResourceModel
		want string
	}{
		alertBlockIssuance:   {alertResourceModel{Issuance: &eventConditionModel{}}, alertEventTypeCertManagerApplicationCertIssuance},
		alertBlockRenewal:    {alertResourceModel{Renewal: &eventConditionModel{}}, alertEventTypeCertManagerApplicationCertRenewal},
		alertBlockRevocation: {alertResourceModel{Revocation: &eventConditionModel{}}, alertEventTypeCertManagerApplicationCertRevocation},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			c.plan.ResourceType = types.StringValue(alertResourceTypeCertManagerApplication)

			event, condition, diags := alertConditionForAPI(c.plan)
			if diags.HasError() {
				t.Fatalf("alertConditionForAPI() diagnostics = %v", diags)
			}
			if event.eventType != c.want {
				t.Errorf("event type = %q, want %q", event.eventType, c.want)
			}
			if condition != nil {
				t.Errorf("condition = %#v, want none", condition)
			}
		})
	}
}

func TestAlertConditionForAPIWithApplicationExpiry(t *testing.T) {
	event, condition, diags := alertConditionForAPI(alertResourceModel{
		ResourceType: types.StringValue(alertResourceTypeCertManagerApplication),
		Expiry:       &expiryConditionModel{AlertBeforeDays: types.Int64Value(200), DailyReminder: types.BoolValue(false)},
	})
	if diags.HasError() {
		t.Fatalf("alertConditionForAPI() diagnostics = %v", diags)
	}
	if event.eventType != alertEventTypeCertManagerApplicationCertExpiry {
		t.Errorf("event type = %q, want %q", event.eventType, alertEventTypeCertManagerApplicationCertExpiry)
	}
	got, ok := condition.(expiryCondition)
	if !ok {
		t.Fatalf("condition = %T, want an expiryCondition", condition)
	}
	if got.AlertBefore != "200d" {
		t.Errorf("alertBefore = %q, want 200d", got.AlertBefore)
	}
}

func TestSetAlertConditionFromAPIWithAnEventBlock(t *testing.T) {
	event, ok := alertEventForEventType(alertEventTypeCertManagerApplicationCertRevocation)
	if !ok {
		t.Fatal("no event registered for the application certificate revocation event type")
	}

	for _, raw := range []string{`null`, `{}`, ``} {
		state := alertResourceModel{Expiry: &expiryConditionModel{}}
		if diags := setAlertConditionFromAPI(&state, event, json.RawMessage(raw)); diags.HasError() {
			t.Fatalf("setAlertConditionFromAPI(%q) diagnostics = %v", raw, diags)
		}
		if state.Revocation == nil {
			t.Errorf("setAlertConditionFromAPI(%q): revocation block not set", raw)
		}
		if state.Expiry != nil || state.Issuance != nil || state.Renewal != nil {
			t.Errorf("setAlertConditionFromAPI(%q): another condition block is still set", raw)
		}
	}
}

func TestAlertResourceTypeTraits(t *testing.T) {
	if !alertResourceTypes[alertResourceTypeCertManagerApplication].resolvesProject {
		t.Error("an application alert's project should be resolved from the application")
	}
	if alertResourceTypes[alertResourceTypeIdentityAuthentication].resolvesProject {
		t.Error("an identity alert's project is whatever the configuration says, so it must not be resolved")
	}
	if !alertResourceTypes[alertResourceTypeCertManagerApplication].allowsEmailRecipients {
		t.Error("application alerts should accept email address recipients")
	}
	if alertResourceTypes[alertResourceTypeIdentityAuthentication].allowsEmailRecipients {
		t.Error("identity alerts should not accept email address recipients")
	}

	for _, event := range alertEvents {
		if _, ok := alertResourceTypes[event.resourceType]; !ok {
			t.Errorf("event %q is on resource type %q, which has no entry in alertResourceTypes", event.eventType, event.resourceType)
		}
		if event.block == alertBlockExpiry && event.maxAlertBeforeDays < minAlertBeforeDays {
			t.Errorf("expiry event %q has no alert_before_days limit", event.eventType)
		}
	}
}

func TestAlertConditionForAPIWithFilters(t *testing.T) {
	filters := &alertFiltersModel{
		ApplicationIDs: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("app-2"), types.StringValue("app-1")}),
		ProfileIDs:     types.SetNull(types.StringType),
		Sources:        types.SetValueMust(types.StringType, []attr.Value{types.StringValue("issued")}),
	}

	_, expiry, diags := alertConditionForAPI(alertResourceModel{
		ResourceType: types.StringValue(alertResourceTypeCertManager),
		Expiry:       &expiryConditionModel{AlertBeforeDays: types.Int64Value(30), DailyReminder: types.BoolValue(false)},
		Filters:      filters,
	})
	if diags.HasError() {
		t.Fatalf("alertConditionForAPI() diagnostics = %v", diags)
	}
	raw, _ := json.Marshal(expiry)
	if string(raw) != `{"alertBefore":"30d","dailyReminder":false,"applicationIds":["app-1","app-2"],"sources":["issued"]}` {
		t.Errorf("expiry condition = %s, want the filters merged into it and no empty profileIds", raw)
	}

	event, issuance, diags := alertConditionForAPI(alertResourceModel{
		ResourceType: types.StringValue(alertResourceTypeCertManager),
		Issuance:     &eventConditionModel{},
		Filters:      filters,
	})
	if diags.HasError() {
		t.Fatalf("alertConditionForAPI() diagnostics = %v", diags)
	}
	if event.eventType != alertEventTypeCertManagerCertIssuance {
		t.Errorf("event type = %q, want %q", event.eventType, alertEventTypeCertManagerCertIssuance)
	}
	raw, _ = json.Marshal(issuance)
	if string(raw) != `{"applicationIds":["app-1","app-2"],"sources":["issued"]}` {
		t.Errorf("issuance condition = %s, want only the filters", raw)
	}
}

func TestSetAlertConditionFromAPIWithFilters(t *testing.T) {
	event, _ := alertEventForEventType(alertEventTypeCertManagerCertExpiry)

	var state alertResourceModel
	diags := setAlertConditionFromAPI(&state, event, json.RawMessage(`{"alertBefore":"2w","dailyReminder":true,"profileIds":["profile-1"]}`))
	if diags.HasError() {
		t.Fatalf("setAlertConditionFromAPI() diagnostics = %v", diags)
	}
	if state.Expiry == nil || state.Expiry.AlertBeforeDays.ValueInt64() != 14 {
		t.Errorf("expiry = %+v, want 14 days", state.Expiry)
	}
	if state.Filters == nil || len(state.Filters.ProfileIDs.Elements()) != 1 {
		t.Fatalf("filters = %+v, want the profile filter", state.Filters)
	}
	if !state.Filters.ApplicationIDs.IsNull() || !state.Filters.Sources.IsNull() {
		t.Errorf("filters = %+v, want the lists Infisical left out to stay null", state.Filters)
	}

	issuance, _ := alertEventForEventType(alertEventTypeCertManagerCertIssuance)
	for _, raw := range []string{`null`, `{}`} {
		state := alertResourceModel{Filters: &alertFiltersModel{}}
		if diags := setAlertConditionFromAPI(&state, issuance, json.RawMessage(raw)); diags.HasError() {
			t.Fatalf("setAlertConditionFromAPI(%q) diagnostics = %v", raw, diags)
		}
		if state.Filters != nil || state.Issuance == nil {
			t.Errorf("setAlertConditionFromAPI(%q): filters = %+v, issuance = %v, want no filters and the issuance block", raw, state.Filters, state.Issuance)
		}
	}
}

func TestDescribeAlertBeforeDaysLimits(t *testing.T) {
	got := describeAlertBeforeDaysLimits()
	want := "90 for identity.authentication; 365 for cert-manager, cert-manager.application, cert-manager.signer"
	if got != want {
		t.Errorf("describeAlertBeforeDaysLimits() = %q, want %q", got, want)
	}
}

func TestValidateConfigWithEventOnlyAlerts(t *testing.T) {
	ctx := context.Background()
	s := alertSchema(t)

	for _, resourceType := range []string{alertResourceTypeCertManagerApplication, alertResourceTypeCertManager} {
		for _, block := range []string{alertBlockIssuance, alertBlockRenewal, alertBlockRevocation} {
			t.Run(resourceType+"/"+block, func(t *testing.T) {
				model := alertModel(t, map[string]alertChannelModel{})
				model.ResourceType = types.StringValue(resourceType)
				if alertResourceTypes[resourceType].scopeWide {
					model.ResourceID = types.StringNull()
				}
				model.Expiry = nil
				switch block {
				case alertBlockIssuance:
					model.Issuance = &eventConditionModel{}
				case alertBlockRenewal:
					model.Renewal = &eventConditionModel{}
				case alertBlockRevocation:
					model.Revocation = &eventConditionModel{}
				}

				resp := &resource.ValidateConfigResponse{}
				(&alertResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: alertConfig(t, s, model)}, resp)
				if resp.Diagnostics.HasError() {
					t.Errorf("ValidateConfig() diagnostics = %v, want none", resp.Diagnostics)
				}
			})
		}
	}
}
