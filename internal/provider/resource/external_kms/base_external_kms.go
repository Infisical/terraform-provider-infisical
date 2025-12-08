package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ExternalKmsBaseResource is the resource implementation.
type ExternalKmsBaseResource struct {
	Provider                           infisical.ExternalKmsProvider // used for identifying external KMS route
	ResourceTypeName                   string                        // terraform resource name suffix
	ExternalKmsProviderName            string                        // complete descriptive name of the external KMS provider
	client                             *infisical.Client
	AllowedMethods                     []string
	ConfigurationAttributes            map[string]schema.Attribute
	ReadConfigurationForCreateFromPlan func(ctx context.Context, plan ExternalKmsBaseResourceModel) (map[string]any, diag.Diagnostics)
	ReadConfigurationForUpdateFromPlan func(ctx context.Context, plan ExternalKmsBaseResourceModel, state ExternalKmsBaseResourceModel) (map[string]any, diag.Diagnostics)
	OverwriteConfigurationFields       func(state *ExternalKmsBaseResourceModel) diag.Diagnostics
}

type ExternalKmsBaseResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Configuration   types.Object `tfsdk:"configuration"`
	CredentialsHash types.String `tfsdk:"credentials_hash"`
}

// Metadata returns the resource type name.
func (r *ExternalKmsBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

func (r *ExternalKmsBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s External KMS", r.ExternalKmsProviderName),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the KMS",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the KMS to create. Must be slug-friendly",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "An optional description for the KMS.",
			},
			"configuration": schema.SingleNestedAttribute{
				Required:    true,
				Description: fmt.Sprintf("The configuration for the %s External KMS", r.ExternalKmsProviderName),
				Attributes:  r.ConfigurationAttributes,
			},
			"credentials_hash": schema.StringAttribute{
				Computed:    true,
				Description: fmt.Sprintf("The hash of the %s External KMS credentials", r.ExternalKmsProviderName),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ExternalKmsBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ExternalKmsBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create external KMS",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan ExternalKmsBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configurationMap, diags := r.ReadConfigurationForCreateFromPlan(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	kms, err := r.client.CreateExternalKms(infisical.CreateExternalKmsRequest{
		Provider:      r.Provider,
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		Configuration: configurationMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating external KMS",
			"Couldn't create external KMS, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(kms.KmsId)
	plan.CredentialsHash = types.StringValue(kms.ExternalKms.CredentialsHash)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *ExternalKmsBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read external KMS",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state ExternalKmsBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	kms, err := r.client.GetExternalKmsById(infisical.GetExternalKmsByIdRequest{
		Provider: r.Provider,
		ID:       state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading external KMS",
				"Couldn't read external KMS, unexpected error: "+err.Error(),
			)
			return
		}
	}

	if state.CredentialsHash.ValueString() != kms.ExternalKms.CredentialsHash {
		resp.Diagnostics.AddWarning(
			"External KMS credentials conflict",
			fmt.Sprintf("The credentials for the %s External KMS with ID %s have been updated outside of Terraform.", r.ExternalKmsProviderName, state.ID.ValueString()),
		)

		// force TF update
		diags = r.OverwriteConfigurationFields(&state)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}

		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		return
	}

	if !(state.Description.IsNull() && kms.Description == "") {
		state.Description = types.StringValue(kms.Description)
	}

	state.Name = types.StringValue(kms.Name)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *ExternalKmsBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update external KMS",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan ExternalKmsBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ExternalKmsBaseResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configurationMap, diags := r.ReadConfigurationForUpdateFromPlan(ctx, plan, state)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	externalKms, err := r.client.UpdateExternalKms(infisical.UpdateExternalKmsRequest{
		ID:            state.ID.ValueString(),
		Provider:      r.Provider,
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		Configuration: configurationMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating external KMS",
			"Couldn't update external KMS, unexpected error: "+err.Error(),
		)
		return
	}

	plan.CredentialsHash = types.StringValue(externalKms.ExternalKms.CredentialsHash)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ExternalKmsBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete external KMS",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state ExternalKmsBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteExternalKms(infisical.DeleteExternalKmsRequest{
		Provider: r.Provider,
		ID:       state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting external KMS",
			"Couldn't delete external KMS from Infisical, unexpected error: "+err.Error(),
		)
	}
}
