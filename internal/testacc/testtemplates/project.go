package testtemplates

import (
	"bytes"
	"html/template"
	"testing"
)

func ProjectResource(t *testing.T, resourceName string, projectName string) string {
	tmpl, err := template.New("").Parse(`
resource "infisical_project" "{{ .resourceName }}" {
  name = "{{ .projectName }}"
  slug = "test"
}`)

	if err != nil {
		t.Errorf("Failed to compile template %v", err)
	}

	data := map[string]string{
		"resourceName": resourceName,
		"projectName":  projectName,
	}

	var result bytes.Buffer
	if err := tmpl.Execute(&result, data); err != nil {
		t.Errorf("Failed to compile template %v", err)
	}

	return result.String()
}
