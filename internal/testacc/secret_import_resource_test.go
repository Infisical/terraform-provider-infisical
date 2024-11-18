package testAcc

/*

import (
	"fmt"
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

func TestAccSecretImport(t *testing.T) {
	slug := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	tfResourceName := "infisical_secret_import.import-1"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/secret_imports/secret_imports_simple.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(slug),
					"project_slug": config.StringVariable(slug),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// check in infisical
					ExpectExternalResource(tfResourceName,
						func(resource *tfjson.StateResource) error {
							environmentSlug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("environment_slug"))
							if err != nil {
								return err
							}

							projectId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_id"))
							if err != nil {
								return err
							}

							secretImportsList, err := infisicalApiClient.GetSecretImportList(infisicalclient.ListSecretImportRequest{
								Environment: environmentSlug.(string),
								ProjectID:   projectId.(string),
								SecretPath:  "/",
							})

							if len(secretImportsList.SecretImports) == 3 {
								return fmt.Errorf("Must be only 3 imports. Found :%d", len(secretImportsList.SecretImports))
							}

							secretImportsList, err = infisicalApiClient.GetSecretImportList(infisicalclient.ListSecretImportRequest{
								Environment: environmentSlug.(string),
								ProjectID:   projectId.(string),
								SecretPath:  "/nested",
							})

							if len(secretImportsList.SecretImports) == 1 {
								return fmt.Errorf("Must be only 1 imports. Found :%d", len(secretImportsList.SecretImports))
							}

							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/secret_imports/secret_imports_simple.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(slug),
					"project_slug": config.StringVariable(slug),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/secret_imports/secret_imports_simple_2.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(slug),
					"project_slug": config.StringVariable(slug),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// check in infisical
					ExpectExternalResource(tfResourceName,
						func(resource *tfjson.StateResource) error {
							environmentSlug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("environment_slug"))
							if err != nil {
								return err
							}

							projectId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_id"))
							if err != nil {
								return err
							}

							secretImportsList, err := infisicalApiClient.GetSecretImportList(infisicalclient.ListSecretImportRequest{
								Environment: environmentSlug.(string),
								ProjectID:   projectId.(string),
								SecretPath:  "/",
							})

							if len(secretImportsList.SecretImports) == 1 {
								return fmt.Errorf("Must be only 1 imports. Found :%d", len(secretImportsList.SecretImports))
							}

							secretImportsList, err = infisicalApiClient.GetSecretImportList(infisicalclient.ListSecretImportRequest{
								Environment: environmentSlug.(string),
								ProjectID:   projectId.(string),
								SecretPath:  "/nested",
							})

							if len(secretImportsList.SecretImports) == 1 {
								return fmt.Errorf("Must be only 1 imports. Found :%d", len(secretImportsList.SecretImports))
							}

							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/secret_imports/secret_imports_simple.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(slug),
					"project_slug": config.StringVariable(slug),
				},
			},
		},
	})
}

*/
