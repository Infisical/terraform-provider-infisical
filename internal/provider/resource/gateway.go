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
}

type gatewayTokenAuthModel struct{}

type GatewayResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	AwsAuth        *gatewayAwsAuthModel        `tfsdk:"aws_auth"`
	GcpAuth        *gatewayGcpAuthModel        `tfsdk:"gcp_auth"`
	KubernetesAuth *gatewayKubernetesAuthModel `tfsdk:"kubernetes_auth"`
	TokenAuth      *gatewayTokenAuthModel      `tfsdk:"token_auth"`
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
				Description: "The URL of the Kubernetes API server, for example https://my-cluster.example.com:6443. Required unless `token_review_mode` is `gateway`, where it must be omitted.",
				Optional:    true,
				Computed:    true,
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

	switch {
	case config.AwsAuth != nil:
		if isEmpty(config.AwsAuth.AllowedPrincipalArns) && isEmpty(config.AwsAuth.AllowedAccountIds) {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_auth"),
				"No AWS allowlist configured",
				"Set allowed_principal_arns or allowed_account_ids. Without one, any AWS caller could authenticate as this gateway.",
			)
		}

	case config.GcpAuth != nil:
		if isEmpty(config.GcpAuth.AllowedServiceAccounts) && isEmpty(config.GcpAuth.AllowedProjects) {
			resp.Diagnostics.AddAttributeError(
				path.Root("gcp_auth"),
				"No GCP allowlist configured",
				"Set allowed_service_accounts or allowed_projects. A zone on its own restricts nothing, because anyone can create an instance in a given zone.",
			)
		}

		if config.GcpAuth.Type.ValueString() == infisical.GatewayGcpAuthTypeIam {
			if !isEmpty(config.GcpAuth.AllowedProjects) || !isEmpty(config.GcpAuth.AllowedZones) {
				resp.Diagnostics.AddAttributeError(
					path.Root("gcp_auth"),
					"Projects and zones do not apply to the iam type",
					"allowed_projects and allowed_zones only narrow a Compute Engine token. Restrict an iam service account token with allowed_service_accounts instead.",
				)
			}
		}

	case config.KubernetesAuth != nil:
		if isEmpty(config.KubernetesAuth.AllowedNamespaces) && isEmpty(config.KubernetesAuth.AllowedServiceAccountName) {
			resp.Diagnostics.AddAttributeError(
				path.Root("kubernetes_auth"),
				"No Kubernetes allowlist configured",
				"Set allowed_namespaces or allowed_service_account_names. Without one, any service account in the cluster could authenticate as this gateway.",
			)
		}

		hasReviewerGateway := config.KubernetesAuth.ReviewerGatewayID.ValueString() != ""
		hasReviewerPool := config.KubernetesAuth.ReviewerGatewayPoolID.ValueString() != ""

		if hasReviewerGateway && hasReviewerPool {
			resp.Diagnostics.AddAttributeError(
				path.Root("kubernetes_auth"),
				"Two TokenReview reviewers configured",
				"Set reviewer_gateway_id or reviewer_gateway_pool_id, not both. A pool picks any healthy member, so naming a gateway as well has no meaning.",
			)
		}

		if config.KubernetesAuth.TokenReviewMode.ValueString() == infisical.GatewayKubernetesTokenReviewModeGateway {
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

	resp.Diagnostics.Append(r.applyGatewayToModel(&plan, gateway)...)
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

	resp.Diagnostics.Append(r.applyGatewayToModel(&state, gateway)...)
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

	resp.Diagnostics.Append(r.applyGatewayToModel(&plan, gateway)...)
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

	switch {
	case plan.AwsAuth != nil:
		principalArns, d := csvFromSet(ctx, plan.AwsAuth.AllowedPrincipalArns)
		diags.Append(d...)
		accountIds, d := csvFromSet(ctx, plan.AwsAuth.AllowedAccountIds)
		diags.Append(d...)
		if diags.HasError() {
			return infisical.GatewayAuthMethodInput{}, diags
		}

		return infisical.GatewayAuthMethodInput{
			Method:               infisical.GatewayAuthMethodAws,
			AllowedPrincipalArns: infisicalstrings.StringToPtr(principalArns),
			AllowedAccountIds:    infisicalstrings.StringToPtr(accountIds),
		}, diags

	case plan.GcpAuth != nil:
		serviceAccounts, d := csvFromSet(ctx, plan.GcpAuth.AllowedServiceAccounts)
		diags.Append(d...)
		projects, d := csvFromSet(ctx, plan.GcpAuth.AllowedProjects)
		diags.Append(d...)
		zones, d := csvFromSet(ctx, plan.GcpAuth.AllowedZones)
		diags.Append(d...)
		if diags.HasError() {
			return infisical.GatewayAuthMethodInput{}, diags
		}

		return infisical.GatewayAuthMethodInput{
			Method:                 infisical.GatewayAuthMethodGcp,
			Type:                   infisicalstrings.StringToPtr(plan.GcpAuth.Type.ValueString()),
			AllowedServiceAccounts: infisicalstrings.StringToPtr(serviceAccounts),
			AllowedProjects:        infisicalstrings.StringToPtr(projects),
			AllowedZones:           infisicalstrings.StringToPtr(zones),
		}, diags

	case plan.KubernetesAuth != nil:
		namespaces, d := csvFromSet(ctx, plan.KubernetesAuth.AllowedNamespaces)
		diags.Append(d...)
		names, d := csvFromSet(ctx, plan.KubernetesAuth.AllowedServiceAccountName)
		diags.Append(d...)
		if diags.HasError() {
			return infisical.GatewayAuthMethodInput{}, diags
		}

		input := infisical.GatewayAuthMethodInput{
			Method:               infisical.GatewayAuthMethodKubernetes,
			TokenReviewMode:      infisicalstrings.StringToPtr(plan.KubernetesAuth.TokenReviewMode.ValueString()),
			AllowedNamespaces:    infisicalstrings.StringToPtr(namespaces),
			AllowedNames:         infisicalstrings.StringToPtr(names),
			AllowedAudience:      infisicalstrings.StringToPtr(plan.KubernetesAuth.AllowedAudience.ValueString()),
			VerifyTlsCertificate: plan.KubernetesAuth.VerifyTlsCertificate.ValueBoolPointer(),
		}

		// Gateway review mode rejects a host outright, and "" fails the format check.
		if host := plan.KubernetesAuth.KubernetesHost.ValueString(); host != "" {
			input.KubernetesHost = infisicalstrings.StringToPtr(host)
		}
		// Always sent: "" is how the API is told to clear a stored certificate.
		input.CaCertificate = infisicalstrings.StringToPtr(plan.KubernetesAuth.CaCertificate.ValueString())
		if jwt := plan.KubernetesAuth.TokenReviewerJwt.ValueString(); jwt != "" {
			input.TokenReviewerJwt = infisicalstrings.StringToPtr(jwt)
		}
		if gatewayID := plan.KubernetesAuth.ReviewerGatewayID.ValueString(); gatewayID != "" {
			input.GatewayID = infisicalstrings.StringToPtr(gatewayID)
		}
		if poolID := plan.KubernetesAuth.ReviewerGatewayPoolID.ValueString(); poolID != "" {
			input.GatewayPoolID = infisicalstrings.StringToPtr(poolID)
		}

		return input, diags

	case plan.TokenAuth != nil:
		return infisical.GatewayAuthMethodInput{Method: infisical.GatewayAuthMethodToken}, diags
	}

	diags.AddError(
		"No auth method configured",
		"Set exactly one of aws_auth, gcp_auth, kubernetes_auth or token_auth on the gateway.",
	)
	return infisical.GatewayAuthMethodInput{}, diags
}

// The reviewer JWT is kept from config: the API never returns it.
func (r *GatewayResource) applyGatewayToModel(model *GatewayResourceModel, gateway infisical.GatewayDetails) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(gateway.ID)
	model.Name = types.StringValue(gateway.Name)

	config := gateway.AuthMethod.Config

	switch gateway.AuthMethod.Method {
	case infisical.GatewayAuthMethodAws:
		prior := gatewayAwsAuthModel{
			AllowedPrincipalArns: types.SetNull(types.StringType),
			AllowedAccountIds:    types.SetNull(types.StringType),
		}
		if model.AwsAuth != nil {
			prior = *model.AwsAuth
		}

		model.AwsAuth = &gatewayAwsAuthModel{
			AllowedPrincipalArns: keepUnsetSet(config.AllowedPrincipalArns, prior.AllowedPrincipalArns, &diags),
			AllowedAccountIds:    keepUnsetSet(config.AllowedAccountIds, prior.AllowedAccountIds, &diags),
		}
		model.GcpAuth, model.KubernetesAuth, model.TokenAuth = nil, nil, nil

	case infisical.GatewayAuthMethodGcp:
		prior := gatewayGcpAuthModel{
			AllowedServiceAccounts: types.SetNull(types.StringType),
			AllowedProjects:        types.SetNull(types.StringType),
			AllowedZones:           types.SetNull(types.StringType),
		}
		if model.GcpAuth != nil {
			prior = *model.GcpAuth
		}

		model.GcpAuth = &gatewayGcpAuthModel{
			Type:                   types.StringValue(config.Type),
			AllowedServiceAccounts: keepUnsetSet(config.AllowedServiceAccounts, prior.AllowedServiceAccounts, &diags),
			AllowedProjects:        keepUnsetSet(config.AllowedProjects, prior.AllowedProjects, &diags),
			AllowedZones:           keepUnsetSet(config.AllowedZones, prior.AllowedZones, &diags),
		}
		model.AwsAuth, model.KubernetesAuth, model.TokenAuth = nil, nil, nil

	case infisical.GatewayAuthMethodKubernetes:
		prior := gatewayKubernetesAuthModel{
			TokenReviewerJwt:          types.StringNull(),
			CaCertificate:             customtypes.NewTrimmedStringNull(),
			KubernetesHost:            customtypes.NewKubernetesHostNull(),
			AllowedAudience:           types.StringNull(),
			AllowedNamespaces:         types.SetNull(types.StringType),
			AllowedServiceAccountName: types.SetNull(types.StringType),
		}
		if model.KubernetesAuth != nil {
			prior = *model.KubernetesAuth
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

		model.KubernetesAuth = &gatewayKubernetesAuthModel{
			TokenReviewMode:           types.StringValue(config.TokenReviewMode),
			KubernetesHost:            host,
			CaCertificate:             ca,
			TokenReviewerJwt:          prior.TokenReviewerJwt,
			ReviewerGatewayID:         optionalString(config.GatewayID),
			ReviewerGatewayPoolID:     optionalString(config.GatewayPoolID),
			AllowedNamespaces:         namespaces,
			AllowedServiceAccountName: names,
			AllowedAudience:           keepUnsetString(config.AllowedAudience, prior.AllowedAudience),
			VerifyTlsCertificate:      types.BoolValue(config.VerifyTlsCertificate),
		}
		model.AwsAuth, model.GcpAuth, model.TokenAuth = nil, nil, nil

	case infisical.GatewayAuthMethodToken:
		model.TokenAuth = &gatewayTokenAuthModel{}
		model.AwsAuth, model.GcpAuth, model.KubernetesAuth = nil, nil, nil

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
