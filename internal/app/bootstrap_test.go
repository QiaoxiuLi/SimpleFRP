package app

import "testing"

func TestExplicitIPv6UsesMatchingListener(t *testing.T) {
	t.Setenv("SIMPLEFRP_HOME", t.TempDir())
	cfg, err := BootstrapServer("::1")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BindAddress != "::" {
		t.Fatal("IPv6 invitation cannot be served by an IPv4-only listener")
	}
}

func TestInvalidPublicEndpointIsRejected(t *testing.T) {
	for _, ip := range []string{"0.0.0.0", "::", "224.0.0.1", "ff02::1"} {
		t.Run(ip, func(t *testing.T) {
			t.Setenv("SIMPLEFRP_HOME", t.TempDir())
			if _, err := BootstrapServer(ip); err == nil {
				t.Fatal("unusable endpoint accepted")
			}
		})
	}
}
