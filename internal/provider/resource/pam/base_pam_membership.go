package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &PamMembershipBaseResource{}
	_ resource.ResourceWithConfigure   = &PamMembershipBaseResource{}
	_ resource.ResourceWithImportState = &PamMembershipBaseResource{}
)

type pamMembershipScope string

const (
	pamMembershipScopeProduct pamMembershipScope = "product"
	pamMembershipScopeFolder  pamMembershipScope = "folder"
	pamMembershipScopeAccount pamMembershipScope = "account"
)

var pamProductRoles = []string{"admin", "member"}

var pamResourceRoles = []string{"admin", "operator", "connector", "auditor"}

// PamMembershipBaseResource manages one kind of member in the PAM product, a folder or an account.
type PamMembershipBaseResource struct {
	Scope            pamMembershipScope
	Kind             infisical.PamMemberKind
	ResourceTypeName string // terraform resource name suffix
	client           *infisical.Client
}

// Read and written per attribute, since the schema depends on the scope and kind.
type pamMembershipValues struct {
	ID       types.String
	ParentID types.String // the folder or account ID, null in the product scope
	MemberID types.String // the user, group or identity ID
	Email    types.String // users only
	Role     types.String
}

type pamAttributeGetter interface {
	GetAttribute(ctx context.Context, attributePath path.Path, target any) diag.Diagnostics
}

func (r *PamMembershipBaseResource) parentAttribute() string {
	switch r.Scope {
	case pamMembershipScopeFolder:
		return "folder_id"
	case pamMembershipScopeAccount:
		return "account_id"
	}
	return ""
}

func (r *PamMembershipBaseResource) memberAttribute() string {
	switch r.Kind {
	case infisical.PamMemberKindUser:
		return "user_id"
	case infisical.PamMemberKindGroup:
		return "group_id"
	}
	return "identity_id"
}

func (r *PamMembershipBaseResource) kindName() string {
	switch r.Kind {
	case infisical.PamMemberKindUser:
		return "user"
	case infisical.PamMemberKindGroup:
		return "group"
	}
	return "machine identity"
}

func (r *PamMembershipBaseResource) scopeName() string {
	switch r.Scope {
	case pamMembershipScopeFolder:
		return "PAM folder"
	case pamMembershipScopeAccount:
		return "PAM account"
	}
	return "PAM"
}

func (r *PamMembershipBaseResource) apiScope(parentID string) infisical.PamMembershipScope {
	switch r.Scope {
	case pamMembershipScopeFolder:
		return infisical.PamMembershipScope{Resource: infisical.PamMembershipResourceFolder, ResourceID: parentID}
	case pamMembershipScopeAccount:
		return infisical.PamMembershipScope{Resource: infisical.PamMembershipResourceAccount, ResourceID: parentID}
	}
	return infisical.PamMembershipScope{}
}

func (r *PamMembershipBaseResource) getValues(ctx context.Context, source pamAttributeGetter) (pamMembershipValues, diag.Diagnostics) {
	var diags diag.Diagnostics
	values := pamMembershipValues{ParentID: types.StringNull(), Email: types.StringNull()}

	diags.Append(source.GetAttribute(ctx, path.Root("id"), &values.ID)...)
	diags.Append(source.GetAttribute(ctx, path.Root(r.memberAttribute()), &values.MemberID)...)
	diags.Append(source.GetAttribute(ctx, path.Root("role"), &values.Role)...)
	if r.Scope != pamMembershipScopeProduct {
		diags.Append(source.GetAttribute(ctx, path.Root(r.parentAttribute()), &values.ParentID)...)
	}
	if r.Kind == infisical.PamMemberKindUser {
		diags.Append(source.GetAttribute(ctx, path.Root("email"), &values.Email)...)
	}
	return values, diags
}

func (r *PamMembershipBaseResource) setValues(ctx context.Context, state *tfsdk.State, values pamMembershipValues) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(state.SetAttribute(ctx, path.Root("id"), values.ID)...)
	diags.Append(state.SetAttribute(ctx, path.Root(r.memberAttribute()), values.MemberID)...)
	diags.Append(state.SetAttribute(ctx, path.Root("role"), values.Role)...)
	if r.Scope != pamMembershipScopeProduct {
		diags.Append(state.SetAttribute(ctx, path.Root(r.parentAttribute()), values.ParentID)...)
	}
	if r.Kind == infisical.PamMemberKindUser {
		diags.Append(state.SetAttribute(ctx, path.Root("email"), values.Email)...)
	}
	return diags
}

func (r *PamMembershipBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

func (r *PamMembershipBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   "The ID of the membership.",
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}

	var description string
	switch r.Scope {
	case pamMembershipScopeProduct:
		description = fmt.Sprintf("Manage a %s's membership in Infisical PAM. A %s must be a PAM member before it can be added to folders or accounts.", r.kindName(), r.kindName())
		attributes["role"] = schema.StringAttribute{
			Description: "The role in PAM: `admin` or `member`. Members only get the access their folder and account memberships grant.",
			Required:    true,
			Validators:  []validator.String{stringvalidator.OneOf(pamProductRoles...)},
		}
	default:
		subject := "every account in the folder"
		if r.Scope == pamMembershipScopeAccount {
			subject = "the account"
		}
		description = fmt.Sprintf("Manage a %s's membership in a %s, which grants its role on %s. The %s must already be a PAM member.", r.kindName(), r.scopeName(), subject, r.kindName())
		attributes[r.parentAttribute()] = schema.StringAttribute{
			Description:   fmt.Sprintf("The ID of the %s. Changing it creates a new membership.", r.scopeName()),
			Required:      true,
			Validators:    []validator.String{infisicaltf.UuidValidator},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		}
		attributes["role"] = schema.StringAttribute{
			Description: fmt.Sprintf("The role on %s: `admin`, `operator` (sessions and credentials), `connector` (sessions only) or `auditor` (audit logs and recordings).", subject),
			Required:    true,
			Validators:  []validator.String{stringvalidator.OneOf(pamResourceRoles...)},
		}
	}

	if r.Kind == infisical.PamMemberKindUser {
		attributes["email"] = schema.StringAttribute{
			Description: "The email of the user, in lowercase. Changing it creates a new membership.",
			Required:    true,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
				stringvalidator.RegexMatches(regexp.MustCompile(`^[^A-Z]+$`), "must be lowercase"),
			},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		}
		attributes["user_id"] = schema.StringAttribute{
			Description:   "The ID of the user.",
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	} else {
		attributes[r.memberAttribute()] = schema.StringAttribute{
			Description:   fmt.Sprintf("The ID of the %s. Changing it creates a new membership.", r.kindName()),
			Required:      true,
			Validators:    []validator.String{infisicaltf.UuidValidator},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		}
	}

	resp.Schema = schema.Schema{
		Description: description + " Only Machine Identity authentication is supported for this resource.",
		Attributes:  attributes,
	}
}

func (r *PamMembershipBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configurePamClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *PamMembershipBaseResource) resolveUserID(email string) (string, error) {
	userID, err := r.client.FindPamProductUserByEmail(email)
	if err == infisical.ErrNotFound {
		return "", fmt.Errorf("no PAM member has the email %q. Add the user to PAM first, with infisical_pam_user for instance", email)
	}
	return userID, err
}

func (r *PamMembershipBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	summary := fmt.Sprintf("Error adding %s to %s", r.kindName(), r.scopeName())
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, summary) {
		return
	}

	plan, diags := r.getValues(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var member infisical.PamMember
	var err error
	if r.Kind == infisical.PamMemberKindUser && r.Scope == pamMembershipScopeProduct {
		member, err = r.addProductUser(plan.Email.ValueString(), plan.Role.ValueString())
	} else {
		if r.Kind == infisical.PamMemberKindUser {
			var userID string
			userID, err = r.resolveUserID(plan.Email.ValueString())
			plan.MemberID = types.StringValue(userID)
		}
		if err == nil {
			member, err = r.client.AddPamMember(infisical.AddPamMemberRequest{
				Scope:    r.apiScope(plan.ParentID.ValueString()),
				Kind:     r.Kind,
				MemberID: plan.MemberID.ValueString(),
				Role:     plan.Role.ValueString(),
			})
		}
	}
	if err != nil {
		resp.Diagnostics.AddError(summary, fmt.Sprintf("Couldn't add the %s, unexpected error: %s", r.kindName(), err.Error()))
		return
	}

	plan.ID = types.StringValue(member.MembershipID)
	if r.Kind == infisical.PamMemberKindUser && member.UserID != nil {
		plan.MemberID = types.StringValue(*member.UserID)
	}
	if plan.MemberID.IsUnknown() {
		resp.Diagnostics.AddError(summary, "The user was added, but Infisical didn't return their user ID.")
		return
	}
	resp.Diagnostics.Append(r.setValues(ctx, &resp.State, plan)...)
}

// The API skips existing members instead of failing, so that case is reported as an error.
func (r *PamMembershipBaseResource) addProductUser(email, role string) (infisical.PamMember, error) {
	added, err := r.client.AddPamProductUsers(infisical.AddPamProductUsersRequest{
		Emails: []string{strings.ToLower(email)},
		Role:   role,
	})
	if err != nil {
		return infisical.PamMember{}, err
	}
	if len(added.Memberships) == 0 {
		return infisical.PamMember{}, fmt.Errorf("%s is already a PAM member. Import the membership with `terraform import infisical%s.<name> %s` to manage it", email, r.ResourceTypeName, email)
	}
	return added.Memberships[0], nil
}

func (r *PamMembershipBaseResource) findMember(values pamMembershipValues) (*infisical.PamMember, error) {
	members, err := r.client.ListPamMembers(infisical.ListPamMembersRequest{
		Scope: r.apiScope(values.ParentID.ValueString()),
		Kind:  r.Kind,
	})
	if err != nil {
		return nil, err
	}

	memberID := values.MemberID.ValueString()
	for i := range members.Members {
		member := &members.Members[i]
		var id *string
		switch r.Kind {
		case infisical.PamMemberKindUser:
			id = member.UserID
		case infisical.PamMemberKindGroup:
			id = member.GroupID
		default:
			id = member.IdentityID
		}
		// An expired temporary membership grants nothing, so it reads as gone and the next apply adds it again.
		if id != nil && *id == memberID && (member.ExpiresAt == nil || member.ExpiresAt.After(time.Now())) {
			return member, nil
		}
	}
	return nil, nil
}

func (r *PamMembershipBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	summary := fmt.Sprintf("Error reading %s membership", r.scopeName())
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, summary) {
		return
	}

	state, diags := r.getValues(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.findMember(state)
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(summary, "Couldn't read the membership, unexpected error: "+err.Error())
		return
	}
	if member == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(member.MembershipID)
	state.Role = types.StringValue(member.Role)
	resp.Diagnostics.Append(r.setValues(ctx, &resp.State, state)...)
}

func (r *PamMembershipBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	summary := fmt.Sprintf("Error updating %s membership", r.scopeName())
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, summary) {
		return
	}

	plan, diags := r.getValues(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	state, diags := r.getValues(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdatePamMemberRole(infisical.UpdatePamMemberRoleRequest{
		Scope:    r.apiScope(state.ParentID.ValueString()),
		Kind:     r.Kind,
		MemberID: state.MemberID.ValueString(),
		Role:     plan.Role.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(summary, "Couldn't update the membership's role, unexpected error: "+err.Error())
		return
	}

	state.Role = plan.Role
	resp.Diagnostics.Append(r.setValues(ctx, &resp.State, state)...)
}

func (r *PamMembershipBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	summary := fmt.Sprintf("Error removing %s from %s", r.kindName(), r.scopeName())
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, summary) {
		return
	}

	state, diags := r.getValues(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemovePamMember(infisical.RemovePamMemberRequest{
		Scope:    r.apiScope(state.ParentID.ValueString()),
		Kind:     r.Kind,
		MemberID: state.MemberID.ValueString(),
	})
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError(summary, "Couldn't remove the membership, unexpected error: "+err.Error())
	}
}

func (r *PamMembershipBaseResource) importFormat() string {
	member := "<" + r.memberAttribute() + ">"
	if r.Kind == infisical.PamMemberKindUser {
		member = "<email>"
	}
	if r.Scope == pamMembershipScopeProduct {
		return member
	}
	return "<" + r.parentAttribute() + ">:" + member
}

func (r *PamMembershipBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, fmt.Sprintf("Unable to import %s membership", r.scopeName())) {
		return
	}

	parentID, memberKey := "", req.ID
	if r.Scope != pamMembershipScopeProduct {
		parts := strings.Split(req.ID, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError(
				"Unexpected Import Identifier",
				fmt.Sprintf("Expected import identifier in the format %s, got: %s", r.importFormat(), req.ID),
			)
			return
		}
		parentID, memberKey = parts[0], parts[1]
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.parentAttribute()), parentID)...)
	}

	memberID := memberKey
	if r.Kind == infisical.PamMemberKindUser {
		userID, err := r.resolveUserID(memberKey)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Unable to import %s membership", r.scopeName()), err.Error())
			return
		}
		memberID = userID
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), strings.ToLower(memberKey))...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.memberAttribute()), memberID)...)
}
