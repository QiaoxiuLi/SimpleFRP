package cli

import (
	"fmt"
	"github.com/simplefrp/simplefrp/internal/app"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/service"
	"github.com/simplefrp/simplefrp/internal/ux"
	"github.com/spf13/cobra"
	"net"
	"strconv"
	"strings"
)

func setupCommand() *cobra.Command {
	var role, server string
	var local, public, control int
	cmd := &cobra.Command{Use: "setup", Short: "Install startup support and configure a working server or client", RunE: func(cmd *cobra.Command, args []string) error {
		if role == "server" {
			if cmd.Flags().Changed("control-port") && (control < 1025 || control > 65535) {
				return fmt.Errorf("control port must be 1025–65535")
			}
			if public != 0 && (public < 1025 || public > 65535) {
				return fmt.Errorf("public port must be 1025–65535")
			}
			password, err := ux.AskSecret("Server password (8+ letters/numbers; hidden):")
			if err != nil {
				return err
			}
			if err = app.BootstrapServer(password); err != nil {
				return err
			}
			cfg, err := config.LoadServer()
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("control-port") {
				if control < 1025 || control > 65535 {
					return fmt.Errorf("control port must be 1025–65535")
				}
				cfg.ControlPort = control
			}
			if public != 0 {
				if public < 1025 || public > 65535 {
					return fmt.Errorf("public port must be 1025–65535")
				}
				cfg.PortMin = public
				cfg.PortMax = public
			}
			if err = config.SaveServer(cfg); err != nil {
				return err
			}
			if err = restartConfiguredService(app.RoleServer); err != nil {
				return err
			}
			fmt.Printf("Server started. Control: %d; public range: %d–%d; dashboard: 127.0.0.1:%d\n", cfg.ControlPort, cfg.PortMin, cfg.PortMax, cfg.DashboardPort)
			fmt.Println("Allow the control port and chosen public ports in your cloud firewall; existing rules were not changed.")
			return nil
		}
		if role != "client" {
			return fmt.Errorf("role must be server or client")
		}
		cfg, err := config.LoadClient()
		if err != nil {
			cfg = config.DefaultClientConfig()
		}
		if server == "" {
			server, err = ux.Ask("Server address (host:port, default control port 8388):")
			if err != nil {
				return err
			}
		}
		if !strings.Contains(server, ":") {
			server = net.JoinHostPort(server, "8388")
		}
		host, port, err := net.SplitHostPort(server)
		if err != nil || host == "" {
			return fmt.Errorf("enter a valid server host:port")
		}
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return fmt.Errorf("invalid server port")
		}
		password, err := ux.AskSecret("Server password (hidden):")
		if err != nil {
			return err
		}
		if local == 0 {
			answer, askErr := ux.Ask("Local service port [3000]:")
			if askErr != nil {
				return askErr
			}
			local = 3000
			if answer != "" {
				local, err = strconv.Atoi(answer)
				if err != nil {
					return err
				}
			}
		}
		if local < 1 || local > 65535 {
			return fmt.Errorf("local port must be 1–65535")
		}
		candidate := cfg
		candidate.ServerAddress = server
		candidate.PasswordKey = crypto.RandomPasswordKey(password, config.SharedAuthSalt)
		// Reuse the existing first mapping for repeated setup against the same server.
		id := 0
		if cfg.ServerAddress == server && len(cfg.Tunnels) > 0 {
			id = cfg.Tunnels[0].ID
			if public == 0 {
				public = cfg.Tunnels[0].PublicPort
			}
		}
		response, err := frpwrap.NewClientEngine(candidate).CreateTunnel(id, local, public)
		if err != nil {
			return fmt.Errorf("server connection failed; existing configuration kept: %w", err)
		}
		if !response.OK {
			return fmt.Errorf("server rejected configuration: %s", response.Message)
		}
		tunnel := config.Tunnel{ID: id, LocalPort: local, PublicPort: response.PublicPort, Status: "success"}
		if cfg.ServerAddress != server || len(cfg.Tunnels) == 0 {
			candidate.Tunnels = []config.Tunnel{tunnel}
		} else {
			candidate.Tunnels = append([]config.Tunnel(nil), cfg.Tunnels...)
			candidate.Tunnels[0] = tunnel
		}
		if err = config.SaveClient(candidate); err != nil {
			return err
		}
		if err = app.WriteRole(app.RoleClient); err != nil {
			return err
		}
		if err = restartConfiguredService(app.RoleClient); err != nil {
			return err
		}
		fmt.Printf("Client started. %s:%d -> 127.0.0.1:%d\n", host, tunnel.PublicPort, local)
		return nil
	}}
	cmd.Flags().StringVar(&role, "role", "client", "server or client")
	cmd.Flags().StringVar(&server, "server", "", "server host:port (no password)")
	cmd.Flags().IntVar(&local, "local-port", 0, "local service port, including an already-listening port")
	cmd.Flags().IntVar(&public, "public-port", 0, "requested public port; server setup restricts allocation to this port")
	cmd.Flags().IntVar(&control, "control-port", 8388, "server control port")
	return cmd
}
func restartConfiguredService(role app.Role) error {
	if err := service.Enable(role); err != nil {
		return fmt.Errorf("configuration saved; autostart setup failed: %w", err)
	}
	if err := service.Stop(role); err != nil {
		return fmt.Errorf("configuration saved; previous daemon could not stop: %w", err)
	}
	if err := service.Start(role); err != nil {
		return fmt.Errorf("configuration saved; start manually with simplefrp daemon --role %s: %w", role, err)
	}
	return nil
}
