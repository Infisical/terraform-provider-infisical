package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &IntegrationGCPSecretManagerResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewIntegrationGcpSecretManagerResource() resource.Resource {
	return &IntegrationGCPSecretManagerResource{}
}

// IntegrationGcpSecretManager is the resource implementation.
type IntegrationGCPSecretManagerResource struct {
	client *infisical.Client
}

// projectResourceSourceModel describes the data source data model.
type IntegrationGCPSecretManagerResourceModel struct {
	EnvironmentID      types.String `tfsdk:"env_id"`
	IntegrationAuthID  types.String `tfsdk:"integration_auth_id"`
	IntegrationID      types.String `tfsdk:"integration_id"`
	ServiceAccountJson types.String `tfsdk:"service_account_json"`
	ProjectID          types.String `tfsdk:"project_id"`
	Environment        types.String `tfsdk:"environment"`
	SecretPath         types.String `tfsdk:"secret_path"`
	GCPProjectID       types.String `tfsdk:"gcp_project_id"`

	Options types.Object `tfsdk:"options"`
}

type GcpIntegrationMetadataStruct struct {
	SecretPrefix string `json:"secretPrefix,omitempty"`
	SecretSuffix string `json:"secretSuffix,omitempty"`
}

// Metadata returns the resource type name.
func (r *IntegrationGCPSecretManagerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_gcp_secret_manager"
}

// Schema defines the schema for the resource.
func (r *IntegrationGCPSecretManagerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create GCP Secret Manager integration & save to Infisical. Only Machine Identity authentication is supported for this data source",
		Attributes: map[string]schema.Attribute{
			"options": schema.SingleNestedAttribute{
				Description: "Integration options",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"secret_prefix": schema.StringAttribute{
						Description: "The prefix to add to the secret name in GCP Secret Manager.",
						Optional:    true,
					},
					"secret_suffix": schema.StringAttribute{
						Description: "The suffix to add to the secret name in GCP Secret Manager.",
						Optional:    true,
					},
				},
			},

			"integration_auth_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the integration auth, used internally by Infisical.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"env_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the environment, used internally by Infisical.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"integration_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the integration, used internally by Infisical.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"service_account_json": schema.StringAttribute{
				Sensitive:   true,
				Required:    true,
				Description: "Service account json for the GCP project.",
			},

			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of your Infisical project.",
			},

			"environment": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to sync to GCP Secret Manager (prod, dev, staging, etc).",
			},

			"secret_path": schema.StringAttribute{
				Required:    true,
				Description: "The secret path in Infisical to sync secrets from.",
			},

			"gcp_project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the GCP project.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IntegrationGCPSecretManagerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *IntegrationGCPSecretManagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationGCPSecretManagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create integration auth first
	auth, err := r.client.CreateIntegrationAuth(infisical.CreateIntegrationAuthRequest{
		RefreshToken: plan.ServiceAccountJson.ValueString(),
		ProjectID:    plan.ProjectID.ValueString(),
		Integration:  infisical.IntegrationAuthTypeGcpSecretManager,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create integration auth",
			err.Error(),
		)
		return
	}

	metadata := GcpIntegrationMetadataStruct{}

	if !plan.Options.IsNull() && !plan.Options.IsUnknown() {
		var options struct {
			SecretPrefix types.String `tfsdk:"secret_prefix"`
			SecretSuffix types.String `tfsdk:"secret_suffix"`
		}
		diags := plan.Options.As(ctx, &options, basetypes.ObjectAsOptions{})

		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !options.SecretPrefix.IsNull() && !options.SecretPrefix.IsUnknown() {
			metadata.SecretPrefix = options.SecretPrefix.ValueString()
		}
		if !options.SecretSuffix.IsNull() && !options.SecretSuffix.IsUnknown() {
			metadata.SecretSuffix = options.SecretSuffix.ValueString()
		}
	}

	metadataMap := map[string]interface{}{
		"secretPrefix": metadata.SecretPrefix,
		"secretSuffix": metadata.SecretSuffix,
	}

	// Create the integration
	integration, err := r.client.CreateIntegration(infisical.CreateIntegrationRequest{
		IntegrationAuthID: auth.IntegrationAuth.ID,
		App:               plan.GCPProjectID.ValueString(),
		AppID:             plan.GCPProjectID.ValueString(),
		SecretPath:        plan.SecretPath.ValueString(),
		SourceEnvironment: plan.Environment.ValueString(),
		Metadata:          metadataMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			err.Error(),
		)
		return
	}

	// Set the state

	plan.IntegrationAuthID = types.StringValue(auth.IntegrationAuth.ID)
	plan.IntegrationID = types.StringValue(integration.Integration.ID)
	plan.EnvironmentID = types.StringValue(integration.Integration.EnvID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *IntegrationGCPSecretManagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IntegrationGCPSecretManagerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration, err := r.client.GetIntegration(infisical.GetIntegrationRequest{
		ID: state.IntegrationID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Unable to get integration",
				err.Error(),
			)
		}
		return
	}

	var planOptions struct {
		SecretPrefix types.String `tfsdk:"secret_prefix"`
		SecretSuffix types.String `tfsdk:"secret_suffix"`
	}

	if !state.Options.IsNull() {
		diags := state.Options.As(ctx, &planOptions, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateNeeded := false

	if integration.Integration.Metadata.SecretPrefix != planOptions.SecretPrefix.ValueString() {
		planOptions.SecretPrefix = types.StringValue(integration.Integration.Metadata.SecretPrefix)
		updateNeeded = true
	}

	if planOptions.SecretSuffix.ValueString() != integration.Integration.Metadata.SecretSuffix {
		planOptions.SecretSuffix = types.StringValue(integration.Integration.Metadata.SecretSuffix)
		updateNeeded = true
	}

	if updateNeeded {
		// Create a map of the updated options
		optionsMap := map[string]attr.Value{
			"secret_prefix": planOptions.SecretPrefix,
			"secret_suffix": planOptions.SecretSuffix,
		}

		// Create a new types.Object with the updated options
		newOptions, diags := types.ObjectValue(
			map[string]attr.Type{
				"secret_prefix": types.StringType,
				"secret_suffix": types.StringType,
			},
			optionsMap,
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Set the new options in the state
		state.Options = newOptions
	}

	// Set the state.Options.
	state.GCPProjectID = types.StringValue(integration.Integration.AppID)
	state.SecretPath = types.StringValue(integration.Integration.SecretPath)
	state.EnvironmentID = types.StringValue(integration.Integration.EnvID)
	state.IntegrationAuthID = types.StringValue(integration.Integration.IntegrationAuthID)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IntegrationGCPSecretManagerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationGCPSecretManagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IntegrationGCPSecretManagerResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ProjectID != state.ProjectID {
		resp.Diagnostics.AddError(
			"Project ID cannot be updated",
			"Project ID cannot be updated",
		)
		return
	}

	var options struct {
		SecretPrefix types.String `tfsdk:"secret_prefix"`
		SecretSuffix types.String `tfsdk:"secret_suffix"`
	}
	plan.Options.As(ctx, &options, basetypes.ObjectAsOptions{})

	metadataMap := map[string]interface{}{
		"secretPrefix": options.SecretPrefix.ValueString(),
		"secretSuffix": options.SecretSuffix.ValueString(),
	}

	updatedIntegration, err := r.client.UpdateIntegration(infisical.UpdateIntegrationRequest{
		IsActive:    true,
		ID:          state.IntegrationID.ValueString(),
		Environment: plan.Environment.ValueString(),
		App:         plan.GCPProjectID.ValueString(),
		AppID:       plan.GCPProjectID.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
		Metadata:    metadataMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating integration",
			err.Error(),
		)
		return
	}

	plan.EnvironmentID = types.StringValue(updatedIntegration.Integration.EnvID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IntegrationGCPSecretManagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete GCP Secret Manager integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IntegrationGCPSecretManagerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteIntegrationAuth(infisical.DeleteIntegrationAuthRequest{
		ID: state.IntegrationAuthID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting GCP Secret Manager Integration",
			"Couldn't delete GCP Secret Manager integration from your Infiscial project, unexpected error: "+err.Error(),
		)
		return
	}

}
