package resource

import (
	"context"
	"testing"

	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// token_auth is an object with no attributes, an odd shape worth pinning.
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

func TestNormalizeKubernetesHostMatchesTheApi(t *testing.T) {
	canonical := "https://cluster.example.com:6443"

	for _, raw := range []string{
		canonical,
		canonical + "/",
		"HTTPS://cluster.example.com:6443",
		"cluster.example.com:6443",
		"  cluster.example.com:6443  ",
		"https://CLUSTER.example.com:6443",
	} {
		if got := customtypes.NormalizeKubernetesHost(raw); got != canonical {
			t.Errorf("%q normalized to %q, want %q", raw, got, canonical)
		}
	}
}

// Anything the API rejects is left alone, so the rejection is reported rather than hidden.
func TestNormalizeKubernetesHostLeavesRejectableInputAlone(t *testing.T) {
	for _, raw := range []string{
		"http://cluster.example.com:6443",
		"https://cluster.example.com:6443/api",
		"https://user:pass@cluster.example.com:6443",
	} {
		if got := customtypes.NormalizeKubernetesHost(raw); got != raw {
			t.Errorf("%q was rewritten to %q, want it untouched", raw, got)
		}
	}
}

func TestKeepUnsetSetPreservesTheConfiguredFormOfUnset(t *testing.T) {
	var diags diag.Diagnostics

	if got := keepUnsetSet("", types.SetNull(types.StringType), &diags); !got.IsNull() {
		t.Errorf("a null prior should stay null, got %v", got)
	}

	empty, _ := types.SetValue(types.StringType, []attr.Value{})
	if got := keepUnsetSet("", empty, &diags); got.IsNull() {
		t.Error("an explicitly empty set should stay an empty set, got null")
	}

	if got := keepUnsetSet("a,b", types.SetNull(types.StringType), &diags); len(got.Elements()) != 2 {
		t.Errorf("expected two elements, got %d", len(got.Elements()))
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestKeepUnsetStringPreservesNull(t *testing.T) {
	if got := keepUnsetString("", types.StringNull()); !got.IsNull() {
		t.Errorf("a null prior should stay null, got %v", got)
	}
	if got := keepUnsetString("aud", types.StringNull()); got.ValueString() != "aud" {
		t.Errorf("expected the API value, got %v", got)
	}
}

// An empty response means "unset" only when our value was unset too. A value that was set and
// came back empty was cleared outside Terraform and has to read as drift.
func TestKeepUnsetSetReportsAnOutOfBandClear(t *testing.T) {
	var diags diag.Diagnostics

	populated, _ := types.SetValue(types.StringType, []attr.Value{types.StringValue("infisical")})
	if got := keepUnsetSet("", populated, &diags); !got.IsNull() {
		t.Errorf("a cleared allowlist should read as null, got %v", got)
	}

	empty, _ := types.SetValue(types.StringType, []attr.Value{})
	if got := keepUnsetSet("", empty, &diags); got.IsNull() {
		t.Error("an explicitly empty set should stay an empty set, got null")
	}
	if got := keepUnsetSet("", types.SetNull(types.StringType), &diags); !got.IsNull() {
		t.Errorf("a null prior should stay null, got %v", got)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestKeepUnsetStringReportsAnOutOfBandClear(t *testing.T) {
	if got := keepUnsetString("", types.StringValue("my-audience")); !got.IsNull() {
		t.Errorf("a cleared audience should read as null, got %v", got)
	}
	if got := keepUnsetString("", types.StringNull()); !got.IsNull() {
		t.Errorf("a null prior should stay null, got %v", got)
	}
	if got := keepUnsetString("", types.StringValue("")); got.IsNull() {
		t.Error("an explicitly empty string should stay empty, got null")
	}
	if got := keepUnsetString("aud", types.StringNull()); got.ValueString() != "aud" {
		t.Errorf("expected the API value, got %v", got)
	}
}
