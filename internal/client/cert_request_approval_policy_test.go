package infisicalclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestCertRequestApprovalPolicyNotFound(t *testing.T) {
	cases := map[string]struct {
		status       int
		body         string
		wantNotFound bool
	}{
		"policy deleted":     {status: http.StatusForbidden, body: `{"message":"Policy not found"}`, wantNotFound: true},
		"plain 404":          {status: http.StatusNotFound, body: `{"message":"Not found"}`, wantNotFound: true},
		"missing permission": {status: http.StatusForbidden, body: `{"message":"User has insufficient privileges"}`},
		"unreadable 403":     {status: http.StatusForbidden, body: `forbidden`},
		"type mismatch":      {status: http.StatusBadRequest, body: `{"message":"Policy type mismatch: expected cert-request, got pam-access"}`},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client := testClient(t, jsonResponse(c.status, c.body))

			_, getErr := client.GetCertRequestApprovalPolicy(GetCertRequestApprovalPolicyRequest{ID: "policy-1"})
			_, updateErr := client.UpdateCertRequestApprovalPolicy(UpdateCertRequestApprovalPolicyRequest{ID: "policy-1"})
			deleteErr := client.DeleteCertRequestApprovalPolicy(DeleteCertRequestApprovalPolicyRequest{ID: "policy-1"})

			for operation, err := range map[string]error{"get": getErr, "update": updateErr, "delete": deleteErr} {
				if err == nil {
					t.Fatalf("%s: no error, want one", operation)
				}
				if got := errors.Is(err, ErrNotFound); got != c.wantNotFound {
					t.Errorf("%s: errors.Is(err, ErrNotFound) = %v, want %v (err: %v)", operation, got, c.wantNotFound, err)
				}
			}
		})
	}
}

func TestCreateCertRequestApprovalPolicy(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		jsonResponse(http.StatusOK, `{"policy":{"id":"policy-1","projectId":"project-1","scopeType":"pki-application","scopeId":"app-1","conditions":{"version":1,"conditions":[{"profileNames":["web"]}]},"steps":[]}}`)(w, r)
	})

	notify := true
	created, err := client.CreateCertRequestApprovalPolicy(CreateCertRequestApprovalPolicyRequest{
		Scope:      "pki-application",
		ScopeID:    "app-1",
		Name:       "web approvals",
		Conditions: []CertRequestApprovalPolicyCondition{{ProfileNames: []string{"web"}}},
		Steps: []ApprovalPolicyStep{{
			RequiredApprovals: 1,
			NotifyApprovers:   &notify,
			Approvers:         []ApprovalPolicyApprover{{Type: "user", ID: "user-1"}},
		}},
	})
	if err != nil {
		t.Fatalf("CreateCertRequestApprovalPolicy() error = %v", err)
	}

	if gotPath != "/api/v1/approval-policies/cert-request" {
		t.Errorf("path = %q, want the cert-request policies endpoint", gotPath)
	}
	if gotBody["scope"] != "pki-application" || gotBody["scopeId"] != "app-1" {
		t.Errorf("scope = %v / %v, want pki-application / app-1", gotBody["scope"], gotBody["scopeId"])
	}
	steps, ok := gotBody["steps"].([]any)
	if !ok || len(steps) != 1 {
		t.Fatalf("steps = %v, want one", gotBody["steps"])
	}
	step, ok := steps[0].(map[string]any)
	if !ok {
		t.Fatalf("step = %T, want an object", steps[0])
	}
	if _, ok := step["name"]; ok {
		t.Errorf("unnamed step was sent with a name: %v", step)
	}
	if step["notifyApprovers"] != true {
		t.Errorf("notifyApprovers = %v, want true", step["notifyApprovers"])
	}

	if created.Policy.ID != "policy-1" {
		t.Errorf("created policy = %+v, want policy-1", created.Policy)
	}
	if created.Policy.ScopeID == nil || *created.Policy.ScopeID != "app-1" {
		t.Errorf("scopeId = %v, want app-1", created.Policy.ScopeID)
	}
}
