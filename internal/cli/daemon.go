package cli

import (
	"fmt"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

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
				engine := frpwrap.NewServerEngine(cfg, store)
				defer engine.Close()
				errors := make(chan error, 2)
				go func() { errors <- engine.ListenAndServe() }()
				go func() { errors <- dashboard.New(cfg, store).ListenAndServe() }()
				return <-errors
			}
			_, err := config.LoadClient()
			if err != nil {
				return err
			}
			if runtime.GOOS == "windows" {
				if err = sysutil.EnsureBaseDirs(); err != nil {
					return err
				}
				if err = os.WriteFile(filepath.Join(sysutil.DataDir(), "client.pid"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
					return err
				}
			}
			fmt.Println("SimpleFRP client daemon started.")
			return frpwrap.RunConfiguredClient(config.LoadClient)
		},
	}
	cmd.Flags().StringVar(&role, "role", "client", "daemon role: server or client")
	return cmd
}
