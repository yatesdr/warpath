#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

echo "Pulling latest changes..."
git pull

echo "Starting shingo-edge..."
cd shingo-edge
go run ./cmd/shingoedge "$@"
