#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

echo "Pulling latest changes..."
git pull

echo "Starting shingo-core..."
cd shingo-core
go run ./cmd/shingocore "$@"
