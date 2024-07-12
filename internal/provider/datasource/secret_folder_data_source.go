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
var _ datasource.DataSource = &SecretFoldersDataSource{}

func NewSecretFolderDataSource() datasource.DataSource {
	return &SecretFoldersDataSource{}
}

// SecretDataSource defines the data source implementation.
type SecretFoldersDataSource struct {
	client *infisical.Client
}

// ExampleDataSourceModel describes the data source data model.
type SecretFolderDataSourceModel struct {
	ProjectID       types.String `tfsdk:"project_id"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	SecretPath      types.String `tfsdk:"folder_path"`
	Folders         types.List   `tfsdk:"folders"`
}

type InfisicalSecretFolderDetails struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *SecretFoldersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_folders"
}

func (d *SecretFoldersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Infisical secret folders.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The project id associated with the secret folder",
				Required:    true,
			},
			"folder_path": schema.StringAttribute{
				Description: "The path to the folder from where folder should be fetched from",
				Required:    true,
				Computed:    false,
			},
			"environment_slug": schema.StringAttribute{
				Description: "The environment from where folder should be fetched from",
				Required:    true,
				Computed:    false,
			},
			"folders": schema.ListNestedAttribute{
				Description: "The folder list",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the folder",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the folder",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *SecretFoldersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SecretFoldersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if d.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to create secretFolder folder",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data SecretFolderDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	secretFolder, err := d.client.GetSecretFolderList(infisical.ListSecretFolderRequest{
		ProjectID:   data.ProjectID.ValueString(),
		Environment: data.EnvironmentSlug.ValueString(),
		SecretPath:  data.SecretPath.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the secret folders",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
	}

	planFolders := make([]InfisicalSecretFolderDetails, len(secretFolder.Folders))
	for i, el := range secretFolder.Folders {
		planFolders[i] = InfisicalSecretFolderDetails{
			ID:   types.StringValue(el.ID),
			Name: types.StringValue(el.Name),
		}
	}

	stateFolders, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":   types.StringType,
			"name": types.StringType,
		},
	}, planFolders)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Folders = stateFolders
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
