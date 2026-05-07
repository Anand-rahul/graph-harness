#!/usr/bin/env bash
# tests/smoke/p0/run.sh — Phase-0 end-to-end smoke (P0.T49).
#
# The canonical demo is the gotit spec `boilerplate/tracer-demo.yaml`. Running it
# directly here keeps `just smoke` and `just e2e` aligned, and lets CI keep
# both targets independently.
set -euo pipefail
cd "$(dirname "$0")/../../.."
go test -count=1 -run "TestE2E/boilerplate/tracer-demo" ./tests/e2e/...
