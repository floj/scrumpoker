#!/usr/bin/env bash
set -euo pipefail

projectDir=$(dirname "${BASH_SOURCE[0]}")

echo "==> Building UI..."
cd "$projectDir/ui"
pnpm install --frozen-lockfile
pnpm build

echo "==> Building backend..."
cd "$projectDir"
CGO_ENABLED=0 go build -o scrumpoker ./cmd/scrumpoker

echo "==> Done! Binary: ./scrumpoker"
