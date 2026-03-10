package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &GroupMachineIdentityResource{}
	_ resource.ResourceWithImportState = &GroupMachineIdentityResource{}
)

func NewGroupMachineIdentityResource() resource.Resource {
	return &GroupMachineIdentityResource{}
}

type GroupMachineIdentityResource struct {
	client *infisical.Client
}

type GroupMachineIdentityResourceModel struct {
	GroupID    types.String `tfsdk:"group_id"`
	IdentityID types.String `tfsdk:"identity_id"`
}

func (r *GroupMachineIdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_machine_identity"
}

func (r *GroupMachineIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Add a machine identity as a member of a group. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Description:   "The ID of the group.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the machine identity.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *GroupMachineIdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *GroupMachineIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create group machine identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan GroupMachineIdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.AddGroupMachineIdentity(infisical.AddGroupMachineIdentityRequest{
		GroupID:    plan.GroupID.ValueString(),
		IdentityID: plan.IdentityID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding machine identity to group",
			"Couldn't add machine identity to group, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *GroupMachineIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read group machine identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GroupMachineIdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.GroupID.ValueString() == "" || state.IdentityID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	listResp, err := r.client.ListGroupMachineIdentities(infisical.ListGroupMachineIdentitiesRequest{
		GroupID: state.GroupID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading group machine identity",
			"Couldn't read group machine identity from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	for _, identity := range listResp.MachineIdentities {
		if identity.ID == state.IdentityID.ValueString() && identity.IsPartOfGroup {
			diags = resp.State.Set(ctx, state)
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *GroupMachineIdentityResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Both group_id and identity_id are ForceNew; Update is never called.
}

func (r *GroupMachineIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete group machine identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GroupMachineIdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check membership before attempting removal so that a 403 from the delete
	// endpoint unambiguously signals a permission error rather than "not a member".
	listResp, err := r.client.ListGroupMachineIdentities(infisical.ListGroupMachineIdentitiesRequest{
		GroupID: state.GroupID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			// Group no longer exists; desired state already achieved.
			return
		}
		resp.Diagnostics.AddError(
			"Error removing machine identity from group",
			"Couldn't list group machine identities, unexpected error: "+err.Error(),
		)
		return
	}

	isMember := false
	for _, identity := range listResp.MachineIdentities {
		if identity.ID == state.IdentityID.ValueString() && identity.IsPartOfGroup {
			isMember = true
			break
		}
	}
	if !isMember {
		// Identity is already not a member; nothing to do.
		return
	}

	_, err = r.client.RemoveGroupMachineIdentity(infisical.RemoveGroupMachineIdentityRequest{
		GroupID:    state.GroupID.ValueString(),
		IdentityID: state.IdentityID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing machine identity from group",
			"Couldn't remove machine identity from group, unexpected error: "+err.Error(),
		)
	}
}

func (r *GroupMachineIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format <group_id>,<identity_id>",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identity_id"), parts[1])...)
}
