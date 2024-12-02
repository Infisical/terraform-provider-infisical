package testAcc

import (
	infisicalclient "terraform-provider-infisical/internal/client"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectUser(t *testing.T) {
	membershipIdStateCompare := statecheck.CompareValue(compare.ValuesSame())
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	email := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum) + "@example.com"

	resourceName := "infisical_project_user.test_user"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/project_user/project_user.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"email":        config.StringVariable(email),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					membershipIdStateCompare.AddStateValue(
						resourceName,
						tfjsonpath.New("membership_id"),
					),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("username"), knownvalue.StringExact(email)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("user").AtMapKey("email"), knownvalue.StringExact(email)),
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							projectId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_id"))
							if err != nil {
								return err
							}

							userDetails, err := infisicalApiClient.GetProjectUserByUsername(infisicalclient.GetProjectUserByUserNameRequest{
								ProjectID: projectId.(string),
								Username:  email,
							})
							if err != nil {
								return err
							}

							membershipId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("membership_id"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(userDetails.Membership.ID).CheckValue(membershipId); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/project_user/project_user.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"email":        config.StringVariable(email),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					membershipIdStateCompare.AddStateValue(
						resourceName,
						tfjsonpath.New("membership_id"),
					),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("username"), knownvalue.StringExact(email)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("user").AtMapKey("email"), knownvalue.StringExact(email)),
				},
			},
		},
	})
}
