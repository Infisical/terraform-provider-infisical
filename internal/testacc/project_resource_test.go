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

func TestAccProject(t *testing.T) {
	idValueCompare := statecheck.CompareValue(compare.ValuesSame())
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	changedRandomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	slug := "test-project"

	projectResource := "infisical_project.test_project"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/project_resource/project_1.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idValueCompare.AddStateValue(
						projectResource,
						tfjsonpath.New("id"),
					),
					statecheck.ExpectKnownValue(projectResource, tfjsonpath.New("name"), knownvalue.StringExact(randomName)),
					statecheck.ExpectKnownValue(projectResource, tfjsonpath.New("slug"), knownvalue.StringExact(slug)),
					// check in infisical
					ExpectExternalResource(projectResource,
						func(resource *tfjson.StateResource) error {
							id, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("id"))
							if err != nil {
								return err
							}

							projectDetail, err := infisicalApiClient.GetProjectById(infisicalclient.GetProjectByIdRequest{
								ID: id.(string),
							})

							slug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("slug"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(projectDetail.Slug).CheckValue(slug); err != nil {
								return err
							}
							return nil
						},
					),
				},
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(randomName),
					"project_slug": config.StringVariable(slug),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/project_resource/project_1.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idValueCompare.AddStateValue(
						projectResource,
						tfjsonpath.New("id"),
					),
					statecheck.ExpectKnownValue(projectResource, tfjsonpath.New("name"), knownvalue.StringExact(randomName)),
					statecheck.ExpectKnownValue(projectResource, tfjsonpath.New("slug"), knownvalue.StringExact(slug)),
				},
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(randomName),
					"project_slug": config.StringVariable(slug),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/project_resource/project_1.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idValueCompare.AddStateValue(
						projectResource,
						tfjsonpath.New("id"),
					),
					statecheck.ExpectKnownValue(projectResource, tfjsonpath.New("name"), knownvalue.StringExact(changedRandomName)),
					statecheck.ExpectKnownValue(projectResource, tfjsonpath.New("slug"), knownvalue.StringExact(slug)),
					// check in infisical
					ExpectExternalResource(projectResource,
						func(resource *tfjson.StateResource) error {
							id, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("id"))
							if err != nil {
								return err
							}
							projectDetail, err := infisicalApiClient.GetProjectById(infisicalclient.GetProjectByIdRequest{
								ID: id.(string),
							})
							slug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("slug"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(projectDetail.Slug).CheckValue(slug); err != nil {
								return err
							}
							return nil
						},
					),
				},
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(changedRandomName),
					"project_slug": config.StringVariable(slug),
				},
			},
		},
	})
}
