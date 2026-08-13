import { describe, expect, it, vi } from "vitest"
import { createAgentDoctorHooks, sanitizeOpenCodeEvent } from "./index.js"

describe("OpenCode adapter", () => {
  it("keeps only allowlisted request metadata", () => {
    const normalized = sanitizeOpenCodeEvent("request", {
      sessionID: "session-1",
      model: { id: "gpt-5", provider: "private-provider" },
      directory: "/Users/alice/private-project",
      system: ["private instructions"],
      messages: [{ role: "user", content: "private prompt" }],
    })
    const encoded = JSON.stringify(normalized)
    expect(encoded).toContain('"event":"request"')
    expect(encoded).toContain('"model":"gpt-5"')
    expect(encoded).toContain('"projectHash":"sha256:')
    for (const forbidden of ["alice", "private-project", "private-provider", "private prompt", "private instructions", "messages", "system"]) {
      expect(encoded).not.toContain(forbidden)
    }
  })

  it("registers request and tool hooks and fails open", async () => {
    const submitted: unknown[] = []
    const hooks = createAgentDoctorHooks({
      directory: "/repo",
      submit: async (event) => { submitted.push(event); throw new Error("collector unavailable") },
      getCapsule: async () => "bounded project context",
    })

    const systemOutput = { system: [] as string[] }
    await expect(hooks["experimental.chat.system.transform"]?.({ sessionID: "s", model: { id: "model" } } as never, systemOutput)).resolves.toBeUndefined()
    expect(systemOutput.system).toEqual(["bounded project context"])
    await expect(hooks["tool.execute.before"]?.({ sessionID: "s", callID: "c", tool: "bash" }, { args: { command: "secret" } })).resolves.toBeUndefined()
    await expect(hooks["tool.execute.after"]?.({ sessionID: "s", callID: "c", tool: "bash", args: { command: "secret" } }, { title: "run", output: "secret output", metadata: {} })).resolves.toBeUndefined()
    expect(submitted).toHaveLength(3)
  })

  it("bounds context injection", async () => {
    const hooks = createAgentDoctorHooks({ directory: "/repo", submit: async () => {}, getCapsule: async () => "x".repeat(10000) })
    const output = { system: [] as string[] }
    await hooks["experimental.chat.system.transform"]?.({ sessionID: "s", model: { id: "model" } } as never, output)
    expect(output.system[0].length).toBeLessThanOrEqual(3200)
  })
})
