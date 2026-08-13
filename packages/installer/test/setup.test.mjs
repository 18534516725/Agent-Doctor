import assert from "node:assert/strict"
import { createHash } from "node:crypto"
import { mkdtemp, mkdir, readFile, writeFile } from "node:fs/promises"
import os from "node:os"
import path from "node:path"
import test from "node:test"
import { artifactName, installRelease, selectAsset, validateAsset } from "../bin/setup.mjs"

const matrix = [
  ["darwin", "arm64", "agent-doctor_1.0.0_darwin_arm64.tar.gz"], ["darwin", "x64", "agent-doctor_1.0.0_darwin_amd64.tar.gz"],
  ["linux", "arm64", "agent-doctor_1.0.0_linux_arm64.tar.gz"], ["linux", "x64", "agent-doctor_1.0.0_linux_amd64.tar.gz"],
  ["win32", "arm64", "agent-doctor_1.0.0_windows_arm64.zip"], ["win32", "x64", "agent-doctor_1.0.0_windows_amd64.zip"],
]

test("selects an exact artifact for every supported platform", () => {
  for (const [platform, arch, filename] of matrix) {
    assert.equal(artifactName("1.0.0", platform, arch), filename)
    const asset = {
      platform, arch, filename,
      url: `https://github.com/18534516725/Agent-Doctor/releases/download/v0.1.0/${filename}`,
      size: 10, sha256: "a".repeat(64),
    }
    assert.equal(selectAsset({ version: "1.0.0", assets: [asset] }, platform, arch).filename, filename)
  }
  assert.throws(() => artifactName("1.0.0", "aix", "x64"), /unsupported platform/)
  assert.throws(() => selectAsset({ version: "1.0.0", assets: [] }, "darwin", "arm64"), /does not contain/)
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
    platform: "linux", arch: "x64", filename: "agent-doctor_0.1.0_linux_amd64.tar.gz",
    url: "https://github.com/18534516725/Agent-Doctor/releases/download/v0.1.0/agent-doctor_0.1.0_linux_amd64.tar.gz",
    size: payload.length, sha256: createHash("sha256").update("expected-download").digest("hex"),
  }
  const fetchImpl = async () => new Response(payload, { status: 200 })
  await assert.rejects(() => installRelease({ manifest: { version: "0.1.0", assets: [asset] }, platform: "linux", arch: "x64", binDir, fetchImpl, runSetup: async () => {} }), /checksum/)
  assert.equal(await readFile(binaryPath, "utf8"), "existing-version")
})

test("verified download is installed before setup runs", async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), "agent-doctor-installer-"))
  const binDir = path.join(root, "bin")
  const payload = Buffer.from("verified-binary")
  const asset = {
    platform: "darwin", arch: "arm64", filename: "agent-doctor_0.1.0_darwin_arm64.tar.gz",
    url: "https://github.com/18534516725/Agent-Doctor/releases/download/v0.1.0/agent-doctor_0.1.0_darwin_arm64.tar.gz",
    size: payload.length, sha256: createHash("sha256").update(payload).digest("hex"),
  }
  let setupPath = ""
  await installRelease({
    manifest: { version: "0.1.0", assets: [asset] }, platform: "darwin", arch: "arm64", binDir,
    fetchImpl: async () => new Response(payload, { status: 200 }),
    extractArchive: async (_archive, target) => writeFile(target, "verified-binary"),
    runSetup: async (value) => { setupPath = value },
  })
  assert.equal(await readFile(path.join(binDir, "agent-doctor"), "utf8"), "verified-binary")
  assert.equal(setupPath, path.join(binDir, "agent-doctor"))
})
