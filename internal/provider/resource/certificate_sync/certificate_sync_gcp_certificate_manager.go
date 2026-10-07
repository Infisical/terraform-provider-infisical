package resource

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	gcpCertificateManagerScopeDefault    = "default"
	gcpCertificateManagerScopeEdgeCache  = "edge-cache"
	gcpCertificateManagerScopeAllRegions = "all-regions"
	gcpCertificateManagerScopeClientAuth = "client-auth"

	gcpCertificateManagerGlobalLocation                = "global"
	gcpCertificateManagerMaxUserLabels                 = 62
	gcpCertificateManagerMaxCertificatesPerMapEntry    = 4
	gcpCertificateManagerManagedByLabelKey             = "managed-by"
	gcpCertificateManagerCertificateIDLabelKey         = "infisical-certificate-id"
	gcpCertificateManagerCertificateIDPlaceholder      = "{{certificateId}}"
	gcpCertificateManagerShortCertificateIDPlaceholder = "{{shortCertificateId}}"
)

var (
	gcpCertificateManagerLabelKeyPattern   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
	gcpCertificateManagerLabelValuePattern = regexp.MustCompile(`^[a-z0-9_-]{0,63}$`)
	gcpCertificateManagerLeadingLetter     = regexp.MustCompile(`^[a-z]`)
)

type CertificateSyncGcpCertificateManagerCertificateMapBindingModel struct {
	CertificateMap types.String `tfsdk:"certificate_map"`
	Hostname       types.String `tfsdk:"hostname"`
}

type CertificateSyncGcpCertificateManagerDestinationConfigModel struct {
	GcpProjectID          types.String `tfsdk:"gcp_project_id"`
	Location              types.String `tfsdk:"location"`
	Scope                 types.String `tfsdk:"scope"`
	CertificateMapBinding types.Object `tfsdk:"certificate_map_binding"`
}

type CertificateSyncGcpCertificateManagerSyncOptionsModel struct {
	CertificateNameSchema customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa         types.Bool                     `tfsdk:"include_root_ca"`
	PreserveItemOnRenewal types.Bool                     `tfsdk:"preserve_item_on_renewal"`
	Labels                types.Map                      `tfsdk:"labels"`
}

var certificateSyncGcpCertificateManagerCertificateMapBindingAttrTypes = map[string]attr.Type{
	"certificate_map": types.StringType,
	"hostname":        types.StringType,
}

var certificateSyncGcpCertificateManagerDestinationConfigAttrTypes = map[string]attr.Type{
	"gcp_project_id":          types.StringType,
	"location":                types.StringType,
	"scope":                   types.StringType,
	"certificate_map_binding": types.ObjectType{AttrTypes: certificateSyncGcpCertificateManagerCertificateMapBindingAttrTypes},
}

var certificateSyncGcpCertificateManagerSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema":  customtypes.TrimmedStringType{},
	"can_remove_certificates":  types.BoolType,
	"include_root_ca":          types.BoolType,
	"preserve_item_on_renewal": types.BoolType,
	"labels":                   types.MapType{ElemType: types.StringType},
}

func NewCertificateSyncGcpCertificateManagerResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:              infisical.CertificateSyncAppGCPCertificateManager,
		SyncName:         "GCP Certificate Manager",
		ResourceTypeName: "_certificate_sync_gcp_certificate_manager",
		AppConnection:    infisical.AppConnectionAppGCP,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"gcp_project_id": schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
				Description:   "The ID of the GCP project to sync certificates to.",
			},
			"location": schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
				Description:   "The Certificate Manager location to create certificates in, either global or a region ID such as us-central1.",
			},
			"scope": schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Optional:      true,
				Computed:      true,
				Description:   "The scope of the synced certificates: default, edge-cache, all-regions or client-auth. Defaults to default.",
				Default:       stringdefault.StaticString(gcpCertificateManagerScopeDefault),
				Validators: []validator.String{
					stringvalidator.OneOf(
						gcpCertificateManagerScopeDefault,
						gcpCertificateManagerScopeEdgeCache,
						gcpCertificateManagerScopeAllRegions,
						gcpCertificateManagerScopeClientAuth,
					),
				},
			},
			"certificate_map_binding": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Adds the synced certificates to an entry in an existing certificate map. Requires the global location and the default scope.",
				Attributes: map[string]schema.Attribute{
					"certificate_map": schema.StringAttribute{
						Required:    true,
						Description: "The name of the certificate map to add the certificates to.",
					},
					"hostname": schema.StringAttribute{
						Validators:  []validator.String{notBlank()},
						Optional:    true,
						Description: "The hostname of the certificate map entry, such as www.example.com or *.example.com. Leave unset to use the primary entry.",
					},
				},
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for synced certificates. Must start with a lowercase letter and include the {{certificateId}} or {{shortCertificateId}} placeholder, and compiled names may contain only lowercase letters, digits and hyphens (1-63 characters). Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove certificates from GCP Certificate Manager when they are no longer managed in Infisical. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
			"include_root_ca": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to include the root CA certificate in the synced certificate chain. Defaults to false.",
				Default:     booldefault.StaticBool(false),
			},
			"preserve_item_on_renewal": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to update the existing GCP certificate when a certificate is renewed instead of creating a new one. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: fmt.Sprintf("Labels to set on the synced GCP certificates, up to %d. The keys %s and %s are reserved.", gcpCertificateManagerMaxUserLabels, gcpCertificateManagerManagedByLabelKey, gcpCertificateManagerCertificateIDLabelKey),
				Validators: []validator.Map{
					mapvalidator.SizeBetween(1, gcpCertificateManagerMaxUserLabels),
					mapvalidator.KeysAre(
						stringvalidator.RegexMatches(gcpCertificateManagerLabelKeyPattern, "must start with a lowercase letter and contain only lowercase letters, digits, hyphens and underscores (up to 63 characters)"),
						stringvalidator.NoneOf(gcpCertificateManagerManagedByLabelKey, gcpCertificateManagerCertificateIDLabelKey),
					),
					mapvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(gcpCertificateManagerLabelValuePattern, "must contain only lowercase letters, digits, hyphens and underscores (up to 63 characters)"),
					),
				},
			},
		},

		ValidateConfigFunc: func(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
			if !config.SyncOptions.IsNull() && !config.SyncOptions.IsUnknown() {
				var syncOptions CertificateSyncGcpCertificateManagerSyncOptionsModel
				diags.Append(config.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})...)
				if diags.HasError() {
					return
				}
				if !syncOptions.CertificateNameSchema.IsNull() && !syncOptions.CertificateNameSchema.IsUnknown() {
					nameSchema := strings.TrimSpace(syncOptions.CertificateNameSchema.ValueString())
					namePath := path.Root(attrSyncOptions).AtName("certificate_name_schema")
					if !strings.Contains(nameSchema, gcpCertificateManagerCertificateIDPlaceholder) && !strings.Contains(nameSchema, gcpCertificateManagerShortCertificateIDPlaceholder) {
						diags.AddAttributeError(namePath, "Invalid certificate name schema", "The certificate name schema must include the {{certificateId}} or {{shortCertificateId}} placeholder.")
					}
					if !gcpCertificateManagerLeadingLetter.MatchString(nameSchema) {
						diags.AddAttributeError(namePath, "Invalid certificate name schema", `The certificate name schema must start with a lowercase letter, for example "infisical-{{certificateId}}".`)
					}
				}
			}

			if config.DestinationConfig.IsNull() || config.DestinationConfig.IsUnknown() {
				return
			}

			var destinationConfig CertificateSyncGcpCertificateManagerDestinationConfigModel
			diags.Append(config.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return
			}

			locationKnown := !destinationConfig.Location.IsUnknown()
			isGlobal := destinationConfig.Location.ValueString() == gcpCertificateManagerGlobalLocation
			scopeKnown := !destinationConfig.Scope.IsUnknown()
			scope := destinationConfig.Scope.ValueString()
			hasBinding := !destinationConfig.CertificateMapBinding.IsNull() && !destinationConfig.CertificateMapBinding.IsUnknown()
			destinationPath := path.Root(attrDestinationConfig)

			if scopeKnown && locationKnown && scope == gcpCertificateManagerScopeAllRegions && !isGlobal {
				diags.AddAttributeError(destinationPath.AtName("scope"), "Invalid scope", `The all-regions scope is only available when location is "global".`)
			}

			if !hasBinding {
				return
			}

			if locationKnown && !isGlobal {
				diags.AddAttributeError(destinationPath.AtName("certificate_map_binding"), "Invalid certificate map binding", `A certificate map binding is only available when location is "global".`)
			}
			if scopeKnown && !destinationConfig.Scope.IsNull() && scope != gcpCertificateManagerScopeDefault {
				diags.AddAttributeError(destinationPath.AtName("certificate_map_binding"), "Invalid certificate map binding", `A certificate map binding requires the "default" scope.`)
			}

			if config.CertificateFilters.IsNull() || config.CertificateFilters.IsUnknown() {
				return
			}
			var filters certificateFiltersModel
			diags.Append(config.CertificateFilters.As(ctx, &filters, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return
			}
			if !filters.CertificateIDs.IsNull() && !filters.CertificateIDs.IsUnknown() && len(filters.CertificateIDs.Elements()) > gcpCertificateManagerMaxCertificatesPerMapEntry {
				diags.AddAttributeError(path.Root(attrCertificateFilters).AtName("certificate_ids"), "Too many certificates", fmt.Sprintf("A certificate map binding supports up to %d certificates, which is the GCP limit for one certificate map entry.", gcpCertificateManagerMaxCertificatesPerMapEntry))
			}
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncGcpCertificateManagerSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			payload := map[string]interface{}{
				"certificateNameSchema": syncOptions.CertificateNameSchema.ValueString(),
				"canRemoveCertificates": syncOptions.CanRemoveCertificates.ValueBool(),
				"includeRootCa":         syncOptions.IncludeRootCa.ValueBool(),
				"preserveItemOnRenewal": syncOptions.PreserveItemOnRenewal.ValueBool(),
				"canImportCertificates": false,
				"labels":                []map[string]interface{}{},
			}

			if !syncOptions.Labels.IsNull() && !syncOptions.Labels.IsUnknown() {
				labels := map[string]string{}
				diags.Append(syncOptions.Labels.ElementsAs(ctx, &labels, false)...)
				if diags.HasError() {
					return nil, diags
				}
				keys := make([]string, 0, len(labels))
				for key := range labels {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				labelsPayload := make([]map[string]interface{}, 0, len(keys))
				for _, key := range keys {
					labelsPayload = append(labelsPayload, map[string]interface{}{"key": key, "value": labels[key]})
				}
				payload["labels"] = labelsPayload
			}

			return payload, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncGcpCertificateManagerSyncOptionsAttrTypes), diags
			}

			labels := types.MapNull(types.StringType)
			if rawLabels, ok := certificateSync.SyncOptions["labels"].([]interface{}); ok && len(rawLabels) > 0 {
				labelValues := make(map[string]attr.Value, len(rawLabels))
				for i, rawLabel := range rawLabels {
					label, ok := rawLabel.(map[string]interface{})
					if !ok {
						diags.AddError("Invalid label type", fmt.Sprintf("Expected label %d to be an object but got something else", i))
						return types.ObjectNull(certificateSyncGcpCertificateManagerSyncOptionsAttrTypes), diags
					}
					key := stringFromMap(label, "key", &diags)
					value, _ := label["value"].(string)
					if diags.HasError() {
						return types.ObjectNull(certificateSyncGcpCertificateManagerSyncOptionsAttrTypes), diags
					}
					labelValues[key.ValueString()] = types.StringValue(value)
				}
				var mapDiags diag.Diagnostics
				labels, mapDiags = types.MapValue(types.StringType, labelValues)
				diags.Append(mapDiags...)
				if diags.HasError() {
					return types.ObjectNull(certificateSyncGcpCertificateManagerSyncOptionsAttrTypes), diags
				}
			}

			obj, objDiags := types.ObjectValue(certificateSyncGcpCertificateManagerSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema":  certificateNameSchema,
				"can_remove_certificates":  boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
				"include_root_ca":          boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
				"preserve_item_on_renewal": boolFromMap(certificateSync.SyncOptions, "preserveItemOnRenewal", true),
				"labels":                   labels,
			})
			diags.Append(objDiags...)
			return obj, diags
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncGcpCertificateManagerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			payload := map[string]interface{}{
				"gcpProjectId": destinationConfig.GcpProjectID.ValueString(),
				"location":     destinationConfig.Location.ValueString(),
			}
			setOptionalString(payload, "scope", destinationConfig.Scope)

			if !destinationConfig.CertificateMapBinding.IsNull() && !destinationConfig.CertificateMapBinding.IsUnknown() {
				var binding CertificateSyncGcpCertificateManagerCertificateMapBindingModel
				diags.Append(destinationConfig.CertificateMapBinding.As(ctx, &binding, basetypes.ObjectAsOptions{})...)
				if diags.HasError() {
					return nil, diags
				}
				bindingPayload := map[string]interface{}{
					"certificateMap": binding.CertificateMap.ValueString(),
				}
				setOptionalString(bindingPayload, "hostname", binding.Hostname)
				payload["certificateMapBinding"] = bindingPayload
			}

			return payload, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			gcpProjectID := stringFromMap(certificateSync.DestinationConfig, "gcpProjectId", &diags)
			location := stringFromMap(certificateSync.DestinationConfig, "location", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncGcpCertificateManagerDestinationConfigAttrTypes), diags
			}

			scope := types.StringValue(gcpCertificateManagerScopeDefault)
			if value, ok := certificateSync.DestinationConfig["scope"].(string); ok && value != "" {
				scope = types.StringValue(value)
			}

			binding := types.ObjectNull(certificateSyncGcpCertificateManagerCertificateMapBindingAttrTypes)
			if rawBinding, ok := certificateSync.DestinationConfig["certificateMapBinding"].(map[string]interface{}); ok {
				certificateMap := stringFromMap(rawBinding, "certificateMap", &diags)
				if diags.HasError() {
					return types.ObjectNull(certificateSyncGcpCertificateManagerDestinationConfigAttrTypes), diags
				}
				var bindingDiags diag.Diagnostics
				binding, bindingDiags = types.ObjectValue(certificateSyncGcpCertificateManagerCertificateMapBindingAttrTypes, map[string]attr.Value{
					"certificate_map": certificateMap,
					"hostname":        optionalStringFromMap(rawBinding, "hostname"),
				})
				diags.Append(bindingDiags...)
				if diags.HasError() {
					return types.ObjectNull(certificateSyncGcpCertificateManagerDestinationConfigAttrTypes), diags
				}
			}

			obj, objDiags := types.ObjectValue(certificateSyncGcpCertificateManagerDestinationConfigAttrTypes, map[string]attr.Value{
				"gcp_project_id":          gcpProjectID,
				"location":                location,
				"scope":                   scope,
				"certificate_map_binding": binding,
			})
			diags.Append(objDiags...)
			return obj, diags
		},
	}
}
