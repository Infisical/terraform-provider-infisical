package resource

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPaloAltoNetworksCredentialsPayload(t *testing.T) {
	payload := paloAltoNetworksCredentialsPayload(AppConnectionPaloAltoNetworksCredentialsModel{
		Hostname:              types.StringValue("fw.example.com"),
		Port:                  types.Int32Value(8443),
		Username:              types.StringValue("admin"),
		Password:              types.StringValue("secret"),
		SslRejectUnauthorized: types.BoolValue(false),
		SslCertificate:        types.StringValue("PEM"),
	})
	want := map[string]any{
		"hostname":              "fw.example.com",
		"port":                  int32(8443),
		"username":              "admin",
		"password":              "secret",
		"sslRejectUnauthorized": false,
		"sslCertificate":        "PEM",
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload = %v, want %v", payload, want)
	}

	payload = paloAltoNetworksCredentialsPayload(AppConnectionPaloAltoNetworksCredentialsModel{
		Hostname:              types.StringValue("fw.example.com"),
		Port:                  types.Int32Null(),
		Username:              types.StringValue("admin"),
		Password:              types.StringValue("secret"),
		SslRejectUnauthorized: types.BoolNull(),
		SslCertificate:        types.StringNull(),
	})
	want = map[string]any{"hostname": "fw.example.com", "username": "admin", "password": "secret"}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload without optional fields = %v, want %v", payload, want)
	}
}
