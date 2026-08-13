#!/usr/bin/env sh
set -eu

dist_dir=${1:-dist}
test -f "$dist_dir/SHA256SUMS.txt"
test -f "$dist_dir/release-manifest.json"

for target in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64 windows_amd64 windows_arm64; do
  os=${target%_*}
  arch=${target#*_}
  matches=$(find "$dist_dir" -maxdepth 1 -type f \( -name "agent-doctor_*_${os}_${arch}.tar.gz" -o -name "agent-doctor_*_${os}_${arch}.zip" \) | wc -l | tr -d ' ')
  test "$matches" = "1" || { echo "missing or duplicate archive for $target" >&2; exit 1; }
done

(cd "$dist_dir" && shasum -a 256 -c SHA256SUMS.txt)
node -e '
const fs = require("node:fs")
const path = require("node:path")
const root = process.argv[1]
const manifest = JSON.parse(fs.readFileSync(path.join(root, "release-manifest.json"), "utf8"))
if (manifest.schemaVersion !== 1 || !/^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(manifest.version) || manifest.assets.length !== 6) process.exit(1)
for (const asset of manifest.assets) {
  if (!fs.statSync(path.join(root, asset.filename)).isFile()) process.exit(1)
  if (!asset.url.startsWith(`https://github.com/18534516725/Agent-Doctor/releases/download/v${manifest.version}/`)) process.exit(1)
}
' "$dist_dir"
