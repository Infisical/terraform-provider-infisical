package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &IntegrationAWSParameterStoreResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewIntegrationAwsParameterStoreResource() resource.Resource {
	return &IntegrationAWSParameterStoreResource{}
}

// IntegrationAwsParameterStore is the resource implementation.
type IntegrationAWSParameterStoreResource struct {
	client *infisical.Client
}

type AwsParameterStoreMetadataStruct struct {
	SecretAWSTag []struct {
		Key   string `json:"key,omitempty"`
		Value string `json:"value,omitempty"`
	} `json:"secretAWSTag,omitempty"`
	ShouldDisableDelete bool `json:"shouldDisableDelete,omitempty"`
}

type AwsParameterStoreOptions struct {
	AwsTags             []awsTag `tfsdk:"aws_tags"`
	ShouldDisableDelete bool     `tfsdk:"should_disable_delete"`
}

type awsTag struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// projectResourceSourceModel describes the data source data model.
type IntegrationAWSParameterStoreResourceModel struct {
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	ProjectID       types.String `tfsdk:"project_id"`

	EnvironmentID     types.String `tfsdk:"env_id"`
	IntegrationAuthID types.String `tfsdk:"integration_auth_id"`
	IntegrationID     types.String `tfsdk:"integration_id"`
	Environment       types.String `tfsdk:"environment"`
	SecretPath        types.String `tfsdk:"secret_path"`
	AWSPath           types.String `tfsdk:"parameter_store_path"`
	AWSRegion         types.String `tfsdk:"aws_region"`

	Options types.Object `tfsdk:"options"`
}

func extractOptionsParameterStore(ctx context.Context, diagnostics *diag.Diagnostics, inputOptions types.Object) (parsedOptions AwsParameterStoreMetadataStruct, hasError bool) {
	metadata := AwsParameterStoreMetadataStruct{}

	if !inputOptions.IsNull() && !inputOptions.IsUnknown() {
		var options AwsParameterStoreOptions
		diags := inputOptions.As(ctx, &options, basetypes.ObjectAsOptions{})
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return AwsParameterStoreMetadataStruct{}, true
		}

		// Load AWS tags into metadata
		for _, tag := range options.AwsTags {
			metadata.SecretAWSTag = append(metadata.SecretAWSTag, struct {
				Key   string `json:"key,omitempty"`
				Value string `json:"value,omitempty"`
			}{
				Key:   tag.Key.ValueString(),
				Value: tag.Value.ValueString(),
			})
		}

		metadata.ShouldDisableDelete = options.ShouldDisableDelete
	}

	return metadata, false
}

// Metadata returns the resource type name.
func (r *IntegrationAWSParameterStoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_aws_parameter_store"
}

// Schema defines the schema for the resource.
func (r *IntegrationAWSParameterStoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create AWS Parameter Store integration & save to Infisical. Only Machine Identity authentication is supported for this data source",
		Attributes: map[string]schema.Attribute{
			"options": schema.SingleNestedAttribute{
				Description: "Integration options",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"should_disable_delete": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether to disable deletion of existing secrets in AWS Parameter Store.",
					},

					"aws_tags": schema.ListNestedAttribute{
						Description:   "Tags to attach to the AWS parameter store secrets.",
						Optional:      true,
						PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Description:   "The key of the tag.",
									Optional:      true,
									PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
								},
								"value": schema.StringAttribute{
									Description:   "The value of the tag.",
									Optional:      true,
									PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
								},
							},
						},
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

			"aws_region": schema.StringAttribute{
				Required:      true,
				Description:   "The AWS region to sync secrets to.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"access_key_id": schema.StringAttribute{
				Sensitive:     true,
				Required:      true,
				Description:   "The AWS access key ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"secret_access_key": schema.StringAttribute{
				Sensitive:     true,
				Required:      true,
				Description:   "The AWS secret access key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of your Infisical project.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"parameter_store_path": schema.StringAttribute{
				Required:      true,
				Description:   "The path in AWS Parameter Store to sync secrets to.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"environment": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to sync to AWS Parameter Store (prod, dev, staging, etc).",
			},

			"secret_path": schema.StringAttribute{
				Required:    true,
				Description: "The secret path in Infisical to sync secrets from.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IntegrationAWSParameterStoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *IntegrationAWSParameterStoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationAWSParameterStoreResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create integration auth first
	auth, err := r.client.CreateIntegrationAuth(infisical.CreateIntegrationAuthRequest{
		AccessId:    plan.AccessKeyID.ValueString(),
		AccessToken: plan.SecretAccessKey.ValueString(),
		ProjectID:   plan.ProjectID.ValueString(),
		Integration: infisical.IntegrationAuthTypeAwsParameterStore,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create integration auth",
			err.Error(),
		)
		return
	}

	parsedOptions, hasError := extractOptionsParameterStore(ctx, &resp.Diagnostics, plan.Options)

	if hasError {
		return
	}

	// Convert metadata to map[string]interface{} if needed
	metadataMap := map[string]interface{}{
		"secretAWSTag":        parsedOptions.SecretAWSTag,
		"shouldDisableDelete": parsedOptions.ShouldDisableDelete,
	}

	// Create the integration
	integration, err := r.client.CreateIntegration(infisical.CreateIntegrationRequest{
		IntegrationAuthID: auth.IntegrationAuth.ID,
		Region:            plan.AWSRegion.ValueString(),
		Metadata:          metadataMap,
		SecretPath:        plan.SecretPath.ValueString(),
		Path:              plan.AWSPath.ValueString(),
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
	plan.EnvironmentID = types.StringValue(integration.Integration.EnvID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *IntegrationAWSParameterStoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IntegrationAWSParameterStoreResourceModel
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

	var planOptions AwsParameterStoreOptions

	if !state.Options.IsNull() {
		diags := state.Options.As(ctx, &planOptions, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateNeeded := false

	if integration.Integration.Metadata.ShouldDisableDelete != planOptions.ShouldDisableDelete {
		planOptions.ShouldDisableDelete = integration.Integration.Metadata.ShouldDisableDelete
		updateNeeded = true
	}

	found := false
	for _, tag := range integration.Integration.Metadata.SecretAWSTag {
		for _, planTag := range planOptions.AwsTags {
			if tag.Key == planTag.Key.ValueString() && tag.Value == planTag.Value.ValueString() {
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	if !found {
		// Create a new list of tags
		newTags := make([]awsTag, 0, len(integration.Integration.Metadata.SecretAWSTag))
		for _, tag := range integration.Integration.Metadata.SecretAWSTag {
			newTags = append(newTags, awsTag{
				Key:   types.StringValue(tag.Key),
				Value: types.StringValue(tag.Value),
			})
		}

		planOptions.AwsTags = newTags
		updateNeeded = true
	}

	if updateNeeded {
		// Convert AwsTags to types.List
		awsTagsValue, diags := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":   types.StringType,
				"value": types.StringType,
			},
		}, planOptions.AwsTags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Create a map of the updated options
		optionsMap := map[string]attr.Value{
			"aws_tags":              awsTagsValue,
			"should_disable_delete": types.BoolValue(planOptions.ShouldDisableDelete),
		}

		// Create a new types.Object with the updated options
		newOptions, diags := types.ObjectValue(
			map[string]attr.Type{
				"aws_tags":              types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType, "value": types.StringType}}},
				"should_disable_delete": types.BoolType,
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
func (r *IntegrationAWSParameterStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationAWSParameterStoreResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IntegrationAWSParameterStoreResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	parsedPlanOptions, hasError := extractOptionsParameterStore(ctx, &resp.Diagnostics, plan.Options)
	if hasError {
		return
	}

	// Convert metadata to map[string]interface{} if needed
	metadataMap := map[string]interface{}{
		"shouldDisableDelete": parsedPlanOptions.ShouldDisableDelete,
	}

	// Update the integration
	_, err := r.client.UpdateIntegration(infisical.UpdateIntegrationRequest{
		ID:          state.IntegrationID.ValueString(),
		Metadata:    metadataMap,
		Environment: plan.Environment.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
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

func (r *IntegrationAWSParameterStoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete AWS Parameter Store integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IntegrationAWSParameterStoreResourceModel
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
			"Error deleting AWS Parameter Store Integration",
			"Couldn't delete AWS Parameter Store integration from your Infiscial project, unexpected error: "+err.Error(),
		)
		return
	}
}
