package cli

import "github.com/spf13/cobra"

// Keep the original command as an alias of the complete setup workflow.
// Both paths validate the server before replacing an existing configuration.
func setCommand() *cobra.Command {
	return &cobra.Command{Use: "set server", Short: "Configure this client using the setup wizard", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "server" {
			return cmd.Help()
		}
		wizard := setupCommand()
		return wizard.RunE(wizard, nil)
	}}
}
