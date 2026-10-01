package cli

import (
	"context"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/app"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/service"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func installCommand() *cobra.Command {
	var role string
	var noStart bool
	var prepareUpgrade bool
	cmd := &cobra.Command{Use: "install", Hidden: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if role != "client" && role != "server" {
			return fmt.Errorf("invalid role")
		}
		if role == "server" && runtime.GOOS != "linux" && os.Getenv("SIMPLEFRP_TEST_SERVER") != "1" {
			return fmt.Errorf("only Linux supports the server")
		}
		if _, err := os.Stat(sysutil.ServerConfigPath()); err == nil {
			if _, err = config.LoadServer(); err != nil {
				return err
			}
		}
		if _, err := os.Stat(sysutil.ClientConfigPath()); err == nil {
			if _, err = config.LoadClient(); err != nil {
				return err
			}
		}
		if _, statErr := os.Stat(sysutil.ReceiptPath()); statErr == nil {
			existing, err := sysutil.LoadReceipt()
			if err != nil {
				return err
			}
			if existing.Role != role {
				return fmt.Errorf("this installation already has a different role; it was not changed")
			}
		}
		if os.Getenv("SIMPLEFRP_HOME") == "" {
			if err := service.ValidateStartup(app.Role(role)); err != nil {
				return err
			}
		}
		if prepareUpgrade {
			r, err := sysutil.LoadReceipt()
			if err != nil {
				return err
			}
			if os.Getenv("SIMPLEFRP_HOME") == "" {
				err = service.Stop(app.Role(r.Role))
			} else {
				err = sysutil.StopOwnedProcess("daemon")
			}
			if err != nil {
				return err
			}
			if err = sysutil.StopOwnedProcess("monitor"); err != nil {
				return err
			}
			return nil
		}
		if err := sysutil.EnsureBaseDirs(); err != nil {
			return err
		}
		if err := app.WriteRole(app.Role(role)); err != nil {
			return err
		}
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		r := sysutil.Receipt{Version: 2, Role: role, Binary: exe, OwnBinary: service.OwnsInstalledBinary(exe)}
		if previous, loadErr := sysutil.LoadReceipt(); loadErr == nil {
			r.PathAdded = previous.PathAdded
			r.PathFile = previous.PathFile
			r.PathFileCreated = previous.PathFileCreated
			r.ServiceUID = previous.ServiceUID
			r.OwnServiceUser = previous.OwnServiceUser
			r.OwnServiceGroup = previous.OwnServiceGroup
		}
		if os.Getenv("SIMPLEFRP_HOME") == "" {
			if err = service.PrepareAccount(&r); err != nil {
				return err
			}
		}
		if err = config.WriteJSON(sysutil.ReceiptPath(), r); err != nil {
			return err
		}
		if os.Getenv("SIMPLEFRP_HOME") == "" {
			if err = service.ConfigureCommandPath(); err != nil {
				return err
			}
			if err = service.Enable(app.Role(role)); err != nil {
				return err
			}
		}
		if role == "client" && !noStart {
			if err = startConfigured(app.RoleClient); err != nil {
				return err
			}
			if err = service.OpenStatus(); err != nil {
				return err
			}
		}
		fmt.Printf("Installed SimpleFRP %s (%s).\n", Version, role)
		return nil
	}}
	cmd.Flags().StringVar(&role, "role", "client", "installation role")
	cmd.Flags().BoolVar(&noStart, "no-start", false, "configure startup without launching")
	cmd.Flags().BoolVar(&prepareUpgrade, "prepare-upgrade", false, "stop this verified installation before replacement")
	cmd.Flags().MarkHidden("prepare-upgrade")
	return cmd
}
func desktopCommand() *cobra.Command {
	return &cobra.Command{Use: "desktop", Hidden: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if err := startConfigured(app.RoleClient); err != nil {
			return err
		}
		return service.OpenStatus()
	}}
}
func uninstallCommand() *cobra.Command {
	return &cobra.Command{Use: "uninstall", Short: "Remove only this installation and its startup entries", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		r, err := sysutil.LoadReceipt()
		if err != nil {
			return fmt.Errorf("refusing unverified or legacy uninstall; no processes or files were changed: %w", err)
		}
		if os.Getenv("SIMPLEFRP_HOME") == "" {
			if err = service.ValidateStartup(app.Role(r.Role)); err != nil {
				return err
			}
		}
		if r.Role == "client" {
			if cfg, loadErr := config.LoadClient(); loadErr == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_, cleanupErr := frpwrap.NewClientEngine(cfg).RequestContext(ctx, protocol.Message{Type: protocol.TypeUnpair})
				cancel()
				if cleanupErr != nil {
					fmt.Fprintln(os.Stderr, "Server cleanup could not be confirmed. Local uninstall will continue; use server state/delete to remove any remaining tunnels.")
				}
			}
		}
		if os.Getenv("SIMPLEFRP_HOME") == "" {
			if err = service.ValidateStartup(app.Role(r.Role)); err != nil {
				return err
			}
			if err = service.Stop(app.Role(r.Role)); err != nil {
				return err
			}
			if err = service.Disable(app.Role(r.Role)); err != nil {
				return err
			}
		}
		for _, kind := range []string{"monitor", "daemon"} {
			if err = sysutil.StopOwnedProcess(kind); err != nil {
				return err
			}
		}
		if os.Getenv("SIMPLEFRP_HOME") == "" {
			if err = service.RemoveCommandPath(r); err != nil {
				return err
			}
			if err = service.RemoveAccount(r); err != nil {
				return err
			}
		}
		if os.Getenv("SIMPLEFRP_HOME") == "" {
			if err = service.RemoveInstalledBinary(r); err != nil {
				return err
			}
		} else if r.OwnBinary {
			root, _ := sysutil.SafeHome(os.Getenv("SIMPLEFRP_HOME"))
			relative, relErr := filepath.Rel(root, r.Binary)
			if relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) && relative != "." {
				if err = service.RemoveInstalledBinary(r); err != nil {
					return err
				}
			}
		}
		for _, dir := range []string{sysutil.LogDir(), sysutil.DataDir(), sysutil.ConfigDir()} {
			if err = os.RemoveAll(dir); err != nil {
				return fmt.Errorf("cleanup incomplete for %s: %w", dir, err)
			}
		}
		if home := os.Getenv("SIMPLEFRP_HOME"); home != "" {
			// Remove the dedicated home only when empty; unrelated user files stay.
			_ = os.Remove(home)
		}
		fmt.Println("This SimpleFRP installation, its startup entries and owned runtime files were removed.")
		return nil
	}}
}
