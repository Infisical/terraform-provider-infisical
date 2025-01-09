package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ ephemeral.EphemeralResource = &ephemeralSecretResource{}
)

func NewEphemeralSecretResource() ephemeral.EphemeralResourceWithConfigure {
	return &ephemeralSecretResource{}
}

// secretResource is the resource implementation.
type ephemeralSecretResource struct {
	client *infisical.Client
}

type ephemeralSecretResourceModel struct {
	FolderPath  types.String `tfsdk:"folder_path"`
	EnvSlug     types.String `tfsdk:"env_slug"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
}

// Metadata returns the resource type name.
func (r *ephemeralSecretResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

// Schema defines the schema for the resource.
func (r *ephemeralSecretResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read ephemeral secrets from Infisical",
		Attributes: map[string]schema.Attribute{
			"folder_path": schema.StringAttribute{
				Description: "The path to the folder where the given secret resides",
				Required:    true,
				Computed:    false,
			},
			"env_slug": schema.StringAttribute{
				Description: "The environment slug of the secret to fetch",
				Required:    true,
				Computed:    false,
			},
			"name": schema.StringAttribute{
				Description: "The name of the secret",
				Required:    true,
				Computed:    false,
			},
			"workspace_id": schema.StringAttribute{
				Description: "The Infisical project ID",
				Required:    true,
				Computed:    false,
			},
			"value": schema.StringAttribute{
				Description: "The value of the secret",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ephemeralSecretResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

	r.client = client

}

func (r *ephemeralSecretResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client not configured",
			"The provider client is nil. Please report this issue to the Infisical provider developers.",
		)
		return
	}

	// Read configuration from the request
	var config ephemeralSecretResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Error Reading Infisical secret",
			"Unknown authentication strategy",
		)
		return
	}

	res, err := r.client.GetSingleRawSecretByNameV3(infisical.GetSingleSecretByNameV3Request{
		SecretName:  config.Name.ValueString(),
		Type:        "shared",
		WorkspaceId: config.WorkspaceId.ValueString(),
		Environment: config.EnvSlug.ValueString(),
		SecretPath:  config.FolderPath.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Infisical secret",
			"Could not read Infisical secret named "+config.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Result.Set(ctx, ephemeralSecretResourceModel{
		Value:       types.StringValue(res.Secret.SecretValue),
		Name:        types.StringValue(res.Secret.SecretKey),
		FolderPath:  config.FolderPath,
		EnvSlug:     config.EnvSlug,
		WorkspaceId: config.WorkspaceId,
	})
}
