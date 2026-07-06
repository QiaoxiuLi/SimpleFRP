package cli

import (
	"fmt"

	"github.com/simplefrp/simplefrp/internal/app"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/service"
	"github.com/simplefrp/simplefrp/internal/ux"
	"github.com/spf13/cobra"
)

func setCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set server",
		Short: "Reset client server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "server" {
				return cmd.Help()
			}
			addr, err := ux.Ask(ux.ServerPrompt)
			if err != nil {
				return err
			}
			password, err := ux.Ask(ux.PasswordShortPrompt)
			if err != nil {
				return err
			}
			cfg, err := config.LoadClient()
			if err != nil {
				cfg = config.DefaultClientConfig()
			}
			cfg.ServerAddress = addr
			cfg.PasswordKey = crypto.RandomPasswordKey(password, config.SharedAuthSalt)
			cfg.Tunnels = nil
			if err := config.SaveClient(cfg); err != nil {
				return err
			}
			if err := createNextTunnel(); err != nil {
				return err
			}
			_ = service.Enable(app.RoleClient)
			_ = service.Stop(app.RoleClient)
			_ = service.Start(app.RoleClient)
			fmt.Println()
			fmt.Println("Server updated successfully.")
			fmt.Println("Connected to SimpleFRP server successfully.")
			return nil
		},
	}
}
