package resource

import (
	"context"
	"regexp"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const privateKeyImported = "imported"

var objectAsOptions = basetypes.ObjectAsOptions{}

var certificateMetadataFilterAttrTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

var certificateMetadataFilterType = types.ObjectType{AttrTypes: certificateMetadataFilterAttrTypes}

// certificateFiltersAttrTypes returns the certificate_filters attributes a sync declares. A sync
// limited to a fixed number of certificates only accepts an explicit certificate list, so it has no
// profile_ids or metadata.
func certificateFiltersAttrTypes(orderIDsOnly bool) map[string]attr.Type {
	attrTypes := map[string]attr.Type{
		"certificate_order_ids": types.SetType{ElemType: types.StringType},
	}
	if !orderIDsOnly {
		attrTypes["profile_ids"] = types.SetType{ElemType: types.StringType}
		attrTypes["metadata"] = types.SetType{ElemType: certificateMetadataFilterType}
	}
	return attrTypes
}

type certificateFiltersModel struct {
	CertificateOrderIDs types.Set
	ProfileIDs          types.Set
	Metadata            types.Set
}

type certificateMetadataFilterModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// decodeCertificateFilters reads a non-null certificate_filters block. Attributes the sync does not
// declare read as null.
func decodeCertificateFilters(filters types.Object) certificateFiltersModel {
	model := certificateFiltersModel{
		CertificateOrderIDs: types.SetNull(types.StringType),
		ProfileIDs:          types.SetNull(types.StringType),
		Metadata:            types.SetNull(certificateMetadataFilterType),
	}
	for name, target := range map[string]*types.Set{
		"certificate_order_ids": &model.CertificateOrderIDs,
		"profile_ids":           &model.ProfileIDs,
		"metadata":              &model.Metadata,
	} {
		if value, ok := filters.Attributes()[name].(types.Set); ok {
			*target = value
		}
	}
	return model
}

func certificateFiltersSchema(orderIDsOnly bool) schema.SingleNestedAttribute {
	certificateOrderIDs := schema.SetAttribute{
		Optional:    true,
		ElementType: types.StringType,
		Description: "The certificate orders to sync, from the `order_id` attribute of `infisical_cert_manager_certificate`. A certificate and all of its renewals share one order, so the sync keeps holding the newest certificate of each order.",
	}
	unmanaged := "Leave the block out to manage the certificates outside Terraform, for example in the Infisical UI. An empty block, or an empty `certificate_order_ids`, makes the sync hold no certificates."

	if orderIDsOnly {
		return schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Which certificate this sync holds. Leave the block out to manage the certificate outside Terraform, for example in the Infisical UI.",
			Attributes: map[string]schema.Attribute{"certificate_order_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The certificate order to sync, from the `order_id` attribute of `infisical_cert_manager_certificate`. Set exactly one order: this sync holds a single certificate, and keeps holding the newest renewal of that order.",
			}},
		}
	}

	return schema.SingleNestedAttribute{
		Optional: true,
		Description: "Which of the application's certificates this sync holds. A certificate must match every field that is set: setting both `certificate_order_ids` and `profile_ids` selects only the certificates in both. " +
			unmanaged,
		Attributes: map[string]schema.Attribute{
			"certificate_order_ids": certificateOrderIDs,
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
							Validators:  []validator.String{notBlank(), noSurroundingWhitespace()},
							Description: "The metadata key the certificate must carry.",
						},
						"value": schema.StringAttribute{
							Optional:    true,
							Validators:  []validator.String{noSurroundingWhitespace()},
							Description: "The value the key must have. Leave unset to match any value.",
						},
					},
				},
			},
		},
	}
}

func emptyCertificateFilters(orderIDsOnly bool) types.Object {
	attrTypes := certificateFiltersAttrTypes(orderIDsOnly)
	values := make(map[string]attr.Value, len(attrTypes))
	for name, attrType := range attrTypes {
		if setType, ok := attrType.(types.SetType); ok {
			values[name] = types.SetNull(setType.ElemType)
		}
	}
	return types.ObjectValueMust(attrTypes, values)
}

func hasAnyCertificateFilter(filters *infisical.CertificateSyncFilters) bool {
	if filters == nil {
		return false
	}
	return len(derefStrings(filters.CertificateOrderIDs)) > 0 || len(derefStrings(filters.ProfileIDs)) > 0 || (filters.Metadata != nil && len(*filters.Metadata) > 0)
}

func derefStrings(values *[]string) []string {
	if values == nil {
		return nil
	}
	return *values
}

func validateCertificateFilters(filters types.Object) diag.Diagnostics {
	var diags diag.Diagnostics
	if filters.IsNull() || filters.IsUnknown() {
		return diags
	}

	model := decodeCertificateFilters(filters)
	if model.CertificateOrderIDs.IsNull() || model.CertificateOrderIDs.IsUnknown() {
		return diags
	}

	if len(model.CertificateOrderIDs.Elements()) > 0 && !model.ProfileIDs.IsNull() {
		diags.AddAttributeWarning(
			path.Root(attrCertificateFilters),
			"Certificate filters are combined",
			"certificate_order_ids and profile_ids are both set, so the sync only holds certificates in the listed orders that were also issued from one of the profiles.",
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

// setFromApi returns null when the API leaves the field out, since an absent filter means no condition
// while an empty one matches nothing.
func setFromApi(present bool, values []string) types.Set {
	if !present {
		return types.SetNull(types.StringType)
	}
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elements)
}

// certificateFiltersForRequest builds the API payload. A null block returns nil, which leaves the filters out.
func certificateFiltersForRequest(ctx context.Context, filters types.Object) (*infisical.CertificateSyncFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if filters.IsNull() || filters.IsUnknown() {
		return nil, diags
	}

	model := decodeCertificateFilters(filters)
	request := &infisical.CertificateSyncFilters{}

	if !model.CertificateOrderIDs.IsNull() && !model.CertificateOrderIDs.IsUnknown() {
		orderIDs, d := stringsFromSet(ctx, model.CertificateOrderIDs)
		diags.Append(d...)
		request.CertificateOrderIDs = &orderIDs
	}

	if !model.ProfileIDs.IsNull() && !model.ProfileIDs.IsUnknown() {
		profileIDs, d := stringsFromSet(ctx, model.ProfileIDs)
		diags.Append(d...)
		request.ProfileIDs = &profileIDs
	}

	if !model.Metadata.IsNull() && !model.Metadata.IsUnknown() {
		var metadata []certificateMetadataFilterModel
		diags.Append(model.Metadata.ElementsAs(ctx, &metadata, false)...)
		filtersList := []infisical.CertificateSyncMetadataFilter{}
		for _, pair := range metadata {
			filter := infisical.CertificateSyncMetadataFilter{Key: pair.Key.ValueString()}
			if !pair.Value.IsNull() {
				value := pair.Value.ValueString()
				filter.Value = &value
			}
			filtersList = append(filtersList, filter)
		}
		request.Metadata = &filtersList
	}

	return request, diags
}

// certificateFiltersFromApi reads the API's filters into the block, keeping it null while it is
// unmanaged. It fills only the attributes the prior block declares.
func certificateFiltersFromApi(ctx context.Context, prior types.Object, api *infisical.CertificateSyncFilters) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if prior.IsNull() {
		return prior, diags
	}
	if api == nil {
		api = &infisical.CertificateSyncFilters{}
	}

	orderIDSet := setFromApi(api.CertificateOrderIDs != nil, derefStrings(api.CertificateOrderIDs))
	profileIDSet := setFromApi(api.ProfileIDs != nil, derefStrings(api.ProfileIDs))

	metadataSet := types.SetNull(certificateMetadataFilterType)
	if api.Metadata != nil {
		elements := make([]attr.Value, 0, len(*api.Metadata))
		for _, pair := range *api.Metadata {
			value := types.StringNull()
			if pair.Value != nil {
				value = types.StringValue(*pair.Value)
			}
			elements = append(elements, types.ObjectValueMust(certificateMetadataFilterAttrTypes, map[string]attr.Value{
				"key":   types.StringValue(pair.Key),
				"value": value,
			}))
		}
		metadataSet = types.SetValueMust(certificateMetadataFilterType, elements)
	}

	all := map[string]attr.Value{
		"certificate_order_ids": orderIDSet,
		"profile_ids":           profileIDSet,
		"metadata":              metadataSet,
	}
	attrTypes := prior.AttributeTypes(ctx)
	values := make(map[string]attr.Value, len(attrTypes))
	for name := range attrTypes {
		values[name] = all[name]
	}
	result, d := types.ObjectValue(attrTypes, values)
	diags.Append(d...)
	return result, diags
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

var (
	nonBlankPattern  = regexp.MustCompile(`\S`)
	untrimmedPattern = regexp.MustCompile(`^(\S(.*\S)?)?$`)
)

// notBlank rejects values the API would trim to nothing, which would read back as null and never settle.
func notBlank() validator.String {
	return stringvalidator.RegexMatches(nonBlankPattern, "must not be empty or only whitespace")
}

// noSurroundingWhitespace rejects values the API would trim, which would read back changed on every refresh.
func noSurroundingWhitespace() validator.String {
	return stringvalidator.RegexMatches(untrimmedPattern, "must not start or end with whitespace")
}
