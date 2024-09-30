package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewSecretApprovalPolicyResource is a helper function to simplify the provider implementation.
func NewSecretApprovalPolicyResource() resource.Resource {
	return &secretApprovalPolicyResource{}
}

// secretApprovalPolicyResource is the resource implementation.
type secretApprovalPolicyResource struct {
	client *infisical.Client
}

type SecretApprover struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"username"`
}

// secretApprovalPolicyResourceModel describes the data source data model.
type secretApprovalPolicyResourceModel struct {
	ID                types.String     `tfsdk:"id"`
	ProjectID         types.String     `tfsdk:"project_id"`
	Name              types.String     `tfsdk:"name"`
	EnvironmentSlug   types.String     `tfsdk:"environment_slug"`
	SecretPath        types.String     `tfsdk:"secret_path"`
	Approvers         []SecretApprover `tfsdk:"approvers"`
	RequiredApprovals types.Int64      `tfsdk:"required_approvals"`
	EnforcementLevel  types.String     `tfsdk:"enforcement_level"`
}

// Metadata returns the resource type name.
func (r *secretApprovalPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_approval_policy"
}

// Schema defines the schema for the resource.
func (r *secretApprovalPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create secret approval policy for your projects",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the secret approval policy",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the project to add the secret approval policy",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description:   "The name of the secret approval policy",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_slug": schema.StringAttribute{
				Description: "The environment to apply the secret approval policy to",
				Required:    true,
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path to apply the secret approval policy to",
				Required:    true,
			},
			"approvers": schema.SetNestedAttribute{
				Required:    true,
				Description: "The required approvers",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "The type of approver. Either group or user",
							Required:    true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the approver",
							Optional:    true,
						},
						"username": schema.StringAttribute{
							Description: "The username of the approver. By default, this is the email",
							Optional:    true,
						},
					},
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
func (r *secretApprovalPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *secretApprovalPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secret approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan secretApprovalPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var approvers []infisical.CreateSecretApprovalPolicyApprover
	for _, el := range plan.Approvers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username is required for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
			if !el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID is required for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
			if !el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field name cannot be used for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
		}

		approvers = append(approvers, infisical.CreateSecretApprovalPolicyApprover{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}

	secretApprovalPolicy, err := r.client.CreateSecretApprovalPolicy(infisical.CreateSecretApprovalPolicyRequest{
		Name:              plan.Name.ValueString(),
		ProjectID:         plan.ProjectID.ValueString(),
		Environment:       plan.EnvironmentSlug.ValueString(),
		SecretPath:        plan.SecretPath.ValueString(),
		Approvers:         approvers,
		RequiredApprovals: plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:  plan.EnforcementLevel.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret approval policy",
			"Couldn't save secret approval policy, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(secretApprovalPolicy.SecretApprovalPolicy.ID)
	plan.Name = types.StringValue(secretApprovalPolicy.SecretApprovalPolicy.Name)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *secretApprovalPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read secret approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state secretApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretApprovalPolicy, err := r.client.GetSecretApprovalPolicyByID(infisical.GetSecretApprovalPolicyByIDRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error fetching secret approval policy from your project",
				"Couldn't read secret approval policy from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.Name = types.StringValue(secretApprovalPolicy.SecretApprovalPolicy.Name)
	state.SecretPath = types.StringValue(secretApprovalPolicy.SecretApprovalPolicy.SecretPath)
	state.RequiredApprovals = types.Int64Value(secretApprovalPolicy.SecretApprovalPolicy.RequiredApprovals)
	state.EnforcementLevel = types.StringValue(secretApprovalPolicy.SecretApprovalPolicy.EnforcementLevel)

	approvers := make([]SecretApprover, len(secretApprovalPolicy.SecretApprovalPolicy.Approvers))
	for i, el := range secretApprovalPolicy.SecretApprovalPolicy.Approvers {
		if el.Type == "user" {
			approvers[i] = SecretApprover{
				Name: types.StringValue(el.Name),
				Type: types.StringValue(el.Type),
			}
		} else {
			approvers[i] = SecretApprover{
				ID:   types.StringValue(el.ID),
				Type: types.StringValue(el.Type),
			}
		}
	}

	state.Approvers = approvers

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *secretApprovalPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan secretApprovalPolicyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state secretApprovalPolicyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectID != plan.ProjectID {
		resp.Diagnostics.AddError(
			"Unable to update secret approval policy",
			fmt.Sprintf("Cannot change project ID, previous project ID: %s, new project ID: %s", state.ProjectID, plan.ProjectID),
		)
		return
	}

	if state.EnvironmentSlug != plan.EnvironmentSlug {
		resp.Diagnostics.AddError(
			"Unable to update secret approval policy",
			fmt.Sprintf("Cannot change environment, previous environment: %s, new environment: %s", state.EnvironmentSlug, plan.EnvironmentSlug),
		)
		return
	}

	var approvers []infisical.UpdateSecretApprovalPolicyApprover
	for _, el := range plan.Approvers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username is required for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
			if !el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide username for user approvers. By default, this is the email",
				)
				return
			}
		}

		if el.Type.ValueString() == "group" {
			if el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID is required for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
			if !el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field username cannot be used for group approvers",
					"Must provide ID for group approvers",
				)
				return
			}
		}

		approvers = append(approvers, infisical.UpdateSecretApprovalPolicyApprover{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}

	_, err := r.client.UpdateSecretApprovalPolicy(infisical.UpdateSecretApprovalPolicyRequest{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		SecretPath:        plan.SecretPath.ValueString(),
		Approvers:         approvers,
		RequiredApprovals: plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:  plan.EnforcementLevel.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret approval policy",
			"Couldn't update secret approval policy, unexpected error: "+err.Error(),
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
func (r *secretApprovalPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret approval policy",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state secretApprovalPolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretApprovalPolicy(infisical.DeleteSecretApprovalPolicyRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting secret approval policy",
			"Couldn't delete secret approval policy, unexpected error: "+err.Error(),
		)
		return
	}
}
