package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"tag_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Tag ids to be attached for the secrets.",
			},
			"secret_reminder": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"note": schema.StringAttribute{
						Description: "Note for the secret rotation reminder",
						Computed:    true,
					},
					"repeat_days": schema.Int64Attribute{
						Description: "Frequency of secret rotation reminder in days",
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
							int64validator.AtMost(365),
						},
					},
				},
				Computed: true,
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
	var config secretResourceModel
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

	tagIds := []string{}
	for _, tag := range res.Secret.Tags {
		tagIds = append(tagIds, tag.ID)
	}

	tagsList, diags := types.ListValueFrom(ctx, types.StringType, tagIds)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Result.Set(ctx, secretResourceModel{
		Value:      types.StringValue(res.Secret.SecretValue),
		Name:       types.StringValue(res.Secret.SecretKey),
		FolderPath: config.FolderPath,
		EnvSlug:    config.EnvSlug,
		SecretReminder: &SecretReminder{
			Note:       types.StringValue(res.Secret.SecretReminderNote),
			RepeatDays: types.Int64Value(res.Secret.SecretReminderRepeatDays),
		},
		WorkspaceId: config.WorkspaceId,
		Tags:        tagsList,
	})
}
