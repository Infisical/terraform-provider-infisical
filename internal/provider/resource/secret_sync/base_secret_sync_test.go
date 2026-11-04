package resource

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// secretSyncTestConfig builds a raw config for the given sync with every attribute null except the
// sync_options attributes passed in.
func secretSyncTestConfig(t *testing.T, r *SecretSyncBaseResource, syncOptions map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	ctx := context.Background()

	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatal(schemaResp.Diagnostics)
	}

	rootType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	syncOptionsType := rootType.AttributeTypes["sync_options"].(tftypes.Object)

	syncOptionsVals := map[string]tftypes.Value{}
	for name, attrType := range syncOptionsType.AttributeTypes {
		syncOptionsVals[name] = tftypes.NewValue(attrType, nil)
	}
	for name, val := range syncOptions {
		syncOptionsVals[name] = val
	}

	rootVals := map[string]tftypes.Value{}
	for name, attrType := range rootType.AttributeTypes {
		rootVals[name] = tftypes.NewValue(attrType, nil)
	}
	rootVals["sync_options"] = tftypes.NewValue(syncOptionsType, syncOptionsVals)

	return tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(rootType, rootVals)}
}

func TestSecretSyncValidateConfigIncludeAllSubFolders(t *testing.T) {
	behavior := func(v string) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	subFolders := func(v bool) tftypes.Value { return tftypes.NewValue(tftypes.Bool, v) }

	for name, tc := range map[string]struct {
		resource             *SecretSyncBaseResource
		includeAllSubFolders tftypes.Value
		initialSyncBehavior  tftypes.Value
		wantError            bool
	}{
		"subfolders with overwrite":        {includeAllSubFolders: subFolders(true), initialSyncBehavior: behavior("overwrite-destination")},
		"subfolders with import":           {includeAllSubFolders: subFolders(true), initialSyncBehavior: behavior("import-prioritize-source"), wantError: true},
		"no subfolders with import":        {includeAllSubFolders: subFolders(false), initialSyncBehavior: behavior("import-prioritize-destination")},
		"unset subfolders with import":     {includeAllSubFolders: tftypes.NewValue(tftypes.Bool, nil), initialSyncBehavior: behavior("import-prioritize-source")},
		"unknown subfolders with import":   {includeAllSubFolders: tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue), initialSyncBehavior: behavior("import-prioritize-source")},
		"subfolders with unknown behavior": {includeAllSubFolders: subFolders(true), initialSyncBehavior: tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
	} {
		t.Run(name, func(t *testing.T) {
			r := NewSecretSyncAwsSecretsManagerResource().(*SecretSyncBaseResource)
			config := secretSyncTestConfig(t, r, map[string]tftypes.Value{
				"include_all_sub_folders": tc.includeAllSubFolders,
				"initial_sync_behavior":   tc.initialSyncBehavior,
			})

			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

			if !tc.wantError {
				if resp.Diagnostics.HasError() {
					t.Errorf("expected no error, got: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), "overwrite-destination") {
				t.Errorf("expected an error about overwrite-destination, got: %v", resp.Diagnostics)
			}
		})
	}
}

// Spacelift has include_all_sub_folders but no initial_sync_behavior, since it always overwrites.
func TestSecretSyncValidateConfigSkipsSyncsWithoutInitialSyncBehavior(t *testing.T) {
	r := NewSecretSyncSpaceliftResource().(*SecretSyncBaseResource)
	config := secretSyncTestConfig(t, r, map[string]tftypes.Value{
		"include_all_sub_folders": tftypes.NewValue(tftypes.Bool, true),
	})

	resp := resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error, got: %v", resp.Diagnostics)
	}
}

// A null sync_options block must not make ValidateConfig fail on the nested lookups.
func TestSecretSyncValidateConfigNullSyncOptions(t *testing.T) {
	ctx := context.Background()
	r := NewSecretSyncAwsSecretsManagerResource().(*SecretSyncBaseResource)
	config := secretSyncTestConfig(t, r, nil)

	rootType := config.Raw.Type().(tftypes.Object)
	rootVals := map[string]tftypes.Value{}
	for name, attrType := range rootType.AttributeTypes {
		rootVals[name] = tftypes.NewValue(attrType, nil)
	}
	config.Raw = tftypes.NewValue(rootType, rootVals)

	resp := resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config}, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error, got: %v", resp.Diagnostics)
	}
}
