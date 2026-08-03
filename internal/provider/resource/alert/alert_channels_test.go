package resource

import (
	"context"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func emailChannel(t *testing.T, recipients ...alertChannelRecipientModel) *alertEmailChannelModel {
	t.Helper()

	set, diags := types.SetValueFrom(context.Background(), alertChannelRecipientObjectType, recipients)
	if diags.HasError() {
		t.Fatalf("building recipients set: %v", diags)
	}
	return &alertEmailChannelModel{Recipients: set}
}

func TestChannelType(t *testing.T) {
	cases := map[string]struct {
		channel alertChannelModel
		want    string
	}{
		"email":     {alertChannelModel{Email: emailChannel(t)}, AlertChannelTypeEmail},
		"slack":     {alertChannelModel{Slack: &alertSlackChannelModel{}}, AlertChannelTypeSlack},
		"webhook":   {alertChannelModel{Webhook: &alertWebhookChannelModel{}}, AlertChannelTypeWebhook},
		"pagerduty": {alertChannelModel{PagerDuty: &alertPagerDutyChannelModel{}}, AlertChannelTypePagerDuty},
		"none":      {alertChannelModel{}, ""},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := c.channel.channelType(); got != c.want {
				t.Errorf("channelType() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestAlertChannelsToAPIInput(t *testing.T) {
	channels := map[string]alertChannelModel{
		"Security team": {
			Enabled: types.BoolValue(true),
			Email: emailChannel(t, alertChannelRecipientModel{
				Type: types.StringValue(AlertRecipientTypeGroup),
				ID:   types.StringValue("group-1"),
			}),
		},
		"Internal automation": {
			Enabled: types.BoolValue(false),
			Webhook: &alertWebhookChannelModel{
				URL:           types.StringValue("https://example.com/alerts"),
				SigningSecret: types.StringNull(),
			},
		},
	}

	// Only the email channel is already in state, so only it has an ID to reuse.
	prior := map[string]alertChannelModel{
		"Security team": {
			ID:    types.StringValue("channel-1"),
			Email: emailChannel(t),
		},
	}

	inputs, diags := alertChannelsToAPIInput(context.Background(), channels, prior)
	if diags.HasError() {
		t.Fatalf("alertChannelsToAPIInput() diagnostics = %v", diags)
	}

	if len(inputs) != 2 {
		t.Fatalf("alertChannelsToAPIInput() returned %d channels, want 2", len(inputs))
	}

	// Sorted by name, so the order does not depend on Go's map iteration.
	if inputs[0].Name != "Internal automation" || inputs[1].Name != "Security team" {
		t.Fatalf("channels = %q, %q, want them ordered by name", inputs[0].Name, inputs[1].Name)
	}

	webhook, email := inputs[0], inputs[1]

	if email.ChannelType != AlertChannelTypeEmail {
		t.Errorf("channel type = %q, want it derived from the email block", email.ChannelType)
	}
	if email.ID == nil || *email.ID != "channel-1" {
		t.Errorf("channel already in state did not reuse its ID, got %v", email.ID)
	}
	if len(email.Recipients) != 1 || email.Recipients[0].PrincipalID != "group-1" {
		t.Errorf("recipients = %v, want a single group-1 recipient", email.Recipients)
	}
	if len(email.Config) != 0 {
		t.Errorf("email config = %v, want it empty", email.Config)
	}

	if webhook.ChannelType != AlertChannelTypeWebhook {
		t.Errorf("channel type = %q, want it derived from the webhook block", webhook.ChannelType)
	}
	if webhook.ID != nil {
		t.Errorf("channel that is not in state got ID %v, want none", *webhook.ID)
	}
	if webhook.Enabled {
		t.Error("channel enabled = true, want false")
	}
	if webhook.Recipients == nil {
		t.Error("recipients = nil, want an empty slice so the API does not receive null")
	}
	if got := webhook.Config[channelConfigKeyWebhookURL]; got != "https://example.com/alerts" {
		t.Errorf("webhook url = %v, want https://example.com/alerts", got)
	}
	// Terraform never held a signing secret for this channel, so the field is left out and whatever
	// the API has stored survives.
	if _, ok := webhook.Config[channelConfigKeySigningSecret]; ok {
		t.Errorf("signing secret = %v, want the field to be absent", webhook.Config[channelConfigKeySigningSecret])
	}
}

// Editing a channel must update it in place, because a replaced channel loses the delivery history
// the alerting engine dedups against and would notify about everything that is still expiring.
func TestAlertChannelsToAPIInputUpdatesInPlace(t *testing.T) {
	prior := map[string]alertChannelModel{"Platform Slack": {
		ID:      types.StringValue("channel-1"),
		Enabled: types.BoolValue(true),
		Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
	}}

	edited := map[string]alertChannelModel{"Platform Slack": {
		Enabled: types.BoolValue(false),
		Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/xyz")},
	}}

	inputs, diags := alertChannelsToAPIInput(context.Background(), edited, prior)
	if diags.HasError() {
		t.Fatalf("alertChannelsToAPIInput() diagnostics = %v", diags)
	}

	if inputs[0].ID == nil || *inputs[0].ID != "channel-1" {
		t.Fatalf("edited channel ID = %v, want channel-1 so the channel is updated in place", inputs[0].ID)
	}
	if got := inputs[0].Config[channelConfigKeySlackWebhookURL]; got != "https://hooks.slack.com/services/xyz" {
		t.Errorf("webhook url = %v, want the new one", got)
	}
}

// A renamed channel is a new key, so it is created and the old one is left out for the API to delete.
func TestAlertChannelsToAPIInputReplacesOnRename(t *testing.T) {
	prior := map[string]alertChannelModel{"Platform Slack": {
		ID:    types.StringValue("channel-1"),
		Slack: &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
	}}

	renamed := map[string]alertChannelModel{"Platform alerts": {
		Enabled: types.BoolValue(true),
		Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
	}}

	inputs, diags := alertChannelsToAPIInput(context.Background(), renamed, prior)
	if diags.HasError() {
		t.Fatalf("alertChannelsToAPIInput() diagnostics = %v", diags)
	}

	if len(inputs) != 1 || inputs[0].Name != "Platform alerts" {
		t.Fatalf("inputs = %v, want only the renamed channel", inputs)
	}
	if inputs[0].ID != nil {
		t.Errorf("renamed channel ID = %v, want none so the old channel is deleted and this one created", *inputs[0].ID)
	}
}

// The API refuses to change a channel's type, so swapping the block has to be sent as a replacement.
func TestAlertChannelsToAPIInputReplacesOnTypeChange(t *testing.T) {
	prior := map[string]alertChannelModel{"On-call": {
		ID:    types.StringValue("channel-1"),
		Slack: &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
	}}

	retyped := map[string]alertChannelModel{"On-call": {
		Enabled:   types.BoolValue(true),
		PagerDuty: &alertPagerDutyChannelModel{IntegrationKey: types.StringValue("00000000000000000000000000000000")},
	}}

	inputs, diags := alertChannelsToAPIInput(context.Background(), retyped, prior)
	if diags.HasError() {
		t.Fatalf("alertChannelsToAPIInput() diagnostics = %v", diags)
	}

	if inputs[0].ID != nil {
		t.Errorf("retyped channel ID = %v, want none so the API replaces it instead of rejecting the write", *inputs[0].ID)
	}
	if inputs[0].ChannelType != AlertChannelTypePagerDuty {
		t.Errorf("channel type = %q, want pagerduty", inputs[0].ChannelType)
	}
}

func TestAlertChannelsToAPIInputSigningSecret(t *testing.T) {
	webhookChannel := func(id types.String, signingSecret types.String) map[string]alertChannelModel {
		return map[string]alertChannelModel{"Internal automation": {
			ID:      id,
			Enabled: types.BoolValue(true),
			Webhook: &alertWebhookChannelModel{
				URL:           types.StringValue("https://example.com/alerts"),
				SigningSecret: signingSecret,
			},
		}}
	}

	cases := []struct {
		name    string
		planned types.String
		prior   types.String
		want    any // nil means the field must be absent
	}{
		{
			name:    "configured secret is sent",
			planned: types.StringValue("sekret"),
			prior:   types.StringNull(),
			want:    "sekret",
		},
		{
			// The only way to clear a stored secret, and only safe when Terraform held one before.
			name:    "secret removed from the configuration is cleared",
			planned: types.StringNull(),
			prior:   types.StringValue("sekret"),
			want:    "",
		},
		{
			// An imported channel: Terraform cannot know the value, so it must not clobber it.
			name:    "secret Terraform never held is left alone",
			planned: types.StringNull(),
			prior:   types.StringNull(),
			want:    nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prior := webhookChannel(types.StringValue("channel-1"), c.prior)
			planned := webhookChannel(types.StringNull(), c.planned)

			inputs, diags := alertChannelsToAPIInput(context.Background(), planned, prior)
			if diags.HasError() {
				t.Fatalf("alertChannelsToAPIInput() diagnostics = %v", diags)
			}

			got, ok := inputs[0].Config[channelConfigKeySigningSecret]
			if c.want == nil {
				if ok {
					t.Errorf("signing secret = %v, want the field to be absent", got)
				}
				return
			}
			if !ok || got != c.want {
				t.Errorf("signing secret = %v (present = %v), want %q", got, ok, c.want)
			}
		})
	}
}

func TestAlertChannelIDsFromAPI(t *testing.T) {
	channels := map[string]alertChannelModel{
		"Platform Slack": {Slack: &alertSlackChannelModel{}},
		"Security team":  {Email: emailChannel(t)},
	}

	apiChannels := []infisical.AlertChannel{
		{ID: "channel-2", Name: "Security team", ChannelType: AlertChannelTypeEmail},
		{ID: "channel-1", Name: "Platform Slack", ChannelType: AlertChannelTypeSlack},
	}

	withIDs, diags := alertChannelIDsFromAPI(channels, apiChannels)
	if diags.HasError() {
		t.Fatalf("alertChannelIDsFromAPI() diagnostics = %v", diags)
	}

	if got := withIDs["Platform Slack"].ID.ValueString(); got != "channel-1" {
		t.Errorf("Platform Slack ID = %q, want channel-1", got)
	}
	if got := withIDs["Security team"].ID.ValueString(); got != "channel-2" {
		t.Errorf("Security team ID = %q, want channel-2", got)
	}

}

// The alert has already been written by the time the IDs are read back, so a channel missing from the
// response is still recorded, without an ID, rather than costing the caller the whole alert.
func TestAlertChannelIDsFromAPIWithAChannelMissing(t *testing.T) {
	channels := map[string]alertChannelModel{
		"Platform Slack": {Slack: &alertSlackChannelModel{}},
		"Security team":  {Email: emailChannel(t)},
	}

	apiChannels := []infisical.AlertChannel{
		{ID: "channel-2", Name: "Security team", ChannelType: AlertChannelTypeEmail},
	}

	withIDs, diags := alertChannelIDsFromAPI(channels, apiChannels)
	if !diags.HasError() {
		t.Error("alertChannelIDsFromAPI() with a channel missing from the response: no error, want one")
	}

	if len(withIDs) != 2 {
		t.Fatalf("alertChannelIDsFromAPI() returned %d channels, want both so the alert is still recorded", len(withIDs))
	}
	if !withIDs["Platform Slack"].ID.IsNull() {
		t.Errorf("missing channel ID = %v, want null", withIDs["Platform Slack"].ID)
	}
	if got := withIDs["Security team"].ID.ValueString(); got != "channel-2" {
		t.Errorf("channel that was returned has ID %q, want channel-2", got)
	}
}

func TestAlertChannelsFromAPI(t *testing.T) {
	stateChannels := map[string]alertChannelModel{
		"Platform Slack": {
			ID:      types.StringValue("channel-1"),
			Enabled: types.BoolValue(true),
			Slack:   &alertSlackChannelModel{WebhookURL: types.StringValue("https://hooks.slack.com/services/abc")},
		},
		"Internal automation": {
			ID:      types.StringValue("channel-2"),
			Enabled: types.BoolValue(true),
			Webhook: &alertWebhookChannelModel{
				URL:           types.StringValue("https://example.com/alerts"),
				SigningSecret: types.StringValue("signing-secret"),
			},
		},
		"Deleted remotely": {
			ID:      types.StringValue("channel-deleted"),
			Enabled: types.BoolValue(true),
			Email:   emailChannel(t),
		},
	}

	// The API returns no secret values, one channel renamed outside Terraform, one channel Terraform
	// has never seen, and nothing at all for the channel that was deleted remotely.
	apiChannels := []infisical.AlertChannel{
		{
			ID:          "channel-2",
			Name:        "Internal automation",
			ChannelType: AlertChannelTypeWebhook,
			Enabled:     true,
			Config: map[string]any{
				channelConfigKeyWebhookURL:       "https://example.com/other-alerts",
				channelConfigKeyHasSigningSecret: false,
			},
		},
		{
			ID:          "channel-1",
			Name:        "Renamed in the UI",
			ChannelType: AlertChannelTypeSlack,
			Enabled:     false,
			Config:      map[string]any{channelConfigKeyHasWebhookURL: true},
		},
		{
			ID:          "channel-3",
			Name:        "Security team",
			ChannelType: AlertChannelTypeEmail,
			Enabled:     true,
			Config:      map[string]any{},
			Recipients:  []infisical.AlertChannelRecipient{{PrincipalType: AlertRecipientTypeUser, PrincipalID: "user-1"}},
		},
	}

	refreshed, diags := alertChannelsFromAPI(context.Background(), apiChannels, stateChannels)
	if diags.HasError() {
		t.Fatalf("alertChannelsFromAPI() diagnostics = %v", diags)
	}

	if len(refreshed) != 3 {
		t.Fatalf("alertChannelsFromAPI() returned %d channels, want 3", len(refreshed))
	}
	if _, ok := refreshed["Deleted remotely"]; ok {
		t.Error("channel deleted in Infisical is still in state, want it dropped as drift")
	}

	// Keyed by the name Infisical holds, so a rename there shows up as a different key. The channel
	// is still matched by ID, so the secret Terraform holds for it is carried over.
	if _, ok := refreshed["Platform Slack"]; ok {
		t.Error("channel renamed in Infisical kept its old key, want the rename to show as drift")
	}
	slack, ok := refreshed["Renamed in the UI"]
	if !ok {
		t.Fatalf("renamed channel is missing, got keys %v", refreshed)
	}
	if slack.Enabled.ValueBool() {
		t.Error("slack channel enabled = true, want the API's false")
	}
	if got := slack.Slack.WebhookURL.ValueString(); got != "https://hooks.slack.com/services/abc" {
		t.Errorf("slack webhook url = %q, want the value carried over from state by ID", got)
	}
	if slack.Email != nil || slack.Webhook != nil || slack.PagerDuty != nil {
		t.Error("slack channel carries another type's block, want only the slack one")
	}

	webhook := refreshed["Internal automation"]
	if got := webhook.Webhook.URL.ValueString(); got != "https://example.com/other-alerts" {
		t.Errorf("webhook url = %q, want the API's value", got)
	}
	// The API reports no stored signing secret, so the one in state is dropped as drift.
	if !webhook.Webhook.SigningSecret.IsNull() {
		t.Errorf("signing secret = %v, want null", webhook.Webhook.SigningSecret)
	}

	// A channel Terraform has never seen shows up under its name.
	email, ok := refreshed["Security team"]
	if !ok {
		t.Fatalf("channel added outside Terraform is missing, got keys %v", refreshed)
	}
	if got := email.ID.ValueString(); got != "channel-3" {
		t.Errorf("ID = %q, want channel-3", got)
	}
	var recipients []alertChannelRecipientModel
	if diags := email.Email.Recipients.ElementsAs(context.Background(), &recipients, false); diags.HasError() {
		t.Fatalf("reading recipients: %v", diags)
	}
	if len(recipients) != 1 || recipients[0].ID.ValueString() != "user-1" {
		t.Errorf("recipients = %v, want a single user-1 recipient", recipients)
	}
}

// Infisical does not require an alert's channel names to be unique, so a colliding channel created
// outside Terraform falls back to its ID for a key rather than displacing the one already keyed.
func TestAlertChannelsFromAPIKeysCollidingChannelByID(t *testing.T) {
	apiChannels := []infisical.AlertChannel{
		{
			ID:          "channel-2",
			Name:        "On-call",
			ChannelType: AlertChannelTypePagerDuty,
			Enabled:     true,
			Config:      map[string]any{channelConfigKeyHasIntegrationKey: true},
		},
		{
			ID:          "channel-1",
			Name:        "On-call",
			ChannelType: AlertChannelTypeSlack,
			Enabled:     true,
			Config:      map[string]any{channelConfigKeyHasWebhookURL: true},
		},
	}

	refreshed, diags := alertChannelsFromAPI(context.Background(), apiChannels, nil)
	if diags.HasError() {
		t.Fatalf("alertChannelsFromAPI() diagnostics = %v", diags)
	}

	if len(refreshed) != 2 {
		t.Fatalf("alertChannelsFromAPI() returned %d channels, want 2", len(refreshed))
	}
	// Sorted by ID within the shared name, so the keys do not depend on the API's ordering.
	if got := refreshed["On-call"].ID.ValueString(); got != "channel-1" {
		t.Errorf("channel keyed On-call = %q, want the lowest ID (channel-1)", got)
	}
	if _, ok := refreshed["channel-2"]; !ok {
		t.Errorf("colliding channel is missing, got keys %v", refreshed)
	}
}

// A channel type the provider does not know cannot be put into state as a channel with no type at
// all, so it is reported the way an unknown event type is.
func TestAlertChannelsFromAPIRejectsUnknownChannelTypes(t *testing.T) {
	apiChannels := []infisical.AlertChannel{{
		ID:          "channel-1",
		Name:        "Teams",
		ChannelType: "microsoft-teams",
		Enabled:     true,
		Config:      map[string]any{},
	}}

	if _, diags := alertChannelsFromAPI(context.Background(), apiChannels, nil); !diags.HasError() {
		t.Error("alertChannelsFromAPI() with an unknown channel type: no error, want one")
	}
}

func TestAlertChannelsFromAPIDropsRemovedChannels(t *testing.T) {
	stateChannels := map[string]alertChannelModel{"Removed remotely": {
		ID:      types.StringValue("channel-1"),
		Enabled: types.BoolValue(true),
		Email:   emailChannel(t),
	}}

	refreshed, diags := alertChannelsFromAPI(context.Background(), nil, stateChannels)
	if diags.HasError() {
		t.Fatalf("alertChannelsFromAPI() diagnostics = %v", diags)
	}

	if len(refreshed) != 0 {
		t.Errorf("alertChannelsFromAPI() returned %d channels, want none", len(refreshed))
	}
}

// Import has no prior state, so every channel is keyed by its name and carries no secrets.
func TestAlertChannelsFromAPIImport(t *testing.T) {
	apiChannels := []infisical.AlertChannel{{
		ID:          "channel-1",
		Name:        "Platform Slack",
		ChannelType: AlertChannelTypeSlack,
		Enabled:     true,
		Config:      map[string]any{channelConfigKeyHasWebhookURL: true},
	}}

	refreshed, diags := alertChannelsFromAPI(context.Background(), apiChannels, nil)
	if diags.HasError() {
		t.Fatalf("alertChannelsFromAPI() diagnostics = %v", diags)
	}

	channel, ok := refreshed["Platform Slack"]
	if !ok {
		t.Fatalf("imported channel is not keyed by its name, got keys %v", refreshed)
	}
	if got := channel.ID.ValueString(); got != "channel-1" {
		t.Errorf("ID = %q, want channel-1", got)
	}
	if !channel.Slack.WebhookURL.IsNull() {
		t.Errorf("webhook url = %v, want null since the API never returns it", channel.Slack.WebhookURL)
	}
}

func TestSortedChannelNames(t *testing.T) {
	names := sortedChannelNames(map[string]alertChannelModel{"c": {}, "a": {}, "b": {}})
	if strings.Join(names, ",") != "a,b,c" {
		t.Errorf("sortedChannelNames() = %v, want them sorted", names)
	}
}
