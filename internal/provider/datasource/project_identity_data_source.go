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
var _ datasource.DataSource = &ProjectIdentityDataSource{}

func NewProjectIdentityDataSource() datasource.DataSource {
	return &ProjectIdentityDataSource{}
}

// ProjectIdentityDataSource defines the data source implementation.
type ProjectIdentityDataSource struct {
	client *infisical.Client
}

// ProjectIdentityDataSourceModel describes the data source data model.
type ProjectIdentityDataSourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	IdentityID   types.String `tfsdk:"identity_id"`
	MembershipID types.String `tfsdk:"membership_id"`
	RoleSlug     types.String `tfsdk:"role_slug"`
	CustomRoleID types.String `tfsdk:"custom_role_id"`
}

func (d *ProjectIdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_identity"
}

func (d *ProjectIdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a project identity membership by project ID and identity ID. Returns null values if the membership does not exist. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the project",
				Required:    true,
			},
			"identity_id": schema.StringAttribute{
				Description: "The ID of the identity",
				Required:    true,
			},
			"membership_id": schema.StringAttribute{
				Description: "The membership UUID. Null if the membership does not exist.",
				Computed:    true,
			},
			"role_slug": schema.StringAttribute{
				Description: "The slug of the first assigned role. Null if the membership does not exist.",
				Computed:    true,
			},
			"custom_role_id": schema.StringAttribute{
				Description: "The custom role ID of the first assigned role, if applicable. Null if not set.",
				Computed:    true,
			},
		},
	}
}

func (d *ProjectIdentityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectIdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch project identity membership",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data ProjectIdentityDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	membership, err := d.client.GetProjectIdentityByID(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  data.ProjectID.ValueString(),
		IdentityID: data.IdentityID.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			tflog.Debug(ctx, "Project identity membership not found", map[string]interface{}{
				"identity_id": data.IdentityID.ValueString(),
				"project_id":  data.ProjectID.ValueString(),
			})
			data.MembershipID = types.StringNull()
			data.RoleSlug = types.StringNull()
			data.CustomRoleID = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the project identity membership",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.MembershipID = types.StringValue(membership.Membership.ID)

	if len(membership.Membership.Roles) > 0 {
		role := membership.Membership.Roles[0]
		slug := role.Role
		if role.CustomRoleSlug != "" {
			slug = role.CustomRoleSlug
		}
		data.RoleSlug = types.StringValue(slug)
		if role.CustomRoleId != "" {
			data.CustomRoleID = types.StringValue(role.CustomRoleId)
		} else {
			data.CustomRoleID = types.StringNull()
		}
	} else {
		data.RoleSlug = types.StringNull()
		data.CustomRoleID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
