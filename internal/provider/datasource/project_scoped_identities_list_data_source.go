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
var _ datasource.DataSource = &ProjectScopedIdentitiesListDataSource{}

func NewProjectScopedIdentitiesListDataSource() datasource.DataSource {
	return &ProjectScopedIdentitiesListDataSource{}
}

// ProjectScopedIdentitiesListDataSource defines the data source implementation.
type ProjectScopedIdentitiesListDataSource struct {
	client *infisical.Client
}

// ProjectScopedIdentitiesListDataSourceModel describes the data source data model.
type ProjectScopedIdentitiesListDataSourceModel struct {
	ProjectID  types.String `tfsdk:"project_id"`
	Names      types.List   `tfsdk:"names"`
	Identities types.List   `tfsdk:"identities"`
}

// ProjectScopedIdentitiesListItemModel describes a single identity in the list output.
type ProjectScopedIdentitiesListItemModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	OrgID               types.String `tfsdk:"org_id"`
	HasDeleteProtection types.Bool   `tfsdk:"has_delete_protection"`
	AuthMethods         types.List   `tfsdk:"auth_methods"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

// projectScopedIdentitiesListItemAttrTypes is the attribute type map for a single
// identity object. Shared by the schema and types.ListValueFrom to keep the list
// element type in sync.
var projectScopedIdentitiesListItemAttrTypes = map[string]attr.Type{
	"id":                    types.StringType,
	"name":                  types.StringType,
	"org_id":                types.StringType,
	"has_delete_protection": types.BoolType,
	"auth_methods":          types.ListType{ElemType: types.StringType},
	"created_at":            types.StringType,
	"updated_at":            types.StringType,
}

func (d *ProjectScopedIdentitiesListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_scoped_identities_list"
}

func (d *ProjectScopedIdentitiesListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch the list of project-scoped machine identities for a project, optionally filtered by name. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the project to fetch machine identities for.",
				Required:    true,
			},
			"names": schema.ListAttribute{
				Description: "The names of the identities to fetch. If omitted or empty, all project-scoped identities are returned. Names that do not match any identity are ignored.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"identities": schema.ListNestedAttribute{
				Description: "The list of project-scoped machine identities matching the provided names (or all identities when no names are provided).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The UUID of the machine identity.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the machine identity.",
							Computed:    true,
						},
						"org_id": schema.StringAttribute{
							Description: "The UUID of the organization this identity belongs to.",
							Computed:    true,
						},
						"has_delete_protection": schema.BoolAttribute{
							Description: "Whether the identity is protected from deletion.",
							Computed:    true,
						},
						"auth_methods": schema.ListAttribute{
							Description: "The authentication methods configured on this identity.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"created_at": schema.StringAttribute{
							Description: "The creation date of the identity.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The last update date of the identity.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ProjectScopedIdentitiesListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectScopedIdentitiesListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch project-scoped machine identities",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data ProjectScopedIdentitiesListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read the optional names filter. When no names are provided, all identities for the project are returned.
	var requestedNames []string
	if !data.Names.IsNull() && !data.Names.IsUnknown() {
		resp.Diagnostics.Append(data.Names.ElementsAs(ctx, &requestedNames, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	nameFilter := make(map[string]struct{}, len(requestedNames))
	for _, name := range requestedNames {
		nameFilter[name] = struct{}{}
	}

	allIdentities, err := d.client.ListProjectScopedIdentities(infisical.ListProjectScopedIdentitiesRequest{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the project-scoped machine identities",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	items := make([]ProjectScopedIdentitiesListItemModel, 0, len(allIdentities))
	for _, identity := range allIdentities {
		// Skip identities not in the requested names when a filter is set.
		if len(nameFilter) > 0 {
			if _, ok := nameFilter[identity.Name]; !ok {
				continue
			}
		}

		authMethods, diags := types.ListValueFrom(ctx, types.StringType, identity.AuthMethods)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		items = append(items, ProjectScopedIdentitiesListItemModel{
			ID:                  types.StringValue(identity.ID),
			Name:                types.StringValue(identity.Name),
			OrgID:               types.StringValue(identity.OrgID),
			HasDeleteProtection: types.BoolValue(identity.HasDeleteProtection),
			AuthMethods:         authMethods,
			CreatedAt:           types.StringValue(identity.CreatedAt.Format(time.RFC3339Nano)),
			UpdatedAt:           types.StringValue(identity.UpdatedAt.Format(time.RFC3339Nano)),
		})
	}

	// Sort by name for deterministic output regardless of API ordering.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Name.ValueString() < items[j].Name.ValueString()
	})

	identityList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: projectScopedIdentitiesListItemAttrTypes}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Identities = identityList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
