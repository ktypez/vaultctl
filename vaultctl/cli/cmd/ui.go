package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ktypez/vaultctl/cli/internal/spin"
	"github.com/ktypez/vaultctl/cli/internal/theme"
	"github.com/ktypez/vaultctl/cli/internal/ui"
)

var uiCheck bool

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Open the interactive terminal dashboard",
	Long:  "Open the interactive terminal dashboard: search, copy, add, edit,\nrename and delete entries, plus an audit view.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureSession(); err != nil {
			return err
		}
		if uiCheck {
			names, err := Vault.List()
			if err != nil {
				return err
			}
			fmt.Printf("%s %d entries at %s\n",
				theme.Ok.Render("✓"), len(names), theme.Dim.Render(Vault.Base))
			return nil
		}
		if !spin.IsTTY(os.Stdout) {
			return fmt.Errorf("ui needs a terminal (stdout is not a TTY)")
		}
		return ui.Run(Vault)
	},
}

func init() {
	uiCmd.Flags().BoolVar(&uiCheck, "check", false, "non-interactive backend self-test")
}
