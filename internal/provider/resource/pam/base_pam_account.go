package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &PamAccountBaseResource{}
	_ resource.ResourceWithConfigure      = &PamAccountBaseResource{}
	_ resource.ResourceWithImportState    = &PamAccountBaseResource{}
	_ resource.ResourceWithValidateConfig = &PamAccountBaseResource{}
)

// PamAccountBaseResource is the resource implementation shared by every PAM account type.
type PamAccountBaseResource struct {
	AccountType      string // the API's account type, e.g. "ssh"
	AccountTypeName  string // e.g. "SSH"
	ResourceTypeName string // terraform resource name suffix
	SupportsRotation bool
	Prerequisite     string // optional sentence added to the resource description
	client           *infisical.Client

	ConnectionDetailsAttributes map[string]schema.Attribute
	CredentialsAttributes       map[string]schema.Attribute

	ConnectionDetailsToAPI func(ctx context.Context, connectionDetails types.Object) (map[string]any, diag.Diagnostics)
	// Returns the type's connection details model.
	ConnectionDetailsFromAPI func(connectionDetails map[string]any) any

	CredentialsToAPI func(ctx context.Context, credentials types.Object) (map[string]any, diag.Diagnostics)
	// Returns the type's credentials model, taking secrets from prior since the API never returns them.
	CredentialsFromAPI func(ctx context.Context, credentials map[string]any, prior types.Object) (any, diag.Diagnostics)

	// Optional checks the schema can't express, like the fields an auth method requires.
	ValidateTypeConfig func(ctx context.Context, connectionDetails types.Object, credentials types.Object) diag.Diagnostics
}

type PamAccountBaseResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	FolderID          types.String `tfsdk:"folder_id"`
	TemplateID        types.String `tfsdk:"template_id"`
	GatewayID         types.String `tfsdk:"gateway_id"`
	GatewayPoolID     types.String `tfsdk:"gateway_pool_id"`
	ConnectionDetails types.Object `tfsdk:"connection_details"`
	Credentials       types.Object `tfsdk:"credentials"`
}

type pamAccountRotationModel struct {
	RotationAccountID types.String `tfsdk:"rotation_account_id"`
	SelfRotation      types.Bool   `tfsdk:"self_rotation"`
}

// The model of account types that support credential rotation.
type pamRotatableAccountResourceModel struct {
	PamAccountBaseResourceModel
	pamAccountRotationModel
}

type pamModelGetter interface {
	Get(ctx context.Context, target any) diag.Diagnostics
}

func (r *PamAccountBaseResource) getModel(ctx context.Context, source pamModelGetter) (PamAccountBaseResourceModel, pamAccountRotationModel, diag.Diagnostics) {
	if !r.SupportsRotation {
		var model PamAccountBaseResourceModel
		diags := source.Get(ctx, &model)
		return model, pamAccountRotationModel{RotationAccountID: types.StringNull(), SelfRotation: types.BoolNull()}, diags
	}

	var model pamRotatableAccountResourceModel
	diags := source.Get(ctx, &model)
	return model.PamAccountBaseResourceModel, model.pamAccountRotationModel, diags
}

func (r *PamAccountBaseResource) setModel(ctx context.Context, state *tfsdk.State, model PamAccountBaseResourceModel, rotation pamAccountRotationModel) diag.Diagnostics {
	if !r.SupportsRotation {
		return state.Set(ctx, model)
	}
	return state.Set(ctx, pamRotatableAccountResourceModel{PamAccountBaseResourceModel: model, pamAccountRotationModel: rotation})
}

func (r *PamAccountBaseResource) attrTypes(attributes map[string]schema.Attribute) map[string]attr.Type {
	result := make(map[string]attr.Type, len(attributes))
	for name, attribute := range attributes {
		result[name] = attribute.GetType()
	}
	return result
}

func (r *PamAccountBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

func (r *PamAccountBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   "The ID of the account.",
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			Description: "The name of the account, unique in its folder. " + pamNameDescription,
			Required:    true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 64),
				infisicaltf.SlugRegexValidator,
			},
		},
		"description": schema.StringAttribute{
			Description: "An optional description of the account.",
			Optional:    true,
			Validators:  []validator.String{stringvalidator.LengthBetween(1, 256)},
		},
		"folder_id": schema.StringAttribute{
			Description: "The ID of the folder the account is in. Changing it moves the account.",
			Required:    true,
			Validators:  []validator.String{infisicaltf.UuidValidator},
		},
		"template_id": schema.StringAttribute{
			Description: fmt.Sprintf("The ID of the account template the account uses, which must be a `%s` template.", r.AccountType),
			Required:    true,
			Validators:  []validator.String{infisicaltf.UuidValidator},
		},
		"gateway_id": schema.StringAttribute{
			Description: "The ID of the gateway to connect through instead of the template's.",
			Optional:    true,
			Validators: []validator.String{
				infisicaltf.UuidValidator,
				stringvalidator.ConflictsWith(path.MatchRoot("gateway_pool_id")),
			},
		},
		"gateway_pool_id": schema.StringAttribute{
			Description: "The ID of the gateway pool to connect through instead of the template's.",
			Optional:    true,
			Validators:  []validator.String{infisicaltf.UuidValidator},
		},
		"connection_details": schema.SingleNestedAttribute{
			Description: fmt.Sprintf("How to reach the %s target.", r.AccountTypeName),
			Required:    true,
			Attributes:  r.ConnectionDetailsAttributes,
		},
		"credentials": schema.SingleNestedAttribute{
			Description: "The credentials of the account. Secrets are never read back, so changes made outside of Terraform, such as rotation, aren't detected.",
			Required:    true,
			Attributes:  r.CredentialsAttributes,
		},
	}

	if r.SupportsRotation {
		attributes["rotation_account_id"] = schema.StringAttribute{
			Description: "The ID of another account of the same type and target that rotates this account's password. Rotation is enabled on the template.",
			Optional:    true,
			Validators:  []validator.String{infisicaltf.UuidValidator},
		}
		attributes["self_rotation"] = schema.BoolAttribute{
			Description: "Whether the account rotates its own password with its own credentials. Can't be combined with `rotation_account_id`. Defaults to false.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		}
	}

	description := fmt.Sprintf("Create and manage %s accounts in Infisical PAM. The connection is tested when the account is created or its connection changes.", r.AccountTypeName)
	if r.Prerequisite != "" {
		description += " " + r.Prerequisite
	}
	resp.Schema = schema.Schema{
		Description: description + " Only Machine Identity authentication is supported for this resource.",
		Attributes:  attributes,
	}
}

func (r *PamAccountBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configurePamClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *PamAccountBaseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	model, rotation, diags := r.getModel(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if rotation.SelfRotation.ValueBool() && !rotation.RotationAccountID.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("self_rotation"), "Conflicting rotation settings",
			"An account rotates either itself or through `rotation_account_id`, not both.")
	}

	if r.ValidateTypeConfig != nil && !model.ConnectionDetails.IsUnknown() && !model.Credentials.IsNull() && !model.Credentials.IsUnknown() {
		resp.Diagnostics.Append(r.ValidateTypeConfig(ctx, model.ConnectionDetails, model.Credentials)...)
	}
}

func desiredRotationAccountID(accountID string, rotation pamAccountRotationModel) *string {
	if rotation.SelfRotation.ValueBool() {
		return &accountID
	}
	return optionalString(rotation.RotationAccountID)
}

// The current value is read first since the API clears it when the connection or username changes.
func (r *PamAccountBaseResource) syncRotationAccount(accountID string, rotation pamAccountRotationModel) error {
	desired := desiredRotationAccountID(accountID, rotation)

	current, err := r.client.GetPamAccountRotation(infisical.GetPamAccountRotationRequest{AccountID: accountID})
	if err != nil {
		return err
	}
	if stringOrNull(current.RotationAccountID).Equal(stringOrNull(desired)) {
		return nil
	}

	return r.client.SetPamAccountRotationAccount(infisical.SetPamAccountRotationAccountRequest{
		AccountID:         accountID,
		RotationAccountID: desired,
	})
}

func (r *PamAccountBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, fmt.Sprintf("Unable to create %s PAM account", r.AccountTypeName)) {
		return
	}

	plan, rotation, diags := r.getModel(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectionDetails, diags := r.ConnectionDetailsToAPI(ctx, plan.ConnectionDetails)
	resp.Diagnostics.Append(diags...)
	credentials, diags := r.CredentialsToAPI(ctx, plan.Credentials)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	account, err := r.client.CreatePamAccount(infisical.CreatePamAccountRequest{
		AccountType:       r.AccountType,
		Name:              plan.Name.ValueString(),
		Description:       optionalString(plan.Description),
		FolderID:          plan.FolderID.ValueString(),
		TemplateID:        plan.TemplateID.ValueString(),
		ConnectionDetails: connectionDetails,
		Credentials:       credentials,
		GatewayID:         optionalString(plan.GatewayID),
		GatewayPoolID:     optionalString(plan.GatewayPoolID),
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error creating %s PAM account", r.AccountTypeName), "Couldn't create PAM account, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(account.ID)

	if r.SupportsRotation && desiredRotationAccountID(account.ID, rotation) != nil {
		if err := r.syncRotationAccount(account.ID, rotation); err != nil {
			// State is still saved, so Terraform taints the account instead of losing it.
			resp.Diagnostics.AddError("Error setting PAM account rotation account", "The account was created, but its rotation account couldn't be set, unexpected error: "+err.Error())
		}
	}

	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, plan, rotation)...)
}

func (r *PamAccountBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, fmt.Sprintf("Unable to read %s PAM account", r.AccountTypeName)) {
		return
	}

	state, rotation, diags := r.getModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	account, err := r.client.GetPamAccount(infisical.GetPamAccountRequest{ID: state.ID.ValueString()})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PAM account", "Couldn't read PAM account, unexpected error: "+err.Error())
		return
	}

	if account.AccountType != r.AccountType {
		resp.Diagnostics.AddError(
			"Unexpected PAM account type",
			fmt.Sprintf("PAM account %s has type %q, but infisical%s manages %q accounts.", account.ID, account.AccountType, r.ResourceTypeName, r.AccountType),
		)
		return
	}

	state.Name = types.StringValue(account.Name)
	state.Description = stringOrNull(account.Description)
	state.FolderID = stringOrNull(account.FolderID)
	state.TemplateID = types.StringValue(account.TemplateID)
	state.GatewayID = stringOrNull(account.GatewayID)
	state.GatewayPoolID = stringOrNull(account.GatewayPoolID)

	connectionDetails, diags := types.ObjectValueFrom(ctx, r.attrTypes(r.ConnectionDetailsAttributes), r.ConnectionDetailsFromAPI(account.ConnectionDetails))
	resp.Diagnostics.Append(diags...)
	credentialsModel, diags := r.CredentialsFromAPI(ctx, account.Credentials, state.Credentials)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	credentials, diags := types.ObjectValueFrom(ctx, r.attrTypes(r.CredentialsAttributes), credentialsModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ConnectionDetails, diags = keepTrimmedStrings(ctx, state.ConnectionDetails, connectionDetails)
	resp.Diagnostics.Append(diags...)
	state.Credentials, diags = keepTrimmedStrings(ctx, state.Credentials, credentials)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.SupportsRotation {
		current, err := r.client.GetPamAccountRotation(infisical.GetPamAccountRotationRequest{AccountID: account.ID})
		if err != nil {
			resp.Diagnostics.AddError("Error reading PAM account rotation", "Couldn't read PAM account rotation, unexpected error: "+err.Error())
			return
		}

		rotation.SelfRotation = types.BoolValue(current.RotationAccountID != nil && *current.RotationAccountID == account.ID)
		rotation.RotationAccountID = types.StringNull()
		if current.RotationAccountID != nil && *current.RotationAccountID != account.ID {
			rotation.RotationAccountID = types.StringValue(*current.RotationAccountID)
		}
	}

	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, state, rotation)...)
}

func (r *PamAccountBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, fmt.Sprintf("Unable to update %s PAM account", r.AccountTypeName)) {
		return
	}

	plan, rotation, diags := r.getModel(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	state, priorRotation, diags := r.getModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only changed fields: connection fields trigger a connection test and credentials overwrite a rotated password.
	body := map[string]any{}
	if !plan.Name.Equal(state.Name) {
		body["name"] = plan.Name.ValueString()
	}
	if !plan.Description.Equal(state.Description) {
		body["description"] = optionalString(plan.Description)
	}
	if !plan.FolderID.Equal(state.FolderID) {
		body["folderId"] = plan.FolderID.ValueString()
	}
	if !plan.TemplateID.Equal(state.TemplateID) {
		body["templateId"] = plan.TemplateID.ValueString()
	}
	if !plan.GatewayID.Equal(state.GatewayID) {
		body["gatewayId"] = optionalString(plan.GatewayID)
	}
	if !plan.GatewayPoolID.Equal(state.GatewayPoolID) {
		body["gatewayPoolId"] = optionalString(plan.GatewayPoolID)
	}
	if !plan.ConnectionDetails.Equal(state.ConnectionDetails) {
		connectionDetails, diags := r.ConnectionDetailsToAPI(ctx, plan.ConnectionDetails)
		resp.Diagnostics.Append(diags...)
		body["connectionDetails"] = connectionDetails
	}
	if !plan.Credentials.Equal(state.Credentials) {
		credentials, diags := r.CredentialsToAPI(ctx, plan.Credentials)
		resp.Diagnostics.Append(diags...)
		body["credentials"] = credentials
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if len(body) > 0 {
		_, err := r.client.UpdatePamAccount(infisical.UpdatePamAccountRequest{
			AccountType: r.AccountType,
			ID:          state.ID.ValueString(),
			Body:        body,
		})
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error updating %s PAM account", r.AccountTypeName), "Couldn't update PAM account, unexpected error: "+err.Error())
			return
		}
	}

	plan.ID = state.ID

	if r.SupportsRotation {
		if err := r.syncRotationAccount(state.ID.ValueString(), rotation); err != nil {
			// The account itself was updated, so its new values are saved.
			resp.Diagnostics.Append(r.setModel(ctx, &resp.State, plan, priorRotation)...)
			resp.Diagnostics.AddError("Error setting PAM account rotation account", "Couldn't set the account's rotation account, unexpected error: "+err.Error())
			return
		}
	}

	resp.Diagnostics.Append(r.setModel(ctx, &resp.State, plan, rotation)...)
}

func (r *PamAccountBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, fmt.Sprintf("Unable to delete %s PAM account", r.AccountTypeName)) {
		return
	}

	state, _, diags := r.getModel(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePamAccount(infisical.DeletePamAccountRequest{AccountType: r.AccountType, ID: state.ID.ValueString()})
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError(fmt.Sprintf("Error deleting %s PAM account", r.AccountTypeName), "Couldn't delete PAM account, unexpected error: "+err.Error())
	}
}

func (r *PamAccountBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
