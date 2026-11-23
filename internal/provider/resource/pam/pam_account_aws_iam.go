package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type PamAccountAwsIamConnectionDetailsModel struct {
	RoleArn types.String `tfsdk:"role_arn"`
}

// NewPamAccountAwsIamResource manages accounts that store no credentials: Infisical assumes the role
// with its own AWS identity and hands out short-lived STS credentials.
func NewPamAccountAwsIamResource() resource.Resource {
	return &PamAccountBaseResource{
		AccountType:      "aws-iam",
		AccountTypeName:  "AWS IAM",
		ResourceTypeName: "_pam_account_aws_iam",
		Prerequisite:     "The test assumes `role_arn`, so the role must trust Infisical before the account is created.",
		ConnectionDetailsAttributes: map[string]schema.Attribute{
			"role_arn": schema.StringAttribute{
				Description: "The ARN of the IAM role Infisical assumes to give access to the account. Its trust policy must allow Infisical, with your organization ID as the external ID. See [AWS IAM accounts](https://infisical.com/docs/documentation/platform/pam/accounts/aws-iam) for the trust policy to use.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 2048)},
			},
		},

		ConnectionDetailsToAPI: func(ctx context.Context, connectionDetails types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountAwsIamConnectionDetailsModel
			diags := connectionDetails.As(ctx, &model, basetypes.ObjectAsOptions{})
			return map[string]any{"roleArn": model.RoleArn.ValueString()}, diags
		},

		ConnectionDetailsFromAPI: func(connectionDetails map[string]any) any {
			return PamAccountAwsIamConnectionDetailsModel{RoleArn: apiString(connectionDetails, "roleArn")}
		},
	}
}
