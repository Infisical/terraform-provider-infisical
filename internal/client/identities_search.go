package infisicalclient

import (
	"fmt"
	"strings"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationSearchIdentityIDsByName = "CallSearchIdentityIDsByName"
)

type SearchIdentityIDsByNameRequest struct {
	IdentityName string
	Mode         string   // "eq" | "contains"
	Scopes       []string // "organization" | "project"
	Limit        int      // optional; defaults to 100
}

type identitySearchResponse struct {
	Identities []IdentitySearchMatch `json:"identities"`
	TotalCount int                   `json:"totalCount"`
}

type IdentitySearchMatch struct {
	// Top-level identifier fields from `/api/v2/identities/search`.
	ID         string `json:"id"`
	IdentityID string `json:"identityId"`

	Scope  string `json:"scope"`
	OrgID  string `json:"orgId"`
	ProjectID *string `json:"projectId"`
	Project *IdentitySearchProject `json:"project"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`

	LastLoginAuthMethod *string `json:"lastLoginAuthMethod"`
	LastLoginTime       *string `json:"lastLoginTime"`

	Roles    []IdentitySearchRole    `json:"roles"`
	Identity IdentitySearchIdentity `json:"identity"`
}

type IdentitySearchProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"`
}

type IdentitySearchRole struct {
	ID   string `json:"id"`
	Role string `json:"role"`

	CustomRoleID          *string `json:"customRoleId"`
	CustomRoleName        *string `json:"customRoleName"`
	CustomRoleSlug        *string `json:"customRoleSlug"`
	CustomRoleDescription *string `json:"customRoleDescription"`

	IsTemporary bool `json:"isTemporary"`

	TemporaryMode  *string `json:"temporaryMode"`
	TemporaryRange *string `json:"temporaryRange"`

	TemporaryAccessStartTime *string `json:"temporaryAccessStartTime"`
	TemporaryAccessEndTime   *string `json:"temporaryAccessEndTime"`
}

type IdentitySearchIdentity struct {
	Name                string   `json:"name"`
	ID                  string   `json:"id"`
	HasDeleteProtection bool     `json:"hasDeleteProtection"`
	OrgID               string   `json:"orgId"`
	AuthMethods         []string `json:"authMethods"`

	// Field is present in Infisical v2 responses.
	ActiveLockoutAuthMethods []string `json:"activeLockoutAuthMethods"`
}

// SearchIdentityIDsByName searches for identities and returns:
//  - identityIds: deduped list of matching identity ids
//  - identities: deduped list of full, typed match objects
//  - totalCount: total matches as returned by Infisical for the last page request
func (client Client) SearchIdentityIDsByName(request SearchIdentityIDsByNameRequest) ([]string, []IdentitySearchMatch, int, error) {
	identityName := strings.TrimSpace(request.IdentityName)
	if identityName == "" {
		return nil, nil, 0, fmt.Errorf("%s: identity name cannot be empty", operationSearchIdentityIDsByName)
	}

	mode := request.Mode
	if mode == "" {
		mode = "contains"
	}

	scopes := request.Scopes
	if len(scopes) == 0 {
		scopes = []string{"organization", "project"}
	}

	limit := request.Limit
	if limit <= 0 || limit > 100 {
		limit = 100 // API docs maximum
	}

	offset := 0
	seen := make(map[string]struct{}, limit) // dedupe by identity id
	identityIDs := make([]string, 0, limit)
	identities := make([]IdentitySearchMatch, 0, limit)

	var totalCount int
	for {
		nameOperator := "$contains"
		if mode == "eq" {
			nameOperator = "$eq"
		}

		body := map[string]any{
			"scope":           scopes,
			"orderBy":         "name",
			"orderDirection":  "asc",
			"limit":           limit,
			"offset":          offset,
			"search": map[string]any{
				"name": map[string]any{
					nameOperator: identityName,
				},
			},
		}

		var respBody identitySearchResponse
		httpRequest := client.Config.HttpClient.
			R().
			SetResult(&respBody).
			SetHeader("User-Agent", USER_AGENT)

		response, err := httpRequest.SetBody(body).Post("api/v2/identities/search")
		if err != nil {
			return nil, nil, 0, errors.NewGenericRequestError(operationSearchIdentityIDsByName, err)
		}

		if response.IsError() {
			return nil, nil, 0, errors.NewAPIErrorWithResponse(operationSearchIdentityIDsByName, response, nil)
		}

		totalCount = respBody.TotalCount

		for _, item := range respBody.Identities {
			identityID := strings.TrimSpace(item.IdentityID)
			if identityID == "" {
				identityID = strings.TrimSpace(item.Identity.ID)
			}
			if identityID == "" {
				continue
			}

			if _, ok := seen[identityID]; ok {
				continue
			}

			seen[identityID] = struct{}{}
			identityIDs = append(identityIDs, identityID)
			// Ensure the returned match has `identityId` populated consistently for Terraform consumers.
			if item.IdentityID == "" {
				item.IdentityID = identityID
			}
			identities = append(identities, item)
		}

		// Stop conditions:
		// - fetched fewer than the requested limit (likely last page)
		// - or we advanced beyond totalCount
		if len(respBody.Identities) < limit || offset+limit >= totalCount {
			break
		}

		offset += limit
	}

	return identityIDs, identities, totalCount, nil
}

