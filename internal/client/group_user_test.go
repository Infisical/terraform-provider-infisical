package infisicalclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

const groupUserJSON = `{"id":"u1","email":"alice@example.com","username":"alice@example.com","firstName":"Alice","lastName":"A"}`

func TestGroupUserNotFound(t *testing.T) {
	notFound := jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"Failed to find group with ID missing"}`)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/groups/missing/users", notFound)
	mux.HandleFunc("/api/v1/groups/missing/users/", notFound)
	client := testClient(t, mux.ServeHTTP)

	if _, err := client.ListGroupUsers(ListGroupUsersRequest{GroupID: "missing"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound from list, got: %v", err)
	}

	if err := client.RemoveGroupUser(RemoveGroupUserRequest{GroupID: "missing", Username: "alice@example.com"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound from remove, got: %v", err)
	}

	// A 404 on add means the group or the username does not exist. The user has to see that error.
	_, err := client.AddGroupUser(AddGroupUserRequest{GroupID: "missing", Username: "alice@example.com"})
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a non-ErrNotFound error from add, got: %v", err)
	}
}

// The API answers 403 both for a missing permission and for a user who is not in the group.
func TestGroupUserForbidden(t *testing.T) {
	forbidden := jsonResponse(http.StatusForbidden, `{"statusCode":403,"message":"User(s) are not part of the group platform"}`)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/groups/g1/users", forbidden)
	mux.HandleFunc("/api/v1/groups/g1/users/", forbidden)
	client := testClient(t, mux.ServeHTTP)

	if _, err := client.ListGroupUsers(ListGroupUsersRequest{GroupID: "g1"}); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a non-ErrNotFound error from list, got: %v", err)
	}

	if err := client.RemoveGroupUser(RemoveGroupUserRequest{GroupID: "g1", Username: "alice@example.com"}); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a non-ErrNotFound error from remove, got: %v", err)
	}
}

// The username is user input that goes into the URL path, so it must arrive intact.
func TestAddAndRemoveGroupUserRequest(t *testing.T) {
	for name, tc := range map[string]struct {
		username string
		wantPath string
	}{
		"plain email":      {username: "alice@example.com", wantPath: "/api/v1/groups/g1/users/alice@example.com"},
		"plus addressing":  {username: "a+b@example.com", wantPath: "/api/v1/groups/g1/users/a+b@example.com"},
		"space and slash":  {username: "odd name/x@example.com", wantPath: "/api/v1/groups/g1/users/odd%20name%2Fx@example.com"},
		"percent in value": {username: "100%@example.com", wantPath: "/api/v1/groups/g1/users/100%25@example.com"},
	} {
		t.Run(name, func(t *testing.T) {
			var gotMethods, gotPaths []string
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/groups/g1/users/", func(w http.ResponseWriter, r *http.Request) {
				gotMethods = append(gotMethods, r.Method)
				gotPaths = append(gotPaths, r.URL.EscapedPath())
				jsonResponse(http.StatusOK, groupUserJSON)(w, r)
			})
			client := testClient(t, mux.ServeHTTP)

			added, err := client.AddGroupUser(AddGroupUserRequest{GroupID: "g1", Username: tc.username})
			if err != nil {
				t.Fatalf("expected the user to be added, got: %v", err)
			}
			if added.ID != "u1" || added.Username != "alice@example.com" {
				t.Errorf("unexpected user from add: %+v", added)
			}

			if err := client.RemoveGroupUser(RemoveGroupUserRequest{GroupID: "g1", Username: tc.username}); err != nil {
				t.Fatalf("expected the user to be removed, got: %v", err)
			}
			if len(gotMethods) != 2 || gotMethods[0] != http.MethodPost || gotMethods[1] != http.MethodDelete {
				t.Errorf("expected POST then DELETE, got %v", gotMethods)
			}
			for _, got := range gotPaths {
				if got != tc.wantPath {
					t.Errorf("expected path %s, got %s", tc.wantPath, got)
				}
			}
		})
	}
}

// Without filter=existingMembers the API returns every user in the organization, and without an explicit limit it returns only 10.
func TestListGroupUsersQuery(t *testing.T) {
	var queries []map[string][]string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/groups/g1/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		queries = append(queries, r.URL.Query())
		jsonResponse(http.StatusOK, `{"users":[{"id":"u1","email":"alice@example.com","username":"alice@example.com","firstName":"Alice","lastName":"A","isPartOfGroup":true}],"totalCount":1}`)(w, r)
	})
	client := testClient(t, mux.ServeHTTP)

	users, err := client.ListGroupUsers(ListGroupUsersRequest{GroupID: "g1", Username: "alice@example.com"})
	if err != nil {
		t.Fatalf("expected the list to succeed, got: %v", err)
	}
	if len(users) != 1 || users[0].ID != "u1" || users[0].Username != "alice@example.com" || !users[0].IsPartOfGroup {
		t.Errorf("unexpected users: %+v", users)
	}

	if _, err := client.ListGroupUsers(ListGroupUsersRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("expected the unfiltered list to succeed, got: %v", err)
	}

	if len(queries) != 2 {
		t.Fatalf("expected one request per call, got %d", len(queries))
	}

	for i, query := range queries {
		for key, want := range map[string]string{"filter": "existingMembers", "limit": "100", "offset": "0"} {
			if got := query[key]; len(got) != 1 || got[0] != want {
				t.Errorf("request %d: expected %s=%s, got %v", i, key, want, got)
			}
		}
	}

	if got := queries[0]["username"]; len(got) != 1 || got[0] != "alice@example.com" {
		t.Errorf("expected the username filter to be sent, got %v", got)
	}
	if _, ok := queries[1]["username"]; ok {
		t.Error("an empty username must not be sent as a filter")
	}
}

func TestListGroupUsersPaginates(t *testing.T) {
	for name, total := range map[string]int{
		"empty group":           0,
		"single page":           3,
		"exactly one full page": 100,
		"second page":           150,
		"exactly two pages":     200,
	} {
		t.Run(name, func(t *testing.T) {
			requests := 0
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/groups/g1/users", func(w http.ResponseWriter, r *http.Request) {
				requests++
				offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

				page := []GroupUser{}
				for i := offset; i < total && i < offset+limit; i++ {
					page = append(page, GroupUser{ID: fmt.Sprintf("u%d", i), Username: fmt.Sprintf("user%d@example.com", i), IsPartOfGroup: true})
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(ListGroupUsersResponse{GroupUsers: page, TotalCount: total})
			})
			client := testClient(t, mux.ServeHTTP)

			users, err := client.ListGroupUsers(ListGroupUsersRequest{GroupID: "g1"})
			if err != nil {
				t.Fatalf("expected the list to succeed, got: %v", err)
			}
			if len(users) != total {
				t.Fatalf("expected %d users, got %d", total, len(users))
			}
			for i, user := range users {
				if want := fmt.Sprintf("u%d", i); user.ID != want {
					t.Fatalf("expected user %s at position %d, got %s", want, i, user.ID)
				}
			}

			wantRequests := (total + 99) / 100
			if wantRequests == 0 {
				wantRequests = 1
			}
			if requests != wantRequests {
				t.Errorf("expected %d request(s), got %d", wantRequests, requests)
			}
		})
	}
}
