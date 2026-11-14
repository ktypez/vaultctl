// Package cmd defines the vaultctl command tree.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ktypez/vaultctl/cli/internal/client"
	"github.com/ktypez/vaultctl/cli/internal/spin"
	"github.com/ktypez/vaultctl/cli/internal/theme"
)

// version is stamped at build time with -ldflags "-X ...cmd.version=...".
var version = "dev"

// Vault is the shared backend client, built once in Execute.
var Vault *client.Client

// jsonOut forces machine-readable output where supported.
var jsonOut bool

var rootCmd = &cobra.Command{
	Use:   "vaultctl",
	Short: "Zero-knowledge credentials vault",
	Long: theme.Banner() + "\n\n" +
		theme.Dim.Render("Store API keys, tokens and passwords encrypted at rest.") + "\n" +
		theme.Dim.Render("Nothing leaves your machine. Master password never stored."),
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		Vault = client.New()
		return nil
	},
}

// Execute runs the root command and exits non-zero on failure.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		die(err)
	}
}

// die prints an error and exits. Styling is applied only on a TTY so
// redirected stderr stays plain.
func die(err error) {
	msg := "✗ " + err.Error()
	if spin.IsTTY(os.Stderr) {
		msg = theme.Error.Render("✗ ") + err.Error()
	}
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")

	rootCmd.AddCommand(
		statusCmd,
		listCmd,
		getCmd,
		setCmd,
		deleteCmd,
		unlockCmd,
		lockCmd,
		changePasswordCmd,
		execCmd,
		auditCmd,
		copyCmd,
		uiCmd,
		versionCmd,
	)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}
