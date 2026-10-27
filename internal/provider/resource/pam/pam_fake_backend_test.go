package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const fakePamProjectID = "pam-project"

type fakePamRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

type fakePamAccount struct {
	ID                string
	Type              string
	Name              string
	Description       *string
	FolderID          string
	TemplateID        string
	GatewayID         *string
	GatewayPoolID     *string
	ConnectionDetails map[string]any
	Credentials       map[string]any
	RotationAccountID *string
}

type fakePamMembership struct {
	ID   string
	Role string
}

// fakePamBackend mimics the PAM API, including its credential merging and rotation clearing.
type fakePamBackend struct {
	t        *testing.T
	mu       sync.Mutex
	nextID   int
	requests []fakePamRequest

	folders   map[string]map[string]any
	templates map[string]map[string]any
	accounts  map[string]*fakePamAccount
	// keyed by scope ("product", "folders/<id>", "accounts/<id>"), then kind, then member ID
	members map[string]map[string]map[string]*fakePamMembership
	// the organization's users, by email
	orgUsers map[string]string
}

func newFakePamBackend(t *testing.T) *fakePamBackend {
	return &fakePamBackend{
		t:         t,
		folders:   map[string]map[string]any{},
		templates: map[string]map[string]any{},
		accounts:  map[string]*fakePamAccount{},
		members:   map[string]map[string]map[string]*fakePamMembership{},
		orgUsers: map[string]string{
			"alice@acme.com": "11111111-1111-1111-1111-111111111111",
			"bob@acme.com":   "22222222-2222-2222-2222-222222222222",
		},
	}
}

func (b *fakePamBackend) newID() string {
	b.nextID++
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", b.nextID)
}

func (b *fakePamBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakePamBackend) apiError(w http.ResponseWriter, status int, message string) {
	b.writeJSON(w, status, map[string]any{"message": message, "reqId": "req-1"})
}

// requestsTo returns the bodies of the recorded requests with this method and path.
func (b *fakePamBackend) requestsTo(method, path string) []map[string]any {
	b.mu.Lock()
	defer b.mu.Unlock()
	var bodies []map[string]any
	for _, request := range b.requests {
		if request.Method == method && request.Path == path {
			bodies = append(bodies, request.Body)
		}
	}
	return bodies
}

func (b *fakePamBackend) membersOf(scope, kind string) map[string]*fakePamMembership {
	if b.members[scope] == nil {
		b.members[scope] = map[string]map[string]*fakePamMembership{}
	}
	if b.members[scope][kind] == nil {
		b.members[scope][kind] = map[string]*fakePamMembership{}
	}
	return b.members[scope][kind]
}

func (b *fakePamBackend) isProductMember(kind, memberID string) bool {
	_, ok := b.membersOf("product", kind)[memberID]
	return ok
}

func (b *fakePamBackend) addMember(scope, kind, memberID, role string) *fakePamMembership {
	membership := &fakePamMembership{ID: b.newID(), Role: role}
	b.membersOf(scope, kind)[memberID] = membership
	return membership
}

func memberResult(kind, memberID string, membership *fakePamMembership) map[string]any {
	result := map[string]any{"membershipId": membership.ID, "role": membership.Role, "isActive": true}
	switch kind {
	case "users":
		result["userId"] = memberID
	case "groups":
		result["groupId"] = memberID
	default:
		result["identityId"] = memberID
	}
	return result
}

var fakePamSecretCredentialKeys = []string{"password", "privateKey", "serviceAccountToken"}

func (a *fakePamAccount) response() map[string]any {
	credentials := maps.Clone(a.Credentials)
	for _, key := range fakePamSecretCredentialKeys {
		delete(credentials, key)
	}
	return map[string]any{
		"id":                a.ID,
		"name":              a.Name,
		"description":       a.Description,
		"folderId":          a.FolderID,
		"templateId":        a.TemplateID,
		"gatewayId":         a.GatewayID,
		"gatewayPoolId":     a.GatewayPoolID,
		"accountType":       a.Type,
		"connectionDetails": a.ConnectionDetails,
		"credentials":       credentials,
	}
}

func optionalBodyString(body map[string]any, key string) *string {
	value, ok := body[key].(string)
	if !ok {
		return nil
	}
	return &value
}

func (b *fakePamBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/pam/project", func(w http.ResponseWriter, _ *http.Request) {
		b.writeJSON(w, http.StatusOK, map[string]any{"projectId": fakePamProjectID})
	})

	// The PAM project's memberships are its product user members.
	mux.HandleFunc("GET /api/v1/projects/{projectId}/memberships", func(w http.ResponseWriter, r *http.Request) {
		memberships := []map[string]any{}
		for email, userID := range b.orgUsers {
			if b.isProductMember("users", userID) {
				memberships = append(memberships, map[string]any{
					"id":     "pm-" + userID,
					"userId": userID,
					"user":   map[string]any{"id": userID, "email": email, "username": email},
				})
			}
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"memberships": memberships})
	})

	mux.HandleFunc("POST /api/v1/pam/folders", func(w http.ResponseWriter, r *http.Request) {
		body := b.body(r)
		id := b.newID()
		folder := map[string]any{"id": id, "name": body["name"], "description": body["description"]}
		b.folders[id] = folder
		b.writeJSON(w, http.StatusOK, map[string]any{"folder": folder})
	})
	mux.HandleFunc("GET /api/v1/pam/folders/{id}", func(w http.ResponseWriter, r *http.Request) {
		folder, ok := b.folders[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Folder not found")
			return
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"folder": folder})
	})
	mux.HandleFunc("PATCH /api/v1/pam/folders/{id}", func(w http.ResponseWriter, r *http.Request) {
		folder, ok := b.folders[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Folder not found")
			return
		}
		maps.Copy(folder, b.body(r))
		b.writeJSON(w, http.StatusOK, map[string]any{"folder": folder})
	})
	mux.HandleFunc("DELETE /api/v1/pam/folders/{id}", func(w http.ResponseWriter, r *http.Request) {
		folder, ok := b.folders[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Folder not found")
			return
		}
		delete(b.folders, r.PathValue("id"))
		b.writeJSON(w, http.StatusOK, map[string]any{"folder": folder})
	})

	mux.HandleFunc("POST /api/v1/pam/account-templates", func(w http.ResponseWriter, r *http.Request) {
		body := b.body(r)
		settings, _ := body["settings"].(map[string]any)
		if settings == nil {
			settings = map[string]any{}
		}
		if _, ok := settings["heartbeat"]; !ok {
			settings["heartbeat"] = map[string]any{"enabled": true, "intervalSeconds": 86400}
		}
		if _, ok := settings["sessionLogMaskingBuiltInDetection"]; !ok {
			accountType, _ := body["type"].(string)
			settings["sessionLogMaskingBuiltInDetection"] = pamBuiltInMaskingDefault(accountType)
		}
		id := b.newID()
		body["id"] = id
		body["settings"] = settings
		b.templates[id] = body
		b.writeJSON(w, http.StatusOK, map[string]any{"template": body})
	})
	mux.HandleFunc("GET /api/v1/pam/account-templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		template, ok := b.templates[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Account template not found")
			return
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"template": template})
	})
	mux.HandleFunc("PATCH /api/v1/pam/account-templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		template, ok := b.templates[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Account template not found")
			return
		}
		maps.Copy(template, b.body(r))
		b.writeJSON(w, http.StatusOK, map[string]any{"template": template})
	})
	mux.HandleFunc("DELETE /api/v1/pam/account-templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		template, ok := b.templates[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Account template not found")
			return
		}
		delete(b.templates, r.PathValue("id"))
		b.writeJSON(w, http.StatusOK, map[string]any{"template": template})
	})

	mux.HandleFunc("POST /api/v1/pam/accounts/{type}", func(w http.ResponseWriter, r *http.Request) {
		body := b.body(r)
		name, _ := body["name"].(string)
		folderID, _ := body["folderId"].(string)
		templateID, _ := body["templateId"].(string)
		connectionDetails, _ := body["connectionDetails"].(map[string]any)
		credentials, _ := body["credentials"].(map[string]any)
		account := &fakePamAccount{
			ID:                b.newID(),
			Type:              r.PathValue("type"),
			Name:              name,
			Description:       optionalBodyString(body, "description"),
			FolderID:          folderID,
			TemplateID:        templateID,
			GatewayID:         optionalBodyString(body, "gatewayId"),
			GatewayPoolID:     optionalBodyString(body, "gatewayPoolId"),
			ConnectionDetails: connectionDetails,
			Credentials:       credentials,
		}
		b.accounts[account.ID] = account
		b.writeJSON(w, http.StatusOK, map[string]any{"account": account.response()})
	})
	mux.HandleFunc("GET /api/v1/pam/accounts/{id}", func(w http.ResponseWriter, r *http.Request) {
		account, ok := b.accounts[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Account not found")
			return
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"account": account.response()})
	})
	mux.HandleFunc("PATCH /api/v1/pam/accounts/{type}/{id}", func(w http.ResponseWriter, r *http.Request) {
		account, ok := b.accounts[r.PathValue("id")]
		if !ok || account.Type != r.PathValue("type") {
			b.apiError(w, http.StatusNotFound, "Account not found")
			return
		}
		body := b.body(r)
		if name, ok := body["name"].(string); ok {
			account.Name = name
		}
		if _, ok := body["description"]; ok {
			account.Description = optionalBodyString(body, "description")
		}
		if folderID, ok := body["folderId"].(string); ok {
			account.FolderID = folderID
		}
		if templateID, ok := body["templateId"].(string); ok {
			account.TemplateID = templateID
		}
		routingChanged := false
		if _, ok := body["gatewayId"]; ok {
			account.GatewayID = optionalBodyString(body, "gatewayId")
			routingChanged = true
		}
		if _, ok := body["gatewayPoolId"]; ok {
			account.GatewayPoolID = optionalBodyString(body, "gatewayPoolId")
			routingChanged = true
		}
		if connectionDetails, ok := body["connectionDetails"].(map[string]any); ok {
			routingChanged = routingChanged || connectionDetails["host"] != account.ConnectionDetails["host"] || connectionDetails["port"] != account.ConnectionDetails["port"]
			account.ConnectionDetails = connectionDetails
		}
		if credentials, ok := body["credentials"].(map[string]any); ok {
			routingChanged = routingChanged || credentials["username"] != account.Credentials["username"]
			merged := maps.Clone(account.Credentials)
			maps.Copy(merged, credentials)
			account.Credentials = merged
		}
		if routingChanged {
			account.RotationAccountID = nil
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"account": account.response()})
	})
	mux.HandleFunc("DELETE /api/v1/pam/accounts/{type}/{id}", func(w http.ResponseWriter, r *http.Request) {
		account, ok := b.accounts[r.PathValue("id")]
		if !ok || account.Type != r.PathValue("type") {
			b.apiError(w, http.StatusNotFound, "Account not found")
			return
		}
		delete(b.accounts, account.ID)
		b.writeJSON(w, http.StatusOK, map[string]any{"account": account.response()})
	})
	mux.HandleFunc("GET /api/v1/pam/accounts/{id}/rotation", func(w http.ResponseWriter, r *http.Request) {
		account, ok := b.accounts[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Account not found")
			return
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"rotation": map[string]any{"enabled": false, "rotationAccountId": account.RotationAccountID}})
	})
	mux.HandleFunc("PATCH /api/v1/pam/accounts/{id}/rotation", func(w http.ResponseWriter, r *http.Request) {
		account, ok := b.accounts[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Account not found")
			return
		}
		account.RotationAccountID = optionalBodyString(b.body(r), "rotationAccountId")
		b.writeJSON(w, http.StatusOK, map[string]any{"rotationAccountId": account.RotationAccountID})
	})

	mux.HandleFunc("POST /api/v1/pam/memberships/users", func(w http.ResponseWriter, r *http.Request) {
		body := b.body(r)
		memberships, skipped := []map[string]any{}, []string{}
		role, _ := body["role"].(string)
		emails, _ := body["emails"].([]any)
		for _, item := range emails {
			email, _ := item.(string)
			userID, ok := b.orgUsers[email]
			if !ok {
				b.apiError(w, http.StatusBadRequest, "User not found")
				return
			}
			if b.isProductMember("users", userID) {
				skipped = append(skipped, email)
				continue
			}
			memberships = append(memberships, memberResult("users", userID, b.addMember("product", "users", userID, role)))
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"memberships": memberships, "skipped": skipped})
	})
	b.handleMembers(mux, "/api/v1/pam/memberships/{kind}", func(*http.Request) string { return "product" })
	b.handleMembers(mux, "/api/v1/pam/folders/{folderId}/{kind}", func(r *http.Request) string { return "folders/" + r.PathValue("folderId") })
	b.handleMembers(mux, "/api/v1/pam/accounts/{accountId}/{kind}", func(r *http.Request) string { return "accounts/" + r.PathValue("accountId") })

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()
		mux.ServeHTTP(w, r)
	})
}

func (b *fakePamBackend) scopeExists(scope string) bool {
	if id, ok := strings.CutPrefix(scope, "folders/"); ok {
		_, exists := b.folders[id]
		return exists
	}
	if id, ok := strings.CutPrefix(scope, "accounts/"); ok {
		_, exists := b.accounts[id]
		return exists
	}
	return true
}

func (b *fakePamBackend) handleMembers(mux *http.ServeMux, prefix string, scopeOf func(*http.Request) string) {
	mux.HandleFunc("GET "+prefix, func(w http.ResponseWriter, r *http.Request) {
		scope, kind := scopeOf(r), r.PathValue("kind")
		if !b.scopeExists(scope) {
			b.apiError(w, http.StatusNotFound, "Not found")
			return
		}
		members := []map[string]any{}
		for memberID, membership := range b.membersOf(scope, kind) {
			members = append(members, memberResult(kind, memberID, membership))
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"members": members})
	})
	mux.HandleFunc("POST "+prefix+"/{memberId}", func(w http.ResponseWriter, r *http.Request) {
		scope, kind, memberID := scopeOf(r), r.PathValue("kind"), r.PathValue("memberId")
		if !b.scopeExists(scope) {
			b.apiError(w, http.StatusNotFound, "Not found")
			return
		}
		if scope != "product" && !b.isProductMember(kind, memberID) {
			b.apiError(w, http.StatusBadRequest, "Must be a PAM product member first")
			return
		}
		if _, exists := b.membersOf(scope, kind)[memberID]; exists {
			b.apiError(w, http.StatusBadRequest, "Already a member")
			return
		}
		role, _ := b.body(r)["role"].(string)
		b.writeJSON(w, http.StatusOK, memberResult(kind, memberID, b.addMember(scope, kind, memberID, role)))
	})
	mux.HandleFunc("PATCH "+prefix+"/{memberId}", func(w http.ResponseWriter, r *http.Request) {
		scope, kind, memberID := scopeOf(r), r.PathValue("kind"), r.PathValue("memberId")
		membership, ok := b.membersOf(scope, kind)[memberID]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Membership not found")
			return
		}
		membership.Role, _ = b.body(r)["role"].(string)
		b.writeJSON(w, http.StatusOK, memberResult(kind, memberID, membership))
	})
	mux.HandleFunc("DELETE "+prefix+"/{memberId}", func(w http.ResponseWriter, r *http.Request) {
		scope, kind, memberID := scopeOf(r), r.PathValue("kind"), r.PathValue("memberId")
		membership, ok := b.membersOf(scope, kind)[memberID]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Membership not found")
			return
		}
		delete(b.membersOf(scope, kind), memberID)
		b.writeJSON(w, http.StatusOK, memberResult(kind, memberID, membership))
	})
}

// body decodes and records the request body.
func (b *fakePamBackend) body(r *http.Request) map[string]any {
	body := map[string]any{}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	b.requests = append(b.requests, fakePamRequest{Method: r.Method, Path: r.URL.Path, Body: body})
	return body
}

func newPamTestClient(t *testing.T, backend *fakePamBackend) *infisical.Client {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}
}

func pamResourceSchema(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func emptyPamPlan(s schema.Schema) tfsdk.Plan {
	return tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

func createPamResource(t *testing.T, r resource.Resource, plan tfsdk.Plan) tfsdk.State {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: tftypes.NewValue(plan.Schema.Type().TerraformType(context.Background()), nil)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

// readPamResource returns the refreshed state, which is null when the resource is gone.
func readPamResource(t *testing.T, r resource.Resource, state tfsdk.State) tfsdk.State {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func updatePamResource(t *testing.T, r resource.Resource, plan tfsdk.Plan, state tfsdk.State) tfsdk.State {
	t.Helper()
	resp := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func deletePamResource(t *testing.T, r resource.Resource, state tfsdk.State) {
	t.Helper()
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

func importPamResource(t *testing.T, r resource.ResourceWithImportState, s schema.Schema, id string) tfsdk.State {
	t.Helper()
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return readPamResource(t, r, resp.State)
}

func configurePamResource(t *testing.T, r resource.ResourceWithConfigure, client *infisical.Client) {
	t.Helper()
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

// Checked accessors, so a wrong shape fails the test instead of panicking.

func jsonObject(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected a JSON object, got %T: %v", value, value)
	}
	return object
}

func objectAttributes(t *testing.T, value attr.Value) map[string]attr.Value {
	t.Helper()
	object, ok := value.(types.Object)
	if !ok {
		t.Fatalf("expected an object, got %T", value)
	}
	return object.Attributes()
}

func boolAttribute(t *testing.T, values map[string]attr.Value, name string) bool {
	t.Helper()
	value, ok := values[name].(types.Bool)
	if !ok {
		t.Fatalf("expected %s to be a bool, got %T", name, values[name])
	}
	return value.ValueBool()
}

func int64Attribute(t *testing.T, values map[string]attr.Value, name string) int64 {
	t.Helper()
	value, ok := values[name].(types.Int64)
	if !ok {
		t.Fatalf("expected %s to be a number, got %T", name, values[name])
	}
	return value.ValueInt64()
}

func stringAttribute(t *testing.T, values map[string]attr.Value, name string) string {
	t.Helper()
	value, ok := values[name].(types.String)
	if !ok {
		t.Fatalf("expected %s to be a string, got %T", name, values[name])
	}
	return value.ValueString()
}
