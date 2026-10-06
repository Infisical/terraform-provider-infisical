package resource

import (
	"context"
	"testing"

	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func baseKubernetesTemplateModel() IdentityKubernetesAuthTemplateResourceModel {
	return IdentityKubernetesAuthTemplateResourceModel{
		ID:                   types.StringValue("template-id"),
		Name:                 types.StringValue("prod"),
		TokenReviewerMode:    types.StringValue(TOKEN_REVIEWER_MODE_API),
		KubernetesHost:       types.StringValue("https://cluster:6443"),
		CaCertificate:        customtypes.NewTrimmedStringValue("CA"),
		VerifyTlsCertificate: types.BoolValue(true),
		TokenReviewerJWT:     types.StringValue("jwt"),
		HasTokenReviewerJWT:  types.BoolValue(true),
		GatewayID:            types.StringNull(),
		GatewayPoolID:        types.StringNull(),
		AllowedAudience:      types.StringValue(""),
	}
}

// Every template edit fans out to all linked identities, and the API re-authorizes gateway
// access whenever a gateway key is present, so an unchanged plan must patch nothing.
func TestKubernetesAuthTemplatePatchIsEmptyWhenNothingChanged(t *testing.T) {
	model := baseKubernetesTemplateModel()
	if patch := kubernetesAuthTemplateFieldsPatch(model, model); len(patch) != 0 {
		t.Errorf("expected no fields, got %v", patch)
	}
}

// The trailing newline file() adds is the same certificate, and the API strips it anyway.
func TestKubernetesAuthTemplatePatchIgnoresCertificateWhitespace(t *testing.T) {
	state := baseKubernetesTemplateModel()
	plan := state
	plan.CaCertificate = customtypes.NewTrimmedStringValue("CA\n")

	if patch := kubernetesAuthTemplateFieldsPatch(plan, state); len(patch) != 0 {
		t.Errorf("expected a whitespace-only difference to patch nothing, got %v", patch)
	}
}

// The API re-derives verification from a certificate patched alone, so clearing the certificate
// while keeping verification explicit must carry the planned verification with it.
func TestKubernetesAuthTemplatePatchSendsVerificationWithCertificate(t *testing.T) {
	state := baseKubernetesTemplateModel()
	plan := state
	plan.CaCertificate = customtypes.NewTrimmedStringValue("NEW-CA")

	patch := kubernetesAuthTemplateFieldsPatch(plan, state)
	if patch["caCert"] != "NEW-CA" {
		t.Errorf("expected the new certificate, got %v", patch["caCert"])
	}
	if patch["verifyTlsCertificate"] != true {
		t.Errorf("expected verification to travel with the certificate, got %v", patch["verifyTlsCertificate"])
	}
	if len(patch) != 2 {
		t.Errorf("expected only the certificate and verification, got %v", patch)
	}
}

func TestKubernetesAuthTemplatePatchClearsAndKeepsTheReviewerJwt(t *testing.T) {
	state := baseKubernetesTemplateModel()

	t.Run("removed from config clears it", func(t *testing.T) {
		plan := state
		plan.TokenReviewerJWT = types.StringNull()
		patch := kubernetesAuthTemplateFieldsPatch(plan, state)
		if value, present := patch["tokenReviewerJwt"]; !present || value != "" {
			t.Errorf("expected an empty string to clear the JWT, got %v (present: %v)", value, present)
		}
	})

	// After an import the JWT is in Infisical but not in state. Leaving the key out is what
	// keeps it there.
	t.Run("absent from both keeps it", func(t *testing.T) {
		imported := state
		imported.TokenReviewerJWT = types.StringNull()
		if _, present := kubernetesAuthTemplateFieldsPatch(imported, imported)["tokenReviewerJwt"]; present {
			t.Error("expected an unmanaged JWT to be left out of the patch")
		}
	})
}

// Moving from a gateway to a pool must clear the gateway in the same merge, or the API sees both
// and rejects the update.
func TestKubernetesAuthTemplatePatchSendsGatewayFieldsAsAPair(t *testing.T) {
	state := baseKubernetesTemplateModel()
	state.TokenReviewerMode = types.StringValue(TOKEN_REVIEWER_MODE_GATEWAY)
	state.GatewayID = types.StringValue("gateway-1")

	plan := state
	plan.GatewayID = types.StringNull()
	plan.GatewayPoolID = types.StringValue("pool-1")

	patch := kubernetesAuthTemplateFieldsPatch(plan, state)
	gatewayID, isPtr := patch["gatewayId"].(*string)
	if !isPtr || gatewayID != nil {
		t.Errorf("expected gatewayId to be sent as null, got %v", patch["gatewayId"])
	}
	if poolID, _ := patch["gatewayPoolId"].(*string); poolID == nil || *poolID != "pool-1" {
		t.Errorf("expected the new pool, got %v", patch["gatewayPoolId"])
	}
}

func TestKubernetesAuthTemplatePatchSendsNullToClearTheHost(t *testing.T) {
	state := baseKubernetesTemplateModel()
	plan := state
	plan.KubernetesHost = types.StringNull()

	patch := kubernetesAuthTemplateFieldsPatch(plan, state)
	host, isPtr := patch["kubernetesHost"].(*string)
	if !isPtr || host != nil {
		t.Errorf("expected kubernetesHost to be sent as null, got %v", patch["kubernetesHost"])
	}
}

func TestPlannedHasTokenReviewerJwt(t *testing.T) {
	state := baseKubernetesTemplateModel()
	imported := state
	imported.TokenReviewerJWT = types.StringNull()

	cases := map[string]struct {
		planned types.String
		state   *IdentityKubernetesAuthTemplateResourceModel
		want    types.Bool
	}{
		"create with a JWT":           {types.StringValue("jwt"), nil, types.BoolValue(true)},
		"create without a JWT":        {types.StringNull(), nil, types.BoolValue(false)},
		"unknown JWT":                 {types.StringUnknown(), &state, types.BoolUnknown()},
		"unchanged JWT":               {types.StringValue("jwt"), &state, types.BoolValue(true)},
		"JWT removed":                 {types.StringNull(), &state, types.BoolValue(false)},
		"JWT replaced":                {types.StringValue("other"), &state, types.BoolValue(true)},
		"imported JWT stays reported": {types.StringNull(), &imported, types.BoolValue(true)},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := plannedHasTokenReviewerJwt(tc.planned, tc.state); !got.Equal(tc.want) {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestApplyKubernetesAuthTemplateToModel(t *testing.T) {
	template := infisical.IdentityKubernetesAuthTemplate{
		ID:   "template-id",
		Name: "prod",
		TemplateFields: infisical.IdentityKubernetesAuthTemplateFields{
			TokenReviewMode:      TOKEN_REVIEWER_MODE_API,
			KubernetesHost:       strPtr("https://cluster:6443"),
			CaCert:               "CA",
			VerifyTlsCertificate: boolPtr(true),
			HasTokenReviewerJwt:  false,
			AllowedAudience:      "",
		},
	}

	// A JWT cleared outside Terraform has to leave state, or the plan would never put it back.
	t.Run("JWT cleared outside Terraform", func(t *testing.T) {
		model := baseKubernetesTemplateModel()
		applyKubernetesAuthTemplateToModel(&model, template)
		if !model.TokenReviewerJWT.IsNull() {
			t.Errorf("expected the JWT to drop from state, got %v", model.TokenReviewerJWT)
		}
		if model.HasTokenReviewerJWT.ValueBool() {
			t.Error("expected has_token_reviewer_jwt to be false")
		}
	})

	t.Run("JWT held in Infisical is kept", func(t *testing.T) {
		model := baseKubernetesTemplateModel()
		withJwt := template
		withJwt.TemplateFields.HasTokenReviewerJwt = true
		applyKubernetesAuthTemplateToModel(&model, withJwt)
		if model.TokenReviewerJWT.ValueString() != "jwt" {
			t.Errorf("expected the configured JWT to stay, got %v", model.TokenReviewerJWT)
		}
	})

	// An unset optional must stay null.
	t.Run("empty API values keep the prior spelling", func(t *testing.T) {
		empty := template
		empty.TemplateFields.KubernetesHost = nil
		empty.TemplateFields.CaCert = ""
		empty.TemplateFields.GatewayID = strPtr("")

		model := IdentityKubernetesAuthTemplateResourceModel{
			KubernetesHost: types.StringNull(),
			CaCertificate:  customtypes.NewTrimmedStringNull(),
			GatewayID:      types.StringValue(""),
		}
		applyKubernetesAuthTemplateToModel(&model, empty)

		if !model.KubernetesHost.IsNull() {
			t.Errorf("expected a null host, got %v", model.KubernetesHost)
		}
		if !model.CaCertificate.IsNull() {
			t.Errorf("expected a null certificate, got %v", model.CaCertificate)
		}
		if model.GatewayID.IsNull() || model.GatewayID.ValueString() != "" {
			t.Errorf("expected the configured empty gateway to stay, got %v", model.GatewayID)
		}
	})

	// Templates written before the field existed have no verification flag, and the API treats
	// them as verifying exactly when a certificate is set.
	t.Run("missing verification follows the certificate", func(t *testing.T) {
		legacy := template
		legacy.TemplateFields.VerifyTlsCertificate = nil
		model := IdentityKubernetesAuthTemplateResourceModel{}
		applyKubernetesAuthTemplateToModel(&model, legacy)
		if !model.VerifyTlsCertificate.ValueBool() {
			t.Error("expected verification to be on when a certificate is set")
		}
	})
}

func baseOidcTemplateModel(t *testing.T) IdentityOidcAuthTemplateResourceModel {
	audiences, diags := types.ListValue(types.StringType, []attr.Value{types.StringValue("a"), types.StringValue("b")})
	if diags.HasError() {
		t.Fatalf("expected the audiences to build, got %v", diags)
	}
	return IdentityOidcAuthTemplateResourceModel{
		ID:               types.StringValue("template-id"),
		Name:             types.StringValue("github"),
		OidcDiscoveryUrl: types.StringValue("https://issuer.example.com"),
		BoundIssuer:      types.StringValue("https://issuer.example.com"),
		BoundAudiences:   audiences,
		CaCertificate:    customtypes.NewTrimmedStringNull(),
	}
}

func TestOidcAuthTemplatePatch(t *testing.T) {
	ctx := context.Background()
	state := baseOidcTemplateModel(t)

	t.Run("nothing changed", func(t *testing.T) {
		var diags diag.Diagnostics
		if patch := oidcAuthTemplateFieldsPatch(ctx, &diags, state, state); len(patch) != 0 {
			t.Errorf("expected no fields, got %v", patch)
		}
	})

	// The API strips a trailing slash, so it is the same URL and must not propagate a no-op edit.
	t.Run("trailing slash is not a change", func(t *testing.T) {
		plan := state
		plan.OidcDiscoveryUrl = types.StringValue("https://issuer.example.com/")
		var diags diag.Diagnostics
		if patch := oidcAuthTemplateFieldsPatch(ctx, &diags, plan, state); len(patch) != 0 {
			t.Errorf("expected no fields, got %v", patch)
		}
	})

	t.Run("audiences are joined", func(t *testing.T) {
		plan := state
		audiences, _ := types.ListValue(types.StringType, []attr.Value{types.StringValue("c"), types.StringValue("d")})
		plan.BoundAudiences = audiences
		var diags diag.Diagnostics
		patch := oidcAuthTemplateFieldsPatch(ctx, &diags, plan, state)
		if patch["boundAudiences"] != "c,d" || len(patch) != 1 {
			t.Errorf("expected only the joined audiences, got %v", patch)
		}
	})

	t.Run("removing the certificate clears it", func(t *testing.T) {
		withCa := state
		withCa.CaCertificate = customtypes.NewTrimmedStringValue("CA")
		var diags diag.Diagnostics
		patch := oidcAuthTemplateFieldsPatch(ctx, &diags, state, withCa)
		if value, present := patch["caCert"]; !present || value != "" {
			t.Errorf("expected an empty string to clear the certificate, got %v (present: %v)", value, present)
		}
	})
}

func TestApplyOidcAuthTemplateToModelKeepsConfiguredUrlSpelling(t *testing.T) {
	ctx := context.Background()
	template := infisical.IdentityOidcAuthTemplate{
		ID:   "template-id",
		Name: "github",
		TemplateFields: infisical.IdentityOidcAuthTemplateFields{
			OidcDiscoveryUrl: "https://issuer.example.com",
			BoundIssuer:      "https://issuer.example.com",
			BoundAudiences:   "a, b",
		},
	}

	model := baseOidcTemplateModel(t)
	model.OidcDiscoveryUrl = types.StringValue("https://issuer.example.com/")
	if diags := applyOidcAuthTemplateToModel(ctx, &model, template); diags.HasError() {
		t.Fatalf("expected the template to map, got %v", diags)
	}
	if model.OidcDiscoveryUrl.ValueString() != "https://issuer.example.com/" {
		t.Errorf("expected the configured spelling to be kept, got %v", model.OidcDiscoveryUrl)
	}

	var audiences []string
	model.BoundAudiences.ElementsAs(ctx, &audiences, false)
	if len(audiences) != 2 || audiences[0] != "a" || audiences[1] != "b" {
		t.Errorf("expected the API's \"a, b\" to split into two audiences, got %v", audiences)
	}

	// A real change made outside Terraform must show up as drift.
	moved := template
	moved.TemplateFields.OidcDiscoveryUrl = "https://other.example.com"
	applyOidcAuthTemplateToModel(ctx, &model, moved)
	if model.OidcDiscoveryUrl.ValueString() != "https://other.example.com" {
		t.Errorf("expected a different URL to replace state, got %v", model.OidcDiscoveryUrl)
	}
}

func TestOidcDiscoveryUrlValidator(t *testing.T) {
	cases := map[string]bool{
		"https://token.actions.githubusercontent.com":                                 true,
		"https://keycloak.example.com/realms/prod/":                                   true,
		"https://issuer.example.com/.well-known/openid-configuration":                 false,
		"https://issuer.example.com/.well-known/openid-configuration/":                false,
		"https://issuer.example.com?tenant=1":                                         false,
		"https://issuer.example.com#fragment":                                         false,
		"not a url":                                                                   false,
		"issuer.example.com":                                                          false,
		"https://login.microsoftonline.com/00000000-0000-0000-0000-000000000000/v2.0": true,
	}

	for value, valid := range cases {
		t.Run(value, func(t *testing.T) {
			resp := &validator.StringResponse{}
			oidcDiscoveryUrlValidator{}.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("oidc_discovery_url"),
				ConfigValue: types.StringValue(value),
			}, resp)
			if resp.Diagnostics.HasError() == valid {
				t.Errorf("expected valid=%v, got diagnostics %v", valid, resp.Diagnostics)
			}
		})
	}
}

func TestUuidValidator(t *testing.T) {
	cases := map[string]bool{
		"3f1c9a8e-2b4d-4c6e-9f10-1a2b3c4d5e6f":  true,
		"00000000-0000-0000-0000-000000000000":  true,
		"":                                      false,
		"3F1C9A8E-2B4D-4C6E-9F10-1A2B3C4D5E6F":  false,
		"3f1c9a8e2b4d4c6e9f101a2b3c4d5e6f":      false,
		" 3f1c9a8e-2b4d-4c6e-9f10-1a2b3c4d5e6f": false,
		"prod-gateway":                          false,
	}

	for value, valid := range cases {
		t.Run(value, func(t *testing.T) {
			resp := &validator.StringResponse{}
			infisicaltf.UuidValidator.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("gateway_id"),
				ConfigValue: types.StringValue(value),
			}, resp)
			if resp.Diagnostics.HasError() == valid {
				t.Errorf("expected valid=%v, got diagnostics %v", valid, resp.Diagnostics)
			}
		})
	}

	// null is how a user leaves the attribute unset, so it must pass.
	resp := &validator.StringResponse{}
	infisicaltf.UuidValidator.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("gateway_id"),
		ConfigValue: types.StringNull(),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected null to pass, got %v", resp.Diagnostics)
	}
}

// A list that cannot convert must surface its error to the caller, which must then not send a
// partial value that the API would propagate to every linked identity.
func TestStringListValuesReportsConversionErrors(t *testing.T) {
	ctx := context.Background()

	withNull, diags := types.ListValue(types.StringType, []attr.Value{types.StringValue("a"), types.StringNull()})
	if diags.HasError() {
		t.Fatalf("expected the list to build, got %v", diags)
	}
	var errs diag.Diagnostics
	stringListValues(ctx, &errs, withNull)
	if !errs.HasError() {
		t.Error("expected a null element to be reported, since it cannot become a string")
	}

	var none diag.Diagnostics
	if values := stringListValues(ctx, &none, types.ListNull(types.StringType)); none.HasError() || len(values) != 0 {
		t.Errorf("expected a null list to be empty without error, got %v %v", values, none)
	}

	audiences, _ := types.ListValue(types.StringType, []attr.Value{types.StringValue("a"), types.StringValue("b")})
	var ok diag.Diagnostics
	if values := stringListValues(ctx, &ok, audiences); ok.HasError() || len(values) != 2 || values[1] != "b" {
		t.Errorf("expected [a b], got %v %v", values, ok)
	}
}

func linkedKubernetesAuthState(t *testing.T) IdentityKubernetesAuthResourceModel {
	namespaces, diags := types.ListValue(types.StringType, []attr.Value{types.StringValue("ns")})
	if diags.HasError() {
		t.Fatalf("expected the list to build, got %v", diags)
	}
	return IdentityKubernetesAuthResourceModel{
		IdentityID:                         types.StringValue("identity"),
		TemplateID:                         types.StringValue("template"),
		KubernetesHost:                     types.StringValue("https://example.com"),
		CaCertificate:                      customtypes.NewTrimmedStringValue(""),
		TokenReviewerJWT:                   types.StringNull(),
		TokenReviewerMode:                  types.StringValue(TOKEN_REVIEWER_MODE_API),
		GatewayID:                          types.StringNull(),
		AllowedAudience:                    types.StringValue("aud"),
		AllowedNamespaces:                  namespaces,
		AllowedServiceAccountNames:         types.ListValueMust(types.StringType, []attr.Value{}),
		AccessTokenTTL:                     types.Int64Value(2592000),
		AccessTokenMaxTTL:                  types.Int64Value(2592000),
		AccessTokenNumUsesLimit:            types.Int64Value(0),
		AccessTokenTrustedIps:              types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{"ip_address": types.StringType}}),
		HasTemplateSourcedTokenReviewerJWT: types.BoolValue(true),
	}
}

// Only an unlink that changes nothing else may go through delete-usage, which a role scoped to
// unlinking can call; anything more is an edit of the auth configuration and needs edit-auth.
func TestKubernetesAuthUnlinksOnly(t *testing.T) {
	state := linkedKubernetesAuthState(t)
	unlinked := func(change func(*IdentityKubernetesAuthResourceModel)) IdentityKubernetesAuthResourceModel {
		plan := state
		plan.TemplateID = types.StringNull()
		change(&plan)
		return plan
	}

	cases := map[string]struct {
		plan IdentityKubernetesAuthResourceModel
		want bool
	}{
		"template values copied into config": {unlinked(func(*IdentityKubernetesAuthResourceModel) {}), true},
		"unset TTLs left to the API":         {unlinked(func(p *IdentityKubernetesAuthResourceModel) { p.AccessTokenTTL = types.Int64Unknown() }), true},
		"still linked":                       {state, false},
		"host changed": {unlinked(func(p *IdentityKubernetesAuthResourceModel) {
			p.KubernetesHost = types.StringValue("https://other.example.com")
		}), false},
		"host cleared":     {unlinked(func(p *IdentityKubernetesAuthResourceModel) { p.KubernetesHost = types.StringNull() }), false},
		"audience cleared": {unlinked(func(p *IdentityKubernetesAuthResourceModel) { p.AllowedAudience = types.StringValue("") }), false},
		"CA added": {unlinked(func(p *IdentityKubernetesAuthResourceModel) {
			p.CaCertificate = customtypes.NewTrimmedStringValue("CA")
		}), false},
		"JWT replaced":                  {unlinked(func(p *IdentityKubernetesAuthResourceModel) { p.TokenReviewerJWT = types.StringValue("eyJ.new") }), false},
		"JWT removed with empty string": {unlinked(func(p *IdentityKubernetesAuthResourceModel) { p.TokenReviewerJWT = types.StringValue("") }), false},
		"TTL changed":                   {unlinked(func(p *IdentityKubernetesAuthResourceModel) { p.AccessTokenTTL = types.Int64Value(60) }), false},
		// An unknown allowlist is sent as empty, which clears the entries state holds.
		"namespaces unknown": {unlinked(func(p *IdentityKubernetesAuthResourceModel) {
			p.AllowedNamespaces = types.ListUnknown(types.StringType)
		}), false},
		// The same with nothing in state clears nothing: an unset allowlist on an identity without one.
		"service account names unknown and empty": {unlinked(func(p *IdentityKubernetesAuthResourceModel) {
			p.AllowedServiceAccountNames = types.ListUnknown(types.StringType)
		}), true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := kubernetesAuthUnlinksOnly(tc.plan, state); got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}

	// The CA is compared the way the API stores it, without the newline file() adds.
	withCa := state
	withCa.CaCertificate = customtypes.NewTrimmedStringValue("CA")
	plan := withCa
	plan.TemplateID = types.StringNull()
	plan.CaCertificate = customtypes.NewTrimmedStringValue("CA\n")
	if !kubernetesAuthUnlinksOnly(plan, withCa) {
		t.Error("expected a trailing newline on the CA not to count as a change")
	}
}

func TestOidcAuthUnlinksOnly(t *testing.T) {
	audiences, _ := types.ListValue(types.StringType, []attr.Value{types.StringValue("a2"), types.StringValue("a3")})
	state := IdentityOidcAuthResourceModel{
		IdentityID:              types.StringValue("identity"),
		TemplateID:              types.StringValue("template"),
		OidcDiscoveryUrl:        types.StringValue("https://issuer.example.com"),
		BoundIssuer:             types.StringValue("https://issuer.example.com"),
		BoundAudiences:          audiences,
		CaCertificate:           customtypes.NewTrimmedStringValue(""),
		BoundSubject:            types.StringValue("repo:org/repo:ref:refs/heads/main"),
		BoundClaims:             types.MapValueMust(types.StringType, map[string]attr.Value{}),
		ClaimMetadataMapping:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
		AccessTokenTTL:          types.Int64Value(2592000),
		AccessTokenMaxTTL:       types.Int64Value(2592000),
		AccessTokenNumUsesLimit: types.Int64Value(0),
		AccessTokenTrustedIps:   types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{"ip_address": types.StringType}}),
	}
	unlinked := func(change func(*IdentityOidcAuthResourceModel)) IdentityOidcAuthResourceModel {
		plan := state
		plan.TemplateID = types.StringNull()
		change(&plan)
		return plan
	}

	cases := map[string]struct {
		plan IdentityOidcAuthResourceModel
		want bool
	}{
		"template values copied into config": {unlinked(func(*IdentityOidcAuthResourceModel) {}), true},
		"still linked":                       {state, false},
		// What Thiago saw: audiences left out are now planned as cleared, which is a change.
		"audiences cleared": {unlinked(func(p *IdentityOidcAuthResourceModel) {
			p.BoundAudiences = types.ListValueMust(types.StringType, []attr.Value{})
		}), false},
		"issuer changed":  {unlinked(func(p *IdentityOidcAuthResourceModel) { p.BoundIssuer = types.StringValue("https://other.example.com") }), false},
		"subject changed": {unlinked(func(p *IdentityOidcAuthResourceModel) { p.BoundSubject = types.StringValue("other") }), false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := oidcAuthUnlinksOnly(tc.plan, state); got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

// The CA is compared the way the API stores it, without the newline file() adds, so a CA restated
// from file() is still an unlink-only plan.
func TestOidcAuthUnlinksOnlyIgnoresCaTrailingNewline(t *testing.T) {
	state := IdentityOidcAuthResourceModel{
		TemplateID:            types.StringValue("template"),
		OidcDiscoveryUrl:      types.StringValue("https://issuer.example.com"),
		BoundIssuer:           types.StringValue("https://issuer.example.com"),
		BoundAudiences:        types.ListValueMust(types.StringType, []attr.Value{}),
		CaCertificate:         customtypes.NewTrimmedStringValue("CA"),
		BoundSubject:          types.StringValue("s"),
		BoundClaims:           types.MapValueMust(types.StringType, map[string]attr.Value{}),
		ClaimMetadataMapping:  types.MapValueMust(types.StringType, map[string]attr.Value{}),
		AccessTokenTrustedIps: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{"ip_address": types.StringType}}),
	}
	plan := state
	plan.TemplateID = types.StringNull()
	plan.CaCertificate = customtypes.NewTrimmedStringValue("CA\n")

	if !oidcAuthUnlinksOnly(plan, state) {
		t.Error("expected a trailing newline on the CA not to count as a change")
	}
}
