package infisicalclient

import (
	"errors"
	"net/http"
	"testing"
)

func TestGetInstanceDeploymentAwsMetadataReadsAccountId(t *testing.T) {
	var path string
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		jsonResponse(http.StatusOK, `{"metadata":{"accountId":"123456789012"}}`)(w, r)
	})

	metadata, err := client.GetInstanceDeploymentAwsMetadata()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != "/api/v1/instance/deployment/aws/metadata" {
		t.Errorf("expected GET /api/v1/instance/deployment/aws/metadata, got %s", path)
	}
	if metadata.AccountId == nil || *metadata.AccountId != "123456789012" {
		t.Errorf("expected accountId 123456789012, got %v", metadata.AccountId)
	}
}

func TestGetInstanceDeploymentAwsMetadataLeavesAccountIdNilWhenNull(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusOK, `{"metadata":{"accountId":null}}`))

	metadata, err := client.GetInstanceDeploymentAwsMetadata()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if metadata.AccountId != nil {
		t.Errorf("expected accountId to be nil, got %q", *metadata.AccountId)
	}
}

func TestGetInstanceDeploymentAwsMetadataReturnsNotFoundOnOlderServers(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusNotFound, `{"message":"Route GET:/api/v1/instance/deployment/aws/metadata not found"}`))

	if _, err := client.GetInstanceDeploymentAwsMetadata(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetInstanceDeploymentAwsMetadataReturnsAPIError(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusUnauthorized, `{"message":"Unauthorized"}`))

	_, err := client.GetInstanceDeploymentAwsMetadata()
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected an API error for a 401 response, got %v", err)
	}
}
