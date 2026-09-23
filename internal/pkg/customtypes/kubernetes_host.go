package customtypes

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// KubernetesHostType is a string type whose values are compared after being normalized the
// way the API normalizes them, to "https://host[:port]".
//
// Without it, a configured "https://cluster:6443/" comes back as "https://cluster:6443" and
// every plan re-proposes the same change forever.
type KubernetesHostType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = KubernetesHostType{}

func (t KubernetesHostType) Equal(o attr.Type) bool {
	other, ok := o.(KubernetesHostType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t KubernetesHostType) String() string {
	return "customtypes.KubernetesHostType"
}

func (t KubernetesHostType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return KubernetesHostValue{StringValue: in}, nil
}

func (t KubernetesHostType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
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

func (t KubernetesHostType) ValueType(_ context.Context) attr.Value {
	return KubernetesHostValue{}
}

// KubernetesHostValue is the value type produced by KubernetesHostType.
type KubernetesHostValue struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuable                   = KubernetesHostValue{}
	_ basetypes.StringValuableWithSemanticEquals = KubernetesHostValue{}
)

// NewKubernetesHostValue returns a known KubernetesHostValue for the given string.
func NewKubernetesHostValue(value string) KubernetesHostValue {
	return KubernetesHostValue{StringValue: basetypes.NewStringValue(value)}
}

// NewKubernetesHostNull returns a null KubernetesHostValue.
func NewKubernetesHostNull() KubernetesHostValue {
	return KubernetesHostValue{StringValue: basetypes.NewStringNull()}
}

func (v KubernetesHostValue) Type(_ context.Context) attr.Type {
	return KubernetesHostType{}
}

func (v KubernetesHostValue) Equal(o attr.Value) bool {
	other, ok := o.(KubernetesHostValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v KubernetesHostValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(KubernetesHostValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("expected value type %T but got %T", v, newValuable),
		)
		return false, diags
	}

	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return false, diags
	}

	return NormalizeKubernetesHost(v.ValueString()) == NormalizeKubernetesHost(newValue.ValueString()), diags
}

// NormalizeKubernetesHost mirrors the API's normalization: lowercase https scheme, lowercase
// host, port preserved, no trailing slash. Anything the API would reject outright is returned
// trimmed only, so the rejection surfaces as its own error rather than a silent rewrite.
func NormalizeKubernetesHost(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	lowered := strings.ToLower(trimmed)
	if strings.Contains(lowered, "://") && !strings.HasPrefix(lowered, "https://") {
		return trimmed
	}

	withScheme := trimmed
	if !strings.HasPrefix(lowered, "https://") {
		withScheme = "https://" + trimmed
	}

	parsed, err := url.Parse(withScheme)
	if err != nil || parsed.Host == "" {
		return trimmed
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return trimmed
	}

	return "https://" + strings.ToLower(parsed.Host)
}
