package resource

import (
	"context"
	"errors"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &GatewayEnrollmentTokenResource{}
	_ resource.ResourceWithConfigure = &GatewayEnrollmentTokenResource{}
)

func NewGatewayEnrollmentTokenResource() resource.Resource {
	return &GatewayEnrollmentTokenResource{}
}

type GatewayEnrollmentTokenResource struct {
	client *infisical.Client
}

type GatewayEnrollmentTokenResourceModel struct {
	ID        types.String `tfsdk:"id"`
	GatewayID types.String `tfsdk:"gateway_id"`
	Keepers   types.Map    `tfsdk:"keepers"`
	Token     types.String `tfsdk:"token"`
	ExpiresAt types.String `tfsdk:"expires_at"`
}

func (r *GatewayEnrollmentTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway_enrollment_token"
}

func (r *GatewayEnrollmentTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Mint the one-time enrollment token a token-auth gateway uses to bootstrap. " +
			"The token is single-use and expires an hour after it is issued, and minting a new one invalidates any token issued earlier for the same gateway. " +
			"Use `keepers` to tie re-minting to whatever forces the machine that consumes the token to be rebuilt. " +
			"Where the platform can vouch for the machine, prefer `aws_auth`, `gcp_auth` or `kubernetes_auth` on the gateway instead, since those re-authenticate on every start and need none of this.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the gateway the token was minted for. An enrollment token has no identifier of its own, because a gateway holds at most one at a time.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"gateway_id": schema.StringAttribute{
				Description:   "The ID of the gateway to mint the token for. The gateway must be configured with `token_auth`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"keepers": schema.MapAttribute{
				Description: "Arbitrary values that force a new token when any of them changes. " +
					"List whatever forces the machine that consumes the token to be rebuilt, for example the AMI ID and subnet of an EC2 instance. " +
					"Nothing validates this list, so an input you leave out means the rebuilt machine boots with a token that was already consumed, stays offline, and shows no drift.",
				ElementType:   types.StringType,
				Optional:      true,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				Description: "The enrollment token. Single-use, valid for one hour, and stored in Terraform state in plaintext.",
				Computed:    true,
				Sensitive:   true,
			},
			"expires_at": schema.StringAttribute{
				Description: "When the token expires. A token that has already been used is invalid before this.",
				Computed:    true,
			},
		},
	}
}

func (r *GatewayEnrollmentTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GatewayEnrollmentTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create gateway enrollment token",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan GatewayEnrollmentTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gatewayID := plan.GatewayID.ValueString()

	minted, err := r.client.MintGatewayEnrollmentToken(gatewayID)
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Gateway not found",
				"No gateway with ID "+gatewayID+" exists in this organization.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error minting gateway enrollment token",
			"Couldn't mint an enrollment token for gateway "+gatewayID+". "+
				"Infisical rejects this when the gateway is not configured for token authentication, so check that its `token_auth` block is still set. "+
				"The full error was: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(gatewayID)
	plan.Token = types.StringValue(minted.Token)
	plan.ExpiresAt = types.StringValue(minted.ExpiresAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// The token itself cannot be read back, so this only checks the gateway still takes token auth.
func (r *GatewayEnrollmentTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read gateway enrollment token",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GatewayEnrollmentTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gatewayID := state.GatewayID.ValueString()

	gateway, err := r.client.GetGatewayById(gatewayID)
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading gateway enrollment token",
			"Couldn't read gateway "+gatewayID+" from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	// The token is deleted server-side when the method changes, so it really is gone.
	if gateway.AuthMethod.Method != infisical.GatewayAuthMethodToken {
		resp.Diagnostics.AddWarning(
			"Gateway no longer uses token authentication",
			fmt.Sprintf(
				"Gateway %q now authenticates with the %s method, which invalidated this enrollment token, so it has been removed from state. "+
					"Set `token_auth` back on the gateway to mint a new one, or drop this resource from your configuration.",
				gateway.Name, gateway.AuthMethod.Method,
			),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Unreachable while every configurable attribute forces replacement.
func (r *GatewayEnrollmentTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GatewayEnrollmentTokenResourceModel
	var state GatewayEnrollmentTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	plan.Token = state.Token
	plan.ExpiresAt = state.ExpiresAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// No-op: the only revoke is gateway-wide, and an unused token expires within the hour.
func (r *GatewayEnrollmentTokenResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
