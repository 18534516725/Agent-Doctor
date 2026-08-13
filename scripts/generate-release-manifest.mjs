#!/usr/bin/env node
import { readFile, stat, writeFile } from "node:fs/promises"
import path from "node:path"
import { pathToFileURL } from "node:url"

const OWNER = "18534516725"
const REPOSITORY = "Agent-Doctor"
const TARGETS = [
  ["darwin", "amd64", "darwin", "x64", "tar.gz"],
  ["darwin", "arm64", "darwin", "arm64", "tar.gz"],
  ["linux", "amd64", "linux", "x64", "tar.gz"],
  ["linux", "arm64", "linux", "arm64", "tar.gz"],
  ["windows", "amd64", "win32", "x64", "zip"],
  ["windows", "arm64", "win32", "arm64", "zip"],
]

export async function buildReleaseManifest(directory, version) {
  if (!/^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(version)) {
    throw new Error("release version is invalid")
  }
  const rawChecksums = await readFile(path.join(directory, "SHA256SUMS.txt"), "utf8")
  const checksums = new Map(rawChecksums.split(/\r?\n/).filter(Boolean).map((line) => {
    const match = line.match(/^([a-f0-9]{64})\s+(.+)$/i)
    if (!match) throw new Error("checksum file contains an invalid line")
    return [match[2], match[1].toLowerCase()]
  }))
  const assets = []
  for (const [releaseOS, releaseArch, platform, arch, suffix] of TARGETS) {
    const filename = `agent-doctor_${version}_${releaseOS}_${releaseArch}.${suffix}`
    const sha256 = checksums.get(filename)
    if (!sha256) throw new Error(`missing release artifact checksum: ${filename}`)
    let file
    try {
      file = await stat(path.join(directory, filename))
    } catch {
      throw new Error(`missing release artifact: ${filename}`)
    }
    if (!file.isFile() || file.size < 1) throw new Error(`missing release artifact: ${filename}`)
    assets.push({
      platform,
      arch,
      filename,
      size: file.size,
      sha256,
      url: `https://github.com/${OWNER}/${REPOSITORY}/releases/download/v${version}/${filename}`,
    })
  }
  const manifest = { schemaVersion: 1, version, assets }
  await writeFile(path.join(directory, "release-manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`, { mode: 0o644 })
  return manifest
}

async function main() {
  const [, , directory = "dist", version] = process.argv
  if (!version) throw new Error("usage: generate-release-manifest.mjs <dist> <version>")
  await buildReleaseManifest(directory, version)
}

if (import.meta.url === pathToFileURL(process.argv[1] || "").href) {
  main().catch((error) => {
    process.stderr.write(`generate-release-manifest: ${error.message}\n`)
    process.exitCode = 1
  })
}
