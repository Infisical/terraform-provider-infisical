package resource

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"terraform-provider-infisical/internal/crypto"
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

	if r.client.Config.AuthStrategy == infisical.AuthStrategy.SERVICE_TOKEN {

		serviceTokenDetails, err := r.client.GetServiceTokenDetailsV2()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Could not get service token details, unexpected error: "+err.Error(),
			)
			return
		}

		// get plain text key
		symmetricKeyFromServiceToken, err := infisical.GetSymmetricKeyFromServiceToken(r.client.Config.ServiceToken)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Could not get encryption key, unexpected error: "+err.Error(),
			)
			return
		}

		decodedSymmetricEncryptionDetails, err := infisical.GetBase64DecodedSymmetricEncryptionDetails(symmetricKeyFromServiceToken, serviceTokenDetails.EncryptedKey, serviceTokenDetails.Iv, serviceTokenDetails.Tag)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"unable to get base 64 decoded encryption details, unexpected error: "+err.Error(),
			)
			return
		}

		plainTextWorkspaceKey, err := crypto.DecryptSymmetric([]byte(symmetricKeyFromServiceToken), decodedSymmetricEncryptionDetails.Cipher, decodedSymmetricEncryptionDetails.Tag, decodedSymmetricEncryptionDetails.IV)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"unable to decrypt the required workspace key, unexpected error: "+err.Error(),
			)
			return
		}

		// encrypt key
		encryptedKey, err := crypto.EncryptSymmetric([]byte(plan.Name.ValueString()), plainTextWorkspaceKey)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Couldn't encrypt secret key, unexpected error: "+err.Error(),
			)
			return
		}

		// encrypt value
		encryptedValue, err := crypto.EncryptSymmetric([]byte(secretData.Value), plainTextWorkspaceKey)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Couldn't encrypt secret value, unexpected error: "+err.Error(),
			)
			return
		}

		secret, err := r.client.CreateSecretsV3(infisical.CreateSecretV3Request{
			Environment: plan.EnvSlug.ValueString(),
			SecretName:  plan.Name.ValueString(),
			Type:        "shared",
			SecretPath:  plan.FolderPath.ValueString(),
			WorkspaceID: serviceTokenDetails.Workspace,

			SecretKeyCiphertext: base64.StdEncoding.EncodeToString(encryptedKey.CipherText),
			SecretKeyIV:         base64.StdEncoding.EncodeToString(encryptedKey.Nonce),
			SecretKeyTag:        base64.StdEncoding.EncodeToString(encryptedKey.AuthTag),

			SecretValueCiphertext: base64.StdEncoding.EncodeToString(encryptedValue.CipherText),
			SecretValueIV:         base64.StdEncoding.EncodeToString(encryptedValue.Nonce),
			SecretValueTag:        base64.StdEncoding.EncodeToString(encryptedValue.AuthTag),
			TagIDs:                secretTagIds,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Couldn't save encrypted secrets to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		// Set state to fully populated data
		plan.WorkspaceId = types.StringValue(serviceTokenDetails.Workspace)
		plan.ID = types.StringValue(secret.ID)
	} else if r.client.Config.IsMachineIdentityAuth {

		// null check secret reminder
		var secretReminderNote string
		var secretReminderRepeatDays int64

		if plan.SecretReminder != nil {
			secretReminderNote = plan.SecretReminder.Note.ValueString()
			secretReminderRepeatDays = plan.SecretReminder.RepeatDays.ValueInt64()
		}

		secret, err := r.client.CreateRawSecretsV3(infisical.CreateRawSecretV3Request{
			Environment:              plan.EnvSlug.ValueString(),
			WorkspaceID:              plan.WorkspaceId.ValueString(),
			Type:                     "shared",
			SecretPath:               plan.FolderPath.ValueString(),
			SecretReminderNote:       secretReminderNote,
			SecretReminderRepeatDays: secretReminderRepeatDays,
			SecretKey:                plan.Name.ValueString(),
			SecretValue:              secretData.Value,
			TagIDs:                   secretTagIds,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Couldn't save encrypted secrets to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		plan.ID = types.StringValue(secret.ID)

		// No need to set workspace ID as it is already set in the plan
		//plan.WorkspaceId = plan.WorkspaceId
	} else {
		resp.Diagnostics.AddError(
			"Error creating secret",
			"Unknown authentication strategy",
		)
		return
	}
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

	if r.client.Config.AuthStrategy == infisical.AuthStrategy.SERVICE_TOKEN {

		serviceTokenDetails, err := r.client.GetServiceTokenDetailsV2()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Could not get service token details, unexpected error: "+err.Error(),
			)
			return
		}

		// get plain text key
		symmetricKeyFromServiceToken, err := infisical.GetSymmetricKeyFromServiceToken(r.client.Config.ServiceToken)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Could not get encryption key, unexpected error: "+err.Error(),
			)
			return
		}

		decodedSymmetricEncryptionDetails, err := infisical.GetBase64DecodedSymmetricEncryptionDetails(symmetricKeyFromServiceToken, serviceTokenDetails.EncryptedKey, serviceTokenDetails.Iv, serviceTokenDetails.Tag)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"unable to get base 64 decoded encryption details, unexpected error: "+err.Error(),
			)
			return
		}

		plainTextWorkspaceKey, err := crypto.DecryptSymmetric([]byte(symmetricKeyFromServiceToken), decodedSymmetricEncryptionDetails.Cipher, decodedSymmetricEncryptionDetails.Tag, decodedSymmetricEncryptionDetails.IV)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"unable to decrypt the required workspace key, unexpected error: "+err.Error(),
			)
			return
		}

		// Get refreshed order value from HashiCups
		response, err := r.client.GetSingleSecretByNameV3(infisical.GetSingleSecretByNameV3Request{
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

		// Decrypt key
		key_iv, err := base64.StdEncoding.DecodeString(response.Secret.SecretKeyIV)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"unable to decode secret IV for secret key "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		key_tag, err := base64.StdEncoding.DecodeString(response.Secret.SecretKeyTag)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"unable to decode secret authentication tag for secret key "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		key_ciphertext, err := base64.StdEncoding.DecodeString(response.Secret.SecretKeyCiphertext)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"unable to decode secret cipher text for secret key "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		plainTextKey, err := crypto.DecryptSymmetric(plainTextWorkspaceKey, key_ciphertext, key_tag, key_iv)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"unable to symmetrically decrypt secret key "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		// Decrypt value
		value_iv, err := base64.StdEncoding.DecodeString(response.Secret.SecretValueIV)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"unable to decode secret IV for secret value "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		value_tag, err := base64.StdEncoding.DecodeString(response.Secret.SecretValueTag)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"unable to decode secret authentication tag for secret value "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		value_ciphertext, err := base64.StdEncoding.DecodeString(response.Secret.SecretValueCiphertext)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"unable to decode secret cipher text for secret key "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		plainTextValue, err := crypto.DecryptSymmetric(plainTextWorkspaceKey, value_ciphertext, value_tag, value_iv)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"unable to symmetrically decrypt secret value "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		if !state.Value.IsNull() && !state.Value.IsUnknown() {
			// Resource was configured with regular Value field
			state.Value = types.StringValue(string(plainTextValue))
		}

		state.Name = types.StringValue(string(plainTextKey))
		state.ID = types.StringValue(response.Secret.ID)

	} else if r.client.Config.IsMachineIdentityAuth {
		// Get refreshed order value from HashiCups
		response, err := r.client.GetSingleRawSecretByNameV3(infisical.GetSingleSecretByNameV3Request{
			SecretName:             state.Name.ValueString(),
			Type:                   "shared",
			WorkspaceId:            state.WorkspaceId.ValueString(),
			Environment:            state.EnvSlug.ValueString(),
			SecretPath:             state.FolderPath.ValueString(),
			ExpandSecretReferences: false,
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

	} else {
		resp.Diagnostics.AddError(
			"Error Reading Infisical secret",
			"Unknown authentication strategy",
		)
		return
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

	// null check secret reminder
	var secretReminderNote string
	var secretReminderRepeatDays int64

	if plan.SecretReminder != nil {
		secretReminderNote = plan.SecretReminder.Note.ValueString()
		secretReminderRepeatDays = plan.SecretReminder.RepeatDays.ValueInt64()
	}

	if r.client.Config.AuthStrategy == infisical.AuthStrategy.SERVICE_TOKEN {

		serviceTokenDetails, err := r.client.GetServiceTokenDetailsV2()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"Could not get service token details, unexpected error: "+err.Error(),
			)
			return
		}

		// get plain text key
		symmetricKeyFromServiceToken, err := infisical.GetSymmetricKeyFromServiceToken(r.client.Config.ServiceToken)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"Could not get encryption key, unexpected error: "+err.Error(),
			)
			return
		}

		decodedSymmetricEncryptionDetails, err := infisical.GetBase64DecodedSymmetricEncryptionDetails(symmetricKeyFromServiceToken, serviceTokenDetails.EncryptedKey, serviceTokenDetails.Iv, serviceTokenDetails.Tag)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"unable to get base 64 decoded encryption details, unexpected error: "+err.Error(),
			)
			return
		}

		plainTextWorkspaceKey, err := crypto.DecryptSymmetric([]byte(symmetricKeyFromServiceToken), decodedSymmetricEncryptionDetails.Cipher, decodedSymmetricEncryptionDetails.Tag, decodedSymmetricEncryptionDetails.IV)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"unable to decrypt the required workspace key, unexpected error: "+err.Error(),
			)
			return
		}

		// encrypt value
		encryptedSecretValue, err := crypto.EncryptSymmetric([]byte(secretData.Value), plainTextWorkspaceKey)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"Couldn't encrypt secret value, unexpected error: "+err.Error(),
			)
			return
		}

		updateRequest := infisical.UpdateSecretByNameV3Request{
			Environment: plan.EnvSlug.ValueString(),
			SecretName:  plan.Name.ValueString(),
			Type:        "shared",
			SecretPath:  plan.FolderPath.ValueString(),
			WorkspaceID: serviceTokenDetails.Workspace,
			TagIDs:      secretTagIds,
		}

		if secretData.ShouldUpdateValue {
			updateRequest.SecretValueCiphertext = pkg.StringToPtr(base64.StdEncoding.EncodeToString(encryptedSecretValue.CipherText))
			updateRequest.SecretValueIV = pkg.StringToPtr(base64.StdEncoding.EncodeToString(encryptedSecretValue.Nonce))
			updateRequest.SecretValueTag = pkg.StringToPtr(base64.StdEncoding.EncodeToString(encryptedSecretValue.AuthTag))
		}

		err = r.client.UpdateSecretsV3(updateRequest)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"Couldn't save encrypted secrets to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		// Set state to fully populated data
		plan.WorkspaceId = types.StringValue(serviceTokenDetails.Workspace)

	} else if r.client.Config.IsMachineIdentityAuth {

		updateRequest := infisical.UpdateRawSecretByNameV3Request{
			Environment:              plan.EnvSlug.ValueString(),
			WorkspaceID:              plan.WorkspaceId.ValueString(),
			Type:                     "shared",
			TagIDs:                   secretTagIds,
			SecretPath:               plan.FolderPath.ValueString(),
			SecretName:               plan.Name.ValueString(),
			SecretReminderNote:       secretReminderNote,
			SecretReminderRepeatDays: secretReminderRepeatDays,
		}

		if secretData.ShouldUpdateValue {
			updateRequest.SecretValue = pkg.StringToPtr(secretData.Value)
		}

		err := r.client.UpdateRawSecretV3(updateRequest)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"Couldn't save encrypted secrets to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		// No need to set workspace ID as it is already set in the plan
		//plan.WorkspaceId = plan.WorkspaceId

	} else {
		resp.Diagnostics.AddError(
			"Error updating secret",
			"Unknown authentication strategy",
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
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

	if r.client.Config.AuthStrategy == infisical.AuthStrategy.SERVICE_TOKEN {
		// Delete existing order
		err := r.client.DeleteSecretsV3(infisical.DeleteSecretV3Request{
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
	} else if r.client.Config.IsMachineIdentityAuth {
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
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), workspace)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("env_slug"), environment)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("folder_path"), secretPath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), secretName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), secretValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag_ids"), tags)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_reminder"), secretReminder)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), secretId)...)
}
