package pkg

import (
	"encoding/json"
	"strings"
)

func StringSplitAndTrim(input string, separator string) []string {
	// strings.Split returns a one-element slice holding "" for an empty input, which surfaces in
	// Terraform state as a list containing an empty string rather than the empty list the user configured.
	if strings.TrimSpace(input) == "" {
		return []string{}
	}

	splittedStrings := strings.Split(input, separator)
	for i, s := range splittedStrings {
		splittedStrings[i] = strings.TrimSpace(s)
	}
	return splittedStrings
}

func NormalizeJSON(input string) (string, error) {
	var parsed interface{}
	if err := json.Unmarshal([]byte(input), &parsed); err != nil {
		return "", err
	}

	// Use canonical JSON encoding settings
	normalized, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}

	return string(normalized), nil
}

func StringToPtr(s string) *string {
	return &s
}
