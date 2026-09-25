package resource

import (
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var dynamicSecretsProviders = []string{
	"sql-database",
	"milvus",
}

func NewSecretValidationRuleDynamicSecretsResource() resource.Resource {
	return &SecretValidationRuleBaseResource{
		Type:                    infisical.SecretValidationRuleTypeDynamicSecrets,
		RuleName:                "Dynamic Secrets",
		ResourceTypeName:        "_secret_validation_rule_dynamic_secrets",
		ConstraintsAttributes:   providerConstraintsAttributes("dynamic secret", dynamicSecretsProviders),
		ReadConstraintsFromPlan: readProviderConstraintsFromPlan,
		ReadConstraintsFromApi:  readProviderConstraintsFromApi,
	}
}
