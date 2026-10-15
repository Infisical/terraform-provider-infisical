package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
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

	alertResourceTypeCertManagerApplication            = "cert-manager.application"
	alertEventTypeCertManagerApplicationCertExpiry     = "cert-manager.application.certificate.expiry"
	alertEventTypeCertManagerApplicationCertIssuance   = "cert-manager.application.certificate.issuance"
	alertEventTypeCertManagerApplicationCertRenewal    = "cert-manager.application.certificate.renewal"
	alertEventTypeCertManagerApplicationCertRevocation = "cert-manager.application.certificate.revocation"

	alertResourceTypeCertManager            = "cert-manager"
	alertEventTypeCertManagerCertExpiry     = "cert-manager.certificate.expiry"
	alertEventTypeCertManagerCertIssuance   = "cert-manager.certificate.issuance"
	alertEventTypeCertManagerCertRenewal    = "cert-manager.certificate.renewal"
	alertEventTypeCertManagerCertRevocation = "cert-manager.certificate.revocation"

	alertResourceTypeCertManagerSigner        = "cert-manager.signer"
	alertEventTypeCertManagerSignerCertExpiry = "cert-manager.signer.certificate.expiry"
)

const (
	alertBlockExpiry     = "expiry"
	alertBlockIssuance   = "issuance"
	alertBlockRenewal    = "renewal"
	alertBlockRevocation = "revocation"
)

const minAlertBeforeDays = 1

type alertEvent struct {
	resourceType string
	eventType    string
	block        string

	maxAlertBeforeDays int64
}

type alertResourceTypeTraits struct {
	resolvesProject       bool
	allowsEmailRecipients bool
	scopeWide             bool
	supportsFilters       bool
}

var alertResourceTypes = map[string]alertResourceTypeTraits{
	alertResourceTypeIdentityAuthentication: {},
	alertResourceTypeCertManagerApplication: {resolvesProject: true, allowsEmailRecipients: true},
	alertResourceTypeCertManager:            {resolvesProject: true, allowsEmailRecipients: true, scopeWide: true, supportsFilters: true},
	alertResourceTypeCertManagerSigner:      {resolvesProject: true, allowsEmailRecipients: true, scopeWide: true},
}

var alertEvents = []alertEvent{
	{
		resourceType:       alertResourceTypeIdentityAuthentication,
		eventType:          alertEventTypeIdentityAuthenticationExpiry,
		block:              alertBlockExpiry,
		maxAlertBeforeDays: 90,
	},
	{
		resourceType:       alertResourceTypeCertManagerApplication,
		eventType:          alertEventTypeCertManagerApplicationCertExpiry,
		block:              alertBlockExpiry,
		maxAlertBeforeDays: 365,
	},
	{
		resourceType: alertResourceTypeCertManagerApplication,
		eventType:    alertEventTypeCertManagerApplicationCertIssuance,
		block:        alertBlockIssuance,
	},
	{
		resourceType: alertResourceTypeCertManagerApplication,
		eventType:    alertEventTypeCertManagerApplicationCertRenewal,
		block:        alertBlockRenewal,
	},
	{
		resourceType: alertResourceTypeCertManagerApplication,
		eventType:    alertEventTypeCertManagerApplicationCertRevocation,
		block:        alertBlockRevocation,
	},
	{
		resourceType:       alertResourceTypeCertManager,
		eventType:          alertEventTypeCertManagerCertExpiry,
		block:              alertBlockExpiry,
		maxAlertBeforeDays: 365,
	},
	{
		resourceType: alertResourceTypeCertManager,
		eventType:    alertEventTypeCertManagerCertIssuance,
		block:        alertBlockIssuance,
	},
	{
		resourceType: alertResourceTypeCertManager,
		eventType:    alertEventTypeCertManagerCertRenewal,
		block:        alertBlockRenewal,
	},
	{
		resourceType: alertResourceTypeCertManager,
		eventType:    alertEventTypeCertManagerCertRevocation,
		block:        alertBlockRevocation,
	},
	{
		resourceType:       alertResourceTypeCertManagerSigner,
		eventType:          alertEventTypeCertManagerSignerCertExpiry,
		block:              alertBlockExpiry,
		maxAlertBeforeDays: 365,
	},
}

func alertResourceTypesWhere(match func(alertResourceTypeTraits) bool) []string {
	resourceTypes := []string{}
	for resourceType, traits := range alertResourceTypes {
		if match(traits) {
			resourceTypes = append(resourceTypes, resourceType)
		}
	}
	sort.Strings(resourceTypes)
	return resourceTypes
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
	return alertResourceTypesWhere(func(alertResourceTypeTraits) bool { return true })
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
	certificateFilters
}

type eventConditionModel struct{}

func expiryConditionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Description: fmt.Sprintf(
			"Fires before something the alert watches expires. Only for alerts on these resource types: %s.",
			strings.Join(alertResourceTypesForBlock(alertBlockExpiry), ", "),
		),
		Attributes: map[string]schema.Attribute{
			"alert_before_days": schema.Int64Attribute{
				Required:    true,
				Description: fmt.Sprintf("How many days before expiry the alert fires. Must be at least %d, and at most %s.", minAlertBeforeDays, describeAlertBeforeDaysLimits()),
				Validators: []validator.Int64{
					int64validator.AtLeast(minAlertBeforeDays),
				},
			},
			"daily_reminder": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to keep notifying once a day until expiry, instead of notifying once. Defaults to false.",
			},
		},
	}
}

func describeAlertBeforeDaysLimits() string {
	typesByLimit := map[int64][]string{}
	for _, event := range alertEvents {
		if event.block == alertBlockExpiry {
			typesByLimit[event.maxAlertBeforeDays] = append(typesByLimit[event.maxAlertBeforeDays], event.resourceType)
		}
	}

	limits := make([]string, 0, len(typesByLimit))
	for _, limit := range slices.Sorted(maps.Keys(typesByLimit)) {
		resourceTypes := typesByLimit[limit]
		sort.Strings(resourceTypes)
		limits = append(limits, fmt.Sprintf("%d for %s", limit, strings.Join(resourceTypes, ", ")))
	}
	return strings.Join(limits, "; ")
}

var eventConditionVerbs = map[string]string{
	alertBlockIssuance:   "issued",
	alertBlockRenewal:    "renewed",
	alertBlockRevocation: "revoked",
}

func eventConditionSchema(block string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Description: fmt.Sprintf(
			"Fires each time something the alert watches is %s. It takes no settings, so write it as `%s = {}`. Only for alerts on these resource types: %s.",
			eventConditionVerbs[block], block, strings.Join(alertResourceTypesForBlock(block), ", "),
		),
		Attributes: map[string]schema.Attribute{},
	}
}

func alertConditionBlocks(model alertResourceModel) map[string]bool {
	return map[string]bool{
		alertBlockExpiry:     model.Expiry != nil,
		alertBlockIssuance:   model.Issuance != nil,
		alertBlockRenewal:    model.Renewal != nil,
		alertBlockRevocation: model.Revocation != nil,
	}
}

func alertConditionForAPI(plan alertResourceModel) (alertEvent, any, diag.Diagnostics) {
	var diags diag.Diagnostics

	resourceType := plan.ResourceType.ValueString()
	blocks := alertConditionBlocks(plan)
	for _, block := range alertConditionBlockNames() {
		if !blocks[block] {
			continue
		}

		event, ok := alertEventFor(resourceType, block)
		if !ok {
			diags.AddAttributeError(
				path.Root(block),
				"Unexpected alert condition",
				fmt.Sprintf(
					"A %s block does not describe an event on the %s resource type, so there is nothing to tell Infisical when the alert should fire.",
					block, resourceType,
				),
			)
			return alertEvent{}, nil, diags
		}

		filters := alertFiltersForAPI(plan.Filters)

		if block != alertBlockExpiry {
			if filters.isEmpty() {
				return event, nil, diags
			}
			return event, filters, diags
		}

		return event, expiryCondition{
			AlertBefore:        fmt.Sprintf("%dd", plan.Expiry.AlertBeforeDays.ValueInt64()),
			DailyReminder:      plan.Expiry.DailyReminder.ValueBool(),
			certificateFilters: filters,
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
	state.Issuance = nil
	state.Renewal = nil
	state.Revocation = nil
	state.Filters = nil

	if alertResourceTypes[event.resourceType].supportsFilters {
		filters, filterDiags := alertFiltersFromAPI(raw)
		diags.Append(filterDiags...)
		if diags.HasError() {
			return diags
		}
		state.Filters = filters
	}

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
	case alertBlockIssuance:
		state.Issuance = &eventConditionModel{}
	case alertBlockRenewal:
		state.Renewal = &eventConditionModel{}
	case alertBlockRevocation:
		state.Revocation = &eventConditionModel{}
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
