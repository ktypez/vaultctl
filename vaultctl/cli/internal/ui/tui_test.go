package ui

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ktypez/vaultctl/cli/internal/client"
)

// fake is an in-memory Vault for driving the TUI without a backend.
type fake struct {
	data map[string]string
}

func newFake(seed map[string]string) *fake {
	d := map[string]string{}
	for k, v := range seed {
		d[k] = v
	}
	return &fake{data: d}
}

func (f *fake) URL() string { return "http://fake" }

func (f *fake) List() ([]string, error) {
	names := make([]string, 0, len(f.data))
	for k := range f.data {
		names = append(names, k)
	}
	sort.Strings(names)
	return names, nil
}

func (f *fake) Get(name string) (string, error) {
	v, ok := f.data[name]
	if !ok {
		return "", fmt.Errorf("not found: %s", name)
	}
	return v, nil
}

func (f *fake) Set(name, value string) error { f.data[name] = value; return nil }

func (f *fake) Delete(name string) error { delete(f.data, name); return nil }

func (f *fake) Audit(n int) ([]client.AuditRecord, error) {
	return []client.AuditRecord{{TS: "2026-11-14T07:00:00Z", Action: "get", Name: "A", Decision: "allow"}}, nil
}

// pump feeds a message through Update and resolves any returned command.
func pump(t *testing.T, m *Model, msg tea.Msg) *Model {
	t.Helper()
	model, _ := m.Update(msg)
	mm, ok := model.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, want *Model", model)
	}
	return mm
}

// runCmd executes a command but abandons it if it is a long-lived timer
// (cursor blink, spinner tick) rather than an immediate data message.
func runCmd(cmd tea.Cmd) (tea.Msg, bool) {
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg, true
	case <-time.After(80 * time.Millisecond):
		return nil, false
	}
}

// settle runs pending commands until the model goes idle. Spinner ticks
// re-schedule themselves forever, so they are treated as terminal.
func settle(t *testing.T, m *Model, cmd tea.Cmd) *Model {
	t.Helper()
	for i := 0; cmd != nil && i < 200; i++ {
		msg, ok := runCmd(cmd)
		if !ok || msg == nil {
			return m
		}
		if _, isBatch := msg.(tea.BatchMsg); isBatch {
			for _, c := range batchCmds(msg) {
				if c != nil {
					m = settle(t, m, c)
				}
			}
			return m
		}
		model, next := m.Update(msg)
		mm, ok := model.(*Model)
		if !ok {
			t.Fatalf("Update returned %T, want *Model", model)
		}
		m = mm
		if _, isTick := msg.(spinner.TickMsg); isTick {
			return m // ticking never ends; stop here
		}
		cmd = next
	}
	return m
}

func batchCmds(msg tea.Msg) []tea.Cmd {
	b, ok := msg.(tea.BatchMsg)
	if !ok {
		return nil
	}
	return []tea.Cmd(b)
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// typeKeys feeds s one rune at a time through the model so state is kept.
func typeKeys(t *testing.T, m *Model, s string) *Model {
	t.Helper()
	for _, r := range s {
		model, cmd := m.Update(key(string(r)))
		m = settle(t, model.(*Model), cmd)
	}
	return m
}

func newTestModel(t *testing.T) *Model {
	t.Helper()
	m := newModel(newFake(map[string]string{
		"ALPHA": "value-alpha",
		"BETA":  "value-beta",
		"GAMMA": "value-gamma",
	}))
	m.width, m.height = 80, 24
	return settle(t, m, m.Init())
}

func TestListRendersNames(t *testing.T) {
	m := newTestModel(t)
	view := m.View()
	for _, name := range []string{"ALPHA", "BETA", "GAMMA"} {
		if !strings.Contains(view, name) {
			t.Errorf("view missing %q:\n%s", name, view)
		}
	}
}

func TestCursorMoves(t *testing.T) {
	m := newTestModel(t)
	if got, _ := m.selected(); got != "ALPHA" {
		t.Fatalf("initial selection = %q, want ALPHA", got)
	}
	m = pump(t, m, key("down"))
	if got, _ := m.selected(); got != "BETA" {
		t.Fatalf("after down = %q, want BETA", got)
	}
	m = pump(t, m, key("j"))
	if got, _ := m.selected(); got != "GAMMA" {
		t.Fatalf("after j = %q, want GAMMA", got)
	}
	// clamp at the bottom
	m = pump(t, m, key("down"))
	if got, _ := m.selected(); got != "GAMMA" {
		t.Fatalf("cursor should clamp at GAMMA, got %q", got)
	}
}

func TestFilter(t *testing.T) {
	m := newTestModel(t)
	m = pump(t, m, key("/"))
	for _, r := range "BET" {
		m = pump(t, m, key(string(r)))
	}
	vis := m.visible()
	if len(vis) != 1 || vis[0] != "BETA" {
		t.Fatalf("filtered = %v, want [BETA]", vis)
	}
	// esc clears the filter
	m = pump(t, m, key("esc"))
	if len(m.visible()) != 3 {
		t.Fatalf("after esc = %v, want all 3", m.visible())
	}
}

func TestAuditMode(t *testing.T) {
	m := newTestModel(t)
	model, cmd := m.Update(key("n"))
	m = settle(t, model.(*Model), cmd)
	if m.mode != modeAudit {
		t.Fatalf("mode = %v, want audit", m.mode)
	}
	if len(m.audit) == 0 {
		t.Fatal("audit records not loaded")
	}
	// esc returns to the list
	m = pump(t, m, key("esc"))
	if m.mode != modeList {
		t.Fatalf("after esc, mode = %v, want list", m.mode)
	}
}

func TestAddEntry(t *testing.T) {
	f := newFake(nil)
	m := newModel(f)
	m.width, m.height = 80, 24
	m = settle(t, m, m.Init())

	// press a -> opens the add form
	model, cmd := m.Update(key("a"))
	m = settle(t, model.(*Model), cmd)
	if m.mod == nil || m.mod.kind != modalAdd {
		t.Fatal("expected add modal")
	}
	// drive the form: name, then value, then submit
	m = typeKeys(t, m, "NEW_KEY")
	model, cmd = m.Update(key("enter"))
	m = settle(t, model.(*Model), cmd)
	m = typeKeys(t, m, "s3cret")
	model, cmd = m.Update(key("enter"))
	m = settle(t, model.(*Model), cmd)

	if got := f.data["NEW_KEY"]; got != "s3cret" {
		t.Fatalf("stored value = %q, want s3cret", got)
	}
}

func TestDeleteEntry(t *testing.T) {
	f := newFake(map[string]string{"DOOMED": "x"})
	m := newModel(f)
	m.width, m.height = 80, 24
	m = settle(t, m, m.Init())

	model, cmd := m.Update(key("d"))
	m = settle(t, model.(*Model), cmd)
	if m.mod == nil || m.mod.kind != modalDelete {
		t.Fatal("expected delete modal")
	}
	// "y" selects Yes; the confirm submits on the same key.
	model, cmd = m.Update(key("y"))
	m = settle(t, model.(*Model), cmd)

	if _, ok := f.data["DOOMED"]; ok {
		t.Fatal("entry was not deleted")
	}
}

func TestRenameEntry(t *testing.T) {
	f := newFake(map[string]string{"OLD": "keepme"})
	m := newModel(f)
	m.width, m.height = 80, 24
	m = settle(t, m, m.Init())

	model, cmd := m.Update(key("r"))
	m = settle(t, model.(*Model), cmd)
	if m.mod == nil || m.mod.kind != modalRename {
		t.Fatal("expected rename modal")
	}
	// clear and type the new name
	for i := 0; i < 3; i++ {
		model, cmd := m.Update(key("backspace"))
		m = settle(t, model.(*Model), cmd)
	}
	m = typeKeys(t, m, "NEW")
	model, cmd = m.Update(key("enter"))
	m = settle(t, model.(*Model), cmd)

	if _, ok := f.data["OLD"]; ok {
		t.Fatal("old name still present")
	}
	if got := f.data["NEW"]; got != "keepme" {
		t.Fatalf("renamed value = %q, want keepme", got)
	}
}
