# Agent Doctor One-command Start Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `agent-doctor start` check managed integrations, start the local service, and open the dashboard in one command.

**Architecture:** Keep orchestration in `internal/app`. Extract small helpers for idempotent integration preparation and platform browser command selection so behavior is testable without opening a real browser. Preserve `--no-open`, `--once`, and the existing setup/uninstall paths.

**Tech Stack:** Go, SQLite, OS-native URL launch commands, Go tests.

---

### Task 1: Lock the command contract with failing tests

**Files:**
- Modify: `internal/app/app_test.go`

- [ ] Add a test proving start preparation installs exactly one owned Codex block and remains idempotent.
- [ ] Add table tests proving macOS, Linux, and Windows select argument-vector browser commands without a shell.
- [ ] Add a test proving `--no-open` is recognized as the explicit opt-out.
- [ ] Run `go test ./internal/app` and confirm the new tests fail because the helpers do not exist.

### Task 2: Implement the single-command lifecycle

**Files:**
- Modify: `internal/app/app.go`

- [ ] Add `prepareManagedIntegrations()` using `BuildCodexMCPPlan` and `Apply`.
- [ ] Add `browserCommand()` and an asynchronous `openBrowserURL()` with OS-specific argument vectors.
- [ ] Invoke preparation before detection and server creation.
- [ ] Open the URL after the listener is ready unless `--no-open` or `--once` is present.
- [ ] Keep configuration and browser failures fail-open with actionable local messages.
- [ ] Run `go test ./internal/app` and confirm all tests pass.

### Task 3: Align documentation and verify the repository

**Files:**
- Modify: `README.md`
- Modify: `docs/install.md`
- Modify: `docs/usage.md`
- Modify: `docs/troubleshooting.md`

- [ ] Make `agent-doctor start` the primary documented command and describe `--no-open` as an opt-out.
- [ ] Run `gofmt -w internal/app/app.go internal/app/app_test.go`.
- [ ] Run `go test ./...` and require success.
- [ ] Run `pnpm --dir dashboard test -- --run` and `pnpm --dir dashboard build` and require success.
- [ ] Run `git diff --check` and inspect the final diff for credentials and unrelated changes.
