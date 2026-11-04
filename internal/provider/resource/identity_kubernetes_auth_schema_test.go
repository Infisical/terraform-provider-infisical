package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestIdentityKubernetesAuthSchemaIsValid(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewIdentityKubernetesAuthResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("schema implementation: %v", diags)
	}

	jwt := resp.Schema.Attributes["token_reviewer_jwt"]
	if !jwt.IsSensitive() {
		t.Error("expected token_reviewer_jwt to be sensitive")
	}
	if wo := resp.Schema.Attributes["token_reviewer_jwt_wo"]; !wo.IsWriteOnly() || !wo.IsSensitive() {
		t.Error("expected token_reviewer_jwt_wo to be write-only and sensitive")
	}
}
