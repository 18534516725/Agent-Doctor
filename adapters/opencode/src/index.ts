import { createHash } from "node:crypto"
import { spawn } from "node:child_process"
import type { Hooks, Plugin, PluginInput } from "@opencode-ai/plugin"

type NormalizedEvent = {
  schemaVersion: 1
  client: "opencode"
  event: "request" | "tool.before" | "tool.after"
  sessionID: string
  projectHash: string
  model?: string
  tool?: string
  callID?: string
  success?: boolean
}

type Dependencies = {
  directory: string
  submit: (event: NormalizedEvent) => Promise<void>
  getCapsule: (sessionID: string) => Promise<string>
}

export function sanitizeOpenCodeEvent(event: NormalizedEvent["event"], raw: Record<string, unknown>): NormalizedEvent {
  const directory = stringValue(raw.directory)
  const sessionID = bounded(stringValue(raw.sessionID), "not-reported")
  const normalized: NormalizedEvent = {
    schemaVersion: 1,
    client: "opencode",
    event,
    sessionID,
    projectHash: `sha256:${createHash("sha256").update(directory || "not-reported").digest("hex")}`,
  }
  const model = raw.model && typeof raw.model === "object" ? stringValue((raw.model as Record<string, unknown>).id ?? (raw.model as Record<string, unknown>).modelID) : ""
  if (model) normalized.model = bounded(model)
  const tool = stringValue(raw.tool)
  if (tool) normalized.tool = bounded(tool)
  const callID = stringValue(raw.callID)
  if (callID) normalized.callID = bounded(callID)
  if (typeof raw.success === "boolean") normalized.success = raw.success
  return normalized
}

export function createAgentDoctorHooks(deps: Dependencies): Hooks {
  const submitFailOpen = async (event: NormalizedEvent) => {
    try { await deps.submit(event) } catch { /* Collection must never fail the OpenCode operation. */ }
  }
  return {
    "experimental.chat.system.transform": async (input, output) => {
      const raw = { ...input, directory: deps.directory, model: input.model }
      await submitFailOpen(sanitizeOpenCodeEvent("request", raw))
      try {
        const capsule = boundedContext(await deps.getCapsule(input.sessionID ?? "not-reported"))
        if (capsule) output.system.push(capsule)
      } catch { /* Context is optional and fail-open. */ }
    },
    "tool.execute.before": async (input) => {
      await submitFailOpen(sanitizeOpenCodeEvent("tool.before", { ...input, directory: deps.directory }))
    },
    "tool.execute.after": async (input, output) => {
      await submitFailOpen(sanitizeOpenCodeEvent("tool.after", { ...input, directory: deps.directory, success: !output.metadata?.error }))
    },
  }
}

function cliSubmit(event: NormalizedEvent): Promise<void> {
  return new Promise((resolve) => {
    const child = spawn("agent-doctor", ["hook", "opencode", event.event], { stdio: ["pipe", "ignore", "ignore"], windowsHide: true })
    const timer = setTimeout(() => { child.kill(); resolve() }, 500)
    child.once("error", () => { clearTimeout(timer); resolve() })
    child.once("exit", () => { clearTimeout(timer); resolve() })
    child.stdin.end(JSON.stringify(event))
  })
}

const plugin: Plugin = async (input: PluginInput) => createAgentDoctorHooks({
  directory: input.directory,
  submit: cliSubmit,
  getCapsule: async () => "",
})

export default plugin

function bounded(value: string, fallback = ""): string { const trimmed = value.trim(); return (trimmed || fallback).slice(0, 128) }
function boundedContext(value: string): string { return value.trim().slice(0, 3200) }
function stringValue(value: unknown): string { return typeof value === "string" ? value : "" }
