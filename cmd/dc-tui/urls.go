package main

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Published host→container pair from docker Ports or a dc-forward sidecar.
type portPair struct {
	Host      int
	Container int
}

type webLink struct {
	URL   string
	Label string
}

var publishedPair = regexp.MustCompile(`:(\d+)->(\d+)(?:/(tcp|udp))?`)

var infraPorts = map[int]struct{}{
	22: {}, 23: {}, 25: {}, 53: {}, 110: {}, 143: {}, 445: {}, 465: {}, 587: {},
	993: {}, 995: {}, 1433: {}, 1521: {}, 2049: {}, 2375: {}, 2376: {},
	3306: {}, 3389: {}, 5432: {}, 5672: {}, 5900: {}, 6379: {},
	9092: {}, 9200: {}, 9300: {}, 11211: {}, 27017: {}, 27018: {},
}

var httpPorts = map[int]struct{}{
	80: {}, 81: {}, 443: {}, 1313: {}, 3000: {}, 3001: {}, 3002: {},
	4000: {}, 4173: {}, 4200: {}, 4321: {}, 5000: {}, 5001: {},
	5173: {}, 5174: {}, 8000: {}, 8001: {}, 8080: {}, 8081: {}, 8088: {},
	8443: {}, 8787: {}, 8888: {}, 9000: {}, 9001: {}, 9090: {}, 9443: {},
}

// browseURL opens a URL in the host browser. Tests replace it.
var browseURL = func(raw string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", raw)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", raw)
	default:
		cmd = exec.Command("xdg-open", raw)
	}
	return cmd.Start()
}

// listFwdMaps returns sidecar host|container pairs owned by this workspace.
// Tests replace it so layout never needs Docker.
var listFwdMaps = func(ws string) []portPair {
	out, err := exec.Command("docker", "ps",
		"--filter", "label=dc.forward.owner=dc-cli",
		"--format", `{{.Label "dc.forward.workspace"}}{{"\t"}}{{.Label "dc.forward.host"}}{{"\t"}}{{.Label "dc.forward.container"}}`,
	).Output()
	if err != nil {
		return nil
	}
	var pairs []portPair
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 3 || !sameWorkspace(cols[0], ws) {
			continue
		}
		h, err1 := strconv.Atoi(cols[1])
		c, err2 := strconv.Atoi(cols[2])
		if err1 != nil || err2 != nil {
			continue
		}
		pairs = append(pairs, portPair{Host: h, Container: c})
	}
	return pairs
}

// listStackPorts reads published host bindings from compose siblings
// (nginx / mitm / app). Tests replace it so layout never needs Docker.
var listStackPorts = func(stack []stackSvc) []portPair {
	if len(stack) == 0 {
		return nil
	}
	out, err := exec.Command("docker", "ps", "--format", "{{.ID}}\t{{.Ports}}").Output()
	if err != nil {
		return nil
	}
	var pairs []portPair
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		id, ports, ok := strings.Cut(line, "\t")
		if !ok || !stackHasID(stack, id) {
			continue
		}
		pairs = append(pairs, parsePublishedPairs(ports)...)
	}
	return pairs
}

func stackHasID(stack []stackSvc, id string) bool {
	for _, s := range stack {
		if s.ID == "" || id == "" {
			continue
		}
		if s.ID == id || strings.HasPrefix(s.ID, id) || strings.HasPrefix(id, s.ID) {
			return true
		}
	}
	return false
}

func sameWorkspace(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	return err1 == nil && err2 == nil && ra == rb
}

func parsePublishedPairs(ports string) []portPair {
	var pairs []portPair
	for _, m := range publishedPair.FindAllStringSubmatch(ports, -1) {
		if len(m) < 3 {
			continue
		}
		if len(m) >= 4 && strings.EqualFold(m[3], "udp") {
			continue
		}
		h, err1 := strconv.Atoi(m[1])
		c, err2 := strconv.Atoi(m[2])
		if err1 != nil || err2 != nil {
			continue
		}
		if h < 1 || h > 65535 || c < 1 || c > 65535 {
			continue
		}
		pairs = append(pairs, portPair{Host: h, Container: c})
	}
	return pairs
}

func isInfraPort(p int) bool {
	_, ok := infraPorts[p]
	return ok
}

func isHTTPPort(p int) bool {
	_, ok := httpPorts[p]
	return ok
}

func isWebPair(p portPair) bool {
	if isInfraPort(p.Host) || isInfraPort(p.Container) {
		return false
	}
	return isHTTPPort(p.Host) || isHTTPPort(p.Container)
}

func mergePairs(groups ...[]portPair) []portPair {
	seen := map[int]portPair{}
	for _, group := range groups {
		for _, p := range group {
			if p.Host < 1 {
				continue
			}
			if prev, ok := seen[p.Host]; ok {
				// Prefer a pair that looks like a website (9001:80 over sidecar 9001:9001).
				if isWebPair(p) && !isWebPair(prev) {
					seen[p.Host] = p
				}
				continue
			}
			seen[p.Host] = p
		}
	}
	out := make([]portPair, 0, len(seen))
	for _, p := range seen {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Host < out[j].Host })
	return out
}

func linkForPair(p portPair) webLink {
	scheme := "http"
	if p.Host == 443 || p.Host == 8443 || p.Container == 443 || p.Container == 8443 {
		scheme = "https"
	}
	url := scheme + "://127.0.0.1:" + strconv.Itoa(p.Host)
	return webLink{URL: url, Label: url}
}

func webLinksFrom(pairs ...[]portPair) []webLink {
	var links []webLink
	for _, p := range mergePairs(pairs...) {
		if !isWebPair(p) {
			continue
		}
		links = append(links, linkForPair(p))
	}
	return links
}

func (m model) webLinks() []webLink {
	var published []portPair
	if len(m.rows) > 0 {
		published = parsePublishedPairs(m.rows[0].Ports)
	}
	return webLinksFrom(published, m.fwdMaps)
}

func (m model) openWebURL(raw string) (tea.Model, tea.Cmd) {
	if raw == "" {
		return m, nil
	}
	if err := browseURL(raw); err != nil {
		return m.withErr("open " + raw + ": " + err.Error()), nil
	}
	return m.withStatus("opened " + raw), nil
}

func (m model) openWebIndex(i int) (tea.Model, tea.Cmd) {
	links := m.webLinks()
	if i < 0 || i >= len(links) {
		return m, nil
	}
	return m.openWebURL(links[i].URL)
}
