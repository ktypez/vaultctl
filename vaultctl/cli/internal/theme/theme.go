// Package theme defines the vaultctl visual language: monochrome ink
// with a single hanko-red accent (the vermilion seal of a Japanese stamp).
package theme

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HankoRed is the vermilion seal ink. Everything else stays monochrome.
var HankoRed = lipgloss.Color("#E5484D")

var (
	inkDim    = lipgloss.Color("243")
	inkFaint  = lipgloss.Color("238")
	inkBright = lipgloss.Color("253")
	inkWhite  = lipgloss.Color("231")
)

var (
	// Name is the product masthead: bold white on hanko red, like a seal.
	Name = lipgloss.NewStyle().Bold(true).Foreground(inkWhite).
		Background(HankoRed).Padding(0, 1)

	// Title styles command titles.
	Title = lipgloss.NewStyle().Bold(true).Foreground(inkBright)

	// Dim is de-emphasized prose.
	Dim = lipgloss.NewStyle().Foreground(inkDim)

	// Faint is barely-there furniture.
	Faint = lipgloss.NewStyle().Foreground(inkFaint)

	// Accent is the hanko-red highlight.
	Accent = lipgloss.NewStyle().Foreground(HankoRed).Bold(true)

	// Error styles failure output.
	Error = lipgloss.NewStyle().Bold(true).Foreground(HankoRed)

	// Ok styles success confirmations (ink, not red — red is for alarms).
	Ok = lipgloss.NewStyle().Foreground(inkBright).Bold(true)

	// Label styles key names in key/value views.
	Label = lipgloss.NewStyle().Foreground(inkDim)

	// Key styles keycaps in help text.
	Key = lipgloss.NewStyle().Bold(true).Foreground(inkBright)

	// Rule renders the hanko-red divider.
	Rule = lipgloss.NewStyle().Foreground(HankoRed)

	// Panel is the boxed surface for banner output.
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inkFaint).
		Padding(1, 2)

	// Entry marks list items.
	Entry = lipgloss.NewStyle().Foreground(inkBright)

	// BulletDot is the list bullet.
	BulletDot = lipgloss.NewStyle().Foreground(HankoRed)
)

// Banner returns the one-line masthead: seal + name + tagline.
func Banner() string {
	return lipgloss.JoinHorizontal(lipgloss.Center,
		Name.Render("vaultctl"),
		"  ",
		Dim.Render("zero-knowledge credentials vault"),
	)
}

// RuleLine renders a hanko-red horizontal rule of the given width.
func RuleLine(w int) string {
	if w < 8 {
		w = 8
	}
	return Rule.Render(strings.Repeat("─", w))
}

// Bullets renders a hanko-marked list of names.
func Bullets(names []string) string {
	if len(names) == 0 {
		return Dim.Render("no entries yet — add one with ") +
			Key.Render("vaultctl set <name> <value>")
	}
	var b strings.Builder
	for i, n := range names {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(BulletDot.Render("◆"))
		b.WriteString("  ")
		b.WriteString(Entry.Render(n))
	}
	return b.String()
}

// Width returns the terminal width from $COLUMNS, or 80 as a fallback.
func Width() int {
	if v := os.Getenv("COLUMNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 80
}
