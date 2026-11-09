package resource

import (
	"context"
	"reflect"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestPaloAltoNetworksCertificateNameRule(t *testing.T) {
	cases := map[string]bool{
		"INF-{{shortCertificateId}}":       true,
		"infisical-cert":                   true,
		"{{commonName}}":                   true,
		"infisical-{{certificateId}}":      false,
		"INF.{{shortCertificateId}}":       false,
		"this-name-is-thirty-two-chars-xx": false,
	}
	for schema, valid := range cases {
		problems := paloAltoNetworksCertificateNameRule.validate(schema)
		if valid && len(problems) > 0 {
			t.Errorf("%q: unexpected problems %v", schema, problems)
		}
		if !valid && len(problems) == 0 {
			t.Errorf("%q: expected a problem", schema)
		}
	}
}

func TestValidatePaloAltoNetworksNamePlaceholders(t *testing.T) {
	cases := map[string]bool{
		"INF-{{shortCertificateId}}": true,
		"INF-{{commonName}}":         true,
		"INF-{{certificateId}}":      true,
		"infisical-cert":             true,
		"INF-{{profileId}}":          false,
		"INF-{{applicationName}}":    false,
	}
	for schema, valid := range cases {
		syncOptions := types.ObjectValueMust(
			map[string]attr.Type{"certificate_name_schema": types.StringType},
			map[string]attr.Value{"certificate_name_schema": types.StringValue(schema)},
		)
		var diags diag.Diagnostics
		validatePaloAltoNetworksNamePlaceholders(context.Background(), CertificateSyncBaseResourceModel{SyncOptions: syncOptions}, &diags)
		if diags.HasError() == valid {
			t.Errorf("%q: valid=%v, got diagnostics %v", schema, valid, diags)
		}
	}
}

func paloAltoNetworksValidateConfig(t *testing.T, r resource.Resource, nameSchema string, orderIDs ...string) diag.Diagnostics {
	t.Helper()
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", schemaResp.Diagnostics)
	}
	s := schemaResp.Schema

	plan := tfsdk.Plan{Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil), Schema: s}
	var diags diag.Diagnostics
	diags.Append(plan.SetAttribute(ctx, path.Root(attrSyncOptions).AtName("certificate_name_schema"), nameSchema)...)
	diags.Append(plan.SetAttribute(ctx, path.Root(attrCertificateFilters).AtName("certificate_order_ids"), orderIDs)...)
	if diags.HasError() {
		t.Fatalf("building the config: %v", diags)
	}

	resp := &resource.ValidateConfigResponse{}
	validator, ok := r.(resource.ResourceWithValidateConfig)
	if !ok {
		t.Fatalf("%T does not implement ResourceWithValidateConfig", r)
	}
	validator.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Raw: plan.Raw, Schema: s}}, resp)
	return resp.Diagnostics
}

func TestPaloAltoNetworksResourcesValidateConfig(t *testing.T) {
	cases := []struct {
		name       string
		resource   resource.Resource
		nameSchema string
		orderIDs   []string
		wantError  bool
	}{
		{"multi valid", NewCertificateSyncPaloAltoNetworksResource(), "INF-{{shortCertificateId}}", []string{"a", "b"}, false},
		{"multi name too long", NewCertificateSyncPaloAltoNetworksResource(), "infisical-{{certificateId}}", []string{"a"}, true},
		{"multi placeholder without identifier", NewCertificateSyncPaloAltoNetworksResource(), "INF-{{applicationName}}", []string{"a"}, true},
		{"multi static name with two orders", NewCertificateSyncPaloAltoNetworksResource(), "infisical-cert", []string{"a", "b"}, true},
		{"profile valid", NewCertificateSyncPaloAltoNetworksSslTlsProfileResource(), "infisical-cert", []string{"a"}, false},
		{"profile name too long", NewCertificateSyncPaloAltoNetworksSslTlsProfileResource(), "infisical-{{certificateId}}", []string{"a"}, true},
		{"profile two orders", NewCertificateSyncPaloAltoNetworksSslTlsProfileResource(), "INF-{{shortCertificateId}}", []string{"a", "b"}, true},
	}
	for _, c := range cases {
		diags := paloAltoNetworksValidateConfig(t, c.resource, c.nameSchema, c.orderIDs...)
		if diags.HasError() != c.wantError {
			t.Errorf("%s: wantError=%v, got diagnostics %v", c.name, c.wantError, diags)
		}
	}
}

func trimmed(value string) customtypes.TrimmedStringValue {
	return customtypes.TrimmedStringValue{StringValue: types.StringValue(value)}
}

func TestPaloAltoNetworksSyncOptionsMapping(t *testing.T) {
	ctx := context.Background()
	syncOptions := types.ObjectValueMust(certificateSyncPaloAltoNetworksSyncOptionsAttrTypes, map[string]attr.Value{
		"certificate_name_schema":  trimmed("INF-{{shortCertificateId}}"),
		"can_remove_certificates":  types.BoolValue(false),
		"include_root_ca":          types.BoolValue(true),
		"preserve_item_on_renewal": types.BoolValue(false),
	})

	request, diags := readPaloAltoNetworksSyncOptionsFromPlan(ctx, CertificateSyncBaseResourceModel{SyncOptions: syncOptions})
	if diags.HasError() {
		t.Fatalf("readPaloAltoNetworksSyncOptionsFromPlan: %v", diags)
	}
	want := map[string]interface{}{
		"certificateNameSchema": "INF-{{shortCertificateId}}",
		"canRemoveCertificates": false,
		"includeRootCa":         true,
		"preserveItemOnRenewal": false,
	}
	if !reflect.DeepEqual(request, want) {
		t.Errorf("request = %v, want %v", request, want)
	}

	state, diags := readPaloAltoNetworksSyncOptionsFromApi(ctx, infisical.CertificateSync{SyncOptions: request})
	if diags.HasError() {
		t.Fatalf("readPaloAltoNetworksSyncOptionsFromApi: %v", diags)
	}
	if !state.Equal(syncOptions) {
		t.Errorf("state = %v, want %v", state, syncOptions)
	}
}

func TestPaloAltoNetworksSslTlsProfileDestinationConfigMapping(t *testing.T) {
	ctx := context.Background()
	r, ok := NewCertificateSyncPaloAltoNetworksSslTlsProfileResource().(*CertificateSyncBaseResource)
	if !ok {
		t.Fatal("NewCertificateSyncPaloAltoNetworksSslTlsProfileResource did not return a *CertificateSyncBaseResource")
	}
	destinationConfig := types.ObjectValueMust(certificateSyncPaloAltoNetworksSslTlsProfileDestinationConfigAttrTypes, map[string]attr.Value{
		"template":                     trimmed("tmpl"),
		"push_to_devices":              types.BoolValue(false),
		"ssl_tls_service_profile_name": trimmed("web-profile"),
		"ssl_tls_service_profile_vsys": trimmed("vsys1"),
	})

	request, diags := r.ReadDestinationConfigFromPlan(ctx, CertificateSyncBaseResourceModel{DestinationConfig: destinationConfig})
	if diags.HasError() {
		t.Fatalf("ReadDestinationConfigFromPlan: %v", diags)
	}
	want := map[string]interface{}{
		"template":                 "tmpl",
		"pushToDevices":            false,
		"sslTlsServiceProfileName": "web-profile",
		"sslTlsServiceProfileVsys": "vsys1",
	}
	if !reflect.DeepEqual(request, want) {
		t.Errorf("request = %v, want %v", request, want)
	}

	state, diags := r.ReadDestinationConfigFromApi(ctx, infisical.CertificateSync{DestinationConfig: request})
	if diags.HasError() {
		t.Fatalf("ReadDestinationConfigFromApi: %v", diags)
	}
	if !state.Equal(destinationConfig) {
		t.Errorf("state = %v, want %v", state, destinationConfig)
	}
}

func TestPaloAltoNetworksDestinationConfigMapping(t *testing.T) {
	ctx := context.Background()
	r, ok := NewCertificateSyncPaloAltoNetworksResource().(*CertificateSyncBaseResource)
	if !ok {
		t.Fatal("NewCertificateSyncPaloAltoNetworksResource did not return a *CertificateSyncBaseResource")
	}
	destinationConfig := types.ObjectValueMust(certificateSyncPaloAltoNetworksDestinationConfigAttrTypes, map[string]attr.Value{
		"template":        trimmed("tmpl"),
		"push_to_devices": types.BoolValue(false),
	})

	request, diags := r.ReadDestinationConfigFromPlan(ctx, CertificateSyncBaseResourceModel{DestinationConfig: destinationConfig})
	if diags.HasError() {
		t.Fatalf("ReadDestinationConfigFromPlan: %v", diags)
	}
	want := map[string]interface{}{"template": "tmpl", "pushToDevices": false}
	if !reflect.DeepEqual(request, want) {
		t.Errorf("request = %v, want %v", request, want)
	}

	state, diags := r.ReadDestinationConfigFromApi(ctx, infisical.CertificateSync{DestinationConfig: request})
	if diags.HasError() {
		t.Fatalf("ReadDestinationConfigFromApi: %v", diags)
	}
	if !state.Equal(destinationConfig) {
		t.Errorf("state = %v, want %v", state, destinationConfig)
	}
}

func TestPaloAltoNetworksNameRulePlaceholders(t *testing.T) {
	problems := paloAltoNetworksCertificateNameRule.validate("{{profileId}}")
	if len(problems) != 1 || strings.Contains(problems[0], "{{profileId}}") {
		t.Errorf("problems = %v, want one that does not offer {{profileId}}", problems)
	}
}
