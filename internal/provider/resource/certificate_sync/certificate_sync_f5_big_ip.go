package resource

import (
	"context"
	"regexp"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	f5BigIpProfileTypeNone      = "none"
	f5BigIpProfileTypeClientSsl = "client-ssl"
	f5BigIpProfileTypeServerSsl = "server-ssl"
)

type CertificateSyncF5BigIpDestinationConfigModel struct {
	Partition              customtypes.TrimmedStringValue `tfsdk:"partition"`
	ProfileType            types.String                   `tfsdk:"profile_type"`
	ProfileName            customtypes.TrimmedStringValue `tfsdk:"profile_name"`
	CreateProfileIfMissing types.Bool                     `tfsdk:"create_profile_if_missing"`
	ParentProfile          customtypes.TrimmedStringValue `tfsdk:"parent_profile"`
}

type CertificateSyncF5BigIpSyncOptionsModel struct {
	CertificateNameSchema customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa         types.Bool                     `tfsdk:"include_root_ca"`
	PreserveItemOnRenewal types.Bool                     `tfsdk:"preserve_item_on_renewal"`
}

var certificateSyncF5BigIpDestinationConfigAttrTypes = map[string]attr.Type{
	"partition":                 customtypes.TrimmedStringType{},
	"profile_type":              types.StringType,
	"profile_name":              customtypes.TrimmedStringType{},
	"create_profile_if_missing": types.BoolType,
	"parent_profile":            customtypes.TrimmedStringType{},
}

var certificateSyncF5BigIpSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema":  customtypes.TrimmedStringType{},
	"can_remove_certificates":  types.BoolType,
	"include_root_ca":          types.BoolType,
	"preserve_item_on_renewal": types.BoolType,
}

func validateF5BigIpDestinationConfig(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
	if config.DestinationConfig.IsNull() || config.DestinationConfig.IsUnknown() {
		return
	}

	var destinationConfig CertificateSyncF5BigIpDestinationConfigModel
	diags.Append(config.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)...)
	if diags.HasError() {
		return
	}

	root := path.Root(attrDestinationConfig)
	profileType := destinationConfig.ProfileType
	profileName := destinationConfig.ProfileName
	parentProfile := destinationConfig.ParentProfile
	createProfile := destinationConfig.CreateProfileIfMissing
	parentProfileSet := !parentProfile.IsNull() && !parentProfile.IsUnknown()

	if !profileType.IsUnknown() {
		if !profileType.IsNull() && profileType.ValueString() != f5BigIpProfileTypeNone {
			if profileName.IsNull() {
				diags.AddAttributeError(root.AtName("profile_name"), "Missing profile name",
					"profile_name is required when profile_type is client-ssl or server-ssl.")
			}
		} else {
			if !profileName.IsNull() && !profileName.IsUnknown() {
				diags.AddAttributeError(root.AtName("profile_name"), "Profile setting without a profile type",
					"profile_name only applies when profile_type is client-ssl or server-ssl.")
			}
			if !createProfile.IsUnknown() && createProfile.ValueBool() {
				diags.AddAttributeError(root.AtName("create_profile_if_missing"), "Profile setting without a profile type",
					"create_profile_if_missing only applies when profile_type is client-ssl or server-ssl.")
			}
			if parentProfileSet {
				diags.AddAttributeError(root.AtName("parent_profile"), "Profile setting without a profile type",
					"parent_profile only applies when profile_type is client-ssl or server-ssl.")
			}
			return
		}
	}

	if !createProfile.IsUnknown() && !createProfile.ValueBool() && parentProfileSet {
		diags.AddAttributeError(root.AtName("parent_profile"), "Parent profile without profile creation",
			"parent_profile only applies when create_profile_if_missing is true.")
	}
}

func NewCertificateSyncF5BigIpResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:                 infisical.CertificateSyncAppF5BigIp,
		SyncName:            "F5 BIG-IP",
		ResourceTypeName:    "_certificate_sync_f5_big_ip",
		CertificateNameRule: &certificateNameRule{pattern: regexp.MustCompile(`^[a-zA-Z0-9._-]{1,255}$`), requirement: "1-255 letters, digits, periods, hyphens or underscores"},
		AppConnection:       infisical.AppConnectionAppF5BigIp,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"partition": schema.StringAttribute{
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The F5 BIG-IP partition to sync certificates to. Defaults to the `Common` partition when unset.",
				Validators:  []validator.String{notBlank(), stringvalidator.LengthBetween(1, 255)},
			},
			"profile_type": schema.StringAttribute{
				Optional:    true,
				Description: "The type of SSL profile to bind synced certificates to. Supported values: `none`, `client-ssl`, `server-ssl`. Leave unset or set to `none` to sync certificates without binding them to a profile.",
				Validators:  []validator.String{stringvalidator.OneOf(f5BigIpProfileTypeNone, f5BigIpProfileTypeClientSsl, f5BigIpProfileTypeServerSsl)},
			},
			"profile_name": schema.StringAttribute{
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The name of the SSL profile to bind synced certificates to. Required when `profile_type` is `client-ssl` or `server-ssl`, and not allowed otherwise.",
				Validators:  []validator.String{notBlank(), stringvalidator.LengthBetween(1, 255)},
			},
			"create_profile_if_missing": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to create the SSL profile when it does not exist. Only applies when `profile_type` is `client-ssl` or `server-ssl`. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
			"parent_profile": schema.StringAttribute{
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The parent profile a newly created SSL profile inherits from. Only allowed when `create_profile_if_missing` is `true`.",
				Validators:  []validator.String{notBlank(), stringvalidator.LengthBetween(1, 511)},
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for synced certificates, which must compile to 1-255 alphanumeric characters, hyphens, underscores, or periods. Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}. A schema with no placeholder can be linked to only one certificate.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove certificates from F5 BIG-IP when they are no longer managed in Infisical. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"include_root_ca": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to include the root CA certificate in the synced certificate chain. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
			"preserve_item_on_renewal": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether a renewed certificate replaces its existing F5 BIG-IP certificate instead of creating a new one. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
		},

		ValidateConfigFunc: validateF5BigIpDestinationConfig,

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncF5BigIpSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"certificateNameSchema": syncOptions.CertificateNameSchema.ValueString(),
				"canRemoveCertificates": syncOptions.CanRemoveCertificates.ValueBool(),
				"includeRootCa":         syncOptions.IncludeRootCa.ValueBool(),
				"preserveItemOnRenewal": syncOptions.PreserveItemOnRenewal.ValueBool(),
			}, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncF5BigIpSyncOptionsAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncF5BigIpSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema":  certificateNameSchema,
				"can_remove_certificates":  boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
				"include_root_ca":          boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
				"preserve_item_on_renewal": boolFromMap(certificateSync.SyncOptions, "preserveItemOnRenewal", true),
			})
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncF5BigIpDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			request := map[string]interface{}{}
			setOptionalString(request, "partition", destinationConfig.Partition)
			setOptionalString(request, "profileType", destinationConfig.ProfileType)
			setOptionalString(request, "profileName", destinationConfig.ProfileName)
			setOptionalBool(request, "createProfileIfMissing", destinationConfig.CreateProfileIfMissing)
			setOptionalString(request, "parentProfile", destinationConfig.ParentProfile)
			return request, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			return types.ObjectValue(certificateSyncF5BigIpDestinationConfigAttrTypes, map[string]attr.Value{
				"partition":                 optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "partition"),
				"profile_type":              optionalStringFromMap(certificateSync.DestinationConfig, "profileType"),
				"profile_name":              optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "profileName"),
				"create_profile_if_missing": boolFromMap(certificateSync.DestinationConfig, "createProfileIfMissing", false),
				"parent_profile":            optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "parentProfile"),
			})
		},
	}
}
