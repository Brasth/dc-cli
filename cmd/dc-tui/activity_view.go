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
	if page := h - 4; page >= 4 {
		return page
	}
	return 4
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
	w := m.width
	if w <= 0 {
		w = 80
	}
	var b strings.Builder
	title := "activity — " + filepath.Base(m.workspace)
	b.WriteString(titleStyle.Render(trunc(title, w)) + "\n")
	b.WriteString(mutedStyle.Render(trunc(m.activityStatus()+" · "+engineLabel(m.engine)+" · latest "+strconv.Itoa(activityCap)+" in memory", w)) + "\n")
	es := m.activity.entries
	page := m.activityPage()
	if len(es) == 0 {
		b.WriteString(mutedStyle.Render("  (no events yet — start / stop / restart something in this workspace)") + "\n")
	} else {
		off := m.activity.off
		if off > len(es) {
			off = len(es)
		}
		end := off + page
		if end > len(es) {
			end = len(es)
		}
		for _, e := range es[off:end] {
			b.WriteString(formatActivityEntry(e, w) + "\n")
		}
	}
	b.WriteString(hintStyle.Render(trunc("j/k scroll  pgup/pgdn  g/G  c clear  esc back", w)) + "\n")
	return clipBlock(b.String(), w)
}

func formatActivityEntry(e activityEntry, w int) string {
	ts := e.at.Local().Format("15:04:05")
	if e.gap {
		return warnStyle.Render(trunc(ts+"  ── "+e.what+" ──", w))
	}
	who := e.who
	line := ts + "  " + padRight(trunc(who, 20), 20) + "  " + e.what
	style := mutedStyle
	switch {
	case strings.HasPrefix(e.what, "exited (code 0)"), e.what == "started", e.what == "health: healthy":
		style = okStyle
	case strings.HasPrefix(e.what, "exited"), e.what == "out of memory", e.what == "health: unhealthy":
		style = badStyle
	}
	return style.Render(trunc(line, w))
}

func padRight(s string, n int) string {
	if d := n - len([]rune(s)); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}
