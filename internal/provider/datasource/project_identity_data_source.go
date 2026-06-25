package datasource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	Roles        types.List   `tfsdk:"roles"`
}

// ProjectIdentityDataSourceRole describes a single role assigned to the membership.
type ProjectIdentityDataSourceRole struct {
	ID                       types.String `tfsdk:"id"`
	RoleSlug                 types.String `tfsdk:"role_slug"`
	CustomRoleID             types.String `tfsdk:"custom_role_id"`
	IsTemporary              types.Bool   `tfsdk:"is_temporary"`
	TemporaryMode            types.String `tfsdk:"temporary_mode"`
	TemporaryRange           types.String `tfsdk:"temporary_range"`
	TemporaryAccessStartTime types.String `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime   types.String `tfsdk:"temporary_access_end_time"`
}

// projectIdentityRoleAttrTypes is the attribute type map for a single role object.
// Shared by types.ListNull and types.ListValueFrom to keep the schema and the
// list element type in sync.
var projectIdentityRoleAttrTypes = map[string]attr.Type{
	"id":                          types.StringType,
	"role_slug":                   types.StringType,
	"custom_role_id":              types.StringType,
	"is_temporary":                types.BoolType,
	"temporary_mode":              types.StringType,
	"temporary_range":             types.StringType,
	"temporary_access_start_time": types.StringType,
	"temporary_access_end_time":   types.StringType,
}

func (d *ProjectIdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_identity"
}

func (d *ProjectIdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a project identity membership by project ID and identity ID. Returns all roles assigned to the membership. Returns null values if the membership does not exist. Only Machine Identity authentication is supported for this data source.",
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
			"roles": schema.ListNestedAttribute{
				Description: "The roles assigned to the project identity. Null if the membership does not exist.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the project identity role.",
							Computed:    true,
						},
						"role_slug": schema.StringAttribute{
							Description: "The slug of the role.",
							Computed:    true,
						},
						"custom_role_id": schema.StringAttribute{
							Description: "The ID of the custom role, if applicable.",
							Computed:    true,
						},
						"is_temporary": schema.BoolAttribute{
							Description: "Flag to indicate whether the assigned role is temporary.",
							Computed:    true,
						},
						"temporary_mode": schema.StringAttribute{
							Description: "Type of temporary access given. Null for permanent roles.",
							Computed:    true,
						},
						"temporary_range": schema.StringAttribute{
							Description: "TTL for the temporary access. Null for permanent roles.",
							Computed:    true,
						},
						"temporary_access_start_time": schema.StringAttribute{
							Description: "ISO time at which temporary access begins. Null for permanent roles.",
							Computed:    true,
						},
						"temporary_access_end_time": schema.StringAttribute{
							Description: "ISO time at which temporary access ends. Null for permanent roles.",
							Computed:    true,
						},
					},
				},
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

	roleObjectType := types.ObjectType{AttrTypes: projectIdentityRoleAttrTypes}

	membership, err := d.client.GetProjectIdentityByID(infisical.GetProjectIdentityByIDRequest{
		ProjectID:  data.ProjectID.ValueString(),
		IdentityID: data.IdentityID.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			data.MembershipID = types.StringNull()
			data.Roles = types.ListNull(roleObjectType)
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

	roles := make([]ProjectIdentityDataSourceRole, 0, len(membership.Membership.Roles))
	for _, el := range membership.Membership.Roles {
		val := ProjectIdentityDataSourceRole{
			ID:                       types.StringValue(el.ID),
			RoleSlug:                 types.StringValue(el.Role),
			CustomRoleID:             types.StringNull(),
			IsTemporary:              types.BoolValue(el.IsTemporary),
			TemporaryMode:            types.StringValue(el.TemporaryMode),
			TemporaryRange:           types.StringValue(el.TemporaryRange),
			TemporaryAccessStartTime: types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339)),
			TemporaryAccessEndTime:   types.StringValue(el.TemporaryAccessEndTime.Format(time.RFC3339)),
		}
		if el.CustomRoleId != "" {
			val.RoleSlug = types.StringValue(el.CustomRoleSlug)
			val.CustomRoleID = types.StringValue(el.CustomRoleId)
		}
		if !el.IsTemporary {
			val.TemporaryMode = types.StringNull()
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccessStartTime = types.StringNull()
			val.TemporaryAccessEndTime = types.StringNull()
		}
		roles = append(roles, val)
	}

	// Sort roles by slug, then id, for deterministic output even when slugs collide.
	sort.SliceStable(roles, func(i, j int) bool {
		if roles[i].RoleSlug.ValueString() != roles[j].RoleSlug.ValueString() {
			return roles[i].RoleSlug.ValueString() < roles[j].RoleSlug.ValueString()
		}
		return roles[i].ID.ValueString() < roles[j].ID.ValueString()
	})

	roleList, diags := types.ListValueFrom(ctx, roleObjectType, roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Roles = roleList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
