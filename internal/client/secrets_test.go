package infisicalclient

import (
	"net/http"
	"testing"
)

// rawSecretsServer serves the list endpoint and records the include_imports query parameter the
// client sent.
func rawSecretsServer(t *testing.T, body string, includeImports *string) Client {
	t.Helper()

	respond := jsonResponse(http.StatusOK, body)
	return testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/secrets/raw" {
			t.Errorf("unexpected request path %s", r.URL.Path)
		}
		*includeImports = r.URL.Query().Get("include_imports")
		respond(w, r)
	})
}

func TestGetRawSecretsSendsIncludeImports(t *testing.T) {
	for _, include := range []bool{true, false} {
		var sent string
		client := rawSecretsServer(t, `{"secrets":[]}`, &sent)

		if _, err := client.GetRawSecrets("/", "dev", "project-id", include); err != nil {
			t.Fatalf("expected the secrets to be fetched, got: %v", err)
		}

		want := "false"
		if include {
			want = "true"
		}
		if sent != want {
			t.Errorf("includeImports=%t: expected include_imports=%s, got %q", include, want, sent)
		}
	}
}

// Mirrors the Infisical CLI: the folder's own secret wins over an import, and the last import wins
// over earlier ones.
func TestGetRawSecretsMergesImports(t *testing.T) {
	var sent string
	client := rawSecretsServer(t, `{
		"secrets": [{"secretKey":"SHARED","secretValue":"local"},{"secretKey":"LOCAL","secretValue":"local"}],
		"imports": [
			{"secretPath":"/a","environment":"dev","secrets":[{"secretKey":"SHARED","secretValue":"a"},{"secretKey":"FROM_A","secretValue":"a"},{"secretKey":"BOTH","secretValue":"a"}]},
			{"secretPath":"/b","environment":"dev","secrets":[{"secretKey":"BOTH","secretValue":"b"}]}
		]
	}`, &sent)

	secrets, err := client.GetRawSecrets("/c", "dev", "project-id", true)
	if err != nil {
		t.Fatalf("expected the secrets to be fetched, got: %v", err)
	}

	got := make(map[string]string, len(secrets))
	for _, secret := range secrets {
		if _, dup := got[secret.SecretKey]; dup {
			t.Errorf("secret %s returned more than once", secret.SecretKey)
		}
		got[secret.SecretKey] = secret.SecretValue
	}

	want := map[string]string{"SHARED": "local", "LOCAL": "local", "FROM_A": "a", "BOTH": "b"}
	if len(got) != len(want) {
		t.Fatalf("expected %d secrets, got %d: %v", len(want), len(got), got)
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("expected %s=%s, got %q", key, value, got[key])
		}
	}
}
