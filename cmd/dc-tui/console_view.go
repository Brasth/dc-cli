package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type consoleFrame struct {
	view                                 string
	buttons                              []button
	rowY, rowStart, rowVisible, rowWidth int
}

func (m model) consoleSize() (int, int) {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return w, h
}

func (m model) consoleHeader(minimal bool) []string {
	w, _ := m.consoleSize()
	state, id, ports := m.workspaceStatusParts()
	kind := "no config"
	if m.hasConfig {
		kind = "devcontainer"
	} else if m.hasCompose {
		kind = "compose"
	}
	name := plainText(filepath.Base(m.workspace))
	if m.fleet {
		name = "fleet"
		state = secondaryStyle.Render(fmt.Sprintf("%d workspaces", len(m.rows)))
		if m.hardLoading() {
			state = warnStyle.Render("checking…")
		} else if m.load == loadFailed && !m.loaded {
			state = badStyle.Render("unknown")
		}
		kind = ""
	}
	if minimal {
		return []string{splitHeader(titleStyle.Render(name), state, w)}
	}
	if kind != "" {
		state += mutedStyle.Render(" · " + kind)
	}
	engine := strings.TrimPrefix(engineLabel(m.engine), "context:")
	left := logoWord.Render("dc-cli") + mutedStyle.Render(" "+cliVersion()) + "  " + titleStyle.Render(name)
	lines := []string{
		splitHeader(left, state, w),
		splitHeader(mutedStyle.Render(plainText(m.workspace)), mutedStyle.Render("engine "+plainText(engine)), w),
	}
	meta := mutedStyle.Render("editor ") + secondaryStyle.Render(plainText(m.editor))
	if id != "" {
		meta = mutedStyle.Render("id "+plainText(id)+"  ports "+plainText(ports)+"  ") + meta
	}
	lines = append(lines, meta)
	var resource []string
	if m.pulse != "" {
		resource = append(resource, "load "+plainText(m.pulse))
	}
	for _, r := range m.optionalRows() {
		resource = append(resource, r[0]+" "+plainText(r[1]))
	}
	resourceLine := mutedStyle.Render(strings.Join(resource, "  "))
	if m.diskCritical {
		resourceLine = warnStyle.Render(strings.Join(resource, "  "))
	}
	lines = append(lines, resourceLine)
	// Four-cell-high brand mark occupies only ten columns in regular layouts.
	if w >= 80 {
		infoW := w - 12
		lines[0] = splitHeader(left, state, infoW)
		lines[1] = splitHeader(mutedStyle.Render(plainText(m.workspace)), mutedStyle.Render("engine "+plainText(engine)), infoW)
		mark := strings.Split(m.headerLogo(), "\n")
		for i := range lines {
			if i < len(mark) {
				lines[i] = cell(mark[i], 10) + "  " + trunc(lines[i], infoW)
			}
		}
	}
	return lines
}

func splitHeader(left, right string, width int) string {
	rw := ansi.StringWidth(right)
	if rw >= width {
		return trunc(right, width)
	}
	lw := max(0, width-rw-2)
	return cell(left, lw) + "  " + right
}

func (m model) serviceRole(s stackSvc) string {
	if len(m.rows) > 0 && s.ID != "" && s.ID == m.rows[0].ID {
		return "labeled"
	}
	return ""
}

func (m model) selectedService() (stackSvc, bool) {
	if len(m.stack) == 0 {
		return stackSvc{}, false
	}
	i := min(max(m.cursor, 0), len(m.stack)-1)
	return m.stack[i], true
}

func stateLabel(state string) string {
	state = plainText(state)
	switch state {
	case "running", "healthy":
		return okStyle.Render(state)
	case "exited", "stopped", "restarting", "starting":
		return warnStyle.Render(state)
	case "dead", "unhealthy":
		return badStyle.Render(state)
	default:
		return mutedStyle.Render(state)
	}
}

func (m model) serviceLine(s stackSvc, index, width int, minimal, wide bool) string {
	mark := "  "
	if index == m.cursor {
		mark = okStyle.Bold(true).Render("> ")
	} else if index == m.hoverStack {
		mark = secondaryStyle.Render("· ")
	}
	name := plainText(s.Service)
	if name == "" {
		name = plainText(s.Name)
	}
	nameW := min(16, max(8, width/5))
	if minimal {
		nameW = min(16, max(1, width-14))
	}
	line := mark + cell(titleStyle.Render(name), nameW) + " " + cell(stateLabel(s.Status), 10)
	if !minimal {
		line += " " + cell(okStyle.Render(m.serviceRole(s)), 8)
		if width >= 70 || wide {
			remaining := width - ansi.StringWidth(line) - 1
			containerW := remaining
			if !wide && width >= 80 {
				containerW = max(8, remaining/2)
			}
			line += " " + cell(mutedStyle.Render(plainText(s.Name)), containerW)
			if !wide && width >= 80 {
				line += " " + mutedStyle.Render(plainText(s.Image))
			}
		}
	}
	line = cell(line, width)
	if index == m.cursor {
		return selectedRow(line, width)
	}
	if index == m.hoverStack {
		return hoveredRow(line, width)
	}
	return line
}

func (m model) serviceColumns(width int, minimal, wide bool) string {
	if minimal {
		return ""
	}
	nameW := min(16, max(8, width/5))
	line := "  " + cell("SERVICE", nameW) + " " + cell("STATE", 10) + " " + cell("ROLE", 8)
	if width >= 70 || wide {
		remaining := width - ansi.StringWidth(line) - 1
		containerW := remaining
		if !wide && width >= 80 {
			containerW = max(8, remaining/2)
		}
		line += " " + cell("CONTAINER", containerW)
		if !wide && width >= 80 {
			line += " IMAGE"
		}
	}
	return mutedStyle.Render(trunc(line, width))
}

func (m model) detailLines(width int) []string {
	if m.more {
		lines := strings.Split(morePanel(m.editor, width), "\n")
		return lines[min(m.moreOff, len(lines)-1):]
	}
	s, ok := m.selectedService()
	if !ok {
		return nil
	}
	role := "compose sibling"
	if m.serviceRole(s) == "labeled" {
		role = "labeled app"
	}
	name := plainText(s.Service)
	if name == "" {
		name = plainText(s.Name)
	}
	lines := []string{titleStyle.Render(name), kv("state", stateLabel(s.Status)), kv("container", plainText(s.Name)), kv("image", plainText(s.Image)), kv("id", shortID(plainText(s.ID))), kv("role", role), "",
		keyHint("Enter", "shell into "+name), keyHint("l", "follow "+name+" logs"), keyHint("R", "restart "+name), keyHint("e", "shell into the labeled app"), keyHint("x", "remove all stack containers (asks)")}
	return lines
}

func (m model) boardFeedback() string {
	switch {
	case m.confirm != "":
		return warnStyle.Render("[y] Confirm · [n/Esc] Cancel · [q] Quit")
	case m.feedbackWarning:
		return warnStyle.Render("! " + plainText(m.status))
	case m.err != "":
		return badStyle.Render("✗ " + plainText(m.err))
	case m.leaving != "":
		return secondaryStyle.Render(leaveLine(m.leaving))
	case m.busy != "":
		return mutedStyle.Render(plainText(m.busy) + "…")
	case m.refreshing():
		return mutedStyle.Render("refreshing… last snapshot stays visible")
	case m.hardLoading():
		return mutedStyle.Render("checking… discovery has not finished")
	case m.feedbackSuccess:
		text := plainText(m.status)
		if text == "" {
			text = "Command completed"
		}
		return okStyle.Render("✓ " + text)
	case m.status != "":
		return secondaryStyle.Render(plainText(m.status))
	default:
		return mutedStyle.Render("Ready")
	}
}

// consoleLayout owns rendered geometry and hitboxes so resizing, scrolling,
// details, and banners cannot shift a click onto a different service.
func (m model) consoleLayout() consoleFrame {
	w, h := m.consoleSize()
	minimal := w < 60 || h < 16
	wide := w >= 110 && h >= 28 && !m.fleet
	frame := consoleFrame{rowY: -1, rowWidth: w}
	if minimal && m.more && m.confirm == "" {
		lines := m.consoleHeader(true)
		lines = append(lines, paneTitle("HELP · PgUp/PgDn", w)...)
		detail := m.detailLines(w)
		room := max(0, h-len(lines)-2)
		lines = append(lines, detail[:min(room, len(detail))]...)
		frame.view = m.finishScreen(lines, mutedStyle.Render("PgUp/PgDn scroll · Esc closes help"), []string{keyHint("?", "Help") + "  " + keyHint("q", "Quit")})
		return frame
	}
	lines := m.consoleHeader(minimal)
	if !minimal {
		lines = append(lines, "")
	}
	if m.updateBanner() != "" && !minimal {
		lines = append(lines, warnStyle.Render(plainText(m.updateBanner())))
	}
	if m.diskCritical {
		lines = append(lines, warnStyle.Render("Disk CRITICAL · [P] prune"))
	}
	if m.err != "" && !minimal {
		lines = append(lines, secondaryStyle.Render("Next: [r] Refresh status · [?] Help"))
	}
	leftW := w
	if wide {
		leftW = w / 2
		frame.rowWidth = leftW
	}
	n := m.rowCount()
	title := fmt.Sprintf("SERVICES %d", n)
	if m.fleet {
		title = fmt.Sprintf("WORKSPACES %d", n)
	}
	sectionY := len(lines)
	if minimal {
		lines = append(lines, secondaryStyle.Bold(true).Render(title))
	} else {
		lines = append(lines, paneTitle(title, leftW)...)
		if m.fleet {
			lines = append(lines, mutedStyle.Render("  STATE      WORKSPACE"))
		} else {
			lines = append(lines, m.serviceColumns(leftW, false, wide))
		}
	}
	frame.rowY = len(lines)
	// Reserve stable feedback, action and navigation rows before allocating list.
	bottomCount := 3
	if minimal {
		bottomCount = 2
	}
	confirmLines := m.confirmationLines(w)
	detailCount := len(confirmLines)
	if !wide && !minimal && !m.fleet {
		detailCount += 3
		if w < 80 {
			detailCount = len(confirmLines) + 2
		}
		if m.more {
			detailCount = len(confirmLines) + 6
		}
	}
	linkCount := 0
	if !minimal && !m.fleet && len(m.webLinks()) > 0 {
		linkCount = 2
	}
	page := max(0, h-len(lines)-bottomCount-detailCount-linkCount)
	frame.rowStart = max(0, min(m.cursor, n-1)-page+1)
	frame.rowVisible = min(page, max(0, n-frame.rowStart))
	if n == 0 || m.hardLoading() || (m.load == loadFailed && !m.loaded) {
		frame.rowVisible = 0
		msg := "No services — [u] Start this workspace"
		if m.fleet {
			msg = "No workspaces — [w] Recent folders"
		}
		if m.hardLoading() {
			msg = "checking containers… actions wait"
		} else if m.load == loadFailed && !m.loaded {
			msg = "Status unknown — [r] Retry discovery"
		} else if !m.hasConfig && !m.hasCompose && !m.fleet {
			msg = "No config — [u] Start a sandbox (asks)"
		}
		if page > 0 {
			lines = append(lines, mutedStyle.Render(trunc(msg, leftW)))
		}
	} else {
		for i := frame.rowStart; i < frame.rowStart+frame.rowVisible; i++ {
			if m.fleet {
				mark := "  "
				if i == m.cursor {
					mark = okStyle.Render("> ")
				} else if i == m.hoverStack {
					mark = "· "
				}
				r := m.rows[i]
				folder := r.LocalFolder
				if folder == "" {
					folder = r.Name
				}
				line := cell(mark+cell(stateLabel(r.Status), 10)+" "+plainText(folder), w)
				if i == m.cursor {
					line = selectedRow(line, w)
				} else if i == m.hoverStack {
					line = hoveredRow(line, w)
				}
				lines = append(lines, line)
			} else {
				lines = append(lines, m.serviceLine(m.stack[i], i, leftW, minimal, wide))
			}
		}
	}
	if frame.rowVisible < n && frame.rowVisible > 0 {
		// The count sits in the section title, never adds an unbudgeted list row.
		lines[sectionY] = secondaryStyle.Bold(true).Render(fmt.Sprintf("%s · %d–%d/%d", title, frame.rowStart+1, frame.rowStart+frame.rowVisible, n))
	}
	if linkCount > 0 {
		lines = append(lines, "")
		var specs []btnSpec
		for i, l := range m.webLinks() {
			key := strconv.Itoa(i + 1)
			if i >= 9 {
				key = "open"
			}
			specs = append(specs, btnSpec{key: "url:" + l.URL, label: "[" + key + "] " + plainText(l.Label), primary: true})
		}
		line, buttons := shortcutRow(specs, max(1, w-5), len(lines), m.hover)
		frame.buttons = append(frame.buttons, buttons...)
		lines = append(lines, mutedStyle.Render("open ")+line)
		// "open " is rendered before the links; move their click rectangles too.
		for i := len(frame.buttons) - len(buttons); i < len(frame.buttons); i++ {
			frame.buttons[i].x0 += 5
			frame.buttons[i].x1 += 5
		}
	}
	if wide {
		rightTitle := "SELECTED SERVICE"
		if m.more {
			rightTitle = "HELP · PgUp/PgDn"
		}
		right := append(paneTitle(rightTitle, w-leftW-3), m.detailLines(w-leftW-3)...)
		maxBody := h - bottomCount - len(confirmLines)
		for i, line := range right {
			y := sectionY + i
			if y >= maxBody {
				break
			}
			for len(lines) <= y {
				lines = append(lines, "")
			}
			lines[y] = cell(lines[y], leftW) + "   " + trunc(line, w-leftW-3)
		}
	} else if !minimal && !m.fleet {
		lines = append(lines, "")
		if m.more {
			detail := m.detailLines(w)
			limit := 5
			lines = append(lines, detail[:min(limit, len(detail))]...)
		} else if s, ok := m.selectedService(); ok {
			name := plainText(s.Service)
			if name == "" {
				name = plainText(s.Name)
			}
			if w >= 80 {
				lines = append(lines, mutedStyle.Render("selected ")+titleStyle.Render(name)+mutedStyle.Render("  image "+plainText(s.Image)+"  id "+shortID(plainText(s.ID))))
			}
			lines = append(lines, keyHint("Enter", "shell "+name)+"  "+keyHint("l", "logs")+"  "+keyHint("R", "restart "+name)+"  "+keyHint("x", "remove stack"))
		}
	}
	lines = append(lines, confirmLines...)
	for len(lines) < h-bottomCount {
		lines = append(lines, "")
	}
	if len(lines) > h-bottomCount {
		lines = lines[:max(0, h-bottomCount)]
	}
	feedback := m.boardFeedback()
	if minimal && m.confirm == "" && m.status == "" && m.err == "" && !m.hardLoading() && !m.refreshing() {
		if svc, ok := m.selectedService(); ok {
			feedback = mutedStyle.Render("Enter shells into " + plainText(svc.Service) + " · l logs · R restart")
		}
	}
	lines = append(lines, feedback)
	groups := m.footerSpecs(minimal, w)
	for _, specs := range groups {
		line, buttons := shortcutRow(specs, w, len(lines), m.hover)
		frame.buttons = append(frame.buttons, buttons...)
		lines = append(lines, line)
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	// Eliminate rectangles outside the clipped viewport, including truncated links.
	visible := frame.buttons[:0]
	for _, b := range frame.buttons {
		if b.y0 < h && b.x0 < w {
			b.x1 = min(b.x1, w)
			visible = append(visible, b)
		}
	}
	frame.buttons = visible
	frame.view = clipBlock(strings.Join(lines, "\n"), w)
	return frame
}

func (m model) footerSpecs(minimal bool, w int) [][]btnSpec {
	blocked := m.hardLoading() || (m.load == loadFailed && !m.loaded)
	if minimal && m.fleet {
		return [][]btnSpec{{{key: "enter", label: "[Enter] Open"}, {key: "?", label: "[?] Help"}, {key: "q", label: "[q] Quit"}}}
	}
	if minimal {
		return [][]btnSpec{{{key: "u", label: "[u] Start", disabled: !m.canStart() || blocked}, {key: "?", label: "[?] Help"}, {key: "q", label: "[q] Quit"}}}
	}
	if m.fleet {
		return [][]btnSpec{
			{{key: "enter", label: "[Enter] Open"}, {key: "w", label: "[w] Recent"}, {key: "f", label: "[f] Folder"}},
			{{key: "r", label: "[r] Refresh"}, {key: "?", label: "[?] Help"}, {key: "q", label: "[q] Quit"}},
		}
	}
	primary := []btnSpec{{key: "u", label: "[u] Start", disabled: !m.canStart() || blocked}, {key: "e", label: "[e] Shell", disabled: blocked}, {key: "s", label: "[s] Stop", disabled: blocked}}
	if w >= 80 {
		primary = append(primary, btnSpec{key: "o", label: "[o] Open", disabled: blocked}, btnSpec{key: "m", label: "[m] Files", disabled: blocked})
	}
	nav := []btnSpec{{key: "enter", label: "[Enter] Shell", disabled: blocked}, {key: "l", label: "[l] Logs", disabled: blocked || !m.canFollowLogs()}}
	if w >= 80 {
		nav = append(nav, btnSpec{key: "f", label: "[f] Fleet"})
	}
	nav = append(nav, btnSpec{key: "r", label: "[r] Refresh"}, btnSpec{key: "?", label: "[?] Help"}, btnSpec{key: "q", label: "[q] Quit"})
	if m.confirm != "" {
		return [][]btnSpec{primary, {{key: "y", label: "[y] Confirm"}, {key: "n", label: "[n/Esc] Cancel"}, {key: "q", label: "[q] Quit"}}}
	}
	return [][]btnSpec{primary, nav}
}

func shortcutRow(specs []btnSpec, width, y int, hover string) (string, []button) {
	var b strings.Builder
	var buttons []button
	x := 0
	for _, s := range specs {
		size := ansi.StringWidth(s.label)
		if x+size > width {
			break
		}
		if x > 0 {
			b.WriteString("  ")
		}
		style := secondaryStyle
		label := s.label
		if j := strings.Index(label, "]"); j >= 0 {
			label = okStyle.Render(label[:j+1]) + secondaryStyle.Render(label[j+1:])
		}
		if s.disabled {
			label = mutedStyle.Render(s.label)
		} else if hover == s.key {
			label = btnHover.Render(s.label)
		} else {
			label = style.Render(label)
		}
		b.WriteString(label)
		buttons = append(buttons, button{key: s.key, label: s.label, disabled: s.disabled, x0: x, x1: x + size, y0: y, y1: y + 1})
		x += size + 2
	}
	return b.String(), buttons
}

func (m model) confirmationLines(w int) []string {
	if m.confirm == "" {
		return nil
	}
	target := plainText(filepath.Base(m.workspace))
	title := "Confirm · " + m.confirm
	message := ""
	switch m.confirm {
	case "rm":
		message = "Remove all stack containers for " + target + "?"
	case "try":
		message = "No config — start a dc-try sandbox?"
	case "upgrade":
		message = "Upgrade dc-cli to " + plainText(m.updateLatest) + "?"
	case "prune":
		message = "Prune cache, dangling images and orphan sidecars?"
	}
	lines := []string{"", warnStyle.Bold(true).Render(title), warnStyle.Render(m.confirmAction())}
	for _, line := range wrapLines(message, w, "", "") {
		lines = append(lines, warnStyle.Render(line))
	}
	return lines
}
