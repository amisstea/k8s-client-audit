#!/usr/bin/env bash
set -uo pipefail

ROOT_DIR=${1:-sources}

if ! command -v k8s-client-audit >/dev/null 2>&1; then
  echo "❌ ERROR: k8s-client-audit not found in PATH. Build/install it first." >&2
  exit 1
fi

if [ ! -d "$ROOT_DIR" ]; then
  echo "❌ ERROR: directory '$ROOT_DIR' does not exist" >&2
  exit 1
fi

echo "🔍 Scanning for Go modules under: $ROOT_DIR"

modules=$(find "$ROOT_DIR" -type f -name go.mod -not -path "*/vendor/*")

while IFS= read -r gomod; do
  moddir=$(dirname "$gomod")
  echo "🧩 Analyzing module: $moddir"
  pushd "$moddir" >/dev/null
  k8s-client-audit -test=false ./...
  popd >/dev/null
done <<< $modules

echo "✅ Done."
