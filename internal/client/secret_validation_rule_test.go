package infisicalclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

// A 404 on update can mean the rule is gone or that a referenced field does not exist. The API
// message says which, so the client has to relay it instead of collapsing the status into ErrNotFound.
func TestUpdateSecretValidationRuleKeepsTheAPIMessage(t *testing.T) {
	const message = "Environment with slug production not found"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/secret-validation-rules/dynamic-secrets/rule-1" {
			t.Errorf("request = %s %s, want PATCH /api/v1/secret-validation-rules/dynamic-secrets/rule-1", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"message":"` + message + `","reqId":"req-1"}`)); err != nil {
			t.Errorf("writing the error body: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	client := Client{Config: Config{
		HostURL:    server.URL,
		HttpClient: resty.New().SetBaseURL(server.URL),
	}}

	_, err := client.UpdateSecretValidationRule(UpdateSecretValidationRuleRequest{
		Type: SecretValidationRuleTypeDynamicSecrets,
		ID:   "rule-1",
	})
	if err == nil {
		t.Fatal("UpdateSecretValidationRule() error = nil, want one")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("UpdateSecretValidationRule() error = %v, want it not to be ErrNotFound", err)
	}
	if got := err.Error(); !strings.Contains(got, message) {
		t.Errorf("UpdateSecretValidationRule() error = %q, want it to carry the API's message", got)
	}
}
