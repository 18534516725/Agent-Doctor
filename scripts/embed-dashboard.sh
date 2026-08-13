#!/usr/bin/env sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
dashboard_dist="$repo_root/dashboard/dist"
embedded_root="$repo_root/internal/server/web"

pnpm --dir "$repo_root/dashboard" build
mkdir -p "$embedded_root/assets"
find "$embedded_root/assets" -type f -maxdepth 1 -delete
cp "$dashboard_dist/index.html" "$embedded_root/index.html"
cp "$dashboard_dist"/assets/* "$embedded_root/assets/"
