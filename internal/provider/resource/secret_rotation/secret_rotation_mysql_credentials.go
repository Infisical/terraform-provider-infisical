package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SecretRotationMySqlCredentialsParametersModel struct {
	Username1 types.String `tfsdk:"username1"`
	Username2 types.String `tfsdk:"username2"`
}

type SecretRotationMySqlCredentialsSecretsMappingModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func NewSecretRotationMySqlCredentialsResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderMySql,
		SecretRotationName: "MySQL Credentials",
		ResourceTypeName:   "_secret_rotation_mysql_credentials",
		AppConnection:      infisical.AppConnectionAppAWS,
		ParametersAttributes: map[string]schema.Attribute{
			"username1": schema.StringAttribute{
				Required:    true,
				Description: "The username of the first login to rotate passwords for. This user must already exists in your database.",
			},
			"username2": schema.StringAttribute{
				Required:    true,
				Description: "The username of the second login to rotate passwords for. This user must already exists in your database.",
			},
		},
		SecretsMappingAttributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the active username will be mapped to.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the generated password will be mapped to.",
			},
		},

		ReadParametersFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			// TODO(andrey): Finish
			return nil, nil
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
			// TODO(andrey): Finish
			return types.Object{}, nil
		},

		ReadSecretsMappingFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			// TODO(andrey): Finish
			return nil, nil
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
			// TODO(andrey): Finish
			return types.Object{}, nil
		},
	}
}
