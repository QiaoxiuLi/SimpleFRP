package cli

import (
	"fmt"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/ux"
	"github.com/spf13/cobra"
)

func deleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <tunnel-id>",
		Short: "Delete tunnel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var id int
			if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
				return fmt.Errorf(ux.Failure("Tunnel deletion failed.", "Tunnel ID must be a number.", `Run "simplefrp state" to view existing tunnels.`, ""))
			}
			cfg, err := config.LoadClient()
			if err != nil {
				return err
			}
			t, idx, ok := config.FindTunnel(cfg.Tunnels, id)
			if !ok {
				return fmt.Errorf(ux.Failure("Tunnel deletion failed.", fmt.Sprintf("Tunnel ID %d does not exist.", id), `Run "simplefrp state" to view existing tunnels.`, ""))
			}
			if cfg.ServerAddress != "" {
				_, _ = frpwrap.NewClientEngine(cfg).DeleteTunnel(t.ID, t.PublicPort)
			}
			cfg.Tunnels = append(cfg.Tunnels[:idx], cfg.Tunnels[idx+1:]...)
			if err := config.SaveClient(cfg); err != nil {
				return err
			}
			fmt.Printf("Tunnel deleted successfully.\nTunnel ID: %d\nReleased local port: %d\nReleased public port: %d\n", t.ID, t.LocalPort, t.PublicPort)
			return nil
		},
	}
}
