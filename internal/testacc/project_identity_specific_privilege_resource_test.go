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

func TestAccProjectIdentitySpecificPrivilege(t *testing.T) {
	projectName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resourceName := "infisical_project_identity_specific_privilege.test_privilege"
	orgId := getIdentityOrgId()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		PreCheck:                 preCheck(t),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/project_identity_specific_privilege/project_identity_specific_privilege.tf"),
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
							identityId, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("identity_id"))
							if err != nil {
								return err
							}

							projectSlug, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("project_slug"))
							if err != nil {
								return err
							}

							_, err = infisicalApiClient.GetProjectIdentitySpecificPrivilegeBySlug(infisicalclient.GetProjectIdentitySpecificPrivilegeRequest{
								IdentityID:  identityId.(string),
								ProjectSlug: projectSlug.(string),
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
				ConfigFile: config.StaticFile("./testdata/project_identity_specific_privilege/project_identity_specific_privilege.tf"),
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
				ConfigFile: config.StaticFile("./testdata/project_identity_specific_privilege/project_identity_specific_privilege_1.tf"),
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
							id, err := tfjsonpath.Traverse(resource.AttributeValues, tfjsonpath.New("id"))
							if err != nil {
								return err
							}

							identitySpecificPrivilegeDetails, err := infisicalApiClient.GetProjectIdentitySpecificPrivilegeV2(infisicalclient.GetProjectIdentitySpecificPrivilegeV2Request{
								ID: id.(string),
							})
							if err != nil {
								return err
							}

							if len(identitySpecificPrivilegeDetails.Privilege.Permissions) != 2 {
								return fmt.Errorf("Must be 1 specific privlege permission. Received %d", len(identitySpecificPrivilegeDetails.Privilege.Permissions))
							}
							return nil
						},
					),
				},
			},
			// check for drift
			{
				ConfigFile: config.StaticFile("./testdata/project_identity_specific_privilege/project_identity_specific_privilege_1.tf"),
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
