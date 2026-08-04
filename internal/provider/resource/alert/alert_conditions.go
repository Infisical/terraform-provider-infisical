package resource

import (
	"context"
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

const alertBlockExpiry = "expiry"

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
		block:        alertBlockExpiry,
	},
}

func alertEventFor(resourceType, block string) (alertEvent, bool) {
	for _, event := range alertEvents {
		if event.resourceType == resourceType && event.block == block {
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

type alertConditionSource interface {
	GetAttribute(ctx context.Context, p path.Path, target any) diag.Diagnostics
}

func alertConditionBlockSet(ctx context.Context, source alertConditionSource) (string, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	for _, block := range alertConditionBlockNames() {
		var value types.Object
		diags.Append(source.GetAttribute(ctx, path.Root(block), &value)...)
		if diags.HasError() {
			return "", false, diags
		}
		if value.IsUnknown() {
			return "", false, diags
		}
		if !value.IsNull() {
			return block, true, diags
		}
	}

	return "", false, diags
}

func alertEventFromBlocks(ctx context.Context, source alertConditionSource) (alertEvent, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	block, ok, blockDiags := alertConditionBlockSet(ctx, source)
	diags.Append(blockDiags...)
	if diags.HasError() || !ok {
		return alertEvent{}, false, diags
	}

	var resourceType types.String
	diags.Append(source.GetAttribute(ctx, path.Root("resource_type"), &resourceType)...)
	if diags.HasError() || resourceType.IsNull() || resourceType.IsUnknown() {
		return alertEvent{}, false, diags
	}

	event, ok := alertEventFor(resourceType.ValueString(), block)
	return event, ok, diags
}

func supportedAlertResourceTypes() []string {
	seen := make(map[string]bool, len(alertEvents))
	resourceTypes := make([]string, 0, len(alertEvents))
	for _, event := range alertEvents {
		if seen[event.resourceType] {
			continue
		}
		seen[event.resourceType] = true
		resourceTypes = append(resourceTypes, event.resourceType)
	}
	sort.Strings(resourceTypes)
	return resourceTypes
}

func alertConditionBlockNames() []string {
	seen := make(map[string]bool, len(alertEvents))
	blocks := make([]string, 0, len(alertEvents))
	for _, event := range alertEvents {
		if seen[event.block] {
			continue
		}
		seen[event.block] = true
		blocks = append(blocks, event.block)
	}
	sort.Strings(blocks)
	return blocks
}

func alertConditionBlockPaths() []path.Expression {
	blocks := alertConditionBlockNames()
	paths := make([]path.Expression, 0, len(blocks))
	for _, block := range blocks {
		paths = append(paths, path.MatchRoot(block))
	}
	return paths
}

func alertBlocksForResourceType(resourceType string) []string {
	seen := make(map[string]bool, len(alertEvents))
	blocks := make([]string, 0, len(alertEvents))
	for _, event := range alertEvents {
		if event.resourceType != resourceType || seen[event.block] {
			continue
		}
		seen[event.block] = true
		blocks = append(blocks, event.block)
	}
	sort.Strings(blocks)
	return blocks
}

func alertResourceTypesForBlock(block string) []string {
	seen := make(map[string]bool, len(alertEvents))
	resourceTypes := make([]string, 0, len(alertEvents))
	for _, event := range alertEvents {
		if event.block != block || seen[event.resourceType] {
			continue
		}
		seen[event.resourceType] = true
		resourceTypes = append(resourceTypes, event.resourceType)
	}
	sort.Strings(resourceTypes)
	return resourceTypes
}

type expiryConditionModel struct {
	AlertBeforeDays types.Int64 `tfsdk:"alert_before_days"`
	DailyReminder   types.Bool  `tfsdk:"daily_reminder"`
}

type expiryCondition struct {
	AlertBefore   string `json:"alertBefore"`
	DailyReminder bool   `json:"dailyReminder"`
}

func expiryConditionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Description: fmt.Sprintf(
			"Fires before the watched resource expires. Only for alerts on these resource types: %s.",
			strings.Join(alertResourceTypesForBlock(alertBlockExpiry), ", "),
		),
		Attributes: map[string]schema.Attribute{
			"alert_before_days": schema.Int64Attribute{
				Required:    true,
				Description: fmt.Sprintf("How many days before the watched resource expires the alert fires. Must be between %d and %d.", minAlertBeforeDays, maxAlertBeforeDays),
				Validators: []validator.Int64{
					int64validator.Between(minAlertBeforeDays, maxAlertBeforeDays),
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

func alertConditionForAPI(plan alertResourceModel) (alertEvent, any, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.Expiry != nil {
		resourceType := plan.ResourceType.ValueString()
		event, ok := alertEventFor(resourceType, alertBlockExpiry)
		if !ok {
			diags.AddAttributeError(
				path.Root(alertBlockExpiry),
				"Unexpected alert condition",
				fmt.Sprintf(
					"A %s block does not describe an event on the %s resource type, so there is nothing to tell Infisical when the alert should fire.",
					alertBlockExpiry, resourceType,
				),
			)
			return alertEvent{}, nil, diags
		}

		return event, expiryCondition{
			AlertBefore:   fmt.Sprintf("%dd", plan.Expiry.AlertBeforeDays.ValueInt64()),
			DailyReminder: plan.Expiry.DailyReminder.ValueBool(),
		}, diags
	}

	diags.AddError(
		"Missing alert condition",
		"The alert has no condition block, so there is nothing to tell Infisical when it should fire. Exactly one of "+strings.Join(alertConditionBlockNames(), ", ")+" is required.",
	)
	return alertEvent{}, nil, diags
}

func setAlertConditionFromAPI(state *alertResourceModel, event alertEvent, raw json.RawMessage) diag.Diagnostics {
	var diags diag.Diagnostics

	state.Expiry = nil

	switch event.block {
	case alertBlockExpiry:
		condition, conditionDiags := parseExpiryCondition(raw)
		diags.Append(conditionDiags...)
		if diags.HasError() {
			return diags
		}

		alertBeforeDays, daysDiags := parseAlertBeforeDays(condition.AlertBefore)
		diags.Append(daysDiags...)
		if diags.HasError() {
			return diags
		}

		state.Expiry = &expiryConditionModel{
			AlertBeforeDays: types.Int64Value(alertBeforeDays),
			DailyReminder:   types.BoolValue(condition.DailyReminder),
		}
	default:
		diags.AddError(
			"Unsupported alert event",
			fmt.Sprintf(
				"Infisical returned an alert that fires on %q, which this provider does not know how to manage. Please upgrade the provider. %s",
				event.eventType, alertRefreshEscapeHatch,
			),
		)
	}

	return diags
}

func parseExpiryCondition(raw json.RawMessage) (expiryCondition, diag.Diagnostics) {
	var diags diag.Diagnostics
	var condition expiryCondition

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
