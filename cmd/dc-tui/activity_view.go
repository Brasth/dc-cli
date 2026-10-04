package main

import (
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openActivity shows the timeline (v). Fleet has no timeline.
func (m model) openActivity() (model, tea.Cmd) {
	if m.fleet {
		return m.refuse("open a folder (enter / click) — activity is per workspace")
	}
	m.activity.open = true
	m.activity.follow = true
	m.activity.off = m.activityMaxOff()
	m.more = false
	m.confirm = ""
	return m, nil
}

func (m model) closeActivity() model {
	m.activity.open = false
	m.activity.off = 0
	return m
}

func (m model) handleActivityKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "ctrl+c":
		return m.quit()
	case "q", "esc", "v":
		return m.closeActivity(), nil
	case "c":
		m.activity.entries = nil
		m.activity.off = 0
		m.activity.follow = true
		return m, nil
	case "j", "down":
		return m.scrollActivity(1), nil
	case "k", "up":
		return m.scrollActivity(-1), nil
	case "pgdown", " ", "space":
		return m.scrollActivity(m.activityPage()), nil
	case "pgup":
		return m.scrollActivity(-m.activityPage()), nil
	case "g", "home":
		m.activity.off = 0
		m.activity.follow = false
		return m, nil
	case "G", "end":
		m.activity.off = m.activityMaxOff()
		m.activity.follow = true
		return m, nil
	}
	return m, nil
}

func (m model) activityPage() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	return max(1, h-8)
}

func (m model) activityMaxOff() int {
	if n := len(m.activity.entries) - m.activityPage(); n > 0 {
		return n
	}
	return 0
}

func (m model) scrollActivity(delta int) model {
	m.activity.off += delta
	maxOff := m.activityMaxOff()
	if m.activity.off > maxOff {
		m.activity.off = maxOff
	}
	if m.activity.off < 0 {
		m.activity.off = 0
	}
	m.activity.follow = m.activity.off >= maxOff
	return m
}

// activityStatus is the stream state shown in the header.
func (m model) activityStatus() string {
	a := m.activity
	switch {
	case a.stream != nil:
		return "live"
	case a.retrying:
		return "reconnecting (" + a.retryIn.String() + ")"
	case m.hostBlock:
		return "Docker unavailable"
	case m.loaded && !m.fleet:
		if _, ok := engineArgs(m.engine); !ok {
			return "engine unknown — no stream"
		}
	}
	if m.hardLoading() || m.refreshing() {
		return "waiting for discovery"
	}
	return "idle"
}

func (m model) activityView() string {
	w, _ := m.consoleSize()
	lines := m.screenHeader("ACTIVITY · "+plainText(filepath.Base(m.workspace)), m.activityStatus())
	lines = append(lines, mutedStyle.Render("  TIME      SERVICE               EVENT"))
	es := m.activity.entries
	page := m.activityPage()
	if len(es) == 0 {
		lines = append(lines, mutedStyle.Render("(no events yet — start / stop / restart something in this workspace)"))
	} else {
		off := min(max(0, m.activity.off), m.activityMaxOff())
		for _, e := range es[off:min(len(es), off+page)] {
			lines = append(lines, formatActivityEntry(e, w))
		}
	}
	feedback := mutedStyle.Render(m.activityStatus() + " · latest " + strconv.Itoa(activityCap) + " in memory")
	if m.activity.retrying {
		feedback = warnStyle.Render("Stream lost — " + m.activityStatus())
	}
	return m.finishScreen(lines, feedback, []string{keyHint("j/k", "Scroll") + "  " + keyHint("PgUp/PgDn", "Page") + "  " + keyHint("g/G", "Top/end") + "  " + keyHint("c", "Clear") + "  " + keyHint("Esc", "Back")})
}

func formatActivityEntry(e activityEntry, w int) string {
	ts := e.at.Local().Format("15:04:05")
	if e.gap {
		return warnStyle.Render(trunc(ts+"  ── "+e.what+" ──", w))
	}
	who := plainText(e.who)
	line := ts + "  " + padRight(trunc(who, 20), 20) + "  " + plainText(e.what)
	style := mutedStyle
	switch {
	case strings.HasPrefix(e.what, "exited (code 0)"), e.what == "started", e.what == "health: healthy":
		style = okStyle
	case strings.HasPrefix(e.what, "exited"), e.what == "out of memory", e.what == "health: unhealthy":
		style = badStyle
	}
	return style.Render(trunc(line, w))
}

func padRight(s string, n int) string { return cell(s, n) }
