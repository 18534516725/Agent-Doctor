#!/usr/bin/env sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
install_dir=${AGENT_DOCTOR_INSTALL_DIR:-"$HOME/.local/bin"}

for required in go pnpm; do
  if ! command -v "$required" >/dev/null 2>&1; then
    echo "Agent Doctor 安装失败：缺少 $required。" >&2
    exit 1
  fi
done

echo "[1/6] 安装前端依赖"
pnpm --dir "$repo_root" install --frozen-lockfile
echo "[2/6] 构建并嵌入本地仪表盘"
"$repo_root/scripts/embed-dashboard.sh"
echo "[3/6] 运行完整测试"
(cd "$repo_root" && go test ./...)
echo "[4/6] 构建 Agent Doctor"
build_dir=$(mktemp -d)
trap 'rm -rf "$build_dir"' EXIT INT TERM
(cd "$repo_root" && go build -o "$build_dir/agent-doctor" ./cmd/agent-doctor)
echo "[5/6] 安装到 $install_dir"
install -d "$install_dir"
install -m 0755 "$build_dir/agent-doctor" "$install_dir/agent-doctor"
echo "[6/6] 配置 Codex、Claude Code，并自动启动"
"$install_dir/agent-doctor" setup --all --yes --json

exec "$install_dir/agent-doctor" start
