package cli

import (
	"fmt"
	"github.com/simplefrp/simplefrp/internal/app"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/service"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/spf13/cobra"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"
)

func runCommand() *cobra.Command {
	var ip string
	cmd := &cobra.Command{Use: "run", Short: "Initialize once and start this installation", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		role, _ := app.DetectRole()
		if role == app.RoleServer {
			if runtime.GOOS != "linux" && os.Getenv("SIMPLEFRP_TEST_SERVER") != "1" {
				return fmt.Errorf("the server is supported only on Linux")
			}
			cfg, err := app.BootstrapServer(ip)
			if err != nil {
				return err
			}
			if err = startConfigured(role); err != nil {
				return err
			}
			e := frpwrap.NewAdmin(cfg)
			deadline := time.Now().Add(8 * time.Second)
			for {
				if _, err = e.Request(protocol.Message{Type: protocol.TypeStatus}); err == nil {
					break
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("server could not start; configuration preserved: %w", err)
				}
				time.Sleep(200 * time.Millisecond)
			}
			code, err := app.ConnectionCode(cfg)
			if err != nil {
				return err
			}
			fmt.Printf("Heartbeat port: %d\nDashboard: http://127.0.0.1:%d\nFirst public port: %d\nConnection string (keep private):\n%s\n", cfg.ControlPort, cfg.DashboardPort, cfg.FirstPublicPort, code)
			fmt.Println("Allow the heartbeat and public tunnel ports in your firewall/security group. The dashboard is local-only.")
			return nil
		}
		if err := startConfigured(app.RoleClient); err != nil {
			return err
		}
		fmt.Println("Client running. Connect with simplefrp set <SF2-connection-string>.")
		return service.OpenStatus()
	}}
	cmd.Flags().StringVar(&ip, "public-ip", "", "explicit server IP when automatic discovery is unavailable")
	return cmd
}
func startConfigured(role app.Role) error {
	if os.Getenv("SIMPLEFRP_HOME") != "" {
		return service.StartIsolated(role)
	}
	if err := service.Enable(role); err != nil {
		return err
	}
	return service.Start(role)
}
func setCommand() *cobra.Command {
	return &cobra.Command{Use: "set <connection-string>", Short: "Pair this client with a server", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		role, _ := app.DetectRole()
		if role == app.RoleServer {
			return fmt.Errorf("set is a client command")
		}
		invite, err := crypto.DecodeInvite(args[0])
		if err != nil {
			return err
		}
		if _, err = os.Stat(sysutil.ClientConfigPath()); err == nil {
			previous, loadErr := config.LoadClient()
			if loadErr != nil {
				return loadErr
			}
			if previous.ServerID == invite.ServerID {
				if previous.Fingerprint != invite.Fingerprint {
					return fmt.Errorf("the certificate fingerprint for this server changed; existing pairing was preserved")
				}
				candidate := previous
				candidate.ServerAddress = invite.Address()
				resp, err := frpwrap.NewClientEngine(candidate).Request(protocol.Message{Type: protocol.TypeStatus})
				if err != nil {
					return fmt.Errorf("existing configuration preserved: %w", err)
				}
				candidate.ServerAddress = net.JoinHostPort(invite.IP, strconv.Itoa(resp.Status.HeartbeatPort))
				candidate.Tunnels = nil
				for _, t := range resp.Status.Tunnels {
					candidate.Tunnels = append(candidate.Tunnels, config.Tunnel{ID: t.ID, LocalPort: t.LocalPort, PublicPort: t.PublicPort, Status: t.Status})
				}
				if err = config.SaveClient(candidate); err != nil {
					return err
				}
				if err = startConfigured(app.RoleClient); err != nil {
					return err
				}
				fmt.Println("Existing pairing reused; no extra tunnel created.")
				return service.OpenStatus()
			}
			return fmt.Errorf("this client is already paired to another server; existing configuration was preserved")
		}
		local, err := sysutil.RandomAvailablePort()
		if err != nil {
			return err
		}
		cfg, err := frpwrap.Pair(invite, local)
		if err != nil {
			return err
		}
		if err = config.SaveClient(cfg); err != nil {
			return err
		}
		if err = app.WriteRole(app.RoleClient); err != nil {
			return err
		}
		if err = startConfigured(app.RoleClient); err != nil {
			return err
		}
		fmt.Printf("Paired. Tunnel %d: public %d -> local %d\nThe local target may not be running yet.\n", cfg.Tunnels[0].ID, cfg.Tunnels[0].PublicPort, cfg.Tunnels[0].LocalPort)
		return service.OpenStatus()
	}}
}
func nextCommand() *cobra.Command {
	return &cobra.Command{Use: "next", Short: "Create an automatically allocated tunnel", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		role, _ := app.DetectRole()
		if role == app.RoleServer {
			return fmt.Errorf("create a new tunnel on the client with simplefrp next")
		}
		local, err := sysutil.RandomAvailablePort()
		if err != nil {
			return err
		}
		resp, err := request(protocol.Message{Type: protocol.TypeCreateTunnel, LocalPort: local})
		if err != nil {
			return err
		}
		fmt.Printf("Tunnel %d: public %d -> local %d\n", resp.TunnelID, resp.PublicPort, resp.LocalPort)
		return nil
	}}
}
func deleteCommand() *cobra.Command {
	return &cobra.Command{Use: "delete <tunnel-id>", Short: "Delete a tunnel on either peer", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if _, err = request(protocol.Message{Type: protocol.TypeDeleteTunnel, TunnelID: id}); err != nil {
			return err
		}
		fmt.Printf("Tunnel %d deleted.\n", id)
		return nil
	}}
}
