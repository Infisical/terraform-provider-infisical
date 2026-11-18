package resource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testAwsKmsID           = "kms-1"
	testAwsCredentialsHash = "hash-1"
)

// fakeAwsKmsBackend answers GET /api/v1/external-kms/aws/:id like the API: the sanitized key with
// externalKms.credentialsHash, externalKms.hasLegacyExternalId only when the server sends it, and a 404
// for an unknown ID.
type fakeAwsKmsBackend struct {
	t *testing.T
	// nil leaves hasLegacyExternalId out of the response, like a server from before the field existed.
	hasLegacyExternalId *bool
	requests            int
}

func (b *fakeAwsKmsBackend) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		b.requests++
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v1/external-kms/aws/{id}", func(w http.ResponseWriter, r *http.Request) {
		b.requests++
		w.Header().Set("Content-Type", "application/json")
		if r.PathValue("id") != testAwsKmsID {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"statusCode": http.StatusNotFound, "message": "External KMS not found"})
			return
		}

		externalKms := map[string]any{
			"id":              "ext-1",
			"provider":        "aws",
			"status":          "active",
			"credentialsHash": testAwsCredentialsHash,
			"configuration": map[string]any{
				"credential": map[string]any{"type": "assume-role"},
				"awsRegion":  "us-east-1",
			},
		}
		if b.hasLegacyExternalId != nil {
			externalKms["hasLegacyExternalId"] = *b.hasLegacyExternalId
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"id":          testAwsKmsID,
			"name":        "aws-kms",
			"description": "",
			"externalKms": externalKms,
		}); err != nil {
			b.t.Error(err)
		}
	})
	return mux
}

func newAwsKmsTestResource(t *testing.T, backend *fakeAwsKmsBackend) *ExternalKmsBaseResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)

	r, ok := NewExternalKmsAwsResource().(*ExternalKmsBaseResource)
	if !ok {
		t.Fatal("NewExternalKmsAwsResource did not return *ExternalKmsBaseResource")
	}
	r.client = &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}
	return r
}

func awsKmsTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewExternalKmsAwsResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// The state after an apply of an assume-role key, with the credentials hash the fake backend returns.
func awsKmsTestState(t *testing.T, id string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	s := awsKmsTestSchema(t)

	configType, ok := s.Attributes["configuration"].GetType().(types.ObjectType)
	if !ok {
		t.Fatal("configuration is not an object type")
	}
	configuration, diags := types.ObjectValueFrom(ctx, configType.AttrTypes, ExternalKmsAwsConfigurationModel{
		AwsRegion:   types.StringValue("us-east-1"),
		Type:        types.StringValue(AwsExternalKmsAssumeRoleType),
		AwsKmsKeyId: types.StringValue("key-1"),
		Credential: ExternalKmsAwsCredentialsModel{
			AccessKeyId:     types.StringNull(),
			SecretAccessKey: types.StringNull(),
			RoleARN:         types.StringValue("arn:aws:iam::123456789012:role/infisical"),
			RoleExternalId:  types.StringValue("legacy-external-id"),
		},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}

	state := tfsdk.State{Schema: s}
	if diags := state.Set(ctx, &ExternalKmsBaseResourceModel{
		ID:              types.StringValue(id),
		Name:            types.StringValue("aws-kms"),
		Description:     types.StringNull(),
		Configuration:   configuration,
		CredentialsHash: types.StringValue(testAwsCredentialsHash),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func readAwsKms(t *testing.T, r *ExternalKmsBaseResource, state tfsdk.State) resource.ReadResponse {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp
}

func TestExternalKmsAwsReadLegacyExternalIdWarning(t *testing.T) {
	legacy, current := true, false
	cases := []struct {
		name                string
		hasLegacyExternalId *bool
		wantWarning         bool
	}{
		{name: "legacy external ID", hasLegacyExternalId: &legacy, wantWarning: true},
		{name: "organization ID", hasLegacyExternalId: &current, wantWarning: false},
		{name: "server without the field", hasLegacyExternalId: nil, wantWarning: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newAwsKmsTestResource(t, &fakeAwsKmsBackend{t: t, hasLegacyExternalId: tc.hasLegacyExternalId})
			resp := readAwsKms(t, r, awsKmsTestState(t, testAwsKmsID))

			warnings := resp.Diagnostics.Warnings()
			if !tc.wantWarning {
				if len(warnings) != 0 {
					t.Fatalf("expected no warnings, got %v", warnings)
				}
				return
			}
			if len(warnings) != 1 {
				t.Fatalf("expected one warning, got %v", warnings)
			}
			if warnings[0].Summary() != "External KMS uses a legacy external ID" {
				t.Errorf("unexpected warning summary %q", warnings[0].Summary())
			}
			if warnings[0].Detail() != r.LegacyExternalIdWarning {
				t.Errorf("unexpected warning detail %q", warnings[0].Detail())
			}

			// The warning must not change state, or the next plan would show a diff.
			var got ExternalKmsBaseResourceModel
			if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
				t.Fatal(diags)
			}
			if got.ID.ValueString() != testAwsKmsID || got.CredentialsHash.ValueString() != testAwsCredentialsHash {
				t.Errorf("unexpected state after read: %+v", got)
			}
		})
	}
}

func TestExternalKmsAwsReadRemovesResource(t *testing.T) {
	cases := []struct {
		name         string
		id           string
		wantRequests int
	}{
		// Read must not call the API without an ID.
		{name: "empty ID", id: "", wantRequests: 0},
		{name: "not found", id: "missing", wantRequests: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := &fakeAwsKmsBackend{t: t}
			r := newAwsKmsTestResource(t, backend)
			resp := readAwsKms(t, r, awsKmsTestState(t, tc.id))
			if !resp.State.Raw.IsNull() {
				t.Fatal("expected the resource to be removed from state")
			}
			if backend.requests != tc.wantRequests {
				t.Errorf("expected %d API requests, got %d", tc.wantRequests, backend.requests)
			}
		})
	}
}
