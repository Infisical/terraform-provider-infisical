package terraform

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AccessApproverInput struct {
	Type     types.String
	ID       types.String
	Name     types.String
	Sequence types.Int64
}

type AccessApproverOutput struct {
	Type     string
	ID       string
	Name     string
	Sequence int64
}

func ValidateAndMapApprovers(approvers []AccessApproverInput, diagnostics *diag.Diagnostics) ([]AccessApproverOutput, bool) {
	var result []AccessApproverOutput
	for _, el := range approvers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				diagnostics.AddError(
					"Field username is required for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return nil, false
			}
			if !el.ID.IsNull() {
				diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return nil, false
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				diagnostics.AddError(
					"Field ID is required for group approvers",
					"Must provide ID for group approvers",
				)
				return nil, false
			}
			if !el.Name.IsNull() {
				diagnostics.AddError(
					"Field username cannot be used for group approvers",
					"Must provide ID for group approvers",
				)
				return nil, false
			}
		}

		result = append(result, AccessApproverOutput{
			ID:       el.ID.ValueString(),
			Name:     el.Name.ValueString(),
			Type:     el.Type.ValueString(),
			Sequence: el.Sequence.ValueInt64(),
		})
	}

	if len(result) == 0 {
		diagnostics.AddError(
			"No approvers provided",
			"Must provide at least one approver, either group IDs or user usernames must be provided",
		)
		return nil, false
	}

	return result, true
}

type BypasserInput struct {
	Type types.String
	ID   types.String
	Name types.String
}

type BypasserOutput struct {
	Type string
	ID   string
	Name string
}

func ValidateAndMapBypassers(bypassers []BypasserInput, diagnostics *diag.Diagnostics) ([]BypasserOutput, bool) {
	var result []BypasserOutput
	for _, el := range bypassers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				diagnostics.AddError(
					"Field username is required for user bypassers",
					"Must provide username for user bypassers. By default, this is the email",
				)
				return nil, false
			}
			if !el.ID.IsNull() {
				diagnostics.AddError(
					"Field ID cannot be used for user bypassers",
					"Must provide username for user bypassers. By default, this is the email",
				)
				return nil, false
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				diagnostics.AddError(
					"Field ID is required for group bypassers",
					"Must provide ID for group bypassers",
				)
				return nil, false
			}
			if !el.Name.IsNull() {
				diagnostics.AddError(
					"Field username cannot be used for group bypassers",
					"Must provide ID for group bypassers",
				)
				return nil, false
			}
		}

		result = append(result, BypasserOutput{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}
	return result, true
}

func OptionalStringPointer(val types.String) *string {
	if !val.IsNull() && !val.IsUnknown() {
		v := val.ValueString()
		return &v
	}
	return nil
}

func StringPointerToTypesString(val *string) types.String {
	if val != nil {
		return types.StringValue(*val)
	}
	return types.StringNull()
}
