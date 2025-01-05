package testAcc

import (
	infisicalclient "terraform-provider-infisical/internal/client"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccIdentity_UniversalAuth(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "infisical_identity.universal-auth"
	authResourceName := "infisical_identity_universal_auth.ua"
	orgId := getIdentityOrgId()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_ua_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectUnknownOutputValue("client_secret"),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("id"))
							if err != nil {
								return err
							}

							identityDetails, err := infisicalApiClient.GetIdentity(infisicalclient.GetIdentityRequest{
								IdentityID: identityId.(string),
							})
							if err != nil {
								return err
							}

							name, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("name"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.Name).CheckValue(name); err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.AuthMethods[0]).CheckValue("universal-auth"); err != nil {
								return err
							}
							return nil
						},
					),
					ExpectExternalResource(authResourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("identity_id"))
							if err != nil {
								return err
							}

							_, err = infisicalApiClient.GetIdentityUniversalAuth(infisicalclient.GetIdentityUniversalAuthRequest{
								IdentityID: identityId.(string),
							})

							if err != nil {
								return err
							}

							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_ua_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccIdentity_AwsAuth(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "infisical_identity.aws-auth"
	authResourceName := "infisical_identity_aws_auth.aws-auth"
	orgId := getIdentityOrgId()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_aws_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("id"))
							if err != nil {
								return err
							}

							identityDetails, err := infisicalApiClient.GetIdentity(infisicalclient.GetIdentityRequest{
								IdentityID: identityId.(string),
							})
							if err != nil {
								return err
							}

							name, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("name"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.Name).CheckValue(name); err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.AuthMethods[0]).CheckValue("aws-auth"); err != nil {
								return err
							}
							return nil
						},
					),
					ExpectExternalResource(authResourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("identity_id"))
							if err != nil {
								return err
							}

							_, err = infisicalApiClient.GetIdentityAwsAuth(infisicalclient.GetIdentityAwsAuthRequest{
								IdentityID: identityId.(string),
							})

							if err != nil {
								return err
							}

							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_aws_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccIdentity_AzureAuth(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "infisical_identity.azure-auth"
	authResourceName := "infisical_identity_azure_auth.azure-auth"
	orgId := getIdentityOrgId()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_azure_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("id"))
							if err != nil {
								return err
							}

							identityDetails, err := infisicalApiClient.GetIdentity(infisicalclient.GetIdentityRequest{
								IdentityID: identityId.(string),
							})
							if err != nil {
								return err
							}

							name, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("name"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.Name).CheckValue(name); err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.AuthMethods[0]).CheckValue("azure-auth"); err != nil {
								return err
							}
							return nil
						},
					),
					ExpectExternalResource(authResourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("identity_id"))
							if err != nil {
								return err
							}

							_, err = infisicalApiClient.GetIdentityAzureAuth(infisicalclient.GetIdentityAzureAuthRequest{
								IdentityID: identityId.(string),
							})

							if err != nil {
								return err
							}

							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_azure_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccIdentity_GcpAuth(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "infisical_identity.gcp-auth"
	authResourceName := "infisical_identity_gcp_auth.gcp-auth"
	orgId := getIdentityOrgId()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_gcp_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("id"))
							if err != nil {
								return err
							}

							identityDetails, err := infisicalApiClient.GetIdentity(infisicalclient.GetIdentityRequest{
								IdentityID: identityId.(string),
							})
							if err != nil {
								return err
							}

							name, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("name"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.Name).CheckValue(name); err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.AuthMethods[0]).CheckValue("gcp-auth"); err != nil {
								return err
							}
							return nil
						},
					),
					ExpectExternalResource(authResourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("identity_id"))
							if err != nil {
								return err
							}

							_, err = infisicalApiClient.GetIdentityGcpAuth(infisicalclient.GetIdentityGcpAuthRequest{
								IdentityID: identityId.(string),
							})

							if err != nil {
								return err
							}

							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_gcp_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccIdentity_K8sAuth(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "infisical_identity.k8-auth"
	authResourceName := "infisical_identity_kubernetes_auth.k8-auth"
	orgId := getIdentityOrgId()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_k8_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("id"))
							if err != nil {
								return err
							}

							identityDetails, err := infisicalApiClient.GetIdentity(infisicalclient.GetIdentityRequest{
								IdentityID: identityId.(string),
							})
							if err != nil {
								return err
							}

							name, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("name"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.Name).CheckValue(name); err != nil {
								return err
							}
							if err := knownvalue.StringExact(identityDetails.Identity.AuthMethods[0]).CheckValue("kubernetes-auth"); err != nil {
								return err
							}
							return nil
						},
					),
					ExpectExternalResource(authResourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("identity_id"))
							if err != nil {
								return err
							}

							_, err = infisicalApiClient.GetIdentityKubernetesAuth(infisicalclient.GetIdentityKubernetesAuthRequest{
								IdentityID: identityId.(string),
							})

							if err != nil {
								return err
							}

							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/identity/identity_k8_auth.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"org_id":       config.StringVariable(orgId),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
