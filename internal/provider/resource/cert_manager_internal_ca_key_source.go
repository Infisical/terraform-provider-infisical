package resource

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	CA_KEY_SOURCE_INFISICAL = "infisical"
	CA_KEY_SOURCE_HSM       = "hsm"
	CA_KEY_SOURCE_AWS_KMS   = "aws-kms"
)

var (
	SUPPORTED_CA_KEY_SOURCES = []string{CA_KEY_SOURCE_INFISICAL, CA_KEY_SOURCE_HSM, CA_KEY_SOURCE_AWS_KMS}

	HSM_CA_KEY_ALGORITHMS = []string{"RSA_2048", "RSA_4096", "EC_prime256v1", "EC_secp384r1", "ML-DSA-44", "ML-DSA-65", "ML-DSA-87"}

	CA_KEY_SOURCE_MIN_VERSIONS = map[string]string{
		CA_KEY_SOURCE_HSM:     "v0.161.13",
		CA_KEY_SOURCE_AWS_KMS: "v0.166.9",
	}

	CA_KEY_SOURCE_ATTRIBUTES = map[string]string{
		CA_KEY_SOURCE_HSM:     "key_source and hsm_connector_id",
		CA_KEY_SOURCE_AWS_KMS: "key_source, app_connection_id and region",
	}
)

func hsmCAKeyAlgorithms() []string {
	return slices.DeleteFunc(slices.Clone(HSM_CA_KEY_ALGORITHMS), func(algorithm string) bool {
		return !slices.Contains(SUPPORTED_ROOT_CA_KEY_ALGORITHMS, algorithm)
	})
}

func certManagerInternalCAKeySourcePlanModifiers() []planmodifier.String {
	return []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
		stringplanmodifier.RequiresReplaceIfConfigured(),
	}
}

func certManagerInternalCAKeySourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"key_source": schema.StringAttribute{
			Description: "Where the CA's signing key is generated and kept. Supported values: " + strings.Join(SUPPORTED_CA_KEY_SOURCES, ", ") + ". " +
				"When omitted, an existing CA keeps the key source it has in Infisical, and a CA that Terraform creates, including one it creates again (replaced, tainted or deleted outside Terraform), uses '" + CA_KEY_SOURCE_INFISICAL + "'. Set it to keep the key of a re-created CA in the HSM or AWS KMS. " +
				"'infisical' keeps the key in Infisical. " +
				"'hsm' generates the key in the HSM reached through `hsm_connector_id`, and supports the key algorithms " + strings.Join(hsmCAKeyAlgorithms(), ", ") + ". An ML-DSA key in an HSM needs Infisical v0.167.5 or later. Infisical does not remove the key from the HSM when the CA is deleted. Requires an Enterprise license and Infisical " + CA_KEY_SOURCE_MIN_VERSIONS[CA_KEY_SOURCE_HSM] + " or later. " +
				"'aws-kms' generates the key in AWS KMS through `app_connection_id` in `region`, and AWS KMS makes every signature. When the CA is deleted, Infisical asks AWS KMS to schedule the key for deletion. Requires an Enterprise license and Infisical " + CA_KEY_SOURCE_MIN_VERSIONS[CA_KEY_SOURCE_AWS_KMS] + " or later. " +
				"Changing it replaces the CA.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: certManagerInternalCAKeySourcePlanModifiers(),
			Validators: []validator.String{
				stringvalidator.OneOf(SUPPORTED_CA_KEY_SOURCES...),
			},
		},
		"hsm_connector_id": schema.StringAttribute{
			Description:   "The ID of the HSM Connector to generate and keep the CA's signing key in. Required when `key_source` is 'hsm', and only allowed then. Find it under Certificate Manager > Settings > HSM Connectors. Changing it replaces the CA.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: certManagerInternalCAKeySourcePlanModifiers(),
			Validators: []validator.String{
				infisicaltf.UuidValidator,
			},
		},
		"app_connection_id": schema.StringAttribute{
			Description:   "The ID of the AWS App Connection used to create the CA's signing key in AWS KMS and to sign with it. Required when `key_source` is 'aws-kms', and only allowed then. Changing it replaces the CA.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: certManagerInternalCAKeySourcePlanModifiers(),
			Validators: []validator.String{
				infisicaltf.UuidValidator,
			},
		},
		"region": schema.StringAttribute{
			Description:   "The AWS region to create the CA's signing key in, for example 'us-east-1'. Required when `key_source` is 'aws-kms', and only allowed then. Changing it replaces the CA.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: certManagerInternalCAKeySourcePlanModifiers(),
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
	}
}

func (r *certManagerInternalCAResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config certManagerInternalCAResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.KeySource.IsUnknown() {
		return
	}

	keySource := plannedKeySource(config.KeySource)

	requireOnlyFor := func(attribute string, value types.String, owner string) {
		if keySource == owner && value.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root(attribute), "Missing "+attribute,
				fmt.Sprintf("%s is required when key_source is '%s'.", attribute, owner))
		}
		if keySource != owner && !value.IsNull() && !value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root(attribute), "Unexpected "+attribute,
				fmt.Sprintf("%s can only be set when key_source is '%s'. Remove it, or set key_source to '%s'.", attribute, owner, owner))
		}
	}
	requireOnlyFor("hsm_connector_id", config.HsmConnectorId, CA_KEY_SOURCE_HSM)
	requireOnlyFor("app_connection_id", config.AppConnectionId, CA_KEY_SOURCE_AWS_KMS)
	requireOnlyFor("region", config.Region, CA_KEY_SOURCE_AWS_KMS)

	if keySource == CA_KEY_SOURCE_HSM && !config.KeyAlgorithm.IsNull() && !config.KeyAlgorithm.IsUnknown() &&
		!slices.Contains(HSM_CA_KEY_ALGORITHMS, config.KeyAlgorithm.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("key_algorithm"), "Unsupported key_algorithm for HSM",
			fmt.Sprintf("key_algorithm '%s' is not supported when key_source is 'hsm'. Use one of: %s.",
				config.KeyAlgorithm.ValueString(), strings.Join(hsmCAKeyAlgorithms(), ", ")))
	}
}

func (r *certManagerInternalCAResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || !req.State.Raw.IsNull() {
		return
	}

	var config, plan certManagerInternalCAResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.KeySource.IsNull() {
		plan.KeySource = types.StringValue(CA_KEY_SOURCE_INFISICAL)
	}
	for _, field := range []struct{ config, planned *types.String }{
		{&config.HsmConnectorId, &plan.HsmConnectorId},
		{&config.AppConnectionId, &plan.AppConnectionId},
		{&config.Region, &plan.Region},
	} {
		if field.config.IsNull() {
			*field.planned = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func plannedKeySource(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return CA_KEY_SOURCE_INFISICAL
	}
	return value.ValueString()
}

func certManagerInternalCAKeySourceConfiguration(plan certManagerInternalCAResourceModel, configuration *infisical.CertificateAuthorityConfiguration) {
	configuration.KeySource = plan.KeySource.ValueString()
	configuration.HsmConnectorId = plan.HsmConnectorId.ValueString()
	configuration.AppConnectionId = plan.AppConnectionId.ValueString()
	configuration.Region = plan.Region.ValueString()
}

func certManagerInternalCAKeySourceFromAPI(configuration infisical.CertificateAuthorityConfiguration, state *certManagerInternalCAResourceModel) {
	keySource := configuration.KeySource
	if keySource == "" {
		keySource = CA_KEY_SOURCE_INFISICAL
	}
	state.KeySource = types.StringValue(keySource)
	state.HsmConnectorId = stringValueOrNull(configuration.HsmConnectorId)
	state.AppConnectionId = stringValueOrNull(configuration.AppConnectionId)
	state.Region = stringValueOrNull(configuration.Region)
}

func fillUnknownKeySourceFromAPI(configuration infisical.CertificateAuthorityConfiguration, plan *certManagerInternalCAResourceModel) {
	var api certManagerInternalCAResourceModel
	certManagerInternalCAKeySourceFromAPI(configuration, &api)
	for _, field := range []struct{ planned, fromAPI *types.String }{
		{&plan.KeySource, &api.KeySource},
		{&plan.HsmConnectorId, &api.HsmConnectorId},
		{&plan.AppConnectionId, &api.AppConnectionId},
		{&plan.Region, &api.Region},
	} {
		if field.planned.IsUnknown() {
			*field.planned = *field.fromAPI
		}
	}
}

var internalCAReadBackDelays = []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}

func (r *certManagerInternalCAResource) readCreatedCA(ctx context.Context, id string) (infisical.GetCAResponse, error) {
	var ca infisical.GetCAResponse
	var err error
	for _, delay := range internalCAReadBackDelays {
		select {
		case <-ctx.Done():
			return ca, ctx.Err()
		case <-time.After(delay):
		}
		if ca, err = r.client.GetInternalCA(infisical.GetCARequest{CAId: id}); err == nil {
			return ca, nil
		}
	}
	return ca, err
}

func (r *certManagerInternalCAResource) confirmCreatedKeySource(ctx context.Context, plan *certManagerInternalCAResourceModel, created infisical.CreateInternalCAResponse, resp *resource.CreateResponse) bool {
	configuration := created.Configuration
	if configuration.KeySource == "" && plannedKeySource(plan.KeySource) == CA_KEY_SOURCE_HSM {
		ca, err := r.readCreatedCA(ctx, created.Id)
		if err != nil {
			plan.Id = types.StringValue(created.Id)
			plan.Status = types.StringValue(created.Status)
			fillUnknownKeySourceFromAPI(created.Configuration, plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
			resp.Diagnostics.AddError("Error creating CA",
				"The CA was created, but reading it back to confirm where its signing key is kept failed: "+err.Error())
			return false
		}
		configuration = ca.Configuration
	}

	planned := plannedKeySource(plan.KeySource)
	actual := configuration.KeySource
	if actual == "" {
		actual = CA_KEY_SOURCE_INFISICAL
	}
	if actual == planned {
		fillUnknownKeySourceFromAPI(configuration, plan)
		return true
	}

	unsupported := fmt.Sprintf("Infisical created the CA with key source '%s' instead of '%s'.", actual, planned)
	fallback := "set key_source to match what Infisical supports."
	if actual == CA_KEY_SOURCE_INFISICAL {
		unsupported += fmt.Sprintf(" Infisical versions older than %s ignore key_source.", CA_KEY_SOURCE_MIN_VERSIONS[planned])
		fallback = fmt.Sprintf("upgrade Infisical, or remove %s from the configuration.", CA_KEY_SOURCE_ATTRIBUTES[planned])
	}

	if _, err := r.client.DeleteInternalCA(infisical.DeleteCARequest{CAId: created.Id}); err != nil {
		plan.Id = types.StringValue(created.Id)
		plan.Status = types.StringValue(created.Status)
		certManagerInternalCAKeySourceFromAPI(configuration, plan)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Error creating CA", fmt.Sprintf(
			"%s Deleting it again failed: %s. Destroy the CA, then %s", unsupported, err.Error(), fallback))
		return false
	}

	resp.Diagnostics.AddError("Error creating CA", fmt.Sprintf("%s The CA was deleted again. To continue, %s", unsupported, fallback))
	return false
}
