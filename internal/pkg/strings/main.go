package pkg

import "strings"

func StringSplitAndTrim(input string, seperator string) []string {
	splittedStrings := strings.Split(input, seperator)
	for i := 0; i < len(splittedStrings); i++ {
		splittedStrings[i] = strings.TrimSpace(splittedStrings[i])
	}
	return splittedStrings
}
