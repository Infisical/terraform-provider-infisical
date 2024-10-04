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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &IntegrationAWSSecretsManagerResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewIntegrationAwsSecretsManagerResource() resource.Resource {
	return &IntegrationAWSSecretsManagerResource{}
}

// IntegrationAwsSecretsManager is the resource implementation.
type IntegrationAWSSecretsManagerResource struct {
	client *infisical.Client
}

type AwsSecretsManagerMetadataStruct struct {
	SecretAWSTag []struct {
		Key   string `json:"key,omitempty"`
		Value string `json:"value,omitempty"`
	} `json:"secretAWSTag,omitempty"`
	SecretPrefix string `json:"secretPrefix,omitempty"`
}

type AwsSecretsManagerOptions struct {
	AwsTags      []awsTag `tfsdk:"aws_tags"`
	SecretPrefix string   `tfsdk:"secret_prefix"`
}

type AwsTag struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// projectResourceSourceModel describes the data source data model.
type IntegrationAWSSecretsManagerResourceModel struct {
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	ProjectID       types.String `tfsdk:"project_id"`

	EnvironmentID     types.String `tfsdk:"env_id"`
	IntegrationAuthID types.String `tfsdk:"integration_auth_id"`
	IntegrationID     types.String `tfsdk:"integration_id"`

	Environment types.String `tfsdk:"environment"`
	SecretPath  types.String `tfsdk:"secret_path"`
	AWSRegion   types.String `tfsdk:"aws_region"`

	MappingBehavior types.String `tfsdk:"mapping_behavior"`
	AWSPath         types.String `tfsdk:"secrets_manager_path"`

	Options types.Object `tfsdk:"options"`
}

const MAPPING_BEHAVIOR_MANY_TO_ONE = "many-to-one"
const MAPPING_BEHAVIOR_ONE_TO_ONE = "one-to-one"

func extractSecretsManagerMetadata(ctx context.Context, diagnostics *diag.Diagnostics, inputOptions types.Object) (parsedOptions AwsSecretsManagerMetadataStruct, hasError bool) {
	metadata := AwsSecretsManagerMetadataStruct{}

	if !inputOptions.IsNull() && !inputOptions.IsUnknown() {
		var options AwsSecretsManagerOptions
		diags := inputOptions.As(ctx, &options, basetypes.ObjectAsOptions{})
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return AwsSecretsManagerMetadataStruct{}, true
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

		metadata.SecretPrefix = options.SecretPrefix
	}

	return metadata, false
}

// Metadata returns the resource type name.
func (r *IntegrationAWSSecretsManagerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_aws_secrets_manager"
}

// Schema defines the schema for the resource.
func (r *IntegrationAWSSecretsManagerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create AWS Secrets Manager integration & save to Infisical. Only Machine Identity authentication is supported for this data source",
		Attributes: map[string]schema.Attribute{
			"options": schema.SingleNestedAttribute{
				Description: "Integration options",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"secret_prefix": schema.StringAttribute{
						Optional:    true,
						Description: "The prefix to add to the secret name in AWS Secrets Manager.",
					},

					"aws_tags": schema.ListNestedAttribute{
						Description:   "Tags to attach to the AWS Secrets Manager secrets.",
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

			"environment": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to sync to AWS Secrets Manager (prod, dev, staging, etc).",
			},

			"mapping_behavior": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString(MAPPING_BEHAVIOR_MANY_TO_ONE),
				Description:   "The behavior of the mapping. Can be 'many-to-one' or 'one-to-one'. Many to One: All Infisical secrets will be mapped to a single AWS secret. One to One: Each Infisical secret will be mapped to its own AWS secret.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"secrets_manager_path": schema.StringAttribute{
				Optional:      true,
				Description:   "The path in AWS Secrets Manager to sync secrets to. This is required if mapping_behavior is 'many-to-one'.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},

			"secret_path": schema.StringAttribute{
				Required:    true,
				Description: "The secret path in Infisical to sync secrets from.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IntegrationAWSSecretsManagerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *IntegrationAWSSecretsManagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationAWSSecretsManagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.MappingBehavior.ValueString() == MAPPING_BEHAVIOR_MANY_TO_ONE && (plan.AWSPath.IsNull() || plan.AWSPath.ValueString() == "") {
		resp.Diagnostics.AddError(
			"Invalid plan",
			"secrets_manager_path is required when mapping_behavior is 'many-to-one'",
		)
		return
	}

	if plan.MappingBehavior.ValueString() == MAPPING_BEHAVIOR_ONE_TO_ONE && (!plan.AWSPath.IsNull() && plan.AWSPath.ValueString() != "") {
		resp.Diagnostics.AddError(
			"Invalid plan",
			"secrets_manager_path is not required when mapping_behavior is 'one-to-one'",
		)
		return
	}

	// Create integration auth first
	auth, err := r.client.CreateIntegrationAuth(infisical.CreateIntegrationAuthRequest{
		AccessId:    plan.AccessKeyID.ValueString(),
		AccessToken: plan.SecretAccessKey.ValueString(),
		ProjectID:   plan.ProjectID.ValueString(),
		Integration: infisical.IntegrationAuthTypeAwsSecretsManager,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create integration auth",
			err.Error(),
		)
		return
	}

	parsedOptions, hasError := extractSecretsManagerMetadata(ctx, &resp.Diagnostics, plan.Options)

	if hasError {
		return
	}

	// Convert metadata to map[string]interface{} if needed
	metadataMap := map[string]interface{}{
		"secretAWSTag":    parsedOptions.SecretAWSTag,
		"mappingBehavior": plan.MappingBehavior.ValueString(),
		"secretPrefix":    parsedOptions.SecretPrefix,
	}

	request := infisical.CreateIntegrationRequest{
		IntegrationAuthID: auth.IntegrationAuth.ID,
		Region:            plan.AWSRegion.ValueString(),
		Metadata:          metadataMap,
		SecretPath:        plan.SecretPath.ValueString(),
		SourceEnvironment: plan.Environment.ValueString(),
	}

	if plan.MappingBehavior.ValueString() == MAPPING_BEHAVIOR_MANY_TO_ONE {
		request.App = plan.AWSPath.ValueString()
	}

	// Create the integration
	integration, err := r.client.CreateIntegration(request)

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
func (r *IntegrationAWSSecretsManagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IntegrationAWSSecretsManagerResourceModel
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

	var planOptions AwsSecretsManagerOptions

	if !state.Options.IsNull() {
		diags := state.Options.As(ctx, &planOptions, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateNeeded := false

	if integration.Integration.Metadata.SecretPrefix != planOptions.SecretPrefix {
		planOptions.SecretPrefix = integration.Integration.Metadata.SecretPrefix
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
			"aws_tags":      awsTagsValue,
			"secret_prefix": types.StringValue(planOptions.SecretPrefix),
		}

		// Create a new types.Object with the updated options
		newOptions, diags := types.ObjectValue(
			map[string]attr.Type{
				"aws_tags":      types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType, "value": types.StringType}}},
				"secret_prefix": types.StringType,
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
func (r *IntegrationAWSSecretsManagerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationAWSSecretsManagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IntegrationAWSSecretsManagerResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	parsedPlanOptions, hasError := extractSecretsManagerMetadata(ctx, &resp.Diagnostics, plan.Options)
	if hasError {
		return
	}

	// Convert metadata to map[string]interface{} if needed
	metadataMap := map[string]interface{}{
		"secretPrefix": parsedPlanOptions.SecretPrefix,
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

func (r *IntegrationAWSSecretsManagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete AWS Secrets Manager integration",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IntegrationAWSSecretsManagerResourceModel
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
			"Error deleting AWS Secrets Manager Integration",
			"Couldn't delete AWS Secrets Manager integration from your Infiscial project, unexpected error: "+err.Error(),
		)
		return
	}
}
