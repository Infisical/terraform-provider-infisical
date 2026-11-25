package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	PROJECT_TYPE_SECRET_MANAGER = "secret-manager"
	PROJECT_TYPE_CERT_MANAGER   = "cert-manager"
	PROJECT_TYPE_KMS            = "kms"
	SUPPORTED_PROJECT_TYPES     = []string{PROJECT_TYPE_SECRET_MANAGER, PROJECT_TYPE_CERT_MANAGER, PROJECT_TYPE_KMS}
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &projectResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *infisical.Client
}

// projectResourceSourceModel describes the data source data model.
type projectResourceModel struct {
	Slug                    types.String `tfsdk:"slug"`
	ID                      types.String `tfsdk:"id"`
	Type                    types.String `tfsdk:"type"`
	KmsSecretManagerKeyId   types.String `tfsdk:"kms_secret_manager_key_id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	DescriptionWO           types.String `tfsdk:"description_wo"`
	DescriptionWOVersion    types.Int64  `tfsdk:"description_wo_version"`
	LastUpdated             types.String `tfsdk:"last_updated"`
	TemplateName            types.String `tfsdk:"template_name"`
	ShouldCreateDefaultEnvs types.Bool   `tfsdk:"should_create_default_envs"`
	HasDeleteProtection     types.Bool   `tfsdk:"has_delete_protection"`
	AuditLogRetentionDays   types.Int64  `tfsdk:"audit_log_retention_days"`
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create projects & save to Infisical. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug of the project",
				Required:    true,
				Validators: []validator.String{
					infisicaltf.SlugRegexValidator,
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the project",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the project. Supported values: " + strings.Join(SUPPORTED_PROJECT_TYPES, ", ") + ". Defaults to '" + PROJECT_TYPE_SECRET_MANAGER + "'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_PROJECT_TYPES...),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the project. To keep it out of the plan output and state, use `description_wo` instead.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("description_wo"),
						path.MatchRoot("description_wo_version"),
					),
				},
			},
			"description_wo": schema.StringAttribute{
				Description: "The description of the project as a write-only value: it is sent to Infisical but never stored in the plan or state. It is sent on every update, so it must stay set for as long as the description should be kept. Changing it alone is not detected, so increment `description_wo_version` to update it. Requires Terraform 1.11.0 or higher.",
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("description_wo_version")),
				},
			},
			"description_wo_version": schema.Int64Attribute{
				Description: "Used with `description_wo` to update the description. Increment it whenever `description_wo` changes, and the next apply sends the new description.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AlsoRequires(path.MatchRoot("description_wo")),
				},
			},
			"template_name": schema.StringAttribute{
				Description: "The name of the template to use for the project",
				Optional:    true,
			},
			"kms_secret_manager_key_id": schema.StringAttribute{
				Description: "The ID of the KMS secret manager key to use for the project",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"should_create_default_envs": schema.BoolAttribute{
				Description: "Whether to create default environments for the project (dev, staging, prod), defaults to true",
				Optional:    true,
			},
			"has_delete_protection": schema.BoolAttribute{
				Description: "Whether the project has delete protection, defaults to false",
				Optional:    true,
				Computed:    true,
			},
			"audit_log_retention_days": schema.Int64Attribute{
				Description: "The audit log retention in days",
				Optional:    true,
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the project",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
				// PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// configuredDescription returns description, or description_wo from the config
// since write-only values never make it into the plan.
func configuredDescription(ctx context.Context, diagnostics *diag.Diagnostics, config tfsdk.Config, plan projectResourceModel) string {
	if !plan.Description.IsNull() {
		return plan.Description.ValueString()
	}

	var writeOnlyDescription types.String
	diagnostics.Append(config.GetAttribute(ctx, path.Root("description_wo"), &writeOnlyDescription)...)
	return writeOnlyDescription.ValueString()
}

// setDescription stores the description returned by Infisical, unless description_wo
// manages it, in which case the description stays out of state.
func (m *projectResourceModel) setDescription(description string) {
	if !m.DescriptionWOVersion.IsNull() {
		return
	}

	if description == "" {
		m.Description = types.StringNull()
	} else {
		m.Description = types.StringValue(description)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	shouldCreateDefaultEnvs := true
	if !plan.ShouldCreateDefaultEnvs.IsNull() {
		shouldCreateDefaultEnvs = plan.ShouldCreateDefaultEnvs.ValueBool()
	}

	projectType := PROJECT_TYPE_SECRET_MANAGER // default type
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		projectType = plan.Type.ValueString()
	}

	description := configuredDescription(ctx, &resp.Diagnostics, req.Config, plan)
	if resp.Diagnostics.HasError() {
		return
	}

	newProject, err := r.client.CreateProject(infisical.CreateProjectRequest{
		ProjectName:             plan.Name.ValueString(),
		ProjectDescription:      description,
		Slug:                    plan.Slug.ValueString(),
		Type:                    projectType,
		Template:                plan.TemplateName.ValueString(),
		KmsSecretManagerKeyId:   plan.KmsSecretManagerKeyId.ValueString(),
		ShouldCreateDefaultEnvs: shouldCreateDefaultEnvs,
		HasDeleteProtection:     plan.HasDeleteProtection.ValueBool(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Couldn't save project to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	if !plan.AuditLogRetentionDays.IsNull() && !plan.AuditLogRetentionDays.IsUnknown() {
		_, err := r.client.UpdateProjectAuditLogRetention(infisical.UpdateProjectAuditLogRetentionRequest{
			ProjectSlug: plan.Slug.ValueString(),
			Days:        plan.AuditLogRetentionDays.ValueInt64(),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project audit log retention",
				"Couldn't update project audit log retention from Infisical, unexpected error: "+err.Error(),
			)
		}
	}

	project, err := r.client.GetProjectById(infisical.GetProjectByIdRequest{
		ID: newProject.Project.ID,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infisical, unexpected error: "+err.Error(),
		)
	}

	plan.LastUpdated = types.StringValue(newProject.Project.UpdatedAt.Format(time.RFC850))
	plan.ID = types.StringValue(newProject.Project.ID)
	plan.Type = types.StringValue(projectType)
	plan.KmsSecretManagerKeyId = types.StringValue(project.KmsSecretManagerKeyId)
	plan.HasDeleteProtection = types.BoolValue(project.HasDeleteProtection)
	plan.AuditLogRetentionDays = types.Int64Value(project.AuditLogRetentionDays)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	// Get the latest data from the API
	project, err := r.client.GetProjectById(infisical.GetProjectByIdRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.setDescription(project.Description)
	state.ID = types.StringValue(project.ID)
	state.Name = types.StringValue(project.Name)
	state.Type = types.StringValue(project.Type)
	state.LastUpdated = types.StringValue(project.UpdatedAt.Format(time.RFC850))
	state.Slug = types.StringValue(project.Slug)
	state.KmsSecretManagerKeyId = types.StringValue(project.KmsSecretManagerKeyId)
	state.HasDeleteProtection = types.BoolValue(project.HasDeleteProtection)
	state.AuditLogRetentionDays = types.Int64Value(project.AuditLogRetentionDays)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.TemplateName != plan.TemplateName {
		resp.Diagnostics.AddError(
			"Unable to update project",
			"Template name cannot be updated",
		)
		return
	}

	if state.ShouldCreateDefaultEnvs != plan.ShouldCreateDefaultEnvs {
		resp.Diagnostics.AddError(
			"Unable to update project",
			"should_create_default_envs cannot be updated after the resource is created",
		)
		return
	}

	if state.Type != plan.Type {
		resp.Diagnostics.AddError(
			"Unable to update project",
			"Project type cannot be updated after creation",
		)
		return
	}

	if state.KmsSecretManagerKeyId != plan.KmsSecretManagerKeyId {
		resp.Diagnostics.AddError(
			"Unable to update project",
			"KMS secret manager key ID cannot be updated",
		)
		return
	}

	description := configuredDescription(ctx, &resp.Diagnostics, req.Config, plan)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateProject(infisical.UpdateProjectRequest{
		ProjectName:         plan.Name.ValueString(),
		ProjectDescription:  description,
		ProjectId:           plan.ID.ValueString(),
		ProjectSlug:         plan.Slug.ValueString(),
		HasDeleteProtection: plan.HasDeleteProtection.ValueBool(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			"Couldn't update project from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	if state.AuditLogRetentionDays != plan.AuditLogRetentionDays && !plan.AuditLogRetentionDays.IsNull() && !plan.AuditLogRetentionDays.IsUnknown() {
		_, err := r.client.UpdateProjectAuditLogRetention(infisical.UpdateProjectAuditLogRetentionRequest{
			ProjectSlug: plan.Slug.ValueString(),
			Days:        plan.AuditLogRetentionDays.ValueInt64(),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project audit log retention",
				"Couldn't update project audit log retention from Infisical, unexpected error: "+err.Error(),
			)
		}
	}

	project, err := r.client.GetProjectById(infisical.GetProjectByIdRequest{
		ID: plan.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.setDescription(project.Description)
	plan.LastUpdated = types.StringValue(project.UpdatedAt.Format(time.RFC850))
	plan.Name = types.StringValue(project.Name)
	plan.HasDeleteProtection = types.BoolValue(project.HasDeleteProtection)
	plan.AuditLogRetentionDays = types.Int64Value(project.AuditLogRetentionDays)
	plan.Slug = types.StringValue(project.Slug)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProject(infisical.DeleteProjectRequest{
		Slug: state.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			"Couldn't delete project from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	project, err := r.client.GetProjectById(infisical.GetProjectByIdRequest{
		ID: req.ID,
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Project not found",
				"The project with the given ID was not found",
			)
		} else {
			resp.Diagnostics.AddError(
				"Error fetching project",
				"Couldn't fetch project from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), project.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), project.Slug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), project.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), project.Type)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), project.Description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("last_updated"), project.UpdatedAt.Format(time.RFC850))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("has_delete_protection"), project.HasDeleteProtection)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("audit_log_retention_days"), project.AuditLogRetentionDays)...)
}
