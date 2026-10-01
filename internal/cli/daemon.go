package cli

import (
	"context"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/dashboard"
	"github.com/simplefrp/simplefrp/internal/frpwrap"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/storage"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"
)

func daemonCommand() *cobra.Command {
	var role string
	cmd := &cobra.Command{Use: "daemon", Hidden: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if role == "server" && runtime.GOOS != "linux" && os.Getenv("SIMPLEFRP_TEST_SERVER") != "1" {
			return fmt.Errorf("only Linux supports the server")
		}
		if err := sysutil.EnsureBaseDirs(); err != nil {
			return err
		}
		release, err := sysutil.ProcessLock("daemon")
		if err != nil {
			return err
		}
		defer release()
		if err = sysutil.WriteProcess("daemon", instance); err != nil {
			return err
		}
		defer sysutil.RemoveProcess("daemon")
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, sysutil.TerminationSignal())
		defer cancel()
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
			e := frpwrap.NewServerEngine(cfg, store)
			web := dashboard.New(cfg.DashboardBindAddress, func() protocol.Status { return e.Status("", true) })
			e.RebindDashboard = web.Start
			if err = e.Start(); err != nil {
				return err
			}
			defer e.Close()
			if err = web.Start(cfg.DashboardPort); err != nil {
				return err
			}
			defer web.Close()
			cached := make(chan struct{})
			go func() { defer close(cached); cacheStatus(ctx, func() protocol.Status { return e.Status("", true) }) }()
			<-ctx.Done()
			<-cached
			return nil
		}
		if role != "client" {
			return fmt.Errorf("invalid daemon role")
		}
		return frpwrap.RunClient(ctx, frpwrap.ClientCallbacks{Load: config.LoadClient, Save: config.SaveClient, Status: saveCachedStatus})
	}}
	cmd.Flags().StringVar(&role, "role", "client", "server or client")
	return cmd
}
func cacheStatus(ctx context.Context, status func() protocol.Status) {
	for {
		_ = config.WriteJSON(filepath.Join(sysutil.DataDir(), "status.json"), status())
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
