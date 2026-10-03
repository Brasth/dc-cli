package main

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/Canvilled/dc-cli/internal/actions"
)

// Project actions picker (c). Uses internal/actions directly — the same code
// as the dc-actions CLI. Shared actions stay disabled until the user reviews
// the exact file and enables it here (or with dc-actions trust).

// actionsLoadedMsg is one read of the workspace's actions, tagged with the
// workspace and picker generation it was read for.
type actionsLoadedMsg struct {
	workspace string
	gen       int
	set       *actions.Set
	warn      string
}

// actionReadyMsg is the pre-run check: a fresh read of both files right
// before leaving. entry is what will execute; set is the fresh read.
type actionReadyMsg struct {
	workspace string
	gen       int
	entry     actions.Entry
	set       *actions.Set
	err       error
}

// actionDoneMsg: a foreground action returned. workspace is where it ran.
type actionDoneMsg struct {
	workspace string
	id        string
	label     string
	err       error
}

// openTrustStore is swapped in tests.
var openTrustStore = actions.OpenTrust

// readActions is one read of both files plus trust (off the Update loop).
func readActions(ws string) (*actions.Set, string) {
	trust, err := openTrustStore()
	warn := ""
	if err != nil {
		warn = "trust store unavailable: " + err.Error()
	}
	set := actions.Load(ws, trust)
	if set.TrustErr != nil {
		warn = set.TrustErr.Error()
	}
	return set, warn
}

func loadActionsCmd(ws string, gen int) tea.Cmd {
	return func() tea.Msg {
		set, warn := readActions(ws)
		return actionsLoadedMsg{workspace: ws, gen: gen, set: set, warn: warn}
	}
}

func (m model) openActionPicker() (model, tea.Cmd) {
	if m.fleet {
		return m.refuse("actions are per folder — open one first (enter / w)")
	}
	m.actGen++
	m.actOpen = true
	m.actReview = false
	m.actPreparing = false
	m.actFilter = ""
	m.actCursor = 0
	m.actSet = nil
	m.actWarn = ""
	m.more = false
	m.confirm = ""
	return m, loadActionsCmd(m.workspace, m.actGen)
}

func (m model) closeActionPicker() model {
	m.actGen++
	m.actOpen = false
	m.actReview = false
	m.actPreparing = false
	m.actFilter = ""
	m.actCursor = 0
	m.actSet = nil
	return m
}

func (m model) actMatches(workspace string, gen int) bool {
	return m.actOpen && workspace == m.workspace && gen == m.actGen
}

func (m model) applyActionsLoaded(msg actionsLoadedMsg) model {
	if !m.actMatches(msg.workspace, msg.gen) {
		return m
	}
	m.actSet = msg.set
	if msg.warn != "" || !m.actPreparing {
		m.actWarn = msg.warn
	}
	m.clampActCursor()
	return m
}

func (m model) actEntries() []actions.Entry {
	if m.actSet == nil {
		return nil
	}
	all := m.actSet.Entries()
	if m.actFilter == "" {
		return all
	}
	q := strings.ToLower(m.actFilter)
	var out []actions.Entry
	for _, e := range all {
		hay := strings.ToLower(e.ID + " " + e.Label + " " + actions.FormatArgv(e.Argv) + " " + e.Service)
		if strings.Contains(hay, q) {
			out = append(out, e)
		}
	}
	return out
}

func (m *model) clampActCursor() {
	n := len(m.actEntries())
	if m.actCursor >= n {
		m.actCursor = n - 1
	}
	if m.actCursor < 0 {
		m.actCursor = 0
	}
}

func (m model) handleActionKey(k string) (tea.Model, tea.Cmd) {
	if m.actReview {
		return m.handleReviewKey(k)
	}
	switch k {
	case "ctrl+c":
		return m.quit()
	case "esc":
		return m.closeActionPicker(), nil
	case "up", "ctrl+p":
		if m.actCursor > 0 {
			m.actCursor--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.actCursor < len(m.actEntries())-1 {
			m.actCursor++
		}
		return m, nil
	case "backspace":
		if m.actFilter != "" {
			_, size := utf8.DecodeLastRuneInString(m.actFilter)
			m.actFilter = m.actFilter[:len(m.actFilter)-size]
			m.clampActCursor()
		}
		return m, nil
	case "ctrl+t":
		// Review the shared file directly (also reached via a disabled entry).
		if m.actSet != nil && m.actSet.Shared.Present {
			return m.openReview(), nil
		}
		return m, nil
	case "enter":
		es := m.actEntries()
		if m.actCursor < 0 || m.actCursor >= len(es) {
			return m, nil
		}
		e := es[m.actCursor]
		if !e.Enabled {
			return m.openReview(), nil
		}
		return m.runProjectAction(e)
	case "space":
		k = " "
	}
	if utf8.RuneCountInString(k) == 1 {
		m.actFilter += k
		m.actCursor = 0
	}
	return m, nil
}

func (m model) openReview() model {
	m.actReview = true
	m.actReviewOff = 0
	m.actReviewSeenEnd = false
	return m.markReviewSeen()
}

// reviewPage is how many review body lines fit under the fixed header and
// above the fixed footer.
func (m model) reviewPage() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	return max(3, h-8)
}

func (m model) reviewMaxOff() int {
	return max(0, len(m.reviewLines(m.viewWidth()))-m.reviewPage())
}

func (m model) viewWidth() int {
	if m.width <= 0 {
		return 80
	}
	return m.width
}

// markReviewSeen records that the last line has been on screen.
func (m model) markReviewSeen() model {
	if m.actReviewOff >= m.reviewMaxOff() {
		m.actReviewSeenEnd = true
	}
	return m
}

func (m model) scrollReview(delta int) model {
	m.actReviewOff = min(max(0, m.actReviewOff+delta), m.reviewMaxOff())
	return m.markReviewSeen()
}

// handleReviewKey: scroll through every shared command; y enables exactly
// the reviewed bytes, and only after the whole list has been shown.
func (m model) handleReviewKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "ctrl+c":
		return m.quit()
	case "down", "j":
		return m.scrollReview(1), nil
	case "up", "k":
		return m.scrollReview(-1), nil
	case "pgdown", " ", "space", "ctrl+f":
		return m.scrollReview(m.reviewPage()), nil
	case "pgup", "ctrl+b":
		return m.scrollReview(-m.reviewPage()), nil
	case "home", "g":
		return m.scrollReview(-m.actReviewOff), nil
	case "end", "G":
		return m.scrollReview(m.reviewMaxOff()), nil
	case "y":
		if m.actSet == nil || m.actSet.Shared.Err != nil || !m.actSet.Shared.Present {
			return m, nil
		}
		m = m.markReviewSeen()
		if !m.actReviewSeenEnd {
			m.actWarn = "scroll to the end (↓ / pgdn / G) — every command must be shown before y"
			return m, nil
		}
		ws, hash, gen := m.actSet.Workspace, m.actSet.Shared.Hash, m.actGen
		m.actReview = false
		return m, func() tea.Msg {
			trust, err := openTrustStore()
			if err == nil {
				err = trust.ApproveReviewed(ws, hash)
			}
			set, warn := readActions(ws)
			if err != nil {
				warn = "not enabled: " + err.Error()
			}
			return actionsLoadedMsg{workspace: ws, gen: gen, set: set, warn: warn}
		}
	case "n", "esc", "q":
		m.actReview = false
		return m, nil
	}
	return m, nil
}

// runProjectAction starts the pre-run check off the Update loop: both files
// are read again and the selection must still resolve to the same source
// and the same command (and, for shared, the same approved bytes).
func (m model) runProjectAction(e actions.Entry) (tea.Model, tea.Cmd) {
	if m.leaving != "" || m.actPreparing || m.actSet == nil {
		return m, nil
	}
	m.actPreparing = true
	m.actWarn = "checking " + actions.DisplayText(e.Label) + "…"
	ws, gen, hash := m.actSet.Workspace, m.actGen, m.actSet.Shared.Hash
	return m, func() tea.Msg {
		set, _ := readActions(ws)
		fresh, err := verifySelection(set, e, hash)
		return actionReadyMsg{workspace: ws, gen: gen, entry: fresh, set: set, err: err}
	}
}

// verifySelection refuses to run anything the user did not pick: a changed
// or removed action, a different source winning precedence, or shared bytes
// that differ from the ones shown (even if those were re-approved since).
func verifySelection(set *actions.Set, sel actions.Entry, shownHash string) (actions.Entry, error) {
	fresh, err := set.Find(sel.ID)
	switch {
	case errors.Is(err, actions.ErrDisabled):
		return fresh, errors.New("shared actions are no longer trusted — review again")
	case errors.Is(err, actions.ErrNotFound):
		return fresh, errors.New(actions.DisplayText(sel.ID) + " is gone from the actions files")
	case err != nil:
		return fresh, errors.New("actions file is no longer valid: " + err.Error())
	}
	if fresh.Source != sel.Source {
		return fresh, errors.New(actions.DisplayText(sel.ID) + " now resolves to the " + string(fresh.Source) + " action — look again before running")
	}
	if sel.Source == actions.SourceShared && set.Shared.Hash != shownHash {
		return fresh, errors.New(".dc/actions.json changed since it was shown — review again")
	}
	if !sameAction(fresh.Action, sel.Action) {
		return fresh, errors.New(actions.DisplayText(sel.ID) + " changed since it was shown — look again before running")
	}
	return fresh, nil
}

func sameAction(a, b actions.Action) bool {
	if a.ID != b.ID || a.Label != b.Label || a.Service != b.Service || len(a.Argv) != len(b.Argv) {
		return false
	}
	for i := range a.Argv {
		if a.Argv[i] != b.Argv[i] {
			return false
		}
	}
	return true
}

func (m model) applyActionReady(msg actionReadyMsg) (model, tea.Cmd) {
	if !m.actMatches(msg.workspace, msg.gen) {
		return m, nil
	}
	m.actPreparing = false
	if msg.err != nil {
		// Show what is true now; nothing runs.
		m.actSet = msg.set
		m.actWarn = msg.err.Error()
		m.clampActCursor()
		return m, nil
	}
	m = m.closeActionPicker()
	m.actPending = &actionRun{workspace: msg.workspace, entry: msg.entry}
	return m.startLeave("action", "action")
}

type actionRun struct {
	workspace string
	entry     actions.Entry
}

// actionExecCmd hands the terminal to dc-exec --no-start with exactly the
// entry verified by the pre-run check (no further file read). The board
// ignores SIGINT while the child owns the terminal, so Ctrl+C stops the
// action only.
func (m model) actionExecCmd() (model, tea.Cmd) {
	run := m.actPending
	m.actPending = nil
	if run == nil {
		m.leaving = ""
		return m, nil
	}
	cmd := actions.Command(run.workspace, run.entry.Action)
	cmd.Stdin = os.Stdin
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return actionDoneMsg{workspace: run.workspace, id: run.entry.ID, label: run.entry.Label, err: err}
	})
}

func (m model) applyActionDone(msg actionDoneMsg) (model, tea.Cmd) {
	m.leaving = ""
	m.pending = ""
	label := actions.DisplayText(msg.label)
	code := actions.ExitCode(msg.err)
	var ee interface{ ExitCode() int }
	if msg.err != nil && !errors.As(msg.err, &ee) {
		m = m.withErr("action " + label + ": " + msg.err.Error())
	} else if code == 0 {
		m = m.withStatus("action " + label + " · exit 0")
	} else {
		m = m.withErr("action " + label + " · exit " + strconv.Itoa(code))
	}
	if msg.workspace != m.workspace || m.fleet {
		// Board moved on meanwhile: report, but do not refresh another context.
		return m, tea.EnableMouseAllMotion
	}
	m, resume := m.resumeActivity()
	m, reload := m.requestRefresh()
	return m, tea.Batch(tea.EnableMouseAllMotion, resume, reload)
}

// --- view ---

func (m model) actionPickerView() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	if m.actReview {
		return m.actionReviewView(w)
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("actions") + mutedStyle.Render("  "+trunc(m.workspace, max(8, w-12))) + "\n\n")
	b.WriteString(kv("filter", m.actFilter+"▏") + "\n\n")
	set := m.actSet
	switch {
	case set == nil:
		b.WriteString(mutedStyle.Render("  (loading…)") + "\n")
	default:
		es := m.actEntries()
		if len(es) == 0 {
			b.WriteString(mutedStyle.Render("  (no actions — add .dc/actions.json or your personal file; dc-actions --help)") + "\n")
		}
		for i, e := range es {
			state := okStyle.Render("on ")
			if !e.Enabled {
				state = warnStyle.Render("off")
			}
			line := "  " + state + "  " + actions.DisplayText(e.ID) + "  " + actions.DisplayText(e.Label) + mutedStyle.Render("  "+string(e.Source)+" · "+actions.Target(e.Action)+" · "+actions.FormatArgv(e.Argv))
			line = trunc(line, w)
			if i == m.actCursor {
				line = rowHover.Width(w).Render(line)
			}
			b.WriteString(line + "\n")
		}
		if set.PersonalErr != nil {
			b.WriteString("\n" + errStyle.Render(trunc("personal actions ignored: "+set.PersonalErr.Error(), w)) + "\n")
		}
		if set.Shared.Err != nil {
			b.WriteString("\n" + errStyle.Render(trunc("shared actions ignored: "+set.Shared.Err.Error(), w)) + "\n")
		} else if set.Shared.Present && !set.Shared.Trusted {
			b.WriteString("\n" + warnStyle.Render(trunc("shared .dc/actions.json is disabled — enter on an off entry (or ctrl+t) to review", w)) + "\n")
		}
	}
	if m.actWarn != "" {
		b.WriteString("\n" + warnStyle.Render(trunc(m.actWarn, w)) + "\n")
	}
	b.WriteString("\n" + hintStyle.Render(trunc("type to filter  ↑/↓  enter run (leaves the board, comes back)  ctrl+t review shared  esc back", w)) + "\n")
	b.WriteString(hintStyle.Render(trunc("actions run inside your containers and can modify data; nothing is started for you", w)) + "\n")
	return clipBlock(b.String(), w)
}

// reviewLines is the review body: every shared action — overridden ones
// too — with target and the complete argv, wrapped (never truncated).
func (m model) reviewLines(w int) []string {
	set := m.actSet
	if set == nil {
		return nil
	}
	if set.Shared.Err != nil {
		return wrapLines(set.Shared.Err.Error(), w, "", "  ")
	}
	var out []string
	for _, a := range set.Shared.Actions {
		head := actions.DisplayText(a.ID) + " — " + actions.DisplayText(a.Label)
		if set.Overridden(a.ID) {
			head += "  (overridden by your personal action)"
		}
		out = append(out, wrapLines(head, w, "  ", "    ")...)
		out = append(out, wrapLines("target: "+actions.Target(a), w, "    ", "      ")...)
		out = append(out, wrapLines("argv:   "+actions.FormatArgv(a.Argv), w, "    ", "            ")...)
		out = append(out, "")
	}
	return out
}

// wrapLines hard-wraps plain text to w columns, keeping every character.
func wrapLines(s string, w int, first, rest string) []string {
	width := max(8, w-ansi.StringWidth(rest))
	wrapped := strings.Split(ansi.Hardwrap(s, width, true), "\n")
	out := make([]string, len(wrapped))
	for i, line := range wrapped {
		if i == 0 {
			out[i] = first + line
		} else {
			out[i] = rest + line
		}
	}
	return out
}

// actionReviewView: fixed header, scrolled body, fixed footer.
func (m model) actionReviewView(w int) string {
	var b strings.Builder
	set := m.actSet
	b.WriteString(titleStyle.Render("review shared actions") + "\n")
	for _, l := range wrapLines(actions.DisplayText(set.Shared.Path), w, "", "") {
		b.WriteString(mutedStyle.Render(l) + "\n")
	}
	b.WriteString(mutedStyle.Render(trunc("sha256 "+set.Shared.Hash, w)) + "\n")
	lines := m.reviewLines(w)
	page := m.reviewPage()
	off := min(m.actReviewOff, max(0, len(lines)-page))
	end := min(len(lines), off+page)
	b.WriteString(hintStyle.Render(trunc("lines "+strconv.Itoa(min(off+1, len(lines)))+"–"+strconv.Itoa(end)+" of "+strconv.Itoa(len(lines))+"  ↑/↓ pgup/pgdn g/G", w)) + "\n")
	for _, l := range lines[off:end] {
		b.WriteString(l + "\n")
	}
	if set.Shared.Err != nil {
		b.WriteString("\n" + hintStyle.Render("invalid file cannot be enabled — esc back") + "\n")
		return b.String()
	}
	b.WriteString(warnStyle.Render(trunc("These run inside your containers and can modify data. Any change to the file disables them again.", w)) + "\n")
	if m.actWarn != "" {
		b.WriteString(warnStyle.Render(trunc(m.actWarn, w)) + "\n")
	}
	if end < len(lines) {
		b.WriteString(hintStyle.Render(trunc("more below — scroll to the end before y  n/esc back", w)) + "\n")
	} else {
		b.WriteString(hintStyle.Render("y enable exactly this file  n/esc back") + "\n")
	}
	return b.String()
}
