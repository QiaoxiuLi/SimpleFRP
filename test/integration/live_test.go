package integration

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

// This opt-in test uses a disposable, explicitly authorized native installation.
func TestLiveClientWorkflow(t *testing.T) {
	file := os.Getenv("SIMPLEFRP_LIVE_INVITE_FILE")
	if file == "" {
		t.Skip("explicit live acceptance inputs were not supplied")
	}
	binary, err := filepath.Abs(os.Getenv("SIMPLEFRP_TEST_BINARY"))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := sysutil.LoadReceipt()
	if err != nil || receipt.Role != "client" || !strings.EqualFold(receipt.Binary, binary) {
		t.Fatal("refusing to test an unverified or non-client native installation")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal("private invitation file unavailable")
	}
	inviteText := strings.TrimSpace(string(data))
	invite, err := crypto.DecodeInvite(inviteText)
	if err != nil {
		t.Fatal("invalid private invitation")
	}
	client := installation{binary: binary, t: t}
	t.Cleanup(func() {
		if _, err := sysutil.LoadReceipt(); err == nil {
			if out, err := client.execute("uninstall"); err != nil {
				t.Errorf("native uninstall failed: %v: %s", err, out)
			}
		}
	})
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	go func() {
		for {
			conn, err := target.Accept()
			if err != nil {
				return
			}
			go func() { defer conn.Close(); io.Copy(conn, conn) }()
		}
	}()
	local := target.Addr().(*net.TCPAddr).Port
	client.must("set", inviteText)
	first := client.waitOnline().Tunnels[0]
	preferred := 0
	if value := os.Getenv("SIMPLEFRP_LIVE_PUBLIC_PORT"); value != "" {
		preferred, err = strconv.Atoi(value)
		if err != nil || preferred < 1025 || preferred > 65535 {
			t.Fatal("invalid explicitly allowed acceptance port")
		}
		client.must(fmt.Sprint(first.ID), "public", fmt.Sprint(preferred))
	}
	originalPublic := first.PublicPort
	if preferred != 0 {
		first.PublicPort = preferred
	}
	client.must(fmt.Sprint(first.ID), "local", fmt.Sprint(local))
	forwardHost(t, invite.IP, first.PublicPort)
	if client.state().Tunnels[0].TotalBytes == 0 {
		t.Fatal("native transferred bytes were not counted")
	}
	client.must("set", inviteText)
	if len(client.state().Tunnels) != 1 {
		t.Fatal("native repeated pairing duplicated tunnels")
	}
	client.must("next")
	tunnels := client.state().Tunnels
	if len(tunnels) != 2 {
		t.Fatal("native next did not create exactly one tunnel")
	}
	second := tunnels[1]
	client.must(fmt.Sprint(second.ID), "local", fmt.Sprint(local))
	if preferred != 0 {
		// Reuse only the existing SimpleFRP cloud rule, never widen server exposure.
		client.must(fmt.Sprint(first.ID), "public", fmt.Sprint(originalPublic))
		client.must(fmt.Sprint(second.ID), "public", fmt.Sprint(preferred))
		second.PublicPort = preferred
	}
	forwardHost(t, invite.IP, second.PublicPort)
	if preferred != 0 {
		client.must("delete", fmt.Sprint(second.ID))
	}
	updated := preferred != 0
	if preferred != 0 {
		client.must(fmt.Sprint(first.ID), "public", fmt.Sprint(preferred))
	}
	for attempt := 0; !updated && attempt < 5; attempt++ {
		candidate := availablePort(t)
		if _, err := client.execute(fmt.Sprint(first.ID), "public", fmt.Sprint(candidate)); err == nil {
			first.PublicPort = candidate
			updated = true
			break
		}
	}
	if !updated {
		t.Fatal("native public port update failed")
	}
	forwardHost(t, invite.IP, first.PublicPort)
	if preferred == 0 {
		client.must("delete", fmt.Sprint(second.ID))
	}
	if len(client.state().Tunnels) != 1 {
		t.Fatal("native tunnel deletion failed")
	}
	// Ending the status viewer must not affect its separate forwarding daemon.
	if sysutil.ProcessRunning("monitor") {
		if err := sysutil.StopOwnedProcess("monitor"); err != nil {
			t.Fatal(err)
		}
		if !sysutil.ProcessRunning("daemon") {
			t.Fatal("closing status viewer stopped forwarding daemon")
		}
		forwardHost(t, invite.IP, first.PublicPort)
	}
	before := client.state().Tunnels[0].TotalBytes
	time.Sleep(1100 * time.Millisecond)
	if after := client.state().Tunnels[0].TotalBytes; after < before {
		t.Fatal("native status lost cumulative traffic")
	}
	client.must("uninstall")
	for _, dir := range []string{sysutil.ConfigDir(), sysutil.DataDir(), sysutil.LogDir()} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatal("native uninstall left runtime files")
		}
	}
}
