// Package spin wraps a synchronous function in a bubbletea spinner.
// Spinning is skipped when stdout/stderr are not TTYs, so pipelines and
// scripts never see escape codes.
package spin

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Run executes fn while rendering a spinner + label on stderr. The returned
// error is fn's error. When not a TTY, fn runs directly with no output.
func Run(label string, fn func() error) error {
	if !IsTTY(os.Stdout) || !IsTTY(os.Stderr) {
		return fn()
	}
	m := &model{label: label, op: fn, spinner: spinner.New()}
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	res, err := tea.NewProgram(m, tea.WithOutput(os.Stderr)).Run()
	if err != nil {
		return fn() // fall back rather than lose the work
	}
	if mm, ok := res.(*model); ok {
		return mm.err
	}
	return nil
}

// IsTTY reports whether f is a character device (a terminal).
func IsTTY(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

type model struct {
	label   string
	op      func() error
	spinner spinner.Model
	err     error
	done    bool
}

// doneMsg signals that op finished. bubbletea drops commands that return a
// nil Msg, so a nil error must be wrapped in a non-nil message type.
type doneMsg struct{ err error }

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg { return doneMsg{err: m.op()} })
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case doneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), m.label)
}
