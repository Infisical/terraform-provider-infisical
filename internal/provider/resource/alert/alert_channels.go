package resource

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	AlertChannelTypeEmail     = "email"
	AlertChannelTypeSlack     = "slack"
	AlertChannelTypeWebhook   = "webhook"
	AlertChannelTypePagerDuty = "pagerduty"
)

var alertChannelTypes = []string{
	AlertChannelTypeEmail,
	AlertChannelTypeSlack,
	AlertChannelTypeWebhook,
	AlertChannelTypePagerDuty,
}

const (
	AlertRecipientTypeUser  = "user"
	AlertRecipientTypeGroup = "group"
)

const (
	maxChannelsPerAlert     = 10
	maxChannelKeyLength     = 255
	maxChannelNameLength    = 255
	maxRecipientsPerChannel = 20
)

const (
	channelConfigKeySlackWebhookURL   = "webhookUrl"
	channelConfigKeyWebhookURL        = "url"
	channelConfigKeySigningSecret     = "signingSecret"
	channelConfigKeyIntegrationKey    = "integrationKey"
	channelConfigKeyHasWebhookURL     = "hasWebhookUrl"
	channelConfigKeyHasSigningSecret  = "hasSigningSecret"
	channelConfigKeyHasIntegrationKey = "hasIntegrationKey"
)

var (
	slackWebhookUrlValidator = stringvalidator.RegexMatches(
		regexp.MustCompile(`^https://hooks\.slack\.com/\S+$`),
		"must be a Slack incoming webhook URL (example: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXX)",
	)

	pagerDutyIntegrationKeyValidator = stringvalidator.RegexMatches(
		regexp.MustCompile(`^[a-fA-F0-9]{32}$`),
		"must be a 32 character hex string",
	)
)

var alertChannelRecipientAttributeTypes = map[string]attr.Type{
	"type": types.StringType,
	"id":   types.StringType,
}

var alertChannelRecipientObjectType = types.ObjectType{AttrTypes: alertChannelRecipientAttributeTypes}

type alertChannelRecipientModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
}

type alertEmailChannelModel struct {
	Recipients types.Set `tfsdk:"recipients"`
}

type alertSlackChannelModel struct {
	WebhookURL types.String `tfsdk:"webhook_url"`
}

type alertWebhookChannelModel struct {
	URL           types.String `tfsdk:"url"`
	SigningSecret types.String `tfsdk:"signing_secret"`
}

type alertPagerDutyChannelModel struct {
	IntegrationKey types.String `tfsdk:"integration_key"`
}

type alertChannelModel struct {
	ID        types.String                `tfsdk:"id"`
	Name      types.String                `tfsdk:"name"`
	Enabled   types.Bool                  `tfsdk:"enabled"`
	Email     *alertEmailChannelModel     `tfsdk:"email"`
	Slack     *alertSlackChannelModel     `tfsdk:"slack"`
	Webhook   *alertWebhookChannelModel   `tfsdk:"webhook"`
	PagerDuty *alertPagerDutyChannelModel `tfsdk:"pagerduty"`
}

func alertChannelTypePaths() []path.Expression {
	paths := make([]path.Expression, 0, len(alertChannelTypes))
	for _, channelType := range alertChannelTypes {
		paths = append(paths, path.MatchRelative().AtName(channelType))
	}
	return paths
}

func (c alertChannelModel) channelType() string {
	switch {
	case c.Email != nil:
		return AlertChannelTypeEmail
	case c.Slack != nil:
		return AlertChannelTypeSlack
	case c.Webhook != nil:
		return AlertChannelTypeWebhook
	case c.PagerDuty != nil:
		return AlertChannelTypePagerDuty
	default:
		return ""
	}
}

func channelTypeFromObject(channel types.Object) (string, bool) {
	attributes := channel.Attributes()

	channelType := ""
	for _, name := range alertChannelTypes {
		block, ok := attributes[name]
		if !ok || block == nil {
			continue
		}
		if block.IsUnknown() {
			return "", false
		}
		if !block.IsNull() {
			channelType = name
		}
	}

	return channelType, true
}

func channelKeepsStoredID(planned, stored types.Object) bool {
	storedID, ok := stored.Attributes()["id"]
	if !ok || storedID == nil || storedID.IsNull() || storedID.IsUnknown() {
		return false
	}

	plannedType, plannedKnown := channelTypeFromObject(planned)
	storedType, storedKnown := channelTypeFromObject(stored)

	return plannedKnown && storedKnown && plannedType == storedType
}

func alertChannelsSchema() schema.MapNestedAttribute {
	return schema.MapNestedAttribute{
		Required: true,
		Description: fmt.Sprintf(
			"The channels the alert is delivered to, keyed by a name of your choosing. The key is what identifies a channel to Terraform and is never sent to Infisical, so renaming a channel updates it in place and it keeps the deliveries it has already made. Changing a key, on the other hand, deletes the channel and creates a new one, so the new channel notifies about everything that is still expiring, even if the old one already did. Each channel carries exactly one configuration block, and that block is what gives the channel its type. At least one and at most %d channels are allowed.",
			maxChannelsPerAlert,
		),
		Validators: []validator.Map{
			mapvalidator.SizeAtLeast(1),
			mapvalidator.SizeAtMost(maxChannelsPerAlert),
			mapvalidator.KeysAre(stringvalidator.LengthBetween(1, maxChannelKeyLength)),
		},
		NestedObject: schema.NestedAttributeObject{
			Validators: []validator.Object{
				objectvalidator.ExactlyOneOf(alertChannelTypePaths()...),
			},
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Computed:    true,
					Description: "The ID of the channel in Infisical.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"name": schema.StringAttribute{
					Required:    true,
					Description: "The name the channel is shown under in Infisical. Can be changed freely, and has to be unique within the alert.",
					Validators: []validator.String{
						stringvalidator.LengthBetween(1, maxChannelNameLength),
					},
				},
				"enabled": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(true),
					Description: "Whether the channel delivers notifications. Defaults to true.",
				},
				"email": schema.SingleNestedAttribute{
					Optional:    true,
					Description: "Notifies principals of your organization or project by email.",
					Attributes: map[string]schema.Attribute{
						"recipients": schema.SetNestedAttribute{
							Required:    true,
							Description: fmt.Sprintf("The principals to notify. At least one and at most %d are allowed.", maxRecipientsPerChannel),
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
								setvalidator.SizeAtMost(maxRecipientsPerChannel),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Required:    true,
										Description: "The type of the recipient. Options: " + AlertRecipientTypeUser + ", " + AlertRecipientTypeGroup + ".",
										Validators: []validator.String{
											stringvalidator.OneOf(AlertRecipientTypeUser, AlertRecipientTypeGroup),
										},
									},
									"id": schema.StringAttribute{
										Required:    true,
										Description: "The ID of the recipient user or group. The principal must belong to the scope the alert is created in.",
									},
								},
							},
						},
					},
				},
				"slack": schema.SingleNestedAttribute{
					Optional:    true,
					Description: "Posts notifications to a Slack channel through an incoming webhook.",
					Attributes: map[string]schema.Attribute{
						"webhook_url": schema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "The Slack incoming webhook URL notifications are posted to. Must be a https://hooks.slack.com URL. Write-only: it is never returned by the API, so it cannot be imported.",
							Validators: []validator.String{
								slackWebhookUrlValidator,
							},
						},
					},
				},
				"webhook": schema.SingleNestedAttribute{
					Optional:    true,
					Description: "Sends notifications to an HTTPS endpoint as a CloudEvents payload.",
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Required:    true,
							Description: "The HTTPS URL the CloudEvents payload is sent to.",
							Validators: []validator.String{
								infisicaltf.HttpsUrlValidator,
							},
						},
						"signing_secret": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "The secret used to sign the payload so the receiver can verify it. Write-only: it is never returned by the API, so an imported channel keeps the secret it was created with until one is set here.",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 256),
							},
						},
					},
				},
				"pagerduty": schema.SingleNestedAttribute{
					Optional:    true,
					Description: "Creates PagerDuty incidents through the Events API v2.",
					Attributes: map[string]schema.Attribute{
						"integration_key": schema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "The PagerDuty Events API v2 integration key incidents are created with, a 32 character hex string. Write-only: it is never returned by the API, so it cannot be imported.",
							Validators: []validator.String{
								pagerDutyIntegrationKeyValidator,
							},
						},
					},
				},
			},
		},
	}
}

func sortedChannelKeys(channels map[string]alertChannelModel) []string {
	keys := make([]string, 0, len(channels))
	for key := range channels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func reusedChannelID(channel alertChannelModel, priorChannel alertChannelModel, hadPrior bool) string {
	if !hadPrior || priorChannel.ID.IsNull() || priorChannel.ID.IsUnknown() {
		return ""
	}
	if priorChannel.channelType() != channel.channelType() {
		return ""
	}
	return priorChannel.ID.ValueString()
}

func validateChannelNamesAreUnique(ctx context.Context, config tfsdk.Config) diag.Diagnostics {
	var diags diag.Diagnostics

	var channels types.Map
	diags.Append(config.GetAttribute(ctx, path.Root("channels"), &channels)...)
	if diags.HasError() || channels.IsNull() || channels.IsUnknown() {
		return diags
	}

	keysByName := make(map[string][]string, len(channels.Elements()))
	for key, element := range channels.Elements() {
		channel, ok := element.(types.Object)
		if !ok || channel.IsNull() || channel.IsUnknown() {
			continue
		}
		name, ok := channel.Attributes()["name"].(types.String)
		if !ok || name.IsNull() || name.IsUnknown() {
			continue
		}
		keysByName[name.ValueString()] = append(keysByName[name.ValueString()], key)
	}

	names := make([]string, 0, len(keysByName))
	for name := range keysByName {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		keys := keysByName[name]
		if len(keys) < 2 {
			continue
		}
		sort.Strings(keys)

		quoted := make([]string, 0, len(keys))
		for _, key := range keys {
			quoted = append(quoted, fmt.Sprintf("%q", key))
		}

		for _, key := range keys[1:] {
			diags.AddAttributeError(
				path.Root("channels").AtMapKey(key).AtName("name"),
				"Duplicate channel name",
				fmt.Sprintf(
					"Channels %s are all named %q. A name identifies a channel in Infisical, so it has to be unique within the alert.",
					strings.Join(quoted, ", "), name,
				),
			)
		}
	}

	return diags
}

func alertChannelsToAPIInput(ctx context.Context, channels map[string]alertChannelModel, prior map[string]alertChannelModel) ([]infisical.AlertChannelInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	inputs := make([]infisical.AlertChannelInput, 0, len(channels))
	for _, key := range sortedChannelKeys(channels) {
		channel := channels[key]

		recipients := make([]infisical.AlertChannelRecipient, 0)
		if channel.Email != nil && !channel.Email.Recipients.IsNull() && !channel.Email.Recipients.IsUnknown() {
			var recipientModels []alertChannelRecipientModel
			diags.Append(channel.Email.Recipients.ElementsAs(ctx, &recipientModels, false)...)
			if diags.HasError() {
				return nil, diags
			}

			for _, recipient := range recipientModels {
				recipients = append(recipients, infisical.AlertChannelRecipient{
					PrincipalType: recipient.Type.ValueString(),
					PrincipalID:   recipient.ID.ValueString(),
				})
			}
		}

		priorChannel, hadPrior := prior[key]
		reusedID := reusedChannelID(channel, priorChannel, hadPrior)

		config := map[string]any{}
		switch {
		case channel.Slack != nil:
			config[channelConfigKeySlackWebhookURL] = channel.Slack.WebhookURL.ValueString()
		case channel.Webhook != nil:
			config[channelConfigKeyWebhookURL] = channel.Webhook.URL.ValueString()

			// Only a channel that is being updated in place can have a stored secret to clear.
			priorHadSigningSecret := reusedID != "" && priorChannel.Webhook != nil && !priorChannel.Webhook.SigningSecret.IsNull()
			switch {
			case !channel.Webhook.SigningSecret.IsNull() && !channel.Webhook.SigningSecret.IsUnknown():
				config[channelConfigKeySigningSecret] = channel.Webhook.SigningSecret.ValueString()
			case priorHadSigningSecret:
				// The secret was deliberately removed from the configuration, and an empty string is
				// what tells the API to drop the stored one.
				config[channelConfigKeySigningSecret] = ""
			}
			// Otherwise the field is left out entirely, which keeps whatever is stored. Terraform
			// never had the value to begin with (an imported channel, say), so clearing it would
			// destroy a secret the configuration never claimed to own.
		case channel.PagerDuty != nil:
			config[channelConfigKeyIntegrationKey] = channel.PagerDuty.IntegrationKey.ValueString()
		}

		input := infisical.AlertChannelInput{
			Name:        channel.Name.ValueString(),
			ChannelType: channel.channelType(),
			Enabled:     channel.Enabled.ValueBool(),
			Config:      config,
			Recipients:  recipients,
		}

		if reusedID != "" {
			input.ID = &reusedID
		}

		inputs = append(inputs, input)
	}

	return inputs, diags
}

func alertChannelIDsFromAPI(channels map[string]alertChannelModel, prior map[string]alertChannelModel, apiChannels []infisical.AlertChannel) (map[string]alertChannelModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	sorted := sortAPIChannels(apiChannels)

	byID := make(map[string]infisical.AlertChannel, len(sorted))
	for _, apiChannel := range sorted {
		byID[apiChannel.ID] = apiChannel
	}

	keys := sortedChannelKeys(channels)
	withIDs := make(map[string]alertChannelModel, len(channels))
	claimed := make(map[string]bool, len(sorted))

	for _, key := range keys {
		priorChannel, hadPrior := prior[key]
		reusedID := reusedChannelID(channels[key], priorChannel, hadPrior)
		if reusedID == "" {
			continue
		}
		if _, ok := byID[reusedID]; !ok {
			continue
		}

		channel := channels[key]
		channel.ID = types.StringValue(reusedID)
		withIDs[key] = channel
		claimed[reusedID] = true
	}

	for _, key := range keys {
		if _, done := withIDs[key]; done {
			continue
		}
		channel := channels[key]

		id := ""
		for _, apiChannel := range sorted {
			if claimed[apiChannel.ID] {
				continue
			}
			if apiChannel.Name == channel.Name.ValueString() && apiChannel.ChannelType == channel.channelType() {
				id = apiChannel.ID
				break
			}
		}

		if id == "" {
			channel.ID = types.StringNull()
			withIDs[key] = channel
			diags.AddAttributeError(
				path.Root("channels").AtMapKey(key),
				"Missing channel in Infisical's response",
				fmt.Sprintf("Infisical did not return a channel named %q after writing the alert, so Terraform cannot record that channel's ID and will replace the channel on the next apply. Please report this issue to the provider developers.", channel.Name.ValueString()),
			)
			continue
		}

		channel.ID = types.StringValue(id)
		withIDs[key] = channel
		claimed[id] = true
	}

	return withIDs, diags
}

func sortAPIChannels(apiChannels []infisical.AlertChannel) []infisical.AlertChannel {
	sorted := make([]infisical.AlertChannel, len(apiChannels))
	copy(sorted, apiChannels)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

func alertChannelsFromAPI(ctx context.Context, apiChannels []infisical.AlertChannel, stateChannels map[string]alertChannelModel) (map[string]alertChannelModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	priorByID := make(map[string]alertChannelModel, len(stateChannels))
	keysByID := make(map[string]string, len(stateChannels))
	for _, key := range sortedChannelKeys(stateChannels) {
		stateChannel := stateChannels[key]
		if id := stateChannel.ID.ValueString(); id != "" {
			priorByID[id] = stateChannel
			keysByID[id] = key
		}
	}

	sorted := sortAPIChannels(apiChannels)

	taken := make(map[string]bool, len(sorted))
	for _, apiChannel := range sorted {
		if key, ok := keysByID[apiChannel.ID]; ok {
			taken[key] = true
		}
	}

	refreshed := make(map[string]alertChannelModel, len(sorted))
	for _, apiChannel := range sorted {
		key, tracked := keysByID[apiChannel.ID]
		if !tracked {
			key = apiChannel.Name
			if key == "" || taken[key] {
				key = apiChannel.ID
			}
			taken[key] = true
		}

		var priorState *alertChannelModel
		if prior, ok := priorByID[apiChannel.ID]; ok {
			priorState = &prior
		}

		channel, channelDiags := alertChannelFromAPI(ctx, apiChannel, priorState)
		diags.Append(channelDiags...)
		if diags.HasError() {
			return nil, diags
		}
		refreshed[key] = channel
	}

	return refreshed, diags
}

func alertChannelFromAPI(ctx context.Context, apiChannel infisical.AlertChannel, priorState *alertChannelModel) (alertChannelModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	channel := alertChannelModel{
		ID:      types.StringValue(apiChannel.ID),
		Name:    types.StringValue(apiChannel.Name),
		Enabled: types.BoolValue(apiChannel.Enabled),
	}

	switch apiChannel.ChannelType {
	case AlertChannelTypeEmail:
		recipientModels := make([]alertChannelRecipientModel, 0, len(apiChannel.Recipients))
		for _, recipient := range apiChannel.Recipients {
			recipientModels = append(recipientModels, alertChannelRecipientModel{
				Type: types.StringValue(recipient.PrincipalType),
				ID:   types.StringValue(recipient.PrincipalID),
			})
		}

		recipients, recipientDiags := types.SetValueFrom(ctx, alertChannelRecipientObjectType, recipientModels)
		diags.Append(recipientDiags...)
		if diags.HasError() {
			return channel, diags
		}
		channel.Email = &alertEmailChannelModel{Recipients: recipients}
	case AlertChannelTypeSlack:
		channel.Slack = &alertSlackChannelModel{
			WebhookURL: preservedChannelSecret(apiChannel.Config, channelConfigKeyHasWebhookURL, func() types.String {
				if priorState != nil && priorState.Slack != nil {
					return priorState.Slack.WebhookURL
				}
				return types.StringNull()
			}),
		}
	case AlertChannelTypeWebhook:
		channel.Webhook = &alertWebhookChannelModel{
			URL: types.StringValue(channelConfigString(apiChannel.Config, channelConfigKeyWebhookURL)),
			SigningSecret: preservedChannelSecret(apiChannel.Config, channelConfigKeyHasSigningSecret, func() types.String {
				if priorState != nil && priorState.Webhook != nil {
					return priorState.Webhook.SigningSecret
				}
				return types.StringNull()
			}),
		}
	case AlertChannelTypePagerDuty:
		channel.PagerDuty = &alertPagerDutyChannelModel{
			IntegrationKey: preservedChannelSecret(apiChannel.Config, channelConfigKeyHasIntegrationKey, func() types.String {
				if priorState != nil && priorState.PagerDuty != nil {
					return priorState.PagerDuty.IntegrationKey
				}
				return types.StringNull()
			}),
		}
	default:
		diags.AddError(
			"Unsupported alert channel",
			fmt.Sprintf(
				"Alert channel %q delivers over %q, which this provider does not know how to manage. Please upgrade the provider.",
				apiChannel.Name, apiChannel.ChannelType,
			),
		)
	}

	return channel, diags
}

func preservedChannelSecret(config map[string]any, hasKey string, fromState func() types.String) types.String {
	if stored, ok := config[hasKey].(bool); ok && !stored {
		return types.StringNull()
	}
	return fromState()
}

func channelConfigString(config map[string]any, key string) string {
	value, _ := config[key].(string)
	return value
}
