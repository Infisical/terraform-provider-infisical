package terraform

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestHttpsPreferredUrlValidator(t *testing.T) {
	tests := []struct {
		name        string
		url         types.String
		wantError   bool
		wantWarning bool
	}{
		{name: "https is clean", url: types.StringValue("https://logs.example.com/ingest")},
		{name: "http warns", url: types.StringValue("http://collector.internal:8080/logs"), wantWarning: true},
		{name: "other schemes are rejected", url: types.StringValue("ftp://logs.example.com"), wantError: true},
		{name: "no host is rejected", url: types.StringValue("https://"), wantError: true},
		{name: "not a url is rejected", url: types.StringValue("logs.example.com"), wantError: true},
		{name: "null is left to the schema", url: types.StringNull()},
		{name: "unknown is left to the schema", url: types.StringUnknown()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			HttpsPreferredUrlValidator.ValidateString(
				context.Background(),
				validator.StringRequest{Path: path.Root("credentials").AtName("url"), ConfigValue: test.url},
				resp,
			)

			if got := resp.Diagnostics.HasError(); got != test.wantError {
				t.Errorf("HasError() = %t, want %t (%v)", got, test.wantError, resp.Diagnostics)
			}
			if got := resp.Diagnostics.WarningsCount() > 0; got != test.wantWarning {
				t.Errorf("warnings = %t, want %t (%v)", got, test.wantWarning, resp.Diagnostics)
			}
		})
	}
}
