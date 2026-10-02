package resource

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"
	infisicalstrings "terraform-provider-infisical/internal/pkg/strings"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &IdentityKubernetesAuthTemplateResource{}
	_ resource.ResourceWithConfigure      = &IdentityKubernetesAuthTemplateResource{}
	_ resource.ResourceWithImportState    = &IdentityKubernetesAuthTemplateResource{}
	_ resource.ResourceWithValidateConfig = &IdentityKubernetesAuthTemplateResource{}
	_ resource.ResourceWithModifyPlan     = &IdentityKubernetesAuthTemplateResource{}
)

// The API trims template names, so surrounding whitespace would diff forever.
var identityAuthTemplateNameValidators = []validator.String{
	stringvalidator.LengthBetween(1, 64),
	stringvalidator.RegexMatches(
		regexp.MustCompile(`^\S(?:.*\S)?$`),
		"must not begin or end with whitespace",
	),
}

func NewIdentityKubernetesAuthTemplateResource() resource.Resource {
	return &IdentityKubernetesAuthTemplateResource{}
}

type IdentityKubernetesAuthTemplateResource struct {
	client *infisical.Client
}

type IdentityKubernetesAuthTemplateResourceModel struct {
	ID                   types.String                   `tfsdk:"id"`
	Name                 types.String                   `tfsdk:"name"`
	TokenReviewerMode    types.String                   `tfsdk:"token_reviewer_mode"`
	KubernetesHost       types.String                   `tfsdk:"kubernetes_host"`
	CaCertificate        customtypes.TrimmedStringValue `tfsdk:"kubernetes_ca_certificate"`
	VerifyTlsCertificate types.Bool                     `tfsdk:"verify_tls_certificate"`
	TokenReviewerJWT     types.String                   `tfsdk:"token_reviewer_jwt"`
	HasTokenReviewerJWT  types.Bool                     `tfsdk:"has_token_reviewer_jwt"`
	GatewayID            types.String                   `tfsdk:"gateway_id"`
	GatewayPoolID        types.String                   `tfsdk:"gateway_pool_id"`
	AllowedAudience      types.String                   `tfsdk:"allowed_audience"`
}

func (r *IdentityKubernetesAuthTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_kubernetes_auth_template"
}

func (r *IdentityKubernetesAuthTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage a Kubernetes auth template in Infisical. A template holds the Kubernetes connection settings once, so many machine identities can share them by setting `template_id` on `infisical_identity_kubernetes_auth`. Editing a template propagates its settings to every identity linked to it. Auth templates are an Enterprise feature, and only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the auth template.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the auth template, at most 64 characters.",
				Required:    true,
				Validators:  identityAuthTemplateNameValidators,
			},
			"token_reviewer_mode": schema.StringAttribute{
				Description: "Who performs the TokenReview. `api` means Infisical calls `kubernetes_host` directly. `gateway` means the TokenReview runs through `gateway_id` or `gateway_pool_id`. Defaults to `api`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(TOKEN_REVIEWER_MODE_API),
				Validators: []validator.String{
					stringvalidator.OneOf(TOKEN_REVIEWER_MODE_API, TOKEN_REVIEWER_MODE_GATEWAY),
				},
			},
			"kubernetes_host": schema.StringAttribute{
				Description: "The host string, host:port pair, or URL to the base of the Kubernetes API server. Required when `token_reviewer_mode` is `api`, and must be omitted when it is `gateway`.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"kubernetes_ca_certificate": schema.StringAttribute{
				// The API strips the trailing newline that file() always adds.
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The PEM-encoded CA certificate used to validate the Kubernetes API server's TLS certificate. Must be omitted when `token_reviewer_mode` is `gateway`.",
				Optional:    true,
			},
			"verify_tls_certificate": schema.BoolAttribute{
				Description: "Whether to verify the Kubernetes API server's TLS certificate against `kubernetes_ca_certificate`. Defaults to true when a CA certificate is set, and false otherwise. In `api` mode, true requires a CA certificate and false forbids one. In `gateway` mode it cannot be true.",
				Optional:    true,
				Computed:    true,
			},
			"token_reviewer_jwt": schema.StringAttribute{
				Description: "A long-lived service account JWT that Infisical uses to call the TokenReview API. If omitted, each identity's own service account token reviews itself. Must be omitted when `token_reviewer_mode` is `gateway`. Write-only: Infisical never returns it, so Terraform cannot detect a change made outside this configuration, and an imported template leaves it empty.",
				Optional:    true,
				Sensitive:   true,
			},
			"has_token_reviewer_jwt": schema.BoolAttribute{
				Description: "Whether Infisical holds a token reviewer JWT for this template. The JWT itself is never returned, so this is the only way to tell a stored one apart from none.",
				Computed:    true,
			},
			"gateway_id": schema.StringAttribute{
				Description: "The ID of the gateway to route Kubernetes API requests through. Mutually exclusive with `gateway_pool_id`.",
				Optional:    true,
				Validators: []validator.String{
					infisicaltf.UuidValidator,
					stringvalidator.ConflictsWith(path.MatchRoot("gateway_pool_id")),
				},
			},
			"gateway_pool_id": schema.StringAttribute{
				Description: "The ID of the gateway pool to route Kubernetes API requests through. Mutually exclusive with `gateway_id`.",
				Optional:    true,
				Validators:  []validator.String{infisicaltf.UuidValidator},
			},
			"allowed_audience": schema.StringAttribute{
				Description: "The audience claim that service account JWTs must carry to authenticate. Leave empty to skip the audience check.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
		},
	}
}

func (r *IdentityKubernetesAuthTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Mirrors the API's connection rules so a bad combination fails at plan time, not apply.
func (r *IdentityKubernetesAuthTemplateResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config IdentityKubernetesAuthTemplateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An unknown mode resolves at apply time, and the API enforces the same rules then.
	if config.TokenReviewerMode.IsUnknown() {
		return
	}
	mode := config.TokenReviewerMode.ValueString()
	if config.TokenReviewerMode.IsNull() {
		mode = TOKEN_REVIEWER_MODE_API
	}

	switch mode {
	case TOKEN_REVIEWER_MODE_API:
		if !isStringConfigured(config.KubernetesHost) {
			resp.Diagnostics.AddAttributeError(
				path.Root("kubernetes_host"),
				"Kubernetes host is required",
				`kubernetes_host is required when token_reviewer_mode is "api".`,
			)
		}

		if config.VerifyTlsCertificate.IsUnknown() || config.VerifyTlsCertificate.IsNull() || config.CaCertificate.IsUnknown() {
			return
		}
		hasCaCertificate := strings.TrimSpace(config.CaCertificate.ValueString()) != ""
		if config.VerifyTlsCertificate.ValueBool() && !hasCaCertificate {
			resp.Diagnostics.AddAttributeError(
				path.Root("verify_tls_certificate"),
				"CA certificate is required",
				"verify_tls_certificate = true needs kubernetes_ca_certificate to verify the API server against. Set the CA certificate, or set verify_tls_certificate = false.",
			)
		}
		if !config.VerifyTlsCertificate.ValueBool() && hasCaCertificate {
			resp.Diagnostics.AddAttributeError(
				path.Root("verify_tls_certificate"),
				"TLS verification cannot be disabled with a CA certificate",
				"verify_tls_certificate = false conflicts with kubernetes_ca_certificate. Remove the CA certificate, or enable verification.",
			)
		}

	case TOKEN_REVIEWER_MODE_GATEWAY:
		if !isStringConfigured(config.GatewayID) && !isStringConfigured(config.GatewayPoolID) {
			resp.Diagnostics.AddAttributeError(
				path.Root("gateway_id"),
				"Gateway is required",
				`token_reviewer_mode "gateway" needs gateway_id or gateway_pool_id.`,
			)
		}

		// The gateway reviews tokens with its own in-cluster service account, so these would be
		// stored and copied to every linked identity without ever being used. The JWT is a
		// credential, which makes spreading an unused copy worse than useless.
		hasKnownString := func(value types.String) bool {
			return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
		}
		hasCaCertificate := !config.CaCertificate.IsNull() && !config.CaCertificate.IsUnknown() &&
			strings.TrimSpace(config.CaCertificate.ValueString()) != ""
		for _, unused := range []struct {
			attribute  string
			configured bool
		}{
			{"kubernetes_host", hasKnownString(config.KubernetesHost)},
			{"kubernetes_ca_certificate", hasCaCertificate},
			{"token_reviewer_jwt", hasKnownString(config.TokenReviewerJWT)},
			{"verify_tls_certificate", !config.VerifyTlsCertificate.IsNull() && !config.VerifyTlsCertificate.IsUnknown() && config.VerifyTlsCertificate.ValueBool()},
		} {
			if unused.configured {
				resp.Diagnostics.AddAttributeError(
					path.Root(unused.attribute),
					"Not used in gateway mode",
					unused.attribute+` has no effect when token_reviewer_mode is "gateway", because the gateway reviews tokens with its own service account. Remove it, or use token_reviewer_mode "api".`,
				)
			}
		}
	}
}

// Computes what the API will derive, so the plan shows real values instead of "known after apply".
func (r *IdentityKubernetesAuthTemplateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, config IdentityKubernetesAuthTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state *IdentityKubernetesAuthTemplateResourceModel
	if !req.State.Raw.IsNull() {
		state = &IdentityKubernetesAuthTemplateResourceModel{}
		resp.Diagnostics.Append(req.State.Get(ctx, state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// The API defaults verification to whether a CA certificate is set.
	if config.VerifyTlsCertificate.IsNull() {
		if plan.CaCertificate.IsUnknown() {
			plan.VerifyTlsCertificate = types.BoolUnknown()
		} else {
			plan.VerifyTlsCertificate = types.BoolValue(strings.TrimSpace(plan.CaCertificate.ValueString()) != "")
		}
	}

	plan.HasTokenReviewerJWT = plannedHasTokenReviewerJwt(plan.TokenReviewerJWT, state)

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// The JWT is sent only when it differs from state, so an unchanged or imported template keeps
// whatever JWT Infisical already holds.
func plannedHasTokenReviewerJwt(plannedJwt types.String, state *IdentityKubernetesAuthTemplateResourceModel) types.Bool {
	if plannedJwt.IsUnknown() {
		return types.BoolUnknown()
	}
	if state == nil || !plannedJwt.Equal(state.TokenReviewerJWT) {
		return types.BoolValue(plannedJwt.ValueString() != "")
	}
	return state.HasTokenReviewerJWT
}

// Unknown means set to something that resolves at apply, so it must not count as absent.
func isStringConfigured(value types.String) bool {
	return !value.IsNull() && (value.IsUnknown() || value.ValueString() != "")
}

func stringPtrIfSet(value types.String) *string {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	return infisicalstrings.StringToPtr(value.ValueString())
}

// An empty API value keeps a configured "" rather than flipping it to null, which would diff.
func stringFromOptionalAPIValue(prior types.String, value *string) types.String {
	if value == nil || *value == "" {
		if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
			return types.StringValue("")
		}
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func trimmedStringFromOptionalAPIValue(prior customtypes.TrimmedStringValue, value string) customtypes.TrimmedStringValue {
	if value == "" && (prior.IsNull() || prior.IsUnknown()) {
		return customtypes.NewTrimmedStringNull()
	}
	return customtypes.NewTrimmedStringValue(value)
}

func applyKubernetesAuthTemplateToModel(model *IdentityKubernetesAuthTemplateResourceModel, template infisical.IdentityKubernetesAuthTemplate) {
	fields := template.TemplateFields

	model.ID = types.StringValue(template.ID)
	model.Name = types.StringValue(template.Name)
	model.TokenReviewerMode = types.StringValue(fields.TokenReviewMode)
	model.KubernetesHost = stringFromOptionalAPIValue(model.KubernetesHost, fields.KubernetesHost)
	model.CaCertificate = trimmedStringFromOptionalAPIValue(model.CaCertificate, fields.CaCert)
	if fields.VerifyTlsCertificate != nil {
		model.VerifyTlsCertificate = types.BoolValue(*fields.VerifyTlsCertificate)
	} else {
		model.VerifyTlsCertificate = types.BoolValue(fields.CaCert != "")
	}
	model.GatewayID = stringFromOptionalAPIValue(model.GatewayID, fields.GatewayID)
	model.GatewayPoolID = stringFromOptionalAPIValue(model.GatewayPoolID, fields.GatewayPoolID)
	model.AllowedAudience = types.StringValue(fields.AllowedAudience)
	model.HasTokenReviewerJWT = types.BoolValue(fields.HasTokenReviewerJwt)

	// A JWT cleared outside Terraform is dropped from state so the next plan puts it back. The
	// reverse cannot be detected: a JWT in Infisical but not in state is kept as is.
	if !fields.HasTokenReviewerJwt && !model.TokenReviewerJWT.IsNull() && !model.TokenReviewerJWT.IsUnknown() && model.TokenReviewerJWT.ValueString() != "" {
		model.TokenReviewerJWT = types.StringNull()
	}
}

func (r *IdentityKubernetesAuthTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity kubernetes auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan IdentityKubernetesAuthTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var verifyTlsCertificate *bool
	if !plan.VerifyTlsCertificate.IsNull() && !plan.VerifyTlsCertificate.IsUnknown() {
		verify := plan.VerifyTlsCertificate.ValueBool()
		verifyTlsCertificate = &verify
	}

	template, err := r.client.CreateIdentityKubernetesAuthTemplate(infisical.CreateIdentityKubernetesAuthTemplateRequest{
		Name: plan.Name.ValueString(),
		TemplateFields: infisical.CreateIdentityKubernetesAuthTemplateFields{
			TokenReviewMode:      plan.TokenReviewerMode.ValueString(),
			KubernetesHost:       stringPtrIfSet(plan.KubernetesHost),
			CaCert:               strings.TrimSpace(plan.CaCertificate.ValueString()),
			VerifyTlsCertificate: verifyTlsCertificate,
			TokenReviewerJwt:     plan.TokenReviewerJWT.ValueString(),
			GatewayID:            stringPtrIfSet(plan.GatewayID),
			GatewayPoolID:        stringPtrIfSet(plan.GatewayPoolID),
			AllowedAudience:      plan.AllowedAudience.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity kubernetes auth template",
			"Couldn't create kubernetes auth template in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	applyKubernetesAuthTemplateToModel(&plan, template)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityKubernetesAuthTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read identity kubernetes auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityKubernetesAuthTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.GetIdentityKubernetesAuthTemplate(state.ID.ValueString())
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading identity kubernetes auth template",
			"Couldn't read kubernetes auth template with ID "+state.ID.ValueString()+" from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	applyKubernetesAuthTemplateToModel(&state, template)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Sends only what changed: the API merges the patch into the stored fields and propagates it to
// every linked identity, and re-authorizes gateway access whenever a gateway field is present.
func kubernetesAuthTemplateFieldsPatch(plan, state IdentityKubernetesAuthTemplateResourceModel) map[string]any {
	fields := map[string]any{}

	if !plan.TokenReviewerMode.Equal(state.TokenReviewerMode) {
		fields["tokenReviewMode"] = plan.TokenReviewerMode.ValueString()
	}
	if !plan.KubernetesHost.Equal(state.KubernetesHost) {
		fields["kubernetesHost"] = stringPtrIfSet(plan.KubernetesHost)
	}

	// The API re-derives verification from the CA certificate whenever the certificate is patched
	// alone, so the planned verification always travels with it.
	caCertificateChanged := strings.TrimSpace(plan.CaCertificate.ValueString()) != strings.TrimSpace(state.CaCertificate.ValueString())
	if caCertificateChanged {
		fields["caCert"] = strings.TrimSpace(plan.CaCertificate.ValueString())
	}
	if caCertificateChanged || !plan.VerifyTlsCertificate.Equal(state.VerifyTlsCertificate) {
		fields["verifyTlsCertificate"] = plan.VerifyTlsCertificate.ValueBool()
	}

	// An empty string clears the stored JWT; leaving the key out keeps it.
	if !plan.TokenReviewerJWT.Equal(state.TokenReviewerJWT) {
		fields["tokenReviewerJwt"] = plan.TokenReviewerJWT.ValueString()
	}

	// Sent as a pair so moving from a gateway to a pool clears the other side in the same merge.
	if !plan.GatewayID.Equal(state.GatewayID) || !plan.GatewayPoolID.Equal(state.GatewayPoolID) {
		fields["gatewayId"] = stringPtrIfSet(plan.GatewayID)
		fields["gatewayPoolId"] = stringPtrIfSet(plan.GatewayPoolID)
	}

	if !plan.AllowedAudience.Equal(state.AllowedAudience) {
		fields["allowedAudience"] = plan.AllowedAudience.ValueString()
	}

	return fields
}

func (r *IdentityKubernetesAuthTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity kubernetes auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan, state IdentityKubernetesAuthTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := infisical.UpdateIdentityAuthTemplateRequest{
		ID:             state.ID.ValueString(),
		TemplateFields: kubernetesAuthTemplateFieldsPatch(plan, state),
	}
	if !plan.Name.Equal(state.Name) {
		updateRequest.Name = infisicalstrings.StringToPtr(plan.Name.ValueString())
	}

	// A plan can differ only in spelling the API treats as equal, such as a trailing newline or
	// slash. The API rejects an empty update, so such a plan just refreshes from Infisical.
	var template infisical.IdentityKubernetesAuthTemplate
	var err error
	if updateRequest.Name == nil && len(updateRequest.TemplateFields) == 0 {
		template, err = r.client.GetIdentityKubernetesAuthTemplate(updateRequest.ID)
	} else {
		template, err = r.client.UpdateIdentityKubernetesAuthTemplate(updateRequest)
	}
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Identity kubernetes auth template not found",
				"Kubernetes auth template with ID "+state.ID.ValueString()+" no longer exists in Infisical. Run `terraform apply -refresh-only` to reconcile state, then apply again.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating identity kubernetes auth template",
			"Couldn't update kubernetes auth template in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	applyKubernetesAuthTemplateToModel(&plan, template)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityKubernetesAuthTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity kubernetes auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityKubernetesAuthTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Linked identities are unlinked by the API and keep the settings copied onto them.
	err := r.client.DeleteIdentityAuthTemplate(state.ID.ValueString())
	if err != nil && !errors.Is(err, infisical.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Error deleting identity kubernetes auth template",
			"Couldn't delete kubernetes auth template from Infisical, unexpected error: "+err.Error(),
		)
	}
}

func (r *IdentityKubernetesAuthTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import identity kubernetes auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
