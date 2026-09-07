package cli

import (
	"fmt"

	"github.com/simplefrp/simplefrp/internal/app"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/service"
	"github.com/simplefrp/simplefrp/internal/ux"
	"github.com/spf13/cobra"
)

func pwdCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pwd",
		Short: "Reset server password",
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := ux.AskSecret(ux.PasswordPrompt)
			if err != nil {
				return err
			}
			if err := config.ValidatePassword(password); err != nil {
				return fmt.Errorf(ux.Failure("Password update failed.", "The new password format is invalid.", "Use at least 8 characters. Only letters and numbers are allowed.", ""))
			}
			if err := app.BootstrapServer(password); err != nil {
				return err
			}
			_ = service.Enable(app.RoleServer)
			_ = service.Stop(app.RoleServer)
			if err := service.Start(app.RoleServer); err != nil {
				fmt.Println("Password updated successfully.")
				fmt.Println("SimpleFRP server status: configured")
				fmt.Println("Start service manually if it is not running: systemctl start simplefrp-server")
				return nil
			}
			fmt.Println("Password updated successfully.")
			fmt.Println("SimpleFRP server status: running")
			return nil
		},
	}
}
