# Agent Doctor 单命令启动设计

## 目标

用户只需执行 `agent-doctor start`，即可完成 Agent Doctor 自有集成配置检查、本机客户端状态刷新、本地服务启动与仪表盘打开。

## 行为

- `start` 在启动服务前检查 Codex 的 Agent Doctor 托管配置块；缺失时原子写入，已存在时跳过。
- 只允许修改带 Agent Doctor 所有权标记的配置，不覆盖用户其他配置。
- 启动后默认用系统浏览器打开随机回环地址。
- `--no-open` 保留给服务器、自动化和用户希望手动打开页面的场景。
- `--once` 只完成启动链路验证并退出，不打开浏览器。
- 集成配置失败时输出明确警告，但不阻止只读仪表盘启动。
- 原有 `setup` 命令继续保留，用于预览变更和显式卸载，不再是普通启动的必需步骤。

## 跨平台

- macOS：`open <url>`
- Windows：`rundll32 url.dll,FileProtocolHandler <url>`
- Linux：`xdg-open <url>`

浏览器进程异步启动，不继承模型凭证，也不阻塞本地服务。

## 验收

1. 一条 `agent-doctor start` 命令完成集成检查和页面打开。
2. 连续启动不会重复配置块。
3. `--no-open` 不调用浏览器。
4. 浏览器启动失败不终止本地服务。
5. 单元测试、全量 Go 测试、前端测试与构建通过。
