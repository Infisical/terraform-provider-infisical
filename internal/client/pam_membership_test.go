package infisicalclient

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func pamMembersHandler(members string) http.HandlerFunc {
	return jsonResponse(http.StatusOK, fmt.Sprintf(`{"members":[%s]}`, members))
}

func pamMember(userID, email, username string) string {
	return fmt.Sprintf(`{"membershipId":"m-%s","userId":%q,"email":%q,"username":%q,"role":"member"}`, userID, userID, email, username)
}

func TestFindPamProductUserByEmail(t *testing.T) {
	tests := map[string]struct {
		members   []string
		wantID    string
		wantError string
	}{
		"an email match wins over a username match": {
			members: []string{pamMember("u1", "other@acme.com", "alice@acme.com"), pamMember("u2", "alice@acme.com", "alice")},
			wantID:  "u2",
		},
		"a username match when no email matches": {
			members: []string{pamMember("u1", "a@acme.com", "Alice@acme.com")},
			wantID:  "u1",
		},
		"two users with the email": {
			members:   []string{pamMember("u1", "alice@acme.com", "alice-ldap"), pamMember("u2", "alice@acme.com", "alice@acme.com")},
			wantError: "more than one PAM member",
		},
		"no match": {
			members:   []string{pamMember("u1", "bob@acme.com", "bob@acme.com")},
			wantError: ErrNotFound.Error(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client := testClient(t, pamMembersHandler(strings.Join(test.members, ",")))

			userID, err := client.FindPamProductUserByEmail("alice@acme.com")
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("expected an error containing %q, got %v (user %q)", test.wantError, err, userID)
				}
				if test.wantError == ErrNotFound.Error() && !errors.Is(err, ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
				return
			}
			if err != nil || userID != test.wantID {
				t.Errorf("expected user %q, got %q (err %v)", test.wantID, userID, err)
			}
		})
	}
}
