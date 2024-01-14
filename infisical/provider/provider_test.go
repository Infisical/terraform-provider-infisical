// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	// service_token is from local instance
	providerConfig = `
		provider "infisical" {
  			host = "http://localhost:8080"
  			service_token = "st.65a376f3693f8c3c745b5067.d68f14db3b1d79fe99abcdd78418c74d.e5bff3309c428dcfd1e80a44f2eb6aca"
		}
	`
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"infisical": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

/*

local instance configuration of secrets

API_KEY='jij3290t233-${staging.WOAH}'
AUTH0_DOMAIN='123456-staging.eu.auth0.com'
AUTH0_ISSUER='https://${AUTH0_DOMAIN}'
DATABASE_PASSWORD='123456'
FIREBASE_API_KEY='54839523521-${dev.service-a.KEY_FOR_SMTH}'
OAUTH_CLIENT_ID='4t34t2'
OAUTH_CLIENT_SECRET='1234567-${OAUTH_CLIENT_ID}'
(in staging) WOAH='egmmg4334'

*/

func TestSecretsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig +
					`data "infisical_secrets" "common-secrets" {
  						env_slug    = "dev"
  						folder_path = "/"
					}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.infisical_secrets.common-secrets", "secrets.AUTH0_ISSUER.value", "https://123456-staging.eu.auth0.com"),
					resource.TestCheckResourceAttr("data.infisical_secrets.common-secrets", "secrets.OAUTH_CLIENT_SECRET.value", "1234567-4t34t2"),
					resource.TestCheckResourceAttr("data.infisical_secrets.common-secrets", "secrets.API_KEY.value", "jij3290t233-123456"),
					resource.TestCheckResourceAttr("data.infisical_secrets.common-secrets", "secrets.FIREBASE_API_KEY.value", "54839523521-egmmg4334"),
				),
			},
		},
	})
}
