package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	placeholderCertificateID      = "{{certificateId}}"
	placeholderShortCertificateID = "{{shortCertificateId}}"

	availableNamePlaceholders = "{{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}"
)

var (
	anyNamePlaceholderPattern = regexp.MustCompile(`\{\{(certificateId|shortCertificateId|profileId|applicationId|applicationName|commonName)\}\}`)
	commandVariablePattern    = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
)

// singleCertificateCommandVariables name one certificate, so a command using them caps the sync at one.
var singleCertificateCommandVariables = map[string]bool{"certificatePath": true, "commonName": true}

// certificateNameRule mirrors a destination's backend rule for certificate_name_schema.
type certificateNameRule struct {
	pattern                     *regexp.Regexp
	minLength, maxLength        int
	requireIdentifier           bool
	requireLiteralCertificateID bool
	requirement                 string
}

// compileCertificateNameSchemaTest fills placeholders with the same stand-in values the backend uses
// to validate a schema, so character and length rules give the same answer at plan time.
func compileCertificateNameSchemaTest(schema string) string {
	replacements := []struct{ placeholder, value string }{
		{placeholderShortCertificateID, strings.Repeat("0", 22)},
		{placeholderCertificateID, strings.Repeat("0", 32)},
		{"{{profileId}}", strings.Repeat("0", 32)},
		{"{{applicationId}}", strings.Repeat("0", 32)},
		{"{{applicationName}}", "application-name"},
		{"{{commonName}}", "common-name"},
	}
	for _, r := range replacements {
		schema = strings.ReplaceAll(schema, r.placeholder, r.value)
	}
	return schema
}

func hasCertificateIdentifier(schema string) bool {
	return strings.Contains(schema, placeholderCertificateID) || strings.Contains(schema, placeholderShortCertificateID)
}

func (rule certificateNameRule) validate(schema string) []string {
	var problems []string
	schema = strings.TrimSpace(schema)

	if rule.requireLiteralCertificateID && !strings.Contains(schema, placeholderCertificateID) {
		problems = append(problems, "It must include the {{certificateId}} placeholder. {{shortCertificateId}} is accepted when the sync is created but fails every sync run for this destination.")
	} else if rule.requireIdentifier && !hasCertificateIdentifier(schema) {
		problems = append(problems, "It must include the {{certificateId}} or {{shortCertificateId}} placeholder.")
	}

	compiled := compileCertificateNameSchemaTest(schema)
	length := len(compiled)
	if !rule.pattern.MatchString(compiled) || (rule.minLength > 0 && length < rule.minLength) || (rule.maxLength > 0 && length > rule.maxLength) {
		problems = append(problems, fmt.Sprintf("With placeholders filled in it must be %s. Available placeholders: %s.", rule.requirement, availableNamePlaceholders))
	}
	return problems
}

// stringAttribute reads a string attribute from an object, reporting whether it is known and set.
func stringAttribute(ctx context.Context, obj types.Object, name string) (value string, known bool, set bool) {
	if obj.IsUnknown() {
		return "", false, false
	}
	if obj.IsNull() {
		return "", true, false
	}
	raw, ok := obj.Attributes()[name]
	if !ok || raw.IsNull() {
		return "", true, false
	}
	if raw.IsUnknown() {
		return "", false, false
	}
	valuable, ok := raw.(basetypes.StringValuable)
	if !ok {
		return "", false, false
	}
	str, diags := valuable.ToStringValue(ctx)
	if diags.HasError() {
		return "", false, false
	}
	return str.ValueString(), true, true
}

func syncOptionsDeclare(r *CertificateSyncBaseResource, name string) bool {
	_, ok := r.SyncOptionsAttributes[name]
	return ok
}

type certificateCap struct {
	max    int
	reason string
}

// certificateCapFromConfig works out the backend's limit on how many certificates the sync can hold.
func (r *CertificateSyncBaseResource) certificateCapFromConfig(ctx context.Context, config CertificateSyncBaseResourceModel) (certificateCap, bool) {
	if r.MaxCertificates > 0 {
		return certificateCap{max: r.MaxCertificates, reason: fmt.Sprintf("%s holds at most %d certificate(s)", r.SyncName, r.MaxCertificates)}, true
	}

	if syncOptionsDeclare(r, "certificate_name_schema") {
		schema, known, set := stringAttribute(ctx, config.SyncOptions, "certificate_name_schema")
		if known && (!set || !anyNamePlaceholderPattern.MatchString(schema)) {
			return certificateCap{max: 1, reason: "certificate_name_schema has no placeholder, so every certificate would get the same name"}, true
		}
	}

	for _, command := range []string{"health_check_command", "post_sync_command"} {
		if !syncOptionsDeclare(r, command) {
			continue
		}
		value, known, set := stringAttribute(ctx, config.SyncOptions, command)
		if !known || !set {
			continue
		}
		for _, match := range commandVariablePattern.FindAllStringSubmatch(value, -1) {
			if singleCertificateCommandVariables[match[1]] {
				return certificateCap{max: 1, reason: fmt.Sprintf("%s uses {{%s}}, which names a single certificate", command, match[1])}, true
			}
		}
	}

	return certificateCap{}, false
}

// enforceCertificateCap rejects filters the backend refuses on a capped sync: it only accepts an
// explicit certificate list, no longer than the cap.
func enforceCertificateCap(ctx context.Context, limit certificateCap, filters types.Object, diags *diag.Diagnostics) {
	if filters.IsNull() || filters.IsUnknown() {
		return
	}
	var model certificateFiltersModel
	diags.Append(filters.As(ctx, &model, objectAsOptions)...)
	if diags.HasError() {
		return
	}

	if !model.ProfileIDs.IsNull() || !model.Metadata.IsNull() {
		diags.AddAttributeError(
			path.Root(attrCertificateFilters),
			"Certificate filter not allowed",
			fmt.Sprintf("This sync can hold at most %d certificate(s) because %s, so certificate_filters can only use certificate_ids. Remove profile_ids and metadata.", limit.max, limit.reason),
		)
	}
	if !model.CertificateIDs.IsNull() && !model.CertificateIDs.IsUnknown() && len(model.CertificateIDs.Elements()) > limit.max {
		diags.AddAttributeError(
			path.Root(attrCertificateFilters).AtName("certificate_ids"),
			"Too many certificates",
			fmt.Sprintf("This sync can hold at most %d certificate(s) because %s.", limit.max, limit.reason),
		)
	}
}

// warnOnNameCollisions flags schemas that can give two certificates the same name, which makes one
// silently overwrite the other at the destination.
func warnOnNameCollisions(ctx context.Context, schema string, filters types.Object, diags *diag.Diagnostics) {
	if hasCertificateIdentifier(schema) || !anyNamePlaceholderPattern.MatchString(schema) || filters.IsNull() || filters.IsUnknown() {
		return
	}
	var model certificateFiltersModel
	diags.Append(filters.As(ctx, &model, objectAsOptions)...)
	if diags.HasError() {
		return
	}
	multiple := !model.ProfileIDs.IsNull() || !model.Metadata.IsNull() ||
		(!model.CertificateIDs.IsNull() && !model.CertificateIDs.IsUnknown() && len(model.CertificateIDs.Elements()) > 1)
	if !multiple {
		return
	}
	diags.AddAttributeWarning(
		path.Root(attrSyncOptions).AtName("certificate_name_schema"),
		"Certificates may share a name",
		"certificate_name_schema has no {{certificateId}} or {{shortCertificateId}} placeholder, so two certificates can resolve to the same name and one silently replaces the other at the destination. This is only safe when every placeholder used differs between the certificates, for example a distinct common name per certificate.",
	)
}

func (r *CertificateSyncBaseResource) validateCertificateNamesAndCap(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
	schema, schemaKnown, schemaSet := stringAttribute(ctx, config.SyncOptions, "certificate_name_schema")

	if r.CertificateNameRule != nil && schemaKnown && schemaSet {
		for _, problem := range r.CertificateNameRule.validate(schema) {
			diags.AddAttributeError(path.Root(attrSyncOptions).AtName("certificate_name_schema"), "Invalid certificate name schema", problem)
		}
	}

	if limit, capped := r.certificateCapFromConfig(ctx, config); capped {
		enforceCertificateCap(ctx, limit, config.CertificateFilters, diags)
		return
	}

	if schemaKnown && schemaSet {
		warnOnNameCollisions(ctx, schema, config.CertificateFilters, diags)
	}
}

// validateFieldMappingsUnique rejects two field mappings that name the same key, which makes the
// later field silently overwrite the earlier one in the delivered secret.
func validateFieldMappingsUnique(ctx context.Context, syncOptions types.Object, defaults map[string]string, diags *diag.Diagnostics) {
	if syncOptions.IsNull() || syncOptions.IsUnknown() {
		return
	}
	mappingsValue, ok := syncOptions.Attributes()["field_mappings"]
	if !ok || mappingsValue.IsUnknown() {
		return
	}
	mappings, _ := mappingsValue.(types.Object)

	seen := map[string]string{}
	for _, field := range []string{"certificate", "private_key", "certificate_chain", "ca_certificate"} {
		value := defaults[field]
		if !mappings.IsNull() {
			configured, known, set := stringAttribute(ctx, mappings, field)
			if !known {
				return
			}
			if set {
				value = configured
			}
		}
		if other, dup := seen[value]; dup {
			diags.AddAttributeError(
				path.Root(attrSyncOptions).AtName("field_mappings").AtName(field),
				"Duplicate field mapping",
				fmt.Sprintf("%s and %s both map to %q, so one would overwrite the other. Give each field a distinct name.", other, field, value),
			)
			continue
		}
		seen[value] = field
	}
}
