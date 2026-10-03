// Package actions is per-project commands for dc-cli: a personal file per
// workspace and an optional shared `.dc/actions.json` that stays disabled
// until the user trusts its exact bytes. The board (`c`) and the
// `dc-actions` CLI both use this package directly.
package actions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Canvilled/dc-cli/internal/localstate"
)

const SchemaVersion = 1

// Source of an action.
type Source string

const (
	SourcePersonal Source = "personal"
	SourceShared   Source = "shared"
)

// Action is one validated command. Argv runs as-is (no shell).
type Action struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Argv    []string `json:"argv"`
	Service string   `json:"service,omitempty"`
}

// rawAction keeps what encoding/json would blur: a null argv element
// (would decode as "") and service absent vs null vs "".
type rawAction struct {
	ID      string          `json:"id"`
	Label   string          `json:"label"`
	Argv    []*string       `json:"argv"`
	Service json.RawMessage `json:"service"`
}

type fileDoc struct {
	SchemaVersion *int        `json:"schemaVersion"`
	Actions       []rawAction `json:"actions"`
}

// ConfigError is an invalid actions file. The whole file is rejected.
type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string { return fmt.Sprintf("%s: %v", e.Path, e.Err) }
func (e *ConfigError) Unwrap() error { return e.Err }

// Parse validates a whole file: schemaVersion 1, an actions array, unique
// non-empty ids, non-empty labels, non-empty argv of strings with a
// non-empty argv[0], optional non-empty service. Unknown fields are errors.
// ids are opaque labels (never used as paths). NUL cannot reach exec(2), so
// a NUL in argv or service rejects the file instead of being cut silently.
func Parse(path string, data []byte) ([]Action, error) {
	fail := func(format string, a ...any) ([]Action, error) {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf(format, a...)}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var doc fileDoc
	if err := dec.Decode(&doc); err != nil {
		return fail("invalid JSON: %v", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fail("trailing data after the JSON object")
	}
	if doc.SchemaVersion == nil {
		return fail("schemaVersion is required")
	}
	if *doc.SchemaVersion != SchemaVersion {
		return fail("schemaVersion %d is not supported (want %d)", *doc.SchemaVersion, SchemaVersion)
	}
	if doc.Actions == nil {
		return fail("actions array is required")
	}
	seen := map[string]bool{}
	out := make([]Action, 0, len(doc.Actions))
	for i, a := range doc.Actions {
		where := fmt.Sprintf("actions[%d]", i)
		if a.ID == "" {
			return fail("%s: id is required", where)
		}
		if seen[a.ID] {
			return fail("%s: duplicate id %q", where, a.ID)
		}
		seen[a.ID] = true
		if a.Label == "" {
			return fail("%s (%s): label is required", where, a.ID)
		}
		if len(a.Argv) == 0 {
			return fail("%s (%s): argv must be a non-empty array of strings", where, a.ID)
		}
		argv := make([]string, len(a.Argv))
		for j, arg := range a.Argv {
			if arg == nil {
				return fail("%s (%s): argv[%d] is null (argv must be strings)", where, a.ID, j)
			}
			if strings.ContainsRune(*arg, 0) {
				return fail("%s (%s): argv[%d] contains a NUL byte", where, a.ID, j)
			}
			argv[j] = *arg
		}
		if argv[0] == "" {
			return fail("%s (%s): argv[0] (the command) must be non-empty", where, a.ID)
		}
		act := Action{ID: a.ID, Label: a.Label, Argv: argv}
		if a.Service != nil {
			var svc *string
			if err := json.Unmarshal(a.Service, &svc); err != nil || svc == nil {
				return fail("%s (%s): service must be a string (omit it to target the app)", where, a.ID)
			}
			if *svc == "" {
				return fail("%s (%s): service must be non-empty when set", where, a.ID)
			}
			if strings.ContainsRune(*svc, 0) {
				return fail("%s (%s): service contains a NUL byte", where, a.ID)
			}
			act.Service = *svc
		}
		out = append(out, act)
	}
	return out, nil
}

// HashContent is the trust identity of the shared file's raw bytes.
func HashContent(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// PersonalPath is ${XDG_CONFIG_HOME:-~/.config}/dc-cli/actions/<sha256(ws)>.json.
func PersonalPath(canonicalWS string) (string, error) {
	dir, err := localstate.ConfigDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(canonicalWS))
	return filepath.Join(dir, "actions", hex.EncodeToString(sum[:])+".json"), nil
}

// SharedPath is <workspace>/.dc/actions.json (read-only to dc-cli).
func SharedPath(ws string) string {
	return filepath.Join(ws, ".dc", "actions.json")
}

// Shared is the shared file exactly as read once: raw bytes, their hash,
// parsed actions (when valid), and whether that hash is trusted.
type Shared struct {
	Path    string
	Present bool
	Raw     []byte
	Hash    string
	Actions []Action
	Err     error
	Trusted bool
}

// Set is everything known about a workspace's actions at one read.
type Set struct {
	Workspace       string
	PersonalPath    string
	PersonalPresent bool
	Personal        []Action
	PersonalErr     error
	Shared          Shared
	TrustErr        error
}

// Entry is one runnable-or-not action after precedence.
type Entry struct {
	Action
	Source  Source
	Enabled bool
}

// Load reads both files once. ws must be canonical. Read errors are kept per
// file so a bad shared file never hides valid personal actions.
func Load(ws string, trust *TrustStore) *Set {
	s := &Set{Workspace: ws}
	if p, err := PersonalPath(ws); err != nil {
		s.PersonalErr = err
	} else {
		s.PersonalPath = p
		if raw, err := os.ReadFile(p); err == nil {
			s.PersonalPresent = true
			s.Personal, s.PersonalErr = Parse(p, raw)
		} else if !errors.Is(err, os.ErrNotExist) {
			s.PersonalErr = err
		}
	}
	sh := Shared{Path: SharedPath(ws)}
	if raw, err := os.ReadFile(sh.Path); err == nil {
		sh.Present = true
		sh.Raw = raw
		sh.Hash = HashContent(raw)
		sh.Actions, sh.Err = Parse(sh.Path, raw)
		if sh.Err == nil && trust != nil {
			sh.Trusted, s.TrustErr = trust.Approved(ws, sh.Hash)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		sh.Present = true
		sh.Err = err
	}
	s.Shared = sh
	return s
}

// Entries applies precedence: personal actions, then shared actions whose id
// no personal action uses. Shared entries are enabled only when trusted.
func (s *Set) Entries() []Entry {
	var out []Entry
	mine := map[string]bool{}
	for _, a := range s.Personal {
		mine[a.ID] = true
		out = append(out, Entry{Action: a, Source: SourcePersonal, Enabled: true})
	}
	if s.Shared.Err == nil {
		for _, a := range s.Shared.Actions {
			if mine[a.ID] {
				continue
			}
			out = append(out, Entry{Action: a, Source: SourceShared, Enabled: s.Shared.Trusted})
		}
	}
	return out
}

// Overridden reports whether a shared id is shadowed by a personal action.
func (s *Set) Overridden(id string) bool {
	for _, a := range s.Personal {
		if a.ID == id {
			return true
		}
	}
	return false
}

// Lookup errors.
var (
	ErrNotFound = errors.New("no such action")
	ErrDisabled = errors.New("shared action is disabled until you trust .dc/actions.json")
)

// Find resolves id with precedence. A broken shared file only matters when
// the id is not a personal action.
func (s *Set) Find(id string) (Entry, error) {
	if s.PersonalErr != nil {
		return Entry{}, s.PersonalErr
	}
	for _, a := range s.Personal {
		if a.ID == id {
			return Entry{Action: a, Source: SourcePersonal, Enabled: true}, nil
		}
	}
	if s.Shared.Err != nil {
		return Entry{}, s.Shared.Err
	}
	for _, a := range s.Shared.Actions {
		if a.ID == id {
			e := Entry{Action: a, Source: SourceShared, Enabled: s.Shared.Trusted}
			if !e.Enabled {
				return e, ErrDisabled
			}
			return e, nil
		}
	}
	return Entry{}, ErrNotFound
}

// IsConfigError reports a bad file (exit 2) vs. anything else.
func IsConfigError(err error) bool {
	var ce *ConfigError
	return errors.As(err, &ce)
}
