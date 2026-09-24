package resource

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"
	infisicalstrings "terraform-provider-infisical/internal/pkg/strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                     = &GatewayResource{}
	_ resource.ResourceWithConfigure        = &GatewayResource{}
	_ resource.ResourceWithImportState      = &GatewayResource{}
	_ resource.ResourceWithConfigValidators = &GatewayResource{}
	_ resource.ResourceWithValidateConfig   = &GatewayResource{}
)

// Mirrors the API's slug rule so a bad name fails at plan time, not apply.
var gatewayNameSlugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func NewGatewayResource() resource.Resource {
	return &GatewayResource{}
}

type GatewayResource struct {
	client *infisical.Client
}

type gatewayAwsAuthModel struct {
	AllowedPrincipalArns types.Set `tfsdk:"allowed_principal_arns"`
	AllowedAccountIds    types.Set `tfsdk:"allowed_account_ids"`
}

type gatewayGcpAuthModel struct {
	Type                   types.String `tfsdk:"type"`
	AllowedServiceAccounts types.Set    `tfsdk:"allowed_service_accounts"`
	AllowedProjects        types.Set    `tfsdk:"allowed_projects"`
	AllowedZones           types.Set    `tfsdk:"allowed_zones"`
}

type gatewayKubernetesAuthModel struct {
	TokenReviewMode           types.String                    `tfsdk:"token_review_mode"`
	KubernetesHost            customtypes.KubernetesHostValue `tfsdk:"kubernetes_host"`
	CaCertificate             customtypes.TrimmedStringValue  `tfsdk:"ca_certificate"`
	TokenReviewerJwt          types.String                    `tfsdk:"token_reviewer_jwt"`
	ReviewerGatewayID         types.String                    `tfsdk:"reviewer_gateway_id"`
	ReviewerGatewayPoolID     types.String                    `tfsdk:"reviewer_gateway_pool_id"`
	AllowedNamespaces         types.Set                       `tfsdk:"allowed_namespaces"`
	AllowedServiceAccountName types.Set                       `tfsdk:"allowed_service_account_names"`
	AllowedAudience           types.String                    `tfsdk:"allowed_audience"`
	VerifyTlsCertificate      types.Bool                      `tfsdk:"verify_tls_certificate"`
	HasTokenReviewerJwt       types.Bool                      `tfsdk:"has_token_reviewer_jwt"`
}

type gatewayTokenAuthModel struct{}

type GatewayResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	// Objects rather than pointers: a pointer has no way to say "unknown", which is what an
	// auth block is when the expression choosing it reads a value that only exists after apply.
	AwsAuth        types.Object `tfsdk:"aws_auth"`
	GcpAuth        types.Object `tfsdk:"gcp_auth"`
	KubernetesAuth types.Object `tfsdk:"kubernetes_auth"`
	TokenAuth      types.Object `tfsdk:"token_auth"`
}

func (r *GatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway"
}

func (r *GatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage a gateway in Infisical. This resource is the gateway's record and its auth method, not the machine that runs it: deploy that separately with the Helm chart or the CLI, pointing it at this resource's `id`. Exactly one auth method block must be set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the gateway.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the gateway. Unique within the organization. Renaming is an in-place update, so the gateway keeps its ID and anything referencing it keeps working.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(
						gatewayNameSlugRegex,
						"must be lowercase letters and numbers separated by single hyphens (e.g. 'prod-us-east')",
					),
				},
			},

			"aws_auth":        gatewayAwsAuthSchema(),
			"gcp_auth":        gatewayGcpAuthSchema(),
			"kubernetes_auth": gatewayKubernetesAuthSchema(),
			"token_auth":      gatewayTokenAuthSchema(),
		},
	}
}

// Derived from the schema so the two cannot drift.
func authBlockAttrTypes(block schema.SingleNestedAttribute) map[string]attr.Type {
	attrTypes := make(map[string]attr.Type, len(block.Attributes))
	for name, attribute := range block.Attributes {
		attrTypes[name] = attribute.GetType()
	}
	return attrTypes
}

// An unknown block decodes to nil: nothing can be read off it yet, so callers treat it the
// way they treat an absent one.
func decodeAuthBlock[T any](ctx context.Context, block types.Object, diags *diag.Diagnostics) *T {
	if block.IsNull() || block.IsUnknown() {
		return nil
	}

	var decoded T
	diags.Append(block.As(ctx, &decoded, basetypes.ObjectAsOptions{})...)
	return &decoded
}

func encodeAuthBlock(ctx context.Context, blockSchema schema.SingleNestedAttribute, value any, diags *diag.Diagnostics) types.Object {
	object, d := types.ObjectValueFrom(ctx, authBlockAttrTypes(blockSchema), value)
	diags.Append(d...)
	return object
}

func nullAuthBlocks(model *GatewayResourceModel) {
	model.AwsAuth = types.ObjectNull(authBlockAttrTypes(gatewayAwsAuthSchema()))
	model.GcpAuth = types.ObjectNull(authBlockAttrTypes(gatewayGcpAuthSchema()))
	model.KubernetesAuth = types.ObjectNull(authBlockAttrTypes(gatewayKubernetesAuthSchema()))
	model.TokenAuth = types.ObjectNull(authBlockAttrTypes(gatewayTokenAuthSchema()))
}

type kubernetesHostValidator struct{}

func (kubernetesHostValidator) Description(_ context.Context) string {
	return "must be an https API server address with no path, query or credentials"
}

func (v kubernetesHostValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (kubernetesHostValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if err := customtypes.ValidateKubernetesHost(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Kubernetes host",
			fmt.Sprintf("Kubernetes host %s.", err.Error()),
		)
	}
}

// Allowlists cross the wire as one comma-separated string, so a value carrying a comma would
// split into two entries and one carrying surrounding whitespace would come back trimmed.
func csvSafeEntries() validator.Set {
	return setvalidator.ValueStringsAre(
		stringvalidator.NoneOf(""),
		stringvalidator.RegexMatches(
			regexp.MustCompile(`^[^,\s](?:[^,]*[^,\s])?$`),
			"must not contain a comma or begin or end with whitespace",
		),
	)
}

func gatewayAwsAuthSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Authenticate the gateway with its AWS IAM identity. The machine re-authenticates on every start, so no secret is stored. At least one of `allowed_principal_arns` or `allowed_account_ids` must be non-empty.",
		Attributes: map[string]schema.Attribute{
			"allowed_principal_arns": schema.SetAttribute{
				Description: "IAM principal ARNs allowed to authenticate as this gateway. Supports `*` wildcards.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  []validator.Set{csvSafeEntries()},
			},
			"allowed_account_ids": schema.SetAttribute{
				Description: "AWS account IDs allowed to authenticate as this gateway.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  []validator.Set{csvSafeEntries()},
			},
		},
	}
}

func gatewayGcpAuthSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Authenticate the gateway with its GCP identity. The machine re-authenticates on every start, so no secret is stored. At least one of `allowed_service_accounts` or `allowed_projects` must be non-empty.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Description: "How the gateway proves its identity. `gce` verifies an ID token from the instance metadata server, which covers Compute Engine VMs and GKE workload identity. `iam` verifies a JWT the service account signed through the IAM Credentials API, for hosts outside Compute Engine. Defaults to `gce`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(infisical.GatewayGcpAuthTypeGce),
				Validators: []validator.String{
					stringvalidator.OneOf(infisical.GatewayGcpAuthTypeGce, infisical.GatewayGcpAuthTypeIam),
				},
			},
			"allowed_service_accounts": schema.SetAttribute{
				Description: "GCP service account emails allowed to authenticate as this gateway.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  []validator.Set{csvSafeEntries()},
			},
			"allowed_projects": schema.SetAttribute{
				Description: "GCP project IDs whose Compute Engine instances are allowed to authenticate as this gateway. Only applies when `type` is `gce`.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  []validator.Set{csvSafeEntries()},
			},
			"allowed_zones": schema.SetAttribute{
				Description: "GCP zones whose Compute Engine instances are allowed to authenticate as this gateway. Only applies when `type` is `gce`.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  []validator.Set{csvSafeEntries()},
			},
		},
	}
}

func gatewayKubernetesAuthSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Authenticate the gateway with its Kubernetes service account token. The pod re-authenticates on every start, so no secret is stored. At least one of `allowed_namespaces` or `allowed_service_account_names` must be non-empty.",
		Attributes: map[string]schema.Attribute{
			"token_review_mode": schema.StringAttribute{
				Description: "Who performs the TokenReview. `api` means Infisical does, calling `kubernetes_host` directly. `gateway` means another already-connected gateway does it with its own in-cluster service account, which needs no host or reviewer token. Defaults to `api`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(infisical.GatewayKubernetesTokenReviewModeApi),
				Validators: []validator.String{
					stringvalidator.OneOf(infisical.GatewayKubernetesTokenReviewModeApi, infisical.GatewayKubernetesTokenReviewModeGateway),
				},
			},
			"kubernetes_host": schema.StringAttribute{
				// The API normalizes to https://host[:port], so a trailing slash would diff forever.
				CustomType:  customtypes.KubernetesHostType{},
				Description: "The URL of the Kubernetes API server, for example https://my-cluster.example.com:6443. Must be https with no path, and reachable from Infisical over the public internet. Required unless `token_review_mode` is `gateway`, where it must be omitted.",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{kubernetesHostValidator{}},
			},
			"ca_certificate": schema.StringAttribute{
				// The API strips the trailing newline that file() always adds.
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The PEM-encoded CA certificate that issued the Kubernetes API server's TLS certificate.",
				Optional:    true,
			},
			"token_reviewer_jwt": schema.StringAttribute{
				Description: "A long-lived service account token with the system:auth-delegator ClusterRole, used to submit TokenReview requests. Write-only: Infisical never returns it, so Terraform cannot detect a change made outside this configuration, and an imported gateway leaves it empty.",
				Optional:    true,
				Sensitive:   true,
			},
			"reviewer_gateway_id": schema.StringAttribute{
				Description: "The gateway that performs the TokenReview. Required when `token_review_mode` is `gateway`, and must be a different gateway that is already connected in the cluster. Mutually exclusive with `reviewer_gateway_pool_id`.",
				Optional:    true,
			},
			"reviewer_gateway_pool_id": schema.StringAttribute{
				Description: "The gateway pool to route TokenReview traffic through. Mutually exclusive with `reviewer_gateway_id`, and rejected when `token_review_mode` is `gateway`.",
				Optional:    true,
			},
			"allowed_namespaces": schema.SetAttribute{
				Description: "Kubernetes namespaces whose service accounts are allowed to authenticate as this gateway. Supports `*` wildcards.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  []validator.Set{csvSafeEntries()},
			},
			"allowed_service_account_names": schema.SetAttribute{
				Description: "Kubernetes service account names allowed to authenticate as this gateway. Supports `*` wildcards.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  []validator.Set{csvSafeEntries()},
			},
			"allowed_audience": schema.StringAttribute{
				Description: "The audience the service account token must carry. Leave empty to skip the audience check.",
				Optional:    true,
			},
			"has_token_reviewer_jwt": schema.BoolAttribute{
				Description: "Whether Infisical holds a token reviewer JWT for this gateway. The JWT itself is never returned, so this is the only way to tell a stored one apart from none.",
				Computed:    true,
			},
			"verify_tls_certificate": schema.BoolAttribute{
				Description: "Whether to verify the Kubernetes API server's TLS certificate. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func gatewayTokenAuthSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Authenticate the gateway with a one-time enrollment token. The token itself is minted by a separate `infisical_gateway_enrollment_token` resource. This block takes no arguments, because token auth has nothing to configure: write `token_auth = {}`. Prefer `aws_auth`, `gcp_auth` or `kubernetes_auth` where the platform can vouch for the machine, since those re-authenticate on every start and put no secret in state.",
		Attributes:  map[string]schema.Attribute{},
	}
}

func (r *GatewayResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("aws_auth"),
			path.MatchRoot("gcp_auth"),
			path.MatchRoot("kubernetes_auth"),
			path.MatchRoot("token_auth"),
		),
	}
}

// resourcevalidator cannot express these: its combinators fire on blocks the config never set.
func (r *GatewayResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config GatewayResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Unknown resolves at apply time, so it must not count as empty.
	isEmpty := func(set types.Set) bool { return !set.IsUnknown() && len(set.Elements()) == 0 }

	// A block that is still unknown holds nothing to check yet, so it decodes to nil here and
	// the API enforces these same rules at apply time.
	awsAuth := decodeAuthBlock[gatewayAwsAuthModel](ctx, config.AwsAuth, &resp.Diagnostics)
	gcpAuth := decodeAuthBlock[gatewayGcpAuthModel](ctx, config.GcpAuth, &resp.Diagnostics)
	kubernetesAuth := decodeAuthBlock[gatewayKubernetesAuthModel](ctx, config.KubernetesAuth, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	switch {
	case awsAuth != nil:
		if isEmpty(awsAuth.AllowedPrincipalArns) && isEmpty(awsAuth.AllowedAccountIds) {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_auth"),
				"No AWS allowlist configured",
				"Set allowed_principal_arns or allowed_account_ids. Without one, any AWS caller could authenticate as this gateway.",
			)
		}

	case gcpAuth != nil:
		if isEmpty(gcpAuth.AllowedServiceAccounts) && isEmpty(gcpAuth.AllowedProjects) {
			resp.Diagnostics.AddAttributeError(
				path.Root("gcp_auth"),
				"No GCP allowlist configured",
				"Set allowed_service_accounts or allowed_projects. A zone on its own restricts nothing, because anyone can create an instance in a given zone.",
			)
		}

		if gcpAuth.Type.ValueString() == infisical.GatewayGcpAuthTypeIam {
			if !isEmpty(gcpAuth.AllowedProjects) || !isEmpty(gcpAuth.AllowedZones) {
				resp.Diagnostics.AddAttributeError(
					path.Root("gcp_auth"),
					"Projects and zones do not apply to the iam type",
					"allowed_projects and allowed_zones only narrow a Compute Engine token. Restrict an iam service account token with allowed_service_accounts instead.",
				)
			}
		}

	case kubernetesAuth != nil:
		if isEmpty(kubernetesAuth.AllowedNamespaces) && isEmpty(kubernetesAuth.AllowedServiceAccountName) {
			resp.Diagnostics.AddAttributeError(
				path.Root("kubernetes_auth"),
				"No Kubernetes allowlist configured",
				"Set allowed_namespaces or allowed_service_account_names. Without one, any service account in the cluster could authenticate as this gateway.",
			)
		}

		// Unknown means set to something that resolves at apply, such as another gateway
		// created in the same run. Reading it as absent would reject that configuration.
		isConfigured := func(value types.String) bool {
			return !value.IsNull() && (value.IsUnknown() || value.ValueString() != "")
		}
		hasReviewerGateway := isConfigured(kubernetesAuth.ReviewerGatewayID)
		hasReviewerPool := isConfigured(kubernetesAuth.ReviewerGatewayPoolID)

		if hasReviewerGateway && hasReviewerPool {
			resp.Diagnostics.AddAttributeError(
				path.Root("kubernetes_auth"),
				"Two TokenReview reviewers configured",
				"Set reviewer_gateway_id or reviewer_gateway_pool_id, not both. A pool picks any healthy member, so naming a gateway as well has no meaning.",
			)
		}

		if kubernetesAuth.TokenReviewMode.ValueString() == infisical.GatewayKubernetesTokenReviewModeGateway {
			if !hasReviewerGateway {
				resp.Diagnostics.AddAttributeError(
					path.Root("kubernetes_auth"),
					"No reviewer gateway configured",
					`token_review_mode "gateway" needs reviewer_gateway_id, naming a different gateway already connected in the cluster that will run the TokenReview.`,
				)
			}
			if hasReviewerPool {
				resp.Diagnostics.AddAttributeError(
					path.Root("kubernetes_auth"),
					"A pool cannot review tokens",
					`token_review_mode "gateway" needs a specific reviewer_gateway_id, because the reviewer decides the outcome and pool membership can change after the config is saved. Use token_review_mode "api" to route through a pool.`,
				)
			}
		}
	}
}

// The API discards a stored reviewer JWT when the destination moves, since a token validated
// against one address must not be sent to another. That is only safe if the operator can see it
// coming, and a write-only attribute cannot show it in the plan, so refuse instead.
func (r *GatewayResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan GatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stored := decodeAuthBlock[gatewayKubernetesAuthModel](ctx, state.KubernetesAuth, &resp.Diagnostics)
	planned := decodeAuthBlock[gatewayKubernetesAuthModel](ctx, plan.KubernetesAuth, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || stored == nil || planned == nil {
		return
	}

	// Only bites when the configuration cannot resupply the token, so an operator who keeps it in
	// Terraform is never stopped.
	if !stored.HasTokenReviewerJwt.ValueBool() || !planned.TokenReviewerJwt.IsNull() {
		return
	}

	// Everything the API counts as moving the destination, each judged on its own. An unknown
	// value may resolve to what is already stored, so it defers rather than refusing, but a
	// field already known to differ still moves the destination whatever its neighbours do.
	// The host is normalized and the CA trimmed, matching how the API compares them.
	moved := false
	differs := func(planned, stored attr.Value, changed func() bool) {
		if !planned.IsUnknown() && !stored.IsUnknown() && changed() {
			moved = true
		}
	}
	differs(planned.KubernetesHost, stored.KubernetesHost, func() bool {
		return customtypes.NormalizeKubernetesHost(planned.KubernetesHost.ValueString()) !=
			customtypes.NormalizeKubernetesHost(stored.KubernetesHost.ValueString())
	})
	differs(planned.CaCertificate, stored.CaCertificate, func() bool {
		return strings.TrimSpace(planned.CaCertificate.ValueString()) != strings.TrimSpace(stored.CaCertificate.ValueString())
	})
	differs(planned.TokenReviewMode, stored.TokenReviewMode, func() bool {
		return !planned.TokenReviewMode.Equal(stored.TokenReviewMode)
	})
	differs(planned.ReviewerGatewayID, stored.ReviewerGatewayID, func() bool {
		return !planned.ReviewerGatewayID.Equal(stored.ReviewerGatewayID)
	})
	differs(planned.ReviewerGatewayPoolID, stored.ReviewerGatewayPoolID, func() bool {
		return !planned.ReviewerGatewayPoolID.Equal(stored.ReviewerGatewayPoolID)
	})
	differs(planned.VerifyTlsCertificate, stored.VerifyTlsCertificate, func() bool {
		return !planned.VerifyTlsCertificate.Equal(stored.VerifyTlsCertificate)
	})
	if !moved {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("kubernetes_auth"),
		"This change would discard the stored token reviewer JWT",
		"Infisical holds a token reviewer JWT for this gateway that it never returns, so Terraform cannot put it back. "+
			"Moving the host, review mode, reviewer gateway, CA certificate or TLS verification makes Infisical drop it, "+
			"and the gateway stops being able to authenticate.\n\n"+
			"Set token_reviewer_jwt to the token you want used from now on, or to \"\" to drop it on purpose and let the "+
			"gateway's own service account review its tokens.",
	)
}

func (r *GatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create gateway",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan GatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	authMethod, diags := gatewayAuthMethodInput(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	gateway, err := r.client.CreateGateway(infisical.CreateGatewayRequest{
		Name:       plan.Name.ValueString(),
		AuthMethod: authMethod,
	})
	if err != nil {
		var alreadyExists *infisical.GatewayAlreadyExistsError
		if errors.As(err, &alreadyExists) {
			resp.Diagnostics.AddError(
				"Gateway already exists",
				fmt.Sprintf(
					"Infisical already has a gateway named %q, and names are unique within an organization. "+
						"Either import it with `terraform import <this resource's address> %s`, or delete it in Infisical and apply again. "+
						"An earlier apply that created the gateway but failed before recording it leaves exactly this behind. "+
						"The full error was: %s",
					plan.Name.ValueString(), alreadyExists.ExistingGatewayID, err.Error(),
				),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error creating gateway",
			"Couldn't create gateway in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.applyGatewayToModel(ctx, &plan, gateway)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read gateway",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gateway, err := r.client.GetGatewayById(state.ID.ValueString())
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading gateway",
			"Couldn't read gateway with ID "+state.ID.ValueString()+" from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.applyGatewayToModel(ctx, &state, gateway)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update gateway",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan GatewayResourceModel
	var state GatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := infisical.UpdateGatewayRequest{ID: state.ID.ValueString()}

	if !plan.Name.Equal(state.Name) {
		updateRequest.Name = infisicalstrings.StringToPtr(plan.Name.ValueString())
	}

	// Resending Kubernetes auth re-validates against the live cluster, so a rename would need it up.
	authChanged, diags := gatewayAuthMethodChanged(ctx, req)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if authChanged {
		authMethod, authDiags := gatewayAuthMethodInput(ctx, &plan)
		resp.Diagnostics.Append(authDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateRequest.AuthMethod = &authMethod
	}

	gateway, err := r.client.UpdateGateway(updateRequest)
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Gateway not found",
				"Gateway with ID "+state.ID.ValueString()+" no longer exists in Infisical. Run `terraform apply -refresh-only` to reconcile state, then apply again.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating gateway",
			"Couldn't update gateway in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.applyGatewayToModel(ctx, &plan, gateway)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete gateway",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteGateway(state.ID.ValueString())
	if err != nil && !errors.Is(err, infisical.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Error deleting gateway",
			"Couldn't delete gateway from Infisical, unexpected error: "+err.Error(),
		)
	}
}

func (r *GatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import gateway",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	gateway, err := r.client.GetGatewayById(req.ID)
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Gateway not found",
				"No gateway with ID "+req.ID+" exists in this organization.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error importing gateway",
			"Couldn't read gateway with ID "+req.ID+" from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	if gateway.AuthMethod.Method == infisical.GatewayAuthMethodIdentity {
		resp.Diagnostics.AddError(
			"Gateway uses legacy machine identity auth",
			fmt.Sprintf(
				"Gateway %q authenticates with a machine identity, which predates gateway auth methods and cannot be managed by this resource or migrated to one. "+
					"Create a new gateway with aws_auth, gcp_auth, kubernetes_auth or token_auth and repoint your deployment at it.",
				gateway.Name,
			),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func gatewayAuthMethodChanged(ctx context.Context, req resource.UpdateRequest) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	for _, block := range []string{"aws_auth", "gcp_auth", "kubernetes_auth", "token_auth"} {
		var planned, stored types.Object
		diags.Append(req.Plan.GetAttribute(ctx, path.Root(block), &planned)...)
		diags.Append(req.State.GetAttribute(ctx, path.Root(block), &stored)...)
		if diags.HasError() {
			return false, diags
		}
		if !planned.Equal(stored) {
			return true, diags
		}
	}

	return false, diags
}

func gatewayAuthMethodInput(ctx context.Context, plan *GatewayResourceModel) (infisical.GatewayAuthMethodInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	awsAuth := decodeAuthBlock[gatewayAwsAuthModel](ctx, plan.AwsAuth, &diags)
	gcpAuth := decodeAuthBlock[gatewayGcpAuthModel](ctx, plan.GcpAuth, &diags)
	kubernetesAuth := decodeAuthBlock[gatewayKubernetesAuthModel](ctx, plan.KubernetesAuth, &diags)
	if diags.HasError() {
		return infisical.GatewayAuthMethodInput{}, diags
	}

	switch {
	case awsAuth != nil:
		principalArns, d := csvFromSet(ctx, awsAuth.AllowedPrincipalArns)
		diags.Append(d...)
		accountIds, d := csvFromSet(ctx, awsAuth.AllowedAccountIds)
		diags.Append(d...)
		if diags.HasError() {
			return infisical.GatewayAuthMethodInput{}, diags
		}

		return infisical.GatewayAuthMethodInput{
			Method:               infisical.GatewayAuthMethodAws,
			AllowedPrincipalArns: infisicalstrings.StringToPtr(principalArns),
			AllowedAccountIds:    infisicalstrings.StringToPtr(accountIds),
		}, diags

	case gcpAuth != nil:
		serviceAccounts, d := csvFromSet(ctx, gcpAuth.AllowedServiceAccounts)
		diags.Append(d...)
		projects, d := csvFromSet(ctx, gcpAuth.AllowedProjects)
		diags.Append(d...)
		zones, d := csvFromSet(ctx, gcpAuth.AllowedZones)
		diags.Append(d...)
		if diags.HasError() {
			return infisical.GatewayAuthMethodInput{}, diags
		}

		return infisical.GatewayAuthMethodInput{
			Method:                 infisical.GatewayAuthMethodGcp,
			Type:                   infisicalstrings.StringToPtr(gcpAuth.Type.ValueString()),
			AllowedServiceAccounts: infisicalstrings.StringToPtr(serviceAccounts),
			AllowedProjects:        infisicalstrings.StringToPtr(projects),
			AllowedZones:           infisicalstrings.StringToPtr(zones),
		}, diags

	case kubernetesAuth != nil:
		namespaces, d := csvFromSet(ctx, kubernetesAuth.AllowedNamespaces)
		diags.Append(d...)
		names, d := csvFromSet(ctx, kubernetesAuth.AllowedServiceAccountName)
		diags.Append(d...)
		if diags.HasError() {
			return infisical.GatewayAuthMethodInput{}, diags
		}

		input := infisical.GatewayAuthMethodInput{
			Method:               infisical.GatewayAuthMethodKubernetes,
			TokenReviewMode:      infisicalstrings.StringToPtr(kubernetesAuth.TokenReviewMode.ValueString()),
			AllowedNamespaces:    infisicalstrings.StringToPtr(namespaces),
			AllowedNames:         infisicalstrings.StringToPtr(names),
			AllowedAudience:      infisicalstrings.StringToPtr(kubernetesAuth.AllowedAudience.ValueString()),
			VerifyTlsCertificate: kubernetesAuth.VerifyTlsCertificate.ValueBoolPointer(),
		}

		// Gateway review mode rejects a host outright, and "" fails the format check.
		if host := kubernetesAuth.KubernetesHost.ValueString(); host != "" {
			input.KubernetesHost = infisicalstrings.StringToPtr(host)
		}
		// Always sent: "" is how the API is told to clear a stored certificate.
		input.CaCertificate = infisicalstrings.StringToPtr(kubernetesAuth.CaCertificate.ValueString())
		// Sent whenever it is set, "" included: the API keeps the stored token when the field is
		// absent and clears it when it arrives empty.
		if !kubernetesAuth.TokenReviewerJwt.IsNull() && !kubernetesAuth.TokenReviewerJwt.IsUnknown() {
			input.TokenReviewerJwt = infisicalstrings.StringToPtr(kubernetesAuth.TokenReviewerJwt.ValueString())
		}
		if gatewayID := kubernetesAuth.ReviewerGatewayID.ValueString(); gatewayID != "" {
			input.GatewayID = infisicalstrings.StringToPtr(gatewayID)
		}
		if poolID := kubernetesAuth.ReviewerGatewayPoolID.ValueString(); poolID != "" {
			input.GatewayPoolID = infisicalstrings.StringToPtr(poolID)
		}

		return input, diags

	case !plan.TokenAuth.IsNull() && !plan.TokenAuth.IsUnknown():
		return infisical.GatewayAuthMethodInput{Method: infisical.GatewayAuthMethodToken}, diags
	}

	// Apply resolves every config expression, so a block still unknown here means the plan was
	// never refined and reporting it as "none configured" would send the reader the wrong way.
	if plan.AwsAuth.IsUnknown() || plan.GcpAuth.IsUnknown() || plan.KubernetesAuth.IsUnknown() || plan.TokenAuth.IsUnknown() {
		diags.AddError(
			"Auth method not resolved",
			"The auth method block is still unknown at apply time. Select it with a value Terraform can resolve during planning, such as a variable or a local, rather than an attribute of a resource created in the same run.",
		)
		return infisical.GatewayAuthMethodInput{}, diags
	}

	diags.AddError(
		"No auth method configured",
		"Set exactly one of aws_auth, gcp_auth, kubernetes_auth or token_auth on the gateway.",
	)
	return infisical.GatewayAuthMethodInput{}, diags
}

// The reviewer JWT is kept from config: the API never returns it.
func (r *GatewayResource) applyGatewayToModel(ctx context.Context, model *GatewayResourceModel, gateway infisical.GatewayDetails) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(gateway.ID)
	model.Name = types.StringValue(gateway.Name)

	config := gateway.AuthMethod.Config

	priorAws := decodeAuthBlock[gatewayAwsAuthModel](ctx, model.AwsAuth, &diags)
	priorGcp := decodeAuthBlock[gatewayGcpAuthModel](ctx, model.GcpAuth, &diags)
	priorKubernetes := decodeAuthBlock[gatewayKubernetesAuthModel](ctx, model.KubernetesAuth, &diags)
	if diags.HasError() {
		return diags
	}
	nullAuthBlocks(model)

	switch gateway.AuthMethod.Method {
	case infisical.GatewayAuthMethodAws:
		prior := gatewayAwsAuthModel{
			AllowedPrincipalArns: types.SetNull(types.StringType),
			AllowedAccountIds:    types.SetNull(types.StringType),
		}
		if priorAws != nil {
			prior = *priorAws
		}

		model.AwsAuth = encodeAuthBlock(ctx, gatewayAwsAuthSchema(), gatewayAwsAuthModel{
			AllowedPrincipalArns: keepUnsetSet(config.AllowedPrincipalArns, prior.AllowedPrincipalArns, &diags),
			AllowedAccountIds:    keepUnsetSet(config.AllowedAccountIds, prior.AllowedAccountIds, &diags),
		}, &diags)

	case infisical.GatewayAuthMethodGcp:
		prior := gatewayGcpAuthModel{
			AllowedServiceAccounts: types.SetNull(types.StringType),
			AllowedProjects:        types.SetNull(types.StringType),
			AllowedZones:           types.SetNull(types.StringType),
		}
		if priorGcp != nil {
			prior = *priorGcp
		}

		model.GcpAuth = encodeAuthBlock(ctx, gatewayGcpAuthSchema(), gatewayGcpAuthModel{
			Type:                   types.StringValue(config.Type),
			AllowedServiceAccounts: keepUnsetSet(config.AllowedServiceAccounts, prior.AllowedServiceAccounts, &diags),
			AllowedProjects:        keepUnsetSet(config.AllowedProjects, prior.AllowedProjects, &diags),
			AllowedZones:           keepUnsetSet(config.AllowedZones, prior.AllowedZones, &diags),
		}, &diags)

	case infisical.GatewayAuthMethodKubernetes:
		prior := gatewayKubernetesAuthModel{
			TokenReviewerJwt:          types.StringNull(),
			CaCertificate:             customtypes.NewTrimmedStringNull(),
			KubernetesHost:            customtypes.NewKubernetesHostNull(),
			AllowedAudience:           types.StringNull(),
			AllowedNamespaces:         types.SetNull(types.StringType),
			AllowedServiceAccountName: types.SetNull(types.StringType),
		}
		if priorKubernetes != nil {
			prior = *priorKubernetes
		}

		namespaces := keepUnsetSet(config.AllowedNamespaces, prior.AllowedNamespaces, &diags)
		names := keepUnsetSet(config.AllowedNames, prior.AllowedServiceAccountName, &diags)

		host := prior.KubernetesHost
		if config.KubernetesHost != "" {
			host = customtypes.NewKubernetesHostValue(config.KubernetesHost)
		}
		ca := prior.CaCertificate
		if config.CaCertificate != "" {
			ca = customtypes.NewTrimmedStringValue(config.CaCertificate)
		}

		// The API reports only whether a reviewer JWT is stored, never its value. Once it is
		// gone, ours has to go too, or a cleared credential leaves a clean plan behind.
		reviewerJwt := prior.TokenReviewerJwt
		if !config.HasTokenReviewerJwt && reviewerJwt.ValueString() != "" {
			reviewerJwt = types.StringNull()
		}

		model.KubernetesAuth = encodeAuthBlock(ctx, gatewayKubernetesAuthSchema(), gatewayKubernetesAuthModel{
			TokenReviewMode:           types.StringValue(config.TokenReviewMode),
			KubernetesHost:            host,
			CaCertificate:             ca,
			TokenReviewerJwt:          reviewerJwt,
			ReviewerGatewayID:         optionalString(config.GatewayID),
			ReviewerGatewayPoolID:     optionalString(config.GatewayPoolID),
			AllowedNamespaces:         namespaces,
			AllowedServiceAccountName: names,
			AllowedAudience:           keepUnsetString(config.AllowedAudience, prior.AllowedAudience),
			VerifyTlsCertificate:      types.BoolValue(config.VerifyTlsCertificate),
			HasTokenReviewerJwt:       types.BoolValue(config.HasTokenReviewerJwt),
		}, &diags)

	case infisical.GatewayAuthMethodToken:
		model.TokenAuth = encodeAuthBlock(ctx, gatewayTokenAuthSchema(), gatewayTokenAuthModel{}, &diags)

	case infisical.GatewayAuthMethodIdentity:
		diags.AddError(
			"Gateway uses legacy machine identity auth",
			fmt.Sprintf(
				"Gateway %q authenticates with machine identity %s, which predates gateway auth methods and cannot be managed by this resource or migrated to one. "+
					"Create a new gateway with aws_auth, gcp_auth, kubernetes_auth or token_auth and repoint your deployment at it. "+
					"To stop managing this one with Terraform, drop it from state with `terraform state rm`; "+
					"`terraform destroy -refresh=false` removes it without reading it back.",
				gateway.Name, config.IdentityID,
			),
		)

	default:
		diags.AddError(
			"Unrecognized gateway auth method",
			fmt.Sprintf(
				"Infisical reported auth method %q for gateway %q, which this provider version does not understand. Upgrade the provider.",
				gateway.AuthMethod.Method, gateway.Name,
			),
		)
	}

	return diags
}

// An empty value from the API means unset. Overwriting a null config with "" or an empty set
// produces "inconsistent result after apply", so the configured form of unset is kept.
func keepUnsetString(apiValue string, prior types.String) types.String {
	if apiValue == "" {
		return prior
	}
	return types.StringValue(apiValue)
}

func keepUnsetSet(csv string, prior types.Set, diags *diag.Diagnostics) types.Set {
	if csv == "" {
		if prior.IsNull() || prior.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		return prior
	}

	set, d := setFromCsv(csv)
	diags.Append(d...)
	return set
}

func optionalString(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func csvFromSet(ctx context.Context, set types.Set) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if set.IsNull() || set.IsUnknown() {
		return "", diags
	}

	var values []string
	diags.Append(set.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return "", diags
	}

	sort.Strings(values)
	return strings.Join(values, ","), diags
}

func setFromCsv(csv string) (types.Set, diag.Diagnostics) {
	entries := infisicalstrings.StringSplitAndTrim(csv, ",")

	elements := make([]attr.Value, 0, len(entries))
	for _, entry := range entries {
		elements = append(elements, types.StringValue(entry))
	}

	return types.SetValue(types.StringType, elements)
}
