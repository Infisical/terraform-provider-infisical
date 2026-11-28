package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AppConnectionAzureBlobStorageCredentialsModel struct {
	AccountName  types.String `tfsdk:"account_name"`
	TenantId     types.String `tfsdk:"tenant_id"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	AccountKey   types.String `tfsdk:"account_key"`
}

const AzureBlobStorageAppConnectionClientSecretMethod = "client-secret"
const AzureBlobStorageAppConnectionAccessKeyMethod = "access-key"

var azureBlobStorageCredentialsAttrTypes = map[string]attr.Type{
	"account_name":  types.StringType,
	"tenant_id":     types.StringType,
	"client_id":     types.StringType,
	"client_secret": types.StringType,
	"account_key":   types.StringType,
}

func buildAzureBlobStorageCredentials(method string, credentials AppConnectionAzureBlobStorageCredentialsModel, action string) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	required := map[string]types.String{"account_name": credentials.AccountName}
	if method == AzureBlobStorageAppConnectionClientSecretMethod {
		required["tenant_id"] = credentials.TenantId
		required["client_id"] = credentials.ClientId
		required["client_secret"] = credentials.ClientSecret
	} else {
		required["account_key"] = credentials.AccountKey
	}

	for name, value := range required {
		if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
			diags.AddError(
				"Unable to "+action+" Azure Blob Storage app connection",
				"The "+name+" field must be set for the "+method+" method",
			)
		}
	}
	if diags.HasError() {
		return nil, diags
	}

	if method == AzureBlobStorageAppConnectionClientSecretMethod {
		return map[string]any{
			"accountName":  credentials.AccountName.ValueString(),
			"tenantId":     credentials.TenantId.ValueString(),
			"clientId":     credentials.ClientId.ValueString(),
			"clientSecret": credentials.ClientSecret.ValueString(),
		}, diags
	}
	return map[string]any{
		"accountName": credentials.AccountName.ValueString(),
		"accountKey":  credentials.AccountKey.ValueString(),
	}, diags
}

func azureBlobStoragePlanOrState(plan types.String, state types.String) types.String {
	if plan.IsUnknown() {
		return state
	}
	return plan
}

func NewAppConnectionAzureBlobStorageResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppAzureBlobStorage,
		AppConnectionName: "Azure Blob Storage",
		ResourceTypeName:  "_app_connection_azure_blob_storage",
		AllowedMethods:    []string{AzureBlobStorageAppConnectionClientSecretMethod, AzureBlobStorageAppConnectionAccessKeyMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"account_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Azure storage account. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-blob-storage",
			},
			"tenant_id": schema.StringAttribute{
				Optional:    true,
				Description: "The Microsoft Entra tenant ID of the app registration. Required for the client-secret method.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "The client ID of the Microsoft Entra app registration. Required for the client-secret method.",
				Sensitive:   true,
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Description: "The client secret of the Microsoft Entra app registration. Required for the client-secret method.",
				Sensitive:   true,
			},
			"account_key": schema.StringAttribute{
				Optional:    true,
				Description: "An access key of the storage account. Required for the access-key method.",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentials AppConnectionAzureBlobStorageCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}
			return buildAzureBlobStorageCredentials(plan.Method.ValueString(), credentials, "create")
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var fromPlan AppConnectionAzureBlobStorageCredentialsModel
			diags := plan.Credentials.As(ctx, &fromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}
			var fromState AppConnectionAzureBlobStorageCredentialsModel
			diags = state.Credentials.As(ctx, &fromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}
			merged := AppConnectionAzureBlobStorageCredentialsModel{
				AccountName:  azureBlobStoragePlanOrState(fromPlan.AccountName, fromState.AccountName),
				TenantId:     azureBlobStoragePlanOrState(fromPlan.TenantId, fromState.TenantId),
				ClientId:     azureBlobStoragePlanOrState(fromPlan.ClientId, fromState.ClientId),
				ClientSecret: azureBlobStoragePlanOrState(fromPlan.ClientSecret, fromState.ClientSecret),
				AccountKey:   azureBlobStoragePlanOrState(fromPlan.AccountKey, fromState.AccountKey),
			}
			return buildAzureBlobStorageCredentials(plan.Method.ValueString(), merged, "update")
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"account_name":  types.StringNull(),
				"tenant_id":     types.StringNull(),
				"client_id":     types.StringNull(),
				"client_secret": types.StringNull(),
				"account_key":   types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(azureBlobStorageCredentialsAttrTypes, credentialsConfig)
			return diags
		},
	}
}
