package main

import (
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// Highlight even when stdout is not a TTY (tests, some tmux).
	lipgloss.SetColorProfile(termenv.ANSI256)
}

type logSpan struct {
	start, end int
	text       string
}

var (
	reApacheReq = regexp.MustCompile(`"(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) ([^"]*?) (HTTP/[0-9.]+)"\s+([1-5][0-9]{2})\b`)
	reIPv4      = regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`)
	reBracketT  = regexp.MustCompile(`\[[0-9]{1,2}/[A-Za-z]{3}/[0-9]{4}:[0-9]{2}:[0-9]{2}:[0-9]{2} [+-][0-9]{4}\]`)
	reISOTime   = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?\b`)
	reLevel     = regexp.MustCompile(`(?i)\b(ERROR|ERR|FATAL|WARN(?:ING)?|INFO|DEBUG|TRACE)\b`)
	reMethod    = regexp.MustCompile(`\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\b`)
)

func methodStyle(m string) lipgloss.Style {
	switch m {
	case "DELETE":
		return logDel
	case "POST", "PUT", "PATCH":
		return logWrite
	default:
		return logGET
	}
}

func logStatusStyle(code string) lipgloss.Style {
	if code == "" {
		return log2xx
	}
	switch code[0] {
	case '3':
		return log3xx
	case '4':
		return log4xx
	case '5':
		return log5xx
	default:
		return log2xx
	}
}

func levelStyle(s string) lipgloss.Style {
	switch s {
	case "ERROR", "ERR", "FATAL", "error", "err", "fatal":
		return logErr
	case "WARN", "WARNING", "warn", "warning":
		return logWarn
	default:
		return logInfo
	}
}

func colorizeLogLine(line string) string {
	if line == "" {
		return line
	}
	var spans []logSpan
	covered := make([]bool, len(line))
	add := func(start, end int, rendered string) {
		if start < 0 || end > len(line) || start >= end {
			return
		}
		for i := start; i < end; i++ {
			if covered[i] {
				return
			}
		}
		for i := start; i < end; i++ {
			covered[i] = true
		}
		spans = append(spans, logSpan{start: start, end: end, text: rendered})
	}

	for _, loc := range reApacheReq.FindAllStringSubmatchIndex(line, -1) {
		add(loc[2], loc[3], methodStyle(line[loc[2]:loc[3]]).Render(line[loc[2]:loc[3]]))
		add(loc[4], loc[5], logPath.Render(line[loc[4]:loc[5]]))
		add(loc[6], loc[7], logProto.Render(line[loc[6]:loc[7]]))
		add(loc[8], loc[9], logStatusStyle(line[loc[8]:loc[9]]).Render(line[loc[8]:loc[9]]))
	}
	for _, loc := range reIPv4.FindAllStringIndex(line, -1) {
		add(loc[0], loc[1], logIP.Render(line[loc[0]:loc[1]]))
	}
	for _, loc := range reBracketT.FindAllStringIndex(line, -1) {
		add(loc[0], loc[1], logTime.Render(line[loc[0]:loc[1]]))
	}
	for _, loc := range reISOTime.FindAllStringIndex(line, -1) {
		add(loc[0], loc[1], logTime.Render(line[loc[0]:loc[1]]))
	}
	for _, loc := range reLevel.FindAllStringIndex(line, -1) {
		add(loc[0], loc[1], levelStyle(line[loc[0]:loc[1]]).Render(line[loc[0]:loc[1]]))
	}
	for _, loc := range reMethod.FindAllStringIndex(line, -1) {
		add(loc[0], loc[1], methodStyle(line[loc[0]:loc[1]]).Render(line[loc[0]:loc[1]]))
	}

	if len(spans) == 0 {
		return mutedStyle.Render(line)
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	var b strings.Builder
	cursor := 0
	for _, s := range spans {
		if cursor < s.start {
			b.WriteString(line[cursor:s.start])
		}
		b.WriteString(s.text)
		cursor = s.end
	}
	if cursor < len(line) {
		b.WriteString(line[cursor:])
	}
	return b.String()
}
