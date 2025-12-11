package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type MetaEntry struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// DynamicSecretBaseResource is the resource implementation.
type DynamicSecretBaseResource struct {
	Provider                  infisical.DynamicSecretProvider
	ResourceTypeName          string // terraform resource name suffix
	DynamicSecretName         string // complete descriptive name of the dynamic secret
	client                    *infisical.Client
	ConfigurationAttributes   map[string]schema.Attribute
	ReadConfigurationFromPlan func(ctx context.Context, plan DynamicSecretBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadConfigurationFromApi  func(ctx context.Context, dynamicSecret infisical.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics)
}

type DynamicSecretBaseResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	ProjectSlug      types.String `tfsdk:"project_slug"`
	EnvironmentSlug  types.String `tfsdk:"environment_slug"`
	Path             types.String `tfsdk:"path"`
	DefaultTTL       types.String `tfsdk:"default_ttl"`
	MaxTTL           types.String `tfsdk:"max_ttl"`
	Configuration    types.Object `tfsdk:"configuration"`
	UsernameTemplate types.String `tfsdk:"username_template"`
	Metadata         []MetaEntry  `tfsdk:"metadata"`
}

// Metadata returns the resource type name.
func (r *DynamicSecretBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

func (r *DynamicSecretBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s Dynamic Secret", r.DynamicSecretName),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the dynamic secret.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the dynamic secret.",
				Required:    true,
			},
			"project_slug": schema.StringAttribute{
				Description:   "The slug of the project to create dynamic secret in.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_slug": schema.StringAttribute{
				Description:   "The slug of the environment to create the dynamic secret in.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"path": schema.StringAttribute{
				Description:   "The path to create the dynamic secret in.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"default_ttl": schema.StringAttribute{
				Description: "The default TTL that will be applied for all the leases.",
				Required:    true,
			},
			"max_ttl": schema.StringAttribute{
				Description: "The maximum limit a TTL can be leased or renewed for.",
				Optional:    true,
			},
			"configuration": schema.SingleNestedAttribute{
				Description: "The configuration of the dynamic secret",
				Required:    true,
				Attributes:  r.ConfigurationAttributes,
			},
			"username_template": schema.StringAttribute{
				Description: "The username template of the dynamic secret",
				Optional:    true,
			},
			"metadata": schema.SetNestedAttribute{
				Description: "The metadata associated with this dynamic secret",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The key of the metadata object",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the metadata object",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *DynamicSecretBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *DynamicSecretBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create dynamic secret",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan DynamicSecretBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configurationMap, diags := r.ReadConfigurationFromPlan(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	metadata := []infisical.MetaEntry{}
	if plan.Metadata != nil {
		for _, el := range plan.Metadata {
			metadata = append(metadata, infisical.MetaEntry{
				Key:   el.Key.ValueString(),
				Value: el.Value.ValueString(),
			})
		}
	}

	dynamicSecret, err := r.client.CreateDynamicSecret(infisical.CreateDynamicSecretRequest{
		Provider: infisical.DynamicSecretProviderObject{
			Provider: r.Provider,
			Inputs:   configurationMap,
		},
		Name:             plan.Name.ValueString(),
		ProjectSlug:      plan.ProjectSlug.ValueString(),
		EnvironmentSlug:  plan.EnvironmentSlug.ValueString(),
		Path:             plan.Path.ValueString(),
		DefaultTTL:       plan.DefaultTTL.ValueString(),
		MaxTTL:           plan.MaxTTL.ValueString(),
		UsernameTemplate: plan.UsernameTemplate.ValueString(),
		Metadata:         metadata,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating dynamic secret",
			"Couldn't create dynamic secret, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(dynamicSecret.Id)

	plan.Configuration, diags = r.ReadConfigurationFromApi(ctx, dynamicSecret, plan.Configuration)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *DynamicSecretBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read dynamic secret",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state DynamicSecretBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dynamicSecret, err := r.client.GetDynamicSecretByName(infisical.GetDynamicSecretByNameRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		Path:            state.Path.ValueString(),
		Name:            state.Name.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading dynamic secret",
				"Couldn't read dynamic secret, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.Name = types.StringValue(dynamicSecret.Name)
	state.DefaultTTL = types.StringValue(dynamicSecret.DefaultTTL)

	if dynamicSecret.MaxTTL == "" {
		state.MaxTTL = types.StringNull()
	} else {
		state.MaxTTL = types.StringValue(dynamicSecret.MaxTTL)
	}

	if dynamicSecret.UsernameTemplate == "" {
		state.UsernameTemplate = types.StringNull()
	} else {
		state.UsernameTemplate = types.StringValue(dynamicSecret.UsernameTemplate)
	}

	state.Configuration, diags = r.ReadConfigurationFromApi(ctx, dynamicSecret, state.Configuration)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Metadata != nil {
		if len(dynamicSecret.Metadata) > 0 {
			var converted []MetaEntry
			for _, m := range dynamicSecret.Metadata {
				converted = append(converted, MetaEntry{
					Key:   types.StringValue(m.Key),
					Value: types.StringValue(m.Value),
				})
			}
			state.Metadata = converted
		} else {
			state.Metadata = []MetaEntry{}
		}
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *DynamicSecretBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update dynamic secret",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan DynamicSecretBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DynamicSecretBaseResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configurationMap, diags := r.ReadConfigurationFromPlan(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	metadata := []infisical.MetaEntry{}
	if plan.Metadata != nil {
		for _, el := range plan.Metadata {
			metadata = append(metadata, infisical.MetaEntry{
				Key:   el.Key.ValueString(),
				Value: el.Value.ValueString(),
			})
		}
	}

	var newName string
	if state.Name.ValueString() != plan.Name.ValueString() {
		newName = plan.Name.ValueString()
	}

	dynamicSecret, err := r.client.UpdateDynamicSecret(infisical.UpdateDynamicSecretRequest{
		Name:            state.Name.ValueString(),
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		Path:            state.Path.ValueString(),
		Data: infisical.UpdateDynamicSecretData{
			Inputs:           configurationMap,
			DefaultTTL:       plan.DefaultTTL.ValueString(),
			MaxTTL:           plan.MaxTTL.ValueString(),
			NewName:          newName,
			Metadata:         metadata,
			UsernameTemplate: plan.UsernameTemplate.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating dynamic secret",
			"Couldn't update dynamic secret, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Configuration, diags = r.ReadConfigurationFromApi(ctx, dynamicSecret, plan.Configuration)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DynamicSecretBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete dynamic secret",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state DynamicSecretBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteDynamicSecret(infisical.DeleteDynamicSecretRequest{
		Name:            state.Name.ValueString(),
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		Path:            state.Path.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting dynamic secret",
			"Couldn't delete dynamic secret from Infisical, unexpected error: "+err.Error(),
		)
	}
}
