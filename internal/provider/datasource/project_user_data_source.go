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
	UserID       types.String `tfsdk:"user_id"`
	MembershipID types.String `tfsdk:"membership_id"`
	Username     types.String `tfsdk:"username"`
	User         types.Object `tfsdk:"user"`
	Roles        types.List   `tfsdk:"roles"`
}

// ProjectUserDataSourceRole describes a single role assigned to the membership.
type ProjectUserDataSourceRole struct {
	ID                       types.String `tfsdk:"id"`
	RoleSlug                 types.String `tfsdk:"role_slug"`
	CustomRoleID             types.String `tfsdk:"custom_role_id"`
	IsTemporary              types.Bool   `tfsdk:"is_temporary"`
	TemporaryMode            types.String `tfsdk:"temporary_mode"`
	TemporaryRange           types.String `tfsdk:"temporary_range"`
	TemporaryAccessStartTime types.String `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime   types.String `tfsdk:"temporary_access_end_time"`
}

// ProjectUserDataSourceUser describes the user personal details of the membership.
type ProjectUserDataSourceUser struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
}

// projectUserRoleAttrTypes is the attribute type map for a single role object.
// Shared by types.ListNull and types.ListValueFrom to keep the schema and the
// list element type in sync.
var projectUserRoleAttrTypes = map[string]attr.Type{
	"id":                          types.StringType,
	"role_slug":                   types.StringType,
	"custom_role_id":              types.StringType,
	"is_temporary":                types.BoolType,
	"temporary_mode":              types.StringType,
	"temporary_range":             types.StringType,
	"temporary_access_start_time": types.StringType,
	"temporary_access_end_time":   types.StringType,
}

// projectUserUserAttrTypes is the attribute type map for the user object.
// Shared by types.ObjectNull and types.ObjectValueFrom to keep the schema and
// the object type in sync.
var projectUserUserAttrTypes = map[string]attr.Type{
	"id":         types.StringType,
	"email":      types.StringType,
	"first_name": types.StringType,
	"last_name":  types.StringType,
}

func (d *ProjectUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_user"
}

func (d *ProjectUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a project user membership by project ID and user ID. Returns all roles assigned to the membership. Returns null values if the membership does not exist. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the project",
				Required:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "The user's Infisical UUID",
				Required:    true,
			},
			"membership_id": schema.StringAttribute{
				Description: "The membership UUID. Null if the membership does not exist.",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username of the user (by default the email address). Null if the membership does not exist.",
				Computed:    true,
			},
			"user": schema.SingleNestedAttribute{
				Description: "The user personal details. Null if the membership does not exist.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The id of the user",
						Computed:    true,
					},
					"email": schema.StringAttribute{
						Description: "The email of the user",
						Computed:    true,
					},
					"first_name": schema.StringAttribute{
						Description: "The first name of the user",
						Computed:    true,
					},
					"last_name": schema.StringAttribute{
						Description: "The last name of the user",
						Computed:    true,
					},
				},
			},
			"roles": schema.ListNestedAttribute{
				Description: "The roles assigned to the project user. Null if the membership does not exist.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the project user role.",
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

	roleObjectType := types.ObjectType{AttrTypes: projectUserRoleAttrTypes}

	projectUserDetails, err := d.client.GetProjectMembershipByUserID(infisical.GetProjectMembershipByUserIDRequest{
		ProjectID: data.ProjectID.ValueString(),
		UserID:    data.UserID.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			data.MembershipID = types.StringNull()
			data.UserID = types.StringNull()
			data.Username = types.StringNull()
			data.User = types.ObjectNull(projectUserUserAttrTypes)
			data.Roles = types.ListNull(roleObjectType)
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

	data.MembershipID = types.StringValue(projectUserDetails.Membership.ID)
	data.Username = types.StringValue(projectUserDetails.Membership.User.Username)

	roles := make([]ProjectUserDataSourceRole, 0, len(projectUserDetails.Membership.Roles))
	for _, el := range projectUserDetails.Membership.Roles {
		val := ProjectUserDataSourceRole{
			ID:                       types.StringValue(el.ID),
			RoleSlug:                 types.StringValue(el.Role),
			CustomRoleID:             types.StringValue(el.CustomRoleId),
			IsTemporary:              types.BoolValue(el.IsTemporary),
			TemporaryMode:            types.StringValue(el.TemporaryMode),
			TemporaryRange:           types.StringValue(el.TemporaryRange),
			TemporaryAccessStartTime: types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339)),
			TemporaryAccessEndTime:   types.StringValue(el.TemporaryAccessEndTime.Format(time.RFC3339)),
		}
		if el.CustomRoleId != "" {
			val.RoleSlug = types.StringValue(el.CustomRoleSlug)
		}
		if !el.IsTemporary {
			val.TemporaryMode = types.StringNull()
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccessStartTime = types.StringNull()
			val.TemporaryAccessEndTime = types.StringNull()
		}
		roles = append(roles, val)
	}

	// Sort roles alphabetically by slug for deterministic output.
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].RoleSlug.ValueString() < roles[j].RoleSlug.ValueString()
	})

	roleList, diags := types.ListValueFrom(ctx, roleObjectType, roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Roles = roleList

	userObj, diags := types.ObjectValueFrom(ctx, projectUserUserAttrTypes, ProjectUserDataSourceUser{
		ID:        types.StringValue(projectUserDetails.Membership.User.ID),
		Email:     types.StringValue(projectUserDetails.Membership.User.Email),
		FirstName: types.StringValue(projectUserDetails.Membership.User.FirstName),
		LastName:  types.StringValue(projectUserDetails.Membership.User.LastName),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.User = userObj

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
