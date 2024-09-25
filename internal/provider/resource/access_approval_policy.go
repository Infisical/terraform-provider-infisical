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
	Name types.String `tfsdk:"name"`
}

// accessApprovalPolicyResourceModel describes the data source data model.
type accessApprovalPolicyResourceModel struct {
	ID                types.String     `tfsdk:"id"`
	ProjectID         types.String     `tfsdk:"project_id"`
	Name              types.String     `tfsdk:"name"`
	EnvironmentSlug   types.String     `tfsdk:"environment_slug"`
	SecretPath        types.String     `tfsdk:"secret_path"`
	Approvers         []AccessApprover `tfsdk:"approvers"`
	RequiredApprovals types.Int64      `tfsdk:"required_approvals"`
	EnforcementLevel  types.String     `tfsdk:"enforcement_level"`
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
			"environment_slug": schema.StringAttribute{
				Description: "The environment to apply the access approval policy to",
				Required:    true,
			},
			"secret_path": schema.StringAttribute{
				Description: "The secret path to apply the access approval policy to",
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
						"name": schema.StringAttribute{
							Description: "The name of the approver. By default, this is the email",
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

	var approvers []infisical.CreateAccessApprovalPolicyApprover
	for _, el := range plan.Approvers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field name is required for user approvers",
					"Must provide name for user approvers. By default, this is the email",
				)
				return
			}
			if !el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide name for user approvers. By default, this is the email",
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

		approvers = append(approvers, infisical.CreateAccessApprovalPolicyApprover{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}

	accessApprovalPolicy, err := r.client.CreateAccessApprovalPolicy(infisical.CreateAccessApprovalPolicyRequest{
		Name:              plan.Name.ValueString(),
		ProjectSlug:       projectDetail.Slug,
		Environment:       plan.EnvironmentSlug.ValueString(),
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

	approvers := make([]AccessApprover, len(accessApprovalPolicy.AccessApprovalPolicy.Approvers))
	for i, el := range accessApprovalPolicy.AccessApprovalPolicy.Approvers {
		if el.Type == "user" {
			approvers[i] = AccessApprover{
				Name: types.StringValue(el.Name),
				Type: types.StringValue(el.Type),
			}
		} else {
			approvers[i] = AccessApprover{
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

	if state.EnvironmentSlug != plan.EnvironmentSlug {
		resp.Diagnostics.AddError(
			"Unable to update access approval policy",
			fmt.Sprintf("Cannot change environment, previous environment: %s, new environment: %s", state.EnvironmentSlug, plan.EnvironmentSlug),
		)
		return
	}

	var approvers []infisical.UpdateAccessApprovalPolicyApprover
	for _, el := range plan.Approvers {
		if el.Type.ValueString() == "user" {
			if el.Name.IsNull() {
				resp.Diagnostics.AddError(
					"Field name is required for user approvers",
					"Must provide name for user approvers. By default, this is the email",
				)
				return
			}
			if !el.ID.IsNull() {
				resp.Diagnostics.AddError(
					"Field ID cannot be used for user approvers",
					"Must provide name for user approvers. By default, this is the email",
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

		approvers = append(approvers, infisical.UpdateAccessApprovalPolicyApprover{
			ID:   el.ID.ValueString(),
			Name: el.Name.ValueString(),
			Type: el.Type.ValueString(),
		})
	}

	_, err := r.client.UpdateAccessApprovalPolicy(infisical.UpdateAccessApprovalPolicyRequest{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		SecretPath:        plan.SecretPath.ValueString(),
		Approvers:         approvers,
		RequiredApprovals: plan.RequiredApprovals.ValueInt64(),
		EnforcementLevel:  plan.EnforcementLevel.ValueString(),
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
