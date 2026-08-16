#!/usr/bin/env sh
set -eu

required="README.md README.zh-CN.md CHANGELOG.md CONTRIBUTING.md SECURITY.md assets/agent-doctor-flow.svg assets/agent-doctor-social-card.svg docs/install.md docs/privacy.md docs/cost-methodology.md docs/diagnosis-methodology.md docs/compatibility.md docs/troubleshooting.md docs/roadmap.md docs/launch/public-beta.md docs/launch/feedback-guide.md docs/launch/release-checklist.md examples/demo-data.json docs/marketing/zhihu-agent-doctor-public-beta.md docs/marketing/nodeloc-agent-doctor-public-beta.md docs/marketing/nodeseek-agent-doctor-public-beta.md"
for file in $required; do
  test -s "$file" || { echo "missing documentation artifact: $file" >&2; exit 1; }
done

grep -q 'README.zh-CN.md' README.md
grep -q 'README.md' README.zh-CN.md
grep -q 'https://www.nexotoken.net/official/tools/agent-doctor' README.md
grep -q 'https://www.nexotoken.net/official/tools/agent-doctor' README.zh-CN.md
grep -q 'https://github.com/18534516725/Agent-Doctor/issues/new/choose' README.md
grep -q 'https://github.com/18534516725/Agent-Doctor/issues/new/choose' README.zh-CN.md

for command in setup start dashboard diagnose compare context costs doctor pause export forget run uninstall version; do
  grep -q "agent-doctor $command\|\`$command\`" README.md || { echo "README does not document $command" >&2; exit 1; }
done

for client in "Codex" "Claude Code" "Cline" "OpenCode" "Cursor" "Windsurf" "Roo Code" "Continue" "Aider" "Cherry Studio"; do
  grep -q "$client" docs/compatibility.md || { echo "compatibility matrix missing $client" >&2; exit 1; }
done

grep -q "exact" docs/cost-methodology.md
grep -q "estimated" docs/cost-methodology.md
grep -q "unavailable" docs/cost-methodology.md
grep -q "local" docs/privacy.md

require_article() {
  file="$1"
  shift
  grep -q '^# ' "$file" || { echo "article has no title: $file" >&2; exit 1; }
  for phrase in "$@"; do
    grep -q "$phrase" "$file" || { echo "article missing '$phrase': $file" >&2; exit 1; }
  done
  grep -q 'https://github.com/18534516725/Agent-Doctor' "$file" || { echo "article missing GitHub URL: $file" >&2; exit 1; }
  grep -q 'https://www.nexotoken.net/official/tools/agent-doctor' "$file" || { echo "article missing NexoToken product URL: $file" >&2; exit 1; }
}

require_article docs/marketing/zhihu-agent-doctor-public-beta.md \
  '为什么长任务越来越难复盘' '它实际记录什么' '它不会做什么' '如何开始使用'
require_article docs/marketing/nodeloc-agent-doctor-public-beta.md \
  '实现架构' '支持范围' '安装与启动' '隐私边界' '已知限制' '反馈方式'
require_article docs/marketing/nodeseek-agent-doctor-public-beta.md \
  '能立刻验证的三件事' '一条命令安装' '当前边界' '如何反馈'

if grep -RniE 'sk-[A-Za-z0-9_-]{16,}|Bearer [A-Za-z0-9._-]{16,}' README.md docs examples; then
  echo "documentation contains credential-shaped example" >&2
  exit 1
fi
