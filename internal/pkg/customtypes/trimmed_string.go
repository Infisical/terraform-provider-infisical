// Package customtypes contains custom terraform-plugin-framework attribute
// types used across the provider.
package customtypes

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TrimmedStringType is a string type whose values are considered semantically
// equal when they match after trimming leading/trailing whitespace.
//
// This is intended for attributes whose backing API normalizes the stored value
// by trimming whitespace (e.g. a PEM certificate). Without semantic equality,
// supplying a value with a trailing newline causes Terraform to report a
// "Provider produced inconsistent result after apply" error and a perpetual
// diff, because the value returned by the API differs from the configured value.
type TrimmedStringType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = TrimmedStringType{}

func (t TrimmedStringType) Equal(o attr.Type) bool {
	other, ok := o.(TrimmedStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t TrimmedStringType) String() string {
	return "customtypes.TrimmedStringType"
}

func (t TrimmedStringType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return TrimmedStringValue{StringValue: in}, nil
}

func (t TrimmedStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}

func (t TrimmedStringType) ValueType(_ context.Context) attr.Value {
	return TrimmedStringValue{}
}

// TrimmedStringValue is the value type produced by TrimmedStringType.
type TrimmedStringValue struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuable                   = TrimmedStringValue{}
	_ basetypes.StringValuableWithSemanticEquals = TrimmedStringValue{}
)

// NewTrimmedStringValue returns a known TrimmedStringValue for the given string.
func NewTrimmedStringValue(value string) TrimmedStringValue {
	return TrimmedStringValue{StringValue: basetypes.NewStringValue(value)}
}

func (v TrimmedStringValue) Type(_ context.Context) attr.Type {
	return TrimmedStringType{}
}

func (v TrimmedStringValue) Equal(o attr.Value) bool {
	other, ok := o.(TrimmedStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals treats two values as equal when they are identical after
// trimming surrounding whitespace.
func (v TrimmedStringValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(TrimmedStringValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("expected value type %T but got %T", v, newValuable),
		)
		return false, diags
	}

	// Only compare when both sides are known; null/unknown are handled by the framework.
	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return false, diags
	}

	return strings.TrimSpace(v.ValueString()) == strings.TrimSpace(newValue.ValueString()), diags
}
