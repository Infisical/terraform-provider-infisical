package infisicalclient

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// duplicateAlertRequest is the create every test here rejects, and the alerts the lookup finds are
// matched against it.
var duplicateAlertRequest = CreateAlertRequest{
	Name:         "Credentials expiring",
	ResourceType: "identity.authentication",
	ResourceID:   "identity-1",
	EventType:    "identity.authentication.credential.expiring",
}

// alertRejectingCreates answers a create with the given bad request and serves listed alerts to the
// lookup that follows, which is what the create reads to tell a duplicate from any other rejection.
func alertRejectingCreates(t *testing.T, message string, listed ...Alert) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(ListAlertsResponse{Alerts: listed}); err != nil {
				t.Errorf("writing the listed alerts: %v", err)
			}
			return
		}
		writeAlertError(t, w, http.StatusBadRequest, message)
	}
}

// The uniqueness check is the one 400 a practitioner can act on, so it has to arrive as something the
// resource can recognize rather than as a generic API error, and it has to name the alert holding the
// spot for the resource to point at.
func TestCreateAlertReportsADuplicate(t *testing.T) {
	client, done := alertTestClient(t, alertRejectingCreates(t,
		"An alert for this resource and event already exists",
		Alert{ID: "alert-1", EventType: duplicateAlertRequest.EventType},
	))
	defer done()

	_, err := client.CreateAlert(duplicateAlertRequest)

	var alreadyExists *AlertAlreadyExistsError
	if !errors.As(err, &alreadyExists) {
		t.Fatalf("CreateAlert() error = %v, want an AlertAlreadyExistsError", err)
	}
	if alreadyExists.ExistingAlertID != "alert-1" {
		t.Errorf("existing alert ID = %q, want alert-1", alreadyExists.ExistingAlertID)
	}

	// The advice replaces none of the diagnosis, so the original response has to survive alongside it.
	if got := err.Error(); !strings.Contains(got, "req-1") || !strings.Contains(got, "already exists") {
		t.Errorf("CreateAlert() error = %q, want it to keep the API error's message and request ID", got)
	}
}

// The message the API rejects a create with says nothing the provider relies on, so a duplicate is
// reported as one even when the wording is not what this provider was written against.
func TestCreateAlertReportsADuplicateWhateverTheMessageSays(t *testing.T) {
	client, done := alertTestClient(t, alertRejectingCreates(t,
		"Duplicate alert",
		Alert{ID: "alert-1", EventType: duplicateAlertRequest.EventType},
	))
	defer done()

	_, err := client.CreateAlert(duplicateAlertRequest)

	var alreadyExists *AlertAlreadyExistsError
	if !errors.As(err, &alreadyExists) {
		t.Fatalf("CreateAlert() error = %v, want an AlertAlreadyExistsError", err)
	}
}

// Any other bad request is a malformed call, and telling the practitioner to import an existing alert
// would send them chasing something that is not there. Nothing watches the resource for this event, so
// the rejection cannot be a duplicate however much its message reads like one.
func TestCreateAlertLeavesOtherBadRequestsAlone(t *testing.T) {
	for name, listed := range map[string][]Alert{
		"nothing watches the resource": nil,
		"another event is watched":     {{ID: "alert-1", EventType: "identity.authentication.credential.rotated"}},
	} {
		t.Run(name, func(t *testing.T) {
			client, done := alertTestClient(t, alertRejectingCreates(t,
				"An alert for this resource and event already exists",
				listed...,
			))
			defer done()

			_, err := client.CreateAlert(duplicateAlertRequest)
			if err == nil {
				t.Fatal("CreateAlert() error = nil, want one")
			}

			var alreadyExists *AlertAlreadyExistsError
			if errors.As(err, &alreadyExists) {
				t.Errorf("CreateAlert() error = %v, want it not to be an AlertAlreadyExistsError", err)
			}
		})
	}
}

// A create that is rejected for anything but a bad request cannot be a duplicate, so it is reported as
// it arrived and costs no lookup.
func TestCreateAlertOnAForbiddenRequest(t *testing.T) {
	lookups := 0
	client, done := alertTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			lookups++
		}
		writeAlertError(t, w, http.StatusForbidden, "You do not have permission to create alerts")
	})
	defer done()

	_, err := client.CreateAlert(duplicateAlertRequest)

	var alreadyExists *AlertAlreadyExistsError
	if errors.As(err, &alreadyExists) {
		t.Errorf("CreateAlert() error = %v, want it not to be an AlertAlreadyExistsError", err)
	}
	if lookups != 0 {
		t.Errorf("CreateAlert() listed alerts %d times, want it not to look one up", lookups)
	}
}

// A lookup that cannot answer leaves the create reporting what the API said, because guessing at a
// duplicate would send the practitioner importing an alert nothing confirmed is there.
func TestCreateAlertWhenTheLookupFails(t *testing.T) {
	client, done := alertTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeAlertError(t, w, http.StatusInternalServerError, "Something went wrong")
			return
		}
		writeAlertError(t, w, http.StatusBadRequest, "An alert for this resource and event already exists")
	})
	defer done()

	_, err := client.CreateAlert(duplicateAlertRequest)

	var alreadyExists *AlertAlreadyExistsError
	if errors.As(err, &alreadyExists) {
		t.Errorf("CreateAlert() error = %v, want it not to be an AlertAlreadyExistsError", err)
	}
	if got := err.Error(); !strings.Contains(got, "already exists") {
		t.Errorf("CreateAlert() error = %q, want it to carry the API's message", got)
	}
}

// The lookup asks for the alerts of the scope the create was rejected for, so an alert of another
// resource or another project cannot be mistaken for the one holding the spot.
func TestListAlertsScopesTheRequest(t *testing.T) {
	var query url.Values
	client, done := alertTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ListAlertsResponse{Alerts: []Alert{{ID: "alert-1"}}}); err != nil {
			t.Errorf("writing the listed alerts: %v", err)
		}
	})
	defer done()

	projectID := "project-1"
	alerts, err := client.ListAlerts(ListAlertsRequest{
		ResourceType: "identity.authentication",
		ResourceID:   "identity-1",
		ProjectID:    &projectID,
	})
	if err != nil {
		t.Fatalf("ListAlerts() error = %v", err)
	}
	if len(alerts.Alerts) != 1 || alerts.Alerts[0].ID != "alert-1" {
		t.Errorf("ListAlerts() = %v, want the listed alert", alerts.Alerts)
	}

	for parameter, want := range map[string]string{
		"resourceType": "identity.authentication",
		"resourceId":   "identity-1",
		"projectId":    "project-1",
	} {
		if got := query.Get(parameter); got != want {
			t.Errorf("%s = %q, want %q", parameter, got, want)
		}
	}
}

// An alert of the organization has no project to scope the lookup to, and asking for one would scope it
// to a project the alert is not in.
func TestListAlertsWithoutAProject(t *testing.T) {
	var query url.Values
	client, done := alertTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ListAlertsResponse{}); err != nil {
			t.Errorf("writing the listed alerts: %v", err)
		}
	})
	defer done()

	if _, err := client.ListAlerts(ListAlertsRequest{ResourceType: "identity.authentication"}); err != nil {
		t.Fatalf("ListAlerts() error = %v", err)
	}

	if _, asked := query["projectId"]; asked {
		t.Errorf("projectId = %q, want it left out", query.Get("projectId"))
	}
	if _, asked := query["resourceId"]; asked {
		t.Errorf("resourceId = %q, want it left out", query.Get("resourceId"))
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
