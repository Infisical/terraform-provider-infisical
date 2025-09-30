package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// SecretSyncAwsParameterStoreDestinationConfigModel describes the data source data model.
type SecretSyncAwsParameterStoreDestinationConfigModel struct {
	Region types.String `tfsdk:"aws_region"`
	Path   types.String `tfsdk:"path"`
}

type AwsParameterStoreTagsModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type SecretSyncAwsParameterStoreSyncOptionsModel struct {
	InitialSyncBehavior      types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion    types.Bool   `tfsdk:"disable_secret_deletion"`
	KeyID                    types.String `tfsdk:"aws_kms_key_id"`
	SyncSecretMetadataAsTags types.Bool   `tfsdk:"sync_secret_metadata_as_tags"`
	KeySchema                types.String `tfsdk:"key_schema"`
	Tags                     types.Set    `tfsdk:"tags"`
}

func NewSecretSyncAwsParameterStoreResource() resource.Resource {
	return &SecretSyncBaseResource{
		CrossplaneCompatible: false,
		App:                  infisical.SecretSyncAppAWSParameterStore,
		SyncName:             "AWS Parameter Store",
		ResourceTypeName:     "_secret_sync_aws_parameter_store",
		AppConnection:        infisical.AppConnectionAppAWS,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"aws_region": schema.StringAttribute{
				Required:    true,
				Description: "The AWS region of your AWS Parameter Store",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "The path in the AWS Parameter Store where the secrets will be stored, Example: /example/path/",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"initial_sync_behavior": schema.StringAttribute{
				Required:    true,
				Description: "Specify how Infisical should resolve the initial sync to the destination. Supported options: overwrite-destination, import-prioritize-source, import-prioritize-destination",
			},
			"disable_secret_deletion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When set to true, Infisical will not remove secrets from AWS Parameter Store. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"aws_kms_key_id": schema.StringAttribute{
				Optional:    true,
				Description: "The AWS KMS key ID to use for encryption",
			},
			"sync_secret_metadata_as_tags": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to sync the secret metadata as tags",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the AWS Parameter Store destination.",
			},
			"tags": schema.SetNestedAttribute{
				Optional:    true,
				Description: "The tags to sync to the secret",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:    true,
							Description: "The key of the tag",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The value of the tag",
						},
					},
				},
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncAwsParameterStoreSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["syncSecretMetadataAsTags"] = syncOptions.SyncSecretMetadataAsTags.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()

			if syncOptions.KeyID.ValueString() != "" {
				syncOptionsMap["keyId"] = syncOptions.KeyID.ValueString()
			}

			if !syncOptions.Tags.IsNull() {
				var tagModels []AwsParameterStoreTagsModel

				diags := syncOptions.Tags.ElementsAs(ctx, &tagModels, false)
				if diags.HasError() {
					return nil, diags
				}

				tagsArray := make([]map[string]interface{}, 0, len(tagModels))
				for _, tag := range tagModels {
					tagsArray = append(tagsArray, map[string]interface{}{
						"key":   tag.Key.ValueString(),
						"value": tag.Value.ValueString(),
					})
				}

				syncOptionsMap["tags"] = tagsArray
			}

			return syncOptionsMap, nil
		},

		ReadSyncOptionsFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			syncOptionsMap := make(map[string]attr.Value)

			initialSyncBehavior, ok := secretSync.SyncOptions["initialSyncBehavior"].(string)
			if !ok {
				initialSyncBehavior = ""
			}

			disableSecretDeletion, ok := secretSync.SyncOptions["disableSecretDeletion"].(bool)
			if !ok {
				disableSecretDeletion = false
			}

			syncOptionsMap["initial_sync_behavior"] = types.StringValue(initialSyncBehavior)
			syncOptionsMap["disable_secret_deletion"] = types.BoolValue(disableSecretDeletion)

			if secretSync.SyncOptions["keyId"] != nil {

				keyId := ""
				if key, ok := secretSync.SyncOptions["keyId"].(string); ok {
					keyId = key
				}
				syncOptionsMap["aws_kms_key_id"] = types.StringValue(keyId)
			} else {
				syncOptionsMap["aws_kms_key_id"] = types.StringNull() // Add a null value for missing attributes
			}

			if secretSync.SyncOptions["syncSecretMetadataAsTags"] != nil {

				syncSecretMetadataAsTags := false
				syncMetadataAsTags, ok := secretSync.SyncOptions["syncSecretMetadataAsTags"].(bool)
				if ok {
					syncSecretMetadataAsTags = syncMetadataAsTags
				}

				syncOptionsMap["sync_secret_metadata_as_tags"] = types.BoolValue(syncSecretMetadataAsTags)
			} else {
				syncOptionsMap["sync_secret_metadata_as_tags"] = types.BoolNull()
			}

			keySchema, ok := secretSync.SyncOptions["keySchema"].(string)
			if keySchema == "" || !ok {
				syncOptionsMap["key_schema"] = types.StringNull()
			} else {
				syncOptionsMap["key_schema"] = types.StringValue(keySchema)
			}

			if secretSync.SyncOptions["tags"] != nil {
				rawTags, ok := secretSync.SyncOptions["tags"].([]interface{})
				if !ok {
					rawTags = []interface{}{}
				}

				tagsObjects := make([]attr.Value, 0, len(rawTags))
				for _, rawTag := range rawTags {
					tag, ok := rawTag.(map[string]interface{})
					if !ok {
						tag = map[string]interface{}{}
					}

					key, ok := tag["key"].(string)
					if !ok {
						key = ""
					}

					value, ok := tag["value"].(string)
					if !ok {
						value = ""
					}

					attrs := map[string]attr.Value{
						"key":   types.StringValue(key),
						"value": types.StringValue(value),
					}

					obj, diags := types.ObjectValue(
						map[string]attr.Type{
							"key":   types.StringType,
							"value": types.StringType,
						},
						attrs,
					)
					if diags.HasError() {
						return types.ObjectNull(map[string]attr.Type{}), diags
					}
					tagsObjects = append(tagsObjects, obj)
				}

				setVal, diags := types.SetValue(
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"key":   types.StringType,
							"value": types.StringType,
						},
					},
					tagsObjects,
				)
				if diags.HasError() {
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				syncOptionsMap["tags"] = setVal
			} else {
				syncOptionsMap["tags"] = types.SetNull(types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"key":   types.StringType,
						"value": types.StringType,
					},
				})
			}

			return types.ObjectValue(map[string]attr.Type{
				"initial_sync_behavior":        types.StringType,
				"disable_secret_deletion":      types.BoolType,
				"aws_kms_key_id":               types.StringType,
				"sync_secret_metadata_as_tags": types.BoolType,
				"key_schema":                   types.StringType,
				"tags": types.SetType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"key":   types.StringType,
							"value": types.StringType,
						},
					},
				},
			}, syncOptionsMap)
		},

		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncAwsParameterStoreSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["syncSecretMetadataAsTags"] = syncOptions.SyncSecretMetadataAsTags.ValueBool()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()

			if syncOptions.KeyID.ValueString() != "" {
				syncOptionsMap["keyId"] = syncOptions.KeyID.ValueString()
			}

			if !syncOptions.Tags.IsNull() {
				// Create a slice of TagsModel to hold our tags
				var tagModels []AwsParameterStoreTagsModel

				// Get the tags from the set
				diags := syncOptions.Tags.ElementsAs(ctx, &tagModels, false)
				if diags.HasError() {
					return nil, diags
				}

				// Convert to the format expected by the API
				tagsArray := make([]map[string]interface{}, 0, len(tagModels))
				for _, tag := range tagModels {
					tagsArray = append(tagsArray, map[string]interface{}{
						"key":   tag.Key.ValueString(),
						"value": tag.Value.ValueString(),
					})
				}

				syncOptionsMap["tags"] = tagsArray
			}

			return syncOptionsMap, nil
		},

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var awsCfg SecretSyncAwsParameterStoreDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &awsCfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			destinationConfig["region"] = awsCfg.Region.ValueString()
			destinationConfig["path"] = awsCfg.Path.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var awsCfg SecretSyncAwsParameterStoreDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &awsCfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			destinationConfig["region"] = awsCfg.Region.ValueString()
			destinationConfig["path"] = awsCfg.Path.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			regionVal, ok := secretSync.DestinationConfig["region"].(string)
			if !ok {
				diags.AddError(
					"Invalid region type",
					"Expected 'region' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			pathVal, ok := secretSync.DestinationConfig["path"].(string)
			if !ok {
				diags.AddError(
					"Invalid path type",
					"Expected 'path' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"aws_region": types.StringValue(regionVal),
				"path":       types.StringValue(pathVal),
			}

			return types.ObjectValue(map[string]attr.Type{
				"aws_region": types.StringType,
				"path":       types.StringType,
			}, destinationConfig)
		},
	}
}
