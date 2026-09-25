package resource

import (
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// secretRotationsProviders are the secret rotation providers a rule of this type can apply to. A
// rotation is only constrained when its provider is listed on the rule.
var secretRotationsProviders = []string{
	"postgres-credentials",
	"mysql-credentials",
	"mssql-credentials",
	"oracledb-credentials",
	"unix-linux-local-account",
	"ldap-password",
}

func NewSecretValidationRuleSecretRotationsResource() resource.Resource {
	return &SecretValidationRuleBaseResource{
		Type:                    infisical.SecretValidationRuleTypeSecretRotations,
		RuleName:                "Secret Rotations",
		ResourceTypeName:        "_secret_validation_rule_secret_rotations",
		ConstraintsAttributes:   providerConstraintsAttributes("secret rotation", secretRotationsProviders),
		ReadConstraintsFromPlan: readProviderConstraintsFromPlan,
		ReadConstraintsFromApi:  readProviderConstraintsFromApi,
	}
}
