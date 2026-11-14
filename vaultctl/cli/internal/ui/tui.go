// Package ui implements the vaultctl terminal dashboard: browse, search,
// copy, and manage entries without leaving the terminal.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ktypez/vaultctl/cli/internal/client"
	"github.com/ktypez/vaultctl/cli/internal/prompt"
	"github.com/ktypez/vaultctl/cli/internal/theme"
)

// Vault is the subset of the backend the dashboard needs. Keeping it an
// interface lets tests drive the UI with a fake.
type Vault interface {
	List() ([]string, error)
	Get(name string) (string, error)
	Set(name, value string) error
	Delete(name string) error
	Audit(n int) ([]client.AuditRecord, error)
	URL() string
}

// Run starts the dashboard against a backend.
func Run(c Vault) error {
	m := newModel(c)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

type mode int

const (
	modeList mode = iota
	modeAudit
)

// modalKind identifies the open overlay form.
type modalKind int

const (
	modalNone modalKind = iota
	modalAdd
	modalEdit
	modalRename
	modalDelete
)

type modal struct {
	kind   modalKind
	title  string
	form   *huh.Form
	orig   string // original name (rename/delete/edit)
	name   string
	value  string
	yes    bool
	action func() (string, error)
}

// Model is the dashboard state.
type Model struct {
	c             Vault
	width, height int

	mode  mode
	names []string
	list  []string
	cur   int

	filter    textinput.Model
	filtering bool

	audit  []client.AuditRecord
	scroll int

	status string
	err    error
	busy   bool
	spin   spinner.Model
	help   bool
	mod    *modal
}

// ---- messages

type loadedMsg struct {
	names []string
	err   error
}

type auditMsg struct {
	recs []client.AuditRecord
	err  error
}

type valueMsg struct {
	name  string
	value string
	err   error
}

type doneMsg struct {
	note string
	err  error
}

func newModel(c Vault) *Model {
	f := textinput.New()
	f.Placeholder = "filter…"
	f.Prompt = ""
	f.CharLimit = 64

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.HankoRed)

	return &Model{
		c:      c,
		filter: f,
		spin:   s,
		status: "ready",
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.reload())
}

// ---- data loading

func (m *Model) reload() tea.Cmd {
	m.busy = true
	c := m.c
	return func() tea.Msg {
		names, err := c.List()
		return loadedMsg{names: names, err: err}
	}
}

func (m *Model) loadAudit() tea.Cmd {
	m.busy = true
	c := m.c
	return func() tea.Msg {
		recs, err := c.Audit(200)
		return auditMsg{recs: recs, err: err}
	}
}

func (m *Model) copy(name string) tea.Cmd {
	m.busy = true
	c := m.c
	return func() tea.Msg {
		v, err := c.Get(name)
		return valueMsg{name: name, value: v, err: err}
	}
}

// ---- list helpers

func (m *Model) visible() []string {
	f := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if f == "" {
		return m.names
	}
	out := make([]string, 0, len(m.names))
	for _, n := range m.names {
		if strings.Contains(strings.ToLower(n), f) {
			out = append(out, n)
		}
	}
	return out
}

func (m *Model) clampCursor() {
	max := len(m.visible()) - 1
	if m.cur > max {
		m.cur = max
	}
	if m.cur < 0 {
		m.cur = 0
	}
}

func (m *Model) move(d int) {
	m.cur += d
	m.clampCursor()
}

func (m *Model) selected() (string, bool) {
	v := m.visible()
	if m.cur < 0 || m.cur >= len(v) {
		return "", false
	}
	return v[m.cur], true
}

// ---- update

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = wsm.Width, wsm.Height
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case loadedMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.names, m.err = msg.names, nil
			m.clampCursor()
		}
		return m, nil
	case auditMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.audit, m.scroll = msg.recs, 0
		m.mode = modeAudit
		m.err = nil
		return m, nil
	case valueMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status = "copied " + theme.Accent.Render(msg.name) +
			theme.Dim.Render(" — clipboard holds the secret")
		return m, tea.Println(ansi.SetSystemClipboard(msg.value))
	case doneMsg:
		m.busy = false
		if msg.err != nil {
			m.err, m.status = msg.err, ""
			return m, nil
		}
		m.err = nil
		m.status = theme.Ok.Render("✓ ") + msg.note
		return m, m.reload()
	}

	// modal takes precedence
	if m.mod != nil {
		return m.updateModal(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	return m.updateList(key)
}

func (m *Model) updateList(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		switch key.String() {
		case "esc":
			m.filtering = false
			m.filter.SetValue("")
			m.filter.Blur()
			m.clampCursor()
			return m, nil
		case "enter":
			m.filtering = false
			m.filter.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(key)
		m.clampCursor()
		return m, cmd
	}

	if m.mode == modeAudit {
		return m.updateAudit(key)
	}

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "home", "g":
		m.cur = 0
	case "end", "G":
		m.cur = len(m.visible()) - 1
		m.clampCursor()
	case "enter", "y":
		if name, ok := m.selected(); ok {
			return m, m.copy(name)
		}
	case "a":
		return m, m.openForm(modalAdd)
	case "e":
		if name, ok := m.selected(); ok {
			return m, m.openForm(modalEdit, name)
		}
	case "r":
		if name, ok := m.selected(); ok {
			return m, m.openForm(modalRename, name)
		}
	case "d", "x":
		if name, ok := m.selected(); ok {
			return m, m.openForm(modalDelete, name)
		}
	case "n":
		return m, m.loadAudit()
	case "R":
		return m, m.reload()
	case "?":
		m.help = !m.help
	case "/":
		m.filtering = true
		m.filter.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m *Model) updateAudit(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "n":
		m.mode = modeList
	case "up", "k":
		if m.scroll > 0 {
			m.scroll--
		}
	case "down", "j":
		if m.scroll < len(m.audit)-1 {
			m.scroll++
		}
	case "R":
		return m, m.loadAudit()
	}
	return m, nil
}

func (m *Model) updateModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			m.mod = nil
			return m, nil
		}
	}
	form, cmd := m.mod.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.mod.form = f
	}
	switch m.mod.form.State {
	case huh.StateCompleted:
		return m, m.submit()
	case huh.StateAborted:
		m.mod = nil
	}
	return m, cmd
}

// ---- modal construction

func (m *Model) openForm(kind modalKind, name ...string) tea.Cmd {
	w := m.width - 10
	if w < 40 {
		w = 40
	}
	cur := ""
	if len(name) > 0 {
		cur = name[0]
	}

	var md *modal
	switch kind {
	case modalAdd:
		md = &modal{kind: kind, title: "Add entry", orig: ""}
		md.form = huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Name").Value(&md.name).
				Placeholder("e.g. OPENAI_API_KEY").
				Validate(func(s string) error {
					if !client.ValidName(s) {
						return fmt.Errorf("letters, numbers, . - _ only")
					}
					return nil
				}),
			huh.NewInput().Title("Value").Value(&md.value).
				EchoMode(huh.EchoModePassword),
		)).WithTheme(prompt.Theme()).WithWidth(w)
	case modalEdit:
		md = &modal{kind: kind, title: "Edit " + cur, orig: cur, name: cur}
		md.form = huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Value").Value(&md.value).
				EchoMode(huh.EchoModePassword),
		)).WithTheme(prompt.Theme()).WithWidth(w)
	case modalRename:
		md = &modal{kind: kind, title: "Rename " + cur, orig: cur, name: cur}
		md.form = huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("New name").Value(&md.name).
				Validate(func(s string) error {
					if !client.ValidName(s) {
						return fmt.Errorf("letters, numbers, . - _ only")
					}
					return nil
				}),
		)).WithTheme(prompt.Theme()).WithWidth(w)
	case modalDelete:
		md = &modal{kind: kind, title: "Delete " + cur, orig: cur, name: cur}
		md.form = huh.NewForm(huh.NewGroup(
			huh.NewConfirm().Title("Delete " + cur + "?").
				Description("This cannot be undone.").
				Affirmative("Yes").Negative("No").Value(&md.yes),
		)).WithTheme(prompt.Theme()).WithWidth(w)
	}
	m.mod = md
	return md.form.Init()
}

func (m *Model) submit() tea.Cmd {
	md := m.mod
	if md == nil {
		return nil
	}
	m.mod = nil
	m.busy = true
	c := m.c

	switch md.kind {
	case modalAdd:
		if md.name == "" || md.value == "" {
			m.busy = false
			m.status = theme.Dim.Render("cancelled")
			return nil
		}
		name, value := md.name, md.value
		return func() tea.Msg {
			return doneMsg{note: "saved " + name, err: c.Set(name, value)}
		}
	case modalEdit:
		if md.value == "" {
			m.busy = false
			m.status = theme.Dim.Render("cancelled")
			return nil
		}
		name, value := md.orig, md.value
		return func() tea.Msg {
			return doneMsg{note: "updated " + name, err: c.Set(name, value)}
		}
	case modalRename:
		from, to := md.orig, md.name
		if to == "" || to == from {
			m.busy = false
			m.status = theme.Dim.Render("cancelled")
			return nil
		}
		return func() tea.Msg {
			v, err := c.Get(from)
			if err != nil {
				return doneMsg{err: err}
			}
			if err := c.Set(to, v); err != nil {
				return doneMsg{err: err}
			}
			if err := c.Delete(from); err != nil {
				return doneMsg{err: err}
			}
			return doneMsg{note: "renamed " + from + " → " + to}
		}
	case modalDelete:
		if !md.yes {
			m.busy = false
			m.status = theme.Dim.Render("cancelled")
			return nil
		}
		name := md.orig
		return func() tea.Msg {
			return doneMsg{note: "removed " + name, err: c.Delete(name)}
		}
	}
	return nil
}

// ---- view

func (m *Model) View() string {
	if m.mod != nil {
		return m.viewModal()
	}
	if m.mode == modeAudit {
		return m.viewAudit()
	}
	return m.viewList()
}

func (m *Model) viewModal() string {
	panel := theme.Panel.Render(
		theme.Title.Render(m.mod.title) + "\n\n" + m.mod.form.View())
	return lipgloss.Place(max(m.width, 40), max(m.height, 12),
		lipgloss.Center, lipgloss.Center, panel)
}

func (m *Model) viewList() string {
	var b strings.Builder

	left := theme.Name.Render("vaultctl")
	count := theme.Faint.Render(fmt.Sprintf("  %d entries", len(m.names)))
	right := theme.Dim.Render(m.c.URL())
	header := padBetween(left+count, right, m.width)
	b.WriteString(header + "\n")

	if m.filtering {
		b.WriteString(theme.Accent.Render("filter ") + m.filter.View() + "\n")
	} else if m.filter.Value() != "" {
		b.WriteString(theme.Dim.Render("filter: ") +
			theme.Entry.Render(m.filter.Value()) + "\n")
	}
	b.WriteString(theme.RuleLine(m.width) + "\n")

	// list window
	items := m.visible()
	reserved := 6
	if m.filtering || m.filter.Value() != "" {
		reserved++
	}
	h := m.height - reserved
	if h < 3 {
		h = 3
	}
	start := 0
	if m.cur >= h {
		start = m.cur - h + 1
	}
	end := start + h
	if end > len(items) {
		end = len(items)
	}

	if len(items) == 0 {
		msg := "no entries — press " + theme.Key.Render("a") + " to add one"
		if m.filter.Value() != "" {
			msg = "no match for " + theme.Accent.Render(m.filter.Value())
		}
		b.WriteString(theme.Dim.Render(msg) + "\n")
	} else {
		for i := start; i < end; i++ {
			b.WriteString(m.renderItem(items[i], i == m.cur) + "\n")
		}
	}

	b.WriteString(theme.RuleLine(m.width) + "\n")
	b.WriteString(m.statusLine() + "\n")
	b.WriteString(m.helpLine())
	return b.String()
}

func (m *Model) renderItem(name string, active bool) string {
	cursor := "  "
	if active {
		cursor = theme.Accent.Render("◆ ")
	}
	style := theme.Entry
	if active {
		style = theme.Title
	}
	return cursor + style.Render(name)
}

func (m *Model) statusLine() string {
	if m.busy {
		return m.spin.View() + " " + theme.Dim.Render("working…")
	}
	if m.err != nil {
		return theme.Error.Render("✗ ") + m.err.Error()
	}
	return m.status
}

func (m *Model) helpLine() string {
	if m.help {
		return theme.Dim.Render(strings.Join([]string{
			"a add", "e edit", "r rename", "d delete", "enter copy",
			"/ filter", "n audit", "R refresh", "q quit",
		}, theme.Faint.Render(" · ")))
	}
	return theme.Faint.Render("enter copy · a add · / filter · n audit · ? help · q quit")
}

func (m *Model) viewAudit() string {
	var b strings.Builder
	left := theme.Name.Render("vaultctl") + theme.Faint.Render("  audit")
	right := theme.Dim.Render(fmt.Sprintf("%d records", len(m.audit)))
	b.WriteString(padBetween(left, right, m.width) + "\n")
	b.WriteString(theme.RuleLine(m.width) + "\n")

	h := m.height - 6
	if h < 3 {
		h = 3
	}
	if len(m.audit) == 0 {
		b.WriteString(theme.Dim.Render("no audit records") + "\n")
	}
	start := m.scroll
	end := start + h
	if end > len(m.audit) {
		end = len(m.audit)
	}
	for i := start; i < end; i++ {
		b.WriteString(auditLine(m.audit[i], m.width) + "\n")
	}

	b.WriteString(theme.RuleLine(m.width) + "\n")
	b.WriteString(m.statusLine() + "\n")
	b.WriteString(theme.Faint.Render("esc back · j/k scroll · R refresh · q quit"))
	return b.String()
}

func auditLine(r client.AuditRecord, width int) string {
	ts := r.TS
	if len(ts) >= 19 && strings.Contains(ts, "T") {
		ts = strings.Replace(ts[:19], "T", " ", 1)
	}
	mark := theme.Dim.Render("·")
	switch r.Decision {
	case "deny", "denied":
		mark = theme.Error.Render("✗")
	case "allow", "allowed":
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
	return trunc(strings.Join(parts, "  "), width)
}

// padBetween places left and right at the edges of a width-wide line.
func padBetween(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// trunc cuts a string to width, respecting escape sequences.
func trunc(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "…")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
