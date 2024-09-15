package pkg

import "strings"

func StringSplitAndTrim(input string, separator string) []string {
	splittedStrings := strings.Split(input, separator)
	for i := 0; i < len(splittedStrings); i++ {
		splittedStrings[i] = strings.TrimSpace(splittedStrings[i])
	}
	return splittedStrings
}
