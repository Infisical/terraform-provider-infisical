package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

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

// boolFromMap extracts a boolean attribute from an API response map, falling back to def when
// the value is missing or not a boolean.
func boolFromMap(m map[string]interface{}, key string, def bool) types.Bool {
	if value, ok := m[key].(bool); ok {
		return types.BoolValue(value)
	}
	return types.BoolValue(def)
}

// PkiSyncBaseResource is the shared implementation behind every PKI sync destination.
// Each destination supplies its own destination_config / sync_options schema and the
// closures that translate between the Terraform model and the API's map-based payloads.
type PkiSyncBaseResource struct {
	App              infisical.PkiSyncApp // identifies the PKI sync destination route
	ResourceTypeName string               // terraform resource name suffix
	SyncName         string               // human friendly name of the destination
	AppConnection    infisical.AppConnectionApp
	client           *infisical.Client

	DestinationConfigAttributes   map[string]schema.Attribute
	ReadDestinationConfigFromPlan func(ctx context.Context, plan PkiSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigFromApi  func(ctx context.Context, pkiSync infisical.PkiSync) (types.Object, diag.Diagnostics)

	SyncOptionsAttributes   map[string]schema.Attribute
	ReadSyncOptionsFromPlan func(ctx context.Context, plan PkiSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsFromApi  func(ctx context.Context, pkiSync infisical.PkiSync) (types.Object, diag.Diagnostics)
}

type PkiSyncBaseResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ConnectionID      types.String `tfsdk:"connection_id"`
	Name              types.String `tfsdk:"name"`
	ApplicationID     types.String `tfsdk:"application_id"`
	Description       types.String `tfsdk:"description"`
	AutoSyncEnabled   types.Bool   `tfsdk:"auto_sync_enabled"`
	SyncOptions       types.Object `tfsdk:"sync_options"`
	DestinationConfig types.Object `tfsdk:"destination_config"`
}

func (r *PkiSyncBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

// ImportState imports an existing PKI sync by its ID. Read populates every attribute
func (r *PkiSyncBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PkiSyncBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s PKI syncs", r.SyncName),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   fmt.Sprintf("The ID of the %s PKI sync", r.SyncName),
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"connection_id": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The ID of the %s Connection to use for syncing.", r.AppConnection),
			},
			"name": schema.StringAttribute{
				Required:    true,
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
				Description: "The destination configuration for the PKI sync.",
				Attributes:  r.DestinationConfigAttributes,
			},
		},
	}
}

func (r *PkiSyncBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PkiSyncBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create PKI sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan PkiSyncBaseResourceModel
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

	pkiSync, err := r.client.CreatePkiSync(infisical.CreatePkiSyncRequest{
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
			"Error creating PKI sync",
			"Couldn't create PKI sync, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(pkiSync.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *PkiSyncBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read PKI sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state PkiSyncBaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pkiSync, err := r.client.GetPkiSyncById(infisical.GetPkiSyncByIdRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading PKI sync",
			"Couldn't read PKI sync, unexpected error: "+err.Error(),
		)
		return
	}

	state.ConnectionID = types.StringValue(pkiSync.ConnectionID)
	state.Name = types.StringValue(pkiSync.Name)
	state.ApplicationID = types.StringValue(pkiSync.ApplicationID)
	state.AutoSyncEnabled = types.BoolValue(pkiSync.IsAutoSyncEnabled)

	// Keep an unset optional description null instead of flipping it to "" on every read.
	if !(state.Description.IsNull() && pkiSync.Description == "") {
		state.Description = types.StringValue(pkiSync.Description)
	}

	var diags diag.Diagnostics
	state.SyncOptions, diags = r.ReadSyncOptionsFromApi(ctx, pkiSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.DestinationConfig, diags = r.ReadDestinationConfigFromApi(ctx, pkiSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PkiSyncBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update PKI sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan PkiSyncBaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PkiSyncBaseResourceModel
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

	_, err := r.client.UpdatePkiSync(infisical.UpdatePkiSyncRequest{
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
			"Error updating PKI sync",
			"Couldn't update PKI sync, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *PkiSyncBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete PKI sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state PkiSyncBaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeletePkiSync(infisical.DeletePkiSyncRequest{
		App: r.App,
		ID:  state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting PKI sync",
			"Couldn't delete PKI sync from Infisical, unexpected error: "+err.Error(),
		)
	}
}
