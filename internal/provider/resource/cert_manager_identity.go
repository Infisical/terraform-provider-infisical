package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &certManagerIdentityResource{}
)

func NewCertManagerIdentityResource() resource.Resource {
	return &certManagerIdentityResource{}
}

type certManagerIdentityResource struct {
	client *infisical.Client
}

type certManagerIdentityResourceModel struct {
	Id           types.String `tfsdk:"id"`
	MembershipId types.String `tfsdk:"membership_id"`
	IdentityId   types.String `tfsdk:"identity_id"`
	Role         types.String `tfsdk:"role"`
}

func (r *certManagerIdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_identity"
}

func (r *certManagerIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage identity memberships in Certificate Manager. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the identity membership",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"membership_id": schema.StringAttribute{
				Description: "The ID of the identity membership",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identity_id": schema.StringAttribute{
				Description: "The ID of the identity",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "The role to assign to the identity (admin, member, or viewer)",
				Required:    true,
			},
		},
	}
}

func (r *certManagerIdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certManagerIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create Certificate Manager identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerIdentityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	added, err := r.client.AddCertManagerIdentity(infisical.AddCertManagerIdentityRequest{
		IdentityId: plan.IdentityId.ValueString(),
		Roles:      []infisical.CertManagerMembershipRoleUpdate{{Role: plan.Role.ValueString()}},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding identity to Certificate Manager",
			"Couldn't add identity to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.IdentityId = types.StringValue(added.IdentityMembership.IdentityId)

	refreshed, err := r.client.GetCertManagerIdentity(infisical.GetCertManagerIdentityRequest{
		IdentityId: plan.IdentityId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Certificate Manager identity",
			"Couldn't read identity membership from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(refreshed.IdentityMembership.ID)
	plan.MembershipId = types.StringValue(refreshed.IdentityMembership.ID)
	plan.Role = types.StringValue(firstRole(refreshed.IdentityMembership.Roles, plan.Role.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read Certificate Manager identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerIdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.client.GetCertManagerIdentity(infisical.GetCertManagerIdentityRequest{
		IdentityId: state.IdentityId.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading Certificate Manager identity",
			"Couldn't read identity membership from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.Id = types.StringValue(refreshed.IdentityMembership.ID)
	state.MembershipId = types.StringValue(refreshed.IdentityMembership.ID)
	state.Role = types.StringValue(firstRole(refreshed.IdentityMembership.Roles, state.Role.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update Certificate Manager identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerIdentityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateCertManagerIdentity(infisical.UpdateCertManagerIdentityRequest{
		IdentityId: plan.IdentityId.ValueString(),
		Roles:      []infisical.CertManagerMembershipRoleUpdate{{Role: plan.Role.ValueString()}},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Certificate Manager identity",
			"Couldn't update identity role in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	refreshed, err := r.client.GetCertManagerIdentity(infisical.GetCertManagerIdentityRequest{
		IdentityId: plan.IdentityId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Certificate Manager identity",
			"Couldn't read identity membership from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(refreshed.IdentityMembership.ID)
	plan.MembershipId = types.StringValue(refreshed.IdentityMembership.ID)
	plan.Role = types.StringValue(firstRole(refreshed.IdentityMembership.Roles, plan.Role.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete Certificate Manager identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerIdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RemoveCertManagerIdentity(infisical.RemoveCertManagerIdentityRequest{
		IdentityId: state.IdentityId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing Certificate Manager identity",
			"Couldn't remove identity from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *certManagerIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import Certificate Manager identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	refreshed, err := r.client.GetCertManagerIdentity(infisical.GetCertManagerIdentityRequest{
		IdentityId: req.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to import Certificate Manager identity",
			fmt.Sprintf("Couldn't find Certificate Manager identity for identity_id %q: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identity_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), refreshed.IdentityMembership.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("membership_id"), refreshed.IdentityMembership.ID)...)
}
