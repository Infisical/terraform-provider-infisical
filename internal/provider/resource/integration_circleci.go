package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &IntegrationCircleCIResource{}
)

// NewIntegrationCircleCiResource is a helper function to simplify the provider implementation.
func NewIntegrationCircleCiResource() resource.Resource {
	return &IntegrationCircleCIResource{}
}

// IntegrationCircleCI is the resource implementation.
type IntegrationCircleCIResource struct {
	client *infisical.Client
}

// projectResourceSourceModel describes the data source data model.
type IntegrationCircleCIResourceModel struct {
	CircleCIToken types.String `tfsdk:"circleci_token"`
	ProjectID     types.String `tfsdk:"project_id"`

	IntegrationAuthID types.String `tfsdk:"integration_auth_id"`
	IntegrationID     types.String `tfsdk:"integration_id"`

	Environment types.String `tfsdk:"environment"`
	SecretPath  types.String `tfsdk:"secret_path"`

	CircleCIOrgSlug   types.String `tfsdk:"circleci_org_slug"`
	CircleCIProjectID types.String `tfsdk:"circleci_project_id"`
}

// Metadata returns the resource type name.
func (r *IntegrationCircleCIResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_circleci"
}

// Schema defines the schema for the resource.
func (r *IntegrationCircleCIResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create CircleCI integration & save to Infisical. Only Machine Identity authentication is supported for this data source",
		Attributes: map[string]schema.Attribute{
			"integration_auth_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the integration auth, used internally by Infisical.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"integration_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the integration, used internally by Infisical.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"circleci_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Your personal CircleCI token to authenticate with.",
			},

			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of your Infisical project.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"environment": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to sync to CircleCI (prod, dev, staging, etc).",
			},

			"secret_path": schema.StringAttribute{
				Required:    true,
				Description: "The secret path in Infisical to sync secrets from.",
			},

			"circleci_org_slug": schema.StringAttribute{
				Required:    true,
				Description: "The organization slug of your CircleCI organization.",
			},

			"circleci_project_id": schema.StringAttribute{
				Required:    true,
				Description: "The project ID of your CircleCI project.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IntegrationCircleCIResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *IntegrationCircleCIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationCircleCIResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create integration auth first
	auth, err := r.client.CreateIntegrationAuth(infisical.CreateIntegrationAuthRequest{
		AccessToken: plan.CircleCIToken.ValueString(),
		ProjectID:   plan.ProjectID.ValueString(),
		Integration: infisical.IntegrationAuthTypeCircleCi,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create integration auth",
			err.Error(),
		)
		return
	}

	// Create the integration
	integration, err := r.client.CreateIntegration(infisical.CreateIntegrationRequest{
		IntegrationAuthID: auth.IntegrationAuth.ID,
		App:               plan.CircleCIProjectID.ValueString(), // Needs to be the project slug
		AppID:             plan.CircleCIProjectID.ValueString(), // Needs to be the project ID
		Owner:             plan.CircleCIOrgSlug.ValueString(),   // Needs to be the organization slug
		SecretPath:        plan.SecretPath.ValueString(),
		SourceEnvironment: plan.Environment.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			err.Error(),
		)
		return
	}

	plan.IntegrationAuthID = types.StringValue(auth.IntegrationAuth.ID)
	plan.IntegrationID = types.StringValue(integration.Integration.ID)
	plan.Environment = types.StringValue(integration.Integration.Environment.Slug)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *IntegrationCircleCIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IntegrationCircleCIResourceModel
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

	state.SecretPath = types.StringValue(integration.Integration.SecretPath)
	state.IntegrationAuthID = types.StringValue(integration.Integration.IntegrationAuthID)
	state.Environment = types.StringValue(integration.Integration.Environment.Slug)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IntegrationCircleCIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationCircleCIResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IntegrationCircleCIResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateIntegrationAuth(infisical.UpdateIntegrationAuthRequest{
		Integration:       infisical.IntegrationAuthTypeCircleCi,
		IntegrationAuthId: plan.IntegrationAuthID.ValueString(),
		AccessToken:       plan.CircleCIToken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating integration auth",
			err.Error(),
		)
		return
	}

	// Update the integration
	_, err = r.client.UpdateIntegration(infisical.UpdateIntegrationRequest{
		ID:          state.IntegrationID.ValueString(),
		Environment: plan.Environment.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
		App:         plan.CircleCIProjectID.ValueString(), // Needs to be the project slug
		AppID:       plan.CircleCIProjectID.ValueString(), // Needs to be the project ID
		Owner:       plan.CircleCIOrgSlug.ValueString(),   // Needs to be the organization slug
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating integration",
			err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *IntegrationCircleCIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete CircleCI integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IntegrationCircleCIResourceModel
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
			"Error deleting CircleCI Integration",
			"Couldn't delete CircleCI integration from your Infiscial project, unexpected error: "+err.Error(),
		)
		return
	}
}
