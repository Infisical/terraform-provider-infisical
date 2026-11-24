package resource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	testHsmConnectorID  = "0b6f4a52-7f1e-4c39-9a51-6f6d2a9e3c11"
	testAppConnectionID = "5d2c8e17-3a4b-4f60-8c2d-9e1f0a7b6c55"
	testInternalCAID    = "3f091b42-1661-41a1-b647-98c6a45af047"
)

// Fake of the internal CA routes in Infisical core. The create response and GET by ID merge the
// key reference that describeKey returns (certificate-authority-service.ts). A server older than
// v0.161.13 strips keySource as an unknown key and returns no keySource at all, and from v0.161.13
// to v0.166.8 only GET by ID returns keySource, not the create response.
type fakeInternalCABackend struct {
	t                    *testing.T
	oldServer            bool
	createOmitsKeySource bool
	failDelete           bool
	failGet              bool
	cas                  map[string]infisical.CertificateAuthority
	lastCreated          map[string]any
	deletes              int
	patches              int
	requests             int
}

func newFakeInternalCABackend(t *testing.T) *fakeInternalCABackend {
	return &fakeInternalCABackend{t: t, cas: map[string]infisical.CertificateAuthority{}}
}

func (b *fakeInternalCABackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeInternalCABackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/cert-manager/ca/internal", func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			b.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			return
		}
		b.lastCreated = raw
		encoded, _ := json.Marshal(raw)
		var request infisical.CreateInternalCARequest
		if err := json.Unmarshal(encoded, &request); err != nil {
			b.t.Fatal(err)
		}

		configuration := request.Configuration
		if b.oldServer {
			configuration.KeySource = ""
			configuration.HsmConnectorId = ""
			configuration.Region = ""
		} else if configuration.KeySource == "" {
			configuration.KeySource = CA_KEY_SOURCE_INFISICAL
		}
		if configuration.KeySource == CA_KEY_SOURCE_HSM && configuration.HsmConnectorId == "" {
			b.writeJSON(w, http.StatusBadRequest, map[string]any{"message": "An HSM Connector is required when the key source is HSM."})
			return
		}
		if configuration.KeySource != CA_KEY_SOURCE_AWS_KMS {
			configuration.AppConnectionId = ""
		}

		ca := infisical.CertificateAuthority{
			Id:            testInternalCAID,
			Name:          request.Name,
			Status:        request.Status,
			Type:          "internal",
			Configuration: configuration,
		}
		b.cas[ca.Id] = ca
		if b.createOmitsKeySource {
			ca.Configuration.KeySource = ""
			ca.Configuration.HsmConnectorId = ""
		}
		b.writeJSON(w, http.StatusOK, ca)
	})

	mux.HandleFunc("GET /api/v1/cert-manager/ca/internal/{id}", func(w http.ResponseWriter, r *http.Request) {
		if b.failGet {
			b.writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Something went wrong"})
			return
		}
		ca, ok := b.cas[r.PathValue("id")]
		if !ok {
			b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "CA not found"})
			return
		}
		b.writeJSON(w, http.StatusOK, ca)
	})

	mux.HandleFunc("PATCH /api/v1/cert-manager/ca/internal/{id}", func(w http.ResponseWriter, r *http.Request) {
		b.patches++
		ca, ok := b.cas[r.PathValue("id")]
		if !ok {
			b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "CA not found"})
			return
		}
		var body struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.t.Fatal(err)
		}
		ca.Name, ca.Status = body.Name, body.Status
		b.cas[ca.Id] = ca
		b.writeJSON(w, http.StatusOK, ca)
	})

	mux.HandleFunc("DELETE /api/v1/cert-manager/ca/internal/{id}", func(w http.ResponseWriter, r *http.Request) {
		b.deletes++
		if b.failDelete {
			b.writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Something went wrong"})
			return
		}
		ca, ok := b.cas[r.PathValue("id")]
		if !ok {
			b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "CA not found"})
			return
		}
		delete(b.cas, ca.Id)
		b.writeJSON(w, http.StatusOK, ca)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.requests++
		mux.ServeHTTP(w, r)
	})
}

func newInternalCATestResource(t *testing.T, backend *fakeInternalCABackend) *certManagerInternalCAResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &certManagerInternalCAResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func internalCATestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewCertManagerInternalCAResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func internalCAModel(keySource string, set func(*certManagerInternalCAResourceModel)) certManagerInternalCAResourceModel {
	model := certManagerInternalCAResourceModel{
		Id:              types.StringUnknown(),
		Type:            types.StringValue(CA_TYPE_ROOT),
		Name:            types.StringValue("root-ca"),
		CommonName:      types.StringValue("Root CA"),
		Organization:    types.StringNull(),
		OU:              types.StringNull(),
		Country:         types.StringNull(),
		Province:        types.StringNull(),
		Locality:        types.StringNull(),
		KeyAlgorithm:    types.StringValue(DEFAULT_CA_KEY_ALGORITHM),
		Status:          types.StringValue(DEFAULT_CA_STATUS),
		KeySource:       types.StringValue(keySource),
		HsmConnectorId:  types.StringNull(),
		AppConnectionId: types.StringNull(),
		Region:          types.StringNull(),
	}
	if set != nil {
		set(&model)
	}
	return model
}

func internalCAPlan(t *testing.T, s schema.Schema, model certManagerInternalCAResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func internalCAStateModel(t *testing.T, state tfsdk.State) certManagerInternalCAResourceModel {
	t.Helper()
	var model certManagerInternalCAResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func createInternalCA(t *testing.T, r *certManagerInternalCAResource, plan tfsdk.Plan) resource.CreateResponse {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: tftypes.NewValue(plan.Schema.Type().TerraformType(context.Background()), nil)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	return resp
}

func readInternalCA(t *testing.T, r *certManagerInternalCAResource, state tfsdk.State) resource.ReadResponse {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp
}

func createdConfiguration(t *testing.T, backend *fakeInternalCABackend) map[string]any {
	t.Helper()
	configuration, ok := backend.lastCreated["configuration"].(map[string]any)
	if !ok {
		t.Fatalf("create request has no configuration: %v", backend.lastCreated)
	}
	return configuration
}

func TestInternalCAHsmLifecycle(t *testing.T) {
	s := internalCATestSchema(t)
	backend := newFakeInternalCABackend(t)
	r := newInternalCATestResource(t, backend)

	created := createInternalCA(t, r, internalCAPlan(t, s, internalCAModel(CA_KEY_SOURCE_HSM, func(m *certManagerInternalCAResourceModel) {
		m.HsmConnectorId = types.StringValue(testHsmConnectorID)
	})))
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}

	configuration := createdConfiguration(t, backend)
	if configuration["keySource"] != CA_KEY_SOURCE_HSM || configuration["hsmConnectorId"] != testHsmConnectorID {
		t.Errorf("expected keySource and hsmConnectorId in the create configuration, got %v", configuration)
	}
	for _, key := range []string{"appConnectionId", "region"} {
		if _, sent := configuration[key]; sent {
			t.Errorf("expected %s to be left out of an HSM create, got %v", key, configuration[key])
		}
	}

	state := readInternalCA(t, r, created.State).State
	model := internalCAStateModel(t, state)
	if model.Id.ValueString() != testInternalCAID || model.KeySource.ValueString() != CA_KEY_SOURCE_HSM ||
		model.HsmConnectorId.ValueString() != testHsmConnectorID || !model.AppConnectionId.IsNull() || !model.Region.IsNull() {
		t.Errorf("unexpected state after read: %+v", model)
	}

	deleteResp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if len(backend.cas) != 0 {
		t.Error("expected the CA to be deleted")
	}
}

func TestInternalCAAwsKmsCreateAndRead(t *testing.T) {
	s := internalCATestSchema(t)
	backend := newFakeInternalCABackend(t)
	r := newInternalCATestResource(t, backend)

	created := createInternalCA(t, r, internalCAPlan(t, s, internalCAModel(CA_KEY_SOURCE_AWS_KMS, func(m *certManagerInternalCAResourceModel) {
		m.AppConnectionId = types.StringValue(testAppConnectionID)
		m.Region = types.StringValue("us-east-1")
	})))
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}

	configuration := createdConfiguration(t, backend)
	if configuration["keySource"] != CA_KEY_SOURCE_AWS_KMS || configuration["appConnectionId"] != testAppConnectionID || configuration["region"] != "us-east-1" {
		t.Errorf("expected keySource, appConnectionId and region in the create configuration, got %v", configuration)
	}
	if _, sent := configuration["hsmConnectorId"]; sent {
		t.Errorf("expected hsmConnectorId to be left out of an AWS KMS create")
	}

	idOnly := internalCAPlan(t, s, internalCAModel("", func(m *certManagerInternalCAResourceModel) {
		m.Id = types.StringValue(testInternalCAID)
		m.KeySource = types.StringNull()
	}))
	model := internalCAStateModel(t, readInternalCA(t, r, tfsdk.State{Schema: s, Raw: idOnly.Raw}).State)
	if model.KeySource.ValueString() != CA_KEY_SOURCE_AWS_KMS || model.AppConnectionId.ValueString() != testAppConnectionID ||
		model.Region.ValueString() != "us-east-1" || !model.HsmConnectorId.IsNull() {
		t.Errorf("unexpected state after read: %+v", model)
	}
}

func TestInternalCAOmittedKeySourceComesFromTheCA(t *testing.T) {
	s := internalCATestSchema(t)
	backend := newFakeInternalCABackend(t)
	r := newInternalCATestResource(t, backend)

	created := createInternalCA(t, r, internalCAPlan(t, s, internalCAModel("", func(m *certManagerInternalCAResourceModel) {
		m.KeySource = types.StringUnknown()
		m.HsmConnectorId = types.StringUnknown()
		m.AppConnectionId = types.StringUnknown()
		m.Region = types.StringUnknown()
	})))
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	if _, sent := createdConfiguration(t, backend)["keySource"]; sent {
		t.Error("expected keySource to be left out of the create request so Infisical applies its default")
	}
	model := internalCAStateModel(t, created.State)
	if model.KeySource.ValueString() != CA_KEY_SOURCE_INFISICAL || !model.HsmConnectorId.IsNull() || !model.AppConnectionId.IsNull() || !model.Region.IsNull() {
		t.Errorf("expected the key source attributes to be filled from the CA, got %+v", model)
	}
}

func TestInternalCAConfirmsKeySourceWhenCreateResponseOmitsIt(t *testing.T) {
	s := internalCATestSchema(t)
	backend := newFakeInternalCABackend(t)
	backend.createOmitsKeySource = true
	r := newInternalCATestResource(t, backend)

	created := createInternalCA(t, r, internalCAPlan(t, s, internalCAModel(CA_KEY_SOURCE_HSM, func(m *certManagerInternalCAResourceModel) {
		m.HsmConnectorId = types.StringValue(testHsmConnectorID)
	})))
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	if backend.deletes != 0 || len(backend.cas) != 1 {
		t.Fatalf("expected the HSM CA to be kept, got %d deletes", backend.deletes)
	}
	model := internalCAStateModel(t, created.State)
	if model.KeySource.ValueString() != CA_KEY_SOURCE_HSM || model.HsmConnectorId.ValueString() != testHsmConnectorID {
		t.Errorf("unexpected state after create: %+v", model)
	}
}

func TestInternalCAKeepsCAWhenKeySourceCannotBeConfirmed(t *testing.T) {
	s := internalCATestSchema(t)
	backend := newFakeInternalCABackend(t)
	backend.createOmitsKeySource = true
	backend.failGet = true
	r := newInternalCATestResource(t, backend)

	created := createInternalCA(t, r, internalCAPlan(t, s, internalCAModel(CA_KEY_SOURCE_HSM, func(m *certManagerInternalCAResourceModel) {
		m.HsmConnectorId = types.StringValue(testHsmConnectorID)
	})))
	if !created.Diagnostics.HasError() || backend.deletes != 0 {
		t.Fatalf("expected an error and no delete, got %d deletes and %v", backend.deletes, created.Diagnostics)
	}
	if model := internalCAStateModel(t, created.State); model.Id.ValueString() != testInternalCAID {
		t.Errorf("expected the CA to be tracked in state, got %+v", model)
	}
}

func TestInternalCADefaultKeySourceOnOldServer(t *testing.T) {
	s := internalCATestSchema(t)
	backend := newFakeInternalCABackend(t)
	backend.oldServer = true
	r := newInternalCATestResource(t, backend)

	created := createInternalCA(t, r, internalCAPlan(t, s, internalCAModel(CA_KEY_SOURCE_INFISICAL, nil)))
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	model := internalCAStateModel(t, readInternalCA(t, r, created.State).State)
	if model.KeySource.ValueString() != CA_KEY_SOURCE_INFISICAL {
		t.Errorf("expected a CA without keySource in the response to read as infisical, got %q", model.KeySource.ValueString())
	}
}

func TestInternalCAOldServerIgnoresKeySource(t *testing.T) {
	s := internalCATestSchema(t)
	hsmPlan := internalCAModel(CA_KEY_SOURCE_HSM, func(m *certManagerInternalCAResourceModel) {
		m.HsmConnectorId = types.StringValue(testHsmConnectorID)
	})

	t.Run("deletes the CA and fails", func(t *testing.T) {
		backend := newFakeInternalCABackend(t)
		backend.oldServer = true
		r := newInternalCATestResource(t, backend)

		created := createInternalCA(t, r, internalCAPlan(t, s, hsmPlan))
		if !created.Diagnostics.HasError() || !strings.Contains(created.Diagnostics.Errors()[0].Detail(), CA_KEY_SOURCE_MIN_VERSIONS[CA_KEY_SOURCE_HSM]) {
			t.Fatalf("expected an error naming the minimum version, got %v", created.Diagnostics)
		}
		if len(backend.cas) != 0 {
			t.Error("expected the CA created with an Infisical key to be deleted again")
		}
		if !created.State.Raw.IsNull() {
			t.Error("expected nothing to be saved in state")
		}
	})

	t.Run("keeps the CA in state when the delete fails", func(t *testing.T) {
		backend := newFakeInternalCABackend(t)
		backend.oldServer = true
		backend.failDelete = true
		r := newInternalCATestResource(t, backend)

		created := createInternalCA(t, r, internalCAPlan(t, s, hsmPlan))
		if !created.Diagnostics.HasError() || backend.deletes != 1 {
			t.Fatalf("expected one delete attempt and an error, got %d deletes and %v", backend.deletes, created.Diagnostics)
		}
		if model := internalCAStateModel(t, created.State); model.Id.ValueString() != testInternalCAID ||
			model.KeySource.ValueString() != CA_KEY_SOURCE_INFISICAL || !model.HsmConnectorId.IsNull() {
			t.Errorf("expected the CA to be tracked in state with the key source Infisical reported, got %+v", model)
		}
	})
}

func TestInternalCAReadRemovesMissingCA(t *testing.T) {
	s := internalCATestSchema(t)

	for name, tc := range map[string]struct {
		id       string
		requests int
	}{"empty id": {id: "", requests: 0}, "deleted outside Terraform": {id: testInternalCAID, requests: 1}} {
		t.Run(name, func(t *testing.T) {
			backend := newFakeInternalCABackend(t)
			r := newInternalCATestResource(t, backend)
			plan := internalCAPlan(t, s, internalCAModel(CA_KEY_SOURCE_INFISICAL, func(m *certManagerInternalCAResourceModel) {
				m.Id = types.StringValue(tc.id)
			}))
			resp := readInternalCA(t, r, tfsdk.State{Schema: s, Raw: plan.Raw})
			if !resp.State.Raw.IsNull() {
				t.Error("expected the CA to be removed from state")
			}
			if backend.requests != tc.requests {
				t.Errorf("expected %d requests, got %d", tc.requests, backend.requests)
			}
		})
	}
}

func TestInternalCADeleteTreatsNotFoundAsSuccess(t *testing.T) {
	s := internalCATestSchema(t)
	r := newInternalCATestResource(t, newFakeInternalCABackend(t))
	plan := internalCAPlan(t, s, internalCAModel(CA_KEY_SOURCE_INFISICAL, func(m *certManagerInternalCAResourceModel) {
		m.Id = types.StringValue(testInternalCAID)
	}))
	state := tfsdk.State{Schema: s, Raw: plan.Raw}

	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

func TestInternalCAImport(t *testing.T) {
	s := internalCATestSchema(t)
	backend := newFakeInternalCABackend(t)
	r := newInternalCATestResource(t, backend)
	backend.cas[testInternalCAID] = infisical.CertificateAuthority{
		Id: testInternalCAID, Name: "root-ca", Status: "active", Type: "internal",
		Configuration: infisical.CertificateAuthorityConfiguration{
			Type: CA_TYPE_ROOT, CommonName: "Root CA", KeyAlgorithm: "EC_prime256v1",
			KeySource: CA_KEY_SOURCE_HSM, HsmConnectorId: testHsmConnectorID,
		},
	}
	emptyState := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}

	rejected := resource.ImportStateResponse{State: emptyState}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "root-ca"}, &rejected)
	if !rejected.Diagnostics.HasError() {
		t.Error("expected a non-UUID import ID to be rejected")
	}

	imported := resource.ImportStateResponse{State: emptyState}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: testInternalCAID}, &imported)
	if imported.Diagnostics.HasError() {
		t.Fatal(imported.Diagnostics)
	}
	model := internalCAStateModel(t, readInternalCA(t, r, imported.State).State)
	if model.KeySource.ValueString() != CA_KEY_SOURCE_HSM || model.HsmConnectorId.ValueString() != testHsmConnectorID ||
		model.KeyAlgorithm.ValueString() != "EC_prime256v1" || model.Name.ValueString() != "root-ca" {
		t.Errorf("unexpected state after import: %+v", model)
	}
}

func TestInternalCAValidateConfig(t *testing.T) {
	s := internalCATestSchema(t)
	r := &certManagerInternalCAResource{}

	cases := map[string]struct {
		model     certManagerInternalCAResourceModel
		errorPath string
	}{
		"infisical": {model: internalCAModel(CA_KEY_SOURCE_INFISICAL, nil)},
		"omitted key source": {model: internalCAModel("", func(m *certManagerInternalCAResourceModel) {
			m.KeySource = types.StringNull()
		})},
		"hsm": {model: internalCAModel(CA_KEY_SOURCE_HSM, func(m *certManagerInternalCAResourceModel) {
			m.HsmConnectorId = types.StringValue(testHsmConnectorID)
		})},
		"hsm with an unknown connector": {model: internalCAModel(CA_KEY_SOURCE_HSM, func(m *certManagerInternalCAResourceModel) {
			m.HsmConnectorId = types.StringUnknown()
		})},
		"hsm without a connector": {model: internalCAModel(CA_KEY_SOURCE_HSM, nil), errorPath: "hsm_connector_id"},
		"hsm with an unsupported key algorithm": {model: internalCAModel(CA_KEY_SOURCE_HSM, func(m *certManagerInternalCAResourceModel) {
			m.HsmConnectorId = types.StringValue(testHsmConnectorID)
			m.KeyAlgorithm = types.StringValue("RSA_3072")
		}), errorPath: "key_algorithm"},
		"connector without hsm": {model: internalCAModel(CA_KEY_SOURCE_INFISICAL, func(m *certManagerInternalCAResourceModel) {
			m.HsmConnectorId = types.StringValue(testHsmConnectorID)
		}), errorPath: "hsm_connector_id"},
		"aws kms": {model: internalCAModel(CA_KEY_SOURCE_AWS_KMS, func(m *certManagerInternalCAResourceModel) {
			m.AppConnectionId = types.StringValue(testAppConnectionID)
			m.Region = types.StringValue("us-east-1")
			m.KeyAlgorithm = types.StringValue("RSA_3072")
		})},
		"aws kms without a region": {model: internalCAModel(CA_KEY_SOURCE_AWS_KMS, func(m *certManagerInternalCAResourceModel) {
			m.AppConnectionId = types.StringValue(testAppConnectionID)
		}), errorPath: "region"},
		"aws kms without a connection": {model: internalCAModel(CA_KEY_SOURCE_AWS_KMS, func(m *certManagerInternalCAResourceModel) {
			m.Region = types.StringValue("us-east-1")
		}), errorPath: "app_connection_id"},
		"aws kms fields with hsm": {model: internalCAModel(CA_KEY_SOURCE_HSM, func(m *certManagerInternalCAResourceModel) {
			m.HsmConnectorId = types.StringValue(testHsmConnectorID)
			m.AppConnectionId = types.StringValue(testAppConnectionID)
		}), errorPath: "app_connection_id"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			plan := internalCAPlan(t, s, tc.model)
			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: plan.Raw}}, &resp)

			if tc.errorPath == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error, got %v", resp.Diagnostics)
				}
				return
			}
			if resp.Diagnostics.ErrorsCount() != 1 {
				t.Fatalf("expected one error on %s, got %v", tc.errorPath, resp.Diagnostics)
			}
			if got, ok := resp.Diagnostics.Errors()[0].(interface{ Path() path.Path }); !ok || !got.Path().Equal(path.Root(tc.errorPath)) {
				t.Errorf("expected the error on %s, got %v", tc.errorPath, resp.Diagnostics)
			}
		})
	}
}

// Builds the planned key source values the way the framework does: an omitted Computed attribute starts
// unknown, then the schema's plan modifiers run against the prior state (null when Terraform plans a create,
// including the create half of a replacement).
func planInternalCAKeySource(t *testing.T, s schema.Schema, prior *certManagerInternalCAResourceModel, planned *certManagerInternalCAResourceModel) bool {
	t.Helper()
	ctx := context.Background()
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	priorValues := certManagerInternalCAResourceModel{}
	if prior != nil {
		state = tfsdk.State{Schema: s, Raw: internalCAPlan(t, s, *prior).Raw}
		priorValues = *prior
	}
	plan := internalCAPlan(t, s, *planned)

	replace := false
	for _, field := range []struct {
		attribute string
		prior     types.String
		planned   *types.String
	}{
		{"key_source", priorValues.KeySource, &planned.KeySource},
		{"hsm_connector_id", priorValues.HsmConnectorId, &planned.HsmConnectorId},
		{"app_connection_id", priorValues.AppConnectionId, &planned.AppConnectionId},
		{"region", priorValues.Region, &planned.Region},
	} {
		stringAttribute, ok := s.Attributes[field.attribute].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s is not a string attribute", field.attribute)
		}
		config := *field.planned
		resp := planmodifier.StringResponse{PlanValue: config}
		if config.IsNull() {
			resp.PlanValue = types.StringUnknown()
		}
		for _, modifier := range stringAttribute.PlanModifiers {
			modifier.PlanModifyString(ctx, planmodifier.StringRequest{
				Path: path.Root(field.attribute), State: state, Plan: plan,
				StateValue: field.prior, ConfigValue: config, PlanValue: resp.PlanValue,
			}, &resp)
		}
		*field.planned = resp.PlanValue
		replace = replace || resp.RequiresReplace
	}
	return replace
}

func TestInternalCAImportedHSMCAWithKeySourceOmitted(t *testing.T) {
	ctx := context.Background()
	s := internalCATestSchema(t)
	backend := newFakeInternalCABackend(t)
	r := newInternalCATestResource(t, backend)
	backend.cas[testInternalCAID] = infisical.CertificateAuthority{
		Id: testInternalCAID, Name: "root-ca", Status: "active", Type: "internal",
		Configuration: infisical.CertificateAuthorityConfiguration{
			Type: CA_TYPE_ROOT, CommonName: "Root CA", KeyAlgorithm: "EC_prime256v1",
			KeySource: CA_KEY_SOURCE_HSM, HsmConnectorId: testHsmConnectorID,
		},
	}

	emptyState := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	imported := resource.ImportStateResponse{State: emptyState}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testInternalCAID}, &imported)
	if imported.Diagnostics.HasError() {
		t.Fatal(imported.Diagnostics)
	}
	state := readInternalCA(t, r, imported.State).State
	prior := internalCAStateModel(t, state)

	omitted := func(name, keyAlgorithm string) certManagerInternalCAResourceModel {
		return internalCAModel("", func(m *certManagerInternalCAResourceModel) {
			m.Id = types.StringValue(testInternalCAID)
			m.Name = types.StringValue(name)
			m.KeyAlgorithm = types.StringValue(keyAlgorithm)
			m.KeySource = types.StringNull()
		})
	}

	renamed := omitted("renamed", "EC_prime256v1")
	if planInternalCAKeySource(t, s, &prior, &renamed) {
		t.Fatal("expected an ordinary update to keep the CA when key_source is omitted")
	}
	if renamed.KeySource.ValueString() != CA_KEY_SOURCE_HSM || renamed.HsmConnectorId.ValueString() != testHsmConnectorID {
		t.Fatalf("expected the plan to keep the CA's key source, got %+v", renamed)
	}
	updated := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: internalCAPlan(t, s, renamed), State: state}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	if backend.patches != 1 || backend.cas[testInternalCAID].Configuration.KeySource != CA_KEY_SOURCE_HSM {
		t.Fatalf("expected an in-place update of the HSM CA, got %d patches and %+v", backend.patches, backend.cas[testInternalCAID])
	}
	if model := internalCAStateModel(t, updated.State); model.KeySource.ValueString() != CA_KEY_SOURCE_HSM || model.HsmConnectorId.ValueString() != testHsmConnectorID {
		t.Errorf("expected state to keep the CA's key source, got %+v", model)
	}

	replacement := omitted("renamed", "RSA_2048")
	planInternalCAKeySource(t, s, nil, &replacement)
	if !replacement.KeySource.IsUnknown() || !replacement.HsmConnectorId.IsUnknown() {
		t.Fatalf("expected the create half of the replacement to leave the key source to Infisical, got %+v", replacement)
	}
	delete(backend.cas, testInternalCAID)
	created := createInternalCA(t, r, internalCAPlan(t, s, replacement))
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	configuration := createdConfiguration(t, backend)
	for _, key := range []string{"keySource", "hsmConnectorId"} {
		if _, sent := configuration[key]; sent {
			t.Errorf("expected %s to be left out of the replacement request, got %v", key, configuration[key])
		}
	}
	if model := internalCAStateModel(t, created.State); model.KeySource.ValueString() != CA_KEY_SOURCE_INFISICAL || !model.HsmConnectorId.IsNull() {
		t.Errorf("expected the replacement CA to use Infisical's default key source, got %+v", model)
	}
}

func TestInternalCAKeySourceReplacement(t *testing.T) {
	ctx := context.Background()
	s := internalCATestSchema(t)
	existing := internalCAPlan(t, s, internalCAModel(CA_KEY_SOURCE_INFISICAL, func(m *certManagerInternalCAResourceModel) {
		m.Id = types.StringValue(testInternalCAID)
	}))

	cases := map[string]struct {
		state, config types.String
		plan          string
		replace       bool
	}{
		"state from a provider without these attributes, omitted in config": {state: types.StringNull(), config: types.StringNull(), plan: "null"},
		"state from a provider without these attributes, set in config":     {state: types.StringNull(), config: types.StringValue("x"), plan: "x", replace: true},
		"omitted in config keeps the value of the CA":                       {state: types.StringValue("a"), config: types.StringNull(), plan: "a"},
		"changed":                 {state: types.StringValue("a"), config: types.StringValue("b"), plan: "b", replace: true},
		"unchanged":               {state: types.StringValue("a"), config: types.StringValue("a"), plan: "a"},
		"unknown from a variable": {state: types.StringValue("a"), config: types.StringUnknown(), plan: "unknown", replace: true},
	}

	for _, attribute := range []string{"key_source", "hsm_connector_id", "app_connection_id", "region"} {
		stringAttribute, ok := s.Attributes[attribute].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s is not a string attribute", attribute)
		}
		for name, tc := range cases {
			t.Run(attribute+"/"+name, func(t *testing.T) {
				planValue := tc.config
				if tc.config.IsNull() {
					planValue = types.StringUnknown()
				}
				resp := planmodifier.StringResponse{PlanValue: planValue}
				for _, modifier := range stringAttribute.PlanModifiers {
					modifier.PlanModifyString(ctx, planmodifier.StringRequest{
						Path:        path.Root(attribute),
						State:       tfsdk.State{Schema: s, Raw: existing.Raw},
						Plan:        existing,
						StateValue:  tc.state,
						ConfigValue: tc.config,
						PlanValue:   resp.PlanValue,
					}, &resp)
				}
				got := resp.PlanValue.ValueString()
				if resp.PlanValue.IsUnknown() {
					got = "unknown"
				} else if resp.PlanValue.IsNull() {
					got = "null"
				}
				if got != tc.plan || resp.RequiresReplace != tc.replace {
					t.Errorf("expected plan %q replace=%v, got %q replace=%v", tc.plan, tc.replace, got, resp.RequiresReplace)
				}
			})
		}
	}
}
