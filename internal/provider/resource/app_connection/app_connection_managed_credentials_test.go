package resource

import (
	"context"
	"testing"

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
