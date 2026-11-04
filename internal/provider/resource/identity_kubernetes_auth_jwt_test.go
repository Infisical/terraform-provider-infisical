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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const testKubernetesAuthIdentityID = "identity-1"

// Records the body of every create/update and answers like the API, which never returns the JWT.
type fakeKubernetesAuthBackend struct {
	t      *testing.T
	bodies []map[string]any
}

func (b *fakeKubernetesAuthBackend) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/v1/auth/kubernetes-auth/identities/"+testKubernetesAuthIdentityID) ||
			(r.Method != http.MethodPost && r.Method != http.MethodPatch) {
			b.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.t.Fatal(err)
		}
		b.bodies = append(b.bodies, body)

		host, _ := body["kubernetesHost"].(string)
		jwt, _ := body["tokenReviewerJwt"].(string)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"identityKubernetesAuth": map[string]any{
			"id":                  "auth-1",
			"identityId":          testKubernetesAuthIdentityID,
			"kubernetesHost":      host,
			"tokenReviewMode":     TOKEN_REVIEWER_MODE_API,
			"hasTokenReviewerJwt": jwt != "",
		}}); err != nil {
			b.t.Error(err)
		}
	})
}

// lastJwt reports whether the last request carried tokenReviewerJwt, and its value.
func (b *fakeKubernetesAuthBackend) lastJwt(t *testing.T) (any, bool) {
	t.Helper()
	if len(b.bodies) == 0 {
		t.Fatal("expected a request to the API")
	}
	value, present := b.bodies[len(b.bodies)-1]["tokenReviewerJwt"]
	return value, present
}

func newKubernetesAuthTestResource(t *testing.T) (*IdentityKubernetesAuthResource, *fakeKubernetesAuthBackend) {
	t.Helper()
	backend := &fakeKubernetesAuthBackend{t: t}
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &IdentityKubernetesAuthResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}, backend
}

// kubernetesAuthTestModel is a custom (no template) configuration with the JWT supplied only as
// token_reviewer_jwt_wo.
func kubernetesAuthTestModel(host string, version int64) IdentityKubernetesAuthResourceModel {
	model := nullKubernetesAuthModel()
	model.IdentityID = types.StringValue(testKubernetesAuthIdentityID)
	model.KubernetesHost = types.StringValue(host)
	model.TokenReviewerJWTWOVersion = types.Int64Value(version)
	model.TokenReviewerMode = types.StringValue(TOKEN_REVIEWER_MODE_API)
	return model
}

// kubernetesAuthPlanAndConfig builds the plan (write-only value always null) and the config
// (write-only value present) Terraform would send for model.
func kubernetesAuthPlanAndConfig(t *testing.T, s schema.Schema, model IdentityKubernetesAuthResourceModel, writeOnlyJwt string) (tfsdk.Plan, tfsdk.Config) {
	t.Helper()
	ctx := context.Background()

	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}

	model.TokenReviewerJWTWO = types.StringValue(writeOnlyJwt)
	holder := tfsdk.Plan{Schema: s}
	if diags := holder.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan, tfsdk.Config{Schema: s, Raw: holder.Raw}
}

func kubernetesAuthStateModel(t *testing.T, state tfsdk.State) IdentityKubernetesAuthResourceModel {
	t.Helper()
	var model IdentityKubernetesAuthResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	if !model.TokenReviewerJWT.IsNull() || !model.TokenReviewerJWTWO.IsNull() {
		t.Errorf("expected no JWT in state, got token_reviewer_jwt=%v token_reviewer_jwt_wo=%v", model.TokenReviewerJWT, model.TokenReviewerJWTWO)
	}
	return model
}

func TestIdentityKubernetesAuthWriteOnlyJwtLifecycle(t *testing.T) {
	ctx := context.Background()
	s := kubernetesAuthTestSchema(t)
	r, backend := newKubernetesAuthTestResource(t)

	createModel := kubernetesAuthTestModel("https://k8s-1.example.com", 1)
	createModel.ID = types.StringUnknown()
	createModel.GatewayID = types.StringUnknown()
	createModel.HasTokenReviewerJWT = types.BoolUnknown()
	createModel.HasTemplateSourcedTokenReviewerJWT = types.BoolUnknown()
	plan, config := kubernetesAuthPlanAndConfig(t, s, createModel, "eyJ.first")

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	if jwt, _ := backend.lastJwt(t); jwt != "eyJ.first" {
		t.Errorf("create: expected the write-only JWT to be sent, got %v", jwt)
	}
	state := createResp.State
	if created := kubernetesAuthStateModel(t, state); !created.HasTokenReviewerJWT.ValueBool() {
		t.Error("create: expected has_token_reviewer_jwt to be true")
	}

	steps := []struct {
		name    string
		host    string
		version int64
		jwt     string
	}{
		// The API rejects a connection change without the JWT, so it goes out even though the
		// version is unchanged.
		{"host change without a version bump", "https://k8s-2.example.com", 1, "eyJ.first"},
		{"rotation", "https://k8s-2.example.com", 2, "eyJ.second"},
	}
	for _, step := range steps {
		model := kubernetesAuthStateModel(t, state)
		model.KubernetesHost = types.StringValue(step.host)
		model.TokenReviewerJWTWOVersion = types.Int64Value(step.version)
		plan, config := kubernetesAuthPlanAndConfig(t, s, model, step.jwt)

		updateResp := resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{Plan: plan, Config: config, State: state}, &updateResp)
		if updateResp.Diagnostics.HasError() {
			t.Fatalf("%s: %v", step.name, updateResp.Diagnostics)
		}
		if jwt, present := backend.lastJwt(t); !present || jwt != step.jwt {
			t.Errorf("%s: expected tokenReviewerJwt %q to be sent, got present=%v value=%v", step.name, step.jwt, present, jwt)
		}
		if host := backend.bodies[len(backend.bodies)-1]["kubernetesHost"]; host != step.host {
			t.Errorf("%s: expected kubernetesHost %q, got %v", step.name, step.host, host)
		}
		state = updateResp.State
		kubernetesAuthStateModel(t, state)
	}
}

func TestIdentityKubernetesAuthUpdateEmptyJwtClearsIt(t *testing.T) {
	ctx := context.Background()
	s := kubernetesAuthTestSchema(t)
	r, backend := newKubernetesAuthTestResource(t)

	stateModel := kubernetesAuthTestModel("https://k8s-1.example.com", 1)
	stateModel.ID = types.StringValue("auth-1")
	stateModel.TokenReviewerJWTWOVersion = types.Int64Null()
	stateModel.TokenReviewerJWT = types.StringValue("eyJ.plain")
	stateModel.HasTokenReviewerJWT = types.BoolValue(true)
	stateModel.HasTemplateSourcedTokenReviewerJWT = types.BoolValue(false)
	state := tfsdk.State{Schema: s}
	if diags := state.Set(ctx, &stateModel); diags.HasError() {
		t.Fatal(diags)
	}

	planModel := stateModel
	planModel.TokenReviewerJWT = types.StringValue("")
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, &planModel); diags.HasError() {
		t.Fatal(diags)
	}

	updateResp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, Config: tfsdk.Config{Schema: s, Raw: plan.Raw}, State: state}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if jwt, present := backend.lastJwt(t); !present || jwt != nil {
		t.Errorf("expected tokenReviewerJwt to be sent as null, got present=%v value=%v", present, jwt)
	}
}
