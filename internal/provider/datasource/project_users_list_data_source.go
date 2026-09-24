package datasource

import (
	"context"
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
var _ datasource.DataSource = &ProjectUsersListDataSource{}

func NewProjectUsersListDataSource() datasource.DataSource {
	return &ProjectUsersListDataSource{}
}

// ProjectUsersListDataSource defines the data source implementation.
type ProjectUsersListDataSource struct {
	client *infisical.Client
}

// ProjectUsersListDataSourceModel describes the data source data model.
type ProjectUsersListDataSourceModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	Usernames types.List   `tfsdk:"usernames"`
	Members   types.List   `tfsdk:"members"`
}

// ProjectUsersListMemberModel describes a single member in the list output.
type ProjectUsersListMemberModel struct {
	MembershipID types.String `tfsdk:"membership_id"`
	UserID       types.String `tfsdk:"user_id"`
	Username     types.String `tfsdk:"username"`
	Email        types.String `tfsdk:"email"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
	Roles        types.List   `tfsdk:"roles"`
}

// ProjectUsersListRoleModel describes a single role assigned to a membership.
type ProjectUsersListRoleModel struct {
	ID                       types.String `tfsdk:"id"`
	RoleSlug                 types.String `tfsdk:"role_slug"`
	CustomRoleID             types.String `tfsdk:"custom_role_id"`
	IsTemporary              types.Bool   `tfsdk:"is_temporary"`
	TemporaryMode            types.String `tfsdk:"temporary_mode"`
	TemporaryRange           types.String `tfsdk:"temporary_range"`
	TemporaryAccessStartTime types.String `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime   types.String `tfsdk:"temporary_access_end_time"`
}

// projectUsersListRoleAttrTypes is the attribute type map for a single role object.
var projectUsersListRoleAttrTypes = map[string]attr.Type{
	"id":                          types.StringType,
	"role_slug":                   types.StringType,
	"custom_role_id":              types.StringType,
	"is_temporary":                types.BoolType,
	"temporary_mode":              types.StringType,
	"temporary_range":             types.StringType,
	"temporary_access_start_time": types.StringType,
	"temporary_access_end_time":   types.StringType,
}

// projectUsersListMemberAttrTypes is the attribute type map for a single member object.
var projectUsersListMemberAttrTypes = map[string]attr.Type{
	"membership_id": types.StringType,
	"user_id":       types.StringType,
	"username":      types.StringType,
	"email":         types.StringType,
	"first_name":    types.StringType,
	"last_name":     types.StringType,
	"roles":         types.ListType{ElemType: types.ObjectType{AttrTypes: projectUsersListRoleAttrTypes}},
}

func (d *ProjectUsersListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_users_list"
}

func (d *ProjectUsersListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch the list of user memberships for a project, optionally filtered by username. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the project to fetch user memberships for.",
				Required:    true,
			},
			"usernames": schema.ListAttribute{
				Description: "The usernames to filter by. If omitted or empty, all project members are returned. Usernames that do not match any member are ignored.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"members": schema.ListNestedAttribute{
				Description: "The list of project user memberships matching the provided usernames (or all members when no usernames are provided).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"membership_id": schema.StringAttribute{
							Description: "The membership UUID.",
							Computed:    true,
						},
						"user_id": schema.StringAttribute{
							Description: "The user's Infisical UUID.",
							Computed:    true,
						},
						"username": schema.StringAttribute{
							Description: "The username of the user (by default the email address).",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The email address of the user.",
							Computed:    true,
						},
						"first_name": schema.StringAttribute{
							Description: "The first name of the user.",
							Computed:    true,
						},
						"last_name": schema.StringAttribute{
							Description: "The last name of the user.",
							Computed:    true,
						},
						"roles": schema.ListNestedAttribute{
							Description: "The roles assigned to the project user.",
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
				},
			},
		},
	}
}

func (d *ProjectUsersListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectUsersListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch project user memberships",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data ProjectUsersListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read the optional usernames filter. When no usernames are provided, all members of the project are returned.
	var requestedUsernames []string
	if !data.Usernames.IsNull() && !data.Usernames.IsUnknown() {
		resp.Diagnostics.Append(data.Usernames.ElementsAs(ctx, &requestedUsernames, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	usernameFilter := make(map[string]struct{}, len(requestedUsernames))
	for _, username := range requestedUsernames {
		usernameFilter[username] = struct{}{}
	}

	membershipsResponse, err := d.client.GetProjectMemberships(infisical.GetProjectMembershipsRequest{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the project user memberships",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	roleObjectType := types.ObjectType{AttrTypes: projectUsersListRoleAttrTypes}

	members := make([]ProjectUsersListMemberModel, 0, len(membershipsResponse.Memberships))
	for _, membership := range membershipsResponse.Memberships {
		// Skip members not in the requested usernames when a filter is set.
		if len(usernameFilter) > 0 {
			if _, ok := usernameFilter[membership.User.Username]; !ok {
				continue
			}
		}

		roles := make([]ProjectUsersListRoleModel, 0, len(membership.Roles))
		for _, el := range membership.Roles {
			role := ProjectUsersListRoleModel{
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
				role.RoleSlug = types.StringValue(el.CustomRoleSlug)
				role.CustomRoleID = types.StringValue(el.CustomRoleId)
			}
			if !el.IsTemporary {
				role.TemporaryMode = types.StringNull()
				role.TemporaryRange = types.StringNull()
				role.TemporaryAccessStartTime = types.StringNull()
				role.TemporaryAccessEndTime = types.StringNull()
			}
			roles = append(roles, role)
		}

		// Sort roles by slug, then id, for deterministic output.
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

		members = append(members, ProjectUsersListMemberModel{
			MembershipID: types.StringValue(membership.ID),
			UserID:       types.StringValue(membership.UserID),
			Username:     types.StringValue(membership.User.Username),
			Email:        types.StringValue(membership.User.Email),
			FirstName:    types.StringValue(membership.User.FirstName),
			LastName:     types.StringValue(membership.User.LastName),
			Roles:        roleList,
		})
	}

	// Sort by username for deterministic output regardless of API ordering.
	sort.SliceStable(members, func(i, j int) bool {
		return members[i].Username.ValueString() < members[j].Username.ValueString()
	})

	memberList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: projectUsersListMemberAttrTypes}, members)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Members = memberList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
