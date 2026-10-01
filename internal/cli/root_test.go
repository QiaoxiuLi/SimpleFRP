package cli

import (
	"bytes"
	"encoding/json"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDisconnectPreservesCachedCounters(t *testing.T) {
	t.Setenv("SIMPLEFRP_HOME", t.TempDir())
	if err := sysutil.EnsureBaseDirs(); err != nil {
		t.Fatal(err)
	}
	saveCachedStatus(protocol.Status{Online: true, Tunnels: []protocol.TunnelStatus{{ID: 4, TotalBytes: 54321, BytesPerSecond: 512}}})
	saveCachedStatus(protocol.Status{Tunnels: []protocol.TunnelStatus{{ID: 4, Status: "offline"}}})
	b, err := os.ReadFile(filepath.Join(sysutil.DataDir(), "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cached protocol.Status
	if err := json.Unmarshal(b, &cached); err != nil {
		t.Fatal(err)
	}
	if cached.Online || cached.Tunnels[0].TotalBytes != 54321 || cached.Tunnels[0].BytesPerSecond != 0 {
		t.Fatal("offline status lost counters or retained a live speed")
	}
}

func TestRepeatedPairingRejectsChangedCertificate(t *testing.T) {
	t.Setenv("SIMPLEFRP_HOME", t.TempDir())
	previous := config.ClientConfig{Version: 2, ServerID: "0011223344556677", Fingerprint: strings.Repeat("a", 64), ServerAddress: "127.0.0.1:30001"}
	if err := config.SaveClient(previous); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(sysutil.ClientConfigPath())
	code, err := crypto.EncodeInvite(crypto.Invite{IP: "127.0.0.1", Port: 30002, ServerID: previous.ServerID, Key: "AAAAAAAAAAAAAAAAAAAAAA", Fingerprint: strings.Repeat("b", 64)})
	if err != nil {
		t.Fatal(err)
	}
	cmd := setCommand()
	cmd.SetArgs([]string{code})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("changed trust pin was not rejected: %v", err)
	}
	after, _ := os.ReadFile(sysutil.ClientConfigPath())
	if !bytes.Equal(before, after) {
		t.Fatal("failed pairing changed the existing configuration")
	}
}

func TestServicePortHelpDistinguishesTunnelCreation(t *testing.T) {
	cmd := portCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"does not create a tunnel", "simplefrp next", "simplefrp <tunnel-id> public|local <port>"} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("port help does not explain %q", text)
		}
	}
}

func TestLegacyConfigurationIsNotOverwritten(t *testing.T) {
	t.Setenv("SIMPLEFRP_HOME", t.TempDir())
	if err := sysutil.EnsureBaseDirs(); err != nil {
		t.Fatal(err)
	}
	original := []byte("server_address = \"old.example:7000\"\npassword_key = \"legacy\"\n")
	if err := os.WriteFile(sysutil.ClientConfigPath(), original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.LoadClient(); err == nil {
		t.Fatal("legacy config accepted")
	}
	got, _ := os.ReadFile(sysutil.ClientConfigPath())
	if string(got) != string(original) {
		t.Fatal("legacy config was altered")
	}
	if err := updateLocalPort(0, 0); err == nil {
		t.Fatal("invalid port accepted")
	}
}
