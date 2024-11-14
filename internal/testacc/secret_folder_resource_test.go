package testAcc

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

func TestAccSecretFolder(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	changedRandomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	slug := "test-project"
	tfResourceName := "infisical_secret_folder.folder1"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/secret_folder/secret_folder_simple.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(randomName),
					"project_slug": config.StringVariable(slug),
					"folder_name":  config.StringVariable(randomName),
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

							folderPaths := []string{"/", fmt.Sprintf("/%s", randomName), fmt.Sprintf("/%s/%s", randomName, randomName)}
							for _, folderPath := range folderPaths {
								folderList, err := infisicalApiClient.GetSecretFolderList(infisicalclient.ListSecretFolderRequest{
									Environment: environmentSlug.(string),
									ProjectID:   projectId.(string),
									SecretPath:  folderPath,
								})
								if err != nil {
									return err
								}

								if len(folderList.Folders) != 1 {
									return fmt.Errorf("Must be only one folder. Found :%d", len(folderList.Folders))
								}

								if folderList.Folders[0].Name != randomName {
									return fmt.Errorf("Invalid folder found. Fouund: %s, Should be: %s", folderList.Folders[0].Name, randomName)
								}
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/secret_folder/secret_folder_simple.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(randomName),
					"project_slug": config.StringVariable(slug),
					"folder_name":  config.StringVariable(randomName),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/secret_folder/secret_folder_simple.tf"),
				ConfigVariables: config.Variables{
					"project_name": config.StringVariable(randomName),
					"project_slug": config.StringVariable(slug),
					"folder_name":  config.StringVariable(changedRandomName),
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

							folderPaths := []string{"/", fmt.Sprintf("/%s", changedRandomName), fmt.Sprintf("/%s/%s", changedRandomName, changedRandomName)}
							for _, folderPath := range folderPaths {
								folderList, err := infisicalApiClient.GetSecretFolderList(infisicalclient.ListSecretFolderRequest{
									Environment: environmentSlug.(string),
									ProjectID:   projectId.(string),
									SecretPath:  folderPath,
								})
								if err != nil {
									return err
								}

								if len(folderList.Folders) != 1 {
									return fmt.Errorf("Must be only one folder. Found :%d", len(folderList.Folders))
								}

								if folderList.Folders[0].Name != changedRandomName {
									return fmt.Errorf("Invalid folder found. Fouund: %s, Should be: %s", folderList.Folders[0].Name, changedRandomName)
								}
							}
							return nil
						},
					),
				},
			},
		},
	})
}
