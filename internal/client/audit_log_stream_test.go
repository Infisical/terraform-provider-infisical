package infisicalclient

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestAuditLogStreamCRUD(t *testing.T) {
	methods := make(map[string]int)
	bodies := make(map[string]map[string]any)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/api/v1/audit-log-streams/splunk/stream-1"
		if r.Method == http.MethodPost {
			wantPath = "/api/v1/audit-log-streams/splunk"
		}
		if r.URL.Path != wantPath {
			t.Errorf("%s path = %q, want %q", r.Method, r.URL.Path, wantPath)
		}
		methods[r.Method]++

		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			body := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			bodies[r.Method] = body
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"auditLogStream": map[string]any{
			"id":          "stream-1",
			"orgId":       "org-1",
			"provider":    "splunk",
			"streamMode":  "batch",
			"credentials": map[string]any{"hostname": "splunk.example.com", "port": 443},
			"filters":     map[string]any{"products": []string{"organization"}},
		}})
	}))
	t.Cleanup(server.Close)

	client := Client{Config: Config{HttpClient: resty.New().SetBaseURL(server.URL)}}
	credentials := map[string]any{"hostname": "splunk.example.com", "port": 443, "token": "secret"}

	created, err := client.CreateAuditLogStream(CreateAuditLogStreamRequest{
		Provider:    AuditLogStreamProviderSplunk,
		Credentials: credentials,
		Filters:     &AuditLogStreamFilters{Products: []string{"organization"}},
	})
	if err != nil || created.ID != "stream-1" || created.StreamMode != "batch" {
		t.Fatalf("CreateAuditLogStream() = %#v, %v", created, err)
	}
	if created.Filters == nil || len(created.Filters.Products) != 1 {
		t.Errorf("filters = %#v", created.Filters)
	}

	read, err := client.GetAuditLogStreamById(AuditLogStreamByIdRequest{ID: "stream-1", Provider: AuditLogStreamProviderSplunk})
	if err != nil || read.Credentials["hostname"] != "splunk.example.com" {
		t.Fatalf("GetAuditLogStreamById() = %#v, %v", read, err)
	}

	if _, err := client.UpdateAuditLogStream(UpdateAuditLogStreamRequest{
		ID:          "stream-1",
		Provider:    AuditLogStreamProviderSplunk,
		Credentials: credentials,
	}); err != nil {
		t.Fatalf("UpdateAuditLogStream() error = %v", err)
	}

	if err := client.DeleteAuditLogStream(AuditLogStreamByIdRequest{ID: "stream-1", Provider: AuditLogStreamProviderSplunk}); err != nil {
		t.Fatalf("DeleteAuditLogStream() error = %v", err)
	}

	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodPatch, http.MethodDelete} {
		if methods[method] != 1 {
			t.Errorf("%s requests = %d, want 1", method, methods[method])
		}
	}

	// An absent filter must be sent as an explicit null, which is how the API clears scoping;
	// omitting the key leaves the existing filter in place.
	filters, present := bodies[http.MethodPatch]["filters"]
	if !present || filters != nil {
		t.Errorf("PATCH filters = %#v (present %t), want explicit null", filters, present)
	}
}

func TestAuditLogStreamNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)
	client := Client{Config: Config{HttpClient: resty.New().SetBaseURL(server.URL)}}
	request := AuditLogStreamByIdRequest{ID: "missing", Provider: AuditLogStreamProviderDatadog}

	if _, err := client.GetAuditLogStreamById(request); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetAuditLogStreamById() error = %v, want ErrNotFound", err)
	}
	if err := client.DeleteAuditLogStream(request); !errors.Is(err, ErrNotFound) {
		t.Errorf("DeleteAuditLogStream() error = %v, want ErrNotFound", err)
	}
	if _, err := client.UpdateAuditLogStream(UpdateAuditLogStreamRequest{ID: "missing", Provider: AuditLogStreamProviderDatadog}); !errors.Is(err, ErrNotFound) {
		t.Errorf("UpdateAuditLogStream() error = %v, want ErrNotFound", err)
	}
}
