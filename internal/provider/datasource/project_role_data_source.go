package datasource

import (
	"context"
	"errors"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ProjectRoleDataSource{}

func NewProjectRoleDataSource() datasource.DataSource {
	return &ProjectRoleDataSource{}
}

// ProjectRoleDataSource defines the data source implementation.
type ProjectRoleDataSource struct {
	client *infisical.Client
}

// ProjectRoleDataSourceModel describes the data source data model.
type ProjectRoleDataSourceModel struct {
	ProjectID   types.String `tfsdk:"project_id"`
	Slug        types.String `tfsdk:"slug"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (d *ProjectRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_role"
}

func (d *ProjectRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a project role by project ID and slug. Returns null values if the role does not exist. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the project",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the role",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The UUID of the role. Null if the role does not exist.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the role. Null if the role does not exist.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the role. Null if the role does not exist.",
				Computed:    true,
			},
		},
	}
}

func (d *ProjectRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch project role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data ProjectRoleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	role, err := d.client.GetProjectRoleBySlugV2(infisical.GetProjectRoleBySlugV2Request{
		ProjectId: data.ProjectID.ValueString(),
		RoleSlug:  data.Slug.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			tflog.Debug(ctx, "Project role not found", map[string]interface{}{
				"slug":       data.Slug.ValueString(),
				"project_id": data.ProjectID.ValueString(),
			})
			data.ID = types.StringNull()
			data.Name = types.StringNull()
			data.Description = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the project role",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(role.Role.ID)
	data.Name = types.StringValue(role.Role.Name)
	data.Description = types.StringValue(role.Role.Description)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
