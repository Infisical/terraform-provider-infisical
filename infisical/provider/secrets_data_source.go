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

		plainTextSecrets, err = expandPlainTextSecrets(plainTextSecrets, d.client)
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
	} else if d.client.Config.AuthStrategy == infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		secrets, err := d.client.GetRawSecrets(data.FolderPath.ValueString(), data.EnvSlug.ValueString(), data.WorkspaceId.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Something went wrong while fetching secrets",
				"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
					"Infisical Client Error: "+err.Error(),
			)
		}

		secrets, err = expandRawSecrets(secrets, data.WorkspaceId.ValueString(), d.client)
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

/*
	TODO: cleanup and merge repetitive code in functions below
	right now 4 total functions with largely same logic because of []infisical.SingleEnvironmentVariable
	and secrets []infisical.RawV3Secret for both auth strategies
*/

func expandPlainTextSecrets(secrets []infisical.SingleEnvironmentVariable, client *infisical.Client) ([]infisical.SingleEnvironmentVariable, error) {
	expandedSecs := make(map[string]string)
	interpolatedSecs := make(map[string]string)

	for _, secret := range secrets {
		refs := secRefRegex.FindAllStringSubmatch(secret.Value, -1)
		if refs == nil {
			expandedSecs[secret.Key] = secret.Value
		} else {
			interpolatedSecs[secret.Key] = secret.Value
		}
	}

	for i, secret := range secrets {
		data, err := recursivelyExpandSecretViaServiceToken(secret.Value, expandedSecs, interpolatedSecs, client)
		if err != nil {
			return nil, errors.New("failed to expand secrets: " + err.Error())
		}
		secrets[i].Value = data
	}

	return secrets, nil
}

func recursivelyExpandSecretViaServiceToken(value string, expandedSecs map[string]string, interpolatedSecs map[string]string, client *infisical.Client) (string, error) {
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
					data, err := recursivelyExpandSecretViaServiceToken(interpolatedSecs[key], expandedSecs, interpolatedSecs, client)
					if err != nil {
						return "", errors.New("failed to expand secret " + key)
					}
					value = strings.Replace(value, repl, data, -1)
				}
			} else if len(query) > 1 {
				env, path, key := query[0], "/", query[len(query)-1]
				if len(query) >= 2 {
					path += strings.Join(query[1:len(query)-1], "/")
				}

				// TODO: add cache to avoid redundant api calls

				relevantSecrets, _, err := client.GetPlainTextSecretsViaServiceToken(path, env)

				if err != nil {
					return "", errors.New("failed to retrieve secret " + " " + path + " " + env)
				}

				relevantSecretsByName := make(map[string]infisical.SingleEnvironmentVariable, len(relevantSecrets))

				for _, secret := range relevantSecrets {
					relevantSecretsByName[secret.Key] = secret
				}

				data, err := recursivelyExpandSecretViaServiceToken(relevantSecretsByName[key].Value, expandedSecs, interpolatedSecs, client)

				if err != nil {
					return "", errors.New("failed to expand secret" + key)
				}
				value = strings.Replace(value, repl, data, -1)
			}
		}
		return value, nil
	}
}

func expandRawSecrets(secrets []infisical.RawV3Secret, workspaceId string, client *infisical.Client) ([]infisical.RawV3Secret, error) {
	expandedSecs := make(map[string]string)
	interpolatedSecs := make(map[string]string)

	for _, secret := range secrets {
		refs := secRefRegex.FindAllStringSubmatch(secret.SecretValue, -1)
		if refs == nil {
			expandedSecs[secret.SecretKey] = secret.SecretValue
		} else {
			interpolatedSecs[secret.SecretKey] = secret.SecretValue
		}
	}

	for i, secret := range secrets {
		data, err := recursivelyExpandRawSecret(secret.SecretValue, expandedSecs, interpolatedSecs, workspaceId, client)
		if err != nil {
			return nil, errors.New("failed to expand secret: " + err.Error())
		}
		secrets[i].SecretValue = data
	}

	return secrets, nil
}

func recursivelyExpandRawSecret(value string, expandedSecs map[string]string, interpolatedSecs map[string]string, workspaceId string, client *infisical.Client) (string, error) {
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
					data, err := recursivelyExpandRawSecret(interpolatedSecs[key], expandedSecs, interpolatedSecs, workspaceId, client)
					if err != nil {
						return "", errors.New("failed to expand secret " + key)
					}
					value = strings.Replace(value, repl, data, -1)
				}
			} else if len(query) > 1 {
				env, path, key := query[0], "/", query[len(query)-1]
				if len(query) >= 2 {
					path += strings.Join(query[1:len(query)-1], "/")
				}

				relevantSecrets, err := client.GetRawSecrets(path, env, workspaceId)

				if err != nil {
					return "", errors.New("failed to retrieve secret " + " " + path + " " + env)
				}

				relevantSecretsByName := make(map[string]infisical.RawV3Secret, len(relevantSecrets))

				for _, secret := range relevantSecrets {
					relevantSecretsByName[secret.SecretKey] = secret
				}

				data, err := recursivelyExpandRawSecret(relevantSecretsByName[key].SecretValue, expandedSecs, interpolatedSecs, workspaceId, client)

				if err != nil {
					return "", errors.New("failed to expand secret" + key)
				}
				value = strings.Replace(value, repl, data, -1)
			}
		}
		return value, nil
	}
}
