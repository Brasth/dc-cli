package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Discovery is split in two so the board paints as soon as it can:
//   essentials (host check, rows, stack) → reloadMsg
//   optionals  (disk, ports, nets)       → optionalMsg, one per section
// Every result carries the context it was produced for; anything that does
// not match the board's current tag is dropped.

// discoveryTag identifies the board context a result belongs to.
type discoveryTag struct {
	workspace string
	fleet     bool
	engine    string
	gen       int
}

func (m model) tag() discoveryTag {
	return discoveryTag{workspace: m.workspace, fleet: m.fleet, engine: m.engine, gen: m.loadGen}
}

// reloadMsg carries the essentials. engine is what the docker CLI targeted
// while they were read.
type reloadMsg struct {
	gen       int
	workspace string
	fleet     bool
	engine    string
	rows      []container
	stack     []stackSvc
	host      hostReport
	hostErr   error
	err       error
	elapsed   time.Duration
}

func (m model) reloadMatches(msg reloadMsg) bool {
	return msg.gen == m.loadGen && msg.workspace == m.workspace && msg.fleet == m.fleet
}

// reloadCmd starts a discovery generation. The previous generation's probes
// (essential or optional) are cancelled.
func (m model) reloadCmd() tea.Cmd {
	ws := m.workspace
	fleet := m.fleet
	gen := m.loadGen
	ctx := m.probes.beginGeneration()
	return func() tea.Msg {
		return runEssentials(ctx, ws, fleet, gen)
	}
}

func runEssentials(parent context.Context, ws string, fleet bool, gen int) (msg reloadMsg) {
	start := time.Now()
	msg = reloadMsg{gen: gen, workspace: ws, fleet: fleet}
	defer func() { msg.elapsed = time.Since(start) }()
	ctx, cancel := context.WithTimeout(parent, essentialDeadline)
	defer cancel()

	msg.engine = probeEngine(ctx)
	msg.host, msg.hostErr = runHostDiagnose(ctx)
	if msg.hostErr == nil && msg.host.blocked() {
		return msg
	}
	if errors.Is(msg.hostErr, errProbeCancelled) {
		msg.err = msg.hostErr
		return msg
	}

	args := []string{"--json"}
	if fleet {
		args = append(args, "--all")
	} else {
		args = append(args, "--workspace", ws)
	}
	var (
		wg      sync.WaitGroup
		rowsOut []byte
		rowsErr error
		stack   []stackSvc
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		rowsOut, rowsErr = probe(ctx, "dc-ls", args...)
	}()
	if !fleet {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Stack is best-effort: a failed list leaves the labeled app usable.
			if out, err := probe(ctx, "dc-exec", "--list", "--json", ws); err == nil {
				_ = json.Unmarshal(out, &stack)
			}
		}()
	}
	wg.Wait()
	if rowsErr != nil {
		msg.err = rowsErr
		return msg
	}
	var rows []container
	if err := json.Unmarshal(rowsOut, &rows); err != nil {
		msg.err = err
		return msg
	}
	msg.rows = rows
	msg.stack = stack
	return msg
}

// handleReload applies essentials and, on success, launches the optional probes.
func (m model) handleReload(msg reloadMsg) (model, tea.Cmd) {
	if !m.reloadMatches(msg) {
		return m, nil
	}
	prevEngine := m.engine
	hadSnapshot := m.loaded
	m = m.applyReload(msg)
	if hadSnapshot && prevEngine != m.engine {
		// Disk / ports / nets belong to the old engine.
		m = m.clearOptional()
	}
	var cmds []tea.Cmd
	if m.load == loadReady {
		var opt tea.Cmd
		m, opt = m.startOptional()
		cmds = append(cmds, opt)
	}
	m, more := m.afterDiscovery(prevEngine)
	cmds = append(cmds, more)
	return m, tea.Batch(cmds...)
}

// --- optional sections ---

const (
	sectionDisk  = "disk"
	sectionPorts = "ports"
	sectionNets  = "nets"
)

type sectionStatus int

const (
	sectionIdle sectionStatus = iota
	sectionLoading
	sectionReady
	sectionUnavailable
	sectionStale
)

type sectionState struct {
	status sectionStatus
	err    string
}

type diskInfo struct {
	compact string
	guest   string
}

type optionalMsg struct {
	tag     discoveryTag
	section string
	disk    diskInfo
	fwd     []portPair
	nets    netReport
	err     error
}

func (m model) clearOptional() model {
	m.disk, m.diskGuest, m.diskCritical = "", "", false
	m.fwdMaps = nil
	m.net = netReport{}
	m.diskSec, m.portsSec, m.netsSec = sectionState{}, sectionState{}, sectionState{}
	return m
}

// startOptional marks each section loading (values stay visible) and runs
// the three probes concurrently under the current generation.
func (m model) startOptional() (model, tea.Cmd) {
	if m.fleet || m.hostBlock {
		return m, nil
	}
	tag := m.tag()
	ctx := m.probes.generation()
	ws := m.workspace
	stack := append([]stackSvc(nil), m.stack...)
	m.diskSec = sectionState{status: sectionLoading}
	m.portsSec = sectionState{status: sectionLoading}
	m.netsSec = sectionState{status: sectionLoading}
	return m, tea.Batch(
		func() tea.Msg { return fetchDisk(ctx, tag) },
		func() tea.Msg { return fetchPorts(ctx, tag, ws, stack) },
		func() tea.Msg { return fetchNetsSection(ctx, tag, ws) },
	)
}

func fetchDisk(ctx context.Context, tag discoveryTag) optionalMsg {
	msg := optionalMsg{tag: tag, section: sectionDisk}
	if info, ok := diskCache.get(tag.engine, time.Now()); ok {
		msg.disk = info
		return msg
	}
	out, err := probe(ctx, "dc-df", "--json")
	if err != nil {
		msg.err = err
		return msg
	}
	info, err := parseDisk(out)
	if err != nil {
		msg.err = err
		return msg
	}
	diskCache.put(tag.engine, info, time.Now())
	msg.disk = info
	return msg
}

func parseDisk(out []byte) (diskInfo, error) {
	var df struct {
		Compact string `json:"compact"`
		Colima  *struct {
			GuestRoot string `json:"guest_root"`
		} `json:"colima"`
	}
	if err := json.Unmarshal(out, &df); err != nil {
		return diskInfo{}, err
	}
	info := diskInfo{compact: strings.TrimSpace(df.Compact)}
	if df.Colima != nil {
		info.guest = strings.TrimSpace(df.Colima.GuestRoot)
	}
	return info, nil
}

func fetchPorts(ctx context.Context, tag discoveryTag, ws string, stack []stackSvc) optionalMsg {
	msg := optionalMsg{tag: tag, section: sectionPorts}
	fwd, err := listFwdMaps(ctx, ws)
	if err != nil {
		msg.err = err
		return msg
	}
	pub, err := listStackPorts(ctx, stack)
	if err != nil {
		msg.err = err
		return msg
	}
	msg.fwd = mergePairs(fwd, pub)
	return msg
}

func fetchNetsSection(ctx context.Context, tag discoveryTag, ws string) optionalMsg {
	msg := optionalMsg{tag: tag, section: sectionNets}
	out, err := probe(ctx, "dc-net", "--json", ws)
	if err != nil {
		msg.err = err
		return msg
	}
	rep, err := parseNet(out)
	if err != nil {
		msg.err = err
		return msg
	}
	msg.nets = rep
	return msg
}

// applyOptional takes one section result. A failed refresh keeps the old
// value as stale; a failure with nothing to show is unavailable.
func (m model) applyOptional(msg optionalMsg) model {
	if msg.tag != m.tag() {
		return m
	}
	fail := func(has bool) sectionState {
		st := sectionState{status: sectionUnavailable, err: compactLines(msg.err.Error(), 1)}
		if has {
			st.status = sectionStale
		}
		return st
	}
	switch msg.section {
	case sectionDisk:
		if msg.err != nil {
			m.diskSec = fail(m.disk != "")
			return m
		}
		m.disk, m.diskGuest = msg.disk.compact, msg.disk.guest
		m.diskCritical = diskLooksCritical(m.disk, m.diskGuest)
		m.diskSec = sectionState{status: sectionReady}
	case sectionPorts:
		if msg.err != nil {
			m.portsSec = fail(len(m.fwdMaps) > 0)
			return m
		}
		m.fwdMaps = msg.fwd
		m.portsSec = sectionState{status: sectionReady}
	case sectionNets:
		if msg.err != nil {
			m.netsSec = fail(len(m.net.Networks) > 0)
			return m
		}
		m.net = msg.nets
		m.netsSec = sectionState{status: sectionReady}
	}
	return m
}

// sectionNote is the header text for a section that has no fresh value.
func sectionNote(st sectionState, hasValue bool) string {
	switch st.status {
	case sectionLoading:
		if !hasValue {
			return "loading…"
		}
	case sectionUnavailable:
		return "unavailable"
	case sectionStale:
		return "stale"
	}
	return ""
}

// --- disk cache: 30s per engine; explicit refresh and prune invalidate ---

var diskCacheTTL = 30 * time.Second

type diskCacheEntry struct {
	info diskInfo
	at   time.Time
}

type diskCacheStore struct {
	mu      sync.Mutex
	entries map[string]diskCacheEntry
}

var diskCache = &diskCacheStore{entries: map[string]diskCacheEntry{}}

func (c *diskCacheStore) get(engine string, now time.Time) (diskInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[engine]
	if !ok || now.Sub(e.at) >= diskCacheTTL {
		return diskInfo{}, false
	}
	return e.info, true
}

func (c *diskCacheStore) put(engine string, info diskInfo, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[engine] = diskCacheEntry{info: info, at: now}
}

func (c *diskCacheStore) invalidate(engine string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, engine)
}

func (c *diskCacheStore) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = map[string]diskCacheEntry{}
}

// --- context lifecycle ---

// renewProbes kills every probe tree of the old context and starts a new
// session. Used whenever the workspace or fleet flag changes.
func (m model) renewProbes() model {
	m.probes.close()
	m.probes = newProbeSession()
	return m
}

// quit tears down background readers and probe trees, then exits.
func (m model) quit() (model, tea.Cmd) {
	m.quitting = true
	m = m.shutdown()
	return m, tea.Quit
}

// shutdown closes everything the board started in the background.
func (m model) shutdown() model {
	m = m.closeLogs()
	m = m.closeTop()
	m = m.stopActivity()
	m.probes.close()
	return m
}

// requestRefresh asks for a soft discovery without stacking them: one in
// flight, at most one queued behind it.
func (m model) requestRefresh() (model, tea.Cmd) {
	if m.load == loadPending {
		m.discoveryPending = true
		return m, nil
	}
	if m.loaded {
		return m.beginSoftReload()
	}
	return m.beginHardReload()
}

// afterDiscovery runs the queued refresh, if any, once a generation settles.
func (m model) afterDiscovery(prevEngine string) (model, tea.Cmd) {
	var cmds []tea.Cmd
	if m.discoveryPending && m.load != loadPending {
		m.discoveryPending = false
		var c tea.Cmd
		m, c = m.requestRefresh()
		cmds = append(cmds, c)
	}
	var ev tea.Cmd
	m, ev = m.syncActivity(prevEngine)
	cmds = append(cmds, ev)
	return m, tea.Batch(cmds...)
}

// explicitRefresh is the user's r: drop cached disk numbers, then reload now
// (a newer generation cancels any in-flight one).
func (m model) explicitRefresh() (model, tea.Cmd) {
	diskCache.invalidate(m.engine)
	if m.loaded {
		return m.beginSoftReload()
	}
	return m.beginHardReload()
}
