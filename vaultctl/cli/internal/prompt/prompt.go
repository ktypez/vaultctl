// Package prompt builds huh forms themed to match the hanko ink palette.
package prompt

import (
	"errors"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/ktypez/vaultctl/cli/internal/theme"
)

// Theme is the vaultctl huh theme: monochrome with one hanko-red accent.
func Theme() *huh.Theme {
	t := huh.ThemeBase()
	red := theme.HankoRed
	dim := lipgloss.Color("243")
	faint := lipgloss.Color("238")
	bright := lipgloss.Color("253")

	t.Focused.Base = t.Focused.Base.BorderForeground(faint)
	t.Focused.Title = lipgloss.NewStyle().Bold(true).Foreground(bright)
	t.Focused.Description = lipgloss.NewStyle().Foreground(dim)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(red)
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(red)
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(red).SetString("◆ ")
	t.Focused.NextIndicator = lipgloss.NewStyle().Foreground(red)
	t.Focused.PrevIndicator = lipgloss.NewStyle().Foreground(red)
	t.Focused.Option = lipgloss.NewStyle().Foreground(bright)
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(red)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(faint)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(red).SetString("› ")

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Title = lipgloss.NewStyle().Foreground(dim)
	t.Blurred.Option = lipgloss.NewStyle().Foreground(faint)
	t.Blurred.TextInput.Prompt = lipgloss.NewStyle().Foreground(faint).SetString("› ")

	t.Help.ShortKey = lipgloss.NewStyle().Foreground(bright)
	t.Help.ShortDesc = lipgloss.NewStyle().Foreground(faint)
	t.Help.ShortSeparator = lipgloss.NewStyle().Foreground(faint).SetString(" · ")
	t.Help.FullKey = t.Help.ShortKey
	t.Help.FullDesc = t.Help.ShortDesc
	t.Help.FullSeparator = t.Help.ShortSeparator
	t.Help.Ellipsis = lipgloss.NewStyle().Foreground(faint)

	return t
}

func run(fields ...huh.Field) error {
	return huh.NewForm(huh.NewGroup(fields...)).WithTheme(Theme()).Run()
}

// Password prompts for the master password (masked).
func Password(title string) (string, error) {
	var pw string
	err := run(huh.NewInput().
		Title(title).
		EchoMode(huh.EchoModePassword).
		Value(&pw).
		Validate(func(s string) error {
			if s == "" {
				return errors.New("password cannot be empty")
			}
			return nil
		}))
	return pw, err
}

// NewPassword prompts twice for a new master password and confirms they match.
func NewPassword() (string, error) {
	var pw, again string
	err := run(
		huh.NewInput().
			Title("New master password").
			Description("There is no recovery. Losing it means losing the vault.").
			EchoMode(huh.EchoModePassword).
			Value(&pw).
			Validate(func(s string) error {
				if len(s) < 8 {
					return errors.New("use at least 8 characters")
				}
				return nil
			}),
		huh.NewInput().
			Title("Confirm new master password").
			EchoMode(huh.EchoModePassword).
			Value(&again).
			Validate(func(s string) error {
				if s != pw {
					return errors.New("passwords do not match")
				}
				return nil
			}),
	)
	return pw, err
}

// Secret prompts for a secret value (masked), used by `set` without a value.
func Secret(title string) (string, error) {
	var v string
	err := run(huh.NewInput().
		Title(title).
		EchoMode(huh.EchoModePassword).
		Value(&v).
		Validate(func(s string) error {
			if s == "" {
				return errors.New("value cannot be empty")
			}
			return nil
		}))
	return v, err
}

// Text prompts for a plain (visible) string.
func Text(title, description string) (string, error) {
	var v string
	in := huh.NewInput().Title(title).Value(&v)
	if description != "" {
		in = in.Description(description)
	}
	err := run(in)
	return v, err
}

// Confirm asks a yes/no question, defaulting to no.
func Confirm(title, description string) (bool, error) {
	var yes bool
	c := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&yes)
	if description != "" {
		c = c.Description(description)
	}
	err := run(c)
	return yes, err
}
