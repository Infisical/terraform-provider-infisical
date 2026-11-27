package infisicalclient

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestMicrosoftADCSCAUsesTheAdcsRoutes(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotBody = nil
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ca-1","name":"corporate-adcs","status":"active","configuration":{"appConnectionId":"conn-1","caName":"corp-ca01-CA"}}`))
	})

	ca, err := client.CreateMicrosoftADCSCA(CreateMicrosoftADCSCARequest{
		Name: "corporate-adcs", Status: "active",
		Configuration: MicrosoftADCSCAConfiguration{AppConnectionId: "conn-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/cert-manager/ca/adcs" {
		t.Errorf("create went to %s %s", gotMethod, gotPath)
	}
	configuration, _ := gotBody["configuration"].(map[string]any)
	if _, sent := configuration["caName"]; sent {
		t.Errorf("a nil caName must be left out of the body, got %v", gotBody)
	}
	if ca.Configuration.CaName == nil || *ca.Configuration.CaName != "corp-ca01-CA" {
		t.Errorf("caName was not parsed from the response: %+v", ca)
	}

	caName := "corp-ca01-CA"
	if _, err := client.UpdateMicrosoftADCSCA(UpdateMicrosoftADCSCARequest{
		CAId: "ca-1", Name: "corporate-adcs", Status: "disabled",
		Configuration: &MicrosoftADCSCAConfiguration{AppConnectionId: "conn-1", CaName: &caName},
	}); err != nil {
		t.Fatal(err)
	}
	configuration, _ = gotBody["configuration"].(map[string]any)
	if gotMethod != http.MethodPatch || gotPath != "/api/v1/cert-manager/ca/adcs/ca-1" || configuration["caName"] != caName {
		t.Errorf("update went to %s %s with body %v", gotMethod, gotPath, gotBody)
	}

	if _, err := client.UpdateMicrosoftADCSCA(UpdateMicrosoftADCSCARequest{CAId: "ca-1", Name: "corporate-adcs", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	if _, sent := gotBody["configuration"]; sent {
		t.Errorf("a nil configuration must be left out of the body, got %v", gotBody)
	}

	if _, err := client.GetMicrosoftADCSCA("ca-1"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/v1/cert-manager/ca/adcs/ca-1" {
		t.Errorf("get went to %s %s", gotMethod, gotPath)
	}

	if err := client.DeleteMicrosoftADCSCA("ca-1"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/cert-manager/ca/adcs/ca-1" {
		t.Errorf("delete went to %s %s", gotMethod, gotPath)
	}
}

func TestMicrosoftADCSCANotFound(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusNotFound, `{"message":"Certificate Authority not found"}`))

	if _, err := client.GetMicrosoftADCSCA("ca-1"); err != ErrNotFound {
		t.Errorf("get: expected ErrNotFound, got %v", err)
	}
	if err := client.DeleteMicrosoftADCSCA("ca-1"); err != ErrNotFound {
		t.Errorf("delete: expected ErrNotFound, got %v", err)
	}
}

func TestDeleteAppConnectionNotFound(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusNotFound, `{"message":"App connection not found"}`))

	if _, err := client.DeleteAppConnection(DeleteAppConnectionRequest{App: AppConnectionAppADCS, ID: "conn-1"}); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
