# Shared compatibility-floor constants. Empty until DC_CLI_FLOOR_QUAL is recorded.
# shellcheck shell=bash
# Override in tests via the environment; do not invent a floor from registry latest.

: "${DC_DEVCONTAINER_MIN_VERSION:=}"
: "${DC_DEVCONTAINER_NPM_VERSION:=}"
# Candidate evidence only (2026-08-16): @devcontainers/cli@0.88.0 engines.node >=20.0.0
: "${DC_DEVCONTAINER_CANDIDATE_VERSION:=0.88.0}"
: "${DC_NPM_NODE_MIN_MAJOR:=}"
