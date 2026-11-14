package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ktypez/vaultctl/cli/internal/client"
	"github.com/ktypez/vaultctl/cli/internal/prompt"
	"github.com/ktypez/vaultctl/cli/internal/spin"
	"github.com/ktypez/vaultctl/cli/internal/theme"
)

// pretty reports whether human-facing, styled output is appropriate:
// stdout must be a terminal and --json must not be set.
func pretty() bool {
	return !jsonOut && spin.IsTTY(os.Stdout)
}

// writeJSON writes v as compact JSON (one line) — the scripting contract.
func writeJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(b))
	return err
}

// ensureSession opens a session, prompting interactively for the password
// when no cached credential is available.
func ensureSession() error {
	if !Vault.IsLocked() {
		return nil
	}
	if _, err := Vault.Password(); err != nil {
		if !spin.IsTTY(os.Stdin) {
			return err
		}
		pw, perr := prompt.Password("Master password")
		if perr != nil {
			return perr
		}
		return Vault.Unlock(pw)
	}
	return spin.Run("unlocking vault", Vault.EnsureUnlocked)
}

// ---------------------------------------------------------------- status

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show initialized / unlocked state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := Vault.Status()
		if err != nil {
			return err
		}
		if !pretty() {
			// Legacy contract: one JSON object.
			return writeJSON(map[string]any{
				"initialized": s.Initialized,
				"unlocked":    s.Unlocked,
				"backend":     Vault.Base,
			})
		}
		fmt.Println(theme.Banner())
		fmt.Println()
		fmt.Println(theme.Panel.Render(strings.Join([]string{
			kv("backend", Vault.Base),
			kv("initialized", boolMark(s.Initialized)),
			kv("unlocked", boolMark(s.Unlocked)),
			kv("session", sessionHint(s.Unlocked)),
		}, "\n")))
		return nil
	},
}

func sessionHint(unlocked bool) string {
	if unlocked {
		return theme.Dim.Render("idle timeout 30m")
	}
	return theme.Dim.Render("run ") + theme.Key.Render("vaultctl unlock")
}

func kv(k, v string) string {
	return theme.Label.Width(12).Render(k) + theme.Entry.Render(v)
}

func boolMark(b bool) string {
	if b {
		return theme.Ok.Render("yes")
	}
	return theme.Dim.Render("no")
}

// ------------------------------------------------------------------ list

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List entry names",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureSession(); err != nil {
			return err
		}
		var names []string
		err := spin.Run("reading vault", func() error {
			var e error
			names, e = Vault.List()
			return e
		})
		if err != nil {
			return err
		}
		if !pretty() {
			// Legacy contract: one name per line.
			for _, n := range names {
				fmt.Println(n)
			}
			return nil
		}
		fmt.Println(theme.Title.Render("entries") + theme.Faint.Render(fmt.Sprintf("  %d", len(names))))
		fmt.Println(theme.RuleLine(theme.Width() - 2))
		fmt.Println(theme.Bullets(names))
		return nil
	},
}

// ------------------------------------------------------------------- get

var getCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Print a secret to stdout",
	Long: "Print a secret to stdout.\n\n" +
		theme.Dim.Render("The value is written raw so it can be piped. Never use it in a\n") +
		theme.Dim.Render("recorded shell — prefer `vaultctl exec`."),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureSession(); err != nil {
			return err
		}
		name := args[0]
		var value string
		if err := spin.Run("reading "+name, func() error {
			var e error
			value, e = Vault.Get(name)
			return e
		}); err != nil {
			return err
		}
		// Raw value on stdout, always — the pipeline contract.
		os.Stdout.WriteString(value)
		if !strings.HasSuffix(value, "\n") {
			os.Stdout.WriteString("\n")
		}
		return nil
	},
}

// ------------------------------------------------------------------- set

var setCmd = &cobra.Command{
	Use:   "set <name> [value]",
	Short: "Create or update an entry",
	Long: "Create or update an entry.\n\n" +
		theme.Dim.Render("Omit the value to be prompted with a masked input (keeps the\n") +
		theme.Dim.Render("secret out of your shell history)."),
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := client.ValidateName(name); err != nil {
			return err
		}
		if err := ensureSession(); err != nil {
			return err
		}

		existed := Vault.Has(name)

		var value string
		if len(args) == 2 {
			value = args[1]
		} else {
			if !spin.IsTTY(os.Stdin) {
				return fmt.Errorf("no value given and stdin is not a terminal")
			}
			v, err := prompt.Secret("Value for " + name)
			if err != nil {
				return err
			}
			value = v
		}
		if err := spin.Run("sealing "+name, func() error {
			return Vault.Set(name, value)
		}); err != nil {
			return err
		}
		verb := "added"
		if existed {
			verb = "updated"
		}
		if !pretty() {
			return writeJSON(map[string]any{"ok": true, "name": name, "created": !existed})
		}
		fmt.Println(theme.Ok.Render("✓ ") + verb + " " + theme.Accent.Render(name))
		return nil
	},
}

// ---------------------------------------------------------------- delete

var deleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Aliases: []string{"rm", "remove"},
	Short:   "Remove an entry",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := ensureSession(); err != nil {
			return err
		}
		if !force && pretty() && spin.IsTTY(os.Stdin) {
			ok, err := prompt.Confirm("Delete "+name+"?",
				"This cannot be undone.")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println(theme.Dim.Render("cancelled"))
				return nil
			}
		}
		if err := spin.Run("removing "+name, func() error {
			return Vault.Delete(name)
		}); err != nil {
			return err
		}
		if !pretty() {
			return writeJSON(map[string]any{"ok": true, "name": name})
		}
		fmt.Println(theme.Ok.Render("✓ ") + "removed " + theme.Accent.Render(name))
		return nil
	},
}

// ---------------------------------------------------------------- unlock

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock the vault for 30 minutes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var pw string
		if p, err := Vault.Password(); err == nil {
			pw = p
		} else {
			if !spin.IsTTY(os.Stdin) {
				return err
			}
			p, perr := prompt.Password("Master password")
			if perr != nil {
				return perr
			}
			pw = p
		}
		if err := spin.Run("unlocking vault", func() error {
			return Vault.Unlock(pw)
		}); err != nil {
			return err
		}
		if !pretty() {
			fmt.Println("unlocked")
			return nil
		}
		fmt.Println(theme.Ok.Render("✓ ") + "unlocked" +
			theme.Dim.Render("  · session idle-timeout 30m"))
		return nil
	},
}

// ------------------------------------------------------------------ lock

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Wipe the local session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := Vault.Lock(); err != nil {
			// Local state is gone either way; report but do not fail hard.
			fmt.Fprintln(os.Stderr, theme.Dim.Render("note: "+err.Error()))
		}
		if !pretty() {
			return writeJSON(map[string]any{"ok": true})
		}
		fmt.Println(theme.Ok.Render("✓ ") + "locked" +
			theme.Dim.Render("  · session files wiped"))
		return nil
	},
}

// ------------------------------------------------------- change-password

var changePasswordCmd = &cobra.Command{
	Use:     "change-password [new]",
	Aliases: []string{"passwd"},
	Short:   "Rotate the master password",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureSession(); err != nil {
			return err
		}
		current, err := Vault.Password()
		if err != nil {
			return err
		}
		var next string
		if len(args) == 1 {
			next = args[0]
		} else {
			if !spin.IsTTY(os.Stdin) {
				return fmt.Errorf("usage: vaultctl change-password <new-password>")
			}
			next, err = prompt.NewPassword()
			if err != nil {
				return err
			}
		}
		if err := spin.Run("rotating master password", func() error {
			return Vault.ChangePassword(current, next)
		}); err != nil {
			return err
		}
		if !pretty() {
			fmt.Println("password changed and " + Vault.PasswordFile() + " updated")
			return nil
		}
		fmt.Println(theme.Ok.Render("✓ ") + "password changed and cached to " +
			theme.Dim.Render(Vault.PasswordFile()))
		return nil
	},
}
