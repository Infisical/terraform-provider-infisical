package datasource

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SecretPropertiesDataSource{}

func NewSecretPropertiesDataSource() datasource.DataSource {
	return &SecretPropertiesDataSource{}
}

type SecretPropertiesDataSource struct {
	client *infisical.Client
}

const defaultSecretType = "shared"

type SecretPropertiesDataSourceModel struct {
	Name            types.String `tfsdk:"name"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	ProjectID       types.String `tfsdk:"project_id"`
	FolderPath      types.String `tfsdk:"folder_path"`
	SecretType      types.String `tfsdk:"secret_type"`
	SecretVersion   types.Int64  `tfsdk:"secret_version"`
	SecretMetadata  types.List   `tfsdk:"secret_metadata"`
	Tags            types.List   `tfsdk:"tags"`
}

func (d *SecretPropertiesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_properties"
}

func (d *SecretPropertiesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve properties for a single Infisical secret without exposing the secret value.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the secret to retrieve properties for.",
				Required:    true,
			},
			"environment_slug": schema.StringAttribute{
				Description: "The environment slug where the secret resides.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Infisical project ID.",
				Required:    true,
			},
			"folder_path": schema.StringAttribute{
				Description: "The path to the folder where the secret is located.",
				Required:    true,
			},
			"secret_version": schema.Int64Attribute{
				Description: "The version number of the secret.",
				Computed:    true,
			},
			"secret_type": schema.StringAttribute{
				Description: "The type of the secret (shared or personal). Defaults to " + defaultSecretType + ".",
				Optional:    true,
				Computed:    true,
			},
			"secret_metadata": schema.ListAttribute{
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"key":          types.StringType,
						"value":        types.StringType,
						"is_encrypted": types.BoolType,
					},
				},
				Description: "Metadata associated with the secret as a list of key-value entries.",
				Computed:    true,
			},
			"tags": schema.ListAttribute{
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":    types.StringType,
						"slug":  types.StringType,
						"name":  types.StringType,
						"color": types.StringType,
					},
				},
				Description: "Tags associated with the secret.",
				Computed:    true,
			},
		},
	}
}

func (d *SecretPropertiesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *SecretPropertiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch secret properties",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data SecretPropertiesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretType := data.SecretType.ValueString()
	if secretType == "" {
		secretType = defaultSecretType
	}

	viewSecretValue := false
	result, err := d.client.GetSingleRawSecretByNameV3(infisical.GetSingleSecretByNameV3Request{
		SecretName:  data.Name.ValueString(),
		WorkspaceId: data.ProjectID.ValueString(),
		Environment: data.EnvironmentSlug.ValueString(),
		SecretPath:  data.FolderPath.ValueString(),
		Type:        secretType,
	}, &viewSecretValue)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to fetch secret properties",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.SecretVersion = types.Int64Value(int64(result.Secret.Version))
	data.SecretType = types.StringValue(result.Secret.Type)

	metadataObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"key":          types.StringType,
			"value":        types.StringType,
			"is_encrypted": types.BoolType,
		},
	}

	metadataItems := make([]attr.Value, len(result.Secret.SecretMetadata))
	for i, item := range result.Secret.SecretMetadata {
		obj, diags := types.ObjectValue(metadataObjType.AttrTypes, map[string]attr.Value{
			"key":          types.StringValue(item.Key),
			"value":        types.StringValue(item.Value),
			"is_encrypted": types.BoolValue(item.IsEncrypted),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		metadataItems[i] = obj
	}
	data.SecretMetadata, _ = types.ListValue(metadataObjType, metadataItems)

	tagObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":    types.StringType,
			"slug":  types.StringType,
			"name":  types.StringType,
			"color": types.StringType,
		},
	}

	tagItems := make([]attr.Value, len(result.Secret.Tags))
	for i, tag := range result.Secret.Tags {
		obj, diags := types.ObjectValue(tagObjType.AttrTypes, map[string]attr.Value{
			"id":    types.StringValue(tag.ID),
			"slug":  types.StringValue(tag.Slug),
			"name":  types.StringValue(tag.Name),
			"color": types.StringValue(tag.Color),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		tagItems[i] = obj
	}
	data.Tags, _ = types.ListValue(tagObjType, tagItems)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
