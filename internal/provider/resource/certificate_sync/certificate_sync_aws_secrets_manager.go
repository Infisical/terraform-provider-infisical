package resource

import (
	"context"
	"regexp"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	awsSecretsManagerDefaultCertificateField      = "certificate"
	awsSecretsManagerDefaultPrivateKeyField       = "private_key"
	awsSecretsManagerDefaultCertificateChainField = "certificate_chain"
	awsSecretsManagerDefaultCaCertificateField    = "ca_certificate"
)

type CertificateSyncAwsSecretsManagerDestinationConfigModel struct {
	Region types.String `tfsdk:"aws_region"`
	KeyID  types.String `tfsdk:"kms_key_id"`
}

type CertificateSyncAwsSecretsManagerFieldMappingsModel struct {
	Certificate      types.String `tfsdk:"certificate"`
	PrivateKey       types.String `tfsdk:"private_key"`
	CertificateChain types.String `tfsdk:"certificate_chain"`
	CaCertificate    types.String `tfsdk:"ca_certificate"`
}

type CertificateSyncAwsSecretsManagerSyncOptionsModel struct {
	CertificateNameSchema      customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates      types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa              types.Bool                     `tfsdk:"include_root_ca"`
	PreserveSecretOnRenewal    types.Bool                     `tfsdk:"preserve_secret_on_renewal"`
	UpdateExistingCertificates types.Bool                     `tfsdk:"update_existing_certificates"`
	FieldMappings              types.Object                   `tfsdk:"field_mappings"`
}

var certificateSyncAwsSecretsManagerDestinationConfigAttrTypes = map[string]attr.Type{
	"aws_region": types.StringType,
	"kms_key_id": types.StringType,
}

var certificateSyncAwsSecretsManagerFieldMappingsAttrTypes = map[string]attr.Type{
	"certificate":       types.StringType,
	"private_key":       types.StringType,
	"certificate_chain": types.StringType,
	"ca_certificate":    types.StringType,
}

var certificateSyncAwsSecretsManagerSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema":      customtypes.TrimmedStringType{},
	"can_remove_certificates":      types.BoolType,
	"include_root_ca":              types.BoolType,
	"preserve_secret_on_renewal":   types.BoolType,
	"update_existing_certificates": types.BoolType,
	"field_mappings":               types.ObjectType{AttrTypes: certificateSyncAwsSecretsManagerFieldMappingsAttrTypes},
}

func awsSecretsManagerFieldMappingFromMap(m map[string]interface{}, key string, def string) types.String {
	if value, ok := m[key].(string); ok && value != "" {
		return types.StringValue(value)
	}
	return types.StringValue(def)
}

func NewCertificateSyncAwsSecretsManagerResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:                 infisical.CertificateSyncAppAWSSecretsManager,
		SyncName:            "AWS Secrets Manager",
		ResourceTypeName:    "_certificate_sync_aws_secrets_manager",
		CertificateNameRule: &certificateNameRule{pattern: regexp.MustCompile(`^[\w-]+$`), minLength: 1, maxLength: 512, requireIdentifier: true, requirement: "1-512 letters, digits, hyphens or underscores"},
		AppConnection:       infisical.AppConnectionAppAWS,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"aws_region": schema.StringAttribute{
				Required:    true,
				Description: "The AWS region to sync certificates to (e.g. us-east-1).",
			},
			"kms_key_id": schema.StringAttribute{
				Validators:  []validator.String{notBlank()},
				Optional:    true,
				Description: "The ID or ARN of the KMS key used to encrypt the secrets. Leave unset to use the AWS managed key.",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for synced secrets. Must include the {{certificateId}} or {{shortCertificateId}} placeholder, and compiled names may contain only alphanumeric characters, underscores and hyphens (1-512 characters). Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove secrets from AWS Secrets Manager when their certificates are no longer managed in Infisical. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
			"include_root_ca": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to include the root CA certificate in the synced certificate chain. Defaults to false.",
				Default:     booldefault.StaticBool(false),
			},
			"preserve_secret_on_renewal": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to update the existing secret when a certificate is renewed instead of creating a new one. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
			"update_existing_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to overwrite existing secrets in AWS Secrets Manager that match a synced certificate's name. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
			"field_mappings": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The JSON field names used for each part of the certificate inside the secret.",
				Default: objectdefault.StaticValue(types.ObjectValueMust(certificateSyncAwsSecretsManagerFieldMappingsAttrTypes, map[string]attr.Value{
					"certificate":       types.StringValue(awsSecretsManagerDefaultCertificateField),
					"private_key":       types.StringValue(awsSecretsManagerDefaultPrivateKeyField),
					"certificate_chain": types.StringValue(awsSecretsManagerDefaultCertificateChainField),
					"ca_certificate":    types.StringValue(awsSecretsManagerDefaultCaCertificateField),
				})),
				Attributes: map[string]schema.Attribute{
					"certificate": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The field name for the certificate. Defaults to certificate.",
						Default:     stringdefault.StaticString(awsSecretsManagerDefaultCertificateField),
					},
					"private_key": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The field name for the private key. Defaults to private_key.",
						Default:     stringdefault.StaticString(awsSecretsManagerDefaultPrivateKeyField),
					},
					"certificate_chain": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The field name for the certificate chain. Defaults to certificate_chain.",
						Default:     stringdefault.StaticString(awsSecretsManagerDefaultCertificateChainField),
					},
					"ca_certificate": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The field name for the CA certificate. Defaults to ca_certificate.",
						Default:     stringdefault.StaticString(awsSecretsManagerDefaultCaCertificateField),
					},
				},
			},
		},

		ValidateConfigFunc: func(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
			validateFieldMappingsUnique(ctx, config.SyncOptions, map[string]string{
				"certificate":       awsSecretsManagerDefaultCertificateField,
				"private_key":       awsSecretsManagerDefaultPrivateKeyField,
				"certificate_chain": awsSecretsManagerDefaultCertificateChainField,
				"ca_certificate":    awsSecretsManagerDefaultCaCertificateField,
			}, diags)
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncAwsSecretsManagerSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			payload := map[string]interface{}{
				"certificateNameSchema":      syncOptions.CertificateNameSchema.ValueString(),
				"canRemoveCertificates":      syncOptions.CanRemoveCertificates.ValueBool(),
				"includeRootCa":              syncOptions.IncludeRootCa.ValueBool(),
				"preserveSecretOnRenewal":    syncOptions.PreserveSecretOnRenewal.ValueBool(),
				"updateExistingCertificates": syncOptions.UpdateExistingCertificates.ValueBool(),
				"canImportCertificates":      false,
			}

			if !syncOptions.FieldMappings.IsNull() && !syncOptions.FieldMappings.IsUnknown() {
				var fieldMappings CertificateSyncAwsSecretsManagerFieldMappingsModel
				diags.Append(syncOptions.FieldMappings.As(ctx, &fieldMappings, basetypes.ObjectAsOptions{})...)
				if diags.HasError() {
					return nil, diags
				}
				payload["fieldMappings"] = map[string]interface{}{
					"certificate":      fieldMappings.Certificate.ValueString(),
					"privateKey":       fieldMappings.PrivateKey.ValueString(),
					"certificateChain": fieldMappings.CertificateChain.ValueString(),
					"caCertificate":    fieldMappings.CaCertificate.ValueString(),
				}
			}

			return payload, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncAwsSecretsManagerSyncOptionsAttrTypes), diags
			}

			rawFieldMappings, _ := certificateSync.SyncOptions["fieldMappings"].(map[string]interface{})
			if rawFieldMappings == nil {
				rawFieldMappings = map[string]interface{}{}
			}

			fieldMappings, objDiags := types.ObjectValue(certificateSyncAwsSecretsManagerFieldMappingsAttrTypes, map[string]attr.Value{
				"certificate":       awsSecretsManagerFieldMappingFromMap(rawFieldMappings, "certificate", awsSecretsManagerDefaultCertificateField),
				"private_key":       awsSecretsManagerFieldMappingFromMap(rawFieldMappings, "privateKey", awsSecretsManagerDefaultPrivateKeyField),
				"certificate_chain": awsSecretsManagerFieldMappingFromMap(rawFieldMappings, "certificateChain", awsSecretsManagerDefaultCertificateChainField),
				"ca_certificate":    awsSecretsManagerFieldMappingFromMap(rawFieldMappings, "caCertificate", awsSecretsManagerDefaultCaCertificateField),
			})
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncAwsSecretsManagerSyncOptionsAttrTypes), diags
			}

			obj, objDiags := types.ObjectValue(certificateSyncAwsSecretsManagerSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema":      certificateNameSchema,
				"can_remove_certificates":      boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
				"include_root_ca":              boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
				"preserve_secret_on_renewal":   boolFromMap(certificateSync.SyncOptions, "preserveSecretOnRenewal", true),
				"update_existing_certificates": boolFromMap(certificateSync.SyncOptions, "updateExistingCertificates", true),
				"field_mappings":               fieldMappings,
			})
			diags.Append(objDiags...)
			return obj, diags
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncAwsSecretsManagerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			payload := map[string]interface{}{
				"region": destinationConfig.Region.ValueString(),
			}
			setOptionalString(payload, "keyId", destinationConfig.KeyID)
			return payload, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			region := stringFromMap(certificateSync.DestinationConfig, "region", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncAwsSecretsManagerDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncAwsSecretsManagerDestinationConfigAttrTypes, map[string]attr.Value{
				"aws_region": region,
				"kms_key_id": optionalStringFromMap(certificateSync.DestinationConfig, "keyId"),
			})
		},
	}
}
