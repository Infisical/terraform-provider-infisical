package customtypes

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// MongoDBConnectionStringType is a string type whose values are compared after being normalized
// the way the API stores them.
//
// Without it, a configured "mongodb://db:27017" comes back as "mongodb://db:27017/" and every
// plan re-proposes the same change forever.
type MongoDBConnectionStringType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = MongoDBConnectionStringType{}

func (t MongoDBConnectionStringType) Equal(o attr.Type) bool {
	other, ok := o.(MongoDBConnectionStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t MongoDBConnectionStringType) String() string {
	return "customtypes.MongoDBConnectionStringType"
}

func (t MongoDBConnectionStringType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return MongoDBConnectionStringValue{StringValue: in}, nil
}

func (t MongoDBConnectionStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}

func (t MongoDBConnectionStringType) ValueType(_ context.Context) attr.Value {
	return MongoDBConnectionStringValue{}
}

// MongoDBConnectionStringValue is the value type produced by MongoDBConnectionStringType.
type MongoDBConnectionStringValue struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuable                   = MongoDBConnectionStringValue{}
	_ basetypes.StringValuableWithSemanticEquals = MongoDBConnectionStringValue{}
)

// NewMongoDBConnectionStringValue returns a known MongoDBConnectionStringValue for the given string.
func NewMongoDBConnectionStringValue(value string) MongoDBConnectionStringValue {
	return MongoDBConnectionStringValue{StringValue: basetypes.NewStringValue(value)}
}

// NewMongoDBConnectionStringNull returns a null MongoDBConnectionStringValue.
func NewMongoDBConnectionStringNull() MongoDBConnectionStringValue {
	return MongoDBConnectionStringValue{StringValue: basetypes.NewStringNull()}
}

func (v MongoDBConnectionStringValue) Type(_ context.Context) attr.Type {
	return MongoDBConnectionStringType{}
}

func (v MongoDBConnectionStringValue) Equal(o attr.Value) bool {
	other, ok := o.(MongoDBConnectionStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v MongoDBConnectionStringValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(MongoDBConnectionStringValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("expected value type %T but got %T", v, newValuable),
		)
		return false, diags
	}

	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return false, diags
	}

	return NormalizeMongoDBConnectionString(v.ValueString()) == NormalizeMongoDBConnectionString(newValue.ValueString()), diags
}

// splitMongoDBConnectionString splits a connection string into its scheme, its hosts, which run
// until the first "/" or "?" as in the API's parser, and the rest.
func splitMongoDBConnectionString(raw string) (scheme, hosts, rest string, ok bool) {
	scheme, afterScheme, found := strings.Cut(raw, "://")
	if !found || (scheme != "mongodb" && scheme != "mongodb+srv") {
		return "", "", "", false
	}
	end := strings.IndexAny(afterScheme, "/?")
	if end == -1 {
		return scheme, afterScheme, "", true
	}
	return scheme, afterScheme[:end], afterScheme[end:], true
}

// ValidateMongoDBConnectionString reports what the API would reject about a connection string, so
// the refusal lands at plan time.
func ValidateMongoDBConnectionString(raw string) error {
	trimmed := strings.TrimSpace(raw)
	// The API's parser drops everything after a line break and strips tabs.
	if strings.ContainsAny(trimmed, "\t\n\r") {
		return fmt.Errorf("must be on one line, with no tabs")
	}
	scheme, hosts, rest, ok := splitMongoDBConnectionString(trimmed)
	if !ok {
		return fmt.Errorf("must start with mongodb:// or mongodb+srv://")
	}
	// The API's parser reads everything before an "@" as credentials, wherever the "@" appears.
	if strings.Contains(hosts+rest, "@") {
		return fmt.Errorf("must not include credentials, which go in the credentials attribute")
	}
	if hosts == "" {
		return fmt.Errorf("must include a host")
	}
	if scheme == "mongodb+srv" && strings.ContainsAny(hosts, ",:") {
		return fmt.Errorf("must name a single host with no port when it uses mongodb+srv://")
	}
	path := rest
	if end := strings.IndexAny(rest, "?#"); end != -1 {
		path = rest[:end]
	}
	if path != "" && path != "/" {
		return fmt.Errorf("must not include a database, which goes in the database attribute")
	}
	return nil
}

// encodeMongoDBOptions percent-encodes the characters the API's URL serializer encodes after the
// hosts: controls, spaces, quotes, angle brackets and non-ASCII bytes. Existing escapes are kept.
func encodeMongoDBOptions(options string) string {
	var encoded strings.Builder
	for i := 0; i < len(options); i++ {
		c := options[i]
		if c <= ' ' || c > '~' || c == '"' || c == '<' || c == '>' {
			fmt.Fprintf(&encoded, "%%%02X", c)
			continue
		}
		encoded.WriteByte(c)
	}
	return encoded.String()
}

// NormalizeMongoDBConnectionString mirrors how the API stores a connection string: trimmed, with a
// "/" after the hosts, and the options and fragment percent-encoded.
func NormalizeMongoDBConnectionString(raw string) string {
	trimmed := strings.TrimSpace(raw)
	scheme, hosts, rest, ok := splitMongoDBConnectionString(trimmed)
	if !ok {
		return trimmed
	}

	if !strings.HasPrefix(rest, "/") {
		rest = "/" + rest
	}
	return scheme + "://" + hosts + encodeMongoDBOptions(rest)
}
