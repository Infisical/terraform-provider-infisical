package pkg

import (
	"encoding/json"
	"strings"
)

func StringSplitAndTrim(input string, separator string) []string {
	splittedStrings := strings.Split(input, separator)
	for i := 0; i < len(splittedStrings); i++ {
		splittedStrings[i] = strings.TrimSpace(splittedStrings[i])
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
