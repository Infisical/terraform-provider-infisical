package resource

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func asSecretSyncBaseResource(t *testing.T, r resource.Resource) *SecretSyncBaseResource {
	t.Helper()
	base, ok := r.(*SecretSyncBaseResource)
	if !ok {
		t.Fatalf("expected *SecretSyncBaseResource, got %T", r)
	}
	return base
}

func asObjectType(t *testing.T, typ tftypes.Type) tftypes.Object {
	t.Helper()
	obj, ok := typ.(tftypes.Object)
	if !ok {
		t.Fatalf("expected tftypes.Object, got %T", typ)
	}
	return obj
}

// secretSyncTestConfig builds a raw config for the given sync with every attribute null except the
// include_all_sub_folders value and the sync_options attributes passed in. A nil syncOptions map
// leaves the whole sync_options block null.
func secretSyncTestConfig(t *testing.T, r *SecretSyncBaseResource, includeAllSubFolders tftypes.Value, syncOptions map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	ctx := context.Background()

	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatal(schemaResp.Diagnostics)
	}

	rootType := asObjectType(t, schemaResp.Schema.Type().TerraformType(ctx))
	syncOptionsType := asObjectType(t, rootType.AttributeTypes["sync_options"])

	rootVals := map[string]tftypes.Value{}
	for name, attrType := range rootType.AttributeTypes {
		rootVals[name] = tftypes.NewValue(attrType, nil)
	}
	rootVals["include_all_sub_folders"] = includeAllSubFolders

	if syncOptions != nil {
		syncOptionsVals := map[string]tftypes.Value{}
		for name, attrType := range syncOptionsType.AttributeTypes {
			syncOptionsVals[name] = tftypes.NewValue(attrType, nil)
		}
		for name, val := range syncOptions {
			syncOptionsVals[name] = val
		}
		rootVals["sync_options"] = tftypes.NewValue(syncOptionsType, syncOptionsVals)
	}

	return tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(rootType, rootVals)}
}

func TestSecretSyncValidateConfigIncludeAllSubFolders(t *testing.T) {
	behavior := func(v string) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	subFolders := func(v bool) tftypes.Value { return tftypes.NewValue(tftypes.Bool, v) }

	// The attribute lives on the base resource, so check it across destinations that can and cannot
	// import secrets.
	syncs := map[string]func() resource.Resource{
		"aws secrets manager": NewSecretSyncAwsSecretsManagerResource,
		"github":              NewSecretSyncGithubResource,
		"render":              NewSecretSyncRenderResource,
	}

	for name, tc := range map[string]struct {
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
		for syncName, newSync := range syncs {
			t.Run(name+"/"+syncName, func(t *testing.T) {
				r := asSecretSyncBaseResource(t, newSync())
				config := secretSyncTestConfig(t, r, tc.includeAllSubFolders, map[string]tftypes.Value{
					"initial_sync_behavior": tc.initialSyncBehavior,
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
}

// Spacelift and Azure DevOps have no initial_sync_behavior, since they always overwrite.
func TestSecretSyncValidateConfigSkipsSyncsWithoutInitialSyncBehavior(t *testing.T) {
	for name, newSync := range map[string]func() resource.Resource{
		"spacelift":    NewSecretSyncSpaceliftResource,
		"azure devops": NewSecretSyncAzureDevOpsResource,
	} {
		t.Run(name, func(t *testing.T) {
			r := asSecretSyncBaseResource(t, newSync())
			config := secretSyncTestConfig(t, r, tftypes.NewValue(tftypes.Bool, true), map[string]tftypes.Value{})

			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error, got: %v", resp.Diagnostics)
			}
		})
	}
}

// A null sync_options block must not make ValidateConfig fail on the nested lookup.
func TestSecretSyncValidateConfigNullSyncOptions(t *testing.T) {
	r := asSecretSyncBaseResource(t, NewSecretSyncAwsSecretsManagerResource())
	config := secretSyncTestConfig(t, r, tftypes.NewValue(tftypes.Bool, true), nil)

	resp := resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error, got: %v", resp.Diagnostics)
	}
}

// Spacelift still accepts the deprecated sync_options.include_all_sub_folders. Whichever of the two
// attributes is left out of the config must be planned as the other's value.
func TestSecretSyncModifyPlanDeprecatedIncludeAllSubFolders(t *testing.T) {
	ctx := context.Background()
	boolVal := func(v interface{}) tftypes.Value { return tftypes.NewValue(tftypes.Bool, v) }

	for name, tc := range map[string]struct {
		root, deprecated         tftypes.Value
		wantRoot, wantDeprecated bool
	}{
		"only deprecated true": {root: boolVal(nil), deprecated: boolVal(true), wantRoot: true, wantDeprecated: true},
		"only root true":       {root: boolVal(true), deprecated: boolVal(nil), wantRoot: true, wantDeprecated: true},
		"neither set":          {root: boolVal(nil), deprecated: boolVal(nil)},
		"both true":            {root: boolVal(true), deprecated: boolVal(true), wantRoot: true, wantDeprecated: true},
	} {
		t.Run(name, func(t *testing.T) {
			r := asSecretSyncBaseResource(t, NewSecretSyncSpaceliftResource())
			config := secretSyncTestConfig(t, r, tc.root, map[string]tftypes.Value{
				"include_all_sub_folders": tc.deprecated,
			})

			resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: config.Schema, Raw: config.Raw}}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{
				Config: config,
				Plan:   tfsdk.Plan{Schema: config.Schema, Raw: config.Raw},
				State:  tfsdk.State{Schema: config.Schema, Raw: tftypes.NewValue(config.Raw.Type(), nil)},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			var plan SecretSyncBaseResourceModel
			resp.Diagnostics.Append(resp.Plan.Get(ctx, &plan)...)
			var syncOptions SecretSyncSpaceliftSyncOptionsModel
			resp.Diagnostics.Append(plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			if !tc.wantRoot && !tc.wantDeprecated {
				if plan.IncludeAllSubFolders.ValueBool() || syncOptions.IncludeAllSubFolders.ValueBool() {
					t.Errorf("expected both false, got root=%v deprecated=%v", plan.IncludeAllSubFolders, syncOptions.IncludeAllSubFolders)
				}
				return
			}
			if plan.IncludeAllSubFolders.ValueBool() != tc.wantRoot || syncOptions.IncludeAllSubFolders.ValueBool() != tc.wantDeprecated {
				t.Errorf("expected root=%v deprecated=%v, got root=%v deprecated=%v", tc.wantRoot, tc.wantDeprecated, plan.IncludeAllSubFolders, syncOptions.IncludeAllSubFolders)
			}
		})
	}
}

// A flag that is unknown during planning must leave the omitted one unknown too. Planning it as a known
// value would let the apply-time plan change it, which Terraform rejects as an inconsistent final plan.
func TestSecretSyncModifyPlanDeprecatedIncludeAllSubFoldersUnknown(t *testing.T) {
	ctx := context.Background()
	unknown := tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)
	null := tftypes.NewValue(tftypes.Bool, nil)

	for name, tc := range map[string]struct {
		root, deprecated   tftypes.Value
		unknownSyncOptions bool
		checkPath          path.Path
	}{
		"unknown deprecated, omitted root": {root: null, deprecated: unknown, checkPath: path.Root("include_all_sub_folders")},
		"unknown root, omitted deprecated": {root: unknown, deprecated: null, checkPath: deprecatedIncludeAllSubFoldersPath},
		"unknown sync_options, omitted root": {
			root: null, deprecated: null, unknownSyncOptions: true, checkPath: path.Root("include_all_sub_folders"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := asSecretSyncBaseResource(t, NewSecretSyncSpaceliftResource())
			config := secretSyncTestConfig(t, r, tc.root, map[string]tftypes.Value{
				"include_all_sub_folders": tc.deprecated,
			})

			if tc.unknownSyncOptions {
				rootType := asObjectType(t, config.Raw.Type())
				rootVals := map[string]tftypes.Value{}
				if err := config.Raw.As(&rootVals); err != nil {
					t.Fatal(err)
				}
				rootVals["sync_options"] = tftypes.NewValue(rootType.AttributeTypes["sync_options"], tftypes.UnknownValue)
				config.Raw = tftypes.NewValue(rootType, rootVals)
			}

			// The framework fills static defaults into the plan before ModifyPlan runs.
			plan := tfsdk.Plan{Schema: config.Schema, Raw: config.Raw}
			if diags := plan.SetAttribute(ctx, path.Root("include_all_sub_folders"), false); diags.HasError() {
				t.Fatal(diags)
			}
			if !tc.unknownSyncOptions {
				if diags := plan.SetAttribute(ctx, deprecatedIncludeAllSubFoldersPath, false); diags.HasError() {
					t.Fatal(diags)
				}
			}
			if !tc.root.IsKnown() {
				if diags := plan.SetAttribute(ctx, path.Root("include_all_sub_folders"), types.BoolUnknown()); diags.HasError() {
					t.Fatal(diags)
				}
			}
			if !tc.unknownSyncOptions && !tc.deprecated.IsKnown() {
				if diags := plan.SetAttribute(ctx, deprecatedIncludeAllSubFoldersPath, types.BoolUnknown()); diags.HasError() {
					t.Fatal(diags)
				}
			}

			resp := resource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{
				Config: config,
				Plan:   plan,
				State:  tfsdk.State{Schema: config.Schema, Raw: tftypes.NewValue(config.Raw.Type(), nil)},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			var got types.Bool
			if diags := resp.Plan.GetAttribute(ctx, tc.checkPath, &got); diags.HasError() {
				t.Fatal(diags)
			}
			if !got.IsUnknown() {
				t.Errorf("expected %s to be planned as unknown, got %v", tc.checkPath, got)
			}
		})
	}
}

func TestSecretSyncValidateConfigConflictingDeprecatedIncludeAllSubFolders(t *testing.T) {
	boolVal := func(v interface{}) tftypes.Value { return tftypes.NewValue(tftypes.Bool, v) }

	for name, tc := range map[string]struct {
		root, deprecated tftypes.Value
		wantError        bool
	}{
		"root false, deprecated true": {root: boolVal(false), deprecated: boolVal(true), wantError: true},
		"root true, deprecated false": {root: boolVal(true), deprecated: boolVal(false), wantError: true},
		"both true":                   {root: boolVal(true), deprecated: boolVal(true)},
		"only deprecated":             {root: boolVal(nil), deprecated: boolVal(true)},
		"unknown root":                {root: tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue), deprecated: boolVal(true)},
	} {
		t.Run(name, func(t *testing.T) {
			r := asSecretSyncBaseResource(t, NewSecretSyncSpaceliftResource())
			config := secretSyncTestConfig(t, r, tc.root, map[string]tftypes.Value{
				"include_all_sub_folders": tc.deprecated,
			})

			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("expected error=%v, got: %v", tc.wantError, resp.Diagnostics)
			}
		})
	}
}
