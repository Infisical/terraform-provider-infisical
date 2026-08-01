package resource

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	alertResourceTypeIdentityAuthentication    = "identity.authentication"
	alertEventTypeIdentityAuthenticationExpiry = "identity.authentication.expiry"
)

const alertBlockAuthenticationExpiry = "authentication_expiry"

const (
	minAlertBeforeDays = 1
	maxAlertBeforeDays = 90
)

type alertEvent struct {
	resourceType string
	eventType    string
	block        string
}

var alertEvents = []alertEvent{
	{
		resourceType: alertResourceTypeIdentityAuthentication,
		eventType:    alertEventTypeIdentityAuthenticationExpiry,
		block:        alertBlockAuthenticationExpiry,
	},
}

func alertEventForBlock(block string) (alertEvent, bool) {
	for _, event := range alertEvents {
		if event.block == block {
			return event, true
		}
	}
	return alertEvent{}, false
}

func alertEventForEventType(eventType string) (alertEvent, bool) {
	for _, event := range alertEvents {
		if event.eventType == eventType {
			return event, true
		}
	}
	return alertEvent{}, false
}

func supportedAlertResourceTypes() []string {
	seen := make(map[string]bool, len(alertEvents))
	types := make([]string, 0, len(alertEvents))
	for _, event := range alertEvents {
		if seen[event.resourceType] {
			continue
		}
		seen[event.resourceType] = true
		types = append(types, event.resourceType)
	}
	sort.Strings(types)
	return types
}

func alertConditionBlockPaths() []path.Expression {
	paths := make([]path.Expression, 0, len(alertEvents))
	for _, event := range alertEvents {
		paths = append(paths, path.MatchRoot(event.block))
	}
	return paths
}

func alertBlocksForResourceType(resourceType string) []string {
	blocks := make([]string, 0, len(alertEvents))
	for _, event := range alertEvents {
		if event.resourceType == resourceType {
			blocks = append(blocks, event.block)
		}
	}
	sort.Strings(blocks)
	return blocks
}

type authenticationExpiryConditionModel struct {
	AlertBeforeDays types.Int64 `tfsdk:"alert_before_days"`
	DailyReminder   types.Bool  `tfsdk:"daily_reminder"`
}

type authenticationExpiryCondition struct {
	AlertBefore   string `json:"alertBefore"`
	DailyReminder bool   `json:"dailyReminder"`
}

func authenticationExpiryConditionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: fmt.Sprintf("Fires before a machine identity's authentication credentials expire. Only for alerts on the %s resource type.", alertResourceTypeIdentityAuthentication),
		Attributes: map[string]schema.Attribute{
			"alert_before_days": schema.Int64Attribute{
				Required:    true,
				Description: fmt.Sprintf("How many days before an authentication credential expires the alert fires. Must be between %d and %d.", minAlertBeforeDays, maxAlertBeforeDays),
				Validators: []validator.Int64{
					int64validator.Between(minAlertBeforeDays, maxAlertBeforeDays),
				},
			},
			"daily_reminder": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to keep notifying once a day until the credential expires, instead of notifying once. Defaults to false.",
			},
		},
	}
}

func alertConditionForAPI(plan alertResourceModel) (alertEvent, any, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.AuthenticationExpiry != nil {
		event, _ := alertEventForBlock(alertBlockAuthenticationExpiry)
		return event, authenticationExpiryCondition{
			AlertBefore:   fmt.Sprintf("%dd", plan.AuthenticationExpiry.AlertBeforeDays.ValueInt64()),
			DailyReminder: plan.AuthenticationExpiry.DailyReminder.ValueBool(),
		}, diags
	}

	diags.AddError(
		"Missing alert condition",
		"The alert has no condition block, so there is nothing to tell Infisical when it should fire. Exactly one of "+strings.Join(alertConditionBlockNames(), ", ")+" is required.",
	)
	return alertEvent{}, nil, diags
}

func alertConditionBlockNames() []string {
	blocks := make([]string, 0, len(alertEvents))
	for _, event := range alertEvents {
		blocks = append(blocks, event.block)
	}
	sort.Strings(blocks)
	return blocks
}

func setAlertConditionFromAPI(state *alertResourceModel, event alertEvent, raw json.RawMessage) diag.Diagnostics {
	var diags diag.Diagnostics

	state.AuthenticationExpiry = nil

	switch event.eventType {
	case alertEventTypeIdentityAuthenticationExpiry:
		condition, conditionDiags := parseAuthenticationExpiryCondition(raw)
		diags.Append(conditionDiags...)
		if diags.HasError() {
			return diags
		}

		alertBeforeDays, daysDiags := parseAlertBeforeDays(condition.AlertBefore)
		diags.Append(daysDiags...)
		if diags.HasError() {
			return diags
		}

		state.AuthenticationExpiry = &authenticationExpiryConditionModel{
			AlertBeforeDays: types.Int64Value(alertBeforeDays),
			DailyReminder:   types.BoolValue(condition.DailyReminder),
		}
	default:
		diags.AddError(
			"Unsupported alert event",
			fmt.Sprintf("Infisical returned an alert that fires on %q, which this provider does not know how to manage. Please upgrade the provider.", event.eventType),
		)
	}

	return diags
}

func parseAuthenticationExpiryCondition(raw json.RawMessage) (authenticationExpiryCondition, diag.Diagnostics) {
	var diags diag.Diagnostics
	var condition authenticationExpiryCondition

	if len(raw) == 0 || string(raw) == "null" {
		diags.AddError(
			"Missing alert condition",
			"Infisical returned an alert without a condition, so the number of days it alerts before expiry is unknown.",
		)
		return condition, diags
	}

	if err := json.Unmarshal(raw, &condition); err != nil {
		diags.AddError(
			"Error reading alert condition",
			"Couldn't parse the condition returned by Infisical, unexpected error: "+err.Error(),
		)
	}

	return condition, diags
}

func parseAlertBeforeDays(alertBefore string) (int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	days, err := strconv.ParseInt(strings.TrimSuffix(alertBefore, "d"), 10, 64)
	if err != nil || !strings.HasSuffix(alertBefore, "d") {
		diags.AddError(
			"Error reading alert condition",
			fmt.Sprintf("Infisical returned an alert that fires %q before expiry, which is not a whole number of days such as \"30d\".", alertBefore),
		)
		return 0, diags
	}

	return days, diags
}
