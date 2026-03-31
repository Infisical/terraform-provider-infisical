package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &ProjectLevelIdentityResource{}
	_ resource.ResourceWithImportState = &ProjectLevelIdentityResource{}
)

// NewProjectLevelIdentityResource is a helper function to simplify the provider implementation.
func NewProjectLevelIdentityResource() resource.Resource {
	return &ProjectLevelIdentityResource{}
}

// ProjectLevelIdentityResource is the resource implementation.
type ProjectLevelIdentityResource struct {
	client *infisical.Client
}

// ProjectLevelIdentityResourceModel describes the resource data model.
type ProjectLevelIdentityResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	Name                types.String `tfsdk:"name"`
	HasDeleteProtection types.Bool   `tfsdk:"has_delete_protection"`
	AuthMethods         types.List   `tfsdk:"auth_methods"`
	Metadata            []MetaEntry  `tfsdk:"metadata"`
}

// Metadata returns the resource type name.
func (r *ProjectLevelIdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_identity_provisioning"
}

// Schema defines the schema for the resource.
func (r *ProjectLevelIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage project-level machine identities in Infisical. Project-level identities are scoped to a single project and cannot be assigned to other projects. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the project-level identity.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the project that owns this identity.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the identity.",
				Required:    true,
			},
			"has_delete_protection": schema.BoolAttribute{
				Description: "Whether the identity has delete protection enabled. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"auth_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "The authentication methods configured for the identity.",
				Computed:    true,
			},
			"metadata": schema.SetNestedAttribute{
				Description: "The metadata associated with this identity.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The key of the metadata entry.",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the metadata entry.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ProjectLevelIdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *ProjectLevelIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project-level identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan ProjectLevelIdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	metadata := buildMetadataFromPlan(plan.Metadata)

	identity, err := r.client.CreateProjectLevelIdentity(infisical.CreateProjectLevelIdentityRequest{
		ProjectID:           plan.ProjectID.ValueString(),
		Name:                plan.Name.ValueString(),
		HasDeleteProtection: plan.HasDeleteProtection.ValueBool(),
		Metadata:            metadata,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project-level identity",
			"Couldn't create project-level identity in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(identity.ID)
	plan.HasDeleteProtection = types.BoolValue(identity.HasDeleteProtection)
	plan.AuthMethods = buildAuthMethodsList(identity.AuthMethods)
	if plan.Metadata != nil {
		setMetadataFromAPI(&plan, identity.Metadata)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ProjectLevelIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project-level identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state ProjectLevelIdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" || state.ProjectID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	identity, err := r.client.GetProjectLevelIdentityByID(infisical.GetProjectLevelIdentityRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading project-level identity",
			"Couldn't read project-level identity from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(identity.Name)
	state.HasDeleteProtection = types.BoolValue(identity.HasDeleteProtection)
	state.AuthMethods = buildAuthMethodsList(identity.AuthMethods)
	if state.Metadata != nil {
		setMetadataFromAPI(&state, identity.Metadata)
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ProjectLevelIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project-level identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan ProjectLevelIdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectLevelIdentityResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	metadata := buildMetadataFromPlan(plan.Metadata)

	identity, err := r.client.UpdateProjectLevelIdentity(infisical.UpdateProjectLevelIdentityRequest{
		ProjectID:           state.ProjectID.ValueString(),
		IdentityID:          state.ID.ValueString(),
		Name:                plan.Name.ValueString(),
		HasDeleteProtection: plan.HasDeleteProtection.ValueBool(),
		Metadata:            metadata,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project-level identity",
			"Couldn't update project-level identity in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID
	plan.HasDeleteProtection = types.BoolValue(identity.HasDeleteProtection)
	plan.AuthMethods = buildAuthMethodsList(identity.AuthMethods)
	if plan.Metadata != nil {
		setMetadataFromAPI(&plan, identity.Metadata)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ProjectLevelIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project-level identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state ProjectLevelIdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectLevelIdentity(infisical.DeleteProjectLevelIdentityRequest{
		ProjectID:  state.ProjectID.ValueString(),
		IdentityID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting project-level identity",
			"Couldn't delete project-level identity from Infisical, unexpected error: "+err.Error(),
		)
	}
}

// ImportState restores state from a <project_id>,<identity_id> import ID.
func (r *ProjectLevelIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import project-level identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format <project_id>,<identity_id>",
		)
		return
	}

	projectID := parts[0]
	identityID := parts[1]

	identity, err := r.client.GetProjectLevelIdentityByID(infisical.GetProjectLevelIdentityRequest{
		ProjectID:  projectID,
		IdentityID: identityID,
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Error importing project-level identity",
				fmt.Sprintf("No project-level identity found with project_id=%s and id=%s", projectID, identityID),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error importing project-level identity",
			"Couldn't read project-level identity from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	var metadata []MetaEntry
	for _, m := range identity.Metadata {
		metadata = append(metadata, MetaEntry{
			Key:   types.StringValue(m.Key),
			Value: types.StringValue(m.Value),
		})
	}

	state := ProjectLevelIdentityResourceModel{
		ID:                  types.StringValue(identity.ID),
		ProjectID:           types.StringValue(identity.ProjectID),
		Name:                types.StringValue(identity.Name),
		HasDeleteProtection: types.BoolValue(identity.HasDeleteProtection),
		AuthMethods:         buildAuthMethodsList(identity.AuthMethods),
		Metadata:            metadata,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// buildMetadataFromPlan converts the plan metadata slice to CreateMetaEntry slice.
func buildMetadataFromPlan(entries []MetaEntry) []infisical.CreateMetaEntry {
	result := []infisical.CreateMetaEntry{}
	for _, e := range entries {
		result = append(result, infisical.CreateMetaEntry{
			Key:   e.Key.ValueString(),
			Value: e.Value.ValueString(),
		})
	}
	return result
}

// buildAuthMethodsList converts a string slice to a Terraform list value.
func buildAuthMethodsList(methods []string) types.List {
	if len(methods) == 0 {
		return types.ListNull(types.StringType)
	}
	elements := make([]attr.Value, len(methods))
	for i, m := range methods {
		elements[i] = types.StringValue(m)
	}
	return types.ListValueMust(types.StringType, elements)
}

// setMetadataFromAPI updates the model's metadata from the API response.
func setMetadataFromAPI(model *ProjectLevelIdentityResourceModel, apiMetadata []infisical.MetaEntry) {
	if len(apiMetadata) > 0 {
		converted := make([]MetaEntry, len(apiMetadata))
		for i, m := range apiMetadata {
			converted[i] = MetaEntry{
				Key:   types.StringValue(m.Key),
				Value: types.StringValue(m.Value),
			}
		}
		model.Metadata = converted
	} else {
		model.Metadata = []MetaEntry{}
	}
}
