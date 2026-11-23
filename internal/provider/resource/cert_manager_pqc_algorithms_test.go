package resource

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Copied from CertKeyAlgorithm and CertSignatureAlgorithm in Infisical core
// (backend/src/services/certificate-common/certificate-constants.ts).
var infisicalCorePqcAlgorithms = []string{
	"ML-DSA-44", "ML-DSA-65", "ML-DSA-87",
	"SLH-DSA-SHA2-128f", "SLH-DSA-SHA2-128s", "SLH-DSA-SHA2-192f", "SLH-DSA-SHA2-192s",
	"SLH-DSA-SHA2-256f", "SLH-DSA-SHA2-256s", "SLH-DSA-SHAKE-128f", "SLH-DSA-SHAKE-128s",
	"SLH-DSA-SHAKE-192f", "SLH-DSA-SHAKE-192s", "SLH-DSA-SHAKE-256f", "SLH-DSA-SHAKE-256s",
}

func pqcTestSchema(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func stringValidatorsAccept(validators []validator.String, value string) bool {
	for _, v := range validators {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("attr"), ConfigValue: types.StringValue(value)}, &resp)
		if resp.Diagnostics.HasError() {
			return false
		}
	}
	return true
}

func listValidatorsAccept(validators []validator.List, value string) bool {
	list := types.ListValueMust(types.StringType, []attr.Value{types.StringValue(value)})
	for _, v := range validators {
		resp := validator.ListResponse{}
		v.ValidateList(context.Background(), validator.ListRequest{Path: path.Root("attr"), ConfigValue: list}, &resp)
		if resp.Diagnostics.HasError() {
			return false
		}
	}
	return true
}

func stringAttributeValidators(t *testing.T, attrs map[string]schema.Attribute, name string) []validator.String {
	t.Helper()
	a, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s is not a string attribute", name)
	}
	return a.Validators
}

func listAttributeValidators(t *testing.T, attrs map[string]schema.Attribute, name string) []validator.List {
	t.Helper()
	a, ok := attrs[name].(schema.ListAttribute)
	if !ok {
		t.Fatalf("%s is not a list attribute", name)
	}
	return a.Validators
}

func nestedBlockAttributes(t *testing.T, s schema.Schema, name string) map[string]schema.Attribute {
	t.Helper()
	b, ok := s.Blocks[name].(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("%s is not a single nested block", name)
	}
	return b.Attributes
}

func TestPqcAlgorithmListsMatchInfisicalCore(t *testing.T) {
	if !slices.Equal(PQC_ALGORITHMS, infisicalCorePqcAlgorithms) {
		t.Errorf("PQC_ALGORITHMS = %v, want %v", PQC_ALGORITHMS, infisicalCorePqcAlgorithms)
	}
}

func TestPqcAlgorithmsAcceptedPerResource(t *testing.T) {
	certSchema := pqcTestSchema(t, NewCertManagerCertificateResource())
	profileDefaults := nestedBlockAttributes(t, pqcTestSchema(t, NewCertManagerCertificateProfileResource()), "defaults")
	policyAlgorithms := nestedBlockAttributes(t, pqcTestSchema(t, NewCertManagerCertificatePolicyResource()), "algorithms")
	caSchema := pqcTestSchema(t, NewCertManagerInternalCAResource())

	stringChecks := map[string][]validator.String{
		"infisical_cert_manager_certificate.key_algorithm":                        stringAttributeValidators(t, certSchema.Attributes, "key_algorithm"),
		"infisical_cert_manager_certificate.signature_algorithm":                  stringAttributeValidators(t, certSchema.Attributes, "signature_algorithm"),
		"infisical_cert_manager_certificate_profile.defaults.key_algorithm":       stringAttributeValidators(t, profileDefaults, "key_algorithm"),
		"infisical_cert_manager_certificate_profile.defaults.signature_algorithm": stringAttributeValidators(t, profileDefaults, "signature_algorithm"),
	}
	for name, validators := range stringChecks {
		for _, alg := range infisicalCorePqcAlgorithms {
			if !stringValidatorsAccept(validators, alg) {
				t.Errorf("%s rejects %s", name, alg)
			}
		}
		if stringValidatorsAccept(validators, "ML-DSA-128") {
			t.Errorf("%s accepts an unknown algorithm", name)
		}
	}

	listChecks := map[string][]validator.List{
		"infisical_cert_manager_certificate_policy.algorithms.signature":     listAttributeValidators(t, policyAlgorithms, "signature"),
		"infisical_cert_manager_certificate_policy.algorithms.key_algorithm": listAttributeValidators(t, policyAlgorithms, "key_algorithm"),
	}
	for name, validators := range listChecks {
		for _, alg := range infisicalCorePqcAlgorithms {
			if !listValidatorsAccept(validators, alg) {
				t.Errorf("%s rejects %s", name, alg)
			}
		}
		// Policies use their own names for classical algorithms, so the certificate names must not pass.
		if listValidatorsAccept(validators, "RSA_2048") || listValidatorsAccept(validators, "RSA-SHA256") {
			t.Errorf("%s accepts a certificate-style classical algorithm name", name)
		}
	}

	caValidators := stringAttributeValidators(t, caSchema.Attributes, "key_algorithm")
	for _, alg := range ML_DSA_ALGORITHMS {
		if !stringValidatorsAccept(caValidators, alg) {
			t.Errorf("infisical_cert_manager_internal_ca.key_algorithm rejects %s", alg)
		}
	}
	// Infisical core refuses SLH-DSA keys for internal CAs ("SLH-DSA algorithms are not currently supported for CA creation").
	for _, alg := range SLH_DSA_ALGORITHMS {
		if stringValidatorsAccept(caValidators, alg) {
			t.Errorf("infisical_cert_manager_internal_ca.key_algorithm accepts %s, which the API refuses", alg)
		}
	}
}

func newPqcTestClient(t *testing.T, handler http.HandlerFunc) *infisical.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}
}

func objectState(t *testing.T, s schema.Schema, values map[string]tftypes.Value) tfsdk.State {
	t.Helper()
	objectType, ok := s.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not an object")
	}
	all := map[string]tftypes.Value{}
	for name, attrType := range objectType.AttributeTypes {
		if v, ok := values[name]; ok {
			all[name] = v
			continue
		}
		all[name] = tftypes.NewValue(attrType, nil)
	}
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(objectType, all)}
}

func pqcCAModel(keyAlgorithm string) certManagerInternalCAResourceModel {
	return certManagerInternalCAResourceModel{
		Id:           types.StringUnknown(),
		Type:         types.StringValue(CA_TYPE_ROOT),
		Name:         types.StringValue("pqc-root"),
		CommonName:   types.StringValue("PQC Root"),
		Organization: types.StringNull(),
		OU:           types.StringNull(),
		Country:      types.StringNull(),
		Province:     types.StringNull(),
		Locality:     types.StringNull(),
		KeyAlgorithm: types.StringValue(keyAlgorithm),
		Status:       types.StringValue(DEFAULT_CA_STATUS),
	}
}

func runCACreate(t *testing.T, client *infisical.Client, model certManagerInternalCAResourceModel) resource.CreateResponse {
	t.Helper()
	s := pqcTestSchema(t, NewCertManagerInternalCAResource())
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: s, Raw: state.Raw}
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}}
	r := &certManagerInternalCAResource{client: client}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: tfsdk.Config{Schema: s, Raw: state.Raw}}, &resp)
	return resp
}

func TestInternalCACreateAndReadKeepMlDsaKeyAlgorithm(t *testing.T) {
	var sentKeyAlgorithm string
	client := newPqcTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ca := `{"id":"ca-1","name":"pqc-root","status":"pending-certificate","configuration":{"type":"root","commonName":"PQC Root","keyAlgorithm":"ML-DSA-65"}}`
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/cert-manager/ca/internal":
			body, _ := io.ReadAll(req.Body)
			var parsed struct {
				Configuration struct {
					KeyAlgorithm string `json:"keyAlgorithm"`
				} `json:"configuration"`
			}
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatal(err)
			}
			sentKeyAlgorithm = parsed.Configuration.KeyAlgorithm
			_, _ = io.WriteString(w, ca)
		case "GET /api/v1/cert-manager/ca/internal/ca-1":
			_, _ = io.WriteString(w, ca)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})

	resp := runCACreate(t, client, pqcCAModel("ML-DSA-65"))
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if sentKeyAlgorithm != "ML-DSA-65" {
		t.Errorf("create sent keyAlgorithm %q, want ML-DSA-65", sentKeyAlgorithm)
	}

	readResp := resource.ReadResponse{State: resp.State}
	(&certManagerInternalCAResource{client: client}).Read(context.Background(), resource.ReadRequest{State: resp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	var got certManagerInternalCAResourceModel
	readResp.State.Get(context.Background(), &got)
	if got.KeyAlgorithm.ValueString() != "ML-DSA-65" || got.Status.ValueString() != "active" {
		t.Errorf("unexpected state after read: key_algorithm=%s status=%s", got.KeyAlgorithm, got.Status)
	}
}

func TestInternalCACreateSurfacesPqcLicenseError(t *testing.T) {
	// The body Infisical core returns from validatePqcLicense when the plan lacks pkiPqc.
	const licenseMessage = "Your license does not include PQC algorithms. Please upgrade to the Enterprise plan to use a PQC algorithm."
	client := newPqcTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"reqId":"req-1","statusCode":400,"message":"`+licenseMessage+`","error":"BadRequest"}`)
	})

	resp := runCACreate(t, client, pqcCAModel("ML-DSA-44"))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, licenseMessage) {
		t.Errorf("error detail does not carry the API's license message: %s", detail)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a refused create must not write state")
	}
}

func TestCertReadKeepsPqcAlgorithms(t *testing.T) {
	cases := []struct{ key, signature string }{
		{"ML-DSA-65", "ML-DSA-65"},
		{"ML-DSA-44", "ML-DSA-65"},
		{"SLH-DSA-SHAKE-256s", "RSA-SHA512"},
	}
	for _, c := range cases {
		t.Run(c.key+"_"+c.signature, func(t *testing.T) {
			r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch req.Method + " " + req.URL.Path {
				case "GET /api/v1/cert-manager/certificates/cert-old":
					details := strings.Replace(certDetailsJSON("cert-old", "serial-old"), `"keyAlgorithm":"RSA_2048","signatureAlgorithm":"RSA-SHA256"`,
						`"keyAlgorithm":"`+c.key+`","signatureAlgorithm":"`+c.signature+`"`, 1)
					_, _ = io.WriteString(w, details)
				case "GET /api/v1/cert-manager/certificates/cert-old/bundle":
					_, _ = io.WriteString(w, `{"certificate":"CERT","certificateChain":"CHAIN","privateKey":"KEY","serialNumber":"serial-old"}`)
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					http.NotFound(w, req)
				}
			})
			state := fieldCertModel()
			state.KeyAlgorithm = types.StringValue(c.key)
			state.SignatureAlgorithm = types.StringValue(c.signature)
			resp := runRead(t, r, state)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			got := stateModel(t, resp.State)
			if got.KeyAlgorithm.ValueString() != c.key || got.SignatureAlgorithm.ValueString() != c.signature {
				t.Errorf("read changed the algorithms to %s / %s", got.KeyAlgorithm, got.SignatureAlgorithm)
			}
		})
	}
}

func TestCertificatePolicyReadKeepsPqcAlgorithms(t *testing.T) {
	client := newPqcTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method+" "+req.URL.Path != "GET /api/v1/cert-manager/certificate-policies/policy-1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"certificatePolicy":{"id":"policy-1","projectId":"project-1","name":"pqc-policy","algorithms":{"signature":["ML-DSA-65","SHA256-RSA"],"keyAlgorithm":["ML-DSA-65","SLH-DSA-SHA2-128f","RSA-2048"]}}}`)
	})

	s := pqcTestSchema(t, NewCertManagerCertificatePolicyResource())
	state := objectState(t, s, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "policy-1")})
	resp := resource.ReadResponse{State: state}
	(&certManagerCertificatePolicyResource{client: client}).Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var signature, keyAlgorithm []string
	resp.Diagnostics.Append(resp.State.GetAttribute(context.Background(), path.Root("algorithms").AtName("signature"), &signature)...)
	resp.Diagnostics.Append(resp.State.GetAttribute(context.Background(), path.Root("algorithms").AtName("key_algorithm"), &keyAlgorithm)...)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !slices.Equal(signature, []string{"ML-DSA-65", "SHA256-RSA"}) || !slices.Equal(keyAlgorithm, []string{"ML-DSA-65", "SLH-DSA-SHA2-128f", "RSA-2048"}) {
		t.Errorf("read changed the algorithm lists: signature=%v key_algorithm=%v", signature, keyAlgorithm)
	}
}

func TestPkiResourcesReadRemovesStateWithEmptyId(t *testing.T) {
	client := newPqcTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		http.NotFound(w, req)
	})
	resources := map[string]resource.Resource{
		"infisical_cert_manager_internal_ca":         &certManagerInternalCAResource{client: client},
		"infisical_cert_manager_certificate":         &certManagerCertificateResource{client: client},
		"infisical_cert_manager_certificate_profile": &certManagerCertificateProfileResource{client: client},
		"infisical_cert_manager_certificate_policy":  &certManagerCertificatePolicyResource{client: client},
	}
	for name, r := range resources {
		t.Run(name, func(t *testing.T) {
			state := objectState(t, pqcTestSchema(t, r), map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "")})
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.State.Raw.IsNull() {
				t.Error("a state with an empty id must be removed")
			}
		})
	}
}
