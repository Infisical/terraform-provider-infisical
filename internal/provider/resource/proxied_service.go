package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	SUPPORTED_PROXIED_SERVICE_CREDENTIAL_ROLES = []string{"header-rewrite", "credential-substitution"}
	SUPPORTED_PROXIED_SERVICE_HEADER_PURPOSES  = []string{"username", "password"}
	SUPPORTED_PROXIED_SERVICE_SURFACES         = []string{"header", "path", "query", "body"}
)

var (
	_ resource.Resource                = &proxiedServiceResource{}
	_ resource.ResourceWithImportState = &proxiedServiceResource{}
)

func NewProxiedServiceResource() resource.Resource {
	return &proxiedServiceResource{}
}

type proxiedServiceResource struct {
	client *infisical.Client
}

type proxiedServiceCredentialModel struct {
	SecretKey            types.String `tfsdk:"secret_key"`
	DynamicSecretName    types.String `tfsdk:"dynamic_secret_name"`
	DynamicSecretField   types.String `tfsdk:"dynamic_secret_field"`
	Role                 types.String `tfsdk:"role"`
	HeaderName           types.String `tfsdk:"header_name"`
	HeaderPrefix         types.String `tfsdk:"header_prefix"`
	HeaderPurpose        types.String `tfsdk:"header_purpose"`
	PlaceholderKey       types.String `tfsdk:"placeholder_key"`
	PlaceholderValue     types.String `tfsdk:"placeholder_value"`
	SubstitutionSurfaces types.List   `tfsdk:"substitution_surfaces"`
}

type proxiedServiceResourceModel struct {
	Id          types.String                    `tfsdk:"id"`
	ProjectId   types.String                    `tfsdk:"project_id"`
	Environment types.String                    `tfsdk:"environment"`
	SecretPath  types.String                    `tfsdk:"secret_path"`
	Name        types.String                    `tfsdk:"name"`
	HostPattern types.String                    `tfsdk:"host_pattern"`
	IsEnabled   types.Bool                      `tfsdk:"is_enabled"`
	Credentials []proxiedServiceCredentialModel `tfsdk:"credentials"`
}

func (r *proxiedServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_proxied_service"
}

func (r *proxiedServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage proxied services for the Infisical Agent Proxy. A proxied service maps a host pattern to the credentials the Agent Proxy applies to matching requests. Credentials reference secrets by key, so no secret values are stored in Terraform state. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the proxied service",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project the proxied service belongs to",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				Description: "The slug of the environment the proxied service belongs to",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path (folder) the proxied service belongs to. Defaults to '/'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the proxied service. Must be a slug, unique within the folder.",
				Required:    true,
			},
			"host_pattern": schema.StringAttribute{
				Description: "Comma-separated hosts this service matches, without a scheme. Supports a leading wildcard label, an optional port, and an optional path prefix (for example 'api.github.com', '*.example.com:8443,internal.example.com/v1').",
				Required:    true,
			},
			"is_enabled": schema.BoolAttribute{
				Description: "Whether the Agent Proxy applies this service. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"credentials": schema.SetNestedAttribute{
				Description: "The credentials this service applies to matching requests. At least one is required. This is a set: the API does not preserve credential ordering, so order in configuration is not meaningful.",
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"secret_key": schema.StringAttribute{
							Description: "The key of the static secret holding the real value. Exactly one of secret_key or dynamic_secret_name must be set.",
							Optional:    true,
						},
						"dynamic_secret_name": schema.StringAttribute{
							Description: "The name of the dynamic secret to lease the real value from. Exactly one of secret_key or dynamic_secret_name must be set.",
							Optional:    true,
						},
						"dynamic_secret_field": schema.StringAttribute{
							Description: "The field of the dynamic secret lease output to use. Required when dynamic_secret_name is set.",
							Optional:    true,
						},
						"role": schema.StringAttribute{
							Description: "How the credential is applied. Supported values: " + strings.Join(SUPPORTED_PROXIED_SERVICE_CREDENTIAL_ROLES, ", ") + ".",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(SUPPORTED_PROXIED_SERVICE_CREDENTIAL_ROLES...),
							},
						},
						"header_name": schema.StringAttribute{
							Description: "The header the Agent Proxy sets. Used with the 'header-rewrite' role, and mutually exclusive with header_purpose.",
							Optional:    true,
						},
						"header_prefix": schema.StringAttribute{
							Description: "A prefix prepended to the value in the header, for example 'Bearer'. The Agent Proxy inserts a single space between the prefix and the value, and the API trims surrounding whitespace, so do not include a trailing space. Used with header_name.",
							Optional:    true,
						},
						"header_purpose": schema.StringAttribute{
							Description: "Populates basic auth instead of a named header. Supported values: " + strings.Join(SUPPORTED_PROXIED_SERVICE_HEADER_PURPOSES, ", ") + ". A password credential requires a username credential on the same service, and basic auth cannot be combined with other header-rewrite credentials.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(SUPPORTED_PROXIED_SERVICE_HEADER_PURPOSES...),
							},
						},
						"placeholder_key": schema.StringAttribute{
							Description: "The environment variable the agent receives holding the placeholder. Required for the 'credential-substitution' role, and unique within the service.",
							Optional:    true,
						},
						"placeholder_value": schema.StringAttribute{
							Description: "The fake value handed to the agent, swapped for the real one as requests pass through. Required for the 'credential-substitution' role, and unique within the service.",
							Optional:    true,
						},
						"substitution_surfaces": schema.ListAttribute{
							Description: "Where in the request the placeholder is swapped. Supported values: " + strings.Join(SUPPORTED_PROXIED_SERVICE_SURFACES, ", ") + ". Required for the 'credential-substitution' role.",
							Optional:    true,
							ElementType: types.StringType,
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
								listvalidator.ValueStringsAre(stringvalidator.OneOf(SUPPORTED_PROXIED_SERVICE_SURFACES...)),
							},
						},
					},
				},
			},
		},
	}
}

func (r *proxiedServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func stringPtrToValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func (r *proxiedServiceResource) buildCredentials(ctx context.Context, plan *proxiedServiceResourceModel, diags *diag.Diagnostics) []infisical.ProxiedServiceCredential {
	credentials := make([]infisical.ProxiedServiceCredential, 0, len(plan.Credentials))

	for _, cred := range plan.Credentials {
		out := infisical.ProxiedServiceCredential{
			Role:               cred.Role.ValueString(),
			SecretKey:          optionalString(cred.SecretKey),
			DynamicSecretName:  optionalString(cred.DynamicSecretName),
			DynamicSecretField: optionalString(cred.DynamicSecretField),
			HeaderName:         optionalString(cred.HeaderName),
			HeaderPrefix:       optionalString(cred.HeaderPrefix),
			HeaderPurpose:      optionalString(cred.HeaderPurpose),
			PlaceholderKey:     optionalString(cred.PlaceholderKey),
			PlaceholderValue:   optionalString(cred.PlaceholderValue),
		}

		if !cred.SubstitutionSurfaces.IsNull() && !cred.SubstitutionSurfaces.IsUnknown() {
			surfaces := make([]string, 0, len(cred.SubstitutionSurfaces.Elements()))
			diags.Append(cred.SubstitutionSurfaces.ElementsAs(ctx, &surfaces, false)...)
			out.SubstitutionSurfaces = surfaces
		}

		credentials = append(credentials, out)
	}

	return credentials
}

func (r *proxiedServiceResource) credentialsToState(ctx context.Context, src []infisical.ProxiedServiceCredential, diags *diag.Diagnostics) []proxiedServiceCredentialModel {
	credentials := make([]proxiedServiceCredentialModel, 0, len(src))

	for _, cred := range src {
		out := proxiedServiceCredentialModel{
			Role:               types.StringValue(cred.Role),
			SecretKey:          stringPtrToValue(cred.SecretKey),
			DynamicSecretName:  stringPtrToValue(cred.DynamicSecretName),
			DynamicSecretField: stringPtrToValue(cred.DynamicSecretField),
			HeaderName:         stringPtrToValue(cred.HeaderName),
			HeaderPrefix:       stringPtrToValue(cred.HeaderPrefix),
			HeaderPurpose:      stringPtrToValue(cred.HeaderPurpose),
			PlaceholderKey:     stringPtrToValue(cred.PlaceholderKey),
			PlaceholderValue:   stringPtrToValue(cred.PlaceholderValue),
		}

		if len(cred.SubstitutionSurfaces) > 0 {
			l, d := types.ListValueFrom(ctx, types.StringType, cred.SubstitutionSurfaces)
			diags.Append(d...)
			out.SubstitutionSurfaces = l
		} else {
			out.SubstitutionSurfaces = types.ListNull(types.StringType)
		}

		credentials = append(credentials, out)
	}

	return credentials
}

func (r *proxiedServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create proxied service",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan proxiedServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentials := r.buildCredentials(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	isEnabled := plan.IsEnabled.ValueBool()
	service, err := r.client.CreateProxiedService(infisical.CreateProxiedServiceRequest{
		ProjectId:   plan.ProjectId.ValueString(),
		Environment: plan.Environment.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
		Name:        plan.Name.ValueString(),
		HostPattern: plan.HostPattern.ValueString(),
		IsEnabled:   &isEnabled,
		Credentials: credentials,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating proxied service", err.Error())
		return
	}

	plan.Id = types.StringValue(service.Service.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *proxiedServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read proxied service",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state proxiedServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.client.GetProxiedService(infisical.GetProxiedServiceRequest{
		ServiceId: state.Id.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading proxied service", err.Error())
		return
	}

	// project_id, environment, and secret_path are carried over from prior state: the API
	// returns the service's folder id rather than the folder scope it was created with.
	state.Id = types.StringValue(service.Service.Id)
	state.Name = types.StringValue(service.Service.Name)
	state.HostPattern = types.StringValue(service.Service.HostPattern)
	state.IsEnabled = types.BoolValue(service.Service.IsEnabled)
	state.Credentials = r.credentialsToState(ctx, service.Service.Credentials, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *proxiedServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update proxied service",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan proxiedServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state proxiedServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentials := r.buildCredentials(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	isEnabled := plan.IsEnabled.ValueBool()
	_, err := r.client.UpdateProxiedService(infisical.UpdateProxiedServiceRequest{
		ServiceId:   state.Id.ValueString(),
		Name:        plan.Name.ValueString(),
		HostPattern: plan.HostPattern.ValueString(),
		IsEnabled:   &isEnabled,
		Credentials: credentials,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating proxied service", err.Error())
		return
	}

	plan.Id = state.Id

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *proxiedServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete proxied service",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state proxiedServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProxiedService(infisical.DeleteProxiedServiceRequest{
		ServiceId: state.Id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting proxied service", err.Error())
		return
	}
}

func (r *proxiedServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The get-by-id response has no folder scope, so an id alone cannot rebuild state.
	// Import takes the scope plus the name and resolves the service through get-by-name.
	parts := strings.Split(req.ID, ",")
	if len(parts) != 4 {
		resp.Diagnostics.AddError(
			"Invalid import identifier",
			fmt.Sprintf("Expected format '<project_id>,<environment>,<secret_path>,<name>', got: %q", req.ID),
		)
		return
	}

	projectId, environment, secretPath, name := parts[0], parts[1], parts[2], parts[3]
	if projectId == "" || environment == "" || secretPath == "" || name == "" {
		resp.Diagnostics.AddError(
			"Invalid import identifier",
			fmt.Sprintf("All of project_id, environment, secret_path, and name must be non-empty, got: %q", req.ID),
		)
		return
	}

	service, err := r.client.GetProxiedServiceByName(infisical.GetProxiedServiceByNameRequest{
		Name:        name,
		ProjectId:   projectId,
		Environment: environment,
		SecretPath:  secretPath,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error importing proxied service", err.Error())
		return
	}

	state := proxiedServiceResourceModel{
		Id:          types.StringValue(service.Service.Id),
		ProjectId:   types.StringValue(projectId),
		Environment: types.StringValue(environment),
		SecretPath:  types.StringValue(secretPath),
		Name:        types.StringValue(service.Service.Name),
		HostPattern: types.StringValue(service.Service.HostPattern),
		IsEnabled:   types.BoolValue(service.Service.IsEnabled),
	}
	state.Credentials = r.credentialsToState(ctx, service.Service.Credentials, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
