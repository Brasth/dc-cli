package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Membership for the timeline. Event labels alone never prove anything:
//   - known ids come from a successful discovery of this context (shell side,
//     fresh reads: labeled app + compose siblings, sidecars dropped) and are
//     retained for the session so a later destroy still attributes;
//   - an unknown id gets one read-only `docker inspect` (pinned engine,
//     5s probe budget) and must prove itself the same way the shell does:
//       devcontainer: devcontainer.local_folder is this workspace, or its
//                     compose project belongs to a verified app of this
//                     workspace;
//       compose-kind: compose working_dir / config_files prove this folder
//                     and the project is the verified one;
//   - dc.forward.for sidecars are always excluded.
// An inspect that cannot decide yet (project not verified) waits for the next
// discovery that started after the event; still unknown then → dropped.

const (
	labelFolder      = "devcontainer.local_folder"
	labelProject     = "com.docker.compose.project"
	labelService     = "com.docker.compose.service"
	labelWorkdir     = "com.docker.compose.project.working_dir"
	labelConfigFiles = "com.docker.compose.project.config_files"
	labelForwardFor  = "dc.forward.for"
)

// activityInspectBatch bounds ids per inspect call.
const activityInspectBatch = 16

type memberInfo struct {
	name    string
	service string
	seen    int64 // last time this id was verified (bounding keeps the newest)
}

// inspectDoc is the slice of `docker inspect` the timeline reads.
type inspectDoc struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

// runPinnedDocker runs one read-only docker call whose engine is chosen only
// by args (--context / --host): inherited DOCKER_HOST / DOCKER_CONTEXT are
// dropped. Own process group, killed on cancel. Tests replace it.
var runPinnedDocker = func(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	setProbeProcAttr(cmd)
	pinEngineEnv(cmd)
	return cmd.Output()
}

type memberDecision int

// The zero value is "no verdict yet"; a missing map entry must never read
// as a rejection.
const (
	memberUndecided memberDecision = iota
	memberReject
	memberAccept
	memberAwait
)

// inspectContainers runs one read-only inspect pinned to engine. Missing ids
// (already removed) are simply absent from the result.
func inspectContainers(ctx context.Context, engine string, ids []string) (map[string]inspectDoc, error) {
	args, ok := engineArgs(engine)
	if !ok {
		return nil, errEngineUnknown
	}
	args = append(args, "inspect", "--type", "container")
	args = append(args, ids...)
	out, err := probeVia(ctx, "docker", func(pctx context.Context) ([]byte, error) {
		return runPinnedDocker(pctx, args...)
	})
	docs := map[string]inspectDoc{}
	var list []inspectDoc
	if jerr := json.Unmarshal(out, &list); jerr == nil {
		for _, d := range list {
			if validContainerID(d.ID) {
				docs[strings.ToLower(d.ID)] = d
			}
		}
	}
	if err != nil {
		// Exit 1 with "No such container" for some ids still prints the rest.
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return nil, err
		}
	}
	return docs, nil
}

// decideMember applies the ownership contract to a fresh inspect.
func decideMember(ws string, composeKind bool, d inspectDoc, projects map[string]bool) (memberDecision, string) {
	l := d.Config.Labels
	if l[labelForwardFor] != "" {
		return memberReject, ""
	}
	proj := l[labelProject]
	if composeKind {
		if !composeProves(l[labelWorkdir], l[labelConfigFiles], ws) {
			return memberReject, ""
		}
		if proj == "" {
			return memberReject, ""
		}
		if projects[proj] {
			return memberAccept, ""
		}
		return memberAwait, ""
	}
	if f := l[labelFolder]; f != "" && sameClaimant(f, ws) {
		// The labeled app itself; its compose project is now verified.
		return memberAccept, proj
	}
	if proj != "" && projects[proj] {
		return memberAccept, ""
	}
	if proj != "" {
		return memberAwait, ""
	}
	return memberReject, ""
}

func docInfo(d inspectDoc) memberInfo {
	return memberInfo{
		name:    safeLabel(strings.TrimPrefix(d.Name, "/"), 64),
		service: safeLabel(d.Config.Labels[labelService], 64),
	}
}

// claimantRoot: compose working_dir is often <project>/.devcontainer.
func claimantRoot(p string) string {
	p = strings.TrimRight(p, "/")
	if filepath.Base(p) == ".devcontainer" {
		p = filepath.Dir(p)
	}
	return p
}

// sameClaimant mirrors dc_same_workspace: equal after .devcontainer
// trimming, or the same physical directory.
func sameClaimant(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return sameWorkspace(claimantRoot(a), claimantRoot(b))
}

// composeProves mirrors dc_compose_proves_fields.
func composeProves(workdir, files, ws string) bool {
	if st, err := os.Stat(ws); err != nil || !st.IsDir() {
		return false
	}
	if workdir != "" && sameClaimant(workdir, ws) {
		return true
	}
	for _, f := range strings.Split(files, ",") {
		if f == "" {
			continue
		}
		parent := filepath.Dir(f)
		if st, err := os.Stat(parent); err == nil && st.IsDir() && sameClaimant(parent, ws) {
			return true
		}
	}
	return false
}

// lookupKnown matches full ids only. Short snapshot ids are normalized to
// full ids by inspect before they count (see normalizeActivityIDs).
func lookupKnown(known map[string]memberInfo, id string) (memberInfo, bool) {
	info, ok := known[id]
	return info, ok
}

// isFullID: docker's 64-hex container id.
func isFullID(id string) bool {
	return len(id) == 64 && validContainerID(id)
}

// snapshotMembers is the verified membership of the current discovery
// snapshot: stack rows (sidecars already dropped by the shell) and, for
// devcontainer folders, the labeled app rows. Compose-kind rows may include
// sidecars, so only their project names are used.
func (m model) snapshotMembers() (map[string]memberInfo, map[string]bool) {
	known := map[string]memberInfo{}
	projects := map[string]bool{}
	for _, s := range m.stack {
		if s.ID == "" {
			continue
		}
		known[strings.ToLower(s.ID)] = memberInfo{name: safeLabel(strings.TrimPrefix(s.Name, "/"), 64), service: safeLabel(s.Service, 64)}
	}
	for _, r := range m.rows {
		if r.Compose != "" {
			projects[r.Compose] = true
		}
		if r.ID == "" || m.isComposeKind() {
			continue
		}
		id := strings.ToLower(r.ID)
		if _, ok := known[id]; !ok {
			known[id] = memberInfo{name: safeLabel(strings.TrimPrefix(r.Name, "/"), 64)}
		}
	}
	return known, projects
}
