package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const gcpSecretManagerScopeGlobal = "global"

// SecretSyncGcpResourceModel describes the data source data model.
type SecretSyncGcpSecretManagerDestinationConfigModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	Scope     types.String `tfsdk:"scope"`
}

func NewSecretSyncGcpSecretManagerResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppGCPSecretManager,
		SyncName:         "GCP Secret Manager",
		ResourceTypeName: "_secret_sync_gcp_secret_manager",
		AppConnection:    "GCP",
		DestinationConfigAttributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the GCP project to sync with",
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Description: "The scope of the sync with GCP Secret Manager. Supported options: global",
				Default:     stringdefault.StaticString("global"),
				Computed:    true,
			},
		},
		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var gcpConfig SecretSyncGcpSecretManagerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &gcpConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if gcpConfig.Scope.ValueString() != gcpSecretManagerScopeGlobal {
				diags.AddError(
					"Unable to create GCP secret manager secret sync",
					"Invalid value for scope field. Possible values are: global",
				)
				return nil, diags
			}

			destinationConfig["scope"] = gcpConfig.Scope.ValueString()
			destinationConfig["projectId"] = gcpConfig.ProjectID.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var gcpConfig SecretSyncGcpSecretManagerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &gcpConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if gcpConfig.Scope.ValueString() != gcpSecretManagerScopeGlobal {
				diags.AddError(
					"Unable to update GCP secret manager secret sync",
					"Invalid value for scope field. Possible values are: global",
				)
				return nil, diags
			}

			destinationConfig["scope"] = gcpConfig.Scope.ValueString()
			destinationConfig["projectId"] = gcpConfig.ProjectID.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			scopeVal, ok := secretSync.DestinationConfig["scope"].(string)
			if !ok {
				diags.AddError(
					"Invalid scope type",
					"Expected 'scope' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			projectIdVal, ok := secretSync.DestinationConfig["projectId"].(string)
			if !ok {
				diags.AddError(
					"Invalid projectId type",
					"Expected 'projectId' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"scope":      types.StringValue(scopeVal),
				"project_id": types.StringValue(projectIdVal),
			}

			return types.ObjectValue(map[string]attr.Type{
				"scope":      types.StringType,
				"project_id": types.StringType,
			}, destinationConfig)
		},
	}
}
