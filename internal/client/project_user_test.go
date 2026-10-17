package infisicalclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	infisicalerrors "terraform-provider-infisical/internal/errors"

	"github.com/go-resty/resty/v2"
)

const (
	userLookupProject = "11111111-1111-4111-8111-111111111111"
	userLookupUser    = "22222222-2222-4222-8222-222222222222"
	userLookupPath    = "/api/v1/projects/" + userLookupProject + "/user/" + userLookupUser + "/membership"
	userListPath      = "/api/v1/projects/" + userLookupProject + "/memberships"
	userDetailsPath   = "/api/v1/workspace/" + userLookupProject + "/memberships/details"
	userLookupDetails = `{"membership":{"id":"membership-1","userId":"` + userLookupUser + `","user":{"id":"` + userLookupUser + `","username":"username-not-email","email":"reader@example.com","firstName":"Read","lastName":"Only"},"roles":[{"id":"role-1","role":"viewer","isTemporary":false},{"id":"role-2","role":"custom","customRoleId":"custom-1","customRoleSlug":"accessor","isTemporary":true,"temporaryMode":"relative","temporaryRange":"1h","temporaryAccessStartTime":"2026-01-01T00:00:00Z","temporaryAccessEndTime":"2026-01-01T01:00:00Z"}]}}`
	userLookupList    = `{"memberships":[{"id":"unrelated","userId":"other-user","user":{"id":"other-user","username":"reader@example.com"}},{"id":"membership-1","userId":"` + userLookupUser + `","user":{"id":"` + userLookupUser + `","username":"username-not-email"}}]}`
)

// projectUserLookupClient rejects every request not explicitly expected by a test,
// including any request that would create, update, or delete a membership.
func projectUserLookupClient(t *testing.T, handlers map[string]http.HandlerFunc) (Client, *[]string) {
	t.Helper()
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		calls = append(calls, key)
		handler, ok := handlers[key]
		if !ok {
			t.Errorf("unexpected request: %s", key)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return Client{Config: Config{HttpClient: resty.New().SetBaseURL(server.URL), IsMachineIdentityAuth: true}}, &calls
}

func missingUserLookupRoute() http.HandlerFunc {
	return jsonResponse(http.StatusNotFound, fmt.Sprintf(`{"statusCode":404,"message":%q,"error":"Not Found"}`, "Route GET:"+userLookupPath+" not found"))
}

func TestGetProjectMembershipByUserIDModernAPI(t *testing.T) {
	client, calls := projectUserLookupClient(t, map[string]http.HandlerFunc{
		"GET " + userLookupPath: jsonResponse(http.StatusOK, userLookupDetails),
	})
	got, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
	if err != nil {
		t.Fatal(err)
	}
	var want GetProjectUserByUserNameResponse
	if err := json.Unmarshal([]byte(userLookupDetails), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) || len(*calls) != 1 {
		t.Fatalf("modern lookup changed response or made extra requests: %+v, %v", got, *calls)
	}
}

func TestGetProjectMembershipByUserIDLegacyAPI(t *testing.T) {
	client, calls := projectUserLookupClient(t, map[string]http.HandlerFunc{
		"GET " + userLookupPath: missingUserLookupRoute(),
		"GET " + userListPath:   jsonResponse(http.StatusOK, userLookupList),
		"POST " + userDetailsPath: func(w http.ResponseWriter, r *http.Request) {
			var body GetProjectUserByUserNameRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode lookup body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body.Username != "username-not-email" || body.ProjectID != userLookupProject {
				t.Errorf("lookup must resolve the exact user ID to its username, not email: %+v", body)
			}
			jsonResponse(http.StatusOK, userLookupDetails)(w, r)
		},
	})
	got, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
	if err != nil {
		t.Fatal(err)
	}
	var want GetProjectUserByUserNameResponse
	if err := json.Unmarshal([]byte(userLookupDetails), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy lookup must preserve user details and all role/temporary fields: got %+v, want %+v", got, want)
	}
	wantCalls := []string{"GET " + userLookupPath, "GET " + userListPath, "POST " + userDetailsPath}
	if !reflect.DeepEqual(*calls, wantCalls) {
		t.Fatalf("unexpected fallback requests: %v", *calls)
	}
}

func TestGetProjectMembershipByUserIDDoesNotFallbackForOtherErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"missing membership", 404, `{"message":"Membership not found"}`},
		{"plain 404", 404, "Not Found"},
		{"empty 404", 404, ""},
		{"different route", 404, `{"message":"Route GET:/another-route not found"}`},
		{"unauthorized", 401, `{"message":"Unauthorized"}`},
		{"forbidden", 403, `{"message":"Forbidden"}`},
		{"server error", 500, `{"message":"Internal error"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := projectUserLookupClient(t, map[string]http.HandlerFunc{
				"GET " + userLookupPath: jsonResponse(tc.status, tc.body),
			})
			_, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
			if tc.status == 404 {
				if !errors.Is(err, ErrNotFound) {
					t.Fatalf("expected ErrNotFound, got %v", err)
				}
			} else {
				var apiErr *infisicalerrors.APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != tc.status {
					t.Fatalf("API error must stay classifiable: %v", err)
				}
			}
			if len(*calls) != 1 {
				t.Fatalf("unexpected fallback: %v", *calls)
			}
		})
	}
}

func TestGetProjectMembershipByUserIDLegacyMissingUser(t *testing.T) {
	client, calls := projectUserLookupClient(t, map[string]http.HandlerFunc{
		"GET " + userLookupPath: missingUserLookupRoute(),
		"GET " + userListPath:   jsonResponse(http.StatusOK, `{"memberships":[{"id":"other","userId":"other-user","user":{"username":"username-not-email"}}]}`),
	})
	_, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
	if !errors.Is(err, ErrNotFound) || len(*calls) != 2 {
		t.Fatalf("a list without the exact user ID must prove absence without username lookup: %v, %v", err, *calls)
	}
}

func TestGetProjectMembershipByUserIDLegacyListErrors(t *testing.T) {
	for _, status := range []int{401, 403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client, _ := projectUserLookupClient(t, map[string]http.HandlerFunc{
				"GET " + userLookupPath: missingUserLookupRoute(),
				"GET " + userListPath:   jsonResponse(status, `{"message":"List failed"}`),
			})
			_, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
			var apiErr *infisicalerrors.APIError
			if errors.Is(err, ErrNotFound) || !errors.As(err, &apiErr) || apiErr.StatusCode != status {
				t.Fatalf("a failed list cannot prove absence and must preserve its error: %v", err)
			}
		})
	}
}

func TestGetProjectMembershipByUserIDLegacyRejectsChangedIdentity(t *testing.T) {
	for name, body := range map[string]string{
		"user changed":       `{"membership":{"id":"membership-1","user":{"id":"other-user"}}}`,
		"membership changed": `{"membership":{"id":"other-membership","user":{"id":"` + userLookupUser + `"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := projectUserLookupClient(t, map[string]http.HandlerFunc{
				"GET " + userLookupPath:   missingUserLookupRoute(),
				"GET " + userListPath:     jsonResponse(http.StatusOK, userLookupList),
				"POST " + userDetailsPath: jsonResponse(http.StatusOK, body),
			})
			_, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
			if err == nil || errors.Is(err, ErrNotFound) {
				t.Fatalf("identity mismatch must fail without reporting absence: %v", err)
			}
		})
	}
}

func TestGetProjectMembershipByUserIDLegacyDetailsErrors(t *testing.T) {
	for _, status := range []int{403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client, _ := projectUserLookupClient(t, map[string]http.HandlerFunc{
				"GET " + userLookupPath:   missingUserLookupRoute(),
				"GET " + userListPath:     jsonResponse(http.StatusOK, userLookupList),
				"POST " + userDetailsPath: jsonResponse(status, `{"message":"Details failed"}`),
			})
			_, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
			if status == 404 {
				if !errors.Is(err, ErrNotFound) {
					t.Fatalf("membership may disappear between list and lookup: %v", err)
				}
				return
			}
			var apiErr *infisicalerrors.APIError
			if errors.Is(err, ErrNotFound) || !errors.As(err, &apiErr) || apiErr.StatusCode != status {
				t.Fatalf("failed details lookup must preserve its error: %v", err)
			}
		})
	}
}

func TestGetProjectMembershipByUserIDLegacyMalformedList(t *testing.T) {
	for name, body := range map[string]string{
		"malformed JSON":   `{"memberships":`,
		"missing list":     `{}`,
		"null list":        `{"memberships":null}`,
		"missing ID":       `{"memberships":[{"userId":"` + userLookupUser + `","user":{"username":"reader"}}]}`,
		"missing username": `{"memberships":[{"id":"membership-1","userId":"` + userLookupUser + `","user":{"id":"` + userLookupUser + `"}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := projectUserLookupClient(t, map[string]http.HandlerFunc{
				"GET " + userLookupPath: missingUserLookupRoute(),
				"GET " + userListPath:   jsonResponse(http.StatusOK, body),
			})
			_, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
			if err == nil || errors.Is(err, ErrNotFound) {
				t.Fatalf("malformed list must not be treated as absence: %v", err)
			}
		})
	}
}

type projectUserLookupTransport func(*http.Request) (*http.Response, error)

func (f projectUserLookupTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGetProjectMembershipByUserIDTransportFailure(t *testing.T) {
	calls := 0
	client := Client{Config: Config{HttpClient: resty.New().SetBaseURL("https://example.invalid").SetTransport(
		projectUserLookupTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, errors.New("transport unavailable")
		}),
	)}}
	_, err := client.GetProjectMembershipByUserID(GetProjectMembershipByUserIDRequest{ProjectID: userLookupProject, UserID: userLookupUser})
	var requestErr *infisicalerrors.GenericRequestError
	if !errors.As(err, &requestErr) || errors.Is(err, ErrNotFound) || calls != 1 {
		t.Fatalf("transport failure must not panic, trigger fallback or prove absence: %v (calls=%d)", err, calls)
	}
}
