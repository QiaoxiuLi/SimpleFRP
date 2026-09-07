package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/simplefrp/simplefrp/internal/ux"
	"github.com/spf13/cobra"
)

func Execute() error {
	root := &cobra.Command{
		Use:   "simplefrp",
		Short: "SimpleFRP TCP forwarding tool",
		Args:  cobra.ArbitraryArgs,
		RunE:  routeImplicitTunnelCommand,
	}
	root.AddCommand(pwdCommand(), stateCommand(), nextCommand(), setCommand(), deleteCommand(), uninstallCommand(), daemonCommand(), setupCommand())
	return root.Execute()
}

func routeImplicitTunnelCommand(cmd *cobra.Command, args []string) error {
	if len(args) != 3 {
		return cmd.Help()
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf(ux.Failure("Port update failed.", "Tunnel ID must be a number.", `Run "simplefrp state" to view existing tunnels.`, ""))
	}
	port, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf(ux.Failure("Port update failed.", "Port must be a number.", "Choose a numeric port and try again.", ""))
	}
	switch args[1] {
	case "local":
		return updateLocalPort(id, port)
	case "public":
		return updatePublicPort(id, port)
	default:
		return cmd.Help()
	}
}

func updateLocalPort(id, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf(ux.Failure("Local port update failed.", "The requested port is invalid or reserved.", "Choose another port and try again.", "port must be between 1 and 65535"))
	}
	cfg, err := config.LoadClient()
	if err != nil {
		return err
	}
	t, idx, ok := config.FindTunnel(cfg.Tunnels, id)
	if !ok {
		return fmt.Errorf(ux.Failure("Local port update failed.", fmt.Sprintf("Tunnel ID %d does not exist.", id), `Run "simplefrp state" to view existing tunnels.`, ""))
	}
	t.LocalPort = port
	t.Status = "success"
	cfg.Tunnels[idx] = t
	if err := config.SaveClient(cfg); err != nil {
		return err
	}
	fmt.Printf("Local port updated successfully.\nTunnel ID: %d\nLocal port: %d\nPublic port: %d\nStatus: success\n", t.ID, t.LocalPort, t.PublicPort)
	return nil
}

func updatePublicPort(id, port int) error {
	if err := sysutil.ValidatePort(port); err != nil {
		return fmt.Errorf(ux.Failure("Public port update failed.", "The requested port is invalid or reserved.", "Choose another public port and try again.", err.Error()))
	}
	cfg, err := config.LoadClient()
	if err != nil {
		return err
	}
	t, idx, ok := config.FindTunnel(cfg.Tunnels, id)
	if !ok {
		return fmt.Errorf(ux.Failure("Public port update failed.", fmt.Sprintf("Tunnel ID %d does not exist.", id), `Run "simplefrp state" to view existing tunnels.`, ""))
	}
	if t.PublicPort == port {
		return nil
	}
	t.PublicPort = port
	t.Status = "success"
	if cfg.ServerAddress != "" {
		resp, err := frpwrap.NewClientEngine(cfg).CreateTunnel(t.ID, t.LocalPort, port)
		if err != nil || !resp.OK {
			return fmt.Errorf(ux.Failure("Public port update failed.", "The server could not reserve the requested public port.", "Choose another public port and try again.", ""))
		}
	}
	cfg.Tunnels[idx] = t
	if err := config.SaveClient(cfg); err != nil {
		return err
	}
	fmt.Printf("Public port updated successfully.\nTunnel ID: %d\nLocal port: %d\nPublic port: %d\nStatus: success\n", t.ID, t.LocalPort, t.PublicPort)
	return nil
}

func mustWritableConfigDir() error {
	if err := os.MkdirAll(sysutil.ConfigDir(), 0750); err != nil {
		return fmt.Errorf(ux.Failure("Operation failed.", "SimpleFRP cannot write its configuration directory.", "Run the command with administrator privileges.", err.Error()))
	}
	return nil
}
