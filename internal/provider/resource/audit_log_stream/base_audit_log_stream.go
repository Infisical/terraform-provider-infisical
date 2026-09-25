package resource

import (
	"context"
	"errors"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &AuditLogStreamBaseResource{}
	_ resource.ResourceWithImportState = &AuditLogStreamBaseResource{}
)

type credentialKind int

const (
	credentialString credentialKind = iota
	credentialInt
	// credentialHeaderMap is a map of header name to value, sent to the API as a list of
	// {key, value} objects.
	credentialHeaderMap
)

// credentialField declares one attribute of a provider's credentials block.
type credentialField struct {
	Name        string
	JSONName    string
	Kind        credentialKind
	Description string
	// Sensitive fields are never reconciled from the API, which redacts or omits them.
	Sensitive bool
	// Optional fields are Optional+Computed: dropping one from the config keeps the stored
	// value, matching the API, which leaves omitted credential fields untouched.
	Optional bool
	// MaskUnchanged sends the redaction sentinel in place of a value that has not changed, so an
	// unrelated update cannot overwrite a secret rotated outside Terraform. Only set this where
	// the API swaps the sentinel back for the stored value.
	MaskUnchanged bool
	Validators    []validator.String
}

// AuditLogStreamBaseResource implements every audit log stream provider. Providers differ only
// in their credentials, declared as CredentialFields.
type AuditLogStreamBaseResource struct {
	Provider         string // API provider slug, e.g. "sumo-logic"
	ProviderName     string // display name used in descriptions
	ResourceTypeName string // terraform resource name suffix
	CredentialFields []credentialField
	client           *infisical.Client
}

type AuditLogStreamResourceModel struct {
	ID          types.String `tfsdk:"id"`
	StreamMode  types.String `tfsdk:"stream_mode"`
	Credentials types.Object `tfsdk:"credentials"`
	Filters     types.Object `tfsdk:"filters"`
}

var auditLogStreamFilterAttrTypes = map[string]attr.Type{
	"products": types.SetType{ElemType: types.StringType},
}

func (f credentialField) attribute() schema.Attribute {
	switch f.Kind {
	case credentialInt:
		return schema.Int64Attribute{
			Required:    !f.Optional,
			Optional:    f.Optional,
			Computed:    f.Optional,
			Sensitive:   f.Sensitive,
			Description: f.Description,
		}
	case credentialHeaderMap:
		return schema.MapAttribute{
			ElementType: types.StringType,
			Required:    !f.Optional,
			Optional:    f.Optional,
			Computed:    f.Optional,
			Sensitive:   f.Sensitive,
			Description: f.Description,
		}
	default:
		return schema.StringAttribute{
			Required:    !f.Optional,
			Optional:    f.Optional,
			Computed:    f.Optional,
			Sensitive:   f.Sensitive,
			Description: f.Description,
			Validators:  f.Validators,
		}
	}
}

func (f credentialField) attrType() attr.Type {
	switch f.Kind {
	case credentialInt:
		return types.Int64Type
	case credentialHeaderMap:
		return types.MapType{ElemType: types.StringType}
	default:
		return types.StringType
	}
}

func (f credentialField) nullValue() attr.Value {
	switch f.Kind {
	case credentialInt:
		return types.Int64Null()
	case credentialHeaderMap:
		return types.MapNull(types.StringType)
	default:
		return types.StringNull()
	}
}

func (r *AuditLogStreamBaseResource) credentialAttrTypes() map[string]attr.Type {
	attrTypes := make(map[string]attr.Type, len(r.CredentialFields))
	for _, field := range r.CredentialFields {
		attrTypes[field.Name] = field.attrType()
	}
	return attrTypes
}

func (r *AuditLogStreamBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

func (r *AuditLogStreamBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	credentialAttributes := make(map[string]schema.Attribute, len(r.CredentialFields))
	for _, field := range r.CredentialFields {
		credentialAttributes[field.Name] = field.attribute()
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Stream organization audit logs to %s. Requires a plan with audit log streaming and an identity with organization settings permissions. Only Machine Identity authentication is supported.", r.ProviderName),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The audit log stream ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"stream_mode": schema.StringAttribute{
				Description: "How events are delivered. Read-only: new streams are always created in batch mode.",
				Computed:    true,
			},
			"credentials": schema.SingleNestedAttribute{
				Description: fmt.Sprintf("The %s delivery credentials. Secret values cannot be read back from the API, so drift in them is not detected.", r.ProviderName),
				Required:    true,
				Attributes:  credentialAttributes,
			},
			"filters": schema.SingleNestedAttribute{
				Description: "Scopes which audit logs the stream receives. Omit to stream every product.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"products": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Products to stream: secret-manager, cert-manager, kms, secret-scanning, pam, agent-vault, organization.",
					},
				},
			},
		},
	}
}

func (r *AuditLogStreamBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *AuditLogStreamBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.machineIdentityAuth(resp.Diagnostics.AddError, "create") {
		return
	}
	var plan AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentials, diags := r.credentialsToAPI(plan.Credentials, types.ObjectNull(r.credentialAttrTypes()))
	resp.Diagnostics.Append(diags...)
	filters, diags := auditLogStreamFiltersToAPI(ctx, plan.Filters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.client.CreateAuditLogStream(infisical.CreateAuditLogStreamRequest{
		Provider:    r.Provider,
		Credentials: credentials,
		Filters:     filters,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating audit log stream", err.Error())
		return
	}

	// The stream exists from here on, so state is written even when reconciling the response
	// fails. Returning early instead would leave it untracked and the next apply would create a
	// second one.
	resp.Diagnostics.Append(r.applyStreamToState(&plan, stream, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *AuditLogStreamBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.machineIdentityAuth(resp.Diagnostics.AddError, "read") {
		return
	}
	var state AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.client.GetAuditLogStreamById(infisical.AuditLogStreamByIdRequest{
		ID:       state.ID.ValueString(),
		Provider: r.Provider,
	})
	if errors.Is(err, infisical.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading audit log stream", err.Error())
		return
	}

	resp.Diagnostics.Append(r.applyStreamToState(&state, stream, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *AuditLogStreamBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.machineIdentityAuth(resp.Diagnostics.AddError, "update") {
		return
	}
	var plan, state AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentials, diags := r.credentialsToAPI(plan.Credentials, state.Credentials)
	resp.Diagnostics.Append(diags...)
	filters, diags := auditLogStreamFiltersToAPI(ctx, plan.Filters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.client.UpdateAuditLogStream(infisical.UpdateAuditLogStreamRequest{
		ID:          state.ID.ValueString(),
		Provider:    r.Provider,
		Credentials: credentials,
		Filters:     filters,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating audit log stream", err.Error())
		return
	}

	// The update has already been applied remotely, so state is written even when reconciling the
	// response fails; keeping the pre-update state would misreport what Infisical now holds.
	resp.Diagnostics.Append(r.applyStreamToState(&plan, stream, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *AuditLogStreamBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.machineIdentityAuth(resp.Diagnostics.AddError, "delete") {
		return
	}
	var state AuditLogStreamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAuditLogStream(infisical.AuditLogStreamByIdRequest{
		ID:       state.ID.ValueString(),
		Provider: r.Provider,
	})
	if err != nil && !errors.Is(err, infisical.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting audit log stream", err.Error())
	}
}

// ImportState imports by stream ID. Secret credential values cannot be read back, so the first
// plan after an import shows them as a change.
func (r *AuditLogStreamBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AuditLogStreamBaseResource) machineIdentityAuth(addError func(string, string), operation string) bool {
	if r.client != nil && r.client.Config.IsMachineIdentityAuth {
		return true
	}
	addError(
		"Unable to "+operation+" audit log stream",
		"Only Machine Identity authentication is supported for this operation.",
	)
	return false
}

// applyStreamToState writes the API response onto the model. reconcileFilters is only set during
// Read, where filters changed outside Terraform must surface as drift; after a write the planned
// value is already what was sent.
// It always leaves the model free of unknown values, even when it reports errors, so callers that
// have already created or updated a stream can persist it.
func (r *AuditLogStreamBaseResource) applyStreamToState(model *AuditLogStreamResourceModel, stream infisical.AuditLogStream, reconcileFilters bool) diag.Diagnostics {
	var diags diag.Diagnostics

	// Identifiers first. They cannot fail to convert, so recording them before anything below
	// can error is what keeps a stream that already exists tracked in state.
	model.ID = types.StringValue(stream.ID)
	model.StreamMode = types.StringValue(stream.StreamMode)

	if stream.Provider != "" && stream.Provider != r.Provider {
		diags.AddError(
			"Unexpected audit log stream provider",
			fmt.Sprintf("Audit log stream %s is for provider %q, not %q.", stream.ID, stream.Provider, r.Provider),
		)
		// Don't reconcile another provider's credentials onto this model.
		model.Credentials = r.knownCredentials(model.Credentials)
		return diags
	}

	credentials, credentialDiags := r.credentialsFromAPI(model.Credentials, stream.Credentials)
	diags.Append(credentialDiags...)
	model.Credentials = credentials

	if reconcileFilters {
		filters, filterDiags := auditLogStreamFiltersFromAPI(stream.Filters)
		diags.Append(filterDiags...)
		if filterDiags.HasError() {
			return diags
		}
		model.Filters = filters
	}

	return diags
}

// knownCredentials resolves unknown credential values to nulls. State cannot hold unknowns, so
// this is the fallback whenever the API response can't be read onto the model.
func (r *AuditLogStreamBaseResource) knownCredentials(current types.Object) types.Object {
	attrTypes := r.credentialAttrTypes()
	values := make(map[string]attr.Value, len(r.CredentialFields))

	currentAttributes := map[string]attr.Value{}
	if !current.IsNull() && !current.IsUnknown() {
		currentAttributes = current.Attributes()
	}

	for _, field := range r.CredentialFields {
		value := currentAttributes[field.Name]
		if value == nil || value.IsUnknown() {
			value = field.nullValue()
		}
		values[field.Name] = value
	}

	credentials, diags := types.ObjectValue(attrTypes, values)
	if diags.HasError() {
		return types.ObjectNull(attrTypes)
	}
	return credentials
}

// credentialsToAPI builds the request credentials, omitting fields with no configured value so
// the API keeps whatever it already has. prior is the state being replaced, or null on create.
func (r *AuditLogStreamBaseResource) credentialsToAPI(credentials, prior types.Object) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make(map[string]any, len(r.CredentialFields))

	if credentials.IsNull() || credentials.IsUnknown() {
		diags.AddError("Missing audit log stream credentials", "The credentials block is required.")
		return nil, diags
	}

	priorAttributes := map[string]attr.Value{}
	if !prior.IsNull() && !prior.IsUnknown() {
		priorAttributes = prior.Attributes()
	}

	attributes := credentials.Attributes()
	for _, field := range r.CredentialFields {
		value := attributes[field.Name]
		if value == nil || value.IsNull() || value.IsUnknown() {
			continue
		}

		if field.MaskUnchanged {
			if masked, ok := field.maskUnchanged(value, priorAttributes[field.Name]); ok {
				result[field.JSONName] = masked
				continue
			}
		}

		converted, err := field.valueToAPI(value)
		if err != nil {
			diags.AddError(
				"Invalid audit log stream credential",
				fmt.Sprintf("Credential %q is invalid: %s.", field.Name, err),
			)
			continue
		}
		result[field.JSONName] = converted
	}

	if diags.HasError() {
		return nil, diags
	}
	return result, diags
}

// maskUnchanged substitutes the redaction sentinel for values matching prior state. Header maps are
// masked per key, so changing one header leaves the rest untouched. Reports false when there is no
// usable prior value and the real one has to be sent.
func (f credentialField) maskUnchanged(value, prior attr.Value) (any, bool) {
	if prior == nil || prior.IsNull() || prior.IsUnknown() {
		return nil, false
	}

	if f.Kind != credentialHeaderMap {
		if !value.Equal(prior) {
			return nil, false
		}
		return infisical.AuditLogStreamRedactedCredential, true
	}

	headers, ok := value.(types.Map)
	priorHeaders, priorOk := prior.(types.Map)
	if !ok || !priorOk {
		return nil, false
	}

	priorElements := priorHeaders.Elements()
	masked := make([]map[string]string, 0, len(headers.Elements()))
	for key, element := range headers.Elements() {
		header, ok := element.(types.String)
		if !ok || header.IsNull() || header.IsUnknown() {
			return nil, false
		}
		text := header.ValueString()
		if priorValue, present := priorElements[key]; present && priorValue.Equal(element) {
			text = infisical.AuditLogStreamRedactedCredential
		}
		masked = append(masked, map[string]string{"key": key, "value": text})
	}
	return masked, true
}

func (f credentialField) valueToAPI(value attr.Value) (any, error) {
	switch f.Kind {
	case credentialInt:
		number, ok := value.(types.Int64)
		if !ok {
			return nil, fmt.Errorf("expected a number, got %T", value)
		}
		return number.ValueInt64(), nil
	case credentialHeaderMap:
		elements, ok := value.(types.Map)
		if !ok {
			return nil, fmt.Errorf("expected a map, got %T", value)
		}
		headers := make([]map[string]string, 0, len(elements.Elements()))
		for key, element := range elements.Elements() {
			header, ok := element.(types.String)
			if !ok || header.IsNull() || header.IsUnknown() {
				return nil, fmt.Errorf("header %q has no value", key)
			}
			headers = append(headers, map[string]string{"key": key, "value": header.ValueString()})
		}
		return headers, nil
	default:
		text, ok := value.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected a string, got %T", value)
		}
		return text.ValueString(), nil
	}
}

// credentialsFromAPI overlays the sanitized API credentials onto the current ones. Sensitive
// fields are left as-is because the API never returns their real values.
func (r *AuditLogStreamBaseResource) credentialsFromAPI(current types.Object, apiCredentials map[string]any) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	values := make(map[string]attr.Value, len(r.CredentialFields))

	currentAttributes := map[string]attr.Value{}
	if !current.IsNull() && !current.IsUnknown() {
		currentAttributes = current.Attributes()
	}

	for _, field := range r.CredentialFields {
		value := currentAttributes[field.Name]
		if value == nil || value.IsUnknown() {
			value = field.nullValue()
		}

		if !field.Sensitive {
			raw, present := apiCredentials[field.JSONName]
			if !present || raw == nil {
				value = field.nullValue()
			} else {
				converted, ok := field.valueFromAPI(raw)
				if ok {
					value = converted
				} else {
					// Keep the configured value rather than dropping the field, so the object
					// stays complete and state stays consistent with the configuration.
					diags.AddError(
						"Unexpected audit log stream credential type",
						fmt.Sprintf("The API returned %T for credential %q.", raw, field.JSONName),
					)
				}
			}
		}

		values[field.Name] = value
	}

	if diags.HasError() {
		return r.knownCredentials(current), diags
	}

	credentials, objectDiags := types.ObjectValue(r.credentialAttrTypes(), values)
	diags.Append(objectDiags...)
	return credentials, diags
}

func (f credentialField) valueFromAPI(raw any) (attr.Value, bool) {
	switch f.Kind {
	case credentialInt:
		number, ok := raw.(float64)
		if !ok {
			return nil, false
		}
		return types.Int64Value(int64(number)), true
	case credentialHeaderMap:
		// The API returns headers as a list of {key, value} objects. Duplicate keys collapse,
		// which Terraform's map cannot represent either.
		list, ok := raw.([]any)
		if !ok {
			return nil, false
		}
		elements := make(map[string]attr.Value, len(list))
		for _, item := range list {
			header, ok := item.(map[string]any)
			if !ok {
				return nil, false
			}
			key, keyOk := header["key"].(string)
			value, valueOk := header["value"].(string)
			if !keyOk || !valueOk {
				return nil, false
			}
			elements[key] = types.StringValue(value)
		}
		headers, diags := types.MapValue(types.StringType, elements)
		if diags.HasError() {
			return nil, false
		}
		return headers, true
	default:
		text, ok := raw.(string)
		if !ok {
			return nil, false
		}
		return types.StringValue(text), true
	}
}

func auditLogStreamFiltersToAPI(ctx context.Context, filters types.Object) (*infisical.AuditLogStreamFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if filters.IsNull() || filters.IsUnknown() {
		return nil, diags
	}

	products, ok := filters.Attributes()["products"].(types.Set)
	if !ok || products.IsNull() || products.IsUnknown() {
		return &infisical.AuditLogStreamFilters{}, diags
	}

	var values []string
	diags.Append(products.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return nil, diags
	}
	return &infisical.AuditLogStreamFilters{Products: values}, diags
}

func auditLogStreamFiltersFromAPI(filters *infisical.AuditLogStreamFilters) (types.Object, diag.Diagnostics) {
	if filters == nil {
		return types.ObjectNull(auditLogStreamFilterAttrTypes), nil
	}

	products := types.SetNull(types.StringType)
	var diags diag.Diagnostics
	if filters.Products != nil {
		values := make([]attr.Value, 0, len(filters.Products))
		for _, product := range filters.Products {
			values = append(values, types.StringValue(product))
		}
		var setDiags diag.Diagnostics
		products, setDiags = types.SetValue(types.StringType, values)
		diags.Append(setDiags...)
		if diags.HasError() {
			return types.ObjectNull(auditLogStreamFilterAttrTypes), diags
		}
	}

	object, objectDiags := types.ObjectValue(auditLogStreamFilterAttrTypes, map[string]attr.Value{"products": products})
	diags.Append(objectDiags...)
	return object, diags
}
