# 使用 Agent Doctor

Agent Doctor 是本地优先的 AI 编程任务诊断工具：它不上传完整对话、源码、命令参数、文件路径或凭证。它会把已验证的本地生命周期事件转成可解释的任务证据，并通过 MCP 提供给 AI 编程工具。

## 当前可实际使用的链路

| 使用场景 | 当前状态 | 能得到什么 |
| --- | --- | --- |
| Claude Code 官方 hooks | 已接通 | 会话开始/结束、工具成功或失败、上下文压缩等经过脱敏的生命周期事件 |
| Cline 官方 hooks | 已接通 | 任务状态、工具事件和上下文压缩等经过脱敏的生命周期事件 |
| Codex / Claude Code MCP | 已接通 | `get_task_evidence` 可读取本机已有任务的安全时间线 |
| Cursor、Windsurf、Roo Code、Continue | MCP 配置已提供 | 可以调用只读工具；如果客户端没有提供兼容本地事件，会明确显示“不可用” |
| OpenCode | 插件基础已提供 | 事件归一化与 fail-open 行为已实现；插件需要按公开说明显式安装 |
| 费用与额度 | 本地计算器与 NexoToken 用户数据导入已实现 | 数据源、精确/估算状态与货币单位始终单独标注 |

“已接通”不表示可以读取任意客户端私有记录。每一项能力都以公开、可验证的接口为边界。

## 本地开发运行

```bash
go build -o ./bin/agent-doctor ./cmd/agent-doctor
./bin/agent-doctor version
./bin/agent-doctor setup --json
./bin/agent-doctor start --no-open
```

Claude Code hook 调用的二进制必须在 `PATH` 中。hook 数据默认存储在系统用户配置目录下的 `AgentDoctor/doctor.db`，数据库文件权限为仅当前用户可读写。

## MCP 验证

以下命令不会打开浏览器，也不会联网；它通过标准输入输出启动本地 MCP 协议服务：

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}' \
  | agent-doctor mcp serve
```

当本机已经采集到某个会话时，AI 客户端可以调用：

```text
get_task_evidence({"sessionId":"…"})
```

返回内容只包含事件发生时间、事件类型、来源和可信度；不会回传原始 payload。

## 隐私与数据质量

- 没有兼容数据时，显示“不可用”，不显示 0，也不猜测费用或额度。
- 需要执行验证命令时，必须由用户明确批准命令；Agent Doctor 不替用户执行项目命令。
- 本地统计只返回聚合计数，仪表盘和 MCP 都不会读取完整 prompt、源码、凭证或上游供应信息。
- `export --json` 仅导出安全聚合；`forget --yes --json` 删除本地数据库；`uninstall --yes --json` 只删除 Agent Doctor 自己的 Codex 配置块。
