package main

import (
	"strings"
	"testing"
)

func TestParsePublishedPairsDedupAndSkipUDP(t *testing.T) {
	got := parsePublishedPairs("0.0.0.0:9001->80/tcp, [::]:9001->80/tcp, 0.0.0.0:3306->3306/udp, 127.0.0.1:3000->3000/tcp")
	if len(got) != 3 {
		t.Fatalf("pairs=%v", got)
	}
	merged := mergePairs(got)
	if len(merged) != 2 {
		t.Fatalf("dedup host 9001: %v", merged)
	}
}

func TestWebLinksSkipInfraKeepWebsite(t *testing.T) {
	links := webLinksFrom(parsePublishedPairs("0.0.0.0:3000->3000/tcp, 0.0.0.0:5432->5432/tcp, 0.0.0.0:9001->80/tcp, 0.0.0.0:6379->6379/tcp"))
	if len(links) != 2 {
		t.Fatalf("want 3000 and 9001, got %v", links)
	}
	if links[0].URL != "http://127.0.0.1:3000" {
		t.Fatalf("first=%s", links[0].URL)
	}
	if links[1].URL != "http://127.0.0.1:9001" {
		t.Fatalf("second=%s", links[1].URL)
	}
}

func TestWebLinksHTTPSAndSidecarAsymmetric(t *testing.T) {
	// Sidecar docker Ports is host:host; label pair keeps 8443:443.
	links := webLinksFrom(
		[]portPair{{Host: 8443, Container: 8443}},
		[]portPair{{Host: 8443, Container: 443}},
	)
	if len(links) != 1 || links[0].URL != "https://127.0.0.1:8443" {
		t.Fatalf("want https on 8443, got %v", links)
	}
}

func TestOpenWebURLSetsStatus(t *testing.T) {
	old := browseURL
	t.Cleanup(func() { browseURL = old })
	var saw string
	browseURL = func(raw string) error {
		saw = raw
		return nil
	}
	m := model{
		workspace:  "/tmp/app",
		hoverStack: -1,
		rows: []container{{
			Status: "running",
			Ports:  "0.0.0.0:9001->80/tcp, 0.0.0.0:3306->3306/tcp",
		}},
	}
	got, cmd := m.handleKey("1")
	if cmd != nil {
		t.Fatal("open url must stay on the board")
	}
	mm := got.(model)
	if saw != "http://127.0.0.1:9001" {
		t.Fatalf("opened %q", saw)
	}
	if mm.status != "opened http://127.0.0.1:9001" {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestClickURLTile(t *testing.T) {
	old := browseURL
	t.Cleanup(func() { browseURL = old })
	var saw string
	browseURL = func(raw string) error {
		saw = raw
		return nil
	}
	m := model{
		workspace:  "/tmp/app",
		hasConfig:  true,
		width:      80,
		hoverStack: -1,
		editor:     "zed",
		rows:       []container{{Status: "running", Ports: "0.0.0.0:5173->5173/tcp"}},
	}
	_, buttons, _ := m.layout()
	var tile button
	for _, b := range buttons {
		if strings.HasPrefix(b.key, "url:") {
			tile = b
			break
		}
	}
	if tile.key == "" {
		t.Fatal("missing url tile")
	}
	got, _ := m.handleClick((tile.x0+tile.x1)/2, tile.y0)
	mm := got.(model)
	if saw != "http://127.0.0.1:5173" {
		t.Fatalf("opened %q", saw)
	}
	if !strings.Contains(mm.status, "5173") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestStackHasIDPrefix(t *testing.T) {
	stack := []stackSvc{{ID: "203467d304351234abcd"}}
	if !stackHasID(stack, "203467d30435") {
		t.Fatal("short docker ps id should match full inspect id")
	}
	if stackHasID(stack, "deadbeef") {
		t.Fatal("foreign id matched")
	}
}

func TestStartHitboxUnchangedWithURLRow(t *testing.T) {
	base := model{workspace: "/tmp/app", hasConfig: true, width: 80, hoverStack: -1, editor: "zed"}
	with := base
	with.rows = []container{{Status: "running", Ports: "0.0.0.0:9001->80/tcp"}}
	_, lb, _ := with.layout()
	_, sb, _ := base.layout()
	ys := func(bs []button) int {
		for _, b := range bs {
			if b.key == "u" {
				return b.y0
			}
		}
		return -1
	}
	if ys(lb) != ys(sb) {
		t.Fatalf("url row moved start hitbox: with=%d without=%d", ys(lb), ys(sb))
	}
}
