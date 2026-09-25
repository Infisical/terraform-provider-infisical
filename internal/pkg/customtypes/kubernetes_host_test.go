package customtypes

import "testing"

func TestValidateKubernetesHostAcceptsBracketedIPv6(t *testing.T) {
	for _, raw := range []string{
		"https://[2001:db8::1]:6443",
		"[2001:db8::1]:6443",
		"https://cluster.example.com:6443",
		"cluster.example.com",
	} {
		if err := ValidateKubernetesHost(raw); err != nil {
			t.Errorf("%q was rejected: %v", raw, err)
		}
	}
}

func TestValidateKubernetesHostRejectsWhatTheApiRejects(t *testing.T) {
	for _, raw := range []string{
		"http://cluster.example.com:6443",
		"https://cluster.example.com:6443/api",
		"https://user:pw@cluster.example.com:6443",
		"https://cluster.example.com:6443?x=1",
	} {
		if err := ValidateKubernetesHost(raw); err == nil {
			t.Errorf("%q was accepted", raw)
		}
	}
}

func TestNormalizeKubernetesHostKeepsIPv6Brackets(t *testing.T) {
	want := "https://[2001:db8::1]:6443"
	for _, raw := range []string{want, want + "/", "[2001:db8::1]:6443"} {
		if got := NormalizeKubernetesHost(raw); got != want {
			t.Errorf("%q normalized to %q, want %q", raw, got, want)
		}
	}
}

// AKS kubeconfigs carry an explicit :443, which the API drops because WHATWG URL treats it as
// the scheme default.
func TestNormalizeKubernetesHostDropsTheDefaultHttpsPort(t *testing.T) {
	want := "https://my-aks.azmk8s.io"
	for _, raw := range []string{
		"https://my-aks.azmk8s.io:443",
		"https://my-aks.azmk8s.io:443/",
		"my-aks.azmk8s.io:443",
		"https://MY-AKS.azmk8s.io:443",
		want,
	} {
		if got := NormalizeKubernetesHost(raw); got != want {
			t.Errorf("%q normalized to %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeKubernetesHostKeepsANonDefaultPort(t *testing.T) {
	for raw, want := range map[string]string{
		"https://cluster.example.com:6443": "https://cluster.example.com:6443",
		"https://[2001:db8::1]:443":        "https://[2001:db8::1]",
		"https://[2001:db8::1]:6443":       "https://[2001:db8::1]:6443",
	} {
		if got := NormalizeKubernetesHost(raw); got != want {
			t.Errorf("%q normalized to %q, want %q", raw, got, want)
		}
	}
}
