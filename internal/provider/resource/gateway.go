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

const defaultStsEndpoint = "https://sts.amazonaws.com/"

// Mirrors the API's slug rule, which refuses anything slugify would rewrite. Validating it here
// turns an apply-time 400 into a plan-time error.
var gatewayNameSlugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func NewGatewayResource() resource.Resource {
	return &GatewayResource{}
}

type GatewayResource struct {
	client *infisical.Client
}

type gatewayAwsAuthModel struct {
	StsEndpoint          types.String `tfsdk:"sts_endpoint"`
	AllowedPrincipalArns types.Set    `tfsdk:"allowed_principal_arns"`
	AllowedAccountIds    types.Set    `tfsdk:"allowed_account_ids"`
}

type gatewayGcpAuthModel struct {
	Type                   types.String `tfsdk:"type"`
	AllowedServiceAccounts types.Set    `tfsdk:"allowed_service_accounts"`
	AllowedProjects        types.Set    `tfsdk:"allowed_projects"`
	AllowedZones           types.Set    `tfsdk:"allowed_zones"`
}

type gatewayKubernetesAuthModel struct {
	TokenReviewMode           types.String                   `tfsdk:"token_review_mode"`
	KubernetesHost            types.String                   `tfsdk:"kubernetes_host"`
	CaCertificate             customtypes.TrimmedStringValue `tfsdk:"ca_certificate"`
	TokenReviewerJwt          types.String                   `tfsdk:"token_reviewer_jwt"`
	ReviewerGatewayID         types.String                   `tfsdk:"reviewer_gateway_id"`
	ReviewerGatewayPoolID     types.String                   `tfsdk:"reviewer_gateway_pool_id"`
	AllowedNamespaces         types.Set                      `tfsdk:"allowed_namespaces"`
	AllowedServiceAccountName types.Set                      `tfsdk:"allowed_service_account_names"`
	AllowedAudience           types.String                   `tfsdk:"allowed_audience"`
	VerifyTlsCertificate      types.Bool                     `tfsdk:"verify_tls_certificate"`
}

type gatewayTokenAuthModel struct{}

type GatewayResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	AwsAuth        *gatewayAwsAuthModel        `tfsdk:"aws_auth"`
	GcpAuth        *gatewayGcpAuthModel        `tfsdk:"gcp_auth"`
	KubernetesAuth *gatewayKubernetesAuthModel `tfsdk:"kubernetes_auth"`
	TokenAuth      *gatewayTokenAuthModel      `tfsdk:"token_auth"`

	IdentityID      types.String `tfsdk:"identity_id"`
	RelayID         types.String `tfsdk:"relay_id"`
	DirectAddress   types.String `tfsdk:"direct_address"`
	Heartbeat       types.String `tfsdk:"heartbeat"`
	DirectHeartbeat types.String `tfsdk:"direct_heartbeat"`
	CanRevoke       types.Bool   `tfsdk:"can_revoke"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
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

			"identity_id": schema.StringAttribute{
				Description: "The machine identity a legacy gateway is bound to. Only set on gateways created before auth methods existed, which cannot be managed by this resource.",
				Computed:    true,
			},
			"relay_id": schema.StringAttribute{
				Description: "The relay the gateway connected through. Written by the gateway process when it connects, so it is empty until then.",
				Computed:    true,
			},
			"direct_address": schema.StringAttribute{
				Description: "The address the gateway advertises for direct connections. Written by the gateway process when it connects.",
				Computed:    true,
			},
			"heartbeat": schema.StringAttribute{
				Description: "When the gateway was last reachable through its relay.",
				Computed:    true,
			},
			"direct_heartbeat": schema.StringAttribute{
				Description: "When the gateway was last reachable at its direct address.",
				Computed:    true,
			},
			"can_revoke": schema.BoolAttribute{
				Description: "Whether the gateway currently holds credentials that revoking would invalidate.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description:   "When the gateway was created.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "When the gateway was last updated.",
				Computed:    true,
			},
		},
	}
}

func gatewayAwsAuthSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Authenticate the gateway with its AWS IAM identity. The machine re-authenticates on every start, so no secret is stored. At least one of `allowed_principal_arns` or `allowed_account_ids` must be non-empty.",
		Attributes: map[string]schema.Attribute{
			"sts_endpoint": schema.StringAttribute{
				Description: "The AWS STS endpoint used to verify the signed request. Defaults to " + defaultStsEndpoint + ".",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultStsEndpoint),
			},
			"allowed_principal_arns": schema.SetAttribute{
				Description: "IAM principal ARNs allowed to authenticate as this gateway. Supports `*` wildcards.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"allowed_account_ids": schema.SetAttribute{
				Description: "AWS account IDs allowed to authenticate as this gateway.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func gatewayGcpAuthSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Authenticate the gateway with its GCP identity. The machine re-authenticates on every start, so no secret is stored. At least one of `allowed_service_accounts` or `allowed_projects` must be non-empty, because a zone on its own restricts nothing.",
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
				Computed:    true,
			},
			"allowed_projects": schema.SetAttribute{
				Description: "GCP project IDs whose Compute Engine instances are allowed to authenticate as this gateway. Only applies when `type` is `gce`.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"allowed_zones": schema.SetAttribute{
				Description: "GCP zones whose Compute Engine instances are allowed to authenticate as this gateway. Only applies when `type` is `gce`.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
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
				Description: "The URL of the Kubernetes API server, for example https://my-cluster.example.com:6443. Required unless `token_review_mode` is `gateway`, where it must be omitted.",
				Optional:    true,
				Computed:    true,
			},
			"ca_certificate": schema.StringAttribute{
				// Infisical strips the trailing newline, and file() always supplies one, so the
				// values are compared with surrounding whitespace ignored.
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The PEM-encoded CA certificate that issued the Kubernetes API server's TLS certificate.",
				Optional:    true,
				Computed:    true,
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
				Computed:    true,
			},
			"allowed_service_account_names": schema.SetAttribute{
				Description: "Kubernetes service account names allowed to authenticate as this gateway. Supports `*` wildcards.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"allowed_audience": schema.StringAttribute{
				Description: "The audience the service account token must carry. Leave empty to skip the audience check.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
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

// ValidateConfig carries the API's cross-field rules into the plan. resourcevalidator cannot
// express them: its combinators would fire on blocks the configuration never set.
func (r *GatewayResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config GatewayResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An unknown set resolves at apply time, so treating it as empty here would reject a valid
	// configuration that draws its allowlist from another resource.
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

		// An IAM token carries no Compute Engine claim, so there is nothing for these to match.
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

	// Resending an unchanged auth method is not free: Kubernetes configs are re-validated against
	// the live cluster, so a rename would fail whenever that cluster happens to be unreachable.
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

// Importing checks the auth method first. Letting a legacy identity-bound gateway into state would
// wedge the resource, because Read refuses it and every later plan, apply and destroy fails on that.
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

// gatewayAuthMethodChanged reports whether the configured auth method differs from the one in
// state, comparing the blocks as Terraform sees them rather than as the API echoes them back.
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

// gatewayAuthMethodInput builds the API payload from whichever auth block the configuration set.
// ConfigValidators has already established that exactly one of them is non-null.
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
			StsEndpoint:          infisicalstrings.StringToPtr(plan.AwsAuth.StsEndpoint.ValueString()),
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

		// Sending an empty host fails the host format check, and gateway review mode rejects a host
		// outright, so anything blank is omitted rather than sent as "".
		if host := plan.KubernetesAuth.KubernetesHost.ValueString(); host != "" {
			input.KubernetesHost = infisicalstrings.StringToPtr(host)
		}
		if ca := plan.KubernetesAuth.CaCertificate.ValueString(); ca != "" {
			input.CaCertificate = infisicalstrings.StringToPtr(ca)
		}
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

// applyGatewayToModel writes the API's view of the gateway onto the model, leaving the write-only
// reviewer JWT as the configuration supplied it, since Infisical never returns it.
func (r *GatewayResource) applyGatewayToModel(model *GatewayResourceModel, gateway infisical.GatewayDetails) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(gateway.ID)
	model.Name = types.StringValue(gateway.Name)
	model.IdentityID = optionalString(gateway.IdentityID)
	model.RelayID = optionalString(gateway.RelayID)
	model.DirectAddress = optionalString(gateway.DirectAddress)
	model.Heartbeat = optionalString(gateway.Heartbeat)
	model.DirectHeartbeat = optionalString(gateway.DirectHeartbeat)
	model.CanRevoke = types.BoolValue(gateway.CanRevoke)
	model.CreatedAt = types.StringValue(gateway.CreatedAt)
	model.UpdatedAt = types.StringValue(gateway.UpdatedAt)

	config := gateway.AuthMethod.Config

	switch gateway.AuthMethod.Method {
	case infisical.GatewayAuthMethodAws:
		principalArns, d := setFromCsv(config.AllowedPrincipalArns)
		diags.Append(d...)
		accountIds, d := setFromCsv(config.AllowedAccountIds)
		diags.Append(d...)

		model.AwsAuth = &gatewayAwsAuthModel{
			StsEndpoint:          types.StringValue(config.StsEndpoint),
			AllowedPrincipalArns: principalArns,
			AllowedAccountIds:    accountIds,
		}
		model.GcpAuth, model.KubernetesAuth, model.TokenAuth = nil, nil, nil

	case infisical.GatewayAuthMethodGcp:
		serviceAccounts, d := setFromCsv(config.AllowedServiceAccounts)
		diags.Append(d...)
		projects, d := setFromCsv(config.AllowedProjects)
		diags.Append(d...)
		zones, d := setFromCsv(config.AllowedZones)
		diags.Append(d...)

		model.GcpAuth = &gatewayGcpAuthModel{
			Type:                   types.StringValue(config.Type),
			AllowedServiceAccounts: serviceAccounts,
			AllowedProjects:        projects,
			AllowedZones:           zones,
		}
		model.AwsAuth, model.KubernetesAuth, model.TokenAuth = nil, nil, nil

	case infisical.GatewayAuthMethodKubernetes:
		namespaces, d := setFromCsv(config.AllowedNamespaces)
		diags.Append(d...)
		names, d := setFromCsv(config.AllowedNames)
		diags.Append(d...)

		reviewerJwt := types.StringNull()
		if model.KubernetesAuth != nil {
			reviewerJwt = model.KubernetesAuth.TokenReviewerJwt
		}

		model.KubernetesAuth = &gatewayKubernetesAuthModel{
			TokenReviewMode:           types.StringValue(config.TokenReviewMode),
			KubernetesHost:            types.StringValue(config.KubernetesHost),
			CaCertificate:             customtypes.NewTrimmedStringValue(config.CaCertificate),
			TokenReviewerJwt:          reviewerJwt,
			ReviewerGatewayID:         optionalString(config.GatewayID),
			ReviewerGatewayPoolID:     optionalString(config.GatewayPoolID),
			AllowedNamespaces:         namespaces,
			AllowedServiceAccountName: names,
			AllowedAudience:           types.StringValue(config.AllowedAudience),
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

func optionalString(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// csvFromSet renders a set as the comma-separated string the API takes. The elements are sorted so
// two applies of the same configuration send the same string, which keeps the audit log readable.
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
