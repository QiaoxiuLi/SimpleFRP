package cli

import (
	"fmt"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/dashboard"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/storage"
	"github.com/spf13/cobra"
)

func daemonCommand() *cobra.Command {
	var role string
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run SimpleFRP daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			if role == "server" {
				cfg, err := config.LoadServer()
				if err != nil {
					return err
				}
				store, err := storage.OpenDefault()
				if err != nil {
					return err
				}
				defer store.Close()
				go func() {
					_ = frpwrap.NewServerEngine(cfg, store).ListenAndServe()
				}()
				return dashboard.New(cfg, store).ListenAndServe()
			}
			cfg, err := config.LoadClient()
			if err != nil {
				return err
			}
			fmt.Println("SimpleFRP client daemon started.")
			return frpwrap.NewClientEngine(cfg).Run(cfg.Tunnels)
		},
	}
	cmd.Flags().StringVar(&role, "role", "client", "daemon role: server or client")
	return cmd
}
