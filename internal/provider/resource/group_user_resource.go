package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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

var (
	_ resource.Resource                = &GroupUserResource{}
	_ resource.ResourceWithConfigure   = &GroupUserResource{}
	_ resource.ResourceWithImportState = &GroupUserResource{}
)

func NewGroupUserResource() resource.Resource {
	return &GroupUserResource{}
}

type GroupUserResource struct {
	client *infisical.Client
}

type GroupUserResourceModel struct {
	GroupID  types.String `tfsdk:"group_id"`
	UserID   types.String `tfsdk:"user_id"`
	Username types.String `tfsdk:"username"`
}

func (r *GroupUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_user"
}

func (r *GroupUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Add a user as a member of a group. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Description:   "The ID of the group.",
				Required:      true,
				Validators:    []validator.String{infisicaltf.UuidValidator},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"username": schema.StringAttribute{
				Description:   "The username of the user to add to the group. This is usually the user's email address, and it is case-sensitive.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^\S(.*\S)?$`), "must not be empty or start or end with whitespace")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"user_id": schema.StringAttribute{
				Description:   "The ID of the user.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *GroupUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create group user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan GroupUserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.AddGroupUser(infisical.AddGroupUserRequest{
		GroupID:  plan.GroupID.ValueString(),
		Username: plan.Username.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating group user membership",
			"Couldn't create group user membership, unexpected error: "+err.Error(),
		)
		return
	}

	plan.UserID = types.StringValue(user.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *GroupUserResource) findGroupUser(groupID, username string) (*infisical.GroupUser, error) {
	users, err := r.client.ListGroupUsers(infisical.ListGroupUsersRequest{
		GroupID:  groupID,
		Username: username,
	})

	if err != nil {
		return nil, err
	}

	for i := range users {
		if users[i].Username == username && users[i].IsPartOfGroup {
			return &users[i], nil
		}
	}

	return nil, nil
}

func (r *GroupUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read group user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GroupUserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.GroupID.ValueString() == "" || state.Username.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	user, err := r.findGroupUser(state.GroupID.ValueString(), state.Username.ValueString())
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading group user membership",
			"Couldn't read group user membership from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.UserID = types.StringValue(user.ID)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *GroupUserResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Both group_id and username are ForceNew; Update is never called.
}

func (r *GroupUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete group user",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state GroupUserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.findGroupUser(state.GroupID.ValueString(), state.Username.ValueString())
	if err != nil {
		if err == infisical.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting group user membership",
			"Couldn't list group users, unexpected error: "+err.Error(),
		)
		return
	}

	if user == nil {
		return
	}

	err = r.client.RemoveGroupUser(infisical.RemoveGroupUserRequest{
		GroupID:  state.GroupID.ValueString(),
		Username: state.Username.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting group user membership",
			"Couldn't delete group user membership, unexpected error: "+err.Error(),
		)
	}
}

func (r *GroupUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ",", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format <group_id>,<username>",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[1])...)
}
