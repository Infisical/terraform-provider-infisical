package infisicalclient

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
)

// The v1 create route has no sourceProjectId and drops it without an error, so a cross-project
// import sent there becomes a same-project import. Same-project imports stay on v1.
func TestCreateSecretImportRoutesBySourceProject(t *testing.T) {
	for name, tc := range map[string]struct {
		sourceProjectID string
		wantPath        string
		wantBody        map[string]any
	}{
		"same project": {
			wantPath: "/api/v1/secret-imports",
			wantBody: map[string]any{
				"workspaceId": "destination", "environment": "prod", "path": "/app", "isReplication": false,
				"import": map[string]any{"environment": "dev", "path": "/shared"},
			},
		},
		"cross project": {
			sourceProjectID: "source",
			wantPath:        "/api/v2/secret-imports",
			wantBody: map[string]any{
				"projectId": "destination", "environment": "prod", "path": "/app", "isReplication": false,
				"import": map[string]any{"sourceProjectId": "source", "environment": "dev", "path": "/shared"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var gotPath string
			var gotBody map[string]any
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Error(err)
				}
				jsonResponse(http.StatusOK, `{"secretImport":{"id":"import-1"}}`)(w, r)
			})

			resp, err := client.CreateSecretImport(CreateSecretImportRequest{
				ProjectID:   "destination",
				Environment: "prod",
				SecretPath:  "/app",
				ImportFrom:  SecretImportSource{SourceProjectID: tc.sourceProjectID, Environment: "dev", SecretPath: "/shared"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if resp.SecretImport.ID != "import-1" {
				t.Errorf("ID = %q, want import-1", resp.SecretImport.ID)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
			if !reflect.DeepEqual(gotBody, tc.wantBody) {
				t.Errorf("body = %v, want %v", gotBody, tc.wantBody)
			}
		})
	}
}

// Messages copied from createImport in Infisical core (secret-import-service.ts). Every case is a 403
// ForbiddenError, so the message is the only thing that tells them apart.
func TestCreateSecretImportRecognizesCrossProjectRefusals(t *testing.T) {
	for name, tc := range map[string]struct {
		message string
		want    error
	}{
		"sharing turned off": {"Cross-project secret sharing is not enabled for this organization", ErrCrossProjectSharingDisabled},
		"no project grant":   {"No project grant found allowing this cross-project import", ErrCrossProjectGrantMissing},
		"missing permission": {"You are not allowed to create on SecretImports", nil},
	} {
		t.Run(name, func(t *testing.T) {
			client := testClient(t, jsonResponse(http.StatusForbidden,
				`{"reqId":"req-1","statusCode":403,"message":"`+tc.message+`","error":"ForbiddenError"}`))

			_, err := client.CreateSecretImport(CreateSecretImportRequest{
				ProjectID:  "destination",
				ImportFrom: SecretImportSource{SourceProjectID: "source", Environment: "dev", SecretPath: "/shared"},
			})

			for _, sentinel := range []error{ErrCrossProjectSharingDisabled, ErrCrossProjectGrantMissing} {
				if got := errors.Is(err, sentinel); got != (sentinel == tc.want) {
					t.Errorf("errors.Is(err, %q) = %v, error: %v", sentinel, got, err)
				}
			}
			if apiErrs := collectAPIErrors(err); len(apiErrs) != 1 || apiErrs[0].ErrorMessage != tc.message {
				t.Errorf("expected the API error to stay reachable, got %v", err)
			}
		})
	}
}
