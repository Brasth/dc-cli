// Command dc-actions lists, trusts and runs per-project actions.
// See internal/actions for the file formats and trust model.
package main

import (
	"fmt"
	"os"

	"github.com/Canvilled/dc-cli/internal/actions"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-V") {
		fmt.Println("dc-actions", cliVersion())
		return 0
	}
	tio := actions.IO{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		IsTTY:  isTerminal(os.Stdin) && isTerminal(os.Stderr),
	}
	return actions.Main(args, tio)
}

// isTerminal: character device (a TTY), not a pipe or file.
func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
