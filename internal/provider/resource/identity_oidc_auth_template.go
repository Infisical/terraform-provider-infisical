package resource

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"
	infisicalstrings "terraform-provider-infisical/internal/pkg/strings"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &IdentityOidcAuthTemplateResource{}
	_ resource.ResourceWithConfigure   = &IdentityOidcAuthTemplateResource{}
	_ resource.ResourceWithImportState = &IdentityOidcAuthTemplateResource{}
)

const oidcWellKnownConfigurationSuffix = "/.well-known/openid-configuration"

func NewIdentityOidcAuthTemplateResource() resource.Resource {
	return &IdentityOidcAuthTemplateResource{}
}

type IdentityOidcAuthTemplateResource struct {
	client *infisical.Client
}

type IdentityOidcAuthTemplateResourceModel struct {
	ID               types.String                   `tfsdk:"id"`
	Name             types.String                   `tfsdk:"name"`
	OidcDiscoveryUrl types.String                   `tfsdk:"oidc_discovery_url"`
	BoundIssuer      types.String                   `tfsdk:"bound_issuer"`
	BoundAudiences   types.List                     `tfsdk:"bound_audiences"`
	CaCertificate    customtypes.TrimmedStringValue `tfsdk:"oidc_ca_certificate"`
}

func (r *IdentityOidcAuthTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_oidc_auth_template"
}

func (r *IdentityOidcAuthTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage an OIDC auth template in Infisical. A template holds the identity provider settings once, so many machine identities can share them by setting `template_id` on `infisical_identity_oidc_auth`. Editing a template propagates its settings to every identity linked to it. Auth templates are an Enterprise feature, and only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the auth template.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the auth template, at most 64 characters.",
				Required:    true,
				Validators:  identityAuthTemplateNameValidators,
			},
			"oidc_discovery_url": schema.StringAttribute{
				Description: "The URL used to retrieve the OpenID Connect configuration from the identity provider, without the `/.well-known/openid-configuration` suffix, which Infisical appends itself.",
				Required:    true,
				Validators:  []validator.String{oidcDiscoveryUrlValidator{}},
			},
			"bound_issuer": schema.StringAttribute{
				Description: "The unique identifier of the identity provider issuing the OIDC tokens.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 2048)},
			},
			"bound_audiences": schema.ListAttribute{
				Description: "The intended recipients that OIDC tokens must carry in their aud claim. Leave empty to accept any audience.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				// Audiences cross the wire as one comma-separated string.
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[^,\s](?:[^,]*[^,\s])?$`),
							"must be non-empty, contain no comma, and not begin or end with whitespace",
						),
					),
				},
			},
			"oidc_ca_certificate": schema.StringAttribute{
				// The API strips the trailing newline that file() always adds.
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The PEM-encoded CA certificate for establishing secure communication with the identity provider endpoints.",
				Optional:    true,
			},
		},
	}
}

// Rejects what the API would, since Infisical builds the configuration URL by appending the
// well-known suffix to this value.
type oidcDiscoveryUrlValidator struct{}

func (oidcDiscoveryUrlValidator) Description(_ context.Context) string {
	return "must be an absolute URL with no query, fragment or /.well-known/openid-configuration suffix"
}

func (v oidcDiscoveryUrlValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (oidcDiscoveryUrlValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	raw := strings.TrimSpace(req.ConfigValue.ValueString())
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid OIDC discovery URL", "The OIDC discovery URL must be an absolute URL, for example https://token.actions.githubusercontent.com.")
		return
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || strings.HasSuffix(raw, "?") || strings.HasSuffix(raw, "#") {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid OIDC discovery URL", "Remove the query string or fragment from the OIDC discovery URL. Infisical appends "+oidcWellKnownConfigurationSuffix+" to it, which would land inside them.")
		return
	}
	if strings.HasSuffix(strings.TrimRight(raw, "/"), oidcWellKnownConfigurationSuffix) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid OIDC discovery URL", "Remove the "+oidcWellKnownConfigurationSuffix+" suffix from the OIDC discovery URL. Infisical appends it automatically.")
	}
}

func (r *IdentityOidcAuthTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func applyOidcAuthTemplateToModel(ctx context.Context, model *IdentityOidcAuthTemplateResourceModel, template infisical.IdentityOidcAuthTemplate) diag.Diagnostics {
	fields := template.TemplateFields

	model.ID = types.StringValue(template.ID)
	model.Name = types.StringValue(template.Name)

	// The API strips trailing slashes, which is the same URL, so the configured spelling is kept.
	prior := model.OidcDiscoveryUrl
	if prior.IsNull() || prior.IsUnknown() || strings.TrimRight(strings.TrimSpace(prior.ValueString()), "/") != fields.OidcDiscoveryUrl {
		model.OidcDiscoveryUrl = types.StringValue(fields.OidcDiscoveryUrl)
	}

	model.BoundIssuer = types.StringValue(fields.BoundIssuer)
	model.CaCertificate = trimmedStringFromOptionalAPIValue(model.CaCertificate, fields.CaCert)

	boundAudiences, diags := types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(fields.BoundAudiences, ","))
	model.BoundAudiences = boundAudiences
	return diags
}

func (r *IdentityOidcAuthTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity oidc auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan IdentityOidcAuthTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	boundAudiences := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.BoundAudiences)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.CreateIdentityOidcAuthTemplate(infisical.CreateIdentityOidcAuthTemplateRequest{
		Name: plan.Name.ValueString(),
		TemplateFields: infisical.CreateIdentityOidcAuthTemplateFields{
			OidcDiscoveryUrl: strings.TrimSpace(plan.OidcDiscoveryUrl.ValueString()),
			BoundIssuer:      plan.BoundIssuer.ValueString(),
			BoundAudiences:   strings.Join(boundAudiences, ","),
			CaCert:           strings.TrimSpace(plan.CaCertificate.ValueString()),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity oidc auth template",
			"Couldn't create oidc auth template in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(applyOidcAuthTemplateToModel(ctx, &plan, template)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityOidcAuthTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read identity oidc auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityOidcAuthTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.GetIdentityOidcAuthTemplate(state.ID.ValueString())
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading identity oidc auth template",
			"Couldn't read oidc auth template with ID "+state.ID.ValueString()+" from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(applyOidcAuthTemplateToModel(ctx, &state, template)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Sends only what changed, since the API merges the patch into the stored fields and propagates
// it to every linked identity.
func oidcAuthTemplateFieldsPatch(ctx context.Context, diagnostics *diag.Diagnostics, plan, state IdentityOidcAuthTemplateResourceModel) map[string]any {
	fields := map[string]any{}

	if strings.TrimRight(strings.TrimSpace(plan.OidcDiscoveryUrl.ValueString()), "/") != strings.TrimRight(strings.TrimSpace(state.OidcDiscoveryUrl.ValueString()), "/") {
		fields["oidcDiscoveryUrl"] = strings.TrimSpace(plan.OidcDiscoveryUrl.ValueString())
	}
	if !plan.BoundIssuer.Equal(state.BoundIssuer) {
		fields["boundIssuer"] = plan.BoundIssuer.ValueString()
	}
	if !plan.BoundAudiences.Equal(state.BoundAudiences) {
		boundAudiences := infisicaltf.StringListToGoStringSlice(ctx, *diagnostics, plan.BoundAudiences)
		fields["boundAudiences"] = strings.Join(boundAudiences, ",")
	}
	if strings.TrimSpace(plan.CaCertificate.ValueString()) != strings.TrimSpace(state.CaCertificate.ValueString()) {
		fields["caCert"] = strings.TrimSpace(plan.CaCertificate.ValueString())
	}

	return fields
}

func (r *IdentityOidcAuthTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity oidc auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan, state IdentityOidcAuthTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := infisical.UpdateIdentityAuthTemplateRequest{
		ID:             state.ID.ValueString(),
		TemplateFields: oidcAuthTemplateFieldsPatch(ctx, &resp.Diagnostics, plan, state),
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Name.Equal(state.Name) {
		updateRequest.Name = infisicalstrings.StringToPtr(plan.Name.ValueString())
	}

	// A plan can differ only in spelling the API treats as equal, such as a trailing newline or
	// slash. The API rejects an empty update, so such a plan just refreshes from Infisical.
	var template infisical.IdentityOidcAuthTemplate
	var err error
	if updateRequest.Name == nil && len(updateRequest.TemplateFields) == 0 {
		template, err = r.client.GetIdentityOidcAuthTemplate(updateRequest.ID)
	} else {
		template, err = r.client.UpdateIdentityOidcAuthTemplate(updateRequest)
	}
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Identity oidc auth template not found",
				"OIDC auth template with ID "+state.ID.ValueString()+" no longer exists in Infisical. Run `terraform apply -refresh-only` to reconcile state, then apply again.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating identity oidc auth template",
			"Couldn't update oidc auth template in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(applyOidcAuthTemplateToModel(ctx, &plan, template)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityOidcAuthTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity oidc auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityOidcAuthTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Linked identities are unlinked by the API and keep the settings copied onto them.
	err := r.client.DeleteIdentityAuthTemplate(state.ID.ValueString())
	if err != nil && !errors.Is(err, infisical.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Error deleting identity oidc auth template",
			"Couldn't delete oidc auth template from Infisical, unexpected error: "+err.Error(),
		)
	}
}

func (r *IdentityOidcAuthTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import identity oidc auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
