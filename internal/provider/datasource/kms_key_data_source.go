package datasource

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &KMSKeyDataSource{}

func NewKMSKeyDataSource() datasource.DataSource {
	return &KMSKeyDataSource{}
}

type KMSKeyDataSource struct {
	client *infisical.Client
}

type KMSKeyDataSourceModel struct {
	KeyId               types.String `tfsdk:"key_id"`
	Name                types.String `tfsdk:"name"`
	KeyUsage            types.String `tfsdk:"key_usage"`
	EncryptionAlgorithm types.String `tfsdk:"encryption_algorithm"`
	PublicKey           types.String `tfsdk:"public_key"`
	SigningAlgorithms   types.List   `tfsdk:"signing_algorithms"`
}

func (d *KMSKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_key_public_key"
}

func (d *KMSKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve the public key and signing algorithms for a KMS key. This data source is specifically designed for getting public keys for cryptographic operations.",

		Attributes: map[string]schema.Attribute{
			"key_id": schema.StringAttribute{
				Description: "The ID of the KMS key to retrieve the public key for.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the KMS key.",
				Computed:    true,
			},
			"key_usage": schema.StringAttribute{
				Description: "The usage of the key ('encrypt-decrypt' or 'sign-verify').",
				Computed:    true,
			},
			"encryption_algorithm": schema.StringAttribute{
				Description: "The encryption algorithm used by the key.",
				Computed:    true,
			},
			"public_key": schema.StringAttribute{
				Description: "The public key. Only available for signing keys (key_usage = 'sign-verify').",
				Computed:    true,
			},
			"signing_algorithms": schema.ListAttribute{
				Description: "List of available signing algorithms. Only available for signing keys (key_usage = 'sign-verify').",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *KMSKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KMSKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config KMSKeyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	kmsKey, err := d.client.GetKMSKey(infisical.GetKMSKeyRequest{
		KeyId: config.KeyId.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read KMS key",
			"An error occurred while reading the KMS key: "+err.Error(),
		)
		return
	}

	config.Name = types.StringValue(kmsKey.Key.Name)
	config.KeyUsage = types.StringValue(kmsKey.Key.KeyUsage)
	config.EncryptionAlgorithm = types.StringValue(kmsKey.Key.EncryptionAlgorithm)

	if kmsKey.Key.KeyUsage == "sign-verify" {
		publicKeyResp, pubKeyErr := d.client.GetKMSKeyPublicKey(infisical.GetKMSKeyPublicKeyRequest{
			KeyId: kmsKey.Key.ID,
		})
		if pubKeyErr == nil {
			config.PublicKey = types.StringValue(publicKeyResp.PublicKey)
		} else {
			resp.Diagnostics.AddWarning(
				"Unable to retrieve public key",
				"The KMS key was found but the public key could not be retrieved: "+pubKeyErr.Error(),
			)
			config.PublicKey = types.StringNull()
		}

		signingAlgResp, sigAlgErr := d.client.GetKMSKeySigningAlgorithms(infisical.GetKMSKeySigningAlgorithmsRequest{
			KeyId: kmsKey.Key.ID,
		})
		if sigAlgErr == nil {
			signingAlgorithms := make([]types.String, len(signingAlgResp.SigningAlgorithms))
			for i, alg := range signingAlgResp.SigningAlgorithms {
				signingAlgorithms[i] = types.StringValue(alg)
			}
			signingAlgList, diags := types.ListValueFrom(ctx, types.StringType, signingAlgorithms)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				config.SigningAlgorithms = signingAlgList
			}
		} else {
			resp.Diagnostics.AddWarning(
				"Unable to retrieve signing algorithms",
				"The KMS key was found but the signing algorithms could not be retrieved: "+sigAlgErr.Error(),
			)
			config.SigningAlgorithms = types.ListNull(types.StringType)
		}
	} else {
		config.PublicKey = types.StringNull()
		config.SigningAlgorithms = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
