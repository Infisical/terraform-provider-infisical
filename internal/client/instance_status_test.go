package infisicalclient

import (
	"net/http"
	"testing"
)

func TestGetInstanceStatusReadsAwsAccountId(t *testing.T) {
	var path string
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		jsonResponse(http.StatusOK, `{"date":"2026-11-10T18:34:36.298Z","message":"Ok","awsAccountId":"123456789012"}`)(w, r)
	})

	status, err := client.GetInstanceStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != "/api/status" {
		t.Errorf("expected GET /api/status, got %s", path)
	}
	if status.AwsAccountId == nil || *status.AwsAccountId != "123456789012" {
		t.Errorf("expected awsAccountId 123456789012, got %v", status.AwsAccountId)
	}
}

func TestGetInstanceStatusLeavesAwsAccountIdNilWhenAbsent(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusOK, `{"date":"2026-11-10T18:34:36.298Z","message":"Ok"}`))

	status, err := client.GetInstanceStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status.AwsAccountId != nil {
		t.Errorf("expected awsAccountId to be nil, got %q", *status.AwsAccountId)
	}
}

func TestGetInstanceStatusReturnsAPIError(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusServiceUnavailable, `{"message":"Service unavailable"}`))

	if _, err := client.GetInstanceStatus(); err == nil {
		t.Fatal("expected an error for a 503 response")
	}
}
