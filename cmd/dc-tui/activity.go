package main

import (
	"context"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Activity timeline (v): latest container events for this workspace, RAM
// only. One pinned `docker events` stream runs while the board shows a
// workspace; it is closed and reaped on switch, engine change, fleet,
// Docker-down, foreground leave and quit. Every async result carries the
// session id (plus workspace + engine); anything else is dropped.

const (
	activityCap        = 200
	activityPendingCap = 100
	activityKnownCap   = 512
	activityVerdictCap = 512
)

// Swapped in tests.
var (
	activityDebounceDelay = 300 * time.Millisecond
	activityBackoff       = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 10 * time.Second}
	// activityHealthyAfter: a stream that lived this long resets the backoff.
	activityHealthyAfter = 10 * time.Second
)

var activitySeq atomic.Int64

type activityEntry struct {
	at   time.Time
	who  string
	what string
	gap  bool
}

type awaitInfo struct {
	project string
	info    memberInfo
}

type pendingEvent struct {
	ev  activityEvent
	gen int // discovery generation current when the event arrived
}

// activitySession is one continuous run for a workspace + engine. A
// reconnect keeps the session; a pause / switch / engine change ends it.
type activitySession struct {
	id     int64
	ws     string
	engine string
	ctx    context.Context
	cancel context.CancelFunc
}

type activityState struct {
	open   bool
	off    int
	follow bool

	entries []activityEntry

	sess      *activitySession
	stream    *activityStream
	streamGen int
	retry     int // next backoff index
	retrying  bool
	retryIn   time.Duration
	gapOpen   bool // a loss marker is shown and not yet followed by a reconnect
	rescan    bool // next connect asks for a full rescan
	resumeGap string

	known    map[string]memberInfo
	projects map[string]bool
	verdict  map[string]memberDecision // reject / await for inspected ids
	// awaiting: inspected ids whose compose project is not verified yet.
	awaiting map[string]awaitInfo
	pending  []pendingEvent
	checking map[string]bool // ids in the inspect in flight
	armed    bool            // debounce tick in flight
	// short snapshot ids waiting for a full-id inspect; normalizing: in flight.
	unresolved  map[string]memberInfo
	normalizing bool
}

type activityEventMsg struct {
	sess int64
	gen  int
	ev   activityEvent
}

type activityEndMsg struct {
	sess  int64
	gen   int
	err   error
	lived time.Duration
}

type activityRetryMsg struct{ sess int64 }

type activityRefreshMsg struct{ sess int64 }

// activityNormalizedMsg maps short snapshot ids to full engine ids.
type activityNormalizedMsg struct {
	sess   int64
	shorts map[string]memberInfo
	docs   map[string]inspectDoc
	err    error
}

type activityVerifiedMsg struct {
	sess int64
	ids  []string
	docs map[string]inspectDoc
	err  error
}

// --- lifecycle ---

// stopActivity ends the session, kills + reaps the stream and forgets the
// timeline (switch, fleet, engine change, quit).
func (m model) stopActivity() model {
	m = m.endActivitySession()
	open := m.activity.open
	m.activity = activityState{}
	if open && !m.fleet && !m.quitting {
		// Keep the view open on a new workspace; it starts empty.
		m.activity.open = true
		m.activity.follow = true
	}
	return m
}

// pauseActivity ends the session but keeps the timeline (foreground leave,
// Docker unavailable). note becomes a gap marker on resume.
func (m model) pauseActivity(note string, rescan bool) model {
	had := m.activity.sess != nil
	m = m.endActivitySession()
	if had && note != "" && len(m.activity.entries) > 0 {
		m.activity.resumeGap = note
	}
	if rescan {
		m.activity.rescan = true
	}
	return m
}

func (m model) endActivitySession() model {
	a := &m.activity
	if a.stream != nil {
		a.stream.close()
	}
	if a.sess != nil {
		a.sess.cancel()
	}
	a.sess = nil
	a.stream = nil
	a.streamGen = 0
	a.retry = 0
	a.retrying = false
	a.gapOpen = false
	a.pending = nil
	a.checking = nil
	a.verdict = nil
	a.awaiting = nil
	a.armed = false
	a.normalizing = false
	return m
}

// activityAllowed: the board shows one workspace with a usable snapshot and
// owns the terminal.
func (m model) activityAllowed() bool {
	if m.fleet || m.quitting || m.hostBlock || m.leaving != "" || m.workspace == "" || !m.loaded {
		return false
	}
	_, ok := engineArgs(m.engine)
	return ok
}

// syncActivity runs after every applied discovery: refresh membership,
// resolve events waiting on it, and start / restart / stop the stream.
func (m model) syncActivity(prevEngine string) (model, tea.Cmd) {
	if m.fleet {
		return m.stopActivity(), nil
	}
	if m.hostBlock {
		return m.pauseActivity("Docker was unavailable — rescanned", true), nil
	}
	if m.leaving != "" || m.quitting {
		return m, nil
	}
	a := m.activity
	if a.sess != nil && (a.sess.engine != m.engine || a.sess.ws != m.workspace) {
		// The engine moved under us: old events belong to the old engine.
		m = m.stopActivity()
		m = m.addActivityGap("engine changed to " + engineLabel(m.engine) + " — timeline cleared")
	}
	var cmds []tea.Cmd
	if m.load == loadReady {
		m = m.mergeSnapshotMembers()
		var c tea.Cmd
		m, c = m.resolveAwaiting()
		cmds = append(cmds, c)
	}
	if !m.activityAllowed() {
		if _, ok := engineArgs(m.engine); !ok && m.activity.sess != nil {
			m = m.pauseActivity("", true)
		}
		return m, tea.Batch(cmds...)
	}
	if m.activity.sess == nil {
		var c tea.Cmd
		m, c = m.startActivity()
		cmds = append(cmds, c)
	}
	var c tea.Cmd
	m, c = m.normalizeActivityIDs()
	cmds = append(cmds, c)
	return m, tea.Batch(cmds...)
}

// resumeActivity restarts the stream when the board gets the terminal back;
// the caller's reload is the rescan.
func (m model) resumeActivity() (model, tea.Cmd) {
	if m.activity.sess != nil || !m.activityAllowed() {
		return m, nil
	}
	m.activity.rescan = false
	return m.startActivity()
}

func (m model) startActivity() (model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.activity.sess = &activitySession{
		id:     activitySeq.Add(1),
		ws:     m.workspace,
		engine: m.engine,
		ctx:    ctx,
		cancel: cancel,
	}
	if note := m.activity.resumeGap; note != "" {
		m.activity.resumeGap = ""
		m = m.addActivityGap(note)
	}
	return m.connectActivity()
}

func (m model) connectActivity() (model, tea.Cmd) {
	a := &m.activity
	a.streamGen++
	a.retrying = false
	s, err := openActivityStream(a.sess.ctx, a.sess.engine)
	if err != nil {
		return m.scheduleActivityRetry()
	}
	a.stream = s
	cmds := []tea.Cmd{waitActivity(s, a.sess.id, a.streamGen)}
	if a.gapOpen {
		a.gapOpen = false
		m = m.addActivityGap("event stream reconnected — rescanning")
	}
	if m.activity.rescan {
		m.activity.rescan = false
		var c tea.Cmd
		m, c = m.requestRefresh()
		cmds = append(cmds, c)
	}
	return m, tea.Batch(cmds...)
}

// scheduleActivityRetry: 1s, 2s, 4s, then every 10s. Marks the gap once.
func (m model) scheduleActivityRetry() (model, tea.Cmd) {
	a := &m.activity
	a.stream = nil
	i := a.retry
	if i >= len(activityBackoff) {
		i = len(activityBackoff) - 1
	}
	d := activityBackoff[i]
	a.retry++
	a.retrying = true
	a.retryIn = d
	a.rescan = true
	if !a.gapOpen {
		a.gapOpen = true
		m = m.addActivityGap("event stream lost — events may be missing until it reconnects")
	}
	sid := m.activity.sess.id
	return m, tea.Tick(d, func(time.Time) tea.Msg { return activityRetryMsg{sess: sid} })
}

// waitActivity delivers the next event or the end of the stream. It never
// outlives the stream: a cancelled stream reports its end (then dropped as stale).
func waitActivity(s *activityStream, sid int64, gen int) tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-s.events:
			if ok {
				return activityEventMsg{sess: sid, gen: gen, ev: ev}
			}
		case <-s.ctx.Done():
		}
		msg := activityEndMsg{sess: sid, gen: gen, lived: time.Since(s.started)}
		select {
		case <-s.done:
			msg.err = s.err
		case <-time.After(activityStopWait):
		}
		return msg
	}
}

func (m model) activitySessionMatches(sid int64) bool {
	s := m.activity.sess
	return s != nil && s.id == sid && s.ws == m.workspace && s.engine == m.engine && !m.fleet
}

func (m model) applyActivityEvent(msg activityEventMsg) (model, tea.Cmd) {
	if !m.activitySessionMatches(msg.sess) || msg.gen != m.activity.streamGen || m.activity.stream == nil {
		return m, nil
	}
	m.activity.retry = 0
	next := waitActivity(m.activity.stream, msg.sess, msg.gen)
	m, c := m.ingestActivity(msg.ev)
	return m, tea.Batch(next, c)
}

func (m model) applyActivityEnd(msg activityEndMsg) (model, tea.Cmd) {
	if !m.activitySessionMatches(msg.sess) || msg.gen != m.activity.streamGen || m.activity.stream == nil {
		return m, nil
	}
	m.activity.stream.close()
	if msg.lived >= activityHealthyAfter {
		m.activity.retry = 0
	}
	return m.scheduleActivityRetry()
}

func (m model) applyActivityRetry(msg activityRetryMsg) (model, tea.Cmd) {
	if !m.activitySessionMatches(msg.sess) || !m.activity.retrying {
		return m, nil
	}
	if !m.activityAllowed() {
		return m, nil
	}
	return m.connectActivity()
}

func (m model) applyActivityRefresh(msg activityRefreshMsg) (model, tea.Cmd) {
	if !m.activitySessionMatches(msg.sess) {
		return m, nil
	}
	m.activity.armed = false
	return m.requestRefresh()
}

// armActivityRefresh coalesces relevant events into one soft discovery per
// 300ms window; requestRefresh keeps one in flight + one queued.
func (m model) armActivityRefresh() (model, tea.Cmd) {
	if m.activity.armed || m.activity.sess == nil {
		return m, nil
	}
	m.activity.armed = true
	sid := m.activity.sess.id
	return m, tea.Tick(activityDebounceDelay, func(time.Time) tea.Msg { return activityRefreshMsg{sess: sid} })
}

// --- membership + ingest ---

func (m model) ingestActivity(ev activityEvent) (model, tea.Cmd) {
	if ev.sidecar {
		return m, nil
	}
	a := &m.activity
	if info, ok := lookupKnown(a.known, ev.id); ok {
		return m.acceptActivity(ev, info)
	}
	if v, ok := a.verdict[ev.id]; ok && v == memberReject {
		return m, nil
	}
	if ev.action == "destroy" && !a.checking[ev.id] && a.verdict[ev.id] != memberAwait {
		// Unknown and already gone: nothing left to verify.
		return m, nil
	}
	a.pending = append(a.pending, pendingEvent{ev: ev, gen: m.loadGen})
	if len(a.pending) > activityPendingCap {
		a.pending = append([]pendingEvent(nil), a.pending[len(a.pending)-activityPendingCap:]...)
	}
	return m.verifyNextActivity()
}

func (m model) acceptActivity(ev activityEvent, info memberInfo) (model, tea.Cmd) {
	m = m.addActivityEntry(activityEntry{at: ev.at, who: entryWho(ev, info), what: describeEvent(ev)})
	// Every accepted event (health included) goes through the same coalesced
	// refresh: one discovery per window, never one per event.
	return m.armActivityRefresh()
}

// verifyNextActivity starts one bounded inspect for unverified pending ids.
func (m model) verifyNextActivity() (model, tea.Cmd) {
	a := &m.activity
	if a.sess == nil || len(a.checking) > 0 {
		return m, nil
	}
	seen := map[string]bool{}
	var ids []string
	for _, p := range a.pending {
		id := p.ev.id
		if seen[id] {
			continue
		}
		seen[id] = true
		if _, ok := a.verdict[id]; ok {
			continue
		}
		ids = append(ids, id)
		if len(ids) == activityInspectBatch {
			break
		}
	}
	if len(ids) == 0 {
		return m, nil
	}
	a.checking = map[string]bool{}
	for _, id := range ids {
		a.checking[id] = true
	}
	ctx, engine, sid := a.sess.ctx, a.sess.engine, a.sess.id
	return m, func() tea.Msg {
		docs, err := inspectContainers(ctx, engine, ids)
		return activityVerifiedMsg{sess: sid, ids: ids, docs: docs, err: err}
	}
}

func (m model) applyActivityVerified(msg activityVerifiedMsg) (model, tea.Cmd) {
	if !m.activitySessionMatches(msg.sess) {
		return m, nil
	}
	a := &m.activity
	a.checking = nil
	known := copyKnown(a.known)
	projects := copyProjects(a.projects)
	verdict := map[string]memberDecision{}
	for k, v := range a.verdict {
		verdict[k] = v
	}
	awaiting := map[string]awaitInfo{}
	for k, v := range a.awaiting {
		awaiting[k] = v
	}
	failed := map[string]bool{}
	if msg.err != nil {
		for _, id := range msg.ids {
			failed[id] = true
		}
	} else {
		composeKind := m.isComposeKind()
		// Apps first: a verified app verifies its compose project for the rest.
		undecided := append([]string(nil), msg.ids...)
		for changed := true; changed; {
			changed = false
			var rest []string
			for _, id := range undecided {
				d, ok := msg.docs[id]
				if !ok {
					verdict[id] = memberReject // gone before it could be verified
					continue
				}
				dec, proj := decideMember(m.workspace, composeKind, d, projects)
				if dec == memberAwait {
					awaiting[id] = awaitInfo{project: d.Config.Labels[labelProject], info: docInfo(d)}
				}
				switch dec {
				case memberAccept:
					known[id] = stamped(docInfo(d))
					delete(verdict, id)
					if proj != "" && !projects[proj] {
						projects[proj] = true
						changed = true
					}
				case memberReject:
					verdict[id] = memberReject
				default:
					rest = append(rest, id)
				}
			}
			undecided = rest
		}
		for _, id := range undecided {
			verdict[id] = memberAwait
		}
	}
	if len(verdict) > activityVerdictCap {
		verdict = map[string]memberDecision{}
		awaiting = map[string]awaitInfo{}
	}
	a.known, a.projects, a.verdict, a.awaiting = capKnown(known), projects, verdict, awaiting
	m = m.promoteAwaiting()
	return m.flushPending(failed)
}

// promoteAwaiting accepts waiting ids whose compose project has since been
// verified (by an app inspect or a discovery snapshot).
func (m model) promoteAwaiting() model {
	a := &m.activity
	if len(a.awaiting) == 0 {
		return m
	}
	known := copyKnown(a.known)
	awaiting := map[string]awaitInfo{}
	verdict := map[string]memberDecision{}
	for k, v := range a.verdict {
		verdict[k] = v
	}
	for id, w := range a.awaiting {
		if w.project != "" && a.projects[w.project] {
			known[id] = stamped(w.info)
			delete(verdict, id)
			continue
		}
		awaiting[id] = w
	}
	a.known, a.awaiting, a.verdict = capKnown(known), awaiting, verdict
	return m
}

// flushPending applies verdicts to waiting events in arrival order.
func (m model) flushPending(failed map[string]bool) (model, tea.Cmd) {
	var cmds []tea.Cmd
	var keep []pendingEvent
	await := false
	pend := m.activity.pending
	m.activity.pending = nil
	for _, p := range pend {
		if info, ok := lookupKnown(m.activity.known, p.ev.id); ok {
			var c tea.Cmd
			m, c = m.acceptActivity(p.ev, info)
			cmds = append(cmds, c)
			continue
		}
		if failed[p.ev.id] || m.activity.verdict[p.ev.id] == memberReject {
			continue
		}
		if m.activity.verdict[p.ev.id] == memberAwait {
			await = true
		}
		keep = append(keep, p)
	}
	m.activity.pending = keep
	if await {
		// Only a discovery can settle these; ask for one.
		var c tea.Cmd
		m, c = m.armActivityRefresh()
		cmds = append(cmds, c)
	}
	var c tea.Cmd
	m, c = m.verifyNextActivity()
	cmds = append(cmds, c)
	return m, tea.Batch(cmds...)
}

// mergeSnapshotMembers adds the snapshot's verified ids / projects to the
// retained set (retained ids keep destroy attributable).
func (m model) mergeSnapshotMembers() model {
	snap, projects := m.snapshotMembers()
	known := copyKnown(m.activity.known)
	unresolved := map[string]memberInfo{}
	for id, info := range snap {
		info = stamped(info)
		if isFullID(id) {
			known[id] = info
			continue
		}
		if full, ok := m.activity.fullFor(id); ok {
			known[full] = info
			continue
		}
		unresolved[id] = info
	}
	m.activity.unresolved = unresolved
	for p := range m.activity.projects {
		projects[p] = true
	}
	m.activity.known, m.activity.projects = capKnown(known), projects
	return m
}

var knownSeq atomic.Int64

func stamped(info memberInfo) memberInfo {
	info.seen = knownSeq.Add(1)
	return info
}

// capKnown keeps at most activityKnownCap ids, dropping the least recently
// verified (current members are re-stamped on every discovery, so retained
// ids of removed containers go first, oldest first).
func capKnown(known map[string]memberInfo) map[string]memberInfo {
	if len(known) <= activityKnownCap {
		return known
	}
	ids := make([]string, 0, len(known))
	for id := range known {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return known[ids[i]].seen > known[ids[j]].seen })
	out := make(map[string]memberInfo, activityKnownCap)
	for _, id := range ids[:activityKnownCap] {
		out[id] = known[id]
	}
	return out
}

// resolveAwaiting settles events that waited for a discovery started after
// they arrived: known now → accepted, otherwise not ours.
func (m model) resolveAwaiting() (model, tea.Cmd) {
	m = m.promoteAwaiting()
	if len(m.activity.pending) == 0 {
		return m, nil
	}
	if len(m.activity.unresolved) > 0 {
		// Snapshot short ids are not normalized yet; decide after that.
		return m, nil
	}
	verdict := map[string]memberDecision{}
	for k, v := range m.activity.verdict {
		verdict[k] = v
	}
	for _, p := range m.activity.pending {
		if verdict[p.ev.id] != memberAwait || p.gen >= m.loadGen {
			continue
		}
		if _, ok := lookupKnown(m.activity.known, p.ev.id); ok {
			delete(verdict, p.ev.id)
		} else {
			verdict[p.ev.id] = memberReject
		}
		delete(m.activity.awaiting, p.ev.id)
	}
	m.activity.verdict = verdict
	return m.flushPending(nil)
}

// --- entries ---

func (m model) addActivityEntry(e activityEntry) model {
	es := make([]activityEntry, 0, len(m.activity.entries)+1)
	es = append(es, m.activity.entries...)
	i := sort.Search(len(es), func(i int) bool { return es[i].at.After(e.at) })
	es = append(es, activityEntry{})
	copy(es[i+1:], es[i:])
	es[i] = e
	if len(es) > activityCap {
		drop := len(es) - activityCap
		es = append([]activityEntry(nil), es[drop:]...)
		m.activity.off -= drop
		if m.activity.off < 0 {
			m.activity.off = 0
		}
	}
	m.activity.entries = es
	if m.activity.follow {
		m.activity.off = m.activityMaxOff()
	}
	return m
}

func (m model) addActivityGap(text string) model {
	return m.addActivityEntry(activityEntry{at: time.Now(), what: text, gap: true})
}

func entryWho(ev activityEvent, info memberInfo) string {
	switch {
	case info.service != "":
		return info.service
	case info.name != "":
		return info.name
	case ev.service != "":
		return ev.service
	case ev.name != "":
		return ev.name
	}
	return shortID(ev.id)
}

func describeEvent(ev activityEvent) string {
	switch ev.action {
	case "create":
		return "created"
	case "start":
		return "started"
	case "stop":
		return "stopped"
	case "die":
		if ev.exit != "" {
			return "exited (code " + ev.exit + ")"
		}
		return "exited"
	case "restart":
		return "restarted"
	case "destroy":
		return "removed"
	case "oom":
		return "out of memory"
	case "health_status":
		return "health: " + ev.health
	}
	return ev.action
}

func engineLabel(engine string) string {
	if engine == "" {
		return "unknown"
	}
	return engine
}

func copyKnown(src map[string]memberInfo) map[string]memberInfo {
	out := make(map[string]memberInfo, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func copyProjects(src map[string]bool) map[string]bool {
	out := make(map[string]bool, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// fullFor finds an already-normalized full id for a short snapshot id.
func (a activityState) fullFor(short string) (string, bool) {
	if len(short) < 12 {
		return "", false
	}
	for full := range a.known {
		if strings.HasPrefix(full, short) {
			return full, true
		}
	}
	return "", false
}

// normalizeActivityIDs resolves short snapshot ids to full ids with one
// pinned, read-only, bounded inspect so events (always full ids) match
// exactly.
func (m model) normalizeActivityIDs() (model, tea.Cmd) {
	a := &m.activity
	if a.sess == nil || a.normalizing || len(a.unresolved) == 0 {
		return m, nil
	}
	shorts := make(map[string]memberInfo, len(a.unresolved))
	ids := make([]string, 0, len(a.unresolved))
	for id, info := range a.unresolved {
		if !validContainerID(id) {
			continue
		}
		shorts[id] = info
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return m, nil
	}
	sort.Strings(ids)
	a.normalizing = true
	ctx, engine, sid := a.sess.ctx, a.sess.engine, a.sess.id
	return m, func() tea.Msg {
		docs, err := inspectContainers(ctx, engine, ids)
		return activityNormalizedMsg{sess: sid, shorts: shorts, docs: docs, err: err}
	}
}

func (m model) applyActivityNormalized(msg activityNormalizedMsg) (model, tea.Cmd) {
	if !m.activitySessionMatches(msg.sess) {
		return m, nil
	}
	m.activity.normalizing = false
	if msg.err != nil {
		// Could not normalize: those snapshot ids stay unmatched.
		m.activity.unresolved = nil
		return m.resolveAwaiting()
	}
	known := copyKnown(m.activity.known)
	unresolved := map[string]memberInfo{}
	for k, v := range m.activity.unresolved {
		unresolved[k] = v
	}
	for full, d := range msg.docs {
		if !isFullID(full) || d.Config.Labels[labelForwardFor] != "" {
			continue
		}
		var match string
		for short := range msg.shorts {
			if strings.HasPrefix(full, short) {
				if match != "" {
					match = "" // ambiguous: two snapshot ids claim it
					break
				}
				match = short
			}
		}
		if match == "" {
			continue
		}
		known[full] = stamped(msg.shorts[match])
		delete(unresolved, match)
	}
	for short := range msg.shorts {
		// Gone or unmatched: nothing left to normalize.
		delete(unresolved, short)
	}
	m.activity.known, m.activity.unresolved = capKnown(known), unresolved
	return m.resolveAwaiting()
}
