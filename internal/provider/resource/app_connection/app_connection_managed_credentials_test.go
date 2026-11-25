package resource

import (
	"context"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The base resource treats every variant the same way, so these tests use the ADCS variant with
// the fake marking its credentials as managed by Infisical, as the dashboard can for
// platform-managed connections (Postgres, for example) and rotation (LDAP and the Azure
// connections). Like the real GET by ID, the fake never returns the rotation itself, only
// isAutoRotationEnabled.

func seedManagedConnection(backend *fakeADCSConnectionBackend, flags map[string]any) {
	connection := map[string]any{
		"id": testADCSConnectionID, "name": "adcs-connection", "description": "", "app": "adcs",
		"method": "username-password", "projectId": nil, "gatewayId": "gateway-1", "gatewayPoolId": nil,
		"credentialsHash": "hash-managed",
		"credentials":     map[string]any{"host": "ca01.corp.example.com", "username": `CORP\svc-infisical`},
	}
	for key, value := range flags {
		if key == "hasRotation" {
			backend.rotations[testADCSConnectionID] = value == true
			continue
		}
		connection[key] = value
	}
	backend.connections[testADCSConnectionID] = connection
}

func importAndRead(t *testing.T, r *AppConnectionBaseResource) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	s := adcsTestSchema(t)
	importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testADCSConnectionID}, &importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatal(importResp.Diagnostics)
	}
	readResp := resource.ReadResponse{State: importResp.State}
	r.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	return readResp.State
}

func renamedPlan(t *testing.T, name string) tfsdk.Plan {
	t.Helper()
	s := adcsTestSchema(t)
	model := adcsGatewayModel(t, types.StringValue(testADCSConnectionID), onGateway("gateway-1"))
	model.Name = types.StringValue(name)
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func TestManagedCredentialsAreNotSentAfterImport(t *testing.T) {
	cases := map[string]map[string]any{
		"platform-managed": {"isPlatformManagedCredentials": true},
		"rotation on":      {"isAutoRotationEnabled": true, "hasRotation": true},
	}

	for name, flags := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			backend := newFakeADCSConnectionBackend(t)
			r := newADCSTestResource(t, backend)
			seedManagedConnection(backend, flags)
			state := importAndRead(t, r)

			updateResp := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Plan: renamedPlan(t, "renamed"), State: state}, &updateResp)
			if updateResp.Diagnostics.HasError() {
				t.Fatalf("update after import must succeed, got %v", updateResp.Diagnostics)
			}
			if _, sent := backend.lastBody["credentials"]; sent {
				t.Errorf("credentials Infisical manages must not be sent, got body %v", backend.lastBody)
			}
			connection := backend.connections[testADCSConnectionID]
			if connection["name"] != "renamed" {
				t.Errorf("the rename must still apply, got %v", connection["name"])
			}
			if connection["isAutoRotationEnabled"] != flags["isAutoRotationEnabled"] {
				t.Errorf("rotation must stay as it was, got isAutoRotationEnabled %v", connection["isAutoRotationEnabled"])
			}
		})
	}
}

// Infisical changes credentials it manages on its own, so a new hash is not a conflict. The
// credentials are cleared from state, so the next apply records the configured ones without
// sending them.
func TestManagedCredentialsHashChangeIsNotAConflict(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: adcsPlan(t, s, types.StringUnknown(), testADCSFirstPassword)}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	// Rotation is turned on in the dashboard, and Infisical rotates the password.
	connection := backend.connections[testADCSConnectionID]
	connection["isAutoRotationEnabled"] = true
	backend.rotations[testADCSConnectionID] = true
	connection["credentialsHash"] = "hash-after-rotation"

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() || readResp.Diagnostics.WarningsCount() != 0 {
		t.Fatalf("a rotation is not a conflict, got %v", readResp.Diagnostics)
	}
	model := adcsModel(t, readResp.State)
	if !credentialsUnset(model.Credentials) {
		t.Error("the credentials must be cleared, so the next apply records the configured ones")
	}
	if model.CredentialsHash.ValueString() != "hash-after-rotation" {
		t.Errorf("expected the new hash in state, got %s", model.CredentialsHash)
	}
}

func TestManagedCredentialsPlanWarning(t *testing.T) {
	for _, managed := range []bool{true, false} {
		ctx := context.Background()
		s := adcsTestSchema(t)
		backend := newFakeADCSConnectionBackend(t)
		r := newADCSTestResource(t, backend)
		seedManagedConnection(backend, map[string]any{"isPlatformManagedCredentials": managed})
		state := importAndRead(t, r)

		plan := adcsPlan(t, s, types.StringValue(testADCSConnectionID), testADCSFirstPassword)
		resp := resource.ModifyPlanResponse{Plan: plan}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state, Config: tfsdk.Config{Schema: s, Raw: plan.Raw}}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if got := resp.Diagnostics.WarningsCount() == 1; got != managed {
			t.Errorf("managed %v: warning = %v, got %v", managed, got, resp.Diagnostics)
		}
	}
}

// A change to credentials already in state would never reach Infisical, so the plan refuses it.
func TestManagedCredentialsChangeIsRefusedAtPlan(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)
	seedManagedConnection(backend, map[string]any{"isPlatformManagedCredentials": true})
	imported := importAndRead(t, r)

	// The first apply after the import records the configured credentials.
	plan := adcsPlan(t, s, types.StringValue(testADCSConnectionID), testADCSFirstPassword)
	updateResp := resource.UpdateResponse{State: imported}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: imported}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}

	changed := adcsModel(t, updateResp.State)
	credentials, diags := types.ObjectValue(adcsCredentialsAttrTypes, map[string]attr.Value{
		"host":     types.StringValue("ca02.corp.example.com"),
		"username": types.StringValue(`CORP\svc-infisical`),
		"password": types.StringValue(testADCSFirstPassword),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	changed.Credentials = credentials
	changedPlan := tfsdk.Plan{Schema: s}
	if diags := changedPlan.Set(ctx, changed); diags.HasError() {
		t.Fatal(diags)
	}

	resp := resource.ModifyPlanResponse{Plan: changedPlan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: changedPlan, State: updateResp.State, Config: tfsdk.Config{Schema: s, Raw: changedPlan.Raw}}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Errorf("a host change on a connection whose credentials Infisical manages must fail at plan, got %v", resp.Diagnostics)
	}
}

// managedStateWithCredentials returns the state of an imported platform-managed connection after
// the first apply recorded its configured credentials.
func managedStateWithCredentials(t *testing.T) (*AppConnectionBaseResource, *fakeADCSConnectionBackend, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)
	seedManagedConnection(backend, map[string]any{"isPlatformManagedCredentials": true})
	imported := importAndRead(t, r)

	updateResp := resource.UpdateResponse{State: imported}
	r.Update(ctx, resource.UpdateRequest{Plan: adcsPlan(t, s, types.StringValue(testADCSConnectionID), testADCSFirstPassword), State: imported}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	return r, backend, updateResp.State
}

func modelWithCredentials(t *testing.T, state tfsdk.State, host string, password attr.Value) appConnectionGatewayResourceModel {
	t.Helper()
	model := adcsModel(t, state)
	credentials, diags := types.ObjectValue(adcsCredentialsAttrTypes, map[string]attr.Value{
		"host":     types.StringValue(host),
		"username": types.StringValue(`CORP\svc-infisical`),
		"password": password,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	model.Credentials = credentials
	return model
}

func modifyPlan(t *testing.T, r *AppConnectionBaseResource, state tfsdk.State, model appConnectionGatewayResourceModel) resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()
	s := adcsTestSchema(t)
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	resp := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state, Config: tfsdk.Config{Schema: s, Raw: plan.Raw}}, &resp)
	return resp
}

// A replacement creates a new connection with the configured credentials, so it is not refused.
func TestManagedCredentialsReplacementIsNotRefused(t *testing.T) {
	r, _, state := managedStateWithCredentials(t)
	model := modelWithCredentials(t, state, "ca02.corp.example.com", types.StringValue(testADCSSecondPassword))
	model.ProjectId = types.StringValue("project-2")
	if resp := modifyPlan(t, r, state, model); resp.Diagnostics.HasError() {
		t.Errorf("a plan that replaces the connection must not be refused, got %v", resp.Diagnostics)
	}
}

// A credential known only at apply can't be compared at plan, so Update refuses the change.
func TestManagedCredentialsUnknownAtPlanIsRefusedAtApply(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	r, backend, state := managedStateWithCredentials(t)

	unknown := modelWithCredentials(t, state, "ca01.corp.example.com", types.StringUnknown())
	if resp := modifyPlan(t, r, state, unknown); resp.Diagnostics.HasError() {
		t.Errorf("a credential unknown at plan must not fail the plan, got %v", resp.Diagnostics)
	}

	// The value resolves at apply to a host Infisical doesn't have.
	resolved := modelWithCredentials(t, state, "ca02.corp.example.com", types.StringValue(testADCSSecondPassword))
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, resolved); diags.HasError() {
		t.Fatal(diags)
	}
	requestsBefore := backend.requests
	updateResp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Error("a host that differs from Infisical's must fail at apply when the plan couldn't refuse it")
	}
	if backend.requests != requestsBefore+1 {
		t.Errorf("only the read before the update may reach the API, got %d requests", backend.requests-requestsBefore)
	}
}

// After an import, the configured credentials are recorded without being sent, so a non-secret
// field that differs from Infisical's must fail the plan, and the apply when the plan couldn't
// tell, instead of being recorded.
func TestManagedCredentialsMismatchAfterImportIsRefused(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)
	seedManagedConnection(backend, map[string]any{"isPlatformManagedCredentials": true})
	imported := importAndRead(t, r)

	mismatched := modelWithCredentials(t, imported, "ca02.corp.example.com", types.StringValue(testADCSFirstPassword))
	if resp := modifyPlan(t, r, imported, mismatched); !resp.Diagnostics.HasError() {
		t.Errorf("a host that differs from Infisical's must fail the plan, got %v", resp.Diagnostics)
	}

	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, mismatched); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: imported}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: imported}, &updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Error("a host that differs from Infisical's must fail the apply too")
	}

	matching := modelWithCredentials(t, imported, "ca01.corp.example.com", types.StringValue(testADCSFirstPassword))
	if resp := modifyPlan(t, r, imported, matching); resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Errorf("matching credentials must only warn, got %v", resp.Diagnostics)
	}
}

func TestSnakeToCamel(t *testing.T) {
	for in, want := range map[string]string{"host": "host", "ssl_reject_unauthorized": "sslRejectUnauthorized", "client_id": "clientId"} {
		if got := snakeToCamel(in); got != want {
			t.Errorf("snakeToCamel(%q) = %q, want %q", in, got, want)
		}
	}
}

// Infisical owns a platform-managed connection's password, so a password change in the config is
// recorded without being sent, not refused.
func TestManagedCredentialsPasswordChangeIsRecorded(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	r, backend, state := managedStateWithCredentials(t)

	changed := modelWithCredentials(t, state, "ca01.corp.example.com", types.StringValue(testADCSSecondPassword))
	if resp := modifyPlan(t, r, state, changed); resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Errorf("a password change must only warn, got %v", resp.Diagnostics)
	}
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, changed); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if _, sent := backend.lastBody["credentials"]; sent {
		t.Errorf("credentials must not be sent to a platform-managed connection, got body %v", backend.lastBody)
	}
}

// With rotation on, the API accepts a credentials change, so once state holds the credentials a
// change the user makes is sent, and an update that leaves them unchanged doesn't resend them.
func TestRotatingConnectionSendsOnlyChangedCredentials(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)
	seedManagedConnection(backend, map[string]any{"isAutoRotationEnabled": true, "hasRotation": true})
	imported := importAndRead(t, r)

	// After the import the credentials are recorded, not sent.
	first := resource.UpdateResponse{State: imported}
	r.Update(ctx, resource.UpdateRequest{Plan: adcsPlan(t, s, types.StringValue(testADCSConnectionID), testADCSFirstPassword), State: imported}, &first)
	if first.Diagnostics.HasError() {
		t.Fatal(first.Diagnostics)
	}
	if _, sent := backend.lastBody["credentials"]; sent {
		t.Errorf("credentials must not be sent after an import, got body %v", backend.lastBody)
	}

	// A rename leaves them out too.
	rename := resource.UpdateResponse{State: first.State}
	r.Update(ctx, resource.UpdateRequest{Plan: renamedPlan(t, "renamed"), State: first.State}, &rename)
	if rename.Diagnostics.HasError() {
		t.Fatal(rename.Diagnostics)
	}
	if _, sent := backend.lastBody["credentials"]; sent {
		t.Errorf("unchanged credentials must not be resent, got body %v", backend.lastBody)
	}

	// A change the user makes is sent, and the API accepts it.
	changed := modelWithCredentials(t, rename.State, "ca02.corp.example.com", types.StringValue(testADCSSecondPassword))
	if resp := modifyPlan(t, r, rename.State, changed); resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 0 {
		t.Errorf("a credentials change on a rotating connection must plan cleanly, got %v", resp.Diagnostics)
	}
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, changed); diags.HasError() {
		t.Fatal(diags)
	}
	changeResp := resource.UpdateResponse{State: rename.State}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: rename.State}, &changeResp)
	if changeResp.Diagnostics.HasError() {
		t.Fatal(changeResp.Diagnostics)
	}
	credentials, _ := backend.lastBody["credentials"].(map[string]any)
	if credentials["host"] != "ca02.corp.example.com" {
		t.Errorf("a changed host must be sent, got body %v", backend.lastBody)
	}
}

func TestCredentialValueMatchesInt32(t *testing.T) {
	if credentialValueMatches(types.Int32Value(5432), float64(15432)) {
		t.Error("an Int32 port that differs from Infisical's must not match")
	}
	if !credentialValueMatches(types.Int32Value(15432), float64(15432)) {
		t.Error("an Int32 port equal to Infisical's must match")
	}
}

// With rotation on, a change to a non-secret field alone would send the configured password,
// which Infisical may have rotated, so it is refused at plan and at apply.
func TestRotatingConnectionRefusesChangeWithoutPassword(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)
	seedManagedConnection(backend, map[string]any{"isAutoRotationEnabled": true, "hasRotation": true})
	imported := importAndRead(t, r)
	first := resource.UpdateResponse{State: imported}
	r.Update(ctx, resource.UpdateRequest{Plan: adcsPlan(t, s, types.StringValue(testADCSConnectionID), testADCSFirstPassword), State: imported}, &first)
	if first.Diagnostics.HasError() {
		t.Fatal(first.Diagnostics)
	}

	hostOnly := modelWithCredentials(t, first.State, "ca02.corp.example.com", types.StringValue(testADCSFirstPassword))
	if resp := modifyPlan(t, r, first.State, hostOnly); !resp.Diagnostics.HasError() {
		t.Errorf("a host change without a new password must fail the plan, got %v", resp.Diagnostics)
	}

	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, hostOnly); diags.HasError() {
		t.Fatal(diags)
	}
	requestsBefore := backend.requests
	updateResp := resource.UpdateResponse{State: first.State}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: first.State}, &updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Error("a host change without a new password must fail at apply too")
	}
	if backend.requests != requestsBefore+1 {
		t.Errorf("only the read before the update may reach the API, got %d requests", backend.requests-requestsBefore)
	}
}

// Removing a setting Infisical still has is a mismatch, or the removal would be recorded but
// never reach Infisical.
func TestManagedCredentialsRemovedFieldIsAMismatch(t *testing.T) {
	r, ok := NewAppConnectionADCSResource().(*AppConnectionBaseResource)
	if !ok {
		t.Fatal("expected the base resource")
	}
	differing := r.credentialsDifferingFromInfisical(
		types.ObjectValueMust(adcsCredentialsAttrTypes, map[string]attr.Value{
			"host": types.StringNull(), "username": types.StringNull(), "password": types.StringValue("secret"),
		}),
		map[string]any{"host": "ca01.corp.example.com", "username": ""},
	)
	if len(differing) != 1 || differing[0] != "credentials.host" {
		t.Errorf("a removed host Infisical still has must differ, and an empty username must not, got %v", differing)
	}
}

// credentialsObject builds a credentials value for a variant from its own schema, with the given
// attributes set and the rest null.
func credentialsObject(t *testing.T, r *AppConnectionBaseResource, values map[string]attr.Value) types.Object {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	objectType, ok := schemaResp.Schema.Attributes["credentials"].GetType().(types.ObjectType)
	if !ok {
		t.Fatal("credentials must be an object")
	}
	attributes := map[string]attr.Value{}
	for name, attrType := range objectType.AttrTypes {
		if value, set := values[name]; set {
			attributes[name] = value
			continue
		}
		nullValue, err := attrType.ValueFromTerraform(context.Background(), tftypes.NewValue(attrType.TerraformType(context.Background()), nil))
		if err != nil {
			t.Fatal(err)
		}
		attributes[name] = nullValue
	}
	object, diags := types.ObjectValue(objectType.AttrTypes, attributes)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return object
}

// Infisical leaves out a non-secret field that has no value: an absent string counts as empty,
// so setting it is a mismatch; an absent bool keeps Infisical's default and is skipped; an absent
// secret is never compared.
func TestManagedCredentialsAbsentFields(t *testing.T) {
	r, ok := NewAppConnectionPostgresResource().(*AppConnectionBaseResource)
	if !ok {
		t.Fatal("expected the base resource")
	}
	planned := credentialsObject(t, r, map[string]attr.Value{
		"host":                    types.StringValue("db.internal"),
		"ssl_certificate":         types.StringValue("pr415-ca"),
		"ssl_reject_unauthorized": types.BoolValue(true),
		"password":                types.StringValue("secret"),
	})
	differing := r.credentialsDifferingFromInfisical(planned, map[string]any{"host": "db.internal"})
	if len(differing) != 1 || differing[0] != "credentials.ssl_certificate" {
		t.Errorf("only the certificate Infisical doesn't have must differ, got %v", differing)
	}
}

// Azure DevOps names the field organization_name, and the API orgName.
func TestManagedCredentialsMappedAPIKey(t *testing.T) {
	r, ok := NewAppConnectionAzureDevOpsResource().(*AppConnectionBaseResource)
	if !ok {
		t.Fatal("expected the base resource")
	}
	planned := credentialsObject(t, r, map[string]attr.Value{"organization_name": types.StringValue("org-a")})
	if differing := r.credentialsDifferingFromInfisical(planned, map[string]any{"orgName": "org-b"}); len(differing) != 1 {
		t.Errorf("an organization_name that differs from orgName must differ, got %v", differing)
	}
	if differing := r.credentialsDifferingFromInfisical(planned, map[string]any{"orgName": "org-a"}); len(differing) != 0 {
		t.Errorf("a matching orgName must not differ, got %v", differing)
	}
}

func TestCredentialValueMatchesFailsClosed(t *testing.T) {
	list := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")})
	if credentialValueMatches(list, []any{"a"}) {
		t.Error("a type the comparison doesn't handle must never match")
	}
}

// An empty configured string matches a field Infisical leaves out or returns as null.
func TestManagedCredentialsEmptyStringMatchesAbsent(t *testing.T) {
	r, ok := NewAppConnectionPostgresResource().(*AppConnectionBaseResource)
	if !ok {
		t.Fatal("expected the base resource")
	}
	planned := credentialsObject(t, r, map[string]attr.Value{
		"host":            types.StringValue("db.internal"),
		"ssl_certificate": types.StringValue(""),
	})
	for _, api := range []map[string]any{{"host": "db.internal"}, {"host": "db.internal", "sslCertificate": nil}} {
		if differing := r.credentialsDifferingFromInfisical(planned, api); len(differing) != 0 {
			t.Errorf("an empty ssl_certificate must match %v, got %v", api, differing)
		}
	}
}

// Azure marks client_id sensitive, but GET returns it and Infisical rotates only the client
// secret: changing client_id alone would send the rotated-away secret, so it needs the current
// one, while a change to client_secret is the user's own and is sent.
func TestRotatingAzureClientIDIsNotASecret(t *testing.T) {
	r, ok := NewAppConnectionAzureResource().(*AppConnectionBaseResource)
	if !ok {
		t.Fatal("expected the base resource")
	}
	connection := infisical.AppConnection{IsAutoRotationEnabled: true, Credentials: map[string]any{"clientId": "client-a", "tenantId": "tenant"}}
	current := credentialsObject(t, r, map[string]attr.Value{
		"client_id": types.StringValue("client-a"), "tenant_id": types.StringValue("tenant"), "client_secret": types.StringValue("secret-a"),
	})
	clientIDOnly := credentialsObject(t, r, map[string]attr.Value{
		"client_id": types.StringValue("client-b"), "tenant_id": types.StringValue("tenant"), "client_secret": types.StringValue("secret-a"),
	})
	if !r.needsCurrentPassword(connection, clientIDOnly, current) {
		t.Error("changing client_id alone must need the current client secret")
	}
	withSecret := credentialsObject(t, r, map[string]attr.Value{
		"client_id": types.StringValue("client-b"), "tenant_id": types.StringValue("tenant"), "client_secret": types.StringValue("secret-b"),
	})
	if r.needsCurrentPassword(connection, withSecret, current) {
		t.Error("a change that sets the client secret is the user's own and must be sent")
	}
}
