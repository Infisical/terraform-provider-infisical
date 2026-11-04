package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func kubernetesAuthTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	NewIdentityKubernetesAuthResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestIdentityKubernetesAuthSchemaIsValid(t *testing.T) {
	resourceSchema := kubernetesAuthTestSchema(t)
	if diags := resourceSchema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("schema implementation: %v", diags)
	}

	if !resourceSchema.Attributes["token_reviewer_jwt"].IsSensitive() {
		t.Error("expected token_reviewer_jwt to be sensitive")
	}
	if wo := resourceSchema.Attributes["token_reviewer_jwt_wo"]; !wo.IsWriteOnly() || !wo.IsSensitive() {
		t.Error("expected token_reviewer_jwt_wo to be write-only and sensitive")
	}
}

func TestConfiguredTokenReviewerJwt(t *testing.T) {
	ctx := context.Background()
	resourceSchema := kubernetesAuthTestSchema(t)

	tests := map[string]struct {
		planJwt   types.String
		configJwt types.String
		writeOnly types.String
		want      types.String
	}{
		"neither set":          {types.StringNull(), types.StringNull(), types.StringNull(), types.StringNull()},
		"plain JWT":            {types.StringValue("eyJ.plain"), types.StringValue("eyJ.plain"), types.StringNull(), types.StringValue("eyJ.plain")},
		"empty string removes": {types.StringValue(""), types.StringValue(""), types.StringNull(), types.StringValue("")},
		"write-only JWT":       {types.StringNull(), types.StringNull(), types.StringValue("eyJ.wo"), types.StringValue("eyJ.wo")},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			configModel := nullKubernetesAuthModel()
			configModel.TokenReviewerJWT = tc.configJwt
			configModel.TokenReviewerJWTWO = tc.writeOnly
			holder := tfsdk.State{Schema: resourceSchema}
			if diags := holder.Set(ctx, &configModel); diags.HasError() {
				t.Fatal(diags)
			}
			config := tfsdk.Config{Schema: resourceSchema, Raw: holder.Raw}

			plan := nullKubernetesAuthModel()
			plan.TokenReviewerJWT = tc.planJwt

			var diags diag.Diagnostics
			got := configuredTokenReviewerJwt(ctx, &diags, config, &plan)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func nullKubernetesAuthModel() IdentityKubernetesAuthResourceModel {
	return IdentityKubernetesAuthResourceModel{
		AllowedServiceAccountNames: types.ListNull(types.StringType),
		AllowedNamespaces:          types.ListNull(types.StringType),
		AccessTokenTrustedIps: types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{"ip_address": types.StringType},
		}),
	}
}
