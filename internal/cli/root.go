package cli

import (
	"fmt"
	"github.com/simplefrp/simplefrp/internal/app"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

const Version = "0.2.0"

var instance string

func Execute() error {
	var home string
	root := &cobra.Command{Use: "simplefrp", Short: "SimpleFRP TCP forwarding", Version: Version, Args: cobra.ArbitraryArgs, PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if home != "" {
			absolute, err := sysutil.SafeHome(home)
			if err != nil {
				return err
			}
			os.Setenv("SIMPLEFRP_HOME", absolute)
		}
		return nil
	}, RunE: routeImplicitTunnelCommand}
	root.PersistentFlags().StringVar(&home, "home", "", "isolated installation home")
	root.PersistentFlags().StringVar(&instance, "instance", "", "installation process identity")
	root.PersistentFlags().MarkHidden("instance")
	root.AddCommand(runCommand(), setCommand(), stateCommand(), nextCommand(), deleteCommand(), portCommand(), uninstallCommand(), daemonCommand(), installCommand(), desktopCommand())
	return root.Execute()
}
func engine() (frpwrap.ClientEngine, error) {
	role, _ := app.DetectRole()
	if role == app.RoleServer {
		cfg, err := config.LoadServer()
		return frpwrap.NewAdmin(cfg), err
	}
	cfg, err := config.LoadClient()
	return frpwrap.NewClientEngine(cfg), err
}
func request(msg protocol.Message) (protocol.Response, error) {
	e, err := engine()
	if err != nil {
		return protocol.Response{}, err
	}
	return e.Request(msg)
}
func routeImplicitTunnelCommand(cmd *cobra.Command, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("use simplefrp <tunnel-id> local|public <port>, or simplefrp --help")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil || id < 0 {
		return fmt.Errorf("invalid tunnel ID")
	}
	port, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("invalid port")
	}
	if args[1] == "local" {
		return updateLocalPort(id, port)
	}
	if args[1] == "public" {
		return updatePublicPort(id, port)
	}
	return fmt.Errorf("use local or public")
}
func updateLocalPort(id, port int) error  { return updatePort(id, "local", port) }
func updatePublicPort(id, port int) error { return updatePort(id, "public", port) }
func updatePort(id int, kind string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be 1-65535")
	}
	resp, err := request(protocol.Message{Type: protocol.TypeUpdate, TunnelID: id, Kind: kind, Port: port})
	if err != nil {
		return fmt.Errorf("port update failed; existing mapping retained: %w", err)
	}
	fmt.Printf("Tunnel %d: public %d -> local %d\n", resp.TunnelID, resp.PublicPort, resp.LocalPort)
	return nil
}
func portCommand() *cobra.Command {
	return &cobra.Command{Use: "port dashboard|heartbeat <port>", Short: "Modify an existing server dashboard or heartbeat port",
		Long:    "Modify one of the two existing server service ports. This command does not create a tunnel or add a forwarding port. Use 'simplefrp next' to create a tunnel, and 'simplefrp <tunnel-id> public|local <port>' to modify its ports.",
		Example: "  simplefrp port dashboard 29181\n  simplefrp port heartbeat 29182",
		Args:    cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
			role, _ := app.DetectRole()
			if role != app.RoleServer {
				return fmt.Errorf("this command is only available on the Linux server")
			}
			port, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			if _, err = request(protocol.Message{Type: protocol.TypePort, Kind: args[0], Port: port}); err != nil {
				return err
			}
			cfg, err := config.LoadServer()
			if err != nil {
				return err
			}
			fmt.Printf("Heartbeat port: %d\nDashboard: http://127.0.0.1:%d\n", cfg.ControlPort, cfg.DashboardPort)
			if args[0] == "heartbeat" {
				code, err := app.ConnectionCode(cfg)
				if err != nil {
					return err
				}
				fmt.Printf("Updated connection string (keep private):\n%s\n", code)
			}
			fmt.Println("Check the firewall/security group for the new port; no unrelated rules were changed.")
			return nil
		}}
}
