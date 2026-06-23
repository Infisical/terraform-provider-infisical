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
var _ datasource.DataSource = &ProjectUserDataSource{}

func NewProjectUserDataSource() datasource.DataSource {
	return &ProjectUserDataSource{}
}

// ProjectUserDataSource defines the data source implementation.
type ProjectUserDataSource struct {
	client *infisical.Client
}

// ProjectUserDataSourceModel describes the data source data model.
type ProjectUserDataSourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	Username     types.String `tfsdk:"username"`
	MembershipID types.String `tfsdk:"membership_id"`
	UserID       types.String `tfsdk:"user_id"`
}

func (d *ProjectUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_user"
}

func (d *ProjectUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a project user membership by project ID and username. Returns null values if the membership does not exist. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the project",
				Required:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username of the user (by default the email address)",
				Required:    true,
			},
			"membership_id": schema.StringAttribute{
				Description: "The membership UUID. Null if the membership does not exist.",
				Computed:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "The user's Infisical UUID. Null if the membership does not exist.",
				Computed:    true,
			},
		},
	}
}

func (d *ProjectUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch project user membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data ProjectUserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	membership, err := d.client.GetProjectUserByUsername(infisical.GetProjectUserByUserNameRequest{
		ProjectID: data.ProjectID.ValueString(),
		Username:  data.Username.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			tflog.Debug(ctx, "Project user membership not found", map[string]interface{}{
				"username":   data.Username.ValueString(),
				"project_id": data.ProjectID.ValueString(),
			})
			data.MembershipID = types.StringNull()
			data.UserID = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the project user membership",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.MembershipID = types.StringValue(membership.Membership.ID)
	data.UserID = types.StringValue(membership.Membership.User.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
