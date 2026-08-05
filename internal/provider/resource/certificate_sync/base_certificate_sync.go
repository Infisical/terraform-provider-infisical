package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringFromMap extracts a string attribute from an API response map, recording an error on
// diags when the value is missing or not a string. Destinations should use it for required
// string fields so type errors are surfaced uniformly.
func stringFromMap(m map[string]interface{}, key string, diags *diag.Diagnostics) types.String {
	value, ok := m[key].(string)
	if !ok {
		diags.AddError(
			fmt.Sprintf("Invalid %s type", key),
			fmt.Sprintf("Expected '%s' to be a string but got something else", key),
		)
		return types.StringNull()
	}
	return types.StringValue(value)
}

// trimmedStringFromMap is stringFromMap for attributes the API stores trimmed. Those attributes
// are declared as customtypes.TrimmedStringType so a configured value with surrounding whitespace
// still matches what the API returns, and their values must be TrimmedStringValue to satisfy that
// declared type.
func trimmedStringFromMap(m map[string]interface{}, key string, diags *diag.Diagnostics) customtypes.TrimmedStringValue {
	value, ok := m[key].(string)
	if !ok {
		diags.AddError(
			fmt.Sprintf("Invalid %s type", key),
			fmt.Sprintf("Expected '%s' to be a string but got something else", key),
		)
		return customtypes.TrimmedStringValue{}
	}
	return customtypes.NewTrimmedStringValue(value)
}

// boolFromMap extracts a boolean attribute from an API response map, falling back to def when
// the value is missing or not a boolean. Pass the same value as the attribute's schema default,
// otherwise an option the API omits reads back as a change and shows up as drift.
func boolFromMap(m map[string]interface{}, key string, def bool) types.Bool {
	if value, ok := m[key].(bool); ok {
		return types.BoolValue(value)
	}
	return types.BoolValue(def)
}

// CertificateSyncBaseResource is the shared implementation behind every certificate sync destination.
// Each destination supplies its own destination_config / sync_options schema and the
// closures that translate between the Terraform model and the API's map-based payloads.
type CertificateSyncBaseResource struct {
	App              infisical.CertificateSyncApp // destination segment of the API path, e.g. "aws-certificate-manager"
	ResourceTypeName string                       // appended to the provider name, e.g. "_certificate_sync_aws_certificate_manager"
	SyncName         string                       // destination name as it appears in docs and errors, e.g. "AWS Certificate Manager"
	AppConnection    infisical.AppConnectionApp   // app connection type this destination authenticates with
	client           *infisical.Client

	DestinationConfigAttributes   map[string]schema.Attribute
	ReadDestinationConfigFromPlan func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigFromApi  func(ctx context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics)

	SyncOptionsAttributes   map[string]schema.Attribute
	ReadSyncOptionsFromPlan func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsFromApi  func(ctx context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics)
}

type CertificateSyncBaseResourceModel struct {
	ID                types.String                   `tfsdk:"id"`
	ConnectionID      types.String                   `tfsdk:"connection_id"`
	Name              customtypes.TrimmedStringValue `tfsdk:"name"`
	ApplicationID     types.String                   `tfsdk:"application_id"`
	Description       types.String                   `tfsdk:"description"`
	AutoSyncEnabled   types.Bool                     `tfsdk:"auto_sync_enabled"`
	SyncOptions       types.Object                   `tfsdk:"sync_options"`
	DestinationConfig types.Object                   `tfsdk:"destination_config"`
}

func (r *CertificateSyncBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

// ImportState imports an existing certificate sync by its ID. Read then populates every attribute.
func (r *CertificateSyncBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.Parse(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected the certificate sync ID to be a valid UUID, got: "+req.ID,
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CertificateSyncBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s certificate syncs", r.SyncName),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   fmt.Sprintf("The ID of the %s certificate sync", r.SyncName),
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"connection_id": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The ID of the %s Connection to use for syncing.", r.AppConnection),
			},
			"name": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: fmt.Sprintf("The name of the %s sync to create.", r.SyncName),
			},
			"application_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Certificate Manager application to create the sync in.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: fmt.Sprintf("An optional description for the %s sync.", r.SyncName),
			},
			"auto_sync_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether certificates should be automatically synced to the destination when they are added or renewed.",
				Default:     booldefault.StaticBool(true),
			},
			"sync_options": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Parameters to modify how certificates are synced.",
				Attributes:  r.SyncOptionsAttributes,
			},
			"destination_config": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The destination configuration for the certificate sync.",
				Attributes:  r.DestinationConfigAttributes,
			},
		},
	}
}

func (r *CertificateSyncBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Source Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *CertificateSyncBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan CertificateSyncBaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	syncOptions, diags := r.ReadSyncOptionsFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationConfig, diags := r.ReadDestinationConfigFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificateSync, err := r.client.CreateCertificateSync(infisical.CreateCertificateSyncRequest{
		App:               r.App,
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		ConnectionID:      plan.ConnectionID.ValueString(),
		ApplicationID:     plan.ApplicationID.ValueString(),
		IsAutoSyncEnabled: plan.AutoSyncEnabled.ValueBool(),
		SyncOptions:       syncOptions,
		DestinationConfig: destinationConfig,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating certificate sync",
			"Couldn't create certificate sync, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(certificateSync.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CertificateSyncBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state CertificateSyncBaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificateSync, err := r.client.GetCertificateSyncById(infisical.GetCertificateSyncByIdRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading certificate sync",
			"Couldn't read certificate sync, unexpected error: "+err.Error(),
		)
		return
	}

	state.ConnectionID = types.StringValue(certificateSync.ConnectionID)
	state.Name = customtypes.NewTrimmedStringValue(certificateSync.Name)
	state.ApplicationID = types.StringValue(certificateSync.ApplicationID)
	state.AutoSyncEnabled = types.BoolValue(certificateSync.IsAutoSyncEnabled)

	// Keep an unset optional description null instead of flipping it to "" on every read.
	if !(state.Description.IsNull() && certificateSync.Description == "") {
		state.Description = types.StringValue(certificateSync.Description)
	}

	var diags diag.Diagnostics
	state.SyncOptions, diags = r.ReadSyncOptionsFromApi(ctx, certificateSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.DestinationConfig, diags = r.ReadDestinationConfigFromApi(ctx, certificateSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *CertificateSyncBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan CertificateSyncBaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CertificateSyncBaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	syncOptions, diags := r.ReadSyncOptionsFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationConfig, diags := r.ReadDestinationConfigFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateCertificateSync(infisical.UpdateCertificateSyncRequest{
		App:               r.App,
		ID:                state.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		ConnectionID:      plan.ConnectionID.ValueString(),
		IsAutoSyncEnabled: plan.AutoSyncEnabled.ValueBool(),
		SyncOptions:       syncOptions,
		DestinationConfig: destinationConfig,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating certificate sync",
			"Couldn't update certificate sync, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CertificateSyncBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state CertificateSyncBaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteCertificateSync(infisical.DeleteCertificateSyncRequest{
		App: r.App,
		ID:  state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting certificate sync",
			"Couldn't delete certificate sync from Infisical, unexpected error: "+err.Error(),
		)
	}
}
