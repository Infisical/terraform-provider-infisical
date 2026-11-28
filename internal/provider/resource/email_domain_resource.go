package resource

import (
	"context"
	"fmt"
	"regexp"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &EmailDomainResource{}
	_ resource.ResourceWithConfigure   = &EmailDomainResource{}
	_ resource.ResourceWithImportState = &EmailDomainResource{}
)

func NewEmailDomainResource() resource.Resource {
	return &EmailDomainResource{}
}

type EmailDomainResource struct {
	client *infisical.Client
}

type EmailDomainResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Domain             types.String `tfsdk:"domain"`
	VerificationRecord types.Object `tfsdk:"verification_record"`
}

var emailDomainVerificationRecordAttrTypes = map[string]attr.Type{
	"type":  types.StringType,
	"name":  types.StringType,
	"value": types.StringType,
}

func (r *EmailDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_domain"
}

func (r *EmailDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Claim an email domain for the organization. The domain belongs to the organization of the machine identity. " +
			"A new domain is pending: publish the DNS record in `verification_record`, then verify the domain with `infisical_email_domain_verification`. " +
			"The verification code expires 7 days after the domain is created. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the email domain.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				Description:   "The email domain to claim, for example `example.com`. Must be lowercase.",
				Required:      true,
				Validators:    []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^[^\sA-Z]+$`), "must be lowercase and must not contain whitespace")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"verification_record": schema.SingleNestedAttribute{
				Description:   "The DNS record that proves ownership of the domain.",
				Computed:      true,
				PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "The type of the DNS record (`TXT`).",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the DNS record (`_infisical-verification.<domain>`).",
						Computed:    true,
					},
					"value": schema.StringAttribute{
						Description: "The content of the DNS record (`infisical-domain-verification=<code>`).",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *EmailDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func emailDomainModelFromAPI(emailDomain infisical.EmailDomain) EmailDomainResourceModel {
	return EmailDomainResourceModel{
		ID:     types.StringValue(emailDomain.ID),
		Domain: types.StringValue(emailDomain.Domain),
		VerificationRecord: types.ObjectValueMust(emailDomainVerificationRecordAttrTypes, map[string]attr.Value{
			"type":  types.StringValue(emailDomain.VerificationRecord.Type),
			"name":  types.StringValue(emailDomain.VerificationRecord.Name),
			"value": types.StringValue(emailDomain.VerificationRecord.Value),
		}),
	}
}

func (r *EmailDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create email domain",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan EmailDomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	emailDomain, err := r.client.CreateEmailDomain(infisical.CreateEmailDomainRequest{
		Domain: plan.Domain.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating email domain",
			"Couldn't create email domain, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, emailDomainModelFromAPI(emailDomain))
	resp.Diagnostics.Append(diags...)
}

func (r *EmailDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read email domain",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state EmailDomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	emailDomain, err := r.client.GetEmailDomainByID(infisical.GetEmailDomainByIDRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading email domain",
			"Couldn't read email domain from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, emailDomainModelFromAPI(emailDomain))
	resp.Diagnostics.Append(diags...)
}

func (r *EmailDomainResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// The API has no update endpoint and domain is ForceNew; Update is never called.
}

func (r *EmailDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete email domain",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state EmailDomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEmailDomain(infisical.DeleteEmailDomainRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting email domain",
			"Couldn't delete email domain, unexpected error: "+err.Error(),
		)
	}
}

func (r *EmailDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.ParseUUID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be the ID of the email domain (a UUID)",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
