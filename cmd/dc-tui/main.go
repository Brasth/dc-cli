package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	workspace := "."
	fleet := false
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch a {
		case "-h", "--help":
			fmt.Print(helpText)
			os.Exit(0)
		case "--version", "-V":
			fmt.Println("dc-tui", cliVersion())
			os.Exit(0)
		case "--all":
			fleet = true
		case "-":
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", a)
			os.Exit(2)
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", a)
				os.Exit(2)
			}
			workspace = a
		}
	}
	ws, err := resolveWorkspace(workspace)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	m := model{
		workspace:  ws,
		fleet:      fleet,
		hasConfig:  hasDevcontainer(ws),
		editor:     pickEditor(),
		hoverStack: -1,
		splashOn:   os.Getenv("DC_TUI_NO_SPLASH") == "",
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		_ = exec.Command("stty", "sane").Run()
		if !benignExecErr(err) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

const helpText = `dc-tui — this folder's devcontainer (click buttons or keys)

  dc-tui [workspace]     this folder (cwd, then git root)
  dc-tui --all           every labeled container (fleet)
  dc-tui --version
  dc-tui --help

Logo splash on start (any key skips). DC_TUI_NO_SPLASH=1 to skip.

Primary: start (u)  shell (e)  stop (s)
Meta:    open (o)  attach (a)  ports (p)  logs (l)  top (t)  nets (n)  db (b)  files (m)  fleet (f)  more (?)  quit (q)
Danger:  rm (x)    asks y/n before dc-down --rm
Urls:    click or 1-9  open a published website (http/https) in the browser
Rows:    j/k or arrows, enter (fleet = open folder, stack = exec)
Disk:    d  (stays in the board). Reclaim is CLI-only: dc-prune --yes
Top:     t  live CPU/RAM for this folder (stays in the board). Fleet refuses.
Nets:    n  this folder's declared compose nets (stays in the board). y creates missing externals then start. Fleet refuses.

Shell / start / files leave the board and come back. Logs, top, and nets stay on the board (q back).
A normal exit is not a crash.

Open vs attach: open = edit files on the Mac/Linux host.
Attach = VS Code Remote URI (a / dc-open --attach).
Zed attaches itself (Project: Open Remote → Connect Dev Container). Sublime cannot.

start is refused if this folder has no .devcontainer.
Use CLI dc-up --ports only if you accept REPLACE of project config.
`

func hasDevcontainer(dir string) bool {
	for _, p := range []string{
		filepath.Join(dir, ".devcontainer", "devcontainer.json"),
		filepath.Join(dir, ".devcontainer.json"),
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func resolveWorkspace(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("not a directory: %s", dir)
	}
	if hasDevcontainer(abs) {
		return abs, nil
	}
	cmd := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return abs, nil
	}
	root := strings.TrimSpace(string(out))
	if root != "" && root != abs {
		return root, nil
	}
	return abs, nil
}

func editorBin(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	var cands []string
	switch name {
	case "zed":
		cands = []string{
			"/Applications/Zed.app/Contents/MacOS/cli",
			filepath.Join(home, "Applications/Zed.app/Contents/MacOS/cli"),
			"/Applications/Zed.app/Contents/MacOS/zed",
		}
	case "code":
		cands = []string{
			"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			filepath.Join(home, "Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"),
		}
	case "subl":
		cands = []string{
			"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl",
			filepath.Join(home, "Applications/Sublime Text.app/Contents/SharedSupport/bin/subl"),
		}
	}
	for _, p := range cands {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func pickEditor() string {
	if e := os.Getenv("DC_EDITOR"); e != "" {
		if editorBin(e) != "" {
			return e
		}
		return e + " (missing)"
	}
	for _, e := range []string{"zed", "code", "subl"} {
		if editorBin(e) != "" {
			return e
		}
	}
	return "(none — install Zed / VS Code / Sublime)"
}
