package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

// optionalStringFromMap reads an optional string attribute, mapping a missing or empty value to null.
func optionalStringFromMap(m map[string]interface{}, key string) types.String {
	if value, ok := m[key].(string); ok && value != "" {
		return types.StringValue(value)
	}
	return types.StringNull()
}

// optionalTrimmedStringFromMap is optionalStringFromMap for attributes declared as customtypes.TrimmedStringType.
func optionalTrimmedStringFromMap(m map[string]interface{}, key string) customtypes.TrimmedStringValue {
	if value, ok := m[key].(string); ok && value != "" {
		return customtypes.NewTrimmedStringValue(value)
	}
	return customtypes.NewTrimmedStringNull()
}

// optionalBoolFromMap reads an optional boolean attribute, mapping a missing value to null.
func optionalBoolFromMap(m map[string]interface{}, key string) types.Bool {
	if value, ok := m[key].(bool); ok {
		return types.BoolValue(value)
	}
	return types.BoolNull()
}

// optionalInt64FromMap reads an optional numeric attribute. JSON numbers decode as float64.
func optionalInt64FromMap(m map[string]interface{}, key string) types.Int64 {
	if value, ok := m[key].(float64); ok {
		return types.Int64Value(int64(value))
	}
	return types.Int64Null()
}

// setOptionalString adds a configured, non-empty string to an API payload.
func setOptionalString(m map[string]interface{}, key string, value interface {
	IsNull() bool
	IsUnknown() bool
	ValueString() string
}) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return
	}
	m[key] = value.ValueString()
}

// setOptionalBool adds a configured boolean to an API payload.
func setOptionalBool(m map[string]interface{}, key string, value types.Bool) {
	if value.IsNull() || value.IsUnknown() {
		return
	}
	m[key] = value.ValueBool()
}

// setOptionalInt64 adds a configured number to an API payload.
func setOptionalInt64(m map[string]interface{}, key string, value types.Int64) {
	if value.IsNull() || value.IsUnknown() {
		return
	}
	m[key] = value.ValueInt64()
}

const (
	attrID                      = "id"
	attrConnectionID            = "connection_id"
	attrName                    = "name"
	attrApplicationID           = "application_id"
	attrDescription             = "description"
	attrAutoSyncEnabled         = "auto_sync_enabled"
	attrSyncOptions             = "sync_options"
	attrDestinationConfig       = "destination_config"
	attrCertificateFilters      = "certificate_filters"
	attrDefaultCertificateID    = "default_certificate_id"
	attrExportPassword          = "export_password"
	attrExportPasswordWO        = "export_password_wo"
	attrExportPasswordWOVersion = "export_password_wo_version"
)

// CertificateSyncBaseResource is the shared implementation behind every certificate sync destination.
// Each destination supplies its own destination_config / sync_options schema and the
// closures that translate between the Terraform model and the API's map-based payloads.
type CertificateSyncBaseResource struct {
	App              infisical.CertificateSyncApp // destination segment of the API path, e.g. "aws-certificate-manager"
	ResourceTypeName string                       // appended to the provider name, e.g. "_certificate_sync_aws_certificate_manager"
	SyncName         string                       // destination name as it appears in docs and errors, e.g. "AWS Certificate Manager"
	AppConnection    infisical.AppConnectionApp   // app connection type this destination authenticates with
	// AdditionalConnections lists other app connection types the destination also accepts.
	AdditionalConnections []infisical.AppConnectionApp
	client                *infisical.Client

	DestinationConfigAttributes   map[string]schema.Attribute
	ReadDestinationConfigFromPlan func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigFromApi  func(ctx context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics)

	// SyncOptionsAttributes is nil for destinations with nothing to configure; sync_options is then left out of the schema.
	SyncOptionsAttributes   map[string]schema.Attribute
	ReadSyncOptionsFromPlan func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsFromApi  func(ctx context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics)

	SupportsDefaultCertificate bool
	SupportsExportPassword     bool

	// ValidateConfigFunc runs destination-specific plan-time checks against the configuration.
	ValidateConfigFunc func(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics)
}

// CertificateSyncBaseResourceModel holds every attribute any destination can declare. Attributes a
// destination does not declare stay null and are never read from or written to plan and state.
type CertificateSyncBaseResourceModel struct {
	ID                      types.String
	ConnectionID            types.String
	Name                    customtypes.TrimmedStringValue
	ApplicationID           types.String
	Description             types.String
	AutoSyncEnabled         types.Bool
	SyncOptions             types.Object
	DestinationConfig       types.Object
	CertificateFilters      types.Object
	DefaultCertificateID    types.String
	ExportPassword          types.String
	ExportPasswordWO        types.String
	ExportPasswordWOVersion types.Int64
}

type attributeGetter interface {
	GetAttribute(ctx context.Context, p path.Path, target interface{}) diag.Diagnostics
}

func (r *CertificateSyncBaseResource) hasSyncOptions() bool {
	return r.SyncOptionsAttributes != nil
}

func (r *CertificateSyncBaseResource) modelFields(m *CertificateSyncBaseResourceModel) map[string]interface{} {
	fields := map[string]interface{}{
		attrID:                 &m.ID,
		attrConnectionID:       &m.ConnectionID,
		attrName:               &m.Name,
		attrApplicationID:      &m.ApplicationID,
		attrDescription:        &m.Description,
		attrAutoSyncEnabled:    &m.AutoSyncEnabled,
		attrDestinationConfig:  &m.DestinationConfig,
		attrCertificateFilters: &m.CertificateFilters,
	}
	if r.hasSyncOptions() {
		fields[attrSyncOptions] = &m.SyncOptions
	}
	if r.SupportsDefaultCertificate {
		fields[attrDefaultCertificateID] = &m.DefaultCertificateID
	}
	if r.SupportsExportPassword {
		fields[attrExportPassword] = &m.ExportPassword
		fields[attrExportPasswordWOVersion] = &m.ExportPasswordWOVersion
	}
	return fields
}

func (r *CertificateSyncBaseResource) newModel() CertificateSyncBaseResourceModel {
	return CertificateSyncBaseResourceModel{
		Name:                    customtypes.NewTrimmedStringNull(),
		SyncOptions:             types.ObjectNull(map[string]attr.Type{}),
		CertificateFilters:      types.ObjectNull(certificateFiltersAttrTypes),
		DefaultCertificateID:    types.StringNull(),
		ExportPassword:          types.StringNull(),
		ExportPasswordWO:        types.StringNull(),
		ExportPasswordWOVersion: types.Int64Null(),
	}
}

func (r *CertificateSyncBaseResource) getModel(ctx context.Context, src attributeGetter) (CertificateSyncBaseResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	m := r.newModel()
	for name, target := range r.modelFields(&m) {
		diags.Append(src.GetAttribute(ctx, path.Root(name), target)...)
	}
	return m, diags
}

// getConfigModel also reads export_password_wo, which only exists in configuration.
func (r *CertificateSyncBaseResource) getConfigModel(ctx context.Context, config tfsdk.Config) (CertificateSyncBaseResourceModel, diag.Diagnostics) {
	m, diags := r.getModel(ctx, config)
	if r.SupportsExportPassword {
		diags.Append(config.GetAttribute(ctx, path.Root(attrExportPasswordWO), &m.ExportPasswordWO)...)
	}
	return m, diags
}

func (r *CertificateSyncBaseResource) setModel(ctx context.Context, state *tfsdk.State, m CertificateSyncBaseResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	for name, value := range r.modelFields(&m) {
		diags.Append(state.SetAttribute(ctx, path.Root(name), value)...)
	}
	return diags
}

func (r *CertificateSyncBaseResource) connectionDescription() string {
	if len(r.AdditionalConnections) == 0 {
		return fmt.Sprintf("The ID of the %s Connection to use for syncing.", r.AppConnection)
	}
	apps := []string{string(r.AppConnection)}
	for _, app := range r.AdditionalConnections {
		apps = append(apps, string(app))
	}
	return fmt.Sprintf("The ID of the App Connection to use for syncing. Supported connection types: %s.", strings.Join(apps, ", "))
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
	resource.ImportStatePassthroughID(ctx, path.Root(attrID), req, resp)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, privateKeyImported, []byte("true"))...)
}

func (r *CertificateSyncBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		attrID: schema.StringAttribute{
			Description:   fmt.Sprintf("The ID of the %s certificate sync", r.SyncName),
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		attrConnectionID: schema.StringAttribute{
			Required:    true,
			Description: r.connectionDescription(),
		},
		attrName: schema.StringAttribute{
			Required:    true,
			CustomType:  customtypes.TrimmedStringType{},
			Description: fmt.Sprintf("The name of the %s sync to create.", r.SyncName),
		},
		attrApplicationID: schema.StringAttribute{
			Required:      true,
			Description:   "The ID of the Certificate Manager application to create the sync in.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		attrDescription: schema.StringAttribute{
			Optional:    true,
			Description: fmt.Sprintf("An optional description for the %s sync.", r.SyncName),
		},
		attrAutoSyncEnabled: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether certificates should be automatically synced to the destination when they are added or renewed.",
			Default:     booldefault.StaticBool(true),
		},
		attrDestinationConfig: schema.SingleNestedAttribute{
			Required:    true,
			Description: "The destination configuration for the certificate sync.",
			Attributes:  r.DestinationConfigAttributes,
		},
		attrCertificateFilters: certificateFiltersSchema(),
	}

	if r.hasSyncOptions() {
		attributes[attrSyncOptions] = schema.SingleNestedAttribute{
			Required:    true,
			Description: "Parameters to modify how certificates are synced.",
			Attributes:  r.SyncOptionsAttributes,
		}
	}

	if r.SupportsDefaultCertificate {
		attributes[attrDefaultCertificateID] = schema.StringAttribute{
			Optional: true,
			Description: "The ID of the certificate to set as the default certificate on every listener. The load balancer serves it when a client's SNI matches no other certificate. " +
				"It must be one of the certificates the sync holds. Leave unset to leave the listeners' default unmanaged; removing it later stops managing the default without changing it.",
		}
	}

	if r.SupportsExportPassword {
		attributes[attrExportPassword] = schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "The password protecting the exported PKCS#12 or Java KeyStore file. Required when `export_format` is `pkcs12` or `jks`, unless `export_password_wo` is set. Stored in state.",
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRoot(attrExportPasswordWO)),
				stringvalidator.ConflictsWith(path.MatchRoot(attrExportPasswordWOVersion)),
			},
		}
		attributes[attrExportPasswordWO] = schema.StringAttribute{
			Optional:    true,
			WriteOnly:   true,
			Sensitive:   true,
			Description: "The keystore export password as a write-only value that is never stored in state. Requires Terraform 1.11 or higher.",
			Validators: []validator.String{
				stringvalidator.AlsoRequires(path.MatchRoot(attrExportPasswordWOVersion)),
			},
		}
		attributes[attrExportPasswordWOVersion] = schema.Int64Attribute{
			Optional:    true,
			Description: "Used together with `export_password_wo` to trigger an update. Increment this value to send a new `export_password_wo`.",
			Validators: []validator.Int64{
				int64validator.AtLeast(1),
				int64validator.AlsoRequires(path.MatchRoot(attrExportPasswordWO)),
			},
		}
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s certificate syncs", r.SyncName),
		Attributes:  attributes,
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

func (r *CertificateSyncBaseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	config, diags := r.getConfigModel(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateCertificateFilters(ctx, config.CertificateFilters)...)

	if r.SupportsDefaultCertificate {
		resp.Diagnostics.Append(validateDefaultCertificateInFilters(ctx, config.DefaultCertificateID, config.CertificateFilters)...)
	}

	if r.ValidateConfigFunc != nil {
		r.ValidateConfigFunc(ctx, config, &resp.Diagnostics)
	}
}

func (r *CertificateSyncBaseResource) syncOptionsForRequest(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
	if r.ReadSyncOptionsFromPlan == nil {
		return map[string]interface{}{}, nil
	}
	return r.ReadSyncOptionsFromPlan(ctx, plan)
}

func (r *CertificateSyncBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	plan, diags := r.getModel(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	syncOptions, diags := r.syncOptionsForRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationConfig, diags := r.ReadDestinationConfigFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orders := newCertificateOrderResolver(r.client, nil)

	filters, diags := certificateFiltersForRequest(ctx, plan.CertificateFilters, orders)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := infisical.CreateCertificateSyncRequest{
		App:               r.App,
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		ConnectionID:      plan.ConnectionID.ValueString(),
		ApplicationID:     plan.ApplicationID.ValueString(),
		IsAutoSyncEnabled: plan.AutoSyncEnabled.ValueBool(),
		SyncOptions:       syncOptions,
		DestinationConfig: destinationConfig,
		Filters:           filters,
	}

	if r.SupportsExportPassword {
		password, diags := exportPasswordForCreate(ctx, plan, req.Config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if password != "" {
			request.Credentials = &infisical.CertificateSyncCredentials{ExportPassword: password}
		}
	}

	certificateSync, err := r.client.CreateCertificateSync(request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating certificate sync",
			"Couldn't create certificate sync, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(certificateSync.ID)

	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.SupportsDefaultCertificate && !plan.DefaultCertificateID.IsNull() {
		// An error would taint the new sync and force a recreate; the next refresh sees the unset default and retries it.
		for _, d := range r.setDefaultCertificate(certificateSync.ID, plan.DefaultCertificateID.ValueString(), orders).Errors() {
			resp.Diagnostics.AddWarning("Default certificate not set: "+d.Summary(), d.Detail())
		}
	}

	resp.Diagnostics.Append(orders.save(ctx, resp.Private)...)
}

func (r *CertificateSyncBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	state, diags := r.getModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
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

	if r.hasSyncOptions() {
		state.SyncOptions, diags = r.ReadSyncOptionsFromApi(ctx, certificateSync)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	state.DestinationConfig, diags = r.ReadDestinationConfigFromApi(ctx, certificateSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	imported, diags := req.Private.GetKey(ctx, privateKeyImported)
	resp.Diagnostics.Append(diags...)
	isImport := string(imported) == "true"

	orders, diags := loadCertificateOrderResolver(ctx, r.client, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	linked := newLinkedCertificates(r.client, certificateSync.ID)

	priorFilters := state.CertificateFilters
	if isImport && priorFilters.IsNull() && hasAnyCertificateFilter(certificateSync.Filters) {
		priorFilters = emptyCertificateFilters()
	}
	state.CertificateFilters, diags = certificateFiltersFromApi(ctx, priorFilters, certificateSync.Filters, orders, linked)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.SupportsDefaultCertificate && (!state.DefaultCertificateID.IsNull() || isImport) {
		state.DefaultCertificateID, diags = defaultCertificateFromApi(state.DefaultCertificateID, orders, linked)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, state)...)
	resp.Diagnostics.Append(orders.save(ctx, resp.Private)...)
	if isImport {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, privateKeyImported, nil)...)
	}
}

func (r *CertificateSyncBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	plan, diags := r.getModel(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := r.getModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	syncOptions, diags := r.syncOptionsForRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationConfig, diags := r.ReadDestinationConfigFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orders, diags := loadCertificateOrderResolver(ctx, r.client, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := infisical.UpdateCertificateSyncRequest{
		App:               r.App,
		ID:                state.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		ConnectionID:      plan.ConnectionID.ValueString(),
		IsAutoSyncEnabled: plan.AutoSyncEnabled.ValueBool(),
		SyncOptions:       syncOptions,
		DestinationConfig: destinationConfig,
	}

	// An omitted block leaves the list unmanaged, so the API's filters are only replaced when the configured block changed.
	if !plan.CertificateFilters.IsNull() && !plan.CertificateFilters.Equal(state.CertificateFilters) {
		request.Filters, diags = certificateFiltersForRequest(ctx, plan.CertificateFilters, orders)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if r.SupportsExportPassword {
		password, diags := exportPasswordForUpdate(ctx, plan, state, req.Config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if password != "" {
			request.Credentials = &infisical.CertificateSyncCredentials{ExportPassword: password}
		}
	}

	_, err := r.client.UpdateCertificateSync(request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating certificate sync",
			"Couldn't update certificate sync, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID

	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An unset default stops managing it rather than clearing the listeners' current default.
	if r.SupportsDefaultCertificate && !plan.DefaultCertificateID.IsNull() && !plan.DefaultCertificateID.Equal(state.DefaultCertificateID) {
		resp.Diagnostics.Append(r.setDefaultCertificate(state.ID.ValueString(), plan.DefaultCertificateID.ValueString(), orders)...)
	}

	resp.Diagnostics.Append(orders.save(ctx, resp.Private)...)
}

func (r *CertificateSyncBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete certificate sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attrID), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteCertificateSync(infisical.DeleteCertificateSyncRequest{
		App: r.App,
		ID:  id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting certificate sync",
			"Couldn't delete certificate sync from Infisical, unexpected error: "+err.Error(),
		)
	}
}

// setDefaultCertificate marks the synced certificate in the same order as certificateID as the default.
// The order is matched rather than the ID because the configured ID may predate a renewal.
func (r *CertificateSyncBaseResource) setDefaultCertificate(certificateSyncID, certificateID string, orders *certificateOrderResolver) diag.Diagnostics {
	var diags diag.Diagnostics

	orderID, err := orders.orderOf(certificateID)
	if err != nil {
		diags.AddError("Error resolving default certificate", fmt.Sprintf("Couldn't look up certificate %q: %s", certificateID, err.Error()))
		return diags
	}

	linked := newLinkedCertificates(r.client, certificateSyncID)
	current, err := linked.byOrder(orderID)
	if err != nil {
		diags.AddError("Error resolving default certificate", "Couldn't list the certificate sync's certificates: "+err.Error())
		return diags
	}
	if current == nil {
		diags.AddError(
			"Invalid default certificate",
			fmt.Sprintf("Certificate %q is not one of the certificates this sync holds. Add it to certificate_filters first.", certificateID),
		)
		return diags
	}

	err = r.client.SetCertificateSyncDefaultCertificate(infisical.SetCertificateSyncDefaultCertificateRequest{
		App:               r.App,
		CertificateSyncID: certificateSyncID,
		CertificateID:     current.CertificateID,
	})
	if err != nil {
		diags.AddError("Error setting default certificate", "Couldn't set the default certificate, unexpected error: "+err.Error())
	}
	return diags
}

func defaultCertificateFromApi(prior types.String, orders *certificateOrderResolver, linked *linkedCertificates) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics

	current, err := linked.defaultCertificate()
	if err != nil {
		diags.AddError("Error reading default certificate", "Couldn't list the certificate sync's certificates: "+err.Error())
		return prior, diags
	}
	if current == nil {
		return types.StringNull(), diags
	}

	if !prior.IsNull() {
		priorOrder, err := orders.orderOf(prior.ValueString())
		if err == nil && priorOrder == current.CertificateOrderID {
			return prior, diags
		}
	}

	orders.remember(current.CertificateID, current.CertificateOrderID)
	return types.StringValue(current.CertificateID), diags
}

func validateDefaultCertificateInFilters(ctx context.Context, defaultCertificateID types.String, filters types.Object) diag.Diagnostics {
	var diags diag.Diagnostics
	if defaultCertificateID.IsNull() || defaultCertificateID.IsUnknown() || filters.IsNull() || filters.IsUnknown() {
		return diags
	}

	var model certificateFiltersModel
	diags.Append(filters.As(ctx, &model, objectAsOptions)...)
	if diags.HasError() || model.CertificateIDs.IsNull() || model.CertificateIDs.IsUnknown() {
		return diags
	}

	for _, element := range model.CertificateIDs.Elements() {
		value, ok := element.(types.String)
		if !ok || value.IsUnknown() {
			return diags
		}
		if value.ValueString() == defaultCertificateID.ValueString() {
			return diags
		}
	}

	diags.AddAttributeError(
		path.Root(attrDefaultCertificateID),
		"Invalid default certificate",
		"default_certificate_id must be one of the IDs in certificate_filters.certificate_ids.",
	)
	return diags
}

func exportPasswordForCreate(ctx context.Context, plan CertificateSyncBaseResourceModel, config tfsdk.Config) (string, diag.Diagnostics) {
	if !plan.ExportPassword.IsNull() && !plan.ExportPassword.IsUnknown() {
		return plan.ExportPassword.ValueString(), nil
	}
	var writeOnly types.String
	diags := config.GetAttribute(ctx, path.Root(attrExportPasswordWO), &writeOnly)
	return writeOnly.ValueString(), diags
}

// exportPasswordForUpdate returns the password to send, or "" to leave the stored one in place.
// The API never returns the password, so a change can only be detected against the previous configuration.
func exportPasswordForUpdate(ctx context.Context, plan, state CertificateSyncBaseResourceModel, config tfsdk.Config) (string, diag.Diagnostics) {
	if !plan.ExportPassword.IsNull() && !plan.ExportPassword.IsUnknown() {
		if plan.ExportPassword.Equal(state.ExportPassword) {
			return "", nil
		}
		return plan.ExportPassword.ValueString(), nil
	}
	if plan.ExportPasswordWOVersion.IsNull() || plan.ExportPasswordWOVersion.Equal(state.ExportPasswordWOVersion) {
		return "", nil
	}
	var writeOnly types.String
	diags := config.GetAttribute(ctx, path.Root(attrExportPasswordWO), &writeOnly)
	return writeOnly.ValueString(), diags
}
