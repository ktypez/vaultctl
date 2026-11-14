package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"

	"github.com/ktypez/vaultctl/cli/internal/client"
	"github.com/ktypez/vaultctl/cli/internal/spin"
	"github.com/ktypez/vaultctl/cli/internal/theme"
)

var force bool

// ------------------------------------------------------------------- copy

var copyCmd = &cobra.Command{
	Use:     "copy <name>",
	Aliases: []string{"cp"},
	Short:   "Copy a secret to the clipboard (OSC52)",
	Long: "Copy a secret to the system clipboard via OSC52 (works over SSH).\n\n" +
		theme.Dim.Render("The secret is redacted from terminal echoes but the clipboard\n") +
		theme.Dim.Render("retains it. Clear the clipboard when you are done."),
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
		if !spin.IsTTY(os.Stdout) {
			return fmt.Errorf("copy needs a terminal (stdout is not a TTY)")
		}
		os.Stdout.WriteString(ansi.SetSystemClipboard(value))
		fmt.Println(theme.Ok.Render("✓ ") + "copied " + theme.Accent.Render(name) +
			theme.Dim.Render(" to clipboard"))
		return nil
	},
}

// ------------------------------------------------------------------- exec

var execEnv string

var execCmd = &cobra.Command{
	Use:   "exec <name> [--env VAR] -- <command> [args...]",
	Short: "Run a command with the secret injected (child output masked)",
	Long: "Inject a secret into a child process environment and stream its\n" +
		"output with the secret redacted. The value is never printed.",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dash := cmd.ArgsLenAtDash()
		if dash < 0 {
			return fmt.Errorf("usage: vaultctl exec <name> [--env VAR] -- <command> [args...]")
		}
		if dash < 1 {
			return fmt.Errorf("usage: vaultctl exec <name> [--env VAR] -- <command> [args...]")
		}
		name := args[0]
		child := args[dash:]
		if len(child) == 0 {
			return fmt.Errorf("missing command after --")
		}
		if err := ensureSession(); err != nil {
			return err
		}

		var value string
		if err := spin.Run("reading "+name, func() error {
			var e error
			value, e = Vault.Get(name)
			return e
		}); err != nil {
			return err
		}

		varname := execEnv
		if varname == "" {
			varname = name
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		c := exec.CommandContext(ctx, child[0], child[1:]...)
		c.Env = append(os.Environ(), varname+"="+value)
		c.Stdin = os.Stdin

		masker := client.NewMasker(value)
		// Always mask: "the value is never printed" is a documented
		// guarantee, and a piped redirect (`> log.txt`) must not leak it.
		outWriter := &maskWriter{w: os.Stdout, m: masker}
		errWriter := &maskWriter{w: os.Stderr, m: masker}
		c.Stdout, c.Stderr = outWriter, errWriter

		err := c.Run()
		outWriter.Flush()
		errWriter.Flush()
		if err == nil {
			return nil
		}
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		return err
	},
}

// maskWriter wraps an io.Writer and redacts the secret, line-buffered so
// fragments split across writes are still caught.
type maskWriter struct {
	w   *os.File
	m   *client.Masker
	buf strings.Builder
}

func (mw *maskWriter) Write(p []byte) (int, error) {
	mw.buf.Write(p)
	for {
		s := mw.buf.String()
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			break
		}
		line := s[:i+1]
		mw.buf.Reset()
		mw.buf.WriteString(s[i+1:])
		if _, err := mw.w.Write(mw.m.Mask([]byte(line))); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (mw *maskWriter) Flush() {
	if mw.buf.Len() > 0 {
		mw.w.Write(mw.m.Mask([]byte(mw.buf.String())))
		mw.buf.Reset()
	}
}

// ------------------------------------------------------------------ audit

var auditCmd = &cobra.Command{
	Use:   "audit [n]",
	Short: "Show the last n audit records",
	Long: "Show audit records (JSONL on the server). Secret values are never\n" +
		"recorded.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureSession(); err != nil {
			return err
		}
		n := 100
		if len(args) == 1 {
			if _, err := fmt.Sscanf(args[0], "%d", &n); err != nil || n <= 0 {
				return fmt.Errorf("n must be a positive integer")
			}
		}
		var recs []client.AuditRecord
		if err := spin.Run("reading audit log", func() error {
			var e error
			recs, e = Vault.Audit(n)
			return e
		}); err != nil {
			return err
		}
		if !pretty() {
			// Legacy contract: one JSON record per line.
			enc := json.NewEncoder(os.Stdout)
			for _, r := range recs {
				if err := enc.Encode(r); err != nil {
					return err
				}
			}
			return nil
		}
		fmt.Println(theme.Title.Render("audit") +
			theme.Faint.Render(fmt.Sprintf("  last %d", len(recs))))
		fmt.Println(theme.RuleLine(theme.Width() - 2))
		if len(recs) == 0 {
			fmt.Println(theme.Dim.Render("no audit records"))
			return nil
		}
		for _, r := range recs {
			fmt.Println(auditLine(r))
		}
		return nil
	},
}

func auditLine(r client.AuditRecord) string {
	ts := r.TS
	if len(ts) >= 19 && strings.Contains(ts, "T") {
		ts = strings.Replace(ts[:19], "T", " ", 1)
	}
	mark := theme.Dim.Render("·")
	if r.Decision == "deny" || r.Decision == "denied" {
		mark = theme.Error.Render("✗")
	} else if r.Decision == "allow" || r.Decision == "allowed" {
		mark = theme.Ok.Render("✓")
	}
	parts := []string{theme.Faint.Render(ts), mark}
	if r.Action != "" {
		parts = append(parts, theme.Dim.Render(r.Action))
	}
	if r.Name != "" {
		parts = append(parts, theme.Entry.Render(r.Name))
	}
	if r.Reason != "" {
		parts = append(parts, theme.Faint.Render(r.Reason))
	}
	if r.Remote != "" {
		parts = append(parts, theme.Faint.Render(r.Remote))
	}
	return strings.Join(parts, "  ")
}
