package resource

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	"terraform-provider-infisical/internal/crypto"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	WorkspaceId    types.String    `tfsdk:"workspace_id"`
	LastUpdated    types.String    `tfsdk:"last_updated"`
	Tags           types.List      `tfsdk:"tag_ids"`
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
				Description: "The value of the secret",
				Required:    true,
				Computed:    false,
				Sensitive:   true,
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
		encryptedValue, err := crypto.EncryptSymmetric([]byte(plan.Value.ValueString()), plainTextWorkspaceKey)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Couldn't encrypt secret value, unexpected error: "+err.Error(),
			)
			return
		}

		err = r.client.CreateSecretsV3(infisical.CreateSecretV3Request{
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
	} else if r.client.Config.IsMachineIdentityAuth {

		// null check secret reminder
		var secretReminderNote string
		var secretReminderRepeatDays int64

		if plan.SecretReminder != nil {
			secretReminderNote = plan.SecretReminder.Note.ValueString()
			secretReminderRepeatDays = plan.SecretReminder.RepeatDays.ValueInt64()
		}

		err := r.client.CreateRawSecretsV3(infisical.CreateRawSecretV3Request{
			Environment:              plan.EnvSlug.ValueString(),
			WorkspaceID:              plan.WorkspaceId.ValueString(),
			Type:                     "shared",
			SecretPath:               plan.FolderPath.ValueString(),
			SecretReminderNote:       secretReminderNote,
			SecretReminderRepeatDays: secretReminderRepeatDays,
			SecretKey:                plan.Name.ValueString(),
			SecretValue:              plan.Value.ValueString(),
			TagIDs:                   secretTagIds,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating secret",
				"Couldn't save encrypted secrets to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

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
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"Could not read Infisical secret named "+state.Name.ValueString()+": "+err.Error(),
			)
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

		state.Name = types.StringValue(string(plainTextKey))
		state.Value = types.StringValue(string(plainTextValue))

	} else if r.client.Config.IsMachineIdentityAuth {
		// Get refreshed order value from HashiCups
		response, err := r.client.GetSingleRawSecretByNameV3(infisical.GetSingleSecretByNameV3Request{
			SecretName:  state.Name.ValueString(),
			Type:        "shared",
			WorkspaceId: state.WorkspaceId.ValueString(),
			Environment: state.EnvSlug.ValueString(),
			SecretPath:  state.FolderPath.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Infisical secret",
				"Could not read Infisical secret named "+state.Name.ValueString()+": "+err.Error(),
			)
			return
		}

		state.Name = types.StringValue(response.Secret.SecretKey)
		state.Value = types.StringValue(response.Secret.SecretValue)
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
		encryptedSecretValue, err := crypto.EncryptSymmetric([]byte(plan.Value.ValueString()), plainTextWorkspaceKey)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"Couldn't encrypt secret value, unexpected error: "+err.Error(),
			)
			return
		}

		err = r.client.UpdateSecretsV3(infisical.UpdateSecretByNameV3Request{
			Environment:           plan.EnvSlug.ValueString(),
			SecretName:            plan.Name.ValueString(),
			Type:                  "shared",
			SecretPath:            plan.FolderPath.ValueString(),
			WorkspaceID:           serviceTokenDetails.Workspace,
			TagIDs:                secretTagIds,
			SecretValueCiphertext: base64.StdEncoding.EncodeToString(encryptedSecretValue.CipherText),
			SecretValueIV:         base64.StdEncoding.EncodeToString(encryptedSecretValue.Nonce),
			SecretValueTag:        base64.StdEncoding.EncodeToString(encryptedSecretValue.AuthTag),
		})

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
		err := r.client.UpdateRawSecretV3(infisical.UpdateRawSecretByNameV3Request{
			Environment:              plan.EnvSlug.ValueString(),
			WorkspaceID:              plan.WorkspaceId.ValueString(),
			Type:                     "shared",
			TagIDs:                   secretTagIds,
			SecretPath:               plan.FolderPath.ValueString(),
			SecretName:               plan.Name.ValueString(),
			SecretValue:              plan.Value.ValueString(),
			SecretReminderNote:       secretReminderNote,
			SecretReminderRepeatDays: secretReminderRepeatDays,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating secret",
				"Couldn't save encrypted secrets to Infiscial, unexpected error: "+err.Error(),
			)
			return
		}

		// No need to set workspace ID as it is already set in the plan
		//plan.WorkspaceId = plan.WorkspaceId
		plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

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
