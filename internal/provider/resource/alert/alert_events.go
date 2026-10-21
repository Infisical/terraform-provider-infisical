package resource

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	alertResourceTypeIdentityAuthentication = "identity.authentication"
	alertResourceTypeCertManagerApplication = "cert-manager.application"
	alertResourceTypeCertManager            = "cert-manager"
	alertResourceTypeCertManagerSigner      = "cert-manager.signer"

	alertEventTypeIdentityAuthenticationExpiry            = "identity.authentication.expiry"
	alertEventTypeIdentityAuthenticationAuthMethodChanged = "identity.authentication.auth-method-changed"

	alertEventTypeCertManagerApplicationCertExpiry     = "cert-manager.application.certificate.expiry"
	alertEventTypeCertManagerApplicationCertIssuance   = "cert-manager.application.certificate.issuance"
	alertEventTypeCertManagerApplicationCertRenewal    = "cert-manager.application.certificate.renewal"
	alertEventTypeCertManagerApplicationCertRevocation = "cert-manager.application.certificate.revocation"

	alertEventTypeCertManagerCertExpiry     = "cert-manager.certificate.expiry"
	alertEventTypeCertManagerCertIssuance   = "cert-manager.certificate.issuance"
	alertEventTypeCertManagerCertRenewal    = "cert-manager.certificate.renewal"
	alertEventTypeCertManagerCertRevocation = "cert-manager.certificate.revocation"

	alertEventTypeCertManagerSignerCertExpiry = "cert-manager.signer.certificate.expiry"
)

type alertResourceTypeTraits struct {
	resolvesProject       bool
	allowsEmailRecipients bool
	scopeWide             bool
}

var alertResourceTypes = map[string]alertResourceTypeTraits{
	alertResourceTypeIdentityAuthentication: {},
	alertResourceTypeCertManagerApplication: {resolvesProject: true, allowsEmailRecipients: true},
	alertResourceTypeCertManager:            {resolvesProject: true, allowsEmailRecipients: true, scopeWide: true},
	alertResourceTypeCertManagerSigner:      {resolvesProject: true, allowsEmailRecipients: true, scopeWide: true},
}

type alertEvent struct {
	resourceType string
	eventType    string

	maxAlertBeforeDays int64
	filters            bool
}

func (e alertEvent) firesBeforeExpiry() bool {
	return e.maxAlertBeforeDays > 0
}

func (e alertEvent) shortEventType() string {
	return strings.TrimPrefix(e.eventType, e.resourceType+".")
}

var alertEvents = []alertEvent{
	{resourceType: alertResourceTypeIdentityAuthentication, eventType: alertEventTypeIdentityAuthenticationExpiry, maxAlertBeforeDays: 90},
	{resourceType: alertResourceTypeIdentityAuthentication, eventType: alertEventTypeIdentityAuthenticationAuthMethodChanged},

	{resourceType: alertResourceTypeCertManagerApplication, eventType: alertEventTypeCertManagerApplicationCertExpiry, maxAlertBeforeDays: 365},
	{resourceType: alertResourceTypeCertManagerApplication, eventType: alertEventTypeCertManagerApplicationCertIssuance},
	{resourceType: alertResourceTypeCertManagerApplication, eventType: alertEventTypeCertManagerApplicationCertRenewal},
	{resourceType: alertResourceTypeCertManagerApplication, eventType: alertEventTypeCertManagerApplicationCertRevocation},

	{resourceType: alertResourceTypeCertManager, eventType: alertEventTypeCertManagerCertExpiry, maxAlertBeforeDays: 365, filters: true},
	{resourceType: alertResourceTypeCertManager, eventType: alertEventTypeCertManagerCertIssuance, filters: true},
	{resourceType: alertResourceTypeCertManager, eventType: alertEventTypeCertManagerCertRenewal, filters: true},
	{resourceType: alertResourceTypeCertManager, eventType: alertEventTypeCertManagerCertRevocation, filters: true},

	{resourceType: alertResourceTypeCertManagerSigner, eventType: alertEventTypeCertManagerSignerCertExpiry, maxAlertBeforeDays: 365},
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

func supportedAlertResourceTypes() []string {
	return alertResourceTypesWhere(func(alertResourceTypeTraits) bool { return true })
}

func alertEventForEventType(eventType string) (alertEvent, bool) {
	for _, event := range alertEvents {
		if event.eventType == eventType {
			return event, true
		}
	}
	return alertEvent{}, false
}

func resolveAlertEvent(resourceType, eventType string) (alertEvent, bool) {
	for _, event := range alertEvents {
		if event.resourceType == resourceType && (event.eventType == eventType || event.shortEventType() == eventType) {
			return event, true
		}
	}
	return alertEvent{}, false
}

func alertEventsFor(resourceType string) []alertEvent {
	events := []alertEvent{}
	for _, event := range alertEvents {
		if event.resourceType == resourceType {
			events = append(events, event)
		}
	}
	return events
}

func describeAlertEvents(resourceType string) string {
	names := []string{}
	for _, event := range alertEventsFor(resourceType) {
		names = append(names, fmt.Sprintf("%s (%s)", event.shortEventType(), event.eventType))
	}
	return strings.Join(names, ", ")
}

func describeAlertBeforeDaysLimits() string {
	typesByLimit := map[int64][]string{}
	for _, event := range alertEvents {
		if event.firesBeforeExpiry() {
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

// Both forms of event_type, with and without the resource type prefix, are semantically equal.
type alertEventTypeType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = alertEventTypeType{}

func (t alertEventTypeType) Equal(o attr.Type) bool {
	other, ok := o.(alertEventTypeType)
	return ok && t.StringType.Equal(other.StringType)
}

func (t alertEventTypeType) String() string {
	return "alertEventTypeType"
}

func (t alertEventTypeType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return alertEventTypeValue{StringValue: in}, nil
}

func (t alertEventTypeType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", attrValue)
	}
	return alertEventTypeValue{StringValue: stringValue}, nil
}

func (t alertEventTypeType) ValueType(_ context.Context) attr.Value {
	return alertEventTypeValue{}
}

type alertEventTypeValue struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuable                   = alertEventTypeValue{}
	_ basetypes.StringValuableWithSemanticEquals = alertEventTypeValue{}
)

func newAlertEventTypeValue(value string) alertEventTypeValue {
	return alertEventTypeValue{StringValue: basetypes.NewStringValue(value)}
}

func (v alertEventTypeValue) Type(_ context.Context) attr.Type {
	return alertEventTypeType{}
}

func (v alertEventTypeValue) Equal(o attr.Value) bool {
	other, ok := o.(alertEventTypeValue)
	return ok && v.StringValue.Equal(other.StringValue)
}

func (v alertEventTypeValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(alertEventTypeValue)
	if !ok {
		diags.AddError("Semantic Equality Check Error", fmt.Sprintf("expected value type %T but got %T", v, newValuable))
		return false, diags
	}
	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return false, diags
	}

	return sameAlertEventType(v.ValueString(), newValue.ValueString()), diags
}

func sameAlertEventType(a, b string) bool {
	if a == b {
		return true
	}
	for resourceType := range alertResourceTypes {
		left, leftOK := resolveAlertEvent(resourceType, a)
		right, rightOK := resolveAlertEvent(resourceType, b)
		if leftOK && rightOK && left.eventType == right.eventType {
			return true
		}
	}
	return false
}
