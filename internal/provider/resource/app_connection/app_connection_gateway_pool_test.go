package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const testGatewayPoolID = "76910349-48d9-428f-9eef-7bacfc2ebc68"

func adcsGatewayModel(t *testing.T, id types.String, gateway appConnectionGateway) appConnectionGatewayResourceModel {
	t.Helper()
	return appConnectionGatewayResourceModel{
		AppConnectionBaseResourceModel: AppConnectionBaseResourceModel{
			ID:              id,
			Name:            types.StringValue("adcs-connection"),
			Method:          types.StringValue(ADCSAppConnectionUsernamePasswordMethod),
			Description:     types.StringNull(),
			ProjectId:       types.StringNull(),
			Credentials:     adcsCredentials(t, testADCSFirstPassword),
			CredentialsHash: types.StringUnknown(),
		},
		GatewayId:     gateway.Id,
		GatewayPoolId: gateway.PoolId,
	}
}

func adcsGatewayPlan(t *testing.T, s schema.Schema, id types.String, gateway appConnectionGateway) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), adcsGatewayModel(t, id, gateway)); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func onPool() appConnectionGateway {
	return appConnectionGateway{Id: types.StringNull(), PoolId: types.StringValue(testGatewayPoolID)}
}

func onGateway(id string) appConnectionGateway {
	return appConnectionGateway{Id: types.StringValue(id), PoolId: types.StringNull()}
}

// A pool chosen in the dashboard must show up in state, and an apply must send both fields so
// the API ends up with exactly what the configuration says.
func TestAppConnectionGatewayPoolLifecycle(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: adcsGatewayPlan(t, s, types.StringUnknown(), onPool())}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	gatewayId, gatewayIdSent := backend.lastBody["gatewayId"]
	if backend.lastBody["gatewayPoolId"] != testGatewayPoolID || !gatewayIdSent || gatewayId != nil {
		t.Errorf("create must send the pool and an explicit null gateway, got body %v", backend.lastBody)
	}
	if got := adcsModel(t, createResp.State); got.GatewayPoolId.ValueString() != testGatewayPoolID || !got.GatewayId.IsNull() {
		t.Errorf("unexpected state after create: gateway %s, pool %s", got.GatewayId, got.GatewayPoolId)
	}

	// The dashboard moves the connection from the pool to a single gateway.
	backend.connections[testADCSConnectionID]["gatewayId"] = "gateway-1"
	backend.connections[testADCSConnectionID]["gatewayPoolId"] = nil
	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	refreshed := adcsModel(t, readResp.State)
	if refreshed.GatewayId.ValueString() != "gateway-1" || !refreshed.GatewayPoolId.IsNull() {
		t.Fatalf("read must show the dashboard's change as drift: gateway %s, pool %s", refreshed.GatewayId, refreshed.GatewayPoolId)
	}

	// The next apply puts it back on the pool.
	updateResp := resource.UpdateResponse{State: readResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: adcsGatewayPlan(t, s, types.StringValue(testADCSConnectionID), onPool()), State: readResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	connection := backend.connections[testADCSConnectionID]
	if connection["gatewayPoolId"] != testGatewayPoolID || connection["gatewayId"] != nil {
		t.Errorf("update must leave the connection on the pool only, got gateway %v, pool %v", connection["gatewayId"], connection["gatewayPoolId"])
	}
}

// Moving from a pool to a gateway must clear the pool, and the reverse.
func TestAppConnectionGatewayPoolSwitch(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: adcsGatewayPlan(t, s, types.StringUnknown(), onPool())}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	updateResp := resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: adcsGatewayPlan(t, s, types.StringValue(testADCSConnectionID), onGateway("gateway-1")), State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	connection := backend.connections[testADCSConnectionID]
	if connection["gatewayId"] != "gateway-1" || connection["gatewayPoolId"] != nil {
		t.Errorf("moving to a gateway must clear the pool, got gateway %v, pool %v", connection["gatewayId"], connection["gatewayPoolId"])
	}
	if got := adcsModel(t, updateResp.State); got.GatewayId.ValueString() != "gateway-1" || !got.GatewayPoolId.IsNull() {
		t.Errorf("unexpected state after update: gateway %s, pool %s", got.GatewayId, got.GatewayPoolId)
	}

	// Back on the pool, then removing both, as a connection that allows the Internet Gateway can.
	// The API keeps a field it doesn't receive, so the pool is only cleared if it is sent as null.
	poolResp := resource.UpdateResponse{State: updateResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: adcsGatewayPlan(t, s, types.StringValue(testADCSConnectionID), onPool()), State: updateResp.State}, &poolResp)
	if poolResp.Diagnostics.HasError() {
		t.Fatal(poolResp.Diagnostics)
	}
	neither := appConnectionGateway{Id: types.StringNull(), PoolId: types.StringNull()}
	clearResp := resource.UpdateResponse{State: poolResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: adcsGatewayPlan(t, s, types.StringValue(testADCSConnectionID), neither), State: poolResp.State}, &clearResp)
	if clearResp.Diagnostics.HasError() {
		t.Fatal(clearResp.Diagnostics)
	}
	connection = backend.connections[testADCSConnectionID]
	if connection["gatewayId"] != nil || connection["gatewayPoolId"] != nil {
		t.Errorf("removing both must clear both, got gateway %v, pool %v", connection["gatewayId"], connection["gatewayPoolId"])
	}
}

func TestAppConnectionGatewayConfigValidators(t *testing.T) {
	ctx := context.Background()
	both := appConnectionGateway{Id: types.StringValue("gateway-1"), PoolId: types.StringValue(testGatewayPoolID)}
	neither := appConnectionGateway{Id: types.StringNull(), PoolId: types.StringNull()}

	cases := []struct {
		name      string
		resource  func() resource.Resource
		gateway   appConnectionGateway
		wantError bool
	}{
		{"required: gateway only", NewAppConnectionADCSResource, onGateway("gateway-1"), false},
		{"required: pool only", NewAppConnectionADCSResource, onPool(), false},
		{"required: both", NewAppConnectionADCSResource, both, true},
		{"required: neither", NewAppConnectionADCSResource, neither, true},
		{"optional: both", NewAppConnectionSshResource, both, true},
		{"optional: neither", NewAppConnectionSshResource, neither, false},
		{"optional: pool only", NewAppConnectionSshResource, onPool(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, ok := tc.resource().(*AppConnectionBaseResource)
			if !ok {
				t.Fatal("expected the base resource")
			}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			s := schemaResp.Schema

			// Only the gateway attributes matter to these validators, so the rest of the
			// configuration can be null.
			credentialsType, ok := s.Attributes["credentials"].GetType().(types.ObjectType)
			if !ok {
				t.Fatal("credentials must be an object")
			}
			model := appConnectionGatewayResourceModel{
				AppConnectionBaseResourceModel: AppConnectionBaseResourceModel{
					Credentials: types.ObjectNull(credentialsType.AttrTypes),
				},
				GatewayId:     tc.gateway.Id,
				GatewayPoolId: tc.gateway.PoolId,
			}
			plan := tfsdk.Plan{Schema: s}
			if diags := plan.Set(ctx, model); diags.HasError() {
				t.Fatal(diags)
			}
			config := tfsdk.Config{Schema: s, Raw: plan.Raw}

			resp := resource.ValidateConfigResponse{}
			for _, v := range r.ConfigValidators(ctx) {
				v.ValidateResource(ctx, resource.ValidateConfigRequest{Config: config}, &resp)
			}
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("error = %v, want %v: %v", resp.Diagnostics.HasError(), tc.wantError, resp.Diagnostics)
			}
		})
	}
}
