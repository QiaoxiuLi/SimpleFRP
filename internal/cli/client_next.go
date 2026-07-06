package cli

import (
	"fmt"
	"strings"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/simplefrp/simplefrp/internal/ux"
	"github.com/spf13/cobra"
)

func nextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Create next tunnel",
		RunE: func(cmd *cobra.Command, args []string) error {
			answer, err := ux.Ask("Create next tunnel? (y/n)")
			if err != nil {
				return err
			}
			if strings.ToLower(answer) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}
			return createNextTunnel()
		},
	}
}

func createNextTunnel() error {
	cfg, err := config.LoadClient()
	if err != nil {
		cfg = config.DefaultClientConfig()
	}
	local, err := sysutil.RandomAvailablePort()
	if err != nil {
		return fmt.Errorf(ux.Failure("Tunnel creation failed.", "No available local port was found on this computer.", "Try again later or close unused local services.", err.Error()))
	}
	id := config.NextTunnelID(cfg.Tunnels)
	public := 0
	if cfg.ServerAddress != "" {
		resp, err := frpwrap.NewClientEngine(cfg).CreateTunnel(id, local, 0)
		if err != nil {
			return fmt.Errorf(ux.Failure("Tunnel creation failed.", "The SimpleFRP server is unreachable.", "Check the server address, firewall, and server service status.", err.Error()))
		}
		if !resp.OK {
			return fmt.Errorf(ux.Failure("Tunnel creation failed.", resp.Message, resp.Suggestion, resp.Code))
		}
		public = resp.PublicPort
	}
	if public == 0 {
		public, err = sysutil.RandomAvailablePort()
		if err != nil {
			return fmt.Errorf(ux.Failure("Tunnel creation failed.", "No available public port was found on the server.", "Try again later.", err.Error()))
		}
	}
	t := config.Tunnel{ID: id, LocalPort: local, PublicPort: public, Status: "success"}
	cfg.Tunnels = append(cfg.Tunnels, t)
	if err := config.SaveClient(cfg); err != nil {
		return err
	}
	fmt.Printf("Tunnel created successfully.\nTunnel ID: %d\nLocal port: %d\nPublic port: %d\nStatus: success\n", t.ID, t.LocalPort, t.PublicPort)
	return nil
}
