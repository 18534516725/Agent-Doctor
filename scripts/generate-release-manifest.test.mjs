import assert from "node:assert/strict"
import { mkdtemp, readFile, writeFile } from "node:fs/promises"
import os from "node:os"
import path from "node:path"
import test from "node:test"

import { buildReleaseManifest } from "./generate-release-manifest.mjs"

test("builds an exact six-platform manifest from verified release artifacts", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "agent-doctor-release-"))
  const targets = [
    ["darwin", "amd64", "tar.gz"], ["darwin", "arm64", "tar.gz"],
    ["linux", "amd64", "tar.gz"], ["linux", "arm64", "tar.gz"],
    ["windows", "amd64", "zip"], ["windows", "arm64", "zip"],
  ]
  const checksumLines = []
  for (const [platform, arch, suffix] of targets) {
    const filename = `agent-doctor_1.0.0_${platform}_${arch}.${suffix}`
    await writeFile(path.join(directory, filename), `${platform}/${arch}`)
    checksumLines.push(`${"a".repeat(64)}  ${filename}`)
  }
  await writeFile(path.join(directory, "SHA256SUMS.txt"), `${checksumLines.join("\n")}\n`)

  const manifest = await buildReleaseManifest(directory, "1.0.0")
  assert.equal(manifest.schemaVersion, 1)
  assert.equal(manifest.version, "1.0.0")
  assert.equal(manifest.assets.length, 6)
  assert.deepEqual(new Set(manifest.assets.map(({ platform }) => platform)), new Set(["darwin", "linux", "win32"]))
  assert.ok(manifest.assets.every(({ url }) => url.startsWith("https://github.com/18534516725/Agent-Doctor/releases/download/v1.0.0/")))
  assert.ok(manifest.assets.every(({ sha256 }) => sha256 === "a".repeat(64)))

  const stored = JSON.parse(await readFile(path.join(directory, "release-manifest.json"), "utf8"))
  assert.deepEqual(stored, manifest)
})

test("refuses incomplete or unchecksummed release matrices", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "agent-doctor-release-"))
  await writeFile(path.join(directory, "SHA256SUMS.txt"), "")
  await assert.rejects(() => buildReleaseManifest(directory, "1.0.0"), /missing release artifact/)
})
