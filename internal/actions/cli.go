package actions

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/Canvilled/dc-cli/internal/workspaces"
)

// Exit codes for dc-actions.
const (
	ExitOK          = 0
	ExitUnavailable = 1 // workspace / action missing, shared disabled, target down
	ExitUsage       = 2 // bad usage or bad config
)

// IO is the CLI's terminal. The child of `run` inherits these directly.
type IO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	IsTTY  bool
}

const Usage = `dc-actions — per-project commands, run inside this folder's dev container

  dc-actions list [workspace] [--json]
  dc-actions run [--] ID [workspace]   (-- before an ID that starts with -)
  dc-actions trust [workspace] [--yes]
  dc-actions untrust [workspace]
  dc-actions --help | --version

Personal:  ${XDG_CONFIG_HOME:-~/.config}/dc-cli/actions/<sha256 of folder>.json
Shared:    <workspace>/.dc/actions.json — disabled until trusted (exact bytes).
           Any change to the file disables it again.

  {"schemaVersion":1,"actions":[
    {"id":"test","label":"Run tests","argv":["go","test","./..."]},
    {"id":"psql","label":"DB shell","argv":["psql","-U","app"],"service":"db"}]}

argv runs as-is (no shell). A personal id overrides the shared one.
Runs via dc-exec --no-start: the container must already be running; nothing
is started for you. Actions run inside your containers and can change data.

Exit: 0 ok · 1 unavailable / missing / disabled · 2 bad usage or config ·
run returns the command's own status.
`

// ResolveWorkspace mirrors the board: the folder itself when it has a
// .devcontainer, else its git root, else the folder. Canonical result.
func ResolveWorkspace(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if !workspaces.Exists(abs) {
		return "", fmt.Errorf("not a directory: %s", dir)
	}
	hasDC := false
	for _, p := range []string{filepath.Join(abs, ".devcontainer", "devcontainer.json"), filepath.Join(abs, ".devcontainer.json")} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			hasDC = true
		}
	}
	if !hasDC {
		if out, err := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel").Output(); err == nil {
			if root := strings.TrimSpace(string(out)); root != "" {
				abs = root
			}
		}
	}
	return workspaces.Canonical(abs)
}

// Main runs the CLI and returns the exit code.
func Main(args []string, tio IO) int {
	if len(args) == 0 {
		fmt.Fprint(tio.Stderr, Usage)
		return ExitUsage
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "-h", "--help", "help":
		fmt.Fprint(tio.Stdout, Usage)
		return ExitOK
	case "list":
		return cmdList(rest, tio)
	case "run":
		return cmdRun(rest, tio)
	case "trust":
		return cmdTrust(rest, tio)
	case "untrust":
		return cmdUntrust(rest, tio)
	}
	fmt.Fprintf(tio.Stderr, "dc-actions: unknown command %q (try --help)\n", verb)
	return ExitUsage
}

// parseArgs splits flags from at most maxPos positionals.
func parseArgs(args []string, flags map[string]*bool, maxPos int) ([]string, error) {
	var pos []string
	for i, a := range args {
		if a == "--" {
			// End of options: everything after is positional (ids may start with -).
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			f, ok := flags[a]
			if !ok {
				return nil, fmt.Errorf("unknown flag %s", a)
			}
			*f = true
			continue
		}
		pos = append(pos, a)
	}
	if len(pos) > maxPos {
		return nil, fmt.Errorf("too many arguments")
	}
	return pos, nil
}

func usageErr(tio IO, err error) int {
	fmt.Fprintf(tio.Stderr, "dc-actions: %v (try --help)\n", err)
	return ExitUsage
}

func workspaceArg(pos []string, i int, tio IO) (string, int) {
	dir := "."
	if len(pos) > i {
		dir = pos[i]
	}
	ws, err := ResolveWorkspace(dir)
	if err != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: %v\n", err)
		return "", ExitUnavailable
	}
	return ws, ExitOK
}

func loadSet(ws string, tio IO) *Set {
	trust, err := OpenTrust()
	if err != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: trust store unavailable: %v\n", err)
	}
	set := Load(ws, trust)
	if set.TrustErr != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: %v\n", set.TrustErr)
	}
	return set
}

type jsonAction struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Argv    []string `json:"argv"`
	Service string   `json:"service"`
	Source  Source   `json:"source"`
	Enabled bool     `json:"enabled"`
}

type jsonList struct {
	SchemaVersion int          `json:"schemaVersion"`
	Workspace     string       `json:"workspace"`
	Actions       []jsonAction `json:"actions"`
}

func cmdList(args []string, tio IO) int {
	asJSON := false
	pos, err := parseArgs(args, map[string]*bool{"--json": &asJSON}, 1)
	if err != nil {
		return usageErr(tio, err)
	}
	ws, code := workspaceArg(pos, 0, tio)
	if code != ExitOK {
		return code
	}
	set := loadSet(ws, tio)
	code = ExitOK
	if set.PersonalErr != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: personal actions ignored: %v\n", set.PersonalErr)
		code = ExitUsage
	}
	if set.Shared.Err != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: shared actions ignored: %v\n", set.Shared.Err)
		code = ExitUsage
	} else if set.Shared.Present && !set.Shared.Trusted {
		fmt.Fprintln(tio.Stderr, "dc-actions: shared actions are disabled — review with: dc-actions trust")
	}
	entries := set.Entries()
	if asJSON {
		out := jsonList{SchemaVersion: SchemaVersion, Workspace: ws, Actions: []jsonAction{}}
		for _, e := range entries {
			out.Actions = append(out.Actions, jsonAction{ID: e.ID, Label: e.Label, Argv: e.Argv, Service: e.Service, Source: e.Source, Enabled: e.Enabled})
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(tio.Stdout, string(b))
		return code
	}
	if len(entries) == 0 {
		fmt.Fprintln(tio.Stderr, "dc-actions: no actions for "+ws)
		return code
	}
	tw := tabwriter.NewWriter(tio.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSOURCE\tSTATE\tTARGET\tLABEL\tARGV")
	for _, e := range entries {
		state := "enabled"
		if !e.Enabled {
			state = "disabled"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", DisplayText(e.ID), e.Source, state, Target(e.Action), DisplayText(e.Label), FormatArgv(e.Argv))
	}
	_ = tw.Flush()
	return code
}

func cmdRun(args []string, tio IO) int {
	pos, err := parseArgs(args, map[string]*bool{}, 2)
	if err != nil {
		return usageErr(tio, err)
	}
	if len(pos) < 1 {
		return usageErr(tio, errors.New("run needs an action ID"))
	}
	ws, code := workspaceArg(pos, 1, tio)
	if code != ExitOK {
		return code
	}
	// One read: the entry run below is exactly what was validated and trusted.
	set := loadSet(ws, tio)
	e, err := set.Find(pos[0])
	switch {
	case err == nil:
	case IsConfigError(err):
		fmt.Fprintf(tio.Stderr, "dc-actions: %v\n", err)
		return ExitUsage
	case errors.Is(err, ErrDisabled):
		fmt.Fprintf(tio.Stderr, "dc-actions: %s: %v — review with: dc-actions trust %s\n", pos[0], err, ws)
		return ExitUnavailable
	case errors.Is(err, ErrNotFound):
		fmt.Fprintf(tio.Stderr, "dc-actions: %s: %v in %s (dc-actions list)\n", pos[0], err, ws)
		return ExitUnavailable
	default:
		fmt.Fprintf(tio.Stderr, "dc-actions: %v\n", err)
		return ExitUnavailable
	}
	return runEntry(ws, e, tio)
}

// runEntry runs one action in the foreground. Ctrl+C from the terminal
// reaches the child (same process group); we only wait and pass on its status.
func runEntry(ws string, e Entry, tio IO) int {
	fmt.Fprintf(tio.Stderr, "dc-actions: %s (%s) → %s: %s\n", e.ID, e.Source, Target(e.Action), FormatArgv(e.Argv))
	cmd := Command(ws, e.Action)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tio.Stdin, tio.Stdout, tio.Stderr
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigs)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: cannot start dc-exec: %v\n", err)
		return ExitUnavailable
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	for {
		select {
		case s := <-sigs:
			// SIGINT already went to the child via the terminal; forward the rest.
			if s != syscall.SIGINT && cmd.Process != nil {
				_ = cmd.Process.Signal(s)
			}
		case err := <-done:
			code := ExitCode(err)
			if code != 0 {
				fmt.Fprintf(tio.Stderr, "dc-actions: %s exited %d\n", e.ID, code)
			}
			return code
		}
	}
}

// PrintReview writes every shared action (overridden ones too) with its
// exact argv and target — what trusting would enable.
func PrintReview(w io.Writer, set *Set) {
	fmt.Fprintf(w, "Shared actions in %s\nsha256 %s\n\n", set.Shared.Path, set.Shared.Hash)
	for _, a := range set.Shared.Actions {
		note := ""
		if set.Overridden(a.ID) {
			note = "  (overridden by your personal action)"
		}
		fmt.Fprintf(w, "  %s — %s%s\n    target: %s\n    argv:   %s\n", DisplayText(a.ID), DisplayText(a.Label), note, Target(a), FormatArgv(a.Argv))
	}
	fmt.Fprintln(w, "\nThese commands run inside your containers and can modify their data.")
}

func cmdTrust(args []string, tio IO) int {
	yes := false
	pos, err := parseArgs(args, map[string]*bool{"--yes": &yes, "-y": &yes}, 1)
	if err != nil {
		return usageErr(tio, err)
	}
	ws, code := workspaceArg(pos, 0, tio)
	if code != ExitOK {
		return code
	}
	set := loadSet(ws, tio)
	if !set.Shared.Present {
		fmt.Fprintf(tio.Stderr, "dc-actions: no %s\n", SharedPath(ws))
		return ExitUnavailable
	}
	if set.Shared.Err != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: %v\n", set.Shared.Err)
		return ExitUsage
	}
	PrintReview(tio.Stdout, set)
	if set.Shared.Trusted {
		fmt.Fprintln(tio.Stderr, "dc-actions: already trusted")
		return ExitOK
	}
	if !yes {
		if !tio.IsTTY {
			fmt.Fprintln(tio.Stderr, "dc-actions: not a terminal — re-run with --yes after reading the list above")
			return ExitUsage
		}
		fmt.Fprint(tio.Stderr, "Enable these shared actions? [y/N] ")
		line, _ := bufio.NewReader(tio.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
			fmt.Fprintln(tio.Stderr, "dc-actions: not trusted")
			return ExitUnavailable
		}
	}
	trust, err := OpenTrust()
	if err != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: %v\n", err)
		return ExitUnavailable
	}
	if err := trust.ApproveReviewed(ws, set.Shared.Hash); err != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: %v\n", err)
		if IsConfigError(err) {
			return ExitUsage
		}
		return ExitUnavailable
	}
	fmt.Fprintln(tio.Stderr, "dc-actions: shared actions enabled for "+ws)
	return ExitOK
}

func cmdUntrust(args []string, tio IO) int {
	pos, err := parseArgs(args, map[string]*bool{}, 1)
	if err != nil {
		return usageErr(tio, err)
	}
	ws, code := workspaceArg(pos, 0, tio)
	if code != ExitOK {
		return code
	}
	trust, err := OpenTrust()
	if err == nil {
		err = trust.Revoke(ws)
	}
	if err != nil {
		fmt.Fprintf(tio.Stderr, "dc-actions: %v\n", err)
		return ExitUnavailable
	}
	fmt.Fprintln(tio.Stderr, "dc-actions: shared actions disabled for "+ws)
	return ExitOK
}
