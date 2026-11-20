package infisicalclient

import (
	"errors"
	"net/http"
	"testing"
)

const privilegeNotFoundBody = `{"statusCode":404,"message":"Additional privilege not found","reqId":"req-1"}`

// A privilege deleted outside Terraform must surface as ErrNotFound so Read can drop it from
// state instead of failing every plan.
func TestGetProjectIdentitySpecificPrivilegeReportsMissingPrivilegeAsNotFound(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusNotFound, privilegeNotFoundBody))

	t.Run("V1 by slug", func(t *testing.T) {
		_, err := client.GetProjectIdentitySpecificPrivilegeBySlug(GetProjectIdentitySpecificPrivilegeRequest{
			PrivilegeSlug: "gone", ProjectSlug: "proj", IdentityID: "identity",
		})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound for a 404, got %v", err)
		}
	})

	t.Run("V2 by id", func(t *testing.T) {
		_, err := client.GetProjectIdentitySpecificPrivilegeV2(GetProjectIdentitySpecificPrivilegeV2Request{ID: "gone"})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound for a 404, got %v", err)
		}
	})
}

// Only a 404 proves absence: any other failure must stay an error.
func TestGetProjectIdentitySpecificPrivilegeKeepsOtherFailuresAsErrors(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusForbidden, `{"statusCode":403,"message":"forbidden","reqId":"req-2"}`))

	_, err := client.GetProjectIdentitySpecificPrivilegeV2(GetProjectIdentitySpecificPrivilegeV2Request{ID: "x"})
	if err == nil {
		t.Fatal("expected an error for a 403")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("a 403 must not be reported as ErrNotFound")
	}
}
