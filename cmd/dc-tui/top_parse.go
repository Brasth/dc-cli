package main

import (
	"encoding/json"
	"strconv"
	"strings"
)

func parseStatsLine(line string) (statsBox, bool) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] != '{' {
		return statsBox{}, false
	}
	var raw struct {
		ID        string `json:"ID"`
		Container string `json:"Container"`
		Name      string `json:"Name"`
		CPUPerc   string `json:"CPUPerc"`
		MemUsage  string `json:"MemUsage"`
		NetIO     string `json:"NetIO"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return statsBox{}, false
	}
	id := raw.ID
	if id == "" {
		id = raw.Container
	}
	cpu := parsePct(raw.CPUPerc)
	used, limit := parseMemUsage(raw.MemUsage)
	rx, tx := parseNetIO(raw.NetIO)
	return statsBox{
		ID:            id,
		Name:          raw.Name,
		CPUPct:        cpu,
		MemUsedBytes:  used,
		MemLimitBytes: limit,
		NetRxBytes:    rx,
		NetTxBytes:    tx,
	}, true
}

func parsePct(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseHumanBytes(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n := 0
	for n < len(s) && (s[n] == '.' || (s[n] >= '0' && s[n] <= '9')) {
		n++
	}
	if n == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(s[:n], 64)
	if err != nil {
		return 0
	}
	unit := strings.TrimSpace(s[n:])
	mul := 1.0
	switch unit {
	case "kB", "KB":
		mul = 1000
	case "MB":
		mul = 1000 * 1000
	case "GB":
		mul = 1000 * 1000 * 1000
	case "KiB":
		mul = 1024
	case "MiB":
		mul = 1024 * 1024
	case "GiB":
		mul = 1024 * 1024 * 1024
	case "TiB":
		mul = 1024 * 1024 * 1024 * 1024
	}
	return int64(v * mul)
}

func parseMemUsage(s string) (used, limit int64) {
	left, right, ok := strings.Cut(s, "/")
	used = parseHumanBytes(left)
	if ok {
		limit = parseHumanBytes(right)
	}
	return used, limit
}

func parseNetIO(s string) (rx, tx int64) {
	left, right, ok := strings.Cut(s, "/")
	rx = parseHumanBytes(left)
	if ok {
		tx = parseHumanBytes(right)
	}
	return rx, tx
}
