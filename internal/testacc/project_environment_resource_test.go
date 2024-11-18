package testAcc

// akhilmhdh: Not possible to test this due to preexisting environment
// can only be done either after template support or no environment when setting up project
/*
import (
	"fmt"
	"strings"
	infisicalclient "terraform-provider-infisical/internal/client"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// akhilmhdh: Missing testcase for position defined environment
// This is because a new project has default environment and causes drift
func TestAccProjectEnvironment(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	environmentSlug := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	projectResource := "infisical_project_environment.uat-1"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/project_environment/project_environment.tf"),
				ConfigVariables: config.Variables{
					"project_name":     config.StringVariable(projectName),
					"project_slug":     config.StringVariable(projectName),
					"environment_slug": config.StringVariable(environmentSlug),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// check in infisical
					ExpectExternalResource(projectResource,
						func(resource *tfjson.StateResource) error {
							newEnvironmentPositions := map[string]int{
								environmentSlug + "-1": 1,
								environmentSlug + "-2": 2,
								environmentSlug + "-3": 3,
							}

							projectId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_id"))
							if err != nil {
								return err
							}

							projectDetail, err := infisicalApiClient.GetProjectById(infisicalclient.GetProjectByIdRequest{
								ID: projectId.(string),
							})
							var newEnvironments []infisicalclient.ProjectEnvironment
							for envPos, env := range projectDetail.Environments {
								if strings.HasPrefix(env.Slug, environmentSlug) {
									newEnvironments = append(newEnvironments, env)
									if newEnvironmentPositions[env.Slug] != envPos+1 {
										return fmt.Errorf("Invalid environment position. Environment %s, should be %d, received %d, %v", env.Slug, newEnvironmentPositions[env.Slug], envPos+1, projectDetail.Environments)
									}
								}
							}

							if len(newEnvironments) != 3 {
								return fmt.Errorf("Missing environments. Found only %d. Need %d", len(newEnvironments), 3)
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/project_environment/project_environment.tf"),
				ConfigVariables: config.Variables{
					"project_name":     config.StringVariable(projectName),
					"project_slug":     config.StringVariable(projectName),
					"environment_slug": config.StringVariable(environmentSlug),
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

*/
