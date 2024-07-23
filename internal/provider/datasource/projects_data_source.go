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
var _ datasource.DataSource = &ProjectsDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectsDataSource{}
}

// SecretDataSource defines the data source implementation.
type ProjectsDataSource struct {
	client *infisical.Client
}

// ExampleDataSourceModel describes the data source data model.
type ProjectDataSourceModel struct {
	ID                 types.String                         `tfsdk:"id"`
	Name               types.String                         `tfsdk:"name"`
	Slug               types.String                         `tfsdk:"slug"`
	AutoCapitalization types.Bool                           `tfsdk:"auto_capitalization"`
	OrgID              types.String                         `tfsdk:"org_id"`
	CreatedAt          types.String                         `tfsdk:"created_at"`
	UpdatedAt          types.String                         `tfsdk:"updated_at"`
	Version            types.Int64                          `tfsdk:"version"`
	UpgradeStatus      types.String                         `tfsdk:"upgrade_status"`
	Environments       map[string]ProjectEnvironmentDetails `tfsdk:"environments"`
}

type ProjectEnvironmentDetails struct {
	Name types.String `tfsdk:"name"`
	Slug types.String `tfsdk:"slug"`
	ID   types.String `tfsdk:"id"`
}

func (d *ProjectsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *ProjectsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Infisical projects. Only Machine Identity authentication is supported for this data source.",

		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug of the project to fetch",
				Required:    true,
			},

			"id": schema.StringAttribute{
				Description: "The ID of the project",
				Computed:    true,
			},

			"name": schema.StringAttribute{
				Description: "The name of the project",
				Computed:    true,
			},

			"auto_capitalization": schema.BoolAttribute{
				Description: "The auto capitalization status of the project",
				Computed:    true,
			},

			"org_id": schema.StringAttribute{
				Description: "The ID of the organization to which the project belongs",
				Computed:    true,
			},

			"created_at": schema.StringAttribute{
				Description: "The creation date of the project",
				Computed:    true,
			},

			"updated_at": schema.StringAttribute{
				Description: "The last update date of the project",
				Computed:    true,
			},

			"version": schema.Int64Attribute{
				Description: "The version of the project",
				Computed:    true,
			},

			"upgrade_status": schema.StringAttribute{
				Description: "The upgrade status of the project",
				Computed:    true,
			},

			"environments": schema.MapNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the environment",
							Computed:    true,
						},
						"slug": schema.StringAttribute{
							Description: "The slug of the environment",
							Computed:    true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the environment",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ProjectsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data ProjectDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, err := d.client.GetProject(infisical.GetProjectRequest{
		Slug: data.Slug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the project",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
	}

	data = ProjectDataSourceModel{
		ID:                 types.StringValue(project.ID),
		Name:               types.StringValue(project.Name),
		Slug:               types.StringValue(project.Slug),
		AutoCapitalization: types.BoolValue(project.AutoCapitalization),
		OrgID:              types.StringValue(project.OrgID),
		CreatedAt:          types.StringValue(project.CreatedAt),
		UpdatedAt:          types.StringValue(project.UpdatedAt),
		Version:            types.Int64Value(project.Version),
		UpgradeStatus:      types.StringValue(project.UpgradeStatus),
		Environments:       data.Environments,
	}

	data.Environments = make(map[string]ProjectEnvironmentDetails)

	for _, env := range project.Environments {
		data.Environments[env.Slug] = ProjectEnvironmentDetails{
			Name: types.StringValue(env.Name),
			Slug: types.StringValue(env.Slug),
			ID:   types.StringValue(env.ID),
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
