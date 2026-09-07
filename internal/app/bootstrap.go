package app

import (
	"fmt"
	"os"

	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/sysutil"
)

func BootstrapServer(password string) error {
	if err := sysutil.EnsureBaseDirs(); err != nil {
		return err
	}
	if err := WriteRole(RoleServer); err != nil {
		return err
	}
	cfg, err := config.NewServerConfig(password)
	if err != nil {
		return err
	}
	if previous, loadErr := config.LoadServer(); loadErr == nil {
		previous.PasswordHash = cfg.PasswordHash
		previous.AuthKey = cfg.AuthKey
		cfg = previous
	}
	if err := config.SaveServer(cfg); err != nil {
		return err
	}
	return nil
}

func BootstrapClient() error {
	if err := sysutil.EnsureBaseDirs(); err != nil {
		return err
	}
	if err := WriteRole(RoleClient); err != nil {
		return err
	}
	if _, err := os.Stat(sysutil.ClientConfigPath()); os.IsNotExist(err) {
		return config.SaveClient(config.DefaultClientConfig())
	}
	return nil
}

func ServerSuccessMessage() string {
	return `SimpleFRP server configuration completed successfully.

Automatically configured:
- Created SimpleFRP system user
- Created configuration directory
- Created data directory
- Created log directory
- Generated server configuration
- Initialized local statistics database
- Registered systemd service
- Enabled autostart
- Started SimpleFRP server
- Checked server heartbeat port
- Checked dashboard port
- Checked service status

Server heartbeat port: 8388
Dashboard port: 8387
Service status: running
Autostart: enabled`
}

func PrintDefaultPorts() {
	fmt.Println("SimpleFRP Server Port: 8388")
	fmt.Println("SimpleFRP Dashboard Port: 8387")
}
