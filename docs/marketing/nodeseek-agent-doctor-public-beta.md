# 做了个本地 AI 编程观察器：看任务有没有循环、验证有没有漏、Token 到底怎么花的

一句话用途：跑 Codex、Claude Code 这类长任务时，用一个本地仪表盘看清任务进度、验证证据、Token / 费用和跨客户端交接，而不是只翻聊天记录。

项目叫 Agent Doctor，是 NexoToken 发起的独立开源工具，目前准备以公开测试版让大家先用起来。

[GitHub：Agent Doctor](https://github.com/18534516725/Agent-Doctor)

[产品页：Agent Doctor by NexoToken](https://www.nexotoken.net/official/tools/agent-doctor?utm_source=nodeseek&utm_medium=community&utm_campaign=agent_doctor_beta)

## 能立刻验证的三件事

**1. 看任务是不是在真实推进**

它会整理本地事件、进度、工具调用和验证结果，区分“正在推进”“重复循环”“缺少验证”等状态。不是看到一次慢请求就下结论，诊断会同时列依据、反向证据、来源和限制。

**2. 看 Token 和费用证据靠不靠谱**

页面不会把没有账单数据写成 `$0`。精确账单叫 `exact`，按用量和带版本价格计算的叫 `estimated`，缺依据就显示 `unavailable`，三种口径不会混成一个假精确总数。

**3. 换客户端时带走有用上下文**

从一个已捕获的 Codex 任务切到 Claude Code 时，可以交接最近目标、结果、已确认项目记忆、来源和限制。它不会把完整聊天一股脑塞进新会话，交接内容会经过长度限制和敏感信息过滤。

## 一条命令安装

现在从源码体验：

```bash
git clone https://github.com/18534516725/Agent-Doctor.git
cd Agent-Doctor
./scripts/install-local.sh
```

安装后启动：

```bash
agent-doctor start
```

要实时采集一段新的 Codex / Claude Code 会话，用 wrapper 启动：

```bash
agent-doctor run -- codex
# 或 agent-doctor run -- claude
```

正式 Release 后会补充带 SHA-256 校验的 macOS、Linux、Windows 安装包；在 Release 地址可访问以前，不建议把远程一键安装命令当成稳定入口。

## 数据放在哪里

数据保存在当前用户自己的 SQLite，数据库由程序首次运行自动生成，不会提交到 GitHub。服务只监听 `127.0.0.1`。完整对话只用于本机私有明细，API Key、Authorization、Cookie 和传输请求头不会写入数据库。

分析规则在本地确定性运行，不需要单独配置一个“裁判模型”API Key。NexoToken 用量导入是可选能力，不注册、不导入也能完成核心诊断。

## 当前边界

- 不能无侵入附着到任意已经运行的客户端，新实时会话要通过 `agent-doctor run -- ...` 启动；
- Codex 的 MCP / Skill 能提供上下文建议，但不能保证强制阻断；
- Claude Code 只在官方支持的部分 Hook 点具备可执行控制；
- Cursor、Windsurf、Aider 等客户端公开接口不同，能看到的证据也不同；
- 目前是公开测试版，UI、安装流程和客户端兼容性还会继续改；
- 不读编辑器私有数据库，不把“检测到客户端”冒充“已经在线连接”。

## 如何反馈

如果你愿意帮忙测，最好拿一个真实长任务跑十几分钟，然后看任务状态、验证缺口和费用精度是否符合实际。提 Issue 时请附系统、架构、Agent Doctor 版本、客户端版本、启动命令和最小复现步骤。

请不要上传完整 SQLite、完整对话、私有源码或任何密钥。能公开复现的日志片段也先检查 Authorization、Cookie 和本地路径。

这个项目不会把 NexoToken 注册塞成使用前提。关系就是 **Agent Doctor by NexoToken**：NexoToken 负责发起和持续维护，工具本身留在独立公开仓库，本地核心能力保持可验证。

仓库：[GitHub：Agent Doctor](https://github.com/18534516725/Agent-Doctor)

介绍：[产品页：Agent Doctor by NexoToken](https://www.nexotoken.net/official/tools/agent-doctor?utm_source=nodeseek&utm_medium=community&utm_campaign=agent_doctor_beta)
