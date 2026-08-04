package infisicalclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

// alertTestClient serves every alert request from one handler, which is enough to drive the status
// code and body each error path keys off.
func alertTestClient(t *testing.T, handler http.HandlerFunc) (Client, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	client := Client{Config: Config{
		HostURL:    server.URL,
		HttpClient: resty.New().SetBaseURL(server.URL),
	}}
	return client, server.Close
}

func writeAlertError(t *testing.T, w http.ResponseWriter, status int, message string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(`{"message":"` + message + `","reqId":"req-1"}`)); err != nil {
		t.Fatalf("writing the error body: %v", err)
	}
}

// The uniqueness check is the one 400 a practitioner can act on, so it has to arrive as something the
// resource can recognize rather than as a generic API error.
func TestCreateAlertReportsADuplicate(t *testing.T) {
	client, done := alertTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeAlertError(t, w, http.StatusBadRequest, "An alert for this resource and event already exists")
	})
	defer done()

	_, err := client.CreateAlert(CreateAlertRequest{Name: "Credentials expiring"})
	if !errors.Is(err, ErrAlertAlreadyExists) {
		t.Fatalf("CreateAlert() error = %v, want it to be ErrAlertAlreadyExists", err)
	}

	// The advice replaces none of the diagnosis, so the original response has to survive alongside it.
	if got := err.Error(); !strings.Contains(got, "req-1") || !strings.Contains(got, "already exists") {
		t.Errorf("CreateAlert() error = %q, want it to keep the API error's message and request ID", got)
	}
}

// Any other bad request is a malformed call, and telling the practitioner to import an existing alert
// would send them chasing something that is not there.
func TestCreateAlertLeavesOtherBadRequestsAlone(t *testing.T) {
	client, done := alertTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeAlertError(t, w, http.StatusBadRequest, "Invalid alert condition: Must be a whole number of days")
	})
	defer done()

	_, err := client.CreateAlert(CreateAlertRequest{Name: "Credentials expiring"})
	if err == nil {
		t.Fatal("CreateAlert() error = nil, want one")
	}
	if errors.Is(err, ErrAlertAlreadyExists) {
		t.Errorf("CreateAlert() error = %v, want it not to be ErrAlertAlreadyExists", err)
	}
}

// An alert deleted outside Terraform has to surface as ErrNotFound the way a read does, so the update
// can explain itself instead of relaying a bare 404.
func TestUpdateAlertOnADeletedAlert(t *testing.T) {
	for name, status := range map[string]int{
		"not found":            http.StatusNotFound,
		"unprocessable entity": http.StatusUnprocessableEntity,
	} {
		t.Run(name, func(t *testing.T) {
			client, done := alertTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				writeAlertError(t, w, status, "Alert with ID 'alert-1' not found")
			})
			defer done()

			_, err := client.UpdateAlert(UpdateAlertRequest{ID: "alert-1"})
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("UpdateAlert() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestUpdateAlertReportsOtherFailures(t *testing.T) {
	client, done := alertTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeAlertError(t, w, http.StatusBadRequest, "Channel 'channel-1' does not belong to this alert")
	})
	defer done()

	_, err := client.UpdateAlert(UpdateAlertRequest{ID: "alert-1"})
	if err == nil {
		t.Fatal("UpdateAlert() error = nil, want one")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("UpdateAlert() error = %v, want it not to be ErrNotFound", err)
	}
	if got := err.Error(); !strings.Contains(got, "does not belong to this alert") {
		t.Errorf("UpdateAlert() error = %q, want it to carry the API's message", got)
	}
}
