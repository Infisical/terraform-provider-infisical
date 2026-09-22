package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// token_auth carries no arguments, so it round-trips as an object with no attributes. That is an
// unusual enough shape to be worth pinning: if the framework ever stops reflecting it, every plan
// against a token-auth gateway fails on a value conversion error rather than anything readable.
func TestTokenAuthModelRoundTripsAsEmptyObject(t *testing.T) {
	ctx := context.Background()

	object, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{}, gatewayTokenAuthModel{})
	if diags.HasError() {
		t.Fatalf("expected the empty token_auth object to convert, got: %v", diags)
	}

	var back gatewayTokenAuthModel
	if diags := object.As(ctx, &back, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("expected the empty token_auth object to convert back, got: %v", diags)
	}
}

func TestCsvFromSetSortsAndJoins(t *testing.T) {
	ctx := context.Background()

	set, diags := types.SetValue(types.StringType, []attr.Value{
		types.StringValue("us-central1-b"),
		types.StringValue("us-central1-a"),
	})
	if diags.HasError() {
		t.Fatalf("expected the set to build, got: %v", diags)
	}

	csv, diags := csvFromSet(ctx, set)
	if diags.HasError() {
		t.Fatalf("expected the set to render, got: %v", diags)
	}
	if csv != "us-central1-a,us-central1-b" {
		t.Errorf("expected a sorted comma-separated list, got %q", csv)
	}
}

// An unknown set is what a computed allowlist the configuration left out looks like during plan, and
// it has to render as an empty allowlist rather than reaching the API as the string "<unknown>".
func TestCsvFromSetTreatsNullAndUnknownAsEmpty(t *testing.T) {
	ctx := context.Background()

	for name, set := range map[string]types.Set{
		"null":    types.SetNull(types.StringType),
		"unknown": types.SetUnknown(types.StringType),
	} {
		csv, diags := csvFromSet(ctx, set)
		if diags.HasError() {
			t.Fatalf("%s: expected no error, got: %v", name, diags)
		}
		if csv != "" {
			t.Errorf("%s: expected an empty string, got %q", name, csv)
		}
	}
}

func TestSetFromCsvDropsEmptyEntries(t *testing.T) {
	set, diags := setFromCsv("")
	if diags.HasError() {
		t.Fatalf("expected an empty csv to convert, got: %v", diags)
	}
	if len(set.Elements()) != 0 {
		t.Errorf("expected no elements from an empty csv, got %d", len(set.Elements()))
	}

	set, diags = setFromCsv(" infisical , kube-system ")
	if diags.HasError() {
		t.Fatalf("expected the csv to convert, got: %v", diags)
	}
	if len(set.Elements()) != 2 {
		t.Fatalf("expected two elements, got %d", len(set.Elements()))
	}
}
