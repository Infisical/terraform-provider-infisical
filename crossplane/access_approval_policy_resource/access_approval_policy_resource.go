package resource

import (
	"context"
	"encoding/json"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewAccessApprovalPolicyResource is a helper function to simplify the provider implementation.
func NewAccessApprovalPolicyResource() resource.Resource {
	return &accessApprovalPolicyResource{}
}

// accessApprovalPolicyResource is the resource implementation.
type accessApprovalPolicyResource struct {
	client *infisical.Client
}

type AccessApprover struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"username"`
}

type AccessApproverJSON struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"username,omitempty"`
}

// accessApprovalPolicyResourceModel describes the data source data model.
type accessApprovalPolicyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	Name              types.String `tfsdk:"name"`
	EnvironmentSlugs  types.List   `tfsdk:"environment_slugs"`
	SecretPath        types.String `tfsdk:"secret_path"`
	Approvers         types.String `tfsdk:"approvers"` // comes in as a string, and then we parse it to []AccessApproverJSON, and then to []AccessApprover
	RequiredApprovals types.Int64  `tfsdk:"required_approvals"`
	EnforcementLevel  types.String `tfsdk:"enforcement_level"`
}

// Metadata returns the resource type name.
func (r *accessApprovalPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_approval_policy"
}

// Schema defines the schema for the resource.
func (r *accessApprovalPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create access approval policy for your projects",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the access approval policy",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project to add the access approval policy",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description:   "The name of the access approval policy",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_slugs": schema.ListAttribute{
				Description: "The environments to apply the access approval policy to",
				Required:    true,
				ElementType: types.StringType,
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path to apply the access approval policy to",
				Required:    true,
			},
			"approvers": schema.StringAttribute{
				Required:    true,
				Description: "The required approvers",
				PlanModifiers: []planmodifier.String{
					pkg.UnorderedJsonEquivalentModifier{},
				},
				Validators: []validator.String{
					infisicaltf.JsonStringValidator,
				},
			},
			"required_approvals": schema.Int64Attribute{
				Description: "The number of required approvers",
				Required:    true,
			},
			"enforcement_level": schema.StringAttribute{
				Description: "The enforcement level of the policy. This can either be hard or soft",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("hard"),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *accessApprovalPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *accessApprovalPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create access approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan accessApprovalPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectDetail, err := r.client.GetProjectById(infisical.GetProjectByIdRequest{
		ID: plan.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating access approval policy",
			"Couldn't fetch project details, unexpected error: "+err.Error(),
		)
		return
	}

	var approversJSON []AccessApproverJSON
	err = json.Unmarshal([]byte(plan.Approvers.ValueString()), &approversJSON)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing approvers JSON",
			fmt.Sprintf("Failed to parse approvers JSON: %s", err.Error()),
		)
		return
	}

	var approvers []infisical.CreateAccessApprovalPolicyApprover
	for _, el := range approversJSON {
		if el.Type == "user" {
			if el.Name == "" {
				resp.Diagnostics.AddError(
					"Field username is required for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
			if el.ID != "" {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
		}

		if el.Type == "group" {
			if el.ID == "" {
				resp.Diagnostics.AddError(
					"Field ID is required for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
			if el.Name != "" {
				resp.Diagnostics.AddError(
					"Field username cannot be used for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
		}

		approvers = append(approvers, infisical.CreateAccessApprovalPolicyApprover{
			ID:   el.ID,
			Name: el.Name,
			Type: el.Type,
		})
	}

	environments := make([]string, 0)

	envSlugs := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.EnvironmentSlugs)
	if envSlugs != nil {
		environments = append(environments, envSlugs...)
	}

	var environment string
	if len(environments) > 0 {
		environment = environments[0]
	}

	accessApprovalPolicy, err := r.client.CreateAccessApprovalPolicy(infisical.CreateAccessApprovalPolicyRequest{
		Name:              plan.Name.ValueString(),
		ProjectSlug:       projectDetail.Slug,
		Environments:      environments,
		Environment:       environment,
		SecretPath:        plan.SecretPath.ValueString(),
		Approvers:         approvers,
		RequiredApprovals: plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:  plan.EnforcementLevel.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating access approval policy",
			"Couldn't save access approval policy, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.ID)
	plan.Name = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.Name)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *accessApprovalPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read access approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state accessApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessApprovalPolicy, err := r.client.GetAccessApprovalPolicyByID(infisical.GetAccessApprovalPolicyByIDRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error fetching access approval policy from your project",
				"Couldn't read access approval policy from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.Name = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.Name)
	state.SecretPath = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.SecretPath)
	state.RequiredApprovals = types.Int64Value(accessApprovalPolicy.AccessApprovalPolicy.RequiredApprovals)
	state.EnforcementLevel = types.StringValue(accessApprovalPolicy.AccessApprovalPolicy.EnforcementLevel)

	approvers := make([]AccessApproverJSON, len(accessApprovalPolicy.AccessApprovalPolicy.Approvers))
	for i, el := range accessApprovalPolicy.AccessApprovalPolicy.Approvers {
		if el.Type == "user" {
			approvers[i] = AccessApproverJSON{
				Name: el.Name,
				Type: el.Type,
			}
		} else {
			approvers[i] = AccessApproverJSON{
				ID:   el.ID,
				Type: el.Type,
			}
		}
	}

	approversJSON, err := json.Marshal(approvers)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling approvers",
			"Couldn't marshall approvers, unexpected error: "+err.Error(),
		)
		return
	}

	state.Approvers = types.StringValue(string(approversJSON))

	if len(accessApprovalPolicy.AccessApprovalPolicy.Environments) > 0 {
		// Extract environment slugs from the environment objects
		var environmentSlugs []string
		for _, env := range accessApprovalPolicy.AccessApprovalPolicy.Environments {
			environmentSlugs = append(environmentSlugs, env.Slug)
		}

		// Always set the new environment_slugs field
		environmentSlugsList, diags := types.ListValueFrom(ctx, types.StringType, environmentSlugs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.EnvironmentSlugs = environmentSlugsList
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *accessApprovalPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update access approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan accessApprovalPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state accessApprovalPolicyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectID != plan.ProjectID {
		resp.Diagnostics.AddError(
			"Unable to update access approval policy",
			fmt.Sprintf("Cannot change project ID, previous project ID: %s, new project ID: %s", state.ProjectID, plan.ProjectID),
		)
		return
	}

	if infisicaltf.IsAttrValueEmpty(plan.EnvironmentSlugs) {
		resp.Diagnostics.AddError(
			"Unable to update access approval policy",
			fmt.Sprintf("Cannot change environment, previous environment: %s, new environment: %s", state.EnvironmentSlugs, plan.EnvironmentSlugs),
		)
		return
	}

	var approvers []infisical.UpdateAccessApprovalPolicyApprover

	var approversJSON []AccessApproverJSON
	err := json.Unmarshal([]byte(plan.Approvers.ValueString()), &approversJSON)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing approvers JSON",
			fmt.Sprintf("Failed to parse approvers JSON: %s", err.Error()),
		)
		return
	}

	for _, el := range approversJSON {
		if el.Type == "user" {
			if el.Name == "" {
				resp.Diagnostics.AddError(
					"Field username is required for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
			if el.ID != "" {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
		}

		if el.Type == "group" {
			if el.ID == "" {
				resp.Diagnostics.AddError(
					"Field ID is required for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
			if el.Name != "" {
				resp.Diagnostics.AddError(
					"Field username cannot be used for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
		}

		approvers = append(approvers, infisical.UpdateAccessApprovalPolicyApprover{
			ID:   el.ID,
			Name: el.Name,
			Type: el.Type,
		})
	}

	environments := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.EnvironmentSlugs)

	_, err = r.client.UpdateAccessApprovalPolicy(infisical.UpdateAccessApprovalPolicyRequest{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		SecretPath:        plan.SecretPath.ValueString(),
		Approvers:         approvers,
		RequiredApprovals: plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:  plan.EnforcementLevel.ValueString(),
		Environments:      environments,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating access approval policy",
			"Couldn't update access approval policy, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *accessApprovalPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete access approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state accessApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteAccessApprovalPolicy(infisical.DeleteAccessApprovalPolicyRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting access approval policy",
			"Couldn't delete access approval policy, unexpected error: "+err.Error(),
		)
		return
	}
}
