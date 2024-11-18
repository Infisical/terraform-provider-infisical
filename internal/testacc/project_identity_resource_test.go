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

func TestAccProjectIdentity(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "infisical_project_identity.test_identity"
	orgId := getIdentityOrgId()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/project_identity/project_identity.tf"),
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
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("roles").AtSliceIndex(0).AtMapKey("role_slug"), knownvalue.StringExact("admin")),
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("identity_id"))
							if err != nil {
								return err
							}

							projectId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_id"))
							if err != nil {
								return err
							}

							identityDetails, err := infisicalApiClient.GetProjectIdentityByID(infisicalclient.GetProjectIdentityByIDRequest{
								IdentityID: identityId.(string),
								ProjectID:  projectId.(string),
							})
							if err != nil {
								return err
							}

							roleSlug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("roles").AtSliceIndex(0).AtMapKey("role_slug"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(identityDetails.Membership.Roles[0].Role).CheckValue(roleSlug); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/project_identity/project_identity.tf"),
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
			{
				ConfigFile: config.StaticFile("./testdata/project_identity/project_identity_1.tf"),
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
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("roles").AtSliceIndex(0).AtMapKey("role_slug"), knownvalue.StringExact("member")),
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("identity_id"))
							if err != nil {
								return err
							}

							projectId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_id"))
							if err != nil {
								return err
							}

							identityDetails, err := infisicalApiClient.GetProjectIdentityByID(infisicalclient.GetProjectIdentityByIDRequest{
								IdentityID: identityId.(string),
								ProjectID:  projectId.(string),
							})
							if err != nil {
								return err
							}

							roleSlug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("roles").AtSliceIndex(0).AtMapKey("role_slug"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(identityDetails.Membership.Roles[0].Role).CheckValue(roleSlug); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/project_identity/project_identity_1.tf"),
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
