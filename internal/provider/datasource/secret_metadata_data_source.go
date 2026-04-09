package datasource

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SecretMetadataDataSource{}

func NewSecretMetadataDataSource() datasource.DataSource {
	return &SecretMetadataDataSource{}
}

type SecretMetadataDataSource struct {
	client *infisical.Client
}

type SecretMetadataDataSourceModel struct {
	Name          types.String `tfsdk:"name"`
	EnvSlug       types.String `tfsdk:"env_slug"`
	WorkspaceId   types.String `tfsdk:"workspace_id"`
	FolderPath    types.String `tfsdk:"folder_path"`
	Type          types.String `tfsdk:"type"`
	Workspace     types.String `tfsdk:"workspace"`
	SecretVersion types.Int64  `tfsdk:"secret_version"`
	Environment   types.String `tfsdk:"environment"`
}

func (d *SecretMetadataDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_metadata"
}

func (d *SecretMetadataDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve metadata for a single Infisical secret without exposing the secret value.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the secret to retrieve metadata for.",
				Required:    true,
			},
			"env_slug": schema.StringAttribute{
				Description: "The environment slug where the secret resides.",
				Required:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "The Infisical project ID.",
				Required:    true,
			},
			"folder_path": schema.StringAttribute{
				Description: "The path to the folder where the secret is located.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the secret (shared or personal). Defaults to shared.",
				Optional:    true,
				Computed:    true,
			},
			"workspace": schema.StringAttribute{
				Description: "The workspace ID of the secret.",
				Computed:    true,
			},
			"secret_version": schema.Int64Attribute{
				Description: "The version number of the secret.",
				Computed:    true,
			},
			"environment": schema.StringAttribute{
				Description: "The environment slug of the secret.",
				Computed:    true,
			},
		},
	}
}

func (d *SecretMetadataDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SecretMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch secret metadata",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data SecretMetadataDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretType := "shared"
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		secretType = data.Type.ValueString()
	}

	result, err := d.client.GetSingleRawSecretMetadataByNameV3(infisical.GetSingleSecretByNameV3Request{
		SecretName:  data.Name.ValueString(),
		WorkspaceId: data.WorkspaceId.ValueString(),
		Environment: data.EnvSlug.ValueString(),
		SecretPath:  data.FolderPath.ValueString(),
		Type:        secretType,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to fetch secret metadata",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.Workspace = types.StringValue(result.Secret.Workspace)
	data.SecretVersion = types.Int64Value(int64(result.Secret.Version))
	data.Environment = types.StringValue(result.Secret.Environment)
	data.Type = types.StringValue(result.Secret.Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
