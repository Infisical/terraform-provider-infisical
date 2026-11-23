package customtypes

import "testing"

// Pairs of a configured value and what the API stored for it, taken from the
// mongodb-connection-string-url package the API normalizes with.
func TestNormalizeMongoDBConnectionStringMatchesTheApi(t *testing.T) {
	for _, pair := range [][2]string{
		{"mongodb://localhost:27017", "mongodb://localhost:27017/"},
		{"mongodb://h1,h2?replicaSet=rs0", "mongodb://h1,h2/?replicaSet=rs0"},
		{"mongodb+srv://cluster0.abcde.mongodb.net", "mongodb+srv://cluster0.abcde.mongodb.net/"},
		{"mongodb://LocalHost:27017/?authSource=admin", "mongodb://LocalHost:27017/?authSource=admin"},
		{"mongodb://db:27017/?tls=true&appName=My App", "mongodb://db:27017/?tls=true&appName=My%20App"},
		{"mongodb://db:27017/?appName=café", "mongodb://db:27017/?appName=caf%C3%A9"},
		{"mongodb://db:27017/?appName=a+b", "mongodb://db:27017/?appName=a+b"},
		{"mongodb://[::1]:27017", "mongodb://[::1]:27017/"},
		{" mongodb://db:27017\n", "mongodb://db:27017/"},
	} {
		if got, want := NormalizeMongoDBConnectionString(pair[0]), NormalizeMongoDBConnectionString(pair[1]); got != want {
			t.Errorf("%q normalized to %q, but the stored %q normalized to %q", pair[0], got, pair[1], want)
		}
	}
}

func TestNormalizeMongoDBConnectionStringKeepsRealChanges(t *testing.T) {
	for _, pair := range [][2]string{
		{"mongodb://db1:27017", "mongodb://db2:27017/"},
		{"mongodb://db:27017/?tls=true", "mongodb://db:27017/?tls=false"},
		{"mongodb://db:27017", "mongodb+srv://db/"},
	} {
		if NormalizeMongoDBConnectionString(pair[0]) == NormalizeMongoDBConnectionString(pair[1]) {
			t.Errorf("%q and %q were treated as the same connection string", pair[0], pair[1])
		}
	}
}

func TestValidateMongoDBConnectionStringRejectsWhatTheApiRejects(t *testing.T) {
	for _, raw := range []string{
		"MONGODB://db",
		"postgres://db:5432",
		"mongodb://",
		"mongodb://user:pass@db:27017",
		"mongodb://db/?appName=a@b",
		"mongodb://db:27017/orders",
		"mongodb+srv://cluster0.example.com:27017",
		"mongodb+srv://a.example.com,b.example.com",
	} {
		if err := ValidateMongoDBConnectionString(raw); err == nil {
			t.Errorf("%q was accepted", raw)
		}
	}
}

func TestValidateMongoDBConnectionStringAcceptsWhatTheApiAccepts(t *testing.T) {
	for _, raw := range []string{
		"mongodb://localhost:27017",
		"mongodb://h1:27017,h2:27017/?replicaSet=rs0",
		"mongodb+srv://cluster0.abcde.mongodb.net/?retryWrites=true&w=majority",
		"mongodb://[::1]:27017",
		"mongodb://db:27017/?w=majority#frag",
	} {
		if err := ValidateMongoDBConnectionString(raw); err != nil {
			t.Errorf("%q was rejected: %v", raw, err)
		}
	}
}
