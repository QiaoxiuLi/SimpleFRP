package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/simplefrp/simplefrp/internal/config"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"
)

func saveCachedStatus(status protocol.Status) {
	path := filepath.Join(sysutil.DataDir(), "status.json")
	if !status.Online {
		var previous protocol.Status
		b, _ := os.ReadFile(path)
		_ = json.Unmarshal(b, &previous)
		for i := range status.Tunnels {
			for _, old := range previous.Tunnels {
				if status.Tunnels[i].ID == old.ID && old.TotalBytes > status.Tunnels[i].TotalBytes {
					status.Tunnels[i].TotalBytes = old.TotalBytes
				}
			}
		}
	}
	_ = config.WriteJSON(path, status)
}

func parseID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil || id < 0 {
		return 0, fmt.Errorf("invalid tunnel ID")
	}
	return id, nil
}
func stateCommand() *cobra.Command {
	var watch, jsonOutput bool
	cmd := &cobra.Command{Use: "state", Short: "Show live tunnel ports, rate and cumulative traffic", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		clearTerminal := watch && configureTerminal()
		if watch {
			release, err := sysutil.ProcessLock("monitor")
			if err != nil {
				return err
			}
			defer release()
			if err = sysutil.WriteProcess("monitor", instance); err != nil {
				return err
			}
			defer sysutil.RemoveProcess("monitor")
		}
		for {
			resp, err := request(protocol.Message{Type: protocol.TypeStatus})
			var status protocol.Status
			if err == nil && resp.Status != nil {
				status = *resp.Status
				saveCachedStatus(status)
			} else {
				b, _ := os.ReadFile(filepath.Join(sysutil.DataDir(), "status.json"))
				_ = json.Unmarshal(b, &status)
				status.Online = false
				for i := range status.Tunnels {
					status.Tunnels[i].Status = "offline"
					status.Tunnels[i].BytesPerSecond = 0
				}
			}
			if clearTerminal {
				fmt.Print("\033[H\033[2J")
			}
			if jsonOutput {
				json.NewEncoder(os.Stdout).Encode(status)
			} else {
				fmt.Printf("SimpleFRP %s\nHeartbeat port: %s\n", Version, portLabel(status.HeartbeatPort))
				if status.DashboardPort != 0 {
					fmt.Printf("Dashboard: http://127.0.0.1:%d\n", status.DashboardPort)
				}
				fmt.Println("Tunnel   Public   Local    Speed (B/s)    Total (bytes)   State")
				for _, t := range status.Tunnels {
					fmt.Printf("%-8d %-8s %-8s %-14.0f %-15d %s\n", t.ID, portLabel(t.PublicPort), portLabel(t.LocalPort), t.BytesPerSecond, t.TotalBytes, t.Status)
				}
				if !status.Online {
					fmt.Println("Offline or not configured; cached counters only.")
				}
				if len(status.Tunnels) == 0 {
					if status.DashboardPort != 0 {
						fmt.Println("No tunnels. Waiting for a client to pair.")
					} else {
						fmt.Println("No tunnels. Pair with simplefrp set <connection-string>.")
					}
				}
			}
			if !watch {
				return nil
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
		}
	}}
	cmd.Flags().BoolVar(&watch, "watch", false, "refresh live status")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "machine-readable status")
	return cmd
}
func portLabel(port int) string {
	if port == 0 {
		return "not set"
	}
	return strconv.Itoa(port)
}
