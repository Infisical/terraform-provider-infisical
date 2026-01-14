package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/strings"
	"time"

	"github.com/hashicorp/go-uuid"
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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &secretResource{}
)

// NewsecretResource is a helper function to simplify the provider implementation.
func NewSecretResource() resource.Resource {
	return &secretResource{}
}

// secretResource is the resource implementation.
type secretResource struct {
	client *infisical.Client
}

type SecretReminder struct {
	Note       types.String `tfsdk:"note"`
	RepeatDays types.Int64  `tfsdk:"repeat_days"`
}

// secretResourceSourceModel describes the data source data model.
type secretResourceModel struct {
	FolderPath     types.String    `tfsdk:"folder_path"`
	EnvSlug        types.String    `tfsdk:"env_slug"`
	Name           types.String    `tfsdk:"name"`
	SecretReminder *SecretReminder `tfsdk:"secret_reminder"`
	Value          types.String    `tfsdk:"value"`
	ValueWO        types.String    `tfsdk:"value_wo"`
	ValueWOVersion types.Int64     `tfsdk:"value_wo_version"`
	WorkspaceId    types.String    `tfsdk:"workspace_id"`
	LastUpdated    types.String    `tfsdk:"last_updated"`
	Tags           types.List      `tfsdk:"tag_ids"`
	Metadata       types.Map       `tfsdk:"metadata"`
	ID             types.String    `tfsdk:"id"`
}

type SecretData struct {
	IsWriteOnly       bool
	ShouldUpdateValue bool
	Value             string
}

func (m *secretResourceModel) getSecretValue(ctx context.Context, config tfsdk.Config, state *tfsdk.State, diags *diag.Diagnostics) (SecretData, error) {
	// check if normal value was configured
	if !m.Value.IsNull() && !m.Value.IsUnknown() {
		return SecretData{
			IsWriteOnly:       false,
			ShouldUpdateValue: true,
			Value:             m.Value.ValueString(),
		}, nil
	}

	// check write-only value if the normal value isn't set
	var secretValue types.String
	diags.Append(config.GetAttribute(ctx, path.Root("value_wo"), &secretValue)...)
	if diags.HasError() {
		return SecretData{}, errors.New("failed to get write-only secret value")
	}

	if !secretValue.IsNull() && !secretValue.IsUnknown() {
		shouldUpdateValue := true

		// if state exists we know its an update operation, so we need to compare versions
		if state != nil && !state.Raw.IsNull() {
			var newVersion types.Int64
			diags.Append(config.GetAttribute(ctx, path.Root("value_wo_version"), &newVersion)...)

			var stateModel secretResourceModel
			diags.Append(state.Get(ctx, &stateModel)...)

			if !diags.HasError() {
				oldVersion := int64(0)
				if !stateModel.ValueWOVersion.IsNull() {
					oldVersion = stateModel.ValueWOVersion.ValueInt64()
				}

				if oldVersion > newVersion.ValueInt64() {
					return SecretData{}, errors.New("new value write-only version is less than old version")
				}

				shouldUpdateValue = newVersion.ValueInt64() != oldVersion

			}
		}

		return SecretData{
			IsWriteOnly:       true,
			Value:             secretValue.ValueString(),
			ShouldUpdateValue: shouldUpdateValue,
		}, nil
	}

	return SecretData{}, errors.New("no secret value provided")
}

// Metadata returns the resource type name.
func (r *secretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

// Schema defines the schema for the resource.
func (r *secretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create secrets & save to Infisical",
		Attributes: map[string]schema.Attribute{
			"folder_path": schema.StringAttribute{
				Description:   "The path to the folder where the given secret resides",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
				Computed:      false,
			},
			"env_slug": schema.StringAttribute{
				Description:   "The environment slug of the secret to modify/create",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
				Computed:      false,
			},
			"name": schema.StringAttribute{
				Description: "The name of the secret",
				Required:    true,
				Computed:    false,
			},
			"value": schema.StringAttribute{
				Description: "The value of the secret in plain text. This is required if `value_wo` is not set.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("value_wo")),
					stringvalidator.ConflictsWith(path.MatchRoot("value_wo_version")),
				},
				Optional:  true,
				Computed:  false,
				Sensitive: true,
			},
			"value_wo": schema.StringAttribute{
				Description: "The value of the secret in plain text as a write-only secret. If set, the secret value will not be stored in state. This is required if `value` is not set. Requires Terraform version 1.11.0 or higher.",
				Optional:    true,
				Computed:    false,
				WriteOnly:   true,

				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.Expressions{
						path.MatchRoot("value_wo_version"),
					}...),
				},
			},
			"value_wo_version": schema.Int64Attribute{
				Description: "Used together with value_wo to trigger an update. Increment this value when an update to the value_wo is required.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AlsoRequires(path.Expressions{
						path.MatchRoot("value_wo"),
					}...),
				},
			},
			"workspace_id": schema.StringAttribute{
				Description:   "The Infisical project ID (Required for Machine Identity auth, and service tokens with multiple scopes)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
				Optional:      true,
				Computed:      true,
			},

			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"tag_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Tag ids to be attached for the secrets.",
			},
			"metadata": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Metadata associated with the secret as key-value pairs.",
			},
			"secret_reminder": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"note": schema.StringAttribute{
						Description: "Note for the secret rotation reminder",
						Computed:    false,
						Optional:    true,
					},
					"repeat_days": schema.Int64Attribute{
						Description: "Frequency of secret rotation reminder in days",
						Computed:    false,
						Required:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
							int64validator.AtMost(365),
						},
					},
				},
				Optional: true,
				Computed: false,
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the secret",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *secretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from plan
	var plan secretResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretData, err := plan.getSecretValue(ctx, req.Config, nil, &resp.Diagnostics)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret",
			"Could not get secret value, unexpected error: "+err.Error(),
		)
		return
	}

	planSecretTagIds := make([]types.String, 0, len(plan.Tags.Elements()))
	diags = plan.Tags.ElementsAs(ctx, &planSecretTagIds, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretTagIds := make([]string, 0, len(planSecretTagIds))
	for _, slug := range planSecretTagIds {
		secretTagIds = append(secretTagIds, strings.ToLower(slug.ValueString()))
	}

	planMetadata := make(map[string]types.String, 0)
	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		diags = plan.Metadata.ElementsAs(ctx, &planMetadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	secretMetadata := make([]infisical.SecretMetadataItem, 0, len(planMetadata))
	for key, value := range planMetadata {
		secretMetadata = append(secretMetadata, infisical.SecretMetadataItem{
			Key:   key,
			Value: value.ValueString(),
		})
	}

	var workspaceId string

	if r.client.Config.AuthStrategy == infisical.AuthStrategy.SERVICE_TOKEN {
		serviceTokenDetails, err := r.client.GetServiceTokenDetailsV2()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Could not get service token details, unexpected error: "+err.Error(),
			)
			return
		}
		workspaceId = serviceTokenDetails.Workspace

	} else if r.client.Config.IsMachineIdentityAuth {
		workspaceId = plan.WorkspaceId.ValueString()
	} else {
		resp.Diagnostics.AddError(
			"Error creating secret",
			"Unknown authentication strategy",
		)
		return
	}

	// null check secret reminder
	var secretReminderNote string
	var secretReminderRepeatDays int64

	if plan.SecretReminder != nil {
		secretReminderNote = plan.SecretReminder.Note.ValueString()
		secretReminderRepeatDays = plan.SecretReminder.RepeatDays.ValueInt64()
	}

	secret, err := r.client.CreateRawSecretsV3(infisical.CreateRawSecretV3Request{
		Environment:              plan.EnvSlug.ValueString(),
		WorkspaceID:              workspaceId,
		Type:                     "shared",
		SecretPath:               plan.FolderPath.ValueString(),
		SecretReminderNote:       secretReminderNote,
		SecretReminderRepeatDays: secretReminderRepeatDays,
		SecretKey:                plan.Name.ValueString(),
		SecretValue:              secretData.Value,
		TagIDs:                   secretTagIds,
		SecretMetadata:           secretMetadata,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret",
			"Couldn't save encrypted secrets to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(secret.ID)

	if len(secret.SecretMetadata) > 0 {
		metadataMap := make(map[string]types.String, len(secret.SecretMetadata))
		for _, item := range secret.SecretMetadata {
			metadataMap[item.Key] = types.StringValue(item.Value)
		}
		plan.Metadata, diags = types.MapValueFrom(ctx, types.StringType, metadataMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if plan.Metadata.IsNull() || plan.Metadata.IsUnknown() {
		plan.Metadata = types.MapNull(types.StringType)
	}

	plan.WorkspaceId = types.StringValue(workspaceId)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state secretResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed order value from HashiCups
	response, err := r.client.GetSingleRawSecretByNameV3(infisical.GetSingleSecretByNameV3Request{
		SecretName:  state.Name.ValueString(),
		Type:        "shared",
		WorkspaceId: state.WorkspaceId.ValueString(),
		Environment: state.EnvSlug.ValueString(),
		SecretPath:  state.FolderPath.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"Could not read Infisical secret named "+state.Name.ValueString()+": "+err.Error(),
			)
		}
		return
	}

	state.Name = types.StringValue(response.Secret.SecretKey)
	state.ID = types.StringValue(response.Secret.ID)
	if !state.Value.IsNull() && !state.Value.IsUnknown() {
		// Resource was configured with regular Value field
		state.Value = types.StringValue(response.Secret.SecretValue)
	}

	if len(response.Secret.SecretMetadata) > 0 {
		metadataMap := make(map[string]types.String, len(response.Secret.SecretMetadata))
		for _, item := range response.Secret.SecretMetadata {
			metadataMap[item.Key] = types.StringValue(item.Value)
		}
		state.Metadata, diags = types.MapValueFrom(ctx, types.StringType, metadataMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		state.Metadata = types.MapNull(types.StringType)
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *secretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan secretResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretData, err := plan.getSecretValue(ctx, req.Config, &req.State, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret",
			"Could not get secret value, unexpected error: "+err.Error(),
		)
		return
	}

	var state secretResourceModel
	diagsFromState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsFromState...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Name != plan.Name {
		resp.Diagnostics.AddError(
			"Unable to update secret",
			"Secret keys cannot be updated via Terraform at this time",
		)
		return
	}

	planSecretTagIds := make([]types.String, 0, len(plan.Tags.Elements()))
	diags = plan.Tags.ElementsAs(ctx, &planSecretTagIds, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretTagIds := make([]string, 0, len(planSecretTagIds))
	for _, slug := range planSecretTagIds {
		secretTagIds = append(secretTagIds, strings.ToLower(slug.ValueString()))
	}

	planMetadata := make(map[string]types.String, 0)
	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		diags = plan.Metadata.ElementsAs(ctx, &planMetadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	secretMetadata := make([]infisical.SecretMetadataItem, 0, len(planMetadata))
	for key, value := range planMetadata {
		secretMetadata = append(secretMetadata, infisical.SecretMetadataItem{
			Key:   key,
			Value: value.ValueString(),
		})
	}

	// null check secret reminder
	var secretReminderNote string
	var secretReminderRepeatDays int64

	if plan.SecretReminder != nil {
		secretReminderNote = plan.SecretReminder.Note.ValueString()
		secretReminderRepeatDays = plan.SecretReminder.RepeatDays.ValueInt64()
	}

	var workspaceId string
	if r.client.Config.AuthStrategy == infisical.AuthStrategy.SERVICE_TOKEN {

		serviceTokenDetails, err := r.client.GetServiceTokenDetailsV2()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"Could not get service token details, unexpected error: "+err.Error(),
			)
			return
		}

		workspaceId = serviceTokenDetails.Workspace

	} else if r.client.Config.IsMachineIdentityAuth {
		workspaceId = plan.WorkspaceId.ValueString()
	} else {
		resp.Diagnostics.AddError(
			"Error updating secret",
			"Unknown authentication strategy",
		)
		return
	}

	updateRequest := infisical.UpdateRawSecretByNameV3Request{
		Environment:              plan.EnvSlug.ValueString(),
		WorkspaceID:              workspaceId,
		Type:                     "shared",
		TagIDs:                   secretTagIds,
		SecretPath:               plan.FolderPath.ValueString(),
		SecretName:               plan.Name.ValueString(),
		SecretReminderNote:       secretReminderNote,
		SecretReminderRepeatDays: secretReminderRepeatDays,
		SecretMetadata:           secretMetadata,
	}

	if secretData.ShouldUpdateValue {
		updateRequest.SecretValue = pkg.StringToPtr(secretData.Value)
	}

	err = r.client.UpdateRawSecretV3(updateRequest)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret",
			"Couldn't save encrypted secrets to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	if len(secretMetadata) > 0 {
		metadataMap := make(map[string]types.String, len(secretMetadata))
		for _, item := range secretMetadata {
			metadataMap[item.Key] = types.StringValue(item.Value)
		}
		plan.Metadata, diags = types.MapValueFrom(ctx, types.StringType, metadataMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if plan.Metadata.IsNull() || plan.Metadata.IsUnknown() {
		plan.Metadata = types.MapNull(types.StringType)
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.WorkspaceId = types.StringValue(workspaceId)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state secretResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client.Config.IsMachineIdentityAuth || r.client.Config.AuthStrategy == infisical.AuthStrategy.SERVICE_TOKEN {
		err := r.client.DeleteRawSecretV3(infisical.DeleteRawSecretV3Request{
			SecretName:  state.Name.ValueString(),
			SecretPath:  state.FolderPath.ValueString(),
			Environment: state.EnvSlug.ValueString(),
			Type:        "shared",
			WorkspaceId: state.WorkspaceId.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting Infisical secret",
				"Could not delete secret, unexpected error: "+err.Error(),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Error deleting secret",
			"Unknown authentication strategy",
		)
		return
	}

}

func (r *secretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var workspace, environment, secretPath, secretName, secretValue, secretId string
	var tags []string
	var secretReminder SecretReminder
	var secretMetadata []infisical.SecretMetadataItem

	if _, err := uuid.ParseUUID(req.ID); err == nil {
		secret, err := r.client.GetSingleSecretByIDV3(infisical.GetSingleSecretByIDV3Request{
			ID: req.ID,
		})

		if err != nil {
			if err == infisical.ErrNotFound {
				resp.Diagnostics.AddError(
					"Secret not found",
					"The secret with the given ID was not found",
				)
			} else {
				resp.Diagnostics.AddError(
					"Error fetching secret",
					"Couldn't fetch secret from Infisical, unexpected error: "+err.Error(),
				)
			}
			return
		}

		for _, tag := range secret.Secret.Tags {
			tags = append(tags, tag.ID)
		}

		secretReminder.Note = types.StringValue(secret.Secret.SecretReminderNote)
		secretReminder.RepeatDays = types.Int64Value(secret.Secret.SecretReminderRepeatDays)

		workspace = secret.Secret.Workspace
		environment = secret.Secret.Environment
		secretPath = secret.Secret.SecretPath
		secretName = secret.Secret.SecretKey
		secretValue = secret.Secret.SecretValue
		secretId = secret.Secret.ID
		secretMetadata = secret.Secret.SecretMetadata
	} else {
		parts := strings.SplitN(req.ID, ":", 4)

		if len(parts) != 4 {
			resp.Diagnostics.AddError(
				"Invalid ID Format",
				"The secret ID must be a uuid or in the format of '<workspace>:<env>:<secret-path>:<secret-name>'",
			)

			return
		}

		secret, err := r.client.GetSingleRawSecretByNameV3(infisical.GetSingleSecretByNameV3Request{
			WorkspaceId: parts[0],
			Environment: parts[1],
			SecretPath:  parts[2],
			SecretName:  parts[3],
			Type:        "shared", // Just use the secret uuid instead if (type is 'personal')
		})

		if err != nil {
			if err == infisical.ErrNotFound {
				resp.Diagnostics.AddError(
					"Secret not found",
					"The secret with the given ID was not found",
				)
			} else {
				resp.Diagnostics.AddError(
					"Error fetching secret",
					"Couldn't fetch secret from Infisical, unexpected error: "+err.Error(),
				)
			}
			return
		}

		for _, tag := range secret.Secret.Tags {
			tags = append(tags, tag.ID)
		}

		secretReminder.Note = types.StringValue(secret.Secret.SecretReminderNote)
		secretReminder.RepeatDays = types.Int64Value(secret.Secret.SecretReminderRepeatDays)

		workspace = secret.Secret.Workspace
		environment = secret.Secret.Environment
		secretPath = secret.Secret.SecretPath
		secretName = secret.Secret.SecretKey
		secretValue = secret.Secret.SecretValue
		secretId = secret.Secret.ID
		secretMetadata = secret.Secret.SecretMetadata
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), workspace)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("env_slug"), environment)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("folder_path"), secretPath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), secretName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), secretValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag_ids"), tags)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_reminder"), secretReminder)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), secretId)...)

	if len(secretMetadata) > 0 {
		metadataMap := make(map[string]types.String, len(secretMetadata))
		for _, item := range secretMetadata {
			metadataMap[item.Key] = types.StringValue(item.Value)
		}
		metadataValue, diags := types.MapValueFrom(ctx, types.StringType, metadataMap)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata"), metadataValue)...)
		}
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata"), types.MapNull(types.StringType))...)
	}
}
