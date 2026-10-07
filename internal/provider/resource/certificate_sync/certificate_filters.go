package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	privateKeyCertificateOrders = "certificate_orders"
	privateKeyImported          = "imported"
)

var objectAsOptions = basetypes.ObjectAsOptions{}

var certificateMetadataFilterAttrTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

var certificateFiltersAttrTypes = map[string]attr.Type{
	"certificate_ids": types.SetType{ElemType: types.StringType},
	"profile_ids":     types.SetType{ElemType: types.StringType},
	"metadata":        types.SetType{ElemType: types.ObjectType{AttrTypes: certificateMetadataFilterAttrTypes}},
}

type certificateFiltersModel struct {
	CertificateIDs types.Set `tfsdk:"certificate_ids"`
	ProfileIDs     types.Set `tfsdk:"profile_ids"`
	Metadata       types.Set `tfsdk:"metadata"`
}

type certificateMetadataFilterModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func certificateFiltersSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Description: "Which of the application's certificates this sync holds. A certificate must match every field that is set: setting both `certificate_ids` and `profile_ids` selects only the certificates in both. " +
			"Leave the block out to manage the certificates outside Terraform, for example in the Infisical UI. An empty block, or an empty `certificate_ids`, makes the sync hold no certificates.",
		Attributes: map[string]schema.Attribute{
			"certificate_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The IDs of the certificates to sync. A certificate keeps syncing across renewals, so an ID that has since been renewed still selects the renewed certificate and does not show as drift.",
			},
			"profile_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Sync certificates issued from any one of these certificate profiles.",
			},
			"metadata": schema.SetNestedAttribute{
				Optional:    true,
				Description: "Sync certificates carrying every one of these metadata pairs.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:    true,
							Description: "The metadata key the certificate must carry.",
						},
						"value": schema.StringAttribute{
							Optional:    true,
							Description: "The value the key must have. Leave unset to match any value.",
						},
					},
				},
			},
		},
	}
}

func emptyCertificateFilters() types.Object {
	return types.ObjectValueMust(certificateFiltersAttrTypes, map[string]attr.Value{
		"certificate_ids": types.SetNull(types.StringType),
		"profile_ids":     types.SetNull(types.StringType),
		"metadata":        types.SetNull(types.ObjectType{AttrTypes: certificateMetadataFilterAttrTypes}),
	})
}

func hasAnyCertificateFilter(filters *infisical.CertificateSyncFilters) bool {
	return filters != nil && (len(filters.CertificateOrderIDs) > 0 || len(filters.ProfileIDs) > 0 || len(filters.Metadata) > 0)
}

func validateCertificateFilters(ctx context.Context, filters types.Object) diag.Diagnostics {
	var diags diag.Diagnostics
	if filters.IsNull() || filters.IsUnknown() {
		return diags
	}

	var model certificateFiltersModel
	diags.Append(filters.As(ctx, &model, objectAsOptions)...)
	if diags.HasError() || model.CertificateIDs.IsNull() || model.CertificateIDs.IsUnknown() {
		return diags
	}

	if len(model.CertificateIDs.Elements()) > 0 && !model.ProfileIDs.IsNull() {
		diags.AddAttributeWarning(
			path.Root(attrCertificateFilters),
			"Certificate filters are combined",
			"certificate_ids and profile_ids are both set, so the sync only holds listed certificates that were also issued from one of the profiles.",
		)
	}
	return diags
}

func stringsFromSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	values := []string{}
	if set.IsNull() || set.IsUnknown() {
		return values, nil
	}
	diags := set.ElementsAs(ctx, &values, false)
	return values, diags
}

func setOrNull(prior types.Set, values []string) types.Set {
	if len(values) == 0 && prior.IsNull() {
		return types.SetNull(types.StringType)
	}
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elements)
}

// certificateFiltersForRequest builds the API payload, translating certificate IDs into the
// certificate orders the API stores. A null block returns nil, which leaves the filters out.
func certificateFiltersForRequest(ctx context.Context, filters types.Object, orders *certificateOrderResolver) (*infisical.CertificateSyncFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if filters.IsNull() || filters.IsUnknown() {
		return nil, diags
	}

	var model certificateFiltersModel
	diags.Append(filters.As(ctx, &model, objectAsOptions)...)
	if diags.HasError() {
		return nil, diags
	}

	request := &infisical.CertificateSyncFilters{}

	certificateIDs, d := stringsFromSet(ctx, model.CertificateIDs)
	diags.Append(d...)
	seenOrders := map[string]bool{}
	for _, certificateID := range certificateIDs {
		orderID, err := orders.orderOf(certificateID)
		if err != nil {
			if err == infisical.ErrNotFound {
				diags.AddAttributeError(
					path.Root(attrCertificateFilters).AtName("certificate_ids"),
					"Certificate not found",
					fmt.Sprintf("Certificate %q does not exist in Infisical.", certificateID),
				)
				continue
			}
			diags.AddError("Error looking up certificate", fmt.Sprintf("Couldn't look up certificate %q: %s", certificateID, err.Error()))
			continue
		}
		if !seenOrders[orderID] {
			seenOrders[orderID] = true
			request.CertificateOrderIDs = append(request.CertificateOrderIDs, orderID)
		}
	}

	request.ProfileIDs, d = stringsFromSet(ctx, model.ProfileIDs)
	diags.Append(d...)

	if !model.Metadata.IsNull() && !model.Metadata.IsUnknown() {
		var metadata []certificateMetadataFilterModel
		diags.Append(model.Metadata.ElementsAs(ctx, &metadata, false)...)
		for _, pair := range metadata {
			filter := infisical.CertificateSyncMetadataFilter{Key: pair.Key.ValueString()}
			if !pair.Value.IsNull() {
				value := pair.Value.ValueString()
				filter.Value = &value
			}
			request.Metadata = append(request.Metadata, filter)
		}
	}

	return request, diags
}

// certificateFiltersFromApi maps the API's filters back onto the configured block. Certificate IDs
// are compared by order, so a configured ID whose certificate has since renewed is kept as is.
func certificateFiltersFromApi(ctx context.Context, prior types.Object, api *infisical.CertificateSyncFilters, orders *certificateOrderResolver, linked *linkedCertificates) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if prior.IsNull() {
		return prior, diags
	}
	if api == nil {
		api = &infisical.CertificateSyncFilters{}
	}

	var model certificateFiltersModel
	diags.Append(prior.As(ctx, &model, objectAsOptions)...)
	if diags.HasError() {
		return prior, diags
	}

	apiOrders := map[string]bool{}
	for _, orderID := range api.CertificateOrderIDs {
		apiOrders[orderID] = true
	}

	priorIDs, d := stringsFromSet(ctx, model.CertificateIDs)
	diags.Append(d...)

	certificateIDs := []string{}
	coveredOrders := map[string]bool{}
	for _, certificateID := range priorIDs {
		orderID, err := orders.orderOf(certificateID)
		if err != nil {
			if err == infisical.ErrNotFound {
				continue
			}
			diags.AddError("Error looking up certificate", fmt.Sprintf("Couldn't look up certificate %q: %s", certificateID, err.Error()))
			return prior, diags
		}
		if apiOrders[orderID] {
			certificateIDs = append(certificateIDs, certificateID)
			coveredOrders[orderID] = true
		}
	}

	missingOrders := []string{}
	for _, orderID := range api.CertificateOrderIDs {
		if !coveredOrders[orderID] {
			missingOrders = append(missingOrders, orderID)
		}
	}
	sort.Strings(missingOrders)

	for _, orderID := range missingOrders {
		current, err := linked.byOrder(orderID)
		if err != nil {
			diags.AddError("Error reading certificate sync certificates", "Couldn't list the certificate sync's certificates: "+err.Error())
			return prior, diags
		}
		if current == nil {
			diags.AddWarning(
				"Certificate order has no active certificate",
				fmt.Sprintf("The certificate sync selects certificate order %q, which has no active certificate to sync, so it can't be shown in certificate_ids.", orderID),
			)
			continue
		}
		orders.remember(current.CertificateID, orderID)
		certificateIDs = append(certificateIDs, current.CertificateID)
	}

	certificateIDSet := setOrNull(model.CertificateIDs, certificateIDs)
	profileIDSet := setOrNull(model.ProfileIDs, api.ProfileIDs)

	metadataType := types.ObjectType{AttrTypes: certificateMetadataFilterAttrTypes}
	metadataSet := types.SetNull(metadataType)
	if len(api.Metadata) > 0 || !model.Metadata.IsNull() {
		elements := make([]attr.Value, 0, len(api.Metadata))
		for _, pair := range api.Metadata {
			value := types.StringNull()
			if pair.Value != nil {
				value = types.StringValue(*pair.Value)
			}
			elements = append(elements, types.ObjectValueMust(certificateMetadataFilterAttrTypes, map[string]attr.Value{
				"key":   types.StringValue(pair.Key),
				"value": value,
			}))
		}
		metadataSet = types.SetValueMust(metadataType, elements)
	}

	result, d := types.ObjectValue(certificateFiltersAttrTypes, map[string]attr.Value{
		"certificate_ids": certificateIDSet,
		"profile_ids":     profileIDSet,
		"metadata":        metadataSet,
	})
	diags.Append(d...)
	return result, diags
}

type privateStateGetter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

type privateStateSetter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// certificateOrderResolver maps certificate IDs to their certificate orders. A certificate never
// changes order, so lookups are cached in private state and only new IDs reach the API.
type certificateOrderResolver struct {
	client *infisical.Client
	cache  map[string]string
	dirty  bool
}

func newCertificateOrderResolver(client *infisical.Client, cache map[string]string) *certificateOrderResolver {
	if cache == nil {
		cache = map[string]string{}
	}
	return &certificateOrderResolver{client: client, cache: cache, dirty: true}
}

func loadCertificateOrderResolver(ctx context.Context, client *infisical.Client, private privateStateGetter) (*certificateOrderResolver, diag.Diagnostics) {
	raw, diags := private.GetKey(ctx, privateKeyCertificateOrders)
	cache := map[string]string{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cache); err != nil {
			cache = map[string]string{}
		}
	}
	resolver := newCertificateOrderResolver(client, cache)
	resolver.dirty = false
	return resolver, diags
}

func (o *certificateOrderResolver) orderOf(certificateID string) (string, error) {
	if orderID, ok := o.cache[certificateID]; ok {
		return orderID, nil
	}
	response, err := o.client.GetCertificate(infisical.GetCertificateRequest{CertificateId: certificateID})
	if err != nil {
		return "", err
	}
	if response.Certificate.OrderId == "" {
		return "", fmt.Errorf("certificate %q has no certificate order", certificateID)
	}
	o.remember(certificateID, response.Certificate.OrderId)
	return response.Certificate.OrderId, nil
}

func (o *certificateOrderResolver) remember(certificateID, orderID string) {
	if o.cache[certificateID] == orderID {
		return
	}
	o.cache[certificateID] = orderID
	o.dirty = true
}

func (o *certificateOrderResolver) save(ctx context.Context, private privateStateSetter) diag.Diagnostics {
	if !o.dirty {
		return nil
	}
	raw, err := json.Marshal(o.cache)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error saving certificate orders", err.Error())
		return diags
	}
	return private.SetKey(ctx, privateKeyCertificateOrders, raw)
}

// certificateSyncCertificatesPageSize is the max page size accepted by the list certificates endpoint.
const certificateSyncCertificatesPageSize = 100

// linkedCertificates lazily lists the certificates a sync currently holds, fetching at most once.
type linkedCertificates struct {
	client            *infisical.Client
	certificateSyncID string
	loaded            bool
	certificates      []infisical.CertificateSyncCertificate
}

func newLinkedCertificates(client *infisical.Client, certificateSyncID string) *linkedCertificates {
	return &linkedCertificates{client: client, certificateSyncID: certificateSyncID}
}

func (l *linkedCertificates) all() ([]infisical.CertificateSyncCertificate, error) {
	if l.loaded {
		return l.certificates, nil
	}
	certificates, err := listCertificateSyncCertificates(l.client, l.certificateSyncID)
	if err != nil {
		return nil, err
	}
	l.certificates = certificates
	l.loaded = true
	return certificates, nil
}

func (l *linkedCertificates) byOrder(orderID string) (*infisical.CertificateSyncCertificate, error) {
	certificates, err := l.all()
	if err != nil {
		return nil, err
	}
	for i := range certificates {
		if certificates[i].CertificateOrderID == orderID {
			return &certificates[i], nil
		}
	}
	return nil, nil
}

func (l *linkedCertificates) defaultCertificate() (*infisical.CertificateSyncCertificate, error) {
	certificates, err := l.all()
	if err != nil {
		return nil, err
	}
	for i := range certificates {
		if certificates[i].SyncMetadata != nil && certificates[i].SyncMetadata.IsDefault {
			return &certificates[i], nil
		}
	}
	return nil, nil
}

func listCertificateSyncCertificates(client *infisical.Client, certificateSyncID string) ([]infisical.CertificateSyncCertificate, error) {
	certificates := []infisical.CertificateSyncCertificate{}
	offset := 0
	for {
		page, err := client.ListCertificateSyncCertificates(infisical.ListCertificateSyncCertificatesRequest{
			CertificateSyncID: certificateSyncID,
			Offset:            offset,
			Limit:             certificateSyncCertificatesPageSize,
		})
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, page.Certificates...)

		// Advance by how many items came back rather than by the limit we asked for: if a page
		// returns fewer items than requested, advancing by the limit would skip the difference.
		offset += len(page.Certificates)
		if len(page.Certificates) == 0 || offset >= page.TotalCount {
			return certificates, nil
		}
	}
}
