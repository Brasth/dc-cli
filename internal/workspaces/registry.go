// Package workspaces is the recent-workspace registry behind the board's
// `w` picker. It records folders the board opened; it never scans disks,
// starts containers, or edits projects.
package workspaces

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Canvilled/dc-cli/internal/localstate"
)

const (
	SchemaVersion = 1
	// MaxRecent caps non-favorite entries. Favorites are unlimited.
	MaxRecent = 30
	fileName  = "workspaces.json"
)

// Entry is one remembered workspace. Path is canonical (absolute, symlinks
// resolved when the folder exists).
type Entry struct {
	Path     string    `json:"path"`
	Favorite bool      `json:"favorite,omitempty"`
	LastOpen time.Time `json:"lastOpen"`
}

type fileDoc struct {
	SchemaVersion int     `json:"schemaVersion"`
	Workspaces    []Entry `json:"workspaces"`
}

// MalformedError: the file exists but is not a registry we can read. It is
// left untouched and updates are skipped until the user fixes or removes it.
type MalformedError struct {
	Path string
	Err  error
}

func (e *MalformedError) Error() string {
	return fmt.Sprintf("%s is malformed (%v); left untouched, recent list not saved", e.Path, e.Err)
}

func (e *MalformedError) Unwrap() error { return e.Err }

// ErrNewerSchema: written by a newer dc-cli; read-only here.
var ErrNewerSchema = errors.New("workspaces.json is from a newer dc-cli; recent list is read-only")

// Store is the registry file plus its lock settings.
type Store struct {
	Path     string
	LockWait time.Duration
	Now      func() time.Time
}

// DefaultPath is ${XDG_STATE_HOME:-~/.local/state}/dc-cli/workspaces.json.
func DefaultPath() (string, error) {
	dir, err := localstate.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// Open returns the store at DefaultPath.
func Open() (*Store, error) {
	p, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return &Store{Path: p}, nil
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Store) lockWait() time.Duration {
	if s.LockWait > 0 {
		return s.LockWait
	}
	return 2 * time.Second
}

// Canonical is the registry identity of a folder: absolute, cleaned, and
// symlinks resolved when it exists (so two spellings dedup to one entry).
func Canonical(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(real), nil
	}
	return filepath.Clean(abs), nil
}

// Exists reports whether the folder is still on disk.
func Exists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// Load reads the registry. Missing file → empty. Malformed → empty plus a
// *MalformedError (nonfatal). Newer schema → its entries plus ErrNewerSchema.
func (s *Store) Load() ([]Entry, error) {
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parse(s.Path, b)
}

func parse(path string, b []byte) ([]Entry, error) {
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, &MalformedError{Path: path, Err: errors.New("empty file")}
	}
	var doc fileDoc
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&doc); err != nil {
		return nil, &MalformedError{Path: path, Err: err}
	}
	if doc.SchemaVersion > SchemaVersion {
		return Sorted(doc.Workspaces), ErrNewerSchema
	}
	if doc.SchemaVersion != SchemaVersion {
		return nil, &MalformedError{Path: path, Err: fmt.Errorf("schemaVersion %d", doc.SchemaVersion)}
	}
	for _, e := range doc.Workspaces {
		if e.Path == "" || !filepath.IsAbs(e.Path) {
			return nil, &MalformedError{Path: path, Err: fmt.Errorf("entry path %q is not absolute", e.Path)}
		}
	}
	return Sorted(doc.Workspaces), nil
}

// Record marks path as opened now (canonical, deduped).
func (s *Store) Record(path string) ([]Entry, error) {
	key, err := Canonical(path)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	return s.update(func(es []Entry) []Entry {
		for i := range es {
			if es[i].Path == key {
				es[i].LastOpen = now
				return es
			}
		}
		return append(es, Entry{Path: key, LastOpen: now})
	})
}

// SetFavorite pins or unpins an existing entry.
func (s *Store) SetFavorite(path string, fav bool) ([]Entry, error) {
	key, err := Canonical(path)
	if err != nil {
		return nil, err
	}
	return s.update(func(es []Entry) []Entry {
		for i := range es {
			if es[i].Path == key || es[i].Path == path {
				es[i].Favorite = fav
			}
		}
		return es
	})
}

// Forget drops the entry. Registry state only; the folder is not touched.
func (s *Store) Forget(path string) ([]Entry, error) {
	key, _ := Canonical(path)
	return s.update(func(es []Entry) []Entry {
		out := es[:0]
		for _, e := range es {
			if e.Path == path || e.Path == key {
				continue
			}
			out = append(out, e)
		}
		return out
	})
}

// update is read-modify-write under the cross-process lock, then an atomic
// replace. A malformed or newer file is never overwritten.
func (s *Store) update(fn func([]Entry) []Entry) ([]Entry, error) {
	var result []Entry
	err := localstate.WithLock(s.Path, s.lockWait(), func() error {
		es, err := s.Load()
		if err != nil {
			return err
		}
		es = retain(dedup(fn(es)))
		data, err := json.MarshalIndent(fileDoc{SchemaVersion: SchemaVersion, Workspaces: es}, "", "  ")
		if err != nil {
			return err
		}
		if err := localstate.AtomicWrite(s.Path, append(data, '\n'), 0o600); err != nil {
			return err
		}
		result = es
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// dedup merges entries with the same path (keeps newest lastOpen, any favorite).
func dedup(es []Entry) []Entry {
	idx := map[string]int{}
	var out []Entry
	for _, e := range es {
		if i, ok := idx[e.Path]; ok {
			if e.LastOpen.After(out[i].LastOpen) {
				out[i].LastOpen = e.LastOpen
			}
			out[i].Favorite = out[i].Favorite || e.Favorite
			continue
		}
		idx[e.Path] = len(out)
		out = append(out, e)
	}
	return out
}

// retain keeps every favorite and the newest MaxRecent others.
func retain(es []Entry) []Entry {
	es = Sorted(es)
	var out []Entry
	recent := 0
	for _, e := range es {
		if !e.Favorite {
			if recent >= MaxRecent {
				continue
			}
			recent++
		}
		out = append(out, e)
	}
	return out
}

// Sorted orders favorites first, then lastOpen descending, then path.
func Sorted(es []Entry) []Entry {
	out := append([]Entry(nil), es...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Favorite != out[j].Favorite {
			return out[i].Favorite
		}
		if !out[i].LastOpen.Equal(out[j].LastOpen) {
			return out[i].LastOpen.After(out[j].LastOpen)
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// Labels names each entry by folder base name. Entries that share a base
// name get their parent path appended (home shown as ~) to tell them apart.
func Labels(es []Entry) []string {
	count := map[string]int{}
	for _, e := range es {
		count[filepath.Base(e.Path)]++
	}
	home, _ := os.UserHomeDir()
	out := make([]string, len(es))
	for i, e := range es {
		base := filepath.Base(e.Path)
		if count[base] < 2 {
			out[i] = base
			continue
		}
		out[i] = base + "  (" + tildePath(filepath.Dir(e.Path), home) + ")"
	}
	return out
}

func tildePath(p, home string) string {
	if home != "" && (p == home || strings.HasPrefix(p, home+string(filepath.Separator))) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}
