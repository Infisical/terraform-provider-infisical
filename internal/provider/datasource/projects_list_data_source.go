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
var _ datasource.DataSource = &ProjectsListDataSource{}

func NewProjectsListDataSource() datasource.DataSource {
	return &ProjectsListDataSource{}
}

// ProjectsListDataSource defines the data source implementation.
type ProjectsListDataSource struct {
	client *infisical.Client
}

// ProjectsListDataSourceModel describes the data source data model.
type ProjectsListDataSourceModel struct {
	Slugs    types.List `tfsdk:"slugs"`
	Projects types.List `tfsdk:"projects"`
}

// ProjectsListItemModel describes a single project in the list output.
type ProjectsListItemModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Slug               types.String `tfsdk:"slug"`
	Description        types.String `tfsdk:"description"`
	Type               types.String `tfsdk:"type"`
	AutoCapitalization types.Bool   `tfsdk:"auto_capitalization"`
	OrgID              types.String `tfsdk:"org_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	Version            types.Int64  `tfsdk:"version"`
	UpgradeStatus      types.String `tfsdk:"upgrade_status"`
	Environments       types.List   `tfsdk:"environments"`
}

// ProjectsListEnvironmentModel describes a single environment of a project.
type ProjectsListEnvironmentModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Slug types.String `tfsdk:"slug"`
}

// projectsListEnvironmentAttrTypes is the attribute type map for a single
// environment object. Shared by the schema and types.ListValueFrom to keep the
// list element type in sync.
var projectsListEnvironmentAttrTypes = map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
	"slug": types.StringType,
}

// projectsListProjectAttrTypes is the attribute type map for a single project
// object. Shared by the schema and types.ListValueFrom to keep the list element
// type in sync.
var projectsListProjectAttrTypes = map[string]attr.Type{
	"id":                  types.StringType,
	"name":                types.StringType,
	"slug":                types.StringType,
	"description":         types.StringType,
	"type":                types.StringType,
	"auto_capitalization": types.BoolType,
	"org_id":              types.StringType,
	"created_at":          types.StringType,
	"updated_at":          types.StringType,
	"version":             types.Int64Type,
	"upgrade_status":      types.StringType,
	"environments":        types.ListType{ElemType: types.ObjectType{AttrTypes: projectsListEnvironmentAttrTypes}},
}

func (d *ProjectsListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects_list"
}

func (d *ProjectsListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch the list of Infisical projects that the machine identity has access to, optionally filtered by project slugs. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"slugs": schema.ListAttribute{
				Description: "The slugs of the projects to fetch. If omitted or empty, all accessible projects are returned. Slugs that do not match any accessible project are ignored.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"projects": schema.ListNestedAttribute{
				Description: "The list of projects matching the provided slugs (or all accessible projects when no slugs are provided).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the project",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the project",
							Computed:    true,
						},
						"slug": schema.StringAttribute{
							Description: "The slug of the project",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the project",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the project ('secret-manager' or 'kms')",
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
						"environments": schema.ListNestedAttribute{
							Description: "The environments of the project",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Description: "The ID of the environment",
										Computed:    true,
									},
									"name": schema.StringAttribute{
										Description: "The name of the environment",
										Computed:    true,
									},
									"slug": schema.StringAttribute{
										Description: "The slug of the environment",
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

func (d *ProjectsListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectsListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch projects",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data ProjectsListDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read the optional slugs filter. When no slugs are provided, all accessible
	// projects are returned.
	var requestedSlugs []string
	if !data.Slugs.IsNull() && !data.Slugs.IsUnknown() {
		resp.Diagnostics.Append(data.Slugs.ElementsAs(ctx, &requestedSlugs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	slugFilter := make(map[string]struct{}, len(requestedSlugs))
	for _, slug := range requestedSlugs {
		slugFilter[slug] = struct{}{}
	}

	projectsResponse, err := d.client.GetProjects()
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the projects",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	environmentObjectType := types.ObjectType{AttrTypes: projectsListEnvironmentAttrTypes}

	items := make([]ProjectsListItemModel, 0, len(projectsResponse.Projects))
	for _, project := range projectsResponse.Projects {
		// Skip projects that are not in the requested slugs when a filter is set.
		if len(slugFilter) > 0 {
			if _, ok := slugFilter[project.Slug]; !ok {
				continue
			}
		}

		environments := make([]ProjectsListEnvironmentModel, 0, len(project.Environments))
		for _, env := range project.Environments {
			environments = append(environments, ProjectsListEnvironmentModel{
				ID:   types.StringValue(env.ID),
				Name: types.StringValue(env.Name),
				Slug: types.StringValue(env.Slug),
			})
		}

		environmentList, diags := types.ListValueFrom(ctx, environmentObjectType, environments)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		items = append(items, ProjectsListItemModel{
			ID:                 types.StringValue(project.ID),
			Name:               types.StringValue(project.Name),
			Slug:               types.StringValue(project.Slug),
			Description:        types.StringValue(project.Description),
			Type:               types.StringValue(project.Type),
			AutoCapitalization: types.BoolValue(project.AutoCapitalization),
			OrgID:              types.StringValue(project.OrgID),
			CreatedAt:          types.StringValue(project.CreatedAt.Format(time.RFC3339Nano)),
			UpdatedAt:          types.StringValue(project.UpdatedAt.Format(time.RFC3339Nano)),
			Version:            types.Int64Value(project.Version),
			UpgradeStatus:      types.StringValue(project.UpgradeStatus),
			Environments:       environmentList,
		})
	}

	// Sort by slug for deterministic output regardless of API ordering.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Slug.ValueString() < items[j].Slug.ValueString()
	})

	projectList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: projectsListProjectAttrTypes}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Projects = projectList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
