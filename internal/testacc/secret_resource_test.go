package testAcc

import (
	"fmt"
	"slices"
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

func TestAccSecret_Simple(t *testing.T) {
	projectName := "test-project"
	type SecretResourceData struct {
		ResourceName    string
		Name            string
		Value           string
		UpdatedValue    string
		Path            string
		EnvironmentSlug string
	}

	secretTfData := SecretResourceData{
		ResourceName:    "infisical_secret.test_secret",
		Name:            acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		Value:           acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		UpdatedValue:    acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		Path:            "/",
		EnvironmentSlug: "dev",
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_simple.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.Value),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.Value)),

					statecheck.ExpectSensitiveValue(secretTfData.ResourceName, tfjsonpath.New("value")),
					// check in infisical
					ExpectExternalResource(secretTfData.ResourceName,
						func(resource *tfjson.StateResource) error {
							workspaceId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("workspace_id"))
							if err != nil {
								return err
							}

							secretDetails, err := infisicalApiClient.GetSingleRawSecretByNameV3(infisicalclient.GetSingleSecretByNameV3Request{
								SecretName:  secretTfData.Name,
								WorkspaceId: workspaceId.(string),
								Environment: secretTfData.EnvironmentSlug,
								SecretPath:  secretTfData.Path,
								Type:        "shared",
							})
							if err != nil {
								return err
							}

							secretValue, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("value"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(secretDetails.Secret.SecretValue).CheckValue(secretValue); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_simple.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.Value),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.Value)),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_simple_with_reminder.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.UpdatedValue),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.UpdatedValue)),
					// check in infisical
					ExpectExternalResource(secretTfData.ResourceName,
						func(resource *tfjson.StateResource) error {
							workspaceId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("workspace_id"))
							if err != nil {
								return err
							}

							secretDetails, err := infisicalApiClient.GetSingleRawSecretByNameV3(infisicalclient.GetSingleSecretByNameV3Request{
								SecretName:  secretTfData.Name,
								WorkspaceId: workspaceId.(string),
								Environment: secretTfData.EnvironmentSlug,
								SecretPath:  secretTfData.Path,
								Type:        "shared",
							})
							if err != nil {
								return err
							}

							secretValue, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("value"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(secretDetails.Secret.SecretValue).CheckValue(secretValue); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			// checking again for drift with reminder
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_simple_with_reminder.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.UpdatedValue),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.UpdatedValue)),
				},
			},
		},
	})
}

func TestAccSecret_WithNestedPath(t *testing.T) {
	projectName := "test-project"
	type SecretResourceData struct {
		ResourceName    string
		Name            string
		Value           string
		UpdatedValue    string
		Path            string
		EnvironmentSlug string
	}

	secretTfData := SecretResourceData{
		ResourceName:    "infisical_secret.test_secret",
		Name:            acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		Value:           acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		UpdatedValue:    acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		Path:            "/deep",
		EnvironmentSlug: "dev",
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_with_nested_path.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.Value),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.Value)),

					statecheck.ExpectSensitiveValue(secretTfData.ResourceName, tfjsonpath.New("value")),
					// check in infisical
					ExpectExternalResource(secretTfData.ResourceName,
						func(resource *tfjson.StateResource) error {
							workspaceId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("workspace_id"))
							if err != nil {
								return err
							}

							secretDetails, err := infisicalApiClient.GetSingleRawSecretByNameV3(infisicalclient.GetSingleSecretByNameV3Request{
								SecretName:  secretTfData.Name,
								WorkspaceId: workspaceId.(string),
								Environment: secretTfData.EnvironmentSlug,
								SecretPath:  fmt.Sprintf("%s/nested", secretTfData.Path),
								Type:        "shared",
							})
							if err != nil {
								return err
							}

							secretValue, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("value"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(secretDetails.Secret.SecretValue).CheckValue(secretValue); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_with_nested_path.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.Value),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.Value)),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_with_nested_path.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.UpdatedValue),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.UpdatedValue)),
					// check in infisical
					ExpectExternalResource(secretTfData.ResourceName,
						func(resource *tfjson.StateResource) error {
							workspaceId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("workspace_id"))
							if err != nil {
								return err
							}

							secretDetails, err := infisicalApiClient.GetSingleRawSecretByNameV3(infisicalclient.GetSingleSecretByNameV3Request{
								SecretName:  secretTfData.Name,
								WorkspaceId: workspaceId.(string),
								Environment: secretTfData.EnvironmentSlug,
								SecretPath:  fmt.Sprintf("%s/nested", secretTfData.Path),
								Type:        "shared",
							})
							if err != nil {
								return err
							}

							secretValue, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("value"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(secretDetails.Secret.SecretValue).CheckValue(secretValue); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
		},
	})
}

func TestAccSecret_WithSecretTag(t *testing.T) {
	projectName := "test-project"
	type SecretResourceData struct {
		ResourceName    string
		Name            string
		Value           string
		UpdatedValue    string
		Path            string
		EnvironmentSlug string
		TagSlug         string
		UpdatedTagSlug  string
	}

	secretTfData := SecretResourceData{
		ResourceName:    "infisical_secret.test_secret",
		Name:            acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		Value:           acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		UpdatedValue:    acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum),
		TagSlug:         acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
		UpdatedTagSlug:  acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum),
		Path:            "/",
		EnvironmentSlug: "dev",
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_with_tag.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.Value),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
					"secret_tag_slug": config.StringVariable(secretTfData.TagSlug),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.Value)),

					statecheck.ExpectSensitiveValue(secretTfData.ResourceName, tfjsonpath.New("value")),
					// check in infisical
					ExpectExternalResource(secretTfData.ResourceName,
						func(resource *tfjson.StateResource) error {
							workspaceId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("workspace_id"))
							if err != nil {
								return err
							}

							secretDetails, err := infisicalApiClient.GetSingleRawSecretByNameV3(infisicalclient.GetSingleSecretByNameV3Request{
								SecretName:  secretTfData.Name,
								WorkspaceId: workspaceId.(string),
								Environment: secretTfData.EnvironmentSlug,
								SecretPath:  secretTfData.Path,
								Type:        "shared",
							})
							if err != nil {
								return err
							}

							containsTag := slices.ContainsFunc(secretDetails.Secret.Tags, func(e struct {
								Slug string `json:"slug"`
							}) bool {
								return e.Slug == secretTfData.TagSlug
							})
							if !containsTag {
								return fmt.Errorf("Tag %s not found in secret", secretTfData.TagSlug)
							}

							secretValue, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("value"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(secretDetails.Secret.SecretValue).CheckValue(secretValue); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_with_tag.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.Value),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
					"secret_tag_slug": config.StringVariable(secretTfData.TagSlug),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.Value)),
				},
			},
			{
				ConfigFile: config.StaticFile("./testdata/secret_resource/secret_with_tag.tf"),
				ConfigVariables: config.Variables{
					"project_name":    config.StringVariable(projectName),
					"project_slug":    config.StringVariable(projectName),
					"secret_name":     config.StringVariable(secretTfData.Name),
					"secret_value":    config.StringVariable(secretTfData.UpdatedValue),
					"secret_env_slug": config.StringVariable(secretTfData.EnvironmentSlug),
					"secret_path":     config.StringVariable(secretTfData.Path),
					"secret_tag_slug": config.StringVariable(secretTfData.UpdatedTagSlug),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("name"), knownvalue.StringExact(secretTfData.Name)),
					statecheck.ExpectKnownValue(secretTfData.ResourceName, tfjsonpath.New("value"), knownvalue.StringExact(secretTfData.UpdatedValue)),
					// check in infisical
					ExpectExternalResource(secretTfData.ResourceName,
						func(resource *tfjson.StateResource) error {
							workspaceId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("workspace_id"))
							if err != nil {
								return err
							}

							secretDetails, err := infisicalApiClient.GetSingleRawSecretByNameV3(infisicalclient.GetSingleSecretByNameV3Request{
								SecretName:  secretTfData.Name,
								WorkspaceId: workspaceId.(string),
								Environment: secretTfData.EnvironmentSlug,
								SecretPath:  secretTfData.Path,
								Type:        "shared",
							})
							if err != nil {
								return err
							}

							containsTag := slices.ContainsFunc(secretDetails.Secret.Tags, func(e struct {
								Slug string `json:"slug"`
							}) bool {
								return e.Slug == secretTfData.UpdatedTagSlug
							})
							if !containsTag {
								return fmt.Errorf("Tag %s not found in secret", secretTfData.TagSlug)
							}

							secretValue, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("value"))
							if err != nil {
								return err
							}

							if err := knownvalue.StringExact(secretDetails.Secret.SecretValue).CheckValue(secretValue); err != nil {
								return err
							}
							return nil
						},
					),
				},
			},
		},
	})
}
