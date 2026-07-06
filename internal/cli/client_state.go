package cli

import (
	"fmt"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/spf13/cobra"
)

func stateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "state",
		Short: "Show client tunnel state",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadClient()
			if err != nil {
				return err
			}
			fmt.Println("Tunnel ID    Local Port    Public Port    Status")
			for _, t := range cfg.Tunnels {
				status := t.Status
				if status != "success" {
					status = "failed"
				}
				fmt.Printf("%-12d %-13d %-14d %s\n", t.ID, t.LocalPort, t.PublicPort, status)
			}
			return nil
		},
	}
}
