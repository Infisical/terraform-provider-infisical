package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	minAlertBeforeDays             = 1
	maxLegacyExpiryAlertBeforeDays = 90
	maxAlertFilterIDs              = 100
)

var certificateSources = []string{"issued", "discovered", "imported"}

type alertConditionModel struct {
	AlertBeforeDays types.Int64 `tfsdk:"alert_before_days"`
	DailyReminder   types.Bool  `tfsdk:"daily_reminder"`
	ApplicationIDs  types.Set   `tfsdk:"application_ids"`
	ProfileIDs      types.Set   `tfsdk:"profile_ids"`
	Sources         types.Set   `tfsdk:"sources"`
	SignerIDs       types.Set   `tfsdk:"signer_ids"`
}

type expiryConditionModel struct {
	AlertBeforeDays types.Int64 `tfsdk:"alert_before_days"`
	DailyReminder   types.Bool  `tfsdk:"daily_reminder"`
}

type alertConditionPayload struct {
	AlertBefore    *string  `json:"alertBefore,omitempty"`
	DailyReminder  *bool    `json:"dailyReminder,omitempty"`
	ApplicationIDs []string `json:"applicationIds,omitempty"`
	ProfileIDs     []string `json:"profileIds,omitempty"`
	Sources        []string `json:"sources,omitempty"`
	SignerIDs      []string `json:"signerIds,omitempty"`
}

func eventsWhere(match func(alertEvent) bool) string {
	names := []string{}
	for _, event := range alertEvents {
		if match(event) {
			names = append(names, event.eventType)
		}
	}
	return strings.Join(names, ", ")
}

func alertConditionSchema() schema.SingleNestedAttribute {
	idsAttribute := func(description string) schema.SetAttribute {
		return schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: fmt.Sprintf("%s Leave it out to cover all of them. At most %d.", description, maxAlertFilterIDs),
			Validators: []validator.Set{
				setvalidator.SizeBetween(1, maxAlertFilterIDs),
				setvalidator.ValueStringsAre(infisicaltf.UuidValidator),
			},
		}
	}
	filterEvents := func(name string) string {
		return eventsWhere(func(event alertEvent) bool { return event.takesFilter(name) })
	}

	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "The settings of the event the alert fires on. Which ones apply depends on event_type, and events without settings take no condition.",
		Attributes: map[string]schema.Attribute{
			"alert_before_days": schema.Int64Attribute{
				Optional:    true,
				Description: fmt.Sprintf("How many days before expiry the alert fires. Required for these events, and not used by the others: %s. Must be at least %d, and at most %s.", eventsWhere(alertEvent.firesBeforeExpiry), minAlertBeforeDays, describeAlertBeforeDaysLimits()),
				Validators: []validator.Int64{
					int64validator.AtLeast(minAlertBeforeDays),
				},
			},
			"daily_reminder": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, sends a reminder every day until expiry. Defaults to false. Only for the events that take alert_before_days.",
			},
			"application_ids": idsAttribute("Only certificates in these Certificate Manager applications. Only for these events: " + filterEvents("application_ids") + "."),
			"profile_ids":     idsAttribute("Only certificates issued from these certificate profiles. Only for these events: " + filterEvents("profile_ids") + "."),
			"sources": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Only certificates with these sources. Options: " + strings.Join(certificateSources, ", ") + ". Leave it out to cover every source. Only for these events: " + filterEvents("sources") + ".",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(certificateSources...)),
				},
			},
			"signer_ids": idsAttribute("Only certificates of these code signing signers, by ID. To get a signer's ID, open it under Code Signing > Signers in Certificate Manager and select Options > Copy Signer ID. Only for these events: " + filterEvents("signer_ids") + "."),
		},
	}
}

func legacyExpirySchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:           true,
		Description:        fmt.Sprintf("Fires before the watched resource expires. Only for %s alerts. Deprecated: use event_type and condition instead.", alertResourceTypeIdentityAuthentication),
		DeprecationMessage: "Use event_type = \"expiry\" with condition = { alert_before_days = ... } instead.",
		Attributes: map[string]schema.Attribute{
			"alert_before_days": schema.Int64Attribute{
				Required:    true,
				Description: fmt.Sprintf("How many days before the watched resource expires the alert fires. Must be between %d and %d.", minAlertBeforeDays, maxLegacyExpiryAlertBeforeDays),
				Validators: []validator.Int64{
					int64validator.Between(minAlertBeforeDays, maxLegacyExpiryAlertBeforeDays),
				},
			},
			"daily_reminder": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to keep notifying once a day until the watched resource expires, instead of notifying once. Defaults to false.",
			},
		},
	}
}

func alertEventFromModel(model alertResourceModel) (alertEvent, bool) {
	if model.Expiry != nil {
		return alertEventForEventType(alertEventTypeIdentityAuthenticationExpiry)
	}
	if model.EventType.IsNull() || model.EventType.IsUnknown() {
		return alertEvent{}, false
	}
	return resolveAlertEvent(model.ResourceType.ValueString(), model.EventType.ValueString())
}

func withExpiry(payload alertConditionPayload, alertBeforeDays types.Int64, dailyReminder types.Bool) alertConditionPayload {
	alertBefore := fmt.Sprintf("%dd", alertBeforeDays.ValueInt64())
	reminder := dailyReminder.ValueBool()
	payload.AlertBefore, payload.DailyReminder = &alertBefore, &reminder
	return payload
}

func alertConditionForAPI(event alertEvent, model alertResourceModel) any {
	if model.Expiry != nil {
		return withExpiry(alertConditionPayload{}, model.Expiry.AlertBeforeDays, model.Expiry.DailyReminder)
	}

	payload := alertConditionPayload{}
	if model.Condition != nil {
		payload.ApplicationIDs = setStrings(model.Condition.ApplicationIDs)
		payload.ProfileIDs = setStrings(model.Condition.ProfileIDs)
		payload.Sources = setStrings(model.Condition.Sources)
		payload.SignerIDs = setStrings(model.Condition.SignerIDs)
		if event.firesBeforeExpiry() {
			payload = withExpiry(payload, model.Condition.AlertBeforeDays, model.Condition.DailyReminder)
		}
	}

	if payload.AlertBefore == nil && len(payload.ApplicationIDs) == 0 && len(payload.ProfileIDs) == 0 && len(payload.Sources) == 0 && len(payload.SignerIDs) == 0 {
		return nil
	}
	return payload
}

func setStrings(set types.Set) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	values := make([]string, 0, len(set.Elements()))
	for _, element := range set.Elements() {
		if value, ok := element.(types.String); ok && !value.IsNull() && !value.IsUnknown() {
			values = append(values, value.ValueString())
		}
	}
	slices.Sort(values)
	return values
}

func stringSet(values []string) types.Set {
	if len(values) == 0 {
		return types.SetNull(types.StringType)
	}
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elements)
}

func setAlertConditionFromAPI(state *alertResourceModel, event alertEvent, raw json.RawMessage) diag.Diagnostics {
	var diags diag.Diagnostics

	var payload alertConditionPayload
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &payload); err != nil {
			diags.AddError("Error reading alert condition", "Couldn't parse the condition returned by Infisical, unexpected error: "+err.Error())
			return diags
		}
	}

	var alertBeforeDays types.Int64
	if event.firesBeforeExpiry() {
		if payload.AlertBefore == nil {
			diags.AddError("Missing alert condition", "Infisical returned an alert without a condition, so the number of days it alerts before expiry is unknown.")
			return diags
		}
		days, daysDiags := parseAlertBeforeDays(*payload.AlertBefore)
		diags.Append(daysDiags...)
		if diags.HasError() {
			return diags
		}
		alertBeforeDays = types.Int64Value(days)
	}
	dailyReminder := payload.DailyReminder != nil && *payload.DailyReminder

	if state.Expiry != nil && event.eventType == alertEventTypeIdentityAuthenticationExpiry {
		state.Expiry = &expiryConditionModel{AlertBeforeDays: alertBeforeDays, DailyReminder: types.BoolValue(dailyReminder)}
		state.Condition = nil
		return diags
	}
	state.Expiry = nil

	condition := &alertConditionModel{
		AlertBeforeDays: types.Int64Null(),
		DailyReminder:   types.BoolNull(),
		ApplicationIDs:  stringSet(payload.ApplicationIDs),
		ProfileIDs:      stringSet(payload.ProfileIDs),
		Sources:         stringSet(payload.Sources),
		SignerIDs:       stringSet(payload.SignerIDs),
	}
	if event.firesBeforeExpiry() {
		condition.AlertBeforeDays = alertBeforeDays
		// A configuration that leaves daily_reminder out means false, so false only shows when it was set.
		if dailyReminder || (state.Condition != nil && !state.Condition.DailyReminder.IsNull()) {
			condition.DailyReminder = types.BoolValue(dailyReminder)
		}
	}

	if condition.AlertBeforeDays.IsNull() && condition.ApplicationIDs.IsNull() && condition.ProfileIDs.IsNull() && condition.Sources.IsNull() && condition.SignerIDs.IsNull() {
		condition = nil
	}
	state.Condition = condition

	return diags
}

var daysPerAlertBeforeUnit = map[string]int64{"d": 1, "w": 7, "m": 30, "y": 365}

func parseAlertBeforeDays(alertBefore string) (int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(alertBefore) >= 2 {
		perUnit, unitOK := daysPerAlertBeforeUnit[alertBefore[len(alertBefore)-1:]]
		count, err := strconv.ParseInt(alertBefore[:len(alertBefore)-1], 10, 64)
		if unitOK && err == nil && count >= 0 {
			return count * perUnit, diags
		}
	}

	diags.AddError(
		"Error reading alert condition",
		fmt.Sprintf("Infisical returned an alert that fires %q before expiry, which is not a duration such as \"30d\", \"2w\", \"1m\" or \"1y\".", alertBefore),
	)
	return 0, diags
}

func validateAlertEvent(ctx context.Context, config tfsdk.Config, resourceType string) diag.Diagnostics {
	var diags diag.Diagnostics

	var eventType alertEventTypeValue
	var condition, expiry types.Object
	diags.Append(config.GetAttribute(ctx, path.Root("event_type"), &eventType)...)
	diags.Append(config.GetAttribute(ctx, path.Root("condition"), &condition)...)
	diags.Append(config.GetAttribute(ctx, path.Root("expiry"), &expiry)...)
	if diags.HasError() {
		return diags
	}

	if !expiry.IsNull() {
		if resourceType != alertResourceTypeIdentityAuthentication {
			diags.AddAttributeError(path.Root("expiry"), "Unexpected expiry",
				fmt.Sprintf("The deprecated expiry block is only for %s alerts. Use event_type and condition instead.", alertResourceTypeIdentityAuthentication))
		}
		if !condition.IsNull() {
			diags.AddAttributeError(path.Root("condition"), "Unexpected condition",
				"condition goes with event_type, not with the deprecated expiry block. Move alert_before_days into condition and set event_type instead.")
		}
		return diags
	}

	if eventType.IsNull() || eventType.IsUnknown() {
		return diags
	}

	event, ok := resolveAlertEvent(resourceType, eventType.ValueString())
	if !ok {
		diags.AddAttributeError(path.Root("event_type"), "Unsupported event_type",
			fmt.Sprintf("%s alerts don't have a %q event. Use one of: %s.", resourceType, eventType.ValueString(), describeAlertEvents(resourceType)))
		return diags
	}

	if condition.IsUnknown() {
		return diags
	}

	attributes := condition.Attributes()
	isSet := func(name string) bool {
		value, ok := attributes[name]
		return ok && !value.IsNull()
	}

	if event.firesBeforeExpiry() && !isSet("alert_before_days") {
		diags.AddAttributeError(path.Root("condition").AtName("alert_before_days"), "Missing alert_before_days",
			fmt.Sprintf("The %s event needs condition.alert_before_days to know when to fire.", event.eventType))
	}

	if condition.IsNull() {
		return diags
	}

	allowed := map[string]bool{
		"alert_before_days": event.firesBeforeExpiry(),
		"daily_reminder":    event.firesBeforeExpiry(),
		"application_ids":   event.takesFilter("application_ids"),
		"profile_ids":       event.takesFilter("profile_ids"),
		"sources":           event.takesFilter("sources"),
		"signer_ids":        event.takesFilter("signer_ids"),
	}
	set := 0
	for _, name := range slices.Sorted(maps.Keys(allowed)) {
		if !isSet(name) {
			continue
		}
		set++
		if !allowed[name] {
			diags.AddAttributeError(path.Root("condition").AtName(name), "Unexpected "+name,
				fmt.Sprintf("The %s event doesn't use %s, so remove it.", event.eventType, name))
		}
		if values, ok := attributes[name].(types.Set); ok && slices.ContainsFunc(values.Elements(), attr.Value.IsNull) {
			diags.AddAttributeError(path.Root("condition").AtName(name), "Invalid "+name,
				fmt.Sprintf("%s can't contain null.", name))
		}
	}

	if set == 0 {
		diags.AddAttributeError(path.Root("condition"), "Empty condition",
			"The condition sets nothing. Remove it, or set the settings the event uses.")
	}

	if alertBeforeDays, ok := attributes["alert_before_days"].(types.Int64); ok && event.firesBeforeExpiry() &&
		!alertBeforeDays.IsNull() && !alertBeforeDays.IsUnknown() && alertBeforeDays.ValueInt64() > event.maxAlertBeforeDays {
		diags.AddAttributeError(path.Root("condition").AtName("alert_before_days"), "Invalid alert_before_days",
			fmt.Sprintf("The %s event can fire at most %d days before expiry, got %d.", event.eventType, event.maxAlertBeforeDays, alertBeforeDays.ValueInt64()))
	}

	return diags
}
