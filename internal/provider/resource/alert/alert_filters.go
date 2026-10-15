package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const maxAlertFilterIDs = 100

var certificateSources = []string{"issued", "discovered", "imported"}

type alertFiltersModel struct {
	ApplicationIDs types.Set `tfsdk:"application_ids"`
	ProfileIDs     types.Set `tfsdk:"profile_ids"`
	Sources        types.Set `tfsdk:"sources"`
}

type certificateFilters struct {
	ApplicationIDs []string `json:"applicationIds,omitempty"`
	ProfileIDs     []string `json:"profileIds,omitempty"`
	Sources        []string `json:"sources,omitempty"`
}

func (f certificateFilters) isEmpty() bool {
	return len(f.ApplicationIDs) == 0 && len(f.ProfileIDs) == 0 && len(f.Sources) == 0
}

func alertFiltersSchema() schema.SingleNestedAttribute {
	idsAttribute := func(description string) schema.SetAttribute {
		return schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: fmt.Sprintf("%s Leave it out to cover all of them. At most %d.", description, maxAlertFilterIDs),
			Validators: []validator.Set{
				setvalidator.SizeBetween(1, maxAlertFilterIDs),
				setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
			},
		}
	}

	return schema.SingleNestedAttribute{
		Optional: true,
		Description: fmt.Sprintf(
			"Narrows the alert to some of what it watches, which has to match every list that is set. Only for alerts on these resource types: %s.",
			strings.Join(alertResourceTypesWhere(func(traits alertResourceTypeTraits) bool { return traits.supportsFilters }), ", "),
		),
		Attributes: map[string]schema.Attribute{
			"application_ids": idsAttribute("Only certificates in these Certificate Manager applications."),
			"profile_ids":     idsAttribute("Only certificates issued from these certificate profiles."),
			"sources": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Only certificates with these sources. Options: " + strings.Join(certificateSources, ", ") + ". Leave it out to cover every source.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(certificateSources...)),
				},
			},
		},
	}
}

func validateAlertFilters(ctx context.Context, config tfsdk.Config) diag.Diagnostics {
	var diags diag.Diagnostics

	var filters types.Object
	diags.Append(config.GetAttribute(ctx, path.Root("filters"), &filters)...)
	if diags.HasError() || filters.IsNull() || filters.IsUnknown() {
		return diags
	}

	attributes := filters.Attributes()
	if attributes["application_ids"].IsNull() && attributes["profile_ids"].IsNull() && attributes["sources"].IsNull() {
		diags.AddAttributeError(
			path.Root("filters"),
			"Empty filters",
			"The filters block sets none of application_ids, profile_ids or sources. Set at least one, or remove the block to cover everything the alert watches.",
		)
	}

	return diags
}

func alertFiltersForAPI(filters *alertFiltersModel) certificateFilters {
	if filters == nil {
		return certificateFilters{}
	}
	return certificateFilters{
		ApplicationIDs: setStrings(filters.ApplicationIDs),
		ProfileIDs:     setStrings(filters.ProfileIDs),
		Sources:        setStrings(filters.Sources),
	}
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

func alertFiltersFromAPI(raw json.RawMessage) (*alertFiltersModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(raw) == 0 || string(raw) == "null" {
		return nil, diags
	}

	var filters certificateFilters
	if err := json.Unmarshal(raw, &filters); err != nil {
		diags.AddError(
			"Error reading alert filters",
			"Couldn't parse the condition returned by Infisical, unexpected error: "+err.Error(),
		)
		return nil, diags
	}
	if filters.isEmpty() {
		return nil, diags
	}

	toSet := func(values []string) types.Set {
		if len(values) == 0 {
			return types.SetNull(types.StringType)
		}
		elements := make([]attr.Value, 0, len(values))
		for _, value := range values {
			elements = append(elements, types.StringValue(value))
		}
		return types.SetValueMust(types.StringType, elements)
	}

	return &alertFiltersModel{
		ApplicationIDs: toSet(filters.ApplicationIDs),
		ProfileIDs:     toSet(filters.ProfileIDs),
		Sources:        toSet(filters.Sources),
	}, diags
}
