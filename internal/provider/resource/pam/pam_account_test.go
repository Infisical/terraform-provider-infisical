package resource

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	testPamFolderID   = "55555555-5555-5555-5555-555555555555"
	testPamTemplateID = "66666666-6666-6666-6666-666666666666"
)

func pamAccountObject(t *testing.T, attributes map[string]schema.Attribute, model any) types.Object {
	t.Helper()
	object, diags := types.ObjectValueFrom(context.Background(), (&PamAccountBaseResource{}).attrTypes(attributes), model)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return object
}

func pamAccountBaseModel(connectionDetails, credentials types.Object) PamAccountBaseResourceModel {
	return PamAccountBaseResourceModel{
		ID:                types.StringUnknown(),
		Name:              types.StringValue("web01-root"),
		Description:       types.StringNull(),
		FolderID:          types.StringValue(testPamFolderID),
		TemplateID:        types.StringValue(testPamTemplateID),
		GatewayID:         types.StringNull(),
		GatewayPoolID:     types.StringNull(),
		ConnectionDetails: connectionDetails,
		Credentials:       credentials,
	}
}

func pamAccountPlan(t *testing.T, r *PamAccountBaseResource, model PamAccountBaseResourceModel, rotation *pamAccountRotationModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: pamResourceSchema(t, r)}
	var diags diag.Diagnostics
	if rotation != nil {
		diags = plan.Set(context.Background(), &pamRotatableAccountResourceModel{PamAccountBaseResourceModel: model, pamAccountRotationModel: *rotation})
	} else {
		diags = plan.Set(context.Background(), &model)
	}
	if diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func asPamAccountResource(t *testing.T, r resource.Resource) *PamAccountBaseResource {
	t.Helper()
	account, ok := r.(*PamAccountBaseResource)
	if !ok {
		t.Fatalf("expected a *PamAccountBaseResource, got %T", r)
	}
	return account
}

func newPamSshTestResource(t *testing.T, backend *fakePamBackend) *PamAccountBaseResource {
	t.Helper()
	r := asPamAccountResource(t, NewPamAccountSshResource())
	configurePamResource(t, r, newPamTestClient(t, backend))
	return r
}

func pamSshCredentials(t *testing.T, r *PamAccountBaseResource, password string) types.Object {
	t.Helper()
	return pamAccountObject(t, r.CredentialsAttributes, PamAccountSshCredentialsModel{
		AuthMethod: types.StringValue(pamSshAuthMethodPassword),
		Username:   types.StringValue("root"),
		Password:   types.StringValue(password),
		PrivateKey: types.StringNull(),
	})
}

func pamSshConnection(t *testing.T, r *PamAccountBaseResource, host string) types.Object {
	t.Helper()
	return pamAccountObject(t, r.ConnectionDetailsAttributes, PamAccountSshConnectionDetailsModel{
		Host: types.StringValue(host),
		Port: types.Int64Value(22),
	})
}

func pamAccountModel(t *testing.T, r *PamAccountBaseResource, state tfsdk.State) (PamAccountBaseResourceModel, pamAccountRotationModel) {
	t.Helper()
	model, rotation, diags := r.getModel(context.Background(), state)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return model, rotation
}

func TestPamAccountLifecycle(t *testing.T) {
	backend := newFakePamBackend(t)
	r := newPamSshTestResource(t, backend)

	state := createPamResource(t, r, pamAccountPlan(t, r, pamAccountBaseModel(pamSshConnection(t, r, "10.0.1.10"), pamSshCredentials(t, r, "hunter2")), nil))
	created, _ := pamAccountModel(t, r, state)
	account := backend.accounts[created.ID.ValueString()]
	if account == nil || account.Credentials["password"] != "hunter2" || account.ConnectionDetails["port"] != float64(22) {
		t.Fatalf("unexpected account on the backend: %+v", account)
	}

	// The API never returns the password, so a refresh keeps the configured one.
	refreshed := readPamResource(t, r, state)
	if !refreshed.Raw.Equal(state.Raw) {
		t.Errorf("expected no drift after a refresh.\ngot:  %v\nwant: %v", refreshed.Raw, state.Raw)
	}

	// Only the changed description is sent.
	described := created
	described.Description = types.StringValue("Web server")
	state = updatePamResource(t, r, pamAccountPlan(t, r, described, nil), refreshed)
	accountPath := "/api/v1/pam/accounts/ssh/" + created.ID.ValueString()
	patch := backend.requestsTo("PATCH", accountPath)[0]
	if len(patch) != 1 || patch["description"] != "Web server" {
		t.Errorf("expected only the description to be sent, got %v", patch)
	}

	backend.accounts[created.ID.ValueString()].Credentials["password"] = "rotated-by-infisical"
	state = readPamResource(t, r, state)
	if refreshed := readPamResource(t, r, state); !refreshed.Raw.Equal(state.Raw) {
		t.Error("expected a password changed outside of Terraform not to show as drift")
	}

	newPassword := described
	newPassword.Credentials = pamSshCredentials(t, r, "correct-horse")
	state = updatePamResource(t, r, pamAccountPlan(t, r, newPassword, nil), state)
	patch = backend.requestsTo("PATCH", accountPath)[1]
	if _, ok := patch["connectionDetails"]; ok {
		t.Error("expected unchanged connection details to be left out")
	}
	if credentials, ok := patch["credentials"].(map[string]any); !ok || credentials["password"] != "correct-horse" {
		t.Errorf("expected the new credentials to be sent, got %v", patch["credentials"])
	}

	// A username changed outside of Terraform is drift.
	backend.accounts[created.ID.ValueString()].Credentials["username"] = "admin"
	drifted, _ := pamAccountModel(t, r, readPamResource(t, r, state))
	if got := stringAttribute(t, drifted.Credentials.Attributes(), "username"); got != "admin" {
		t.Errorf("expected the changed username to be read back, got %q", got)
	}

	deletePamResource(t, r, state)
	if len(backend.accounts) != 0 {
		t.Error("expected the account to be deleted")
	}
	if !readPamResource(t, r, state).Raw.IsNull() {
		t.Error("expected an account deleted outside of Terraform to be dropped from state")
	}
}

func TestPamAccountImportHasNoSecrets(t *testing.T) {
	backend := newFakePamBackend(t)
	backend.accounts["a1"] = &fakePamAccount{
		ID: "a1", Type: "ssh", Name: "web01-root", FolderID: testPamFolderID, TemplateID: testPamTemplateID,
		ConnectionDetails: map[string]any{"host": "10.0.1.10", "port": float64(2222)},
		Credentials:       map[string]any{"authMethod": "password", "username": "root", "password": "hunter2"},
	}
	r := newPamSshTestResource(t, backend)

	imported, _ := pamAccountModel(t, r, importPamResource(t, r, pamResourceSchema(t, r), "a1"))
	credentials := imported.Credentials.Attributes()
	if !credentials["password"].IsNull() || stringAttribute(t, credentials, "username") != "root" {
		t.Errorf("expected the username without the password, got %v", credentials)
	}
	if port := int64Attribute(t, imported.ConnectionDetails.Attributes(), "port"); port != 2222 {
		t.Errorf("expected port 2222, got %d", port)
	}
}

func TestPamAccountReadRejectsAnotherType(t *testing.T) {
	backend := newFakePamBackend(t)
	backend.accounts["a1"] = &fakePamAccount{
		ID: "a1", Type: "ssh", Name: "web01-root", FolderID: testPamFolderID, TemplateID: testPamTemplateID,
		ConnectionDetails: map[string]any{"host": "10.0.1.10", "port": float64(22)},
		Credentials:       map[string]any{"authMethod": "password", "username": "root"},
	}
	r := asPamAccountResource(t, NewPamAccountPostgresResource())
	configurePamResource(t, r, newPamTestClient(t, backend))

	s := pamResourceSchema(t, r)
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
	if diags := state.SetAttribute(context.Background(), path.Root("id"), "a1"); diags.HasError() {
		t.Fatal(diags)
	}
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `has type "ssh"`) {
		t.Errorf("expected an error naming the account's type, got %v", resp.Diagnostics)
	}
}

func newPamPostgresPlan(t *testing.T, r *PamAccountBaseResource, host string, rotation pamAccountRotationModel) tfsdk.Plan {
	t.Helper()
	connection := pamAccountObject(t, r.ConnectionDetailsAttributes, PamAccountPostgresConnectionDetailsModel{
		Host:                  types.StringValue(host),
		Port:                  types.Int64Value(5432),
		Database:              types.StringValue("orders"),
		SslEnabled:            types.BoolValue(true),
		SslRejectUnauthorized: types.BoolValue(true),
		SslCertificate:        types.StringNull(),
	})
	credentials := pamAccountObject(t, r.CredentialsAttributes, PamAccountPostgresCredentialsModel{
		AuthMethod: types.StringValue(pamPostgresAuthMethodPassword),
		Username:   types.StringValue("orders_app"),
		Password:   types.StringValue("hunter2"),
		AwsRegion:  types.StringNull(),
		RoleArn:    types.StringNull(),
	})
	model := pamAccountBaseModel(connection, credentials)
	model.Name = types.StringValue("orders-app")
	return pamAccountPlan(t, r, model, &rotation)
}

func TestPamAccountRotationAccount(t *testing.T) {
	backend := newFakePamBackend(t)
	r := asPamAccountResource(t, NewPamAccountPostgresResource())
	configurePamResource(t, r, newPamTestClient(t, backend))

	selfRotation := pamAccountRotationModel{RotationAccountID: types.StringNull(), SelfRotation: types.BoolValue(true)}
	state := createPamResource(t, r, newPamPostgresPlan(t, r, "db.internal", selfRotation))
	created, _ := pamAccountModel(t, r, state)
	accountID := created.ID.ValueString()
	if got := backend.accounts[accountID].RotationAccountID; got == nil || *got != accountID {
		t.Fatalf("expected the account to rotate itself, got %v", got)
	}

	refreshed := readPamResource(t, r, state)
	if !refreshed.Raw.Equal(state.Raw) {
		t.Errorf("expected no drift after a refresh.\ngot:  %v\nwant: %v", refreshed.Raw, state.Raw)
	}

	// Moving to another host makes the API drop the rotation account, so it's set again.
	movedPlan := newPamPostgresPlan(t, r, "db2.internal", selfRotation)
	if diags := movedPlan.SetAttribute(context.Background(), path.Root("id"), accountID); diags.HasError() {
		t.Fatal(diags)
	}
	state = updatePamResource(t, r, movedPlan, refreshed)
	if got := backend.accounts[accountID].RotationAccountID; got == nil || *got != accountID {
		t.Errorf("expected the rotation account to be set again after the move, got %v", got)
	}

	delegated := pamAccountRotationModel{RotationAccountID: types.StringValue("77777777-7777-7777-7777-777777777777"), SelfRotation: types.BoolValue(false)}
	delegatedPlan := newPamPostgresPlan(t, r, "db2.internal", delegated)
	if diags := delegatedPlan.SetAttribute(context.Background(), path.Root("id"), accountID); diags.HasError() {
		t.Fatal(diags)
	}
	state = updatePamResource(t, r, delegatedPlan, state)
	if got := backend.accounts[accountID].RotationAccountID; got == nil || *got != "77777777-7777-7777-7777-777777777777" {
		t.Errorf("expected the delegated rotation account, got %v", got)
	}
	_, rotation := pamAccountModel(t, r, readPamResource(t, r, state))
	if rotation.SelfRotation.ValueBool() || rotation.RotationAccountID.ValueString() != "77777777-7777-7777-7777-777777777777" {
		t.Errorf("unexpected rotation read back: %+v", rotation)
	}

	backend.accounts[accountID].RotationAccountID = nil
	_, rotation = pamAccountModel(t, r, readPamResource(t, r, state))
	if rotation.SelfRotation.ValueBool() || !rotation.RotationAccountID.IsNull() {
		t.Errorf("expected a cleared rotation account to read back as none, got %+v", rotation)
	}
}

// A failed rotation update must not lose the password the account update already changed.
func TestPamAccountKeepsUpdateWhenRotationFails(t *testing.T) {
	ctx := context.Background()
	backend := newFakePamBackend(t)
	r := asPamAccountResource(t, NewPamAccountPostgresResource())
	configurePamResource(t, r, newPamTestClient(t, backend))

	noRotation := pamAccountRotationModel{RotationAccountID: types.StringNull(), SelfRotation: types.BoolValue(false)}
	state := createPamResource(t, r, newPamPostgresPlan(t, r, "db.internal", noRotation))
	accountID := pamAccountID(t, r, state)

	plan := newPamPostgresPlan(t, r, "db.internal", pamAccountRotationModel{RotationAccountID: types.StringNull(), SelfRotation: types.BoolValue(true)})
	if diags := plan.SetAttribute(ctx, path.Root("id"), accountID); diags.HasError() {
		t.Fatal(diags)
	}
	if diags := plan.SetAttribute(ctx, path.Root("credentials").AtName("password"), "new-password"); diags.HasError() {
		t.Fatal(diags)
	}

	backend.failRotationUpdates = true
	resp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected the rotation update to fail")
	}
	if got := backend.accounts[accountID].Credentials["password"]; got != "new-password" {
		t.Fatalf("expected the backend to hold the new password, got %v", got)
	}

	saved, rotation := pamAccountModel(t, r, resp.State)
	if got := stringAttribute(t, saved.Credentials.Attributes(), "password"); got != "new-password" {
		t.Errorf("expected the new password to be saved in state, got %q", got)
	}
	if rotation.SelfRotation.ValueBool() {
		t.Error("expected the rotation that failed to stay unsaved")
	}
}

func TestPamAccountWindowsAdRoundTrip(t *testing.T) {
	backend := newFakePamBackend(t)
	r := asPamAccountResource(t, NewPamAccountWindowsAdResource())
	configurePamResource(t, r, newPamTestClient(t, backend))

	connection := pamAccountObject(t, r.ConnectionDetailsAttributes, PamAccountWindowsAdConnectionDetailsModel{
		Domain:                  types.StringValue("corp.example.com"),
		DcAddress:               types.StringValue("dc01.corp.example.com"),
		Hosts:                   pamStringList("ws01.corp.example.com", "ws02.corp.example.com"),
		LdapPort:                types.Int64Value(389),
		RdpPort:                 types.Int64Value(3389),
		UseLdaps:                types.BoolValue(false),
		LdapRejectUnauthorized:  types.BoolValue(true),
		LdapCaCert:              types.StringNull(),
		LdapTlsServerName:       types.StringNull(),
		WinrmPort:               types.Int64Value(5985),
		UseWinrmHttps:           types.BoolValue(false),
		WinrmRejectUnauthorized: types.BoolValue(true),
		WinrmCaCert:             types.StringNull(),
	})
	credentials := pamAccountObject(t, r.CredentialsAttributes, PamAccountWindowsAdCredentialsModel{
		Username: types.StringValue(`CORP\pamadmin`),
		Password: types.StringValue("hunter2"),
	})
	model := pamAccountBaseModel(connection, credentials)
	model.Name = types.StringValue("corp-pamadmin")

	state := createPamResource(t, r, pamAccountPlan(t, r, model, &pamAccountRotationModel{RotationAccountID: types.StringNull(), SelfRotation: types.BoolValue(false)}))
	sent := jsonObject(t, backend.requestsTo("POST", "/api/v1/pam/accounts/windows-ad")[0]["connectionDetails"])
	if hosts, _ := sent["hosts"].([]any); sent["port"] != float64(389) || len(hosts) != 2 {
		t.Errorf("expected ldap_port sent as port and both hosts, got %v", sent)
	}
	if len(backend.requestsTo("PATCH", "/api/v1/pam/accounts/"+pamAccountID(t, r, state)+"/rotation")) != 0 {
		t.Error("expected no rotation account to be set when none is configured")
	}
	if refreshed := readPamResource(t, r, state); !refreshed.Raw.Equal(state.Raw) {
		t.Errorf("expected no drift after a refresh.\ngot:  %v\nwant: %v", refreshed.Raw, state.Raw)
	}
}

func pamAccountID(t *testing.T, r *PamAccountBaseResource, state tfsdk.State) string {
	t.Helper()
	model, _ := pamAccountModel(t, r, state)
	return model.ID.ValueString()
}

func TestPamAccountValidateConfig(t *testing.T) {
	ssh := asPamAccountResource(t, NewPamAccountSshResource())
	postgres := asPamAccountResource(t, NewPamAccountPostgresResource())
	kubernetes := asPamAccountResource(t, NewPamAccountKubernetesResource())

	postgresConnection := func(t *testing.T, sslEnabled bool) types.Object {
		return pamAccountObject(t, postgres.ConnectionDetailsAttributes, PamAccountPostgresConnectionDetailsModel{
			Host: types.StringValue("db"), Port: types.Int64Value(5432), Database: types.StringValue("app"),
			SslEnabled: types.BoolValue(sslEnabled), SslRejectUnauthorized: types.BoolValue(true), SslCertificate: types.StringNull(),
		})
	}
	noRotation := &pamAccountRotationModel{RotationAccountID: types.StringNull(), SelfRotation: types.BoolNull()}

	tests := map[string]struct {
		resource  *PamAccountBaseResource
		model     func(t *testing.T) PamAccountBaseResourceModel
		rotation  *pamAccountRotationModel
		wantError string
	}{
		"ssh private key with the password method": {
			resource: ssh,
			model: func(t *testing.T) PamAccountBaseResourceModel {
				return pamAccountBaseModel(pamSshConnection(t, ssh, "host"), pamAccountObject(t, ssh.CredentialsAttributes, PamAccountSshCredentialsModel{
					AuthMethod: types.StringValue(pamSshAuthMethodPassword), Username: types.StringValue("root"),
					Password: types.StringNull(), PrivateKey: types.StringValue("key"),
				}))
			},
			wantError: "only used by the `public-key` auth method",
		},
		"postgres aws-iam without a role": {
			resource: postgres,
			model: func(t *testing.T) PamAccountBaseResourceModel {
				return pamAccountBaseModel(postgresConnection(t, true), pamAccountObject(t, postgres.CredentialsAttributes, PamAccountPostgresCredentialsModel{
					AuthMethod: types.StringValue(pamPostgresAuthMethodAwsIam), Username: types.StringValue("app"), Password: types.StringNull(),
					AwsRegion: types.StringValue("us-east-1"), RoleArn: types.StringNull(),
				}))
			},
			rotation:  noRotation,
			wantError: "`role_arn` is required",
		},
		"postgres aws-iam without tls": {
			resource: postgres,
			model: func(t *testing.T) PamAccountBaseResourceModel {
				return pamAccountBaseModel(postgresConnection(t, false), pamAccountObject(t, postgres.CredentialsAttributes, PamAccountPostgresCredentialsModel{
					AuthMethod: types.StringValue(pamPostgresAuthMethodAwsIam), Username: types.StringValue("app"), Password: types.StringNull(),
					AwsRegion: types.StringValue("us-east-1"), RoleArn: types.StringValue("arn:aws:iam::1:role/pam"),
				}))
			},
			rotation:  noRotation,
			wantError: "`ssl_enabled` must be true",
		},
		"postgres rotating itself and through another account": {
			resource: postgres,
			model: func(t *testing.T) PamAccountBaseResourceModel {
				return pamAccountBaseModel(postgresConnection(t, true), pamAccountObject(t, postgres.CredentialsAttributes, PamAccountPostgresCredentialsModel{
					AuthMethod: types.StringValue(pamPostgresAuthMethodPassword), Username: types.StringValue("app"), Password: types.StringValue("pw"),
					AwsRegion: types.StringNull(), RoleArn: types.StringNull(),
				}))
			},
			rotation:  &pamAccountRotationModel{RotationAccountID: types.StringValue("77777777-7777-7777-7777-777777777777"), SelfRotation: types.BoolValue(true)},
			wantError: "either itself or through `rotation_account_id`",
		},
		"kubernetes token method without a token": {
			resource: kubernetes,
			model: func(t *testing.T) PamAccountBaseResourceModel {
				connection := pamAccountObject(t, kubernetes.ConnectionDetailsAttributes, PamAccountKubernetesConnectionDetailsModel{
					URL: types.StringValue("https://k8s:6443"), SslRejectUnauthorized: types.BoolValue(true), SslCertificate: types.StringNull(),
				})
				return pamAccountBaseModel(connection, pamAccountObject(t, kubernetes.CredentialsAttributes, PamAccountKubernetesCredentialsModel{
					AuthMethod: types.StringValue(pamKubernetesAuthMethodServiceAccountToken), ServiceAccountToken: types.StringNull(),
					Namespace: types.StringNull(), ServiceAccountName: types.StringNull(),
				}))
			},
			wantError: "`service_account_token` is required",
		},
		"kubernetes gateway auth with a token": {
			resource: kubernetes,
			model: func(t *testing.T) PamAccountBaseResourceModel {
				connection := pamAccountObject(t, kubernetes.ConnectionDetailsAttributes, PamAccountKubernetesConnectionDetailsModel{
					URL: types.StringValue("https://k8s:6443"), SslRejectUnauthorized: types.BoolValue(true), SslCertificate: types.StringNull(),
				})
				return pamAccountBaseModel(connection, pamAccountObject(t, kubernetes.CredentialsAttributes, PamAccountKubernetesCredentialsModel{
					AuthMethod: types.StringValue(pamKubernetesAuthMethodGateway), ServiceAccountToken: types.StringValue("token"),
					Namespace: types.StringValue("infisical"), ServiceAccountName: types.StringValue("pam"),
				}))
			},
			wantError: "only used by the `service-account-token` auth method",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			plan := pamAccountPlan(t, test.resource, test.model(t), test.rotation)
			var resp resource.ValidateConfigResponse
			test.resource.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config(plan)}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected an error containing %q", test.wantError)
			}
			if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, test.wantError) {
				t.Errorf("expected an error containing %q, got %q", test.wantError, detail)
			}
		})
	}
}
