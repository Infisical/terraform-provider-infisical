package datasource

import (
	"context"
	"errors"
	"fmt"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource                     = &GroupAssignmentDataSource{}
	_ datasource.DataSourceWithConfigValidators = &GroupAssignmentDataSource{}
)

func NewGroupAssignmentDataSource() datasource.DataSource {
	return &GroupAssignmentDataSource{}
}

// GroupAssignmentDataSource reads a root-organization group's link into a sub-organization.
type GroupAssignmentDataSource struct {
	client *infisical.Client
}

type GroupAssignmentDataSourceModel struct {
	ID         types.String                    `tfsdk:"id"`
	GroupID    types.String                    `tfsdk:"group_id"`
	GroupSlug  types.String                    `tfsdk:"group_slug"`
	GroupName  types.String                    `tfsdk:"group_name"`
	GroupOrgID types.String                    `tfsdk:"group_org_id"`
	Roles      []GroupAssignmentDataSourceRole `tfsdk:"roles"`
	CreatedAt  types.String                    `tfsdk:"created_at"`
	UpdatedAt  types.String                    `tfsdk:"updated_at"`
}

type GroupAssignmentDataSourceRole struct {
	RoleSlug                 types.String `tfsdk:"role_slug"`
	CustomRoleID             types.String `tfsdk:"custom_role_id"`
	IsTemporary              types.Bool   `tfsdk:"is_temporary"`
	TemporaryMode            types.String `tfsdk:"temporary_mode"`
	TemporaryRange           types.String `tfsdk:"temporary_range"`
	TemporaryAccessStartTime types.String `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime   types.String `tfsdk:"temporary_access_end_time"`
}

func (d *GroupAssignmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_assignment"
}

func (d *GroupAssignmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read how a group is linked into the organization the provider is scoped to, typically a root-organization group linked into a sub-organization, along with the roles it holds there. " +
			"Scope the provider to the sub-organization through `auth.organization_slug`. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Description: "The ID of the linked group. Exactly one of `group_id` or `group_slug` must be set.",
				Optional:    true,
				Computed:    true,
			},
			"group_slug": schema.StringAttribute{
				Description: "The slug of the linked group. Exactly one of `group_id` or `group_slug` must be set.",
				Optional:    true,
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the group's membership in the organization.",
				Computed:    true,
			},
			"group_name": schema.StringAttribute{
				Description: "The name of the linked group.",
				Computed:    true,
			},
			"group_org_id": schema.StringAttribute{
				Description: "The ID of the organization that owns the group (the root organization).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "When the group was linked.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "When the membership was last updated.",
				Computed:    true,
			},
			"roles": schema.ListNestedAttribute{
				Description: "The organization roles the group holds.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_slug": schema.StringAttribute{
							Description: "The slug of the role. For custom roles, the custom role's slug.",
							Computed:    true,
						},
						"custom_role_id": schema.StringAttribute{
							Description: "The ID of the custom role, if the role is a custom one.",
							Computed:    true,
						},
						"is_temporary": schema.BoolAttribute{
							Description: "Whether the role is temporary.",
							Computed:    true,
						},
						"temporary_mode": schema.StringAttribute{
							Description: "The temporary access mode, if the role is temporary.",
							Computed:    true,
						},
						"temporary_range": schema.StringAttribute{
							Description: "The TTL of the temporary role, e.g. 1h.",
							Computed:    true,
						},
						"temporary_access_start_time": schema.StringAttribute{
							Description: "When temporary access begins.",
							Computed:    true,
						},
						"temporary_access_end_time": schema.StringAttribute{
							Description: "When temporary access ends.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *GroupAssignmentDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("group_id"),
			path.MatchRoot("group_slug"),
		),
	}
}

func (d *GroupAssignmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func optionalString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func optionalTime(value *time.Time) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(value.UTC().Format(time.RFC3339))
}

func (d *GroupAssignmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read group assignment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data GroupAssignmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var membership infisical.OrgGroupMembership
	var err error
	groupRef := data.GroupID.ValueString()
	if groupRef != "" {
		membership, err = d.client.GetOrgGroupMembership(groupRef)
	} else {
		groupRef = data.GroupSlug.ValueString()
		membership, err = d.client.GetOrgGroupMembershipBySlug(groupRef)
	}

	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Group assignment not found",
				fmt.Sprintf("Group %s is not linked to the organization the provider is scoped to. Check auth.organization_slug points at the intended sub-organization.", groupRef),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the group assignment",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(membership.ID)
	data.GroupID = types.StringValue(membership.GroupID)
	data.GroupSlug = types.StringValue(membership.Group.Slug)
	data.GroupName = types.StringValue(membership.Group.Name)
	data.GroupOrgID = types.StringValue(membership.Group.OrgID)
	data.CreatedAt = types.StringValue(membership.CreatedAt)
	data.UpdatedAt = types.StringValue(membership.UpdatedAt)

	data.Roles = make([]GroupAssignmentDataSourceRole, 0, len(membership.Roles))
	for _, role := range membership.Roles {
		roleSlug := role.Role
		if role.Role == "custom" && role.CustomRoleSlug != nil && *role.CustomRoleSlug != "" {
			roleSlug = *role.CustomRoleSlug
		}

		data.Roles = append(data.Roles, GroupAssignmentDataSourceRole{
			RoleSlug:                 types.StringValue(roleSlug),
			CustomRoleID:             optionalString(role.CustomRoleID),
			IsTemporary:              types.BoolValue(role.IsTemporary),
			TemporaryMode:            optionalString(role.TemporaryMode),
			TemporaryRange:           optionalString(role.TemporaryRange),
			TemporaryAccessStartTime: optionalTime(role.TemporaryAccessStartTime),
			TemporaryAccessEndTime:   optionalTime(role.TemporaryAccessEndTime),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
