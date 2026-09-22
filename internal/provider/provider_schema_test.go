package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// GetProviderSchema is where the framework runs ValidateImplementation over every resource schema,
// so a schema the framework rejects fails here rather than on a user's first plan.
func TestProviderSchemaIsValid(t *testing.T) {
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatalf("expected the provider server to start, got: %v", err)
	}

	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("expected the provider schema to be served, got: %v", err)
	}

	for _, diagnostic := range resp.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("provider schema is invalid: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}

	for _, name := range []string{"infisical_gateway", "infisical_gateway_enrollment_token"} {
		if _, ok := resp.ResourceSchemas[name]; !ok {
			t.Errorf("expected %s to be registered on the provider", name)
		}
	}
}
