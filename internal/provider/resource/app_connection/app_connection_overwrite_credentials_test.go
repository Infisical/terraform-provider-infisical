package resource

import (
	"context"
	"net/http/httptest"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// After an import, or when the credentials hash changed outside Terraform, Read replaces the state
// credentials with OverwriteCredentialsFields. Update then reads them with Credentials.As, which fails
// on a null object, so every variant must write an object with its schema's credential attributes.
func TestEveryAppConnectionOverwritesCredentialsWithSchemaShape(t *testing.T) {
	ctx := context.Background()
	constructors := map[string]func() resource.Resource{
		"gcp": NewAppConnectionGcpResource, "aws": NewAppConnectionAwsResource, "azure": NewAppConnectionAzureResource,
		"azure_key_vault": NewAppConnectionAzureKeyVaultResource, "1password": NewAppConnection1PasswordResource,
		"azure_app_configuration": NewAppConnectionAzureAppConfigurationResource, "render": NewAppConnectionRenderResource,
		"azure_devops": NewAppConnectionAzureDevOpsResource, "mysql": NewAppConnectionMySqlResource,
		"mssql": NewAppConnectionMsSqlResource, "postgres": NewAppConnectionPostgresResource,
		"oracledb": NewAppConnectionOracleDbResource, "bitbucket": NewAppConnectionBitbucketResource,
		"databricks": NewAppConnectionDatabricksResource, "cloudflare": NewAppConnectionCloudflareResource,
		"supabase": NewAppConnectionSupabaseResource, "flyio": NewAppConnectionFlyioResource,
		"ldap": NewAppConnectionLdapResource, "gitlab": NewAppConnectionGitlabResource,
		"github": NewAppConnectionGithubResource, "hashicorp_vault": NewAppConnectionHashicorpVaultResource,
		"datadog": NewAppConnectionDatadogResource, "circleci": NewAppConnectionCircleCIResource,
		"digicert": NewAppConnectionDigiCertResource, "chef": NewAppConnectionChefResource,
		"f5_big_ip": NewAppConnectionF5BigIpResource, "kemp_loadmaster": NewAppConnectionKempLoadMasterResource,
		"netscaler": NewAppConnectionNetScalerResource, "nutanix_prism_central": NewAppConnectionNutanixPrismCentralResource,
		"palo_alto_networks": NewAppConnectionPaloAltoNetworksResource, "ssh": NewAppConnectionSshResource,
		"winrm": NewAppConnectionWinRMResource, "adcs": NewAppConnectionADCSResource,
		"spacelift": NewAppConnectionSpaceliftResource, "litellm": NewAppConnectionLiteLLMResource,
	}

	for name, constructor := range constructors {
		t.Run(name, func(t *testing.T) {
			r, ok := constructor().(*AppConnectionBaseResource)
			if !ok {
				t.Fatal("constructor did not return the base resource")
			}

			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			want := schemaResp.Schema.Attributes["credentials"].GetType()

			model := AppConnectionBaseResourceModel{Credentials: types.ObjectNull(map[string]attr.Type{})}
			if diags := r.OverwriteCredentialsFields(&model); diags.HasError() {
				t.Fatal(diags)
			}
			if model.Credentials.IsNull() {
				t.Fatal("OverwriteCredentialsFields left the credentials null")
			}
			if got := model.Credentials.Type(ctx); !got.Equal(want) {
				t.Errorf("OverwriteCredentialsFields wrote %v, the schema has %v", got, want)
			}
		})
	}
}

func TestCloudflareAppConnectionUpdateAfterImport(t *testing.T) {
	ctx := context.Background()
	backend := newFakeADCSConnectionBackend(t)
	backend.app = "cloudflare"
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)

	r, ok := NewAppConnectionCloudflareResource().(*AppConnectionBaseResource)
	if !ok {
		t.Fatal("NewAppConnectionCloudflareResource did not return the base resource")
	}
	r.client = &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	backend.connections[testADCSConnectionID] = map[string]any{
		"id": testADCSConnectionID, "name": "cloudflare-connection", "description": "", "app": "cloudflare",
		"method": CloudflareAppConnectionApiTokenMethod, "projectId": nil, "credentialsHash": "hash-1",
	}

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

	credentials, diags := types.ObjectValue(map[string]attr.Type{
		"account_id": types.StringType,
		"api_token":  types.StringType,
	}, map[string]attr.Value{
		"account_id": types.StringValue("account-1"),
		"api_token":  types.StringValue("token-1"),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, &AppConnectionBaseResourceModel{
		ID:              types.StringValue(testADCSConnectionID),
		Name:            types.StringValue("cloudflare-connection"),
		Method:          types.StringValue(CloudflareAppConnectionApiTokenMethod),
		Description:     types.StringNull(),
		ProjectId:       types.StringNull(),
		Credentials:     credentials,
		CredentialsHash: types.StringUnknown(),
	}); diags.HasError() {
		t.Fatal(diags)
	}

	updateResp := resource.UpdateResponse{State: readResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: readResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("the first apply after an import must send the configured credentials, got %v", updateResp.Diagnostics)
	}

	sent, ok := backend.lastBody["credentials"].(map[string]any)
	if !ok || sent["accountId"] != "account-1" || sent["apiToken"] != "token-1" {
		t.Errorf("update did not send the configured credentials, sent %v", backend.lastBody)
	}
}
