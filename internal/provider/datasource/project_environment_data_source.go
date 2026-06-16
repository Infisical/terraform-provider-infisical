package datasource

import (
	"context"
	"errors"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ProjectEnvironmentDataSource{}

func NewProjectEnvironmentDataSource() datasource.DataSource {
	return &ProjectEnvironmentDataSource{}
}

// ProjectEnvironmentDataSource defines the data source implementation.
type ProjectEnvironmentDataSource struct {
	client *infisical.Client
}

// ProjectEnvironmentDataSourceModel describes the data source data model.
type ProjectEnvironmentDataSourceModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	Slug      types.String `tfsdk:"slug"`
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
}

func (d *ProjectEnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment"
}

func (d *ProjectEnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a project environment by project ID and slug. Returns null values if the environment does not exist. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the project",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the environment",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The UUID of the environment. Null if the environment does not exist.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the environment. Null if the environment does not exist.",
				Computed:    true,
			},
		},
	}
}

func (d *ProjectEnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data ProjectEnvironmentDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, err := d.client.GetProjectById(infisical.GetProjectByIdRequest{
		ID: data.ProjectID.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			data.ID = types.StringNull()
			data.Name = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the project environment",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	for _, env := range project.Environments {
		if env.Slug == data.Slug.ValueString() {
			data.ID = types.StringValue(env.ID)
			data.Name = types.StringValue(env.Name)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	// Environment not found in project
	data.ID = types.StringNull()
	data.Name = types.StringNull()

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
