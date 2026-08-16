# 开源一个本地 AI 编程观察器：Agent Doctor by NexoToken

最近在整理一个我们自己反复遇到的问题：Codex、Claude Code 这类 Agent 跑短任务很方便，但任务一长，就很难回答“它到底推进了多少”“是不是在循环”“验证有没有真的跑”“这次到底花了多少”。

聊天记录能看到原文，却不等于有一条可核对的证据链。于是我们把内部原型整理成了 Agent Doctor，一个由 NexoToken 发起、独立仓库维护的本地优先开源工具。

[项目仓库](https://github.com/18534516725/Agent-Doctor)

[NexoToken 产品说明](https://www.nexotoken.net/official/tools/agent-doctor?utm_source=nodeloc&utm_medium=community&utm_campaign=agent_doctor_beta)

目前定位是公开测试版：先让真实用户跑起来，再根据可复现 Issue 做兼容性和分析能力迭代。下面尽量只讲实现，不堆宣传词。

## 实现架构

核心链路是本地闭环：

```text
客户端公开接口 / 安全 wrapper
        ↓
事件归一化 + 凭证过滤
        ↓
本机 SQLite
        ↓
确定性诊断 / 成本口径 / 项目记忆
        ↓
127.0.0.1 仪表盘 + MCP / Hook / Skill 指导
```

服务只监听随机 loopback 地址，仪表盘 API 使用本次启动生成的会话令牌，并限制 Origin、frame 和 referrer。核心诊断离线可用，不需要再配一把“分析模型”的 API Key。

这里没有用第二个 LLM 给第一个 LLM 打分。诊断规则基于归一化事件、时间、进度、验证事实、证据指纹和有条件的历史基线，输出诊断代码、严重程度、依据、反向证据、来源、精度和限制。没有证据时返回 unavailable，不补一个看似正常的零。

实时捕获模式会在本机保存完整的 user、assistant、system 和 tool 消息，目的是让设备所有者可以回看真实会话。API Key、Authorization、Cookie 和传输头只在内存转发，不写入 SQLite。聚合导出和 MCP 证据工具也不会返回完整消息正文。

## MCP、Hook、Skill 分别做什么

这几个词经常被混在一起，实际能力差很多：

- **MCP**：提供诊断、费用、上下文交接等本地工具，Agent 可以主动调用；文本结果不能保证客户端一定执行。
- **Skill / 项目规则**：告诉 Agent 在任务开始、失败和结束前什么时候检查 Agent Doctor，适合补工作流约束，但仍是指导。
- **Hook**：客户端在明确生命周期事件上调用本地程序。Claude Code 的部分官方 Hook 点可以在 `guard` 或 `autopilot` 下返回阻断结果；不支持阻断的事件仍然 fail-open。
- **Wrapper / Proxy**：`agent-doctor run -- codex` 或 `agent-doctor run -- claude` 只把本地代理配置注入子进程，不修改全局凭证，也不使用 shell eval。

所以“检测到客户端”和“已经建立实时连接”是两回事。仪表盘只把真正有活动证据的客户端标成连接，不会因为电脑上存在一个配置目录就显示在线。

## 支持范围

当前仓库为 Codex、Claude Code、Cline、OpenCode、Cursor、Windsurf、Roo Code、Continue、Aider、Cherry Studio 和通用 CLI 声明了能力合同，但不是所有客户端都能暴露相同数据。

Codex 和 Claude Code 的验证路径最完整；Cursor、Windsurf 等取决于它们公开的 MCP 或生命周期接口；Aider 和普通命令行工具只能看到 wrapper 能观察到的证据。Agent Doctor 不读取任何编辑器私有数据库，也不会宣称已发现的客户端都已连接。

## 用量与费用

费用区分三类：

1. `exact`：兼容账单来源报告的实际扣费；
2. `estimated`：捕获用量乘以带版本的公开价格目录；
3. `unavailable`：缺少 Token、价格、币种或账单依据。

精确值和估算值分开展示。涉及汇率时还必须有带版本的汇率证据，否则拒绝换算。这样做牺牲了一点“仪表盘数字必须填满”的观感，但避免把不知道写成零。

## 安装与启动

Release 发布后会提供带 SHA-256 校验的 macOS、Linux、Windows 包。现在从源码体验可以运行：

```bash
git clone https://github.com/18534516725/Agent-Doctor.git
cd Agent-Doctor
./scripts/install-local.sh
```

启动仪表盘和集成检查：

```bash
agent-doctor start
```

启动一段可实时采集的新会话：

```bash
agent-doctor run -- codex
# 或 agent-doctor run -- claude
```

只预览受管配置，不写入：

```bash
agent-doctor setup --json
```

本地诊断和费用证据：

```bash
agent-doctor doctor --json
agent-doctor diagnose --json
agent-doctor costs --json
```

## 隐私边界

- SQLite 自动生成在当前用户配置目录，不提交 GitHub；
- 服务不绑定公网接口；
- 不持久化 API Key、认证头、Cookie 和传输头；
- 完整消息只在本机私有时间线中使用，不进入安全聚合导出；
- `pause` 可以暂停采集，`forget --yes` 明确删除本地数据库；
- 跨客户端交接是有长度上限、经过过滤的摘要，不是整段聊天复制。

NexoToken 用量导入和版本检查是可选网络动作。不开启它们，本地诊断仍然工作。

## 已知限制

- 已经打开的任意客户端进程不能被无侵入附着；新的实时会话需要通过 wrapper 启动；
- MCP/Skill 是建议，不等于确定性阻断；
- 客户端支持等级不同，缺失事件会降低诊断精度；
- 两个会话只能做描述性对比，至少 15 个匹配样本后才考虑输出 cohort 结果；
- 公开测试版还会有 UI、安装器和各客户端版本兼容问题；
- 发布工作流和校验文件没有完成前，不应把源码状态当成稳定正式版。

## 反馈方式

希望大家不要只发一句“不能用”，最好附上：操作系统与架构、Agent Doctor 版本、客户端及版本、启动命令、最小复现步骤、预期结果和实际结果。不要上传 API Key、完整 SQLite、完整对话、Authorization 头或私有项目源码。

如果你平时会跑长时间 Codex / Claude Code 任务，欢迎拿一个真实项目试一下。重点不是看页面有没有数据，而是检查它给出的任务状态、验证缺口、费用精度和客户端边界是否诚实。

项目：[Agent Doctor](https://github.com/18534516725/Agent-Doctor)

维护方说明：[Agent Doctor by NexoToken](https://www.nexotoken.net/official/tools/agent-doctor?utm_source=nodeloc&utm_medium=community&utm_campaign=agent_doctor_beta)
