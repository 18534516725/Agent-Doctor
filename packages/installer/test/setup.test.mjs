import assert from "node:assert/strict"
import { createHash } from "node:crypto"
import { mkdtemp, mkdir, readFile, writeFile } from "node:fs/promises"
import os from "node:os"
import path from "node:path"
import test from "node:test"
import { artifactName, installRelease, selectAsset, validateAsset } from "../bin/setup.mjs"

const matrix = [
  ["darwin", "arm64", "agent-doctor-darwin-arm64"], ["darwin", "x64", "agent-doctor-darwin-x64"],
  ["linux", "arm64", "agent-doctor-linux-arm64"], ["linux", "x64", "agent-doctor-linux-x64"],
  ["win32", "arm64", "agent-doctor-windows-arm64.exe"], ["win32", "x64", "agent-doctor-windows-x64.exe"],
]

test("selects an exact artifact for every supported platform", () => {
  for (const [platform, arch, filename] of matrix) {
    assert.equal(artifactName(platform, arch), filename)
    const asset = {
      platform, arch, filename,
      url: `https://github.com/18534516725/Agent-Doctor/releases/download/v0.1.0/${filename}`,
      size: 10, sha256: "a".repeat(64),
    }
    assert.equal(selectAsset({ assets: [asset] }, platform, arch).filename, filename)
  }
  assert.throws(() => artifactName("aix", "x64"), /unsupported platform/)
  assert.throws(() => selectAsset({ assets: [] }, "darwin", "arm64"), /does not contain/)
})

test("rejects non-GitHub, oversized, or malformed release assets", () => {
  const valid = { platform: "linux", arch: "x64", filename: "agent-doctor-linux-x64", url: "https://github.com/18534516725/Agent-Doctor/releases/download/v0.1.0/agent-doctor-linux-x64", size: 10, sha256: "a".repeat(64) }
  assert.doesNotThrow(() => validateAsset(valid))
  assert.throws(() => validateAsset({ ...valid, url: "https://evil.example/binary" }), /GitHub Release/)
  assert.throws(() => validateAsset({ ...valid, size: 200 * 1024 * 1024 }), /size limit/)
  assert.throws(() => validateAsset({ ...valid, sha256: "invalid" }), /SHA-256/)
})

test("checksum failure leaves an existing binary untouched", async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), "agent-doctor-installer-"))
  const binDir = path.join(root, "bin")
  await mkdir(binDir)
  const binaryPath = path.join(binDir, "agent-doctor")
  await writeFile(binaryPath, "existing-version")
  const payload = Buffer.from("tampered-download")
  const asset = {
    platform: "linux", arch: "x64", filename: "agent-doctor-linux-x64",
    url: "https://github.com/18534516725/Agent-Doctor/releases/download/v0.1.0/agent-doctor-linux-x64",
    size: payload.length, sha256: createHash("sha256").update("expected-download").digest("hex"),
  }
  const fetchImpl = async () => new Response(payload, { status: 200 })
  await assert.rejects(() => installRelease({ manifest: { assets: [asset] }, platform: "linux", arch: "x64", binDir, fetchImpl, runSetup: async () => {} }), /checksum/)
  assert.equal(await readFile(binaryPath, "utf8"), "existing-version")
})

test("verified download is installed before setup runs", async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), "agent-doctor-installer-"))
  const binDir = path.join(root, "bin")
  const payload = Buffer.from("verified-binary")
  const asset = {
    platform: "darwin", arch: "arm64", filename: "agent-doctor-darwin-arm64",
    url: "https://github.com/18534516725/Agent-Doctor/releases/download/v0.1.0/agent-doctor-darwin-arm64",
    size: payload.length, sha256: createHash("sha256").update(payload).digest("hex"),
  }
  let setupPath = ""
  await installRelease({ manifest: { assets: [asset] }, platform: "darwin", arch: "arm64", binDir, fetchImpl: async () => new Response(payload, { status: 200 }), runSetup: async (value) => { setupPath = value } })
  assert.equal(await readFile(path.join(binDir, "agent-doctor"), "utf8"), "verified-binary")
  assert.equal(setupPath, path.join(binDir, "agent-doctor"))
})
