package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type CertificateSyncAwsElasticLoadBalancerListenerModel struct {
	ListenerArn types.String `tfsdk:"listener_arn"`
	Port        types.Int64  `tfsdk:"port"`
	Protocol    types.String `tfsdk:"protocol"`
}

type CertificateSyncAwsElasticLoadBalancerDestinationConfigModel struct {
	Region          types.String `tfsdk:"aws_region"`
	LoadBalancerArn types.String `tfsdk:"load_balancer_arn"`
	Listeners       types.List   `tfsdk:"listeners"`
}

type CertificateSyncAwsElasticLoadBalancerSyncOptionsModel struct {
	CertificateNameSchema customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa         types.Bool                     `tfsdk:"include_root_ca"`
	PreserveArn           types.Bool                     `tfsdk:"preserve_arn"`
}

var certificateSyncAwsElasticLoadBalancerListenerAttrTypes = map[string]attr.Type{
	"listener_arn": types.StringType,
	"port":         types.Int64Type,
	"protocol":     types.StringType,
}

var certificateSyncAwsElasticLoadBalancerDestinationConfigAttrTypes = map[string]attr.Type{
	"aws_region":        types.StringType,
	"load_balancer_arn": types.StringType,
	"listeners":         types.ListType{ElemType: types.ObjectType{AttrTypes: certificateSyncAwsElasticLoadBalancerListenerAttrTypes}},
}

var certificateSyncAwsElasticLoadBalancerSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema": customtypes.TrimmedStringType{},
	"can_remove_certificates": types.BoolType,
	"include_root_ca":         types.BoolType,
	"preserve_arn":            types.BoolType,
}

func NewCertificateSyncAwsElasticLoadBalancerResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:                        infisical.CertificateSyncAppAWSElasticLoadBalancer,
		SyncName:                   "AWS Elastic Load Balancer",
		ResourceTypeName:           "_certificate_sync_aws_elastic_load_balancer",
		CertificateNameRule:        &certificateNameRule{pattern: regexp.MustCompile(`^[a-zA-Z0-9\s_-]{1,256}$`), requireLiteralCertificateID: true, requirement: "1-256 letters, digits, spaces, hyphens or underscores"},
		AppConnection:              infisical.AppConnectionAppAWS,
		SupportsDefaultCertificate: true,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"aws_region": schema.StringAttribute{
				Required:    true,
				Description: "The AWS region of the load balancer (e.g. us-east-1).",
			},
			"load_balancer_arn": schema.StringAttribute{
				Required:    true,
				Description: "The ARN of the load balancer to sync certificates to.",
			},
			"listeners": schema.ListNestedAttribute{
				Required:    true,
				Description: "The HTTPS or TLS listeners on the load balancer to attach synced certificates to. At least one listener is required.",
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"listener_arn": schema.StringAttribute{
							Required:    true,
							Description: "The ARN of the listener.",
						},
						"port": schema.Int64Attribute{
							Required:    true,
							Description: "The port the listener serves on (e.g. 443).",
						},
						"protocol": schema.StringAttribute{
							Required:    true,
							Description: "The protocol of the listener (e.g. HTTPS or TLS).",
						},
					},
				},
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Validators:  []validator.String{notBlank()},
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for certificates imported into ACM for the listeners. When set, it must include the {{certificateId}} placeholder; {{shortCertificateId}} is not supported for this destination. Names may contain letters, digits, spaces, hyphens and underscores (1-256 characters). When unset, the sync holds only one certificate.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove certificates from the load balancer when they are no longer managed in Infisical. Defaults to false.",
				Default:     booldefault.StaticBool(false),
			},
			"include_root_ca": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to include the root CA certificate in the synced certificate chain. Defaults to false.",
				Default:     booldefault.StaticBool(false),
			},
			"preserve_arn": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to preserve the certificate ARN when a certificate is renewed, reimporting into the existing certificate instead of creating a new one. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
		},

		ValidateConfigFunc: validateAwsElasticLoadBalancerListeners,

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncAwsElasticLoadBalancerSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			payload := map[string]interface{}{
				"canRemoveCertificates": syncOptions.CanRemoveCertificates.ValueBool(),
				"includeRootCa":         syncOptions.IncludeRootCa.ValueBool(),
				"preserveArn":           syncOptions.PreserveArn.ValueBool(),
				"canImportCertificates": false,
			}
			setOptionalString(payload, "certificateNameSchema", syncOptions.CertificateNameSchema)
			return payload, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			return types.ObjectValue(certificateSyncAwsElasticLoadBalancerSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema": optionalTrimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema"),
				"can_remove_certificates": boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", false),
				"include_root_ca":         boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
				"preserve_arn":            boolFromMap(certificateSync.SyncOptions, "preserveArn", true),
			})
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncAwsElasticLoadBalancerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var listeners []CertificateSyncAwsElasticLoadBalancerListenerModel
			diags.Append(destinationConfig.Listeners.ElementsAs(ctx, &listeners, false)...)
			if diags.HasError() {
				return nil, diags
			}

			listenersPayload := make([]map[string]interface{}, 0, len(listeners))
			for _, listener := range listeners {
				listenersPayload = append(listenersPayload, map[string]interface{}{
					"listenerArn": listener.ListenerArn.ValueString(),
					"port":        listener.Port.ValueInt64(),
					"protocol":    listener.Protocol.ValueString(),
				})
			}

			return map[string]interface{}{
				"region":          destinationConfig.Region.ValueString(),
				"loadBalancerArn": destinationConfig.LoadBalancerArn.ValueString(),
				"listeners":       listenersPayload,
			}, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			listenerObjectType := types.ObjectType{AttrTypes: certificateSyncAwsElasticLoadBalancerListenerAttrTypes}

			region := stringFromMap(certificateSync.DestinationConfig, "region", &diags)
			loadBalancerArn := stringFromMap(certificateSync.DestinationConfig, "loadBalancerArn", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncAwsElasticLoadBalancerDestinationConfigAttrTypes), diags
			}

			rawListeners, ok := certificateSync.DestinationConfig["listeners"].([]interface{})
			if !ok {
				diags.AddError("Invalid listeners type", "Expected 'listeners' to be an array but got something else")
				return types.ObjectNull(certificateSyncAwsElasticLoadBalancerDestinationConfigAttrTypes), diags
			}

			listenerValues := make([]attr.Value, 0, len(rawListeners))
			for i, rawListener := range rawListeners {
				listener, ok := rawListener.(map[string]interface{})
				if !ok {
					diags.AddError("Invalid listener type", fmt.Sprintf("Expected listener %d to be an object but got something else", i))
					return types.ObjectNull(certificateSyncAwsElasticLoadBalancerDestinationConfigAttrTypes), diags
				}

				listenerArn := stringFromMap(listener, "listenerArn", &diags)
				protocol := stringFromMap(listener, "protocol", &diags)
				port := optionalInt64FromMap(listener, "port")
				if port.IsNull() {
					diags.AddError("Invalid port type", "Expected 'port' to be a number but got something else")
				}
				if diags.HasError() {
					return types.ObjectNull(certificateSyncAwsElasticLoadBalancerDestinationConfigAttrTypes), diags
				}

				listenerValue, objDiags := types.ObjectValue(certificateSyncAwsElasticLoadBalancerListenerAttrTypes, map[string]attr.Value{
					"listener_arn": listenerArn,
					"port":         port,
					"protocol":     protocol,
				})
				diags.Append(objDiags...)
				if diags.HasError() {
					return types.ObjectNull(certificateSyncAwsElasticLoadBalancerDestinationConfigAttrTypes), diags
				}
				listenerValues = append(listenerValues, listenerValue)
			}

			listeners, listDiags := types.ListValue(listenerObjectType, listenerValues)
			diags.Append(listDiags...)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncAwsElasticLoadBalancerDestinationConfigAttrTypes), diags
			}

			obj, objDiags := types.ObjectValue(certificateSyncAwsElasticLoadBalancerDestinationConfigAttrTypes, map[string]attr.Value{
				"aws_region":        region,
				"load_balancer_arn": loadBalancerArn,
				"listeners":         listeners,
			})
			diags.Append(objDiags...)
			return obj, diags
		},
	}
}

// validateAwsElasticLoadBalancerListeners checks that every listener belongs to the configured load
// balancer and region; the sync only uses the listener ARNs, so a stray one gets the certificates.
func validateAwsElasticLoadBalancerListeners(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
	if config.DestinationConfig.IsNull() || config.DestinationConfig.IsUnknown() {
		return
	}
	var destinationConfig CertificateSyncAwsElasticLoadBalancerDestinationConfigModel
	diags.Append(config.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)...)
	if diags.HasError() || destinationConfig.LoadBalancerArn.IsUnknown() || destinationConfig.LoadBalancerArn.IsNull() || destinationConfig.Listeners.IsUnknown() {
		return
	}

	destinationPath := path.Root(attrDestinationConfig)
	loadBalancerArn := destinationConfig.LoadBalancerArn.ValueString()
	arnParts := strings.SplitN(loadBalancerArn, ":", 6)
	if len(arnParts) != 6 || arnParts[0] != "arn" || arnParts[2] != "elasticloadbalancing" || !strings.HasPrefix(arnParts[5], "loadbalancer/") {
		diags.AddAttributeError(destinationPath.AtName("load_balancer_arn"), "Invalid load balancer ARN",
			"load_balancer_arn must be an Elastic Load Balancing load balancer ARN, for example arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188.")
		return
	}
	if !destinationConfig.Region.IsUnknown() && !destinationConfig.Region.IsNull() && arnParts[3] != destinationConfig.Region.ValueString() {
		diags.AddAttributeError(destinationPath.AtName("load_balancer_arn"), "Load balancer region mismatch",
			fmt.Sprintf("load_balancer_arn is in %s but aws_region is %s.", arnParts[3], destinationConfig.Region.ValueString()))
	}

	var listeners []CertificateSyncAwsElasticLoadBalancerListenerModel
	diags.Append(destinationConfig.Listeners.ElementsAs(ctx, &listeners, false)...)
	if diags.HasError() {
		return
	}
	listenerPrefix := strings.Replace(loadBalancerArn, ":loadbalancer/", ":listener/", 1) + "/"
	seen := map[string]bool{}
	for i, listener := range listeners {
		if listener.ListenerArn.IsUnknown() || listener.ListenerArn.IsNull() {
			continue
		}
		arn := listener.ListenerArn.ValueString()
		listenerPath := destinationPath.AtName("listeners").AtListIndex(i).AtName("listener_arn")
		if !strings.HasPrefix(arn, listenerPrefix) {
			diags.AddAttributeError(listenerPath, "Listener not on this load balancer",
				fmt.Sprintf("Listener %q does not belong to load balancer %q. The sync attaches certificates to every listener ARN given, so a listener from another load balancer would receive them.", arn, loadBalancerArn))
		}
		if seen[arn] {
			diags.AddAttributeError(listenerPath, "Duplicate listener", fmt.Sprintf("Listener %q is listed more than once.", arn))
		}
		seen[arn] = true
	}
}
