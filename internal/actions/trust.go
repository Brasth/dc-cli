package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Canvilled/dc-cli/internal/localstate"
)

// TrustStore records which exact shared-file bytes the user approved, per
// canonical workspace: ${XDG_STATE_HOME:-~/.local/state}/dc-cli/actions-trust.json.
// Any byte change gives a new hash, which is simply not approved.
type TrustStore struct {
	Path     string
	LockWait time.Duration
}

type approval struct {
	SHA256     string    `json:"sha256"`
	ApprovedAt time.Time `json:"approvedAt"`
}

type trustDoc struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Approvals     map[string]approval `json:"approvals"`
}

// ErrTrustMalformed: the trust file is unreadable. Nothing is trusted and
// the file is left as-is.
var ErrTrustMalformed = errors.New("actions-trust.json is malformed; shared actions stay disabled")

// OpenTrust returns the store at its default path.
func OpenTrust() (*TrustStore, error) {
	dir, err := localstate.StateDir()
	if err != nil {
		return nil, err
	}
	return &TrustStore{Path: filepath.Join(dir, "actions-trust.json")}, nil
}

func (t *TrustStore) wait() time.Duration {
	if t.LockWait > 0 {
		return t.LockWait
	}
	return 2 * time.Second
}

func (t *TrustStore) load() (trustDoc, error) {
	doc := trustDoc{SchemaVersion: 1, Approvals: map[string]approval{}}
	raw, err := os.ReadFile(t.Path)
	if errors.Is(err, os.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return doc, err
	}
	var got trustDoc
	if len(bytes.TrimSpace(raw)) == 0 || json.Unmarshal(raw, &got) != nil || got.SchemaVersion != 1 {
		return doc, ErrTrustMalformed
	}
	if got.Approvals == nil {
		got.Approvals = map[string]approval{}
	}
	return got, nil
}

// Approved reports whether ws's shared file with this hash is trusted.
func (t *TrustStore) Approved(ws, hash string) (bool, error) {
	doc, err := t.load()
	if err != nil {
		return false, err
	}
	a, ok := doc.Approvals[ws]
	return ok && hash != "" && a.SHA256 == hash, nil
}

func (t *TrustStore) update(fn func(map[string]approval)) error {
	return localstate.WithLock(t.Path, t.wait(), func() error {
		doc, err := t.load()
		if err != nil {
			return err
		}
		fn(doc.Approvals)
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		return localstate.AtomicWrite(t.Path, append(data, '\n'), 0o600)
	})
}

// Approve trusts exactly hash for ws (replacing any older approval).
func (t *TrustStore) Approve(ws, hash string) error {
	if ws == "" || hash == "" {
		return fmt.Errorf("approve: empty workspace or hash")
	}
	return t.update(func(m map[string]approval) {
		m[ws] = approval{SHA256: hash, ApprovedAt: time.Now().UTC()}
	})
}

// Revoke drops ws's approval.
func (t *TrustStore) Revoke(ws string) error {
	return t.update(func(m map[string]approval) { delete(m, ws) })
}

// ErrChanged: the shared file differs from what the user reviewed.
var ErrChanged = errors.New(".dc/actions.json changed since it was shown; review it again")

// ApproveReviewed re-reads the shared file and approves only if its bytes
// still hash to what the user just reviewed (no check-then-use gap).
func (t *TrustStore) ApproveReviewed(ws, reviewedHash string) error {
	raw, err := os.ReadFile(SharedPath(ws))
	if err != nil {
		return err
	}
	if HashContent(raw) != reviewedHash {
		return ErrChanged
	}
	if _, err := Parse(SharedPath(ws), raw); err != nil {
		return err
	}
	return t.Approve(ws, reviewedHash)
}
