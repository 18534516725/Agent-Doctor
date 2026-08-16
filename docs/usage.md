# 使用 Agent Doctor

Agent Doctor 是 NexoToken 推出的开源本地 AI 编程任务诊断工具，同时保持独立、本地优先和可离线使用。

Agent Doctor 是本地优先的 AI 编程任务可靠性与引导工具。Codex、Claude Code 负责写代码；Agent Doctor 根据本地证据判断任务是否正在重复失败、原地检查、丢失上下文或未经验证就准备结束，并把下一步指令直接反馈给正在工作的 AI。

判断规则是本地、确定性的，不会为了分析再调用一个模型。完整模型对话只用于当前电脑上的私有对话时间线；运行时引导不读取原始 prompt、源码、命令、工具输入或工具输出，只使用事件 ID、时间、有限工具标签、进展事实和不可逆指纹。

## 当前可实际使用的链路

| 使用场景 | 当前状态 | 能得到什么 |
| --- | --- | --- |
| Claude Code 官方 hooks | 已接通闭环 | 采集脱敏事件，实时返回纠偏/验收建议；`guard`/`autopilot` 可执行客户端明确支持的阻断 |
| Cline 官方 hooks | 已接通 | 任务状态、工具事件和上下文压缩等经过脱敏的生命周期事件 |
| Codex / Claude Code MCP | 已接通引导 | `get_runtime_guidance` 在任务开始、重复失败、压缩上下文和结束前返回当前指令；`get_task_evidence` 提供安全时间线 |
| Codex → Claude Code 跨 AI 接力 | 已接通 | 同一工作目录下，Claude Code 的 `SessionStart` 自动接收由最近 Codex 任务和已确认项目记忆生成的精简交接包；Codex 可通过 `get_context_capsule` 读取同一份共享上下文 |
| Cursor、Windsurf、Roo Code、Continue | MCP 配置已提供 | 可以调用只读工具；如果客户端没有提供兼容本地事件，会明确显示“不可用” |
| OpenCode | 插件基础已提供 | 事件归一化与 fail-open 行为已实现；插件需要按公开说明显式安装 |
| 费用与额度 | 本地计算器与 NexoToken 用户数据导入已实现 | 数据源、精确/估算状态与货币单位始终单独标注 |

“已接通”不表示可以读取任意客户端私有记录。每一项能力都以公开、可验证的接口为边界。

## 跨 AI 任务接力

Agent Doctor 把项目记忆保存在客户端无关的本机 SQLite 中。用户从 Codex 切换到同一项目的 Claude Code 时，启动 Hook 会按工作目录匹配原始路径和隐私哈希，自动生成最多 800 Token 的交接包。交接包按优先级包含：已确认项目记忆、最近用户目标、最近模型进展、来源客户端与会话、数据边界。候选、停用和已删除记忆不会被注入。

交接不复制完整聊天窗口，也不声称两个客户端拥有相同的会话 ID。仪表盘“项目记忆”页会显示实际带入内容、来源、目标客户端、Token 预算和最近交付时间；每次 Codex MCP 获取或 Claude Code 自动注入都会保存本地交付回执。

## 任务守护与介入级别

仪表盘首页首先显示“任务守护中”，而不是先显示 Token 图表。它给出当前状态、一个主要发现、一个下一步指令、证据数量和有效期；项目分析、指标与完整对话按需展开。

| 级别 | 行为 |
| --- | --- |
| `observe` | 只记录和展示，不向正在工作的 AI 注入建议 |
| `guide` | 默认级别；有明确证据时向 AI 提供建议，不阻断 |
| `guard` | 在客户端 Hook 明确支持且规则满足时阻断高风险动作或未经验证的结束 |
| `autopilot` | 使用当前客户端支持的最强本地保护；不会突破客户端权限边界 |

MCP 和 Skill 返回的是建议，客户端可能忽略，因此不能视为强制阻断。只有实际返回强制决策的 Hook 才能称为“已阻断”。

## 本地开发运行

```bash
go build -o ./bin/agent-doctor ./cmd/agent-doctor
./bin/agent-doctor version
./bin/agent-doctor start
```

保持仪表盘运行后，在另一个终端通过安全包装器启动正在使用的客户端：

```bash
./bin/agent-doctor run -- codex
# 或 ./bin/agent-doctor run -- claude
```

包装器复用当前终端已有的 `OPENAI_BASE_URL`、`ANTHROPIC_BASE_URL`，也可显式设置 `AGENT_DOCTOR_UPSTREAM_URL`。它只向子进程注入本地代理地址，不会改写全局配置或保存认证凭证。已经运行的编辑器无法被系统安全地“强行附加”，需要从包装器启动一次。

Claude Code hook 调用的二进制必须在 `PATH` 中。hook 数据默认存储在系统用户配置目录下的 `AgentDoctor/doctor.db`，数据库文件权限为仅当前用户可读写。

## MCP 验证

以下命令不会打开浏览器，也不会联网；它通过标准输入输出启动本地 MCP 协议服务：

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}' \
  | agent-doctor mcp serve
```

当本机已经采集到某个会话时，AI 客户端可以调用：

```text
get_runtime_guidance({"sessionId":"…"})
get_task_evidence({"sessionId":"…"})
get_context_capsule({"projectId":"当前工作目录","budget":800})
```

运行时指导包含决策类型、严重度、指令、证据 ID、禁止动作、验收要求和控制级别。不会回传原始 payload、prompt、源码、命令或工具输出。

## 隐私与数据质量

- 没有兼容数据时，显示“不可用”，不显示 0，也不猜测费用或额度。
- 需要执行验证命令时，必须由用户明确批准命令；Agent Doctor 不替用户执行项目命令。
- 私有仪表盘可以读取本机 SQLite 中的完整模型对话；MCP 与导出接口只返回安全证据和聚合，不返回完整对话、源码、凭证或传输头。
- `export --json` 仅导出安全聚合；`forget --yes --json` 删除本地数据库；`uninstall --yes --json` 只删除 Agent Doctor 自己的 Codex 配置块。
- `pause --json` 会在本机持久暂停生命周期采集；`pause --resume --json` 才会重新启用。
