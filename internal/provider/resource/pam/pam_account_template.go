package resource

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                   = &pamAccountTemplateResource{}
	_ resource.ResourceWithConfigure      = &pamAccountTemplateResource{}
	_ resource.ResourceWithImportState    = &pamAccountTemplateResource{}
	_ resource.ResourceWithModifyPlan     = &pamAccountTemplateResource{}
	_ resource.ResourceWithValidateConfig = &pamAccountTemplateResource{}
)

var pamAccountTypes = []string{
	"ssh", "postgres", "mysql", "mssql", "oracledb", "mongodb", "redis", "snowflake", "clickhouse",
	"kubernetes", "aws-iam", "gcp-service-account", "azure-cli", "windows", "windows-ad",
}

var pamRotatableAccountTypes = []string{"postgres", "mysql", "mssql", "oracledb", "windows", "windows-ad"}

var pamCommandBlockingAccountTypes = []string{"ssh", "snowflake", "clickhouse"}

// The API enables built-in masking by default for types whose sessions it can mask.
func pamBuiltInMaskingDefault(accountType string) bool {
	return accountType != "windows" && accountType != "windows-ad" && accountType != "aws-iam"
}

const (
	pamPolicyRequiresApproval   = "requires-approval"
	pamPolicyAllowAutoApproval  = "allow-auto-approval"
	pamPolicyAllowBreakGlass    = "allow-break-glass"
	pamPolicyRequireMfa         = "require-mfa"
	pamPolicyRequireReason      = "require-reason"
	pamPolicyMaxSessionDuration = "max-session-duration"
	pamPolicyCommandBlocking    = "command-blocking"
)

const (
	pamRecordingBackendPostgres = "postgres"
	pamRecordingBackendS3       = "aws-s3"
)

const pamDefaultHeartbeatIntervalSeconds = 86400

const pamSafePasswordSymbols = "!@#$%^&*()-_=+[]{}|:,.<>/~"

const pamOracleMaxPasswordLength = 30

var pamPatternValidators = []validator.List{
	listvalidator.SizeBetween(1, 20),
	listvalidator.ValueStringsAre(
		stringvalidator.LengthBetween(1, 500),
		stringvalidator.RegexMatches(regexp.MustCompile(`^[^\r\n]*$`), "must be a single line"),
	),
}

func NewPamAccountTemplateResource() resource.Resource {
	return &pamAccountTemplateResource{}
}

type pamAccountTemplateResource struct {
	client *infisical.Client
}

type pamAccountTemplateResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	AccountType   types.String `tfsdk:"account_type"`
	GatewayID     types.String `tfsdk:"gateway_id"`
	GatewayPoolID types.String `tfsdk:"gateway_pool_id"`
	Policies      types.Object `tfsdk:"policies"`
	Settings      types.Object `tfsdk:"settings"`
}

type pamTemplatePoliciesModel struct {
	RequiresApproval   types.Bool  `tfsdk:"requires_approval"`
	AllowAutoApproval  types.Bool  `tfsdk:"allow_auto_approval"`
	AllowBreakGlass    types.Bool  `tfsdk:"allow_break_glass"`
	RequireMfa         types.Bool  `tfsdk:"require_mfa"`
	RequireReason      types.Bool  `tfsdk:"require_reason"`
	MaxSessionDuration types.Int64 `tfsdk:"max_session_duration"`
	CommandBlocking    types.List  `tfsdk:"command_blocking"`
}

type pamTemplateSettingsModel struct {
	Recording            types.Object `tfsdk:"recording"`
	SessionLogMasking    types.Object `tfsdk:"session_log_masking"`
	Heartbeat            types.Object `tfsdk:"heartbeat"`
	Rotation             types.Object `tfsdk:"rotation"`
	PasswordRequirements types.Object `tfsdk:"password_requirements"`
}

type pamTemplateRecordingModel struct {
	Enabled        types.Bool   `tfsdk:"enabled"`
	StorageBackend types.String `tfsdk:"storage_backend"`
	ConnectionID   types.String `tfsdk:"connection_id"`
	S3             types.Object `tfsdk:"s3"`
}

type pamTemplateRecordingS3Model struct {
	Bucket    types.String `tfsdk:"bucket"`
	Region    types.String `tfsdk:"region"`
	KeyPrefix types.String `tfsdk:"key_prefix"`
}

type pamTemplateSessionLogMaskingModel struct {
	Patterns         types.List `tfsdk:"patterns"`
	BuiltInDetection types.Bool `tfsdk:"built_in_detection"`
}

// Shared by heartbeat and rotation.
type pamTemplateScheduleModel struct {
	Enabled         types.Bool  `tfsdk:"enabled"`
	IntervalSeconds types.Int64 `tfsdk:"interval_seconds"`
}

type pamTemplatePasswordRequirementsModel struct {
	Length         types.Int64  `tfsdk:"length"`
	Required       types.Object `tfsdk:"required"`
	AllowedSymbols types.String `tfsdk:"allowed_symbols"`
}

type pamTemplatePasswordRequiredModel struct {
	Digits    types.Int64 `tfsdk:"digits"`
	Lowercase types.Int64 `tfsdk:"lowercase"`
	Uppercase types.Int64 `tfsdk:"uppercase"`
	Symbols   types.Int64 `tfsdk:"symbols"`
}

var pamTemplatePoliciesAttrTypes = map[string]attr.Type{
	"requires_approval":    types.BoolType,
	"allow_auto_approval":  types.BoolType,
	"allow_break_glass":    types.BoolType,
	"require_mfa":          types.BoolType,
	"require_reason":       types.BoolType,
	"max_session_duration": types.Int64Type,
	"command_blocking":     types.ListType{ElemType: types.StringType},
}

var pamTemplateRecordingS3AttrTypes = map[string]attr.Type{
	"bucket":     types.StringType,
	"region":     types.StringType,
	"key_prefix": types.StringType,
}

var pamTemplateRecordingAttrTypes = map[string]attr.Type{
	"enabled":         types.BoolType,
	"storage_backend": types.StringType,
	"connection_id":   types.StringType,
	"s3":              types.ObjectType{AttrTypes: pamTemplateRecordingS3AttrTypes},
}

var pamTemplateSessionLogMaskingAttrTypes = map[string]attr.Type{
	"patterns":           types.ListType{ElemType: types.StringType},
	"built_in_detection": types.BoolType,
}

var pamTemplateScheduleAttrTypes = map[string]attr.Type{
	"enabled":          types.BoolType,
	"interval_seconds": types.Int64Type,
}

var pamTemplatePasswordRequiredAttrTypes = map[string]attr.Type{
	"digits":    types.Int64Type,
	"lowercase": types.Int64Type,
	"uppercase": types.Int64Type,
	"symbols":   types.Int64Type,
}

var pamTemplatePasswordRequirementsAttrTypes = map[string]attr.Type{
	"length":          types.Int64Type,
	"required":        types.ObjectType{AttrTypes: pamTemplatePasswordRequiredAttrTypes},
	"allowed_symbols": types.StringType,
}

var pamTemplateSettingsAttrTypes = map[string]attr.Type{
	"recording":             types.ObjectType{AttrTypes: pamTemplateRecordingAttrTypes},
	"session_log_masking":   types.ObjectType{AttrTypes: pamTemplateSessionLogMaskingAttrTypes},
	"heartbeat":             types.ObjectType{AttrTypes: pamTemplateScheduleAttrTypes},
	"rotation":              types.ObjectType{AttrTypes: pamTemplateScheduleAttrTypes},
	"password_requirements": types.ObjectType{AttrTypes: pamTemplatePasswordRequirementsAttrTypes},
}

func (r *pamAccountTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pam_account_template"
}

func pamScheduleAttributes(subject string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			Description: fmt.Sprintf("Whether %s is on. Set `interval_seconds` as well to run it on a schedule.", subject),
			Required:    true,
		},
		"interval_seconds": schema.Int64Attribute{
			Description: fmt.Sprintf("How often %s runs, in seconds, from 3600 to 31536000. Leave unset to run it only on demand.", subject),
			Optional:    true,
			Validators:  []validator.Int64{int64validator.Between(3600, 31536000)},
		},
	}
}

func pamRequiredCharactersAttribute(kind string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Description: fmt.Sprintf("The minimum number of %s in the password. Defaults to 0.", kind),
		Optional:    true,
		Computed:    true,
		Validators:  []validator.Int64{int64validator.AtLeast(0)},
	}
}

func (r *pamAccountTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage account templates in Infisical PAM. A template holds the account type, gateway, policies and settings shared by its accounts. Only Machine Identity authentication is supported for this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the account template.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the account template, unique in the organization. " + pamNameDescription,
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					infisicaltf.SlugRegexValidator,
				},
			},
			"description": schema.StringAttribute{
				Description: "An optional description of the account template.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 256)},
			},
			"account_type": schema.StringAttribute{
				Description:   "The type of account the template applies to. Changing it creates a new template. Options: " + strings.Join(pamAccountTypes, ", ") + ".",
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf(pamAccountTypes...)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"gateway_id": schema.StringAttribute{
				Description: "The ID of the gateway that accounts using the template connect through, unless an account sets its own.",
				Optional:    true,
				Validators: []validator.String{
					infisicaltf.UuidValidator,
					stringvalidator.ConflictsWith(path.MatchRoot("gateway_pool_id")),
				},
			},
			"gateway_pool_id": schema.StringAttribute{
				Description: "The ID of the gateway pool that accounts using the template connect through, unless an account sets its own.",
				Optional:    true,
				Validators:  []validator.String{infisicaltf.UuidValidator},
			},
			"policies": schema.SingleNestedAttribute{
				Description: "Access policies applied to every account using the template. A policy left unset is off.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"requires_approval": schema.BoolAttribute{
						Description: "Whether users must request and receive approval before launching a session or viewing credentials.",
						Optional:    true,
					},
					"allow_auto_approval": schema.BoolAttribute{
						Description: "Whether requests that pass one of the folder's auto-approval rules are approved instantly. Has no effect unless `requires_approval` is true.",
						Optional:    true,
					},
					"allow_break_glass": schema.BoolAttribute{
						Description: "Whether the folder's break-glass users can grant themselves access in an emergency. Has no effect unless `requires_approval` is true.",
						Optional:    true,
					},
					"require_mfa": schema.BoolAttribute{
						Description: "Whether users must re-authenticate with MFA before accessing an account.",
						Optional:    true,
					},
					"require_reason": schema.BoolAttribute{
						Description: "Whether users must give a reason before accessing an account.",
						Optional:    true,
					},
					"max_session_duration": schema.Int64Attribute{
						Description: "The maximum session length in seconds, from 60 to 86400 (900 to 3600 for `aws-iam` templates).",
						Optional:    true,
						Validators:  []validator.Int64{int64validator.Between(60, 86400)},
					},
					"command_blocking": schema.ListAttribute{
						Description: "Regular expressions, one per command or statement to reject during a session. Only for `" + strings.Join(pamCommandBlockingAccountTypes, "`, `") + "` templates.",
						Optional:    true,
						ElementType: types.StringType,
						Validators:  pamPatternValidators,
					},
				},
			},
			"settings": schema.SingleNestedAttribute{
				Description: "Session recording, log masking, health check and rotation settings. Unset values take their defaults.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"recording": schema.SingleNestedAttribute{
						Description: "Session recording. Defaults to recording every session into Infisical's database.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether sessions are recorded. Defaults to true.",
								Optional:    true,
								Computed:    true,
							},
							"storage_backend": schema.StringAttribute{
								Description: "Where recordings are stored: `postgres` (Infisical's database) or `aws-s3`, which needs `connection_id` and `s3`. Defaults to `postgres`.",
								Optional:    true,
								Computed:    true,
								Validators:  []validator.String{stringvalidator.OneOf(pamRecordingBackendPostgres, pamRecordingBackendS3)},
							},
							"connection_id": schema.StringAttribute{
								Description: "The ID of the AWS or S3-Compatible Storage app connection used to store recordings.",
								Optional:    true,
								Validators:  []validator.String{infisicaltf.UuidValidator},
							},
							"s3": schema.SingleNestedAttribute{
								Description: "The bucket recordings are stored in when `storage_backend` is `aws-s3`.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"bucket": schema.StringAttribute{
										Description: "The name of the bucket.",
										Required:    true,
										Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
									},
									"region": schema.StringAttribute{
										Description: "The AWS region of the bucket. Leave unset for S3-Compatible Storage connections.",
										Optional:    true,
										Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
									},
									"key_prefix": schema.StringAttribute{
										Description: "An optional prefix for the recording object keys.",
										Optional:    true,
										Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
									},
								},
							},
						},
					},
					"session_log_masking": schema.SingleNestedAttribute{
						Description: "Masking of sensitive values in session logs.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"patterns": schema.ListAttribute{
								Description: "Regular expressions whose matches are masked in session logs.",
								Optional:    true,
								ElementType: types.StringType,
								Validators:  pamPatternValidators,
							},
							"built_in_detection": schema.BoolAttribute{
								Description: "Whether common secrets are masked without a pattern. Defaults to true, except for `windows`, `windows-ad` and `aws-iam` templates.",
								Optional:    true,
								Computed:    true,
							},
						},
					},
					"heartbeat": schema.SingleNestedAttribute{
						Description: "Scheduled checks that each account's stored credential still works. Defaults to a daily check.",
						Optional:    true,
						Computed:    true,
						Attributes:  pamScheduleAttributes("credential health checking"),
					},
					"rotation": schema.SingleNestedAttribute{
						Description: "Scheduled credential rotation. Only for `" + strings.Join(pamRotatableAccountTypes, "`, `") + "` templates.",
						Optional:    true,
						Attributes:  pamScheduleAttributes("credential rotation"),
					},
					"password_requirements": schema.SingleNestedAttribute{
						Description: "Requirements for the passwords rotation generates and for passwords set on accounts. Only for `" + strings.Join(pamRotatableAccountTypes, "`, `") + "` templates.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"length": schema.Int64Attribute{
								Description: "The length of the password, from 1 to 250 (at most 30 for `oracledb` templates).",
								Required:    true,
								Validators:  []validator.Int64{int64validator.Between(1, 250)},
							},
							"required": schema.SingleNestedAttribute{
								Description: "The minimum number of each kind of character. At least one must be above 0, and together they can't exceed `length`.",
								Required:    true,
								Attributes: map[string]schema.Attribute{
									"digits":    pamRequiredCharactersAttribute("digits"),
									"lowercase": pamRequiredCharactersAttribute("lowercase letters"),
									"uppercase": pamRequiredCharactersAttribute("uppercase letters"),
									"symbols":   pamRequiredCharactersAttribute("symbols"),
								},
							},
							"allowed_symbols": schema.StringAttribute{
								Description: "The symbols the password may contain, drawn from `" + pamSafePasswordSymbols + "`.",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.RegexMatches(regexp.MustCompile(`^[!@#$%^&*()\-_=+\[\]{}|:,.<>/~]+$`), "may only include "+pamSafePasswordSymbols),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *pamAccountTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configurePamClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *pamAccountTemplateResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config pamAccountTemplateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountType := config.AccountType.ValueString()
	typeKnown := !config.AccountType.IsNull() && !config.AccountType.IsUnknown()

	if !config.Policies.IsNull() && !config.Policies.IsUnknown() {
		var policies pamTemplatePoliciesModel
		resp.Diagnostics.Append(config.Policies.As(ctx, &policies, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		if typeKnown && !policies.CommandBlocking.IsNull() && !slices.Contains(pamCommandBlockingAccountTypes, accountType) {
			resp.Diagnostics.AddAttributeError(path.Root("policies").AtName("command_blocking"), "Policy not supported",
				fmt.Sprintf("Command blocking is only supported for %s templates, not %s.", strings.Join(pamCommandBlockingAccountTypes, ", "), accountType))
		}

		duration := policies.MaxSessionDuration
		if typeKnown && accountType == "aws-iam" && !duration.IsNull() && !duration.IsUnknown() &&
			(duration.ValueInt64() < 900 || duration.ValueInt64() > 3600) {
			resp.Diagnostics.AddAttributeError(path.Root("policies").AtName("max_session_duration"), "Invalid session duration",
				"aws-iam templates take a maximum session duration from 900 to 3600 seconds.")
		}
	}

	if config.Settings.IsNull() || config.Settings.IsUnknown() {
		return
	}

	var settings pamTemplateSettingsModel
	resp.Diagnostics.Append(config.Settings.As(ctx, &settings, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	if typeKnown && !slices.Contains(pamRotatableAccountTypes, accountType) {
		for name, value := range map[string]types.Object{"rotation": settings.Rotation, "password_requirements": settings.PasswordRequirements} {
			if !value.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("settings").AtName(name), "Setting not supported",
					fmt.Sprintf("Credential rotation is only supported for %s templates, not %s.", strings.Join(pamRotatableAccountTypes, ", "), accountType))
			}
		}
	}

	if !settings.Recording.IsNull() && !settings.Recording.IsUnknown() {
		var recording pamTemplateRecordingModel
		resp.Diagnostics.Append(settings.Recording.As(ctx, &recording, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		if recording.StorageBackend.ValueString() == pamRecordingBackendS3 && (recording.ConnectionID.IsNull() || recording.S3.IsNull()) {
			resp.Diagnostics.AddAttributeError(path.Root("settings").AtName("recording"), "Incomplete recording storage",
				"Storing recordings in S3 needs both `connection_id` and `s3`.")
		}
	}

	if !settings.PasswordRequirements.IsNull() && !settings.PasswordRequirements.IsUnknown() {
		var requirements pamTemplatePasswordRequirementsModel
		resp.Diagnostics.Append(settings.PasswordRequirements.As(ctx, &requirements, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() || requirements.Required.IsNull() || requirements.Required.IsUnknown() || requirements.Length.IsUnknown() {
			return
		}
		var required pamTemplatePasswordRequiredModel
		resp.Diagnostics.Append(requirements.Required.As(ctx, &required, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		total := int64(0)
		for _, count := range []types.Int64{required.Digits, required.Lowercase, required.Uppercase, required.Symbols} {
			if count.IsUnknown() {
				return
			}
			total += count.ValueInt64()
		}
		if typeKnown && accountType == "oracledb" && requirements.Length.ValueInt64() > pamOracleMaxPasswordLength {
			resp.Diagnostics.AddAttributeError(path.Root("settings").AtName("password_requirements").AtName("length"), "Invalid password requirements",
				fmt.Sprintf("Oracle passwords are limited to %d characters.", pamOracleMaxPasswordLength))
		}

		requiredPath := path.Root("settings").AtName("password_requirements").AtName("required")
		if total == 0 {
			resp.Diagnostics.AddAttributeError(requiredPath, "Invalid password requirements", "At least one kind of character must be required.")
		} else if total > requirements.Length.ValueInt64() {
			resp.Diagnostics.AddAttributeError(requiredPath, "Invalid password requirements", "The required characters can't add up to more than `length`.")
		}
	}
}

// ModifyPlan fills unset settings with the API's defaults.
func (r *pamAccountTemplateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config pamAccountTemplateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.AccountType.IsUnknown() {
		return
	}

	settings, diags := resolvePamTemplateSettings(ctx, config.Settings, config.AccountType.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("settings"), settings)...)
}

func resolvePamTemplateSettings(ctx context.Context, config types.Object, accountType string) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if config.IsUnknown() {
		return types.ObjectUnknown(pamTemplateSettingsAttrTypes), diags
	}

	settings := pamTemplateSettingsModel{
		Recording:            types.ObjectNull(pamTemplateRecordingAttrTypes),
		SessionLogMasking:    types.ObjectNull(pamTemplateSessionLogMaskingAttrTypes),
		Heartbeat:            types.ObjectNull(pamTemplateScheduleAttrTypes),
		Rotation:             types.ObjectNull(pamTemplateScheduleAttrTypes),
		PasswordRequirements: types.ObjectNull(pamTemplatePasswordRequirementsAttrTypes),
	}
	if !config.IsNull() {
		diags.Append(config.As(ctx, &settings, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return types.ObjectNull(pamTemplateSettingsAttrTypes), diags
		}
	}

	if !settings.Recording.IsUnknown() {
		recording := pamTemplateRecordingModel{
			Enabled:        types.BoolNull(),
			StorageBackend: types.StringNull(),
			ConnectionID:   types.StringNull(),
			S3:             types.ObjectNull(pamTemplateRecordingS3AttrTypes),
		}
		if !settings.Recording.IsNull() {
			diags.Append(settings.Recording.As(ctx, &recording, basetypes.ObjectAsOptions{})...)
		}
		if recording.Enabled.IsNull() {
			recording.Enabled = types.BoolValue(true)
		}
		if recording.StorageBackend.IsNull() {
			recording.StorageBackend = types.StringValue(pamRecordingBackendPostgres)
		}
		settings.Recording = objectValueFrom(ctx, pamTemplateRecordingAttrTypes, recording, &diags)
	}

	if !settings.SessionLogMasking.IsUnknown() {
		masking := pamTemplateSessionLogMaskingModel{
			Patterns:         types.ListNull(types.StringType),
			BuiltInDetection: types.BoolNull(),
		}
		if !settings.SessionLogMasking.IsNull() {
			diags.Append(settings.SessionLogMasking.As(ctx, &masking, basetypes.ObjectAsOptions{})...)
		}
		if masking.BuiltInDetection.IsNull() {
			masking.BuiltInDetection = types.BoolValue(pamBuiltInMaskingDefault(accountType))
		}
		settings.SessionLogMasking = objectValueFrom(ctx, pamTemplateSessionLogMaskingAttrTypes, masking, &diags)
	}

	if settings.Heartbeat.IsNull() {
		settings.Heartbeat = types.ObjectValueMust(pamTemplateScheduleAttrTypes, map[string]attr.Value{
			"enabled":          types.BoolValue(true),
			"interval_seconds": types.Int64Value(pamDefaultHeartbeatIntervalSeconds),
		})
	}

	if !settings.PasswordRequirements.IsNull() && !settings.PasswordRequirements.IsUnknown() {
		var requirements pamTemplatePasswordRequirementsModel
		diags.Append(settings.PasswordRequirements.As(ctx, &requirements, basetypes.ObjectAsOptions{})...)
		if !requirements.Required.IsNull() && !requirements.Required.IsUnknown() {
			var required pamTemplatePasswordRequiredModel
			diags.Append(requirements.Required.As(ctx, &required, basetypes.ObjectAsOptions{})...)
			for _, count := range []*types.Int64{&required.Digits, &required.Lowercase, &required.Uppercase, &required.Symbols} {
				if count.IsNull() {
					*count = types.Int64Value(0)
				}
			}
			requirements.Required = objectValueFrom(ctx, pamTemplatePasswordRequiredAttrTypes, required, &diags)
		}
		settings.PasswordRequirements = objectValueFrom(ctx, pamTemplatePasswordRequirementsAttrTypes, requirements, &diags)
	}

	if diags.HasError() {
		return types.ObjectNull(pamTemplateSettingsAttrTypes), diags
	}

	result, objectDiags := types.ObjectValueFrom(ctx, pamTemplateSettingsAttrTypes, settings)
	diags.Append(objectDiags...)
	return result, diags
}

func objectValueFrom(ctx context.Context, attrTypes map[string]attr.Type, value any, diags *diag.Diagnostics) types.Object {
	object, objectDiags := types.ObjectValueFrom(ctx, attrTypes, value)
	diags.Append(objectDiags...)
	return object
}

func joinPamPatterns(ctx context.Context, patterns types.List, diags *diag.Diagnostics) string {
	var values []string
	diags.Append(patterns.ElementsAs(ctx, &values, false)...)
	return strings.Join(values, "\n")
}

func splitPamPatterns(value any) types.List {
	raw, _ := value.(string)
	patterns := []attr.Value{}
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if line != "" {
			patterns = append(patterns, types.StringValue(line))
		}
	}
	if len(patterns) == 0 {
		return types.ListNull(types.StringType)
	}
	return types.ListValueMust(types.StringType, patterns)
}

func pamPoliciesToAPI(ctx context.Context, policiesObject types.Object) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	policies := map[string]any{}
	if policiesObject.IsNull() || policiesObject.IsUnknown() {
		return policies, diags
	}

	var model pamTemplatePoliciesModel
	diags.Append(policiesObject.As(ctx, &model, basetypes.ObjectAsOptions{})...)

	for key, value := range map[string]types.Bool{
		pamPolicyRequiresApproval:  model.RequiresApproval,
		pamPolicyAllowAutoApproval: model.AllowAutoApproval,
		pamPolicyAllowBreakGlass:   model.AllowBreakGlass,
		pamPolicyRequireMfa:        model.RequireMfa,
		pamPolicyRequireReason:     model.RequireReason,
	} {
		if !value.IsNull() {
			policies[key] = value.ValueBool()
		}
	}
	if !model.MaxSessionDuration.IsNull() {
		policies[pamPolicyMaxSessionDuration] = model.MaxSessionDuration.ValueInt64()
	}
	if !model.CommandBlocking.IsNull() {
		policies[pamPolicyCommandBlocking] = joinPamPatterns(ctx, model.CommandBlocking, &diags)
	}

	return policies, diags
}

// Empty policies stay an object when state had one, so `policies = {}` doesn't show a diff.
func pamPoliciesFromAPI(policies map[string]any, prior types.Object) types.Object {
	boolPolicy := func(key string) types.Bool {
		if value, ok := policies[key].(bool); ok {
			return types.BoolValue(value)
		}
		return types.BoolNull()
	}

	duration := types.Int64Null()
	if value, ok := policies[pamPolicyMaxSessionDuration].(float64); ok {
		duration = types.Int64Value(int64(value))
	}

	values := map[string]attr.Value{
		"requires_approval":    boolPolicy(pamPolicyRequiresApproval),
		"allow_auto_approval":  boolPolicy(pamPolicyAllowAutoApproval),
		"allow_break_glass":    boolPolicy(pamPolicyAllowBreakGlass),
		"require_mfa":          boolPolicy(pamPolicyRequireMfa),
		"require_reason":       boolPolicy(pamPolicyRequireReason),
		"max_session_duration": duration,
		"command_blocking":     splitPamPatterns(policies[pamPolicyCommandBlocking]),
	}

	empty := true
	for _, value := range values {
		if !value.IsNull() {
			empty = false
		}
	}
	if empty && prior.IsNull() {
		return types.ObjectNull(pamTemplatePoliciesAttrTypes)
	}

	return types.ObjectValueMust(pamTemplatePoliciesAttrTypes, values)
}

func pamScheduleToAPI(ctx context.Context, schedule types.Object, diags *diag.Diagnostics) map[string]any {
	var model pamTemplateScheduleModel
	diags.Append(schedule.As(ctx, &model, basetypes.ObjectAsOptions{})...)

	var interval any
	if !model.IntervalSeconds.IsNull() {
		interval = model.IntervalSeconds.ValueInt64()
	}
	return map[string]any{"enabled": model.Enabled.ValueBool(), "intervalSeconds": interval}
}

func pamScheduleFromAPI(value any) types.Object {
	schedule, ok := value.(map[string]any)
	if !ok {
		return types.ObjectNull(pamTemplateScheduleAttrTypes)
	}

	enabled, _ := schedule["enabled"].(bool)
	interval := types.Int64Null()
	if seconds, ok := schedule["intervalSeconds"].(float64); ok {
		interval = types.Int64Value(int64(seconds))
	}
	return types.ObjectValueMust(pamTemplateScheduleAttrTypes, map[string]attr.Value{
		"enabled":          types.BoolValue(enabled),
		"interval_seconds": interval,
	})
}

// Settings are sent whole since the API replaces them. The recording connection is a separate field.
func pamSettingsToAPI(ctx context.Context, settingsObject types.Object) (map[string]any, *string, diag.Diagnostics) {
	var diags diag.Diagnostics
	var settings pamTemplateSettingsModel
	diags.Append(settingsObject.As(ctx, &settings, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, nil, diags
	}

	result := map[string]any{}
	var recordingConnectionID *string

	var recording pamTemplateRecordingModel
	diags.Append(settings.Recording.As(ctx, &recording, basetypes.ObjectAsOptions{})...)
	result["recordingEnabled"] = recording.Enabled.ValueBool()
	result["recordingStorageBackend"] = recording.StorageBackend.ValueString()
	recordingConnectionID = optionalString(recording.ConnectionID)
	if !recording.S3.IsNull() {
		var s3 pamTemplateRecordingS3Model
		diags.Append(recording.S3.As(ctx, &s3, basetypes.ObjectAsOptions{})...)
		s3Config := map[string]any{"bucket": s3.Bucket.ValueString()}
		if region := optionalString(s3.Region); region != nil {
			s3Config["region"] = *region
		}
		if prefix := optionalString(s3.KeyPrefix); prefix != nil {
			s3Config["keyPrefix"] = *prefix
		}
		result["recordingS3Config"] = s3Config
	}

	var masking pamTemplateSessionLogMaskingModel
	diags.Append(settings.SessionLogMasking.As(ctx, &masking, basetypes.ObjectAsOptions{})...)
	if !masking.Patterns.IsNull() {
		result["sessionLogMaskingPatterns"] = joinPamPatterns(ctx, masking.Patterns, &diags)
	}
	result["sessionLogMaskingBuiltInDetection"] = masking.BuiltInDetection.ValueBool()

	result["heartbeat"] = pamScheduleToAPI(ctx, settings.Heartbeat, &diags)
	if !settings.Rotation.IsNull() {
		result["rotation"] = pamScheduleToAPI(ctx, settings.Rotation, &diags)
	}

	if !settings.PasswordRequirements.IsNull() {
		var requirements pamTemplatePasswordRequirementsModel
		diags.Append(settings.PasswordRequirements.As(ctx, &requirements, basetypes.ObjectAsOptions{})...)
		var required pamTemplatePasswordRequiredModel
		diags.Append(requirements.Required.As(ctx, &required, basetypes.ObjectAsOptions{})...)
		passwordRequirements := map[string]any{
			"length": requirements.Length.ValueInt64(),
			"required": map[string]any{
				"digits":    required.Digits.ValueInt64(),
				"lowercase": required.Lowercase.ValueInt64(),
				"uppercase": required.Uppercase.ValueInt64(),
				"symbols":   required.Symbols.ValueInt64(),
			},
		}
		if symbols := optionalString(requirements.AllowedSymbols); symbols != nil {
			passwordRequirements["allowedSymbols"] = *symbols
		}
		result["passwordRequirements"] = passwordRequirements
	}

	return result, recordingConnectionID, diags
}

// Missing values read as the API's defaults, except a missing heartbeat, which means health checks are off.
func pamSettingsFromAPI(template infisical.PamAccountTemplate) types.Object {
	settings := template.Settings

	enabled := true
	if value, ok := settings["recordingEnabled"].(bool); ok {
		enabled = value
	}
	backend := pamRecordingBackendPostgres
	if value, ok := settings["recordingStorageBackend"].(string); ok && value != "" {
		backend = value
	}
	s3 := types.ObjectNull(pamTemplateRecordingS3AttrTypes)
	if s3Config, ok := settings["recordingS3Config"].(map[string]any); ok {
		bucket, _ := s3Config["bucket"].(string)
		region, _ := s3Config["region"].(string)
		keyPrefix, _ := s3Config["keyPrefix"].(string)
		s3 = types.ObjectValueMust(pamTemplateRecordingS3AttrTypes, map[string]attr.Value{
			"bucket":     types.StringValue(bucket),
			"region":     stringOrNull(&region),
			"key_prefix": stringOrNull(&keyPrefix),
		})
	}
	recording := types.ObjectValueMust(pamTemplateRecordingAttrTypes, map[string]attr.Value{
		"enabled":         types.BoolValue(enabled),
		"storage_backend": types.StringValue(backend),
		"connection_id":   stringOrNull(template.RecordingConnectionID),
		"s3":              s3,
	})

	builtInDetection, _ := settings["sessionLogMaskingBuiltInDetection"].(bool)
	masking := types.ObjectValueMust(pamTemplateSessionLogMaskingAttrTypes, map[string]attr.Value{
		"patterns":           splitPamPatterns(settings["sessionLogMaskingPatterns"]),
		"built_in_detection": types.BoolValue(builtInDetection),
	})

	heartbeat := pamScheduleFromAPI(settings["heartbeat"])
	if heartbeat.IsNull() {
		heartbeat = types.ObjectValueMust(pamTemplateScheduleAttrTypes, map[string]attr.Value{
			"enabled":          types.BoolValue(false),
			"interval_seconds": types.Int64Null(),
		})
	}

	passwordRequirements := types.ObjectNull(pamTemplatePasswordRequirementsAttrTypes)
	if requirements, ok := settings["passwordRequirements"].(map[string]any); ok {
		count := func(values map[string]any, key string) types.Int64 {
			value, _ := values[key].(float64)
			return types.Int64Value(int64(value))
		}
		required, _ := requirements["required"].(map[string]any)
		allowedSymbols, _ := requirements["allowedSymbols"].(string)
		passwordRequirements = types.ObjectValueMust(pamTemplatePasswordRequirementsAttrTypes, map[string]attr.Value{
			"length": count(requirements, "length"),
			"required": types.ObjectValueMust(pamTemplatePasswordRequiredAttrTypes, map[string]attr.Value{
				"digits":    count(required, "digits"),
				"lowercase": count(required, "lowercase"),
				"uppercase": count(required, "uppercase"),
				"symbols":   count(required, "symbols"),
			}),
			"allowed_symbols": stringOrNull(&allowedSymbols),
		})
	}

	return types.ObjectValueMust(pamTemplateSettingsAttrTypes, map[string]attr.Value{
		"recording":             recording,
		"session_log_masking":   masking,
		"heartbeat":             heartbeat,
		"rotation":              pamScheduleFromAPI(settings["rotation"]),
		"password_requirements": passwordRequirements,
	})
}

func (r *pamAccountTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to create PAM account template") {
		return
	}

	var plan pamAccountTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, diags := pamPoliciesToAPI(ctx, plan.Policies)
	resp.Diagnostics.Append(diags...)
	settings, recordingConnectionID, diags := pamSettingsToAPI(ctx, plan.Settings)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.CreatePamAccountTemplate(infisical.CreatePamAccountTemplateRequest{
		Name:                  plan.Name.ValueString(),
		Description:           optionalString(plan.Description),
		Type:                  plan.AccountType.ValueString(),
		Policies:              policies,
		Settings:              settings,
		GatewayID:             optionalString(plan.GatewayID),
		GatewayPoolID:         optionalString(plan.GatewayPoolID),
		RecordingConnectionID: recordingConnectionID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating PAM account template", "Couldn't create PAM account template, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(template.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pamAccountTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to read PAM account template") {
		return
	}

	var state pamAccountTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.GetPamAccountTemplate(infisical.GetPamAccountTemplateRequest{ID: state.ID.ValueString()})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PAM account template", "Couldn't read PAM account template, unexpected error: "+err.Error())
		return
	}

	state.Name = types.StringValue(template.Name)
	state.Description = stringOrNull(template.Description)
	state.AccountType = types.StringValue(template.Type)
	state.GatewayID = stringOrNull(template.GatewayID)
	state.GatewayPoolID = stringOrNull(template.GatewayPoolID)
	state.Policies = pamPoliciesFromAPI(template.Policies, state.Policies)
	state.Settings = pamSettingsFromAPI(template)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *pamAccountTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to update PAM account template") {
		return
	}

	var plan, state pamAccountTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, diags := pamPoliciesToAPI(ctx, plan.Policies)
	resp.Diagnostics.Append(diags...)
	settings, recordingConnectionID, diags := pamSettingsToAPI(ctx, plan.Settings)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdatePamAccountTemplate(infisical.UpdatePamAccountTemplateRequest{
		ID:                    state.ID.ValueString(),
		Name:                  plan.Name.ValueString(),
		Description:           optionalString(plan.Description),
		Policies:              policies,
		Settings:              settings,
		GatewayID:             optionalString(plan.GatewayID),
		GatewayPoolID:         optionalString(plan.GatewayPoolID),
		RecordingConnectionID: recordingConnectionID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating PAM account template", "Couldn't update PAM account template, unexpected error: "+err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pamAccountTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !requirePamMachineIdentity(r.client, &resp.Diagnostics, "Unable to delete PAM account template") {
		return
	}

	var state pamAccountTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePamAccountTemplate(infisical.DeletePamAccountTemplateRequest{ID: state.ID.ValueString()})
	if err != nil && err != infisical.ErrNotFound {
		resp.Diagnostics.AddError("Error deleting PAM account template", "Couldn't delete PAM account template, unexpected error: "+err.Error())
	}
}

func (r *pamAccountTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
