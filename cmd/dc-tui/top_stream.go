package main

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type topLineMsg struct{ line string }

type topDoneMsg struct{ err error }

// startStatsFollow is replaced in tests.
var startStatsFollow = dockerStatsFollow

func dockerStatsFollow(ids []string) (io.ReadCloser, func(), error) {
	args := []string{"stats", "--format", "{{json .}}"}
	args = append(args, ids...)
	cmd := exec.Command("docker", args...)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		return nil, nil, err
	}
	var once sync.Once
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
		close(done)
	}()
	stop := func() {
		once.Do(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		})
		<-done
	}
	return pr, stop, nil
}

func (m model) topIDs() []string {
	var ids []string
	for _, s := range m.stack {
		if s.Status == "running" && s.ID != "" {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) == 0 && len(m.rows) > 0 && m.rows[0].ID != "" {
		ids = append(ids, m.rows[0].ID)
	}
	return ids
}

func (m model) beginTopStream() (model, tea.Cmd) {
	ids := m.topIDs()
	if len(ids) == 0 {
		return m, nil
	}
	r, stop, err := startStatsFollow(ids)
	if err != nil {
		return m, nil
	}
	m.topStop = stop
	m.topR = bufio.NewReader(r)
	if m.topHist == nil {
		m.topHist = map[string]sparkHist{}
	}
	m.topLast = time.Now()
	m.topStale = false
	return m, readTopLine(m.topR)
}

func readTopLine(r *bufio.Reader) tea.Cmd {
	if r == nil {
		return nil
	}
	return func() tea.Msg {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			return topLineMsg{line: strings.TrimRight(line, "\r\n")}
		}
		if err != nil {
			return topDoneMsg{err: err}
		}
		return topLineMsg{line: ""}
	}
}

func (m model) applyTopLine(line string) (model, tea.Cmd) {
	if !m.topOpen {
		return m, nil
	}
	box, ok := parseStatsLine(line)
	if ok {
		m = m.mergeTopBox(box)
		m.topLast = time.Now()
		m.topStale = false
	}
	return m, readTopLine(m.topR)
}

func (m model) mergeTopBox(box statsBox) model {
	idx := -1
	for i, c := range m.topSnap.Containers {
		if idsMatch(c.ID, box.ID) || (box.Name != "" && c.Name == box.Name) {
			idx = i
			if box.Service == "" {
				box.Service = c.Service
			}
			if box.Name == "" {
				box.Name = c.Name
			}
			if box.ID == "" || (c.ID != "" && len(c.ID) > len(box.ID)) {
				box.ID = c.ID
			}
			m.topSnap.Containers[i] = box
			break
		}
	}
	if idx < 0 {
		m.topSnap.Containers = append(m.topSnap.Containers, box)
		idx = len(m.topSnap.Containers) - 1
	}
	id := m.topSnap.Containers[idx].ID
	if m.topHist == nil {
		m.topHist = map[string]sparkHist{}
	}
	mem := float64(box.MemUsedBytes)
	m.topHist[id] = appendSpark(m.topHist[id], box.CPUPct, mem)
	return m
}

func idsMatch(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}
