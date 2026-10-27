package resource

import (
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const pamMachineIdentityOnly = "Only Machine Identity authentication is supported for this operation"

const pamNameDescription = "Must be lowercase letters, numbers and hyphens, at most 64 characters."

func configurePamClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *infisical.Client {
	if req.ProviderData == nil {
		return nil
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}

	return client
}

// requirePamMachineIdentity reports summary as an error unless the provider authenticates as a machine identity.
func requirePamMachineIdentity(client *infisical.Client, diags *diag.Diagnostics, summary string) bool {
	if client.Config.IsMachineIdentityAuth {
		return true
	}
	diags.AddError(summary, pamMachineIdentityOnly)
	return false
}

// optionalString returns nil for a null, unknown or empty value.
func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	s := value.ValueString()
	return &s
}

// stringOrNull maps a missing or empty API value to null.
func stringOrNull(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// apiString reads a string from an API response, mapping a missing or empty value to null.
func apiString(values map[string]any, key string) types.String {
	value, _ := values[key].(string)
	return stringOrNull(&value)
}

// apiInt64 reads a number from an API response. JSON numbers decode as float64.
func apiInt64(values map[string]any, key string) types.Int64 {
	if value, ok := values[key].(float64); ok {
		return types.Int64Value(int64(value))
	}
	return types.Int64Null()
}

func apiBool(values map[string]any, key string) types.Bool {
	if value, ok := values[key].(bool); ok {
		return types.BoolValue(value)
	}
	return types.BoolNull()
}

func apiStringList(values map[string]any, key string) types.List {
	items, ok := values[key].([]any)
	if !ok {
		return types.ListNull(types.StringType)
	}
	elements := make([]attr.Value, 0, len(items))
	for _, item := range items {
		value, _ := item.(string)
		elements = append(elements, types.StringValue(value))
	}
	return types.ListValueMust(types.StringType, elements)
}

// priorString reads a secret kept in state, since the API never returns it. Null on import.
func priorString(prior types.Object, name string) types.String {
	if prior.IsNull() || prior.IsUnknown() {
		return types.StringNull()
	}
	if value, ok := prior.Attributes()[name].(types.String); ok {
		return value
	}
	return types.StringNull()
}

// setString adds a configured string to an API payload.
func setString(values map[string]any, key string, value types.String) {
	if s := optionalString(value); s != nil {
		values[key] = *s
	}
}
