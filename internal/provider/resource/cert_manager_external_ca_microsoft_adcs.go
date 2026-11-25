package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/go-uuid"
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
	_ resource.Resource                = &certManagerExternalCAMicrosoftADCSResource{}
	_ resource.ResourceWithImportState = &certManagerExternalCAMicrosoftADCSResource{}
)

// The Gateway-based Microsoft ADCS CA (API CA type "adcs"). It is a separate resource from
// infisical_cert_manager_external_ca_adcs, which manages the older Azure ADCS web-enrollment CA
// (API CA type "azure-ad-cs") and already owns the "adcs" resource name.
func NewCertManagerExternalCAMicrosoftADCSResource() resource.Resource {
	return &certManagerExternalCAMicrosoftADCSResource{}
}

type certManagerExternalCAMicrosoftADCSResource struct {
	client *infisical.Client
}

type certManagerExternalCAMicrosoftADCSResourceModel struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Status          types.String `tfsdk:"status"`
	AppConnectionId types.String `tfsdk:"app_connection_id"`
	CaName          types.String `tfsdk:"ca_name"`
}

func (r *certManagerExternalCAMicrosoftADCSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cert_manager_external_ca_microsoft_adcs"
}

func (r *certManagerExternalCAMicrosoftADCSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage external Microsoft ADCS (Active Directory Certificate Services) certificate authorities in Certificate Manager. " +
			"Infisical reaches the AD CS server through the Gateway of an `infisical_app_connection_adcs` connection. " +
			"For the older Azure ADCS web-enrollment CA, use `infisical_cert_manager_external_ca_adcs`. " +
			"Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Microsoft ADCS CA",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Microsoft ADCS CA. Must be 1 to 64 lowercase letters, numbers and hyphens.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					infisicaltf.SlugRegexValidator,
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the CA. Supported values: " + strings.Join(SUPPORTED_CA_STATUSES, ", ") + ". Defaults to 'active'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("active"),
				Validators: []validator.String{
					stringvalidator.OneOf(SUPPORTED_CA_STATUSES...),
				},
			},
			"app_connection_id": schema.StringAttribute{
				Description: "The ID of the `infisical_app_connection_adcs` connection used to reach the AD CS server",
				Required:    true,
				Validators:  []validator.String{infisicaltf.UuidValidator},
			},
			"ca_name": schema.StringAttribute{
				Description: "The AD CS certification authority name, for example `corp-ca01-CA`. Run `certutil -getreg CA\\CommonName` on the AD CS server to find it. " +
					"If omitted, Infisical reads it from the CA host's registry over SMB (TCP 445) when the CA is created and whenever `app_connection_id` changes. " +
					"Removing a `ca_name` you set keeps the current name. " +
					"On create, and whenever `ca_name` or `app_connection_id` changes, Infisical checks through the Gateway that it can list certificate templates on the CA, and rejects the change if it cannot.",
				Optional: true,
				Computed: true,
				// The API trims ca_name before its minimum length check, so whitespace alone fails at apply.
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must contain a character other than whitespace"),
				},
				PlanModifiers: []planmodifier.String{caNameUseStateUnlessConnectionChanges{}},
			},
		},
	}
}

// When ca_name is not configured, the API discovers it again whenever the connection changes.
// Keeping the old name in the plan then would fail Terraform's consistency check, so the value is
// only carried over from state while app_connection_id stays the same.
type caNameUseStateUnlessConnectionChanges struct{}

func (m caNameUseStateUnlessConnectionChanges) Description(_ context.Context) string {
	return "Keeps the prior ca_name when ca_name is not configured and app_connection_id does not change."
}

func (m caNameUseStateUnlessConnectionChanges) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m caNameUseStateUnlessConnectionChanges) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || !req.PlanValue.IsUnknown() || req.ConfigValue.IsUnknown() {
		return
	}

	var planConnectionId, stateConnectionId types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("app_connection_id"), &planConnectionId)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("app_connection_id"), &stateConnectionId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if planConnectionId.Equal(stateConnectionId) {
		resp.PlanValue = req.StateValue
	}
}

func (r *certManagerExternalCAMicrosoftADCSResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// An unknown ca_name means the user left it out and the API must discover it, so it is omitted
// from the request. A known value, including one carried over from state, is sent so that an
// update does not trigger a new discovery.
func microsoftADCSCAConfigurationFromPlan(plan certManagerExternalCAMicrosoftADCSResourceModel) infisical.MicrosoftADCSCAConfiguration {
	configuration := infisical.MicrosoftADCSCAConfiguration{
		AppConnectionId: plan.AppConnectionId.ValueString(),
	}

	if !plan.CaName.IsNull() && !plan.CaName.IsUnknown() {
		caName := plan.CaName.ValueString()
		configuration.CaName = &caName
	}

	return configuration
}

// The API trims ca_name before comparing, so a change in whitespace alone is not a change. An
// unknown ca_name is one the API must discover again.
func microsoftADCSCAConfigurationChanged(plan, state certManagerExternalCAMicrosoftADCSResourceModel) bool {
	return !plan.AppConnectionId.Equal(state.AppConnectionId) ||
		plan.CaName.IsUnknown() ||
		strings.TrimSpace(plan.CaName.ValueString()) != strings.TrimSpace(state.CaName.ValueString())
}

// The API trims ca_name. Keeping the configured spelling when only whitespace differs avoids a
// permanent diff and Terraform's inconsistent-result error.
func applyMicrosoftADCSCAToModel(model *certManagerExternalCAMicrosoftADCSResourceModel, ca infisical.MicrosoftADCSCA) {
	model.Id = types.StringValue(ca.Id)
	model.Name = types.StringValue(ca.Name)
	model.Status = types.StringValue(ca.Status)
	model.AppConnectionId = types.StringValue(ca.Configuration.AppConnectionId)

	caName := ""
	if ca.Configuration.CaName != nil {
		caName = *ca.Configuration.CaName
	}
	model.CaName = keepIfOnlyWhitespaceDiffers(model.CaName, caName)
}

func (r *certManagerExternalCAMicrosoftADCSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create Microsoft ADCS CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerExternalCAMicrosoftADCSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newCA, err := r.client.CreateMicrosoftADCSCA(infisical.CreateMicrosoftADCSCARequest{
		Name:          plan.Name.ValueString(),
		Status:        plan.Status.ValueString(),
		Configuration: microsoftADCSCAConfigurationFromPlan(plan),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Microsoft ADCS CA",
			"Couldn't create Microsoft ADCS CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	applyMicrosoftADCSCAToModel(&plan, newCA)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerExternalCAMicrosoftADCSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read Microsoft ADCS CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerExternalCAMicrosoftADCSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Id.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	ca, err := r.client.GetMicrosoftADCSCA(state.Id.ValueString())
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading Microsoft ADCS CA",
			"Couldn't read Microsoft ADCS CA from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	applyMicrosoftADCSCAToModel(&state, ca)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *certManagerExternalCAMicrosoftADCSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update Microsoft ADCS CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan certManagerExternalCAMicrosoftADCSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certManagerExternalCAMicrosoftADCSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := infisical.UpdateMicrosoftADCSCARequest{
		CAId:   state.Id.ValueString(),
		Name:   plan.Name.ValueString(),
		Status: plan.Status.ValueString(),
	}

	// When configuration is sent, the API requires permission to use the app connection and checks
	// that the CA is reachable. Leaving it out lets an identity that can only edit CAs rename or
	// disable this one.
	if microsoftADCSCAConfigurationChanged(plan, state) {
		configuration := microsoftADCSCAConfigurationFromPlan(plan)
		request.Configuration = &configuration
	}

	updatedCA, err := r.client.UpdateMicrosoftADCSCA(request)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Microsoft ADCS CA",
			"Couldn't update Microsoft ADCS CA in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	applyMicrosoftADCSCAToModel(&plan, updatedCA)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *certManagerExternalCAMicrosoftADCSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete Microsoft ADCS CA",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state certManagerExternalCAMicrosoftADCSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMicrosoftADCSCA(state.Id.ValueString())
	// Already gone, which is what the delete wanted.
	if err == infisical.ErrNotFound {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Microsoft ADCS CA",
			"Couldn't delete Microsoft ADCS CA from Infisical, unexpected error: "+err.Error(),
		)
	}
}

func (r *certManagerExternalCAMicrosoftADCSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.ParseUUID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected the ID of a Microsoft ADCS CA, such as 3f091b42-1661-41a1-b647-98c6a45af047, got: %q", req.ID),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
