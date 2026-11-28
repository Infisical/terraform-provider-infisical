package infisicalclient

import (
	"errors"
	"net/http"
	"testing"
)

// A privilege deleted outside Terraform comes back as a 404 on read. The resource's Read relies on
// ErrNotFound to drop it from state instead of failing the whole refresh.
func TestGetProjectIdentitySpecificPrivilegeBySlugNotFound(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusNotFound, `{"message":"not found"}`))

	_, err := client.GetProjectIdentitySpecificPrivilegeBySlug(GetProjectIdentitySpecificPrivilegeRequest{
		PrivilegeSlug: "gone", ProjectSlug: "proj", IdentityID: "id",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a 404, got: %v", err)
	}
}

func TestGetProjectIdentitySpecificPrivilegeV2NotFound(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusNotFound, `{"message":"not found"}`))

	_, err := client.GetProjectIdentitySpecificPrivilegeV2(GetProjectIdentitySpecificPrivilegeV2Request{ID: "gone"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a 404, got: %v", err)
	}
}

func TestGetProjectIdentitySpecificPrivilegeServerErrorIsNotNotFound(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusForbidden, `{"message":"forbidden"}`))

	if _, err := client.GetProjectIdentitySpecificPrivilegeV2(GetProjectIdentitySpecificPrivilegeV2Request{ID: "x"}); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a non-NotFound error for a 403, got: %v", err)
	}
}
