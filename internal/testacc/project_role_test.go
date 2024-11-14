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

func TestAccProjectRole(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	roleName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "infisical_project_role.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/project_role/project_role.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"role_slug":    config.StringVariable(roleName),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(roleName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("slug"), knownvalue.StringExact(roleName)),
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							projectSlug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_slug"))
							if err != nil {
								return err
							}

							roleDetails, err := infisicalApiClient.GetProjectRoleBySlug(infisicalclient.GetProjectRoleBySlugRequest{
								ProjectSlug: projectSlug.(string),
								RoleSlug:    roleName,
							})

							slug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("slug"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(roleDetails.Role.Slug).CheckValue(slug); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/project_role/project_role.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"role_slug":    config.StringVariable(roleName),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(roleName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("slug"), knownvalue.StringExact(roleName)),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/project_role/project_role_2.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"role_slug":    config.StringVariable(roleName),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(roleName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("slug"), knownvalue.StringExact(roleName)),
					// check in infisical
					ExpectExternalResource(resourceName,
						func(resource *tfjson.StateResource) error {
							projectSlug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_slug"))
							if err != nil {
								return err
							}

							roleDetails, err := infisicalApiClient.GetProjectRoleBySlug(infisicalclient.GetProjectRoleBySlugRequest{
								ProjectSlug: projectSlug.(string),
								RoleSlug:    roleName,
							})

							slug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("slug"))
							if err != nil {
								return err
							}
							if err := knownvalue.StringExact(roleDetails.Role.Slug).CheckValue(slug); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/project_role/project_role_2.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(projectName),
					"project_slug": config.StringVariable(projectName),
					"role_slug":    config.StringVariable(roleName),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact(roleName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("slug"), knownvalue.StringExact(roleName)),
				},
			},
		},
	})
}
