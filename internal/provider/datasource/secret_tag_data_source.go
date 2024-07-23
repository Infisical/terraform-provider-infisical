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
var _ datasource.DataSource = &SecretTagsDataSource{}

func NewSecretTagDataSource() datasource.DataSource {
	return &SecretTagsDataSource{}
}

// SecretDataSource defines the data source implementation.
type SecretTagsDataSource struct {
	client *infisical.Client
}

// ExampleDataSourceModel describes the data source data model.
type SecretTagDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	Name      types.String `tfsdk:"name"`
	Slug      types.String `tfsdk:"slug"`
	Color     types.String `tfsdk:"color"`
}

func (d *SecretTagsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_tag"
}

func (d *SecretTagsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Infisical secretTag secret tag.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug of the tag to fetch",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project id associated with the secret tag",
				Required:    true,
			},

			"id": schema.StringAttribute{
				Description: "The ID of the secret tag",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the secret tag",
				Computed:    true,
			},
			"color": schema.StringAttribute{
				Description: "The color of the secret tag",
				Computed:    true,
			},
		},
	}
}

func (d *SecretTagsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SecretTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secretTag tag",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data SecretTagDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	secretTag, err := d.client.GetProjectTagBySlug(infisical.GetProjectTagBySlugRequest{
		TagSlug:   data.Slug.ValueString(),
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the secret tag",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
	}

	data = SecretTagDataSourceModel{
		ID:        types.StringValue(secretTag.Tag.ID),
		Name:      types.StringValue(secretTag.Tag.Name),
		Slug:      types.StringValue(secretTag.Tag.Slug),
		ProjectID: data.ProjectID,
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
