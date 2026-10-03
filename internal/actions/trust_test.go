package actions

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const sharedBody = `{"schemaVersion":1,"actions":[{"id":"lint","label":"Lint","argv":["make","lint"]}]}`

func TestTrustExactBytesAndRevokeOnChange(t *testing.T) {
	ws := isolate(t)
	writeShared(t, ws, sharedBody)
	trust, err := OpenTrust()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(trust.Path) != "actions-trust.json" {
		t.Fatalf("path=%s", trust.Path)
	}
	if set := Load(ws, trust); set.Shared.Trusted {
		t.Fatal("never trusted on open")
	}
	set := Load(ws, trust)
	if err := trust.ApproveReviewed(ws, set.Shared.Hash); err != nil {
		t.Fatal(err)
	}
	set = Load(ws, trust)
	if e, err := set.Find("lint"); err != nil || !e.Enabled {
		t.Fatalf("trusted shared must run: %v", err)
	}
	// One byte changes → disabled again.
	writeShared(t, ws, sharedBody+" ")
	set = Load(ws, trust)
	if set.Shared.Trusted {
		t.Fatal("changed content must revoke trust")
	}
	if _, err := set.Find("lint"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("err=%v", err)
	}
}

func TestApproveReviewedRefusesChangedFile(t *testing.T) {
	ws := isolate(t)
	writeShared(t, ws, sharedBody)
	trust, _ := OpenTrust()
	reviewed := Load(ws, trust).Shared.Hash
	writeShared(t, ws, `{"schemaVersion":1,"actions":[{"id":"lint","label":"Lint","argv":["curl","evil"]}]}`)
	if err := trust.ApproveReviewed(ws, reviewed); !errors.Is(err, ErrChanged) {
		t.Fatalf("want ErrChanged, got %v", err)
	}
	if Load(ws, trust).Shared.Trusted {
		t.Fatal("nothing may be trusted after a changed-file refusal")
	}
}

func TestTrustIsPerWorkspace(t *testing.T) {
	ws := isolate(t)
	other, _ := filepath.EvalSymlinks(t.TempDir())
	writeShared(t, ws, sharedBody)
	writeShared(t, other, sharedBody)
	trust, _ := OpenTrust()
	_ = trust.Approve(ws, HashContent([]byte(sharedBody)))
	if !Load(ws, trust).Shared.Trusted || Load(other, trust).Shared.Trusted {
		t.Fatal("approval must be bound to the canonical workspace")
	}
	if err := trust.Revoke(ws); err != nil {
		t.Fatal(err)
	}
	if Load(ws, trust).Shared.Trusted {
		t.Fatal("revoke must disable")
	}
}

func TestMalformedTrustFileDisablesAndIsPreserved(t *testing.T) {
	ws := isolate(t)
	writeShared(t, ws, sharedBody)
	trust, _ := OpenTrust()
	_ = os.MkdirAll(filepath.Dir(trust.Path), 0o755)
	_ = os.WriteFile(trust.Path, []byte("{nope"), 0o600)
	set := Load(ws, trust)
	if set.Shared.Trusted || !errors.Is(set.TrustErr, ErrTrustMalformed) {
		t.Fatalf("trusted=%v err=%v", set.Shared.Trusted, set.TrustErr)
	}
	if err := trust.Approve(ws, set.Shared.Hash); !errors.Is(err, ErrTrustMalformed) {
		t.Fatalf("approve on malformed must refuse: %v", err)
	}
	b, _ := os.ReadFile(trust.Path)
	if string(b) != "{nope" {
		t.Fatal("malformed trust file must be preserved")
	}
}
