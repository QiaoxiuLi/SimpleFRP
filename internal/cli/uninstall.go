package cli

import (
	"fmt"
	"os"

	"github.com/simplefrp/simplefrp/internal/app"
	"github.com/simplefrp/simplefrp/internal/service"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/spf13/cobra"
)

func uninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall SimpleFRP",
		RunE: func(cmd *cobra.Command, args []string) error {
			role, _ := app.DetectRole()
			_ = service.Disable(role)
			_ = service.Stop(role)
			_ = sysutil.StopSimpleFRPProcesses()
			_ = os.RemoveAll(sysutil.ConfigDir())
			_ = os.RemoveAll(sysutil.DataDir())
			_ = os.RemoveAll(sysutil.LogDir())
			if role == app.RoleServer {
				fmt.Println("SimpleFRP server has been uninstalled successfully.")
				fmt.Println("All SimpleFRP ports and processes have been released.")
			} else {
				fmt.Println("SimpleFRP client has been uninstalled successfully.")
				fmt.Println("All SimpleFRP tunnels, ports, and processes have been released.")
			}
			return nil
		},
	}
}
