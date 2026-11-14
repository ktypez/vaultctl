package spin

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func spinnerModel() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return s
}

// TestDoneMessageIsWrapped guards the bug where a nil Msg (successful op)
// was dropped by bubbletea, leaving the program running forever.
func TestDoneMessageIsWrapped(t *testing.T) {
	m := &model{label: "x", op: func() error { return nil }, spinner: spinnerModel()}

	cmd := m.Init()
	msgs := runCmd(t, cmd, 5)

	sawDone := false
	for _, msg := range msgs {
		if _, ok := msg.(doneMsg); ok {
			sawDone = true
		}
		if msg == nil {
			t.Fatal("Init emitted a nil Msg; bubbletea would drop it")
		}
	}
	if !sawDone {
		t.Fatalf("Init commands %v did not include a doneMsg", msgs)
	}
}

func TestDoneMessageCarriesError(t *testing.T) {
	want := errors.New("boom")
	m := &model{label: "x", op: func() error { return want }, spinner: spinnerModel()}
	for _, msg := range runCmd(t, m.Init(), 5) {
		if dm, ok := msg.(doneMsg); ok {
			if !errors.Is(dm.err, want) {
				t.Fatalf("doneMsg.err = %v, want %v", dm.err, want)
			}
		}
	}
}

func TestUpdateQuitsOnDone(t *testing.T) {
	m := &model{label: "x", op: func() error { return nil }, spinner: spinnerModel()}
	got, cmd := m.Update(doneMsg{})
	if !m.done {
		t.Fatal("done flag not set")
	}
	if cmd == nil {
		t.Fatal("expected a quit command")
	}
	if got == nil {
		t.Fatal("Update returned nil model")
	}
}

// runCmd collects the messages produced by a command, expanding batches.
func runCmd(t *testing.T, cmd tea.Cmd, depth int) []tea.Msg {
	t.Helper()
	if cmd == nil || depth <= 0 {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return []tea.Msg{nil}
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, runCmd(t, c, depth-1)...)
		}
		return out
	}
	return []tea.Msg{msg}
}
