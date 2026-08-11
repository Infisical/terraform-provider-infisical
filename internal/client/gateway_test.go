package infisicalclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

// gatewayLookupServer stands in for the gateway list endpoint GetGatewayByName consults. Retries are
// deliberately not configured: these tests assert error plumbing, and a 5xx would otherwise be
// retried before the error surfaces.
func gatewayLookupServer(t *testing.T, list http.HandlerFunc) Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/gateways", list)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return Client{Config: Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}
}

func TestGetGatewayByNameResolves(t *testing.T) {
	client := gatewayLookupServer(t, jsonResponse(http.StatusOK,
		`[{"id":"11111111-1111-1111-1111-111111111111","name":"staging-gateway"},`+
			`{"id":"22222222-2222-2222-2222-222222222222","name":"prod-gateway"}]`))

	gateway, err := client.GetGatewayByName("prod-gateway")
	if err != nil {
		t.Fatalf("expected the gateway to resolve, got: %v", err)
	}
	if gateway.ID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("expected the matching gateway's ID, got %s", gateway.ID)
	}
	if gateway.Name != "prod-gateway" {
		t.Errorf("expected name prod-gateway, got %s", gateway.Name)
	}
}

// Names are matched exactly: the backend stores them in a plain (case-sensitive) column, so a
// near-miss is a different gateway, not the same one.
func TestGetGatewayByNameMatchesExactly(t *testing.T) {
	client := gatewayLookupServer(t, jsonResponse(http.StatusOK,
		`[{"id":"33333333-3333-3333-3333-333333333333","name":"Prod-Gateway"}]`))

	if _, err := client.GetGatewayByName("prod-gateway"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a case-mismatched name, got: %v", err)
	}
}

// The list arrived and holds no match, so absence is proven and ErrNotFound is warranted.
func TestGetGatewayByNameNotFound(t *testing.T) {
	cases := map[string]string{
		"no gateways at all":  `[]`,
		"only other gateways": `[{"id":"44444444-4444-4444-4444-444444444444","name":"other-gateway"}]`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			client := gatewayLookupServer(t, jsonResponse(http.StatusOK, body))

			if _, err := client.GetGatewayByName("prod-gateway"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("expected ErrNotFound when the list arrived without a match, got: %v", err)
			}
		})
	}
}

// A list that never arrived cannot prove the gateway's absence, so a failure must not be reported as
// ErrNotFound, and its cause must stay classifiable so a caller can tell a 403 from a 500.
func TestGetGatewayByNameListFailureIsNotAbsence(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"no permission to read gateways": {
			http.StatusForbidden,
			`{"statusCode":403,"message":"You do not have permission to list gateways","reqId":"req-1"}`,
		},
		"server error": {
			http.StatusInternalServerError,
			`{"statusCode":500,"message":"Something went wrong","reqId":"req-2"}`,
		},
		"endpoint absent on this instance": {
			http.StatusNotFound,
			`{"statusCode":404,"message":"Not Found","reqId":"req-3"}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := gatewayLookupServer(t, jsonResponse(tc.status, tc.body))

			_, err := client.GetGatewayByName("prod-gateway")
			if err == nil {
				t.Fatal("expected an error when the gateway list could not be retrieved")
			}
			if errors.Is(err, ErrNotFound) {
				t.Error("an unavailable list cannot prove absence, so ErrNotFound is wrong here")
			}

			apiErrors := collectAPIErrors(err)
			if len(apiErrors) == 0 {
				t.Fatalf("the underlying API failure must stay classifiable, got: %v", err)
			}
			if apiErrors[0].StatusCode != tc.status {
				t.Errorf("expected status %d to be discoverable, got %d", tc.status, apiErrors[0].StatusCode)
			}
		})
	}
}
