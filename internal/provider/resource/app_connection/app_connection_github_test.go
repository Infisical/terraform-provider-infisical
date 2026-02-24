package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var githubCredentialsAttrTypes = map[string]attr.Type{
	"personal_access_token": types.StringType,
	"instance_type":         types.StringType,
	"host":                  types.StringType,
}

func mustGithubCredentialsObject(t *testing.T, token, instanceType, host string) types.Object {
	t.Helper()
	attrs := map[string]attr.Value{
		"personal_access_token": types.StringValue(token),
		"instance_type":         types.StringValue(instanceType),
		"host":                  types.StringValue(host),
	}
	obj, diags := types.ObjectValue(githubCredentialsAttrTypes, attrs)
	if diags.HasError() {
		t.Fatalf("building credentials object: %v", diags)
	}
	return obj
}

func mustGithubCredentialsObjectWithNulls(t *testing.T, token string, instanceType, host attr.Value) types.Object {
	t.Helper()
	attrs := map[string]attr.Value{
		"personal_access_token": types.StringValue(token),
		"instance_type":         instanceType,
		"host":                  host,
	}
	obj, diags := types.ObjectValue(githubCredentialsAttrTypes, attrs)
	if diags.HasError() {
		t.Fatalf("building credentials object: %v", diags)
	}
	return obj
}

func TestBuildGithubCredentialsForCreate_ValidCloud(t *testing.T) {
	ctx := context.Background()
	plan := AppConnectionBaseResourceModel{
		Method: types.StringValue(AppConnectionGithubAuthMethodPat),
		Credentials: mustGithubCredentialsObject(t, "ghp_secret", "", ""),
	}

	config, diags := buildGithubCredentialsForCreate(ctx, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if config["personalAccessToken"] != "ghp_secret" {
		t.Errorf("personalAccessToken: got %v", config["personalAccessToken"])
	}
	if config["instanceType"] != "cloud" {
		t.Errorf("instanceType: got %v, want cloud", config["instanceType"])
	}
	if _, hasHost := config["host"]; hasHost {
		t.Errorf("host should not be set for cloud")
	}
}

func TestBuildGithubCredentialsForCreate_ValidServer(t *testing.T) {
	ctx := context.Background()
	plan := AppConnectionBaseResourceModel{
		Method:      types.StringValue(AppConnectionGithubAuthMethodPat),
		Credentials: mustGithubCredentialsObject(t, "ghp_secret", "server", "github.mycompany.com"),
	}

	config, diags := buildGithubCredentialsForCreate(ctx, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if config["personalAccessToken"] != "ghp_secret" {
		t.Errorf("personalAccessToken: got %v", config["personalAccessToken"])
	}
	if config["instanceType"] != "server" {
		t.Errorf("instanceType: got %v, want server", config["instanceType"])
	}
	if config["host"] != "github.mycompany.com" {
		t.Errorf("host: got %v", config["host"])
	}
}

func TestBuildGithubCredentialsForCreate_InvalidMethod(t *testing.T) {
	ctx := context.Background()
	plan := AppConnectionBaseResourceModel{
		Method:      types.StringValue("oauth"),
		Credentials: mustGithubCredentialsObject(t, "token", "", ""),
	}

	_, diags := buildGithubCredentialsForCreate(ctx, plan)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for invalid method")
	}
	if diags.Errors()[0].Summary() != "Unable to create GitHub app connection" {
		t.Errorf("unexpected summary: %s", diags.Errors()[0].Summary())
	}
}

func TestBuildGithubCredentialsForCreate_InvalidInstanceType(t *testing.T) {
	ctx := context.Background()
	plan := AppConnectionBaseResourceModel{
		Method:      types.StringValue(AppConnectionGithubAuthMethodPat),
		Credentials: mustGithubCredentialsObject(t, "token", "onprem", ""),
	}

	_, diags := buildGithubCredentialsForCreate(ctx, plan)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for invalid instance_type")
	}
}

func TestBuildGithubCredentialsForCreate_ServerWithoutHost(t *testing.T) {
	ctx := context.Background()
	plan := AppConnectionBaseResourceModel{
		Method:      types.StringValue(AppConnectionGithubAuthMethodPat),
		Credentials: mustGithubCredentialsObject(t, "token", "server", ""),
	}

	_, diags := buildGithubCredentialsForCreate(ctx, plan)
	if !diags.HasError() {
		t.Fatal("expected diagnostics when instance_type is server but host is empty")
	}
}

func TestBuildGithubCredentialsForUpdate_ValidServer(t *testing.T) {
	ctx := context.Background()
	plan := AppConnectionBaseResourceModel{
		Method:      types.StringValue(AppConnectionGithubAuthMethodPat),
		Credentials: mustGithubCredentialsObject(t, "ghp_new", "server", "ghe.example.com"),
	}
	state := AppConnectionBaseResourceModel{
		Method:      types.StringValue(AppConnectionGithubAuthMethodPat),
		Credentials: mustGithubCredentialsObject(t, "ghp_old", "server", "ghe.example.com"),
	}

	config, diags := buildGithubCredentialsForUpdate(ctx, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if config["personalAccessToken"] != "ghp_new" {
		t.Errorf("personalAccessToken: got %v", config["personalAccessToken"])
	}
	if config["instanceType"] != "server" {
		t.Errorf("instanceType: got %v", config["instanceType"])
	}
	if config["host"] != "ghe.example.com" {
		t.Errorf("host: got %v", config["host"])
	}
}

func TestBuildGithubCredentialsForUpdate_ServerWithoutHost(t *testing.T) {
	ctx := context.Background()
	plan := AppConnectionBaseResourceModel{
		Method:      types.StringValue(AppConnectionGithubAuthMethodPat),
		Credentials: mustGithubCredentialsObjectWithNulls(t, "token", types.StringValue("server"), types.StringNull()),
	}
	state := AppConnectionBaseResourceModel{
		Method:      types.StringValue(AppConnectionGithubAuthMethodPat),
		Credentials: mustGithubCredentialsObjectWithNulls(t, "token", types.StringValue("server"), types.StringNull()),
	}

	_, diags := buildGithubCredentialsForUpdate(ctx, plan, state)
	if !diags.HasError() {
		t.Fatal("expected diagnostics when instance_type is server but host is null")
	}
}
