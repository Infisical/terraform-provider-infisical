package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	infisical "terraform-provider-infisical/client"
	"time"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &secretImportsResource{}
)

// NewsecretResource is a helper function to simplify the provider implementation.
func NewSecretImportsResource() resource.Resource {
	return &secretImportsResource{}
}

// secretImportsResource is the resource implementation.
type secretImportsResource struct {
	client *infisical.Client
}

// secretResourceSourceModel describes the data source data model.
type secretImportsResourceModel struct {
	SecretImportsId types.String                     `tfsdk:"secret_imports_id"`
	FolderPath      types.String                     `tfsdk:"folder_path"`
	EnvSlug         types.String                     `tfsdk:"env_slug"`
	WorkspaceId     types.String                     `tfsdk:"workspace_id"`
	SecretImports   []secretImportsSecretImportModel `tfsdk:"import"`
	LastUpdated     types.String                     `tfsdk:"last_updated"`
}

type secretImportsSecretImportModel struct {
	EnvSlug    types.String `tfsdk:"env_slug"`
	FolderPath types.String `tfsdk:"folder_path"`
}

// Metadata returns the resource type name.
func (r *secretImportsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_imports"
}

// Schema defines the schema for the resource.
func (r *secretImportsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create secrets & save to Infisical",
		Attributes: map[string]schema.Attribute{
			"folder_path": schema.StringAttribute{
				Description: "The path to the folder where the given secret-imports resides",
				Required:    true,
				Computed:    false,
			},
			"env_slug": schema.StringAttribute{
				Description: "The environment slug of the secret-imports to modify/create",
				Required:    true,
				Computed:    false,
			},
			"secret_imports_id": schema.StringAttribute{
				Description: "The id of the managed secret-imports",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "The Infisical project ID",
				Optional:    true,
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			"import_secrets": schema.ListNestedBlock{
				Description: "Secret(s) to imports",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"env_slug": schema.StringAttribute{
							Description: "Slug of environment to import from",
							Required:    true,
							Computed:    false,
						},
						"folder_path": schema.StringAttribute{
							Description: "Path where to import from like / or /foo/bar",
							Required:    true,
							Computed:    false,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *secretImportsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *secretImportsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secretImportsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceTokenDetails, err := r.client.CallGetServiceTokenDetailsV2()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret",
			"Could not get service token details, unexpected error: "+err.Error(),
		)
		return
	}

	for _, item := range plan.SecretImports {
		payload := infisical.CreateSecretImportsV1Request{
			Environment: plan.EnvSlug.ValueString(),
			Directory:   plan.FolderPath.ValueString(),
			WorkspaceID: serviceTokenDetails.Workspace,
		}
		payload.SecretImport.Environment = item.EnvSlug.ValueString()
		payload.SecretImport.SecretPath = item.FolderPath.ValueString()

		err = r.client.CallCreateSecretImportsV1(payload)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret imports",
				"Couldn't create secret-imports, unexpected error: "+err.Error(),
			)
			return
		}
	}

	secretImports, err := r.client.CallGetSecretImportsByDirectoryV1(infisical.GetSecretImportsByDirectoryV1Request{
		Environment: plan.EnvSlug.ValueString(),
		Directory:   plan.FolderPath.ValueString(),
		WorkspaceId: serviceTokenDetails.Workspace,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret imports",
			"Couldn't list existing secrets after creation: "+err.Error(),
		)
		return
	}

	// Set state to fully populated data
	plan.SecretImportsId = types.StringValue(secretImports.SecretImport.Id)
	plan.WorkspaceId = types.StringValue(serviceTokenDetails.Workspace)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *secretImportsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *secretImportsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan secretImportsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state secretImportsResourceModel
	diagsFromState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsFromState...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.SecretImportsId != plan.SecretImportsId {
		resp.Diagnostics.AddError(
			"Unable to update secret-imports",
			"Secret imports cannot be updated via Terraform at this time",
		)
		return
	}

	for _, item := range plan.SecretImports {
		payload := infisical.UpdateSecretImportsV1Request{
			SecretId: plan.SecretImportsId.ValueString(),
		}
		payload.SecretImport.Environment = item.EnvSlug.ValueString()
		payload.SecretImport.SecretPath = item.FolderPath.ValueString()

		err := r.client.CallUpdateSecretImportsV1(payload)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret imports",
				"Couldn't create secret-imports, unexpected error: "+err.Error(),
			)
			return
		}
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *secretImportsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secretImportsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.CallDeleteSecretImportsV1(infisical.DeleteSecretImportsV1Request{
		SecretId:         state.SecretImportsId.ValueString(),
		SecretImportEnv:  state.EnvSlug.ValueString(),
		SecretImportPath: state.FolderPath.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting infisical secret-imports",
			"Could not delete secret-imports, unexpected error:"+err.Error(),
		)
		return
	}
}
