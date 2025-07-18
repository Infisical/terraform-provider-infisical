package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RotateAtUtc struct {
	Hours   types.Int64 `tfsdk:"hours"`
	Minutes types.Int64 `tfsdk:"minutes"`
}

// SecretRotationBaseResource is the resource implementation.
type SecretRotationBaseResource struct {
	Provider           infisicalclient.SecretRotationProvider
	ResourceTypeName   string // terraform resource name suffix
	SecretRotationName string // complete descriptive name of the secret rotation
	AppConnection      infisicalclient.AppConnectionApp
	client             *infisical.Client

	ParametersAttributes   map[string]schema.Attribute
	ReadParametersFromPlan func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadParametersFromApi  func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics)

	SecretsMappingAttributes   map[string]schema.Attribute
	ReadSecretsMappingFromPlan func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSecretsMappingFromApi  func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics)
}

type SecretRotationBaseResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	AutoRotationEnabled types.Bool   `tfsdk:"auto_rotation_enabled"`
	ProjectID           types.String `tfsdk:"project_id"`
	ConnectionID        types.String `tfsdk:"connection_id"`
	Environment         types.String `tfsdk:"environment"`
	SecretPath          types.String `tfsdk:"secret_path"`

	RotationInterval types.Int32 `tfsdk:"rotation_interval"`
	RotateAtUtc      RotateAtUtc `tfsdk:"rotate_at_utc"`

	Parameters     types.Object `tfsdk:"parameters"`
	SecretsMapping types.Object `tfsdk:"secrets_mapping"`
}

// Metadata returns the resource type name.
func (r *SecretRotationBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

func (r *SecretRotationBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s Secret Rotations", r.SecretRotationName),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the secret rotation.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the secret rotation.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the secret rotation.",
			},
			"auto_rotation_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether secrets should be automatically rotated.",
				Default:     booldefault.StaticBool(true),
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Infisical project to create the secret rotation in.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"connection_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the connection to use for the secret rotation.",
			},
			"environment": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project environment to rotate secrets from.",
			},
			"secret_path": schema.StringAttribute{
				Required:    true,
				Description: "The folder path to rotate secrets from.",
			},

			"rotation_interval": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "How many days to wait between each rotation.",
				Default:     int32default.StaticInt32(30),
			},
			"rotate_at_utc": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "At which UTC time the rotation should occur.",
				Attributes: map[string]schema.Attribute{
					"hours": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "The hour at which the rotation should occur (UTC).",
						Default:     int64default.StaticInt64(0),
					},
					"minutes": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "The minute at which the rotation should occur (UTC).",
						Default:     int64default.StaticInt64(0),
					},
				},
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"hours":   types.Int64Type,
						"minutes": types.Int64Type,
					},
					map[string]attr.Value{
						"hours":   types.Int64Value(0),
						"minutes": types.Int64Value(0),
					},
				)),
			},

			"parameters": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Parameters to modify how secrets are rotated.",
				Attributes:  r.ParametersAttributes,
			},
			"secrets_mapping": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Secret mappings to modify how secrets are rotated.",
				Attributes:  r.SecretsMappingAttributes,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *SecretRotationBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *SecretRotationBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secret rotation",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan SecretRotationBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	parameters, diags := r.ReadParametersFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretsMapping, diags := r.ReadSecretsMappingFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretRotation, err := r.client.CreateSecretRotation(infisicalclient.CreateSecretRotationRequest{
		Provider:            r.Provider,
		Name:                plan.Name.ValueString(),
		Description:         plan.Description.ValueString(),
		AutoRotationEnabled: plan.AutoRotationEnabled.ValueBool(),
		ProjectID:           plan.ProjectID.ValueString(),
		ConnectionID:        plan.ConnectionID.ValueString(),
		Environment:         plan.Environment.ValueString(),
		SecretPath:          plan.SecretPath.ValueString(),

		RotationInterval: plan.RotationInterval.ValueInt32(),
		RotateAtUtc: infisicalclient.SecretRotationRotateAtUtc{
			Hours:   plan.RotateAtUtc.Hours.ValueInt64(),
			Minutes: plan.RotateAtUtc.Minutes.ValueInt64(),
		},

		Parameters:     parameters,
		SecretsMapping: secretsMapping,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret rotation",
			"Couldn't create secret rotation, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(secretRotation.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *SecretRotationBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read secret rotation",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state SecretRotationBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretRotation, err := r.client.GetSecretRotationById(infisicalclient.GetSecretRotationByIdRequest{
		Provider: r.Provider,
		ID:       state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading secret rotation",
				"Couldn't read secret rotation, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.Name = types.StringValue(secretRotation.Name)

	fmt.Printf("Secret rotation description from API: %s\n", secretRotation.Description)

	if !(state.Description.IsNull() && secretRotation.Description == "") {
		state.Description = types.StringValue(secretRotation.Description)
	}

	state.AutoRotationEnabled = types.BoolValue(secretRotation.AutoRotationEnabled)
	state.ProjectID = types.StringValue(secretRotation.ProjectID)
	state.ConnectionID = types.StringValue(secretRotation.Connection.ConnectionID)
	state.Environment = types.StringValue(secretRotation.Environment.Slug)
	state.SecretPath = types.StringValue(secretRotation.SecretFolder.Path)

	state.RotationInterval = types.Int32Value(secretRotation.RotationInterval)
	if secretRotation.RotateAtUtc != nil {
		state.RotateAtUtc.Hours = types.Int64Value(secretRotation.RotateAtUtc.Hours)
		state.RotateAtUtc.Minutes = types.Int64Value(secretRotation.RotateAtUtc.Minutes)
	} else {
		state.RotateAtUtc.Hours = types.Int64Value(0)
		state.RotateAtUtc.Minutes = types.Int64Value(0)
	}

	state.Parameters, diags = r.ReadParametersFromApi(ctx, secretRotation)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.SecretsMapping, diags = r.ReadSecretsMappingFromApi(ctx, secretRotation)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *SecretRotationBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret rotation",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan SecretRotationBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SecretRotationBaseResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	parameters, diags := r.ReadParametersFromPlan(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	secretsMapping, diags := r.ReadSecretsMappingFromPlan(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	_, err := r.client.UpdateSecretRotation(infisicalclient.UpdateSecretRotationRequest{
		Provider:            r.Provider,
		ID:                  state.ID.ValueString(),
		Name:                plan.Name.ValueString(),
		Description:         plan.Description.ValueString(),
		AutoRotationEnabled: plan.AutoRotationEnabled.ValueBool(),
		ConnectionID:        plan.ConnectionID.ValueString(),
		Environment:         plan.Environment.ValueString(),
		SecretPath:          plan.SecretPath.ValueString(),

		RotationInterval: plan.RotationInterval.ValueInt32(),
		RotateAtUtc: infisicalclient.SecretRotationRotateAtUtc{
			Hours:   plan.RotateAtUtc.Hours.ValueInt64(),
			Minutes: plan.RotateAtUtc.Minutes.ValueInt64(),
		},

		Parameters:     parameters,
		SecretsMapping: secretsMapping,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret rotation",
			"Couldn't update secret rotation, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SecretRotationBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret rotation",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state SecretRotationBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretRotation(infisical.DeleteSecretRotationRequest{
		Provider: r.Provider,
		ID:       state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting secret rotation",
			"Couldn't delete secret rotation from Infisical, unexpected error: "+err.Error(),
		)
	}
}
