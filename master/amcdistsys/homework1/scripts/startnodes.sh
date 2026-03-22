#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/startnodes.sh <config_file> <first_index> <last_index>
# Example: bash scripts/startnodes.sh config.txt 0 2

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <config_file> <first_index> <last_index>" >&2
    exit 1
fi

CONFIG="$1"
FIRST="$2"
LAST="$3"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Determine binary name (platform-aware)
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    BINARY="$ROOT/bcastnode.exe"
else
    BINARY="$ROOT/bcastnode"
fi

echo "Building bcastnode..."
(cd "$ROOT" && go build -o "$(basename "$BINARY")" ./cmd/bcastnode)
echo "Build complete."

pids=()
for idx in $(seq "$FIRST" "$LAST"); do
    "$BINARY" "$CONFIG" "$idx" &
    pids+=($!)
    echo "Started node $idx (pid ${pids[-1]})"
done

echo "All nodes launched. Waiting for completion..."
for pid in "${pids[@]}"; do
    wait "$pid"
done
echo "All nodes done."
