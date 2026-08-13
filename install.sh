#!/usr/bin/env sh
set -eu

version=${AGENT_DOCTOR_VERSION:-1.0.0}
repo=https://github.com/18534516725/Agent-Doctor/releases/download/v${version}
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in x86_64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; esac
case "$os" in darwin|linux) ;; *) echo "Use install.ps1 on Windows." >&2; exit 1 ;; esac
archive=agent-doctor_${version}_${os}_${arch}.tar.gz
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT INT TERM
curl --fail --location --proto '=https' --tlsv1.2 "$repo/$archive" -o "$temporary/$archive"
curl --fail --location --proto '=https' --tlsv1.2 "$repo/SHA256SUMS.txt" -o "$temporary/SHA256SUMS.txt"
(cd "$temporary" && grep " $archive\$" SHA256SUMS.txt | shasum -a 256 -c -)
tar -xzf "$temporary/$archive" -C "$temporary"
install -d "$HOME/.local/bin"
install -m 0755 "$temporary/agent-doctor" "$HOME/.local/bin/agent-doctor"
"$HOME/.local/bin/agent-doctor" setup --yes --json
echo "Installed. Start with: agent-doctor start --no-open"
