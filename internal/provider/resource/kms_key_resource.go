package resource

import (
	"context"
	"fmt"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &kmsKeyResource{}
	_ resource.ResourceWithConfigure   = &kmsKeyResource{}
	_ resource.ResourceWithImportState = &kmsKeyResource{}
)

func NewKMSKeyResource() resource.Resource {
	return &kmsKeyResource{}
}

type kmsKeyResource struct {
	client *infisical.Client
}

type kmsKeyResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectId           types.String `tfsdk:"project_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	KeyUsage            types.String `tfsdk:"key_usage"`
	EncryptionAlgorithm types.String `tfsdk:"encryption_algorithm"`
	IsDisabled          types.Bool   `tfsdk:"is_disabled"`
	OrgId               types.String `tfsdk:"org_id"`
	Version             types.Int64  `tfsdk:"version"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	LastUpdated         types.String `tfsdk:"last_updated"`
}

func (r *kmsKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_key"
}

func (r *kmsKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage KMS keys in Infisical.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the KMS key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project where the KMS key will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the KMS key. Must be 1-32 characters.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the KMS key. Maximum 500 characters.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(500),
				},
			},
			"key_usage": schema.StringAttribute{
				Description: "The usage of the key. Options: 'encrypt-decrypt', 'sign-verify'. Defaults to 'encrypt-decrypt'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("encrypt-decrypt"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("encrypt-decrypt", "sign-verify"),
				},
			},
			"encryption_algorithm": schema.StringAttribute{
				Description: "The encryption algorithm for the key. Options: 'aes-256-gcm', 'aes-128-gcm', 'RSA_4096', 'ECC_NIST_P256'. Defaults to 'aes-256-gcm'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("aes-256-gcm"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("aes-256-gcm", "aes-128-gcm", "RSA_4096", "ECC_NIST_P256"),
				},
			},
			"is_disabled": schema.BoolAttribute{
				Description: "Whether the key is disabled. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"org_id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"version": schema.Int64Attribute{
				Description: "The version of the key.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the key.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the key.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "The last update timestamp of the resource.",
				Computed:    true,
			},
		},
	}
}

func (r *kmsKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *kmsKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan kmsKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createKMSKeyRequest := infisical.CreateKMSKeyRequest{
		ProjectId:   plan.ProjectId.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if !plan.KeyUsage.IsNull() && !plan.KeyUsage.IsUnknown() {
		createKMSKeyRequest.KeyUsage = plan.KeyUsage.ValueString()
	}

	if !plan.EncryptionAlgorithm.IsNull() && !plan.EncryptionAlgorithm.IsUnknown() {
		createKMSKeyRequest.EncryptionAlgorithm = plan.EncryptionAlgorithm.ValueString()
	}

	kmsKey, err := r.client.CreateKMSKey(createKMSKeyRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating KMS key",
			"Could not create KMS key, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(kmsKey.Key.ID)
	plan.OrgId = types.StringValue(kmsKey.Key.OrgId)
	plan.Version = types.Int64Value(int64(kmsKey.Key.Version))
	plan.CreatedAt = types.StringValue(kmsKey.Key.CreatedAt.Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(kmsKey.Key.UpdatedAt.Format(time.RFC3339))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	if plan.KeyUsage.IsNull() || plan.KeyUsage.IsUnknown() {
		plan.KeyUsage = types.StringValue(kmsKey.Key.KeyUsage)
	}

	if plan.EncryptionAlgorithm.IsNull() || plan.EncryptionAlgorithm.IsUnknown() {
		plan.EncryptionAlgorithm = types.StringValue(kmsKey.Key.EncryptionAlgorithm)
	}

	if plan.IsDisabled.IsNull() || plan.IsDisabled.IsUnknown() {
		plan.IsDisabled = types.BoolValue(kmsKey.Key.IsDisabled)
	} else {
		if plan.IsDisabled.ValueBool() != kmsKey.Key.IsDisabled {
			isDisabled := plan.IsDisabled.ValueBool()
			updateRequest := infisical.UpdateKMSKeyRequest{
				KeyId:      kmsKey.Key.ID,
				IsDisabled: &isDisabled,
			}

			updatedKey, err := r.client.UpdateKMSKey(updateRequest)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating KMS key disabled state",
					"Could not update KMS key disabled state, unexpected error: "+err.Error(),
				)
				return
			}

			plan.UpdatedAt = types.StringValue(updatedKey.Key.UpdatedAt.Format(time.RFC3339))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *kmsKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state kmsKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	kmsKey, err := r.client.GetKMSKey(infisical.GetKMSKeyRequest{
		KeyId: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading KMS key",
			"Could not read KMS key with ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(kmsKey.Key.Name)
	state.Description = types.StringValue(kmsKey.Key.Description)
	state.KeyUsage = types.StringValue(kmsKey.Key.KeyUsage)
	state.EncryptionAlgorithm = types.StringValue(kmsKey.Key.EncryptionAlgorithm)
	state.IsDisabled = types.BoolValue(kmsKey.Key.IsDisabled)
	state.OrgId = types.StringValue(kmsKey.Key.OrgId)
	state.Version = types.Int64Value(int64(kmsKey.Key.Version))
	state.CreatedAt = types.StringValue(kmsKey.Key.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(kmsKey.Key.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *kmsKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan kmsKeyResourceModel
	var state kmsKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := infisical.UpdateKMSKeyRequest{
		KeyId: state.ID.ValueString(),
	}

	if !plan.Name.Equal(state.Name) {
		updateRequest.Name = plan.Name.ValueString()
	}

	if !plan.Description.Equal(state.Description) {
		updateRequest.Description = plan.Description.ValueString()
	}

	if !plan.IsDisabled.Equal(state.IsDisabled) {
		isDisabled := plan.IsDisabled.ValueBool()
		updateRequest.IsDisabled = &isDisabled
	}

	updatedKey, err := r.client.UpdateKMSKey(updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating KMS key",
			"Could not update KMS key, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(updatedKey.Key.ID)
	plan.OrgId = types.StringValue(updatedKey.Key.OrgId)
	plan.Version = types.Int64Value(int64(updatedKey.Key.Version))
	plan.CreatedAt = types.StringValue(updatedKey.Key.CreatedAt.Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(updatedKey.Key.UpdatedAt.Format(time.RFC3339))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	plan.Name = types.StringValue(updatedKey.Key.Name)
	plan.Description = types.StringValue(updatedKey.Key.Description)
	plan.KeyUsage = types.StringValue(updatedKey.Key.KeyUsage)
	plan.EncryptionAlgorithm = types.StringValue(updatedKey.Key.EncryptionAlgorithm)
	plan.IsDisabled = types.BoolValue(updatedKey.Key.IsDisabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *kmsKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state kmsKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteKMSKey(infisical.DeleteKMSKeyRequest{
		KeyId: state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting KMS key",
			"Could not delete KMS key, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *kmsKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
