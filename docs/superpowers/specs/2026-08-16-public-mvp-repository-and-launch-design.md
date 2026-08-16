# Agent Doctor 公开 MVP、GitHub 仓库与首发推广设计

日期：2026-08-16（Asia/Shanghai）

## 1. 目标

把现有 Agent Doctor 本地产品以公开测试版形式发布，让 Codex、Claude Code 等 AI 编程工具用户可以：

1. 在 60 秒内理解产品价值；
2. 用一条可信安装路径完成安装；
3. 在本机看到任务健康度、上下文、费用和跨客户端交接结果；
4. 通过 GitHub Issue 提交可复现问题和功能建议；
5. 明确知道 Agent Doctor 是 NexoToken 推出的开源工具，并能在需要模型 API、精确用量导入或平台服务时进入 NexoToken。

首发目标是形成“发布—真实使用—反馈—迭代”的闭环，不以一次性完成所有客户端深度能力为目标。

## 2. 产品定位

统一名称：**Agent Doctor by NexoToken**。

一句话定位：

> Agent Doctor 是一个本地优先的 AI 编程任务观测与可靠性工具，用证据解释任务为什么变慢、反复、丢失上下文或产生异常用量，并把有限、可核验的下一步建议带回正在工作的 AI。

公开内容不得把产品描述为万能监控、强制控制器或自动判定模型优劣的工具。缺少证据时继续显示“不可用”，不伪造零值、费用或排名结论。

## 3. 仓库与品牌架构

Agent Doctor 保持独立公开仓库，不并入 payment-platform 业务仓库：

```text
NexoToken 官网首页
  └─ /official/tools/agent-doctor
       ├─ 产品能力、边界和隐私说明
       ├─ 前往 GitHub
       └─ 前往安装说明

GitHub /18534516725/Agent-Doctor
  ├─ Agent Doctor by NexoToken 品牌说明
  ├─ 安装、截图、文档、Release、Issue
  ├─ 返回 NexoToken 官网
  └─ 可选的 NexoToken 个人用量导入说明
```

双向关联规则：

- NexoToken 官网负责产品介绍与品牌承接，主要 CTA 指向 GitHub；
- GitHub README 首屏明确 `by NexoToken`，链接 `https://www.nexotoken.net/official/tools/agent-doctor`；
- README 同时保留 NexoToken 官网入口，但不得要求用户注册才能使用本地核心功能；
- NexoToken 精确用量导入是可选增强能力，必须与本地离线诊断解耦；
- GitHub 仓库的 Website 字段设置为 Agent Doctor 官网专页；
- Repository topics 使用真实能力词，不堆叠无关热词。

## 4. MVP 发布范围

### 4.1 保留的现有能力

- 本地 SQLite 数据存储；
- Codex 与 Claude Code 的受控安装和运行指导；
- 常见客户端能力边界展示；
- 项目分析驾驶舱；
- 原始对话按需查看；
- 精确、估算、未知费用分离；
- 项目记忆与跨客户端任务交接；
- MCP/Skill 运行指导；
- macOS、Linux、Windows 的构建流水线；
- 本地卸载、暂停、清空数据和隐私说明。

### 4.2 发布前必须修复

跨客户端交接接口和仪表盘只能返回经过同一隐私过滤器处理的目标、结果和记忆。禁止出现“注入给 AI 的 Rendered 已过滤，但 API 仍返回原始 Goal/LatestResult/Memory”的双路径。没有真实任务或已确认记忆时必须返回明确空状态，不得生成虚假交接记录。

该项是公开发布阻塞条件，必须有后端、API 和前端回归测试。

### 4.3 首发不承诺

- 不承诺所有编辑器都能无重启热连接；
- 不承诺每个客户端都能捕获完整 Token、费用或工具事件；
- 不承诺 AI 必然执行 MCP 建议；
- 不提供云端同步、团队后台或远程对话上传；
- 不宣称已经拥有用户量、Star、排名或性能领先结论。

## 5. GitHub 仓库发布包

公开仓库首发至少包含：

1. `README.md`：英文主介绍，首屏提供中文入口；
2. `README.zh-CN.md`：中文完整介绍、安装、适用场景、限制；
3. `assets/`：真实仪表盘截图、工作流程图、社交分享图；
4. `CHANGELOG.md`：从公开测试版开始记录用户可见变化；
5. `SECURITY.md`、`CONTRIBUTING.md`、`LICENSE`：沿用并核验现有内容；
6. `.github/ISSUE_TEMPLATE/bug_report.yml`：环境、客户端、复现步骤、预期/实际结果，明确禁止粘贴凭证；
7. `.github/ISSUE_TEMPLATE/feature_request.yml`：使用场景、问题、期望结果；
8. `.github/ISSUE_TEMPLATE/client_support.yml`：客户端版本、可用能力、缺失信号；
9. `.github/ISSUE_TEMPLATE/config.yml`：指向 Discussions、隐私和安全报告；
10. `.github/PULL_REQUEST_TEMPLATE.md`：测试、隐私、客户端边界检查；
11. `docs/roadmap.md`：只列近期可验证迭代；
12. `docs/launch/`：首发说明、反馈指南和发布检查表；
13. `docs/marketing/`：知乎、NodeLoc、NodeSeek 三套可直接发布的 Markdown。

GitHub About 配置：

- Description：`Local-first diagnostics, context handoff, and cost evidence for AI coding agents — by NexoToken.`
- Website：`https://www.nexotoken.net/official/tools/agent-doctor`
- Topics：`ai-coding`、`codex`、`claude-code`、`developer-tools`、`local-first`、`observability`、`mcp`、`sqlite`。

## 6. 安装与 Release 策略

首次公开版本使用预发布版本 `v0.1.0-beta.1`，不直接冒充成熟的 `v1.0.0`。

发布流程：

1. 修复隐私阻塞并运行完整测试；
2. 更新安装脚本、README 和版本口径到同一预发布版本；
3. 创建公开仓库并推送 `main`；
4. 等待 GitHub CI 的 macOS、Linux、Windows 矩阵全部通过；
5. 创建 `v0.1.0-beta.1` 标签；
6. Release 工作流生成六个平台/架构产物、SBOM、SHA256 和 manifest；
7. 下载公开 Release 产物，在独立临时目录重新校验；
8. 校验通过后才在推广文章中放远程一键安装命令。

当前本机 `gh` 使用的 `GH_TOKEN` 无效，并且公开仓库访问返回 404。代码和发布材料可以先完成，但真正创建远程仓库前必须恢复 GitHub 身份验证；不得把令牌写进仓库。

## 7. 用户反馈闭环

```text
推广文章 / NexoToken 官网
        ↓
GitHub README 与真实截图
        ↓
安装公开测试版
        ↓
本地仪表盘 / MCP 指导
        ↓
Issue 模板提交问题或建议
        ↓
按客户端、严重程度、复现率分类
        ↓
修复 → 测试 → CHANGELOG → 新 Release
```

Issue 中不得要求用户上传完整 SQLite、完整对话、API Key、Authorization Header、Cookie 或私有源码。需要诊断材料时只接受 Agent Doctor 的安全聚合导出和用户主动删减后的复现示例。

## 8. 三个平台的推广内容

### 8.1 知乎

采用故事和真实痛点结构：AI 编程任务越来越长后，用户不知道慢在哪里、Token 为什么变大、切换工具后上下文为什么丢失。正文先讲问题和解决过程，再介绍 Agent Doctor，避免价格硬广。提供 GitHub、NexoToken 产品页和隐私边界。

### 8.2 NodeLoc

采用技术发布帖结构：开源动机、架构、支持边界、本地 SQLite、MCP/Hook/Skill 差异、安装命令、已知限制、反馈入口。强调公开测试版和真实可复现反馈，不使用夸张宣传。

### 8.3 NodeSeek

采用社区短帖结构：先给一句话用途、三项核心能力、安装命令和截图，再说明 NexoToken 关联与本地数据边界。控制篇幅，避免连续堆功能或营销优惠。

三个帖子分别使用独立的 NexoToken 跟踪参数；GitHub 主链接保持统一仓库地址。帖子不得伪造用户评价、Star 数、排名、性能对比或“全客户端自动连接”等未验证结论。

## 9. 验收标准

### 9.1 产品与安全

- 跨客户端交接 API 和仪表盘没有未过滤的目标、结果或记忆；
- 无任务时不生成交接内容或交付记录；
- 本地数据库、日志、凭证和测试私有数据未进入 Git；
- 一键安装、启动、卸载和清除数据路径均有自动测试；
- 中文与英文 README 的安装命令和版本一致。

### 9.2 仓库

- 公开仓库可匿名访问；
- About、Website、topics 和社交分享图完整；
- CI 全绿；
- Issue 模板可用；
- Release 产物与 SHA256 可独立验证；
- README 与 NexoToken 官网互相链接。

### 9.3 推广

- 三篇 Markdown 均可直接复制发布；
- 每篇平台语气、长度和结构不同；
- GitHub 和 NexoToken 产品页链接正确；
- 不包含虚假数据、不可验证承诺或敏感信息；
- 用户可以从文章进入仓库、安装、查看限制并提交反馈。

## 10. 实施顺序

1. 修复交接隐私与空状态阻塞；
2. 完成 README、截图、Issue、CHANGELOG、Roadmap 和 Launch 文档；
3. 生成知乎、NodeLoc、NodeSeek 推广 Markdown；
4. 更新版本与预发布安装口径；
5. 恢复 GitHub 登录并创建公开仓库；
6. 推送 `main`，观察 CI；
7. 发布并验证 `v0.1.0-beta.1`；
8. 回填 NexoToken 官网最终 Release 链接；
9. 发布三篇推广文章并收集首轮 Issue。

