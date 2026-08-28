package datasource

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
)

// projectClient returns a client whose two project lookup endpoints answer with the
// bodies given. An empty body means that endpoint answers 404, which is how the
// not-found path is exercised.
func projectClient(t *testing.T, bySlugBody string, byIDBody string) *infisical.Client {
	t.Helper()

	respond := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			if body == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/workspace/", respond(bySlugBody))
	mux.HandleFunc("/api/v1/workspace/", respond(byIDBody))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &infisical.Client{Config: infisical.Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}
}

// The by-slug endpoint returns the project unwrapped, the by-id endpoint wraps it in a
// "workspace" envelope. Both must yield the same project to the data source.
const (
	projectBySlugBody = `{"id":"11111111-1111-1111-1111-111111111111","name":"Prod","slug":"prod","type":"secret-manager","orgId":"22222222-2222-2222-2222-222222222222","version":3,"environments":[{"id":"33333333-3333-3333-3333-333333333333","name":"Development","slug":"dev"}]}`
	projectByIDBody   = `{"workspace":` + projectBySlugBody + `}`
)

func TestFetchProjectBySlug(t *testing.T) {
	client := projectClient(t, projectBySlugBody, "")

	project, diags := fetchProject(client, "", "prod")

	if diags.HasError() {
		t.Fatalf("a slug lookup must succeed, got %v", diags.Errors())
	}
	if project.ID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("expected the matched project's id, got %q", project.ID)
	}
	if project.Slug != "prod" {
		t.Errorf("expected slug %q, got %q", "prod", project.Slug)
	}
}

func TestFetchProjectByIDUnwrapsTheWorkspaceEnvelope(t *testing.T) {
	client := projectClient(t, "", projectByIDBody)

	project, diags := fetchProject(client, "11111111-1111-1111-1111-111111111111", "")

	if diags.HasError() {
		t.Fatalf("an id lookup must succeed, got %v", diags.Errors())
	}
	if project.ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("the workspace envelope was not unwrapped, got id %q", project.ID)
	}
	// The two endpoints must be interchangeable, so check the fields the data source
	// exposes rather than the id alone.
	if project.Name != "Prod" || project.Slug != "prod" || project.Type != "secret-manager" {
		t.Errorf("unexpected project: %+v", project)
	}
	if project.OrgID != "22222222-2222-2222-2222-222222222222" || project.Version != 3 {
		t.Errorf("unexpected project: %+v", project)
	}
	if len(project.Environments) != 1 || project.Environments[0].Slug != "dev" {
		t.Errorf("expected the dev environment, got %+v", project.Environments)
	}
}

func TestFetchProjectReportsAMissingSlug(t *testing.T) {
	client := projectClient(t, "", "")

	_, diags := fetchProject(client, "", "absent")

	if !diags.HasError() {
		t.Fatal("a slug that matches nothing must be an error, not an empty result")
	}
	if summary := diags.Errors()[0].Summary(); summary != "Project not found" {
		t.Errorf("a 404 must surface as a not-found error, got %q", summary)
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "absent") {
		t.Errorf("the error must name the slug searched for, got %q", detail)
	}
}

func TestFetchProjectReportsAMissingID(t *testing.T) {
	client := projectClient(t, "", "")

	_, diags := fetchProject(client, "44444444-4444-4444-4444-444444444444", "")

	if !diags.HasError() {
		t.Fatal("an id that matches nothing must be an error, not an empty result")
	}
	if summary := diags.Errors()[0].Summary(); summary != "Project not found" {
		t.Errorf("a 404 must surface as a not-found error, got %q", summary)
	}
	if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "44444444-4444-4444-4444-444444444444") {
		t.Errorf("the error must name the id searched for, got %q", detail)
	}
}
