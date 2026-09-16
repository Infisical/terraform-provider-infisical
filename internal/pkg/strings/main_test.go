package pkg

import (
	"reflect"
	"testing"
)

func TestStringSplitAndTrim(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty input yields no elements", input: "", want: []string{}},
		{name: "whitespace only input yields no elements", input: "   ", want: []string{}},
		{name: "single value", input: "project-a", want: []string{"project-a"}},
		{name: "trims surrounding whitespace", input: "  project-a ,	project-b  ", want: []string{"project-a", "project-b"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StringSplitAndTrim(tc.input, ",")
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("StringSplitAndTrim(%q) = %#v, want %#v", tc.input, got, tc.want)
			}
		})
	}
}
