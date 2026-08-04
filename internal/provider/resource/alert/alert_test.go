package resource

import (
	"context"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func alertSchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &resource.SchemaResponse{}
	(&alertResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", resp.Diagnostics)
	}
	return resp.Schema
}

// alertObject renders a model the way Terraform holds it, so a plan or state built from it goes
// through the same conversion the framework does at runtime.
func alertObject(t *testing.T, s schema.Schema, model alertResourceModel) attr.Value {
	t.Helper()

	var value attr.Value
	if diags := tfsdk.ValueFrom(context.Background(), model, s.Type(), &value); diags.HasError() {
		t.Fatalf("converting the model to its schema type: %v", diags)
	}
	return value
}

func alertPlan(t *testing.T, s schema.Schema, model alertResourceModel) tfsdk.Plan {
	t.Helper()

	raw, err := alertObject(t, s, model).ToTerraformValue(context.Background())
	if err != nil {
		t.Fatalf("converting the model to a raw value: %v", err)
	}
	return tfsdk.Plan{Raw: raw, Schema: s}
}

func alertState(t *testing.T, s schema.Schema, model alertResourceModel) tfsdk.State {
	t.Helper()

	plan := alertPlan(t, s, model)
	return tfsdk.State{Raw: plan.Raw, Schema: s}
}

func alertConfig(t *testing.T, s schema.Schema, model alertResourceModel) tfsdk.Config {
	t.Helper()

	plan := alertPlan(t, s, model)
	return tfsdk.Config{Raw: plan.Raw, Schema: s}
}

func alertModel(t *testing.T, channels map[string]alertChannelModel) alertResourceModel {
	t.Helper()

	return alertResourceModel{
		ID:           types.StringValue("alert-1"),
		ResourceType: types.StringValue(alertResourceTypeIdentityAuthentication),
		ResourceID:   types.StringValue("identity-1"),
		ProjectID:    types.StringNull(),
		OrgID:        types.StringValue("org-1"),
		Name:         types.StringValue("Credentials expiring"),
		Description:  types.StringNull(),
		Enabled:      types.BoolValue(true),
		Expiry: &expiryConditionModel{
			AlertBeforeDays: types.Int64Value(30),
			DailyReminder:   types.BoolValue(false),
		},
		Channels: channels,
	}
}

func modifiedChannels(t *testing.T, plan tfsdk.Plan, state tfsdk.State) map[string]alertChannelModel {
	t.Helper()

	ctx := context.Background()
	resp := &resource.ModifyPlanResponse{Plan: plan}
	(&alertResource{}).ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan() diagnostics = %v", resp.Diagnostics)
	}

	var modified alertResourceModel
	if diags := resp.Plan.Get(ctx, &modified); diags.HasError() {
		t.Fatalf("reading the modified plan: %v", diags)
	}
	return modified.Channels
}

// Infisical replaces a channel whose type changed, so the ID Terraform carried into the plan cannot
// survive the apply and has to be planned as unknown. Otherwise the apply fails on an inconsistent
// result for an alert that was in fact written.
func TestModifyPlanUnknownsIDOfRetypedChannel(t *testing.T) {
	s := alertSchema(t)

	stored := alertModel(t, map[string]alertChannelModel{
		"on_call": {
			ID:      types.StringValue("channel-1"),
			Name:    types.StringValue("On-call"),
			Enabled: types.BoolValue(true),
			Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
		},
		"security_team": {
			ID:      types.StringValue("channel-2"),
			Name:    types.StringValue("Security team"),
			Enabled: types.BoolValue(true),
			Email:   emailChannel(t, alertChannelRecipientModel{Type: types.StringValue(AlertRecipientTypeUser), ID: types.StringValue("user-1")}),
		},
	})

	// The plan Terraform proposes keeps the stored ID for both keys, even the one that swapped block.
	planned := alertModel(t, map[string]alertChannelModel{
		"on_call": {
			ID:        types.StringValue("channel-1"),
			Name:      types.StringValue("On-call"),
			Enabled:   types.BoolValue(true),
			PagerDuty: &alertPagerDutyChannelModel{IntegrationKey: types.StringValue("00000000000000000000000000000000")},
		},
		"security_team": {
			ID:      types.StringValue("channel-2"),
			Name:    types.StringValue("Security team"),
			Enabled: types.BoolValue(true),
			Email:   emailChannel(t, alertChannelRecipientModel{Type: types.StringValue(AlertRecipientTypeUser), ID: types.StringValue("user-2")}),
		},
	})

	channels := modifiedChannels(t, alertPlan(t, s, planned), alertState(t, s, stored))

	if !channels["on_call"].ID.IsUnknown() {
		t.Errorf("retyped channel ID = %v, want it unknown so the apply can write the replacement's ID", channels["on_call"].ID)
	}
	// An edited channel of the same type is updated in place, so it keeps the ID it already has.
	if got := channels["security_team"].ID.ValueString(); got != "channel-2" {
		t.Errorf("edited channel ID = %q, want channel-2 kept", got)
	}
}

// A channel that only changed its settings is updated in place, so nothing about its ID may move.
func TestModifyPlanKeepsIDOfEditedChannel(t *testing.T) {
	s := alertSchema(t)

	slack := func(url string) map[string]alertChannelModel {
		return map[string]alertChannelModel{"platform_slack": {
			ID:      types.StringValue("channel-1"),
			Name:    types.StringValue("Platform Slack"),
			Enabled: types.BoolValue(true),
			Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue(url)},
		}}
	}

	channels := modifiedChannels(t,
		alertPlan(t, s, alertModel(t, slack("https://hooks.slack.com/services/xyz"))),
		alertState(t, s, alertModel(t, slack("https://hooks.slack.com/services/abc"))),
	)

	if got := channels["platform_slack"].ID.ValueString(); got != "channel-1" {
		t.Errorf("edited channel ID = %q, want channel-1 kept", got)
	}
}

// A channel whose ID never made it into state cannot be updated in place, so the replacement's ID has
// to be planned as unknown just like a retyped channel's.
func TestModifyPlanUnknownsIDOfChannelWithoutOne(t *testing.T) {
	s := alertSchema(t)

	channel := func(id types.String) map[string]alertChannelModel {
		return map[string]alertChannelModel{"platform_slack": {
			ID:      id,
			Name:    types.StringValue("Platform Slack"),
			Enabled: types.BoolValue(true),
			Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
		}}
	}

	channels := modifiedChannels(t,
		alertPlan(t, s, alertModel(t, channel(types.StringNull()))),
		alertState(t, s, alertModel(t, channel(types.StringNull()))),
	)

	if !channels["platform_slack"].ID.IsUnknown() {
		t.Errorf("channel ID = %v, want it unknown so the apply can write the replacement's ID", channels["platform_slack"].ID)
	}
}

// A renamed channel is the same channel under the same key, so it is updated in place and keeps its ID.
func TestModifyPlanKeepsIDOfRenamedChannel(t *testing.T) {
	s := alertSchema(t)

	slack := func(name string) map[string]alertChannelModel {
		return map[string]alertChannelModel{"platform_slack": {
			ID:      types.StringValue("channel-1"),
			Name:    types.StringValue(name),
			Enabled: types.BoolValue(true),
			Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
		}}
	}

	channels := modifiedChannels(t,
		alertPlan(t, s, alertModel(t, slack("Platform alerts"))),
		alertState(t, s, alertModel(t, slack("Platform Slack"))),
	)

	if got := channels["platform_slack"].ID.ValueString(); got != "channel-1" {
		t.Errorf("renamed channel ID = %q, want channel-1 kept", got)
	}
}

// A rekeyed channel is a new map key that never had an ID, so the plan already reads unknown and
// there is nothing to correct.
func TestModifyPlanLeavesRekeyedChannelAlone(t *testing.T) {
	s := alertSchema(t)

	stored := alertModel(t, map[string]alertChannelModel{"platform_slack": {
		ID:      types.StringValue("channel-1"),
		Name:    types.StringValue("Platform Slack"),
		Enabled: types.BoolValue(true),
		Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
	}})

	planned := alertModel(t, map[string]alertChannelModel{"platform_alerts": {
		ID:      types.StringUnknown(),
		Name:    types.StringValue("Platform Slack"),
		Enabled: types.BoolValue(true),
		Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
	}})

	channels := modifiedChannels(t, alertPlan(t, s, planned), alertState(t, s, stored))

	if !channels["platform_alerts"].ID.IsUnknown() {
		t.Errorf("rekeyed channel ID = %v, want it unknown", channels["platform_alerts"].ID)
	}
}

// There is no stored state to carry an ID over from on create, so the plan is left as it is.
func TestModifyPlanOnCreate(t *testing.T) {
	s := alertSchema(t)

	planned := alertModel(t, map[string]alertChannelModel{"on_call": {
		ID:        types.StringUnknown(),
		Name:      types.StringValue("On-call"),
		Enabled:   types.BoolValue(true),
		PagerDuty: &alertPagerDutyChannelModel{IntegrationKey: types.StringValue("00000000000000000000000000000000")},
	}})

	channels := modifiedChannels(t, alertPlan(t, s, planned), tfsdk.State{Schema: s})

	if !channels["on_call"].ID.IsUnknown() {
		t.Errorf("channel ID = %v, want it left unknown", channels["on_call"].ID)
	}
}

// Editing the condition is an update, so nothing about it may ask for a replacement. Swapping the
// block for another event's would, but there is only one event to swap between so far.
func TestModifyPlanKeepsAnEditedConditionInPlace(t *testing.T) {
	s := alertSchema(t)

	alert := func(alertBeforeDays int64) alertResourceModel {
		model := alertModel(t, map[string]alertChannelModel{"platform_slack": {
			ID:      types.StringValue("channel-1"),
			Name:    types.StringValue("Platform Slack"),
			Enabled: types.BoolValue(true),
			Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
		}})
		model.Expiry.AlertBeforeDays = types.Int64Value(alertBeforeDays)
		return model
	}

	ctx := context.Background()
	plan := alertPlan(t, s, alert(7))
	resp := &resource.ModifyPlanResponse{Plan: plan}
	(&alertResource{}).ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: alertState(t, s, alert(30))}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan() diagnostics = %v", resp.Diagnostics)
	}

	if len(resp.RequiresReplace) != 0 {
		t.Errorf("ModifyPlan() requires replacing %v, want the condition updated in place", resp.RequiresReplace)
	}
}

// An alert firing on an event this version has never heard of cannot be read, and a read is what
// stands between the practitioner and destroying it, so the error has to hand them a way out rather
// than only asking for an upgrade they may not be able to take yet.
func TestSetStateFromAlertOnAnUnknownEvent(t *testing.T) {
	state := alertModel(t, nil)
	diags := (&alertResource{}).setStateFromAlert(context.Background(), &state, infisical.Alert{
		ID:           "alert-1",
		Name:         "Credentials expiring",
		ResourceType: alertResourceTypeIdentityAuthentication,
		EventType:    "identity.authentication.rotation",
	})

	if !diags.HasError() {
		t.Fatal("setStateFromAlert() with an unknown event type: no error, want one")
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, alertRefreshEscapeHatch) {
		t.Errorf("error detail = %q, want it to explain how to stop managing the alert", detail)
	}
}

func TestAlertEventFromBlocks(t *testing.T) {
	ctx := context.Background()
	s := alertSchema(t)

	model := alertModel(t, map[string]alertChannelModel{"security_team": {
		ID:      types.StringValue("channel-1"),
		Name:    types.StringValue("Security team"),
		Enabled: types.BoolValue(true),
		Email:   emailChannel(t, alertChannelRecipientModel{Type: types.StringValue(AlertRecipientTypeUser), ID: types.StringValue("user-1")}),
	}})

	event, known, diags := alertEventFromBlocks(ctx, alertPlan(t, s, model))
	if diags.HasError() {
		t.Fatalf("alertEventFromBlocks() diagnostics = %v", diags)
	}
	if !known {
		t.Fatal("alertEventFromBlocks() could not tell the event of an alert with a condition block")
	}
	if event.eventType != alertEventTypeIdentityAuthenticationExpiry {
		t.Errorf("event type = %q, want %q", event.eventType, alertEventTypeIdentityAuthenticationExpiry)
	}

	// A destroy plan, or a state the resource does not have yet, carries no condition at all.
	model.Expiry = nil
	if _, known, diags := alertEventFromBlocks(ctx, alertPlan(t, s, model)); known || diags.HasError() {
		t.Errorf("alertEventFromBlocks() without a condition block: known = %v, diagnostics = %v, want false and none", known, diags)
	}
}

func TestChannelTypeFromObject(t *testing.T) {
	s := alertSchema(t)

	channelObject := func(model alertChannelModel) types.Object {
		t.Helper()

		alert, ok := alertObject(t, s, alertModel(t, map[string]alertChannelModel{"channel": model})).(types.Object)
		if !ok {
			t.Fatal("the alert did not convert to an object")
		}

		channels, ok := alert.Attributes()["channels"].(types.Map)
		if !ok {
			t.Fatalf("channels is not a map, got %T", alert.Attributes()["channels"])
		}
		channel, ok := channels.Elements()["channel"].(types.Object)
		if !ok {
			t.Fatalf("channel is not an object, got %T", channels.Elements()["channel"])
		}
		return channel
	}

	cases := map[string]struct {
		channel alertChannelModel
		want    string
	}{
		"email":     {alertChannelModel{Email: emailChannel(t)}, AlertChannelTypeEmail},
		"slack":     {alertChannelModel{Slack: &alertSlackChannelModel{WebhookURL: types.StringNull()}}, AlertChannelTypeSlack},
		"webhook":   {alertChannelModel{Webhook: &alertWebhookChannelModel{URL: types.StringNull(), SigningSecret: types.StringNull()}}, AlertChannelTypeWebhook},
		"pagerduty": {alertChannelModel{PagerDuty: &alertPagerDutyChannelModel{IntegrationKey: types.StringNull()}}, AlertChannelTypePagerDuty},
		"none":      {alertChannelModel{}, ""},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, known := channelTypeFromObject(channelObject(c.channel))
			if !known {
				t.Fatal("channelTypeFromObject() reported an unknown type block, want the type")
			}
			if got != c.want {
				t.Errorf("channelTypeFromObject() = %q, want %q", got, c.want)
			}
		})
	}
}

// A block that is still unknown at plan time cannot be told apart from another type, so the caller
// has to be told rather than shown an empty type.
func TestChannelTypeFromObjectWithUnknownBlock(t *testing.T) {
	attributeTypes := map[string]attr.Type{"id": types.StringType, "enabled": types.BoolType}
	attributes := map[string]attr.Value{"id": types.StringValue("channel-1"), "enabled": types.BoolValue(true)}
	for _, channelType := range alertChannelTypes {
		attributeTypes[channelType] = types.ObjectType{AttrTypes: map[string]attr.Type{}}
		attributes[channelType] = types.ObjectNull(map[string]attr.Type{})
	}
	attributes[AlertChannelTypeSlack] = types.ObjectUnknown(map[string]attr.Type{})

	channel, diags := types.ObjectValue(attributeTypes, attributes)
	if diags.HasError() {
		t.Fatalf("building the channel object: %v", diags)
	}

	if _, known := channelTypeFromObject(channel); known {
		t.Error("channelTypeFromObject() reported a known type for a channel with an unknown block")
	}
}
