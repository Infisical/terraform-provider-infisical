// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	infisical "terraform-provider-infisical/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SecretsDataSource{}

func NewSecretDataSource() datasource.DataSource {
	return &SecretsDataSource{}
}

// SecretDataSource defines the data source implementation.
type SecretsDataSource struct {
	client *infisical.Client
}

// ExampleDataSourceModel describes the data source data model.
type SecretDataSourceModel struct {
	ID          types.String                      `tfsdk:"id"`
	FolderPath  types.String                      `tfsdk:"folder_path"`
	WorkspaceId types.String                      `tfsdk:"workspace_id"`
	EnvSlug     types.String                      `tfsdk:"env_slug"`
	Secrets     map[string]InfisicalSecretDetails `tfsdk:"secrets"`
}

type InfisicalSecretDetails struct {
	Value      types.String `tfsdk:"value"`
	Comment    types.String `tfsdk:"comment"`
	SecretType types.String `tfsdk:"secret_type"`
}

func (d *SecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secrets"
}

func (d *SecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get secrets from Infisical",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"folder_path": schema.StringAttribute{
				Description: "The path to the folder from where secrets should be fetched from",
				Required:    true,
				Computed:    false,
			},
			"env_slug": schema.StringAttribute{
				Description: "The environment from where secrets should be fetched from",
				Required:    true,
				Computed:    false,
			},

			"workspace_id": schema.StringAttribute{
				Description: "The Infisical project ID (Required for Machine Identity auth)",
				Optional:    true,
				Computed:    true,
			},

			"secrets": schema.MapNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Computed:    true,
							Description: "The secret value",
						},
						"comment": schema.StringAttribute{
							Computed:    true,
							Description: "The secret comment",
						},
						"secret_type": schema.StringAttribute{
							Computed:    true,
							Description: "The secret type (shared or personal)",
						},
					},
				},
			},
		},
	}
}

func (d *SecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *SecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SecretDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue("example-id") // for testing purposes as test client requires

	if d.client.Config.AuthStrategy == infisical.AuthStrategy.SERVICE_TOKEN {

		plainTextSecrets, _, err := d.client.GetPlainTextSecretsViaServiceToken(data.FolderPath.ValueString(), data.EnvSlug.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Something went wrong while fetching secrets",
				"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
					"Infisical Client Error: "+err.Error(),
			)
		}

		if data.FolderPath.IsNull() {
			data.FolderPath = types.StringValue("/")
		}

		data.Secrets = make(map[string]InfisicalSecretDetails)

		for _, secret := range plainTextSecrets {
			data.Secrets[secret.Key] = InfisicalSecretDetails{Value: types.StringValue(secret.Value), Comment: types.StringValue(secret.Comment), SecretType: types.StringValue(secret.Type)}
		}

		data.Secrets = expandSecrets(data.Secrets, func(env string, path string, key string, cache map[string]string) (map[string]string, error) {
			relevantSecrets, _, err := d.client.GetPlainTextSecretsViaServiceToken(path, env)

			if err != nil {
				return cache, err
			}

			cacheKey := strings.Join([]string{env, path, key}, ".")

			for _, secret := range relevantSecrets {
				cache[cacheKey] = secret.Value
			}

			return cache, nil
		})

	} else if d.client.Config.AuthStrategy == infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		secrets, err := d.client.GetRawSecrets(data.FolderPath.ValueString(), data.EnvSlug.ValueString(), data.WorkspaceId.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Something went wrong while fetching secrets",
				"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
					"Infisical Client Error: "+err.Error(),
			)
		}

		if data.FolderPath.IsNull() {
			data.FolderPath = types.StringValue("/")
		}

		data.Secrets = make(map[string]InfisicalSecretDetails)

		for _, secret := range secrets {
			data.Secrets[secret.SecretKey] = InfisicalSecretDetails{Value: types.StringValue(secret.SecretValue), Comment: types.StringValue(secret.SecretComment), SecretType: types.StringValue(secret.Type)}
		}

		data.Secrets = expandSecrets(data.Secrets, func(env string, path string, key string, cache map[string]string) (map[string]string, error) {
			relevantSecrets, err := d.client.GetRawSecrets(path, env, data.WorkspaceId.ValueString())

			if err != nil {
				return cache, err
			}

			cacheKey := strings.Join([]string{env, path, key}, ".")

			for _, secret := range relevantSecrets {
				cache[cacheKey] = secret.SecretValue
			}

			return cache, nil
		})
	} else {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching secrets",
			"Unable to determine authentication strategy. Please report this issue to the Infisical engineers at infisical.com/slack\n\n",
		)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

var secRefRegex = regexp.MustCompile(`\${([^\}]*)}`)

func expandSecrets(secrets map[string]InfisicalSecretDetails, crossEnvFetch func(env string, path string, key string, cache map[string]string) (map[string]string, error)) map[string]InfisicalSecretDetails {
	expandedSecs := make(map[string]string)
	interpolatedSecs := make(map[string]string)
	crossEnvCache := make(map[string]string)

	for key, value := range secrets {
		refs := secRefRegex.FindAllStringSubmatch(value.Value.ValueString(), -1)
		if refs == nil {
			expandedSecs[key] = value.Value.ValueString()
		} else {
			interpolatedSecs[key] = value.Value.ValueString()
		}
	}

	for key, value := range interpolatedSecs {
		data, err := recursivelyExpandSecret(value, expandedSecs, interpolatedSecs, crossEnvCache, crossEnvFetch)

		if err != nil {
			return nil
		}

		newSecret := secrets[key]
		newSecret.Value = types.StringValue(data)
		secrets[key] = newSecret
	}

	return secrets
}

func recursivelyExpandSecret(value string, expandedSecs map[string]string, interpolatedSecs map[string]string, crossEnvCache map[string]string, crossEnvFetch func(env string, path string, key string, cache map[string]string) (map[string]string, error)) (string, error) {
	refs := secRefRegex.FindAllStringSubmatch(value, -1)
	if refs == nil {
		return value, nil
	} else {
		for _, ref := range refs {
			repl, key := ref[0], ref[1]
			query := strings.Split(key, ".")
			if len(query) == 1 {
				if data, ok := expandedSecs[key]; ok {
					value = strings.Replace(value, repl, data, -1)
				} else {
					data, err := recursivelyExpandSecret(interpolatedSecs[key], expandedSecs, interpolatedSecs, crossEnvCache, crossEnvFetch)
					if err != nil {
						return "", errors.New("failed to expand secret: " + key)
					}
					value = strings.Replace(value, repl, data, -1)
				}
			} else if len(query) > 1 {
				env, path, key := query[0], "/", query[len(query)-1]
				if len(query) >= 2 {
					path += strings.Join(query[1:len(query)-1], "/")
				}

				cacheKey := strings.Join([]string{env, path, key}, ".")

				if crossEnvSec, ok := crossEnvCache[cacheKey]; ok {
					data, err := recursivelyExpandSecret(crossEnvSec, expandedSecs, interpolatedSecs, crossEnvCache, crossEnvFetch)

					if err != nil {
						return "", errors.New("failed to expand secret: " + cacheKey)
					}

					value = strings.Replace(value, repl, data, -1)
				} else {
					crossEnvCache, fetchErr := crossEnvFetch(env, path, key, crossEnvCache)

					if fetchErr != nil {
						return "", errors.New("failed to fetch cross env secrets: " + fetchErr.Error())
					}

					data, err := recursivelyExpandSecret(crossEnvCache[cacheKey], expandedSecs, interpolatedSecs, crossEnvCache, crossEnvFetch)

					if err != nil {
						return "", errors.New("failed to expand secret: " + cacheKey)
					}

					value = strings.Replace(value, repl, data, -1)
				}
			}
		}
		return value, nil
	}
}
