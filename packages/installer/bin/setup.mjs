#!/usr/bin/env node
import { createHash } from "node:crypto"
import { chmod, mkdir, rename, rm, writeFile } from "node:fs/promises"
import os from "node:os"
import path from "node:path"
import { spawn } from "node:child_process"

const OWNER = "18534516725"
const REPOSITORY = "Agent-Doctor"
const MAX_ASSET_BYTES = 128 * 1024 * 1024

export function artifactName(platform, arch) {
  const names = {
    "darwin-arm64": "agent-doctor-darwin-arm64",
    "darwin-x64": "agent-doctor-darwin-x64",
    "linux-arm64": "agent-doctor-linux-arm64",
    "linux-x64": "agent-doctor-linux-x64",
    "win32-arm64": "agent-doctor-windows-arm64.exe",
    "win32-x64": "agent-doctor-windows-x64.exe",
  }
  const artifact = names[`${platform}-${arch}`]
  if (!artifact) throw new Error(`unsupported platform: ${platform}/${arch}`)
  return artifact
}

export function selectAsset(manifest, platform, arch) {
  const filename = artifactName(platform, arch)
  const asset = manifest?.assets?.find((candidate) => candidate?.platform === platform && candidate?.arch === arch && candidate?.filename === filename)
  if (!asset) throw new Error(`release manifest does not contain ${filename}`)
  validateAsset(asset)
  return asset
}

export function validateAsset(asset) {
  const expectedPrefix = `https://github.com/${OWNER}/${REPOSITORY}/releases/download/`
  if (typeof asset?.url !== "string" || !asset.url.startsWith(expectedPrefix)) throw new Error("asset must use the official GitHub Release host")
  const parsed = new URL(asset.url)
  if (parsed.hostname !== "github.com" || parsed.protocol !== "https:") throw new Error("asset must use HTTPS GitHub Release host")
  if (typeof asset.filename !== "string" || !/^[a-z0-9][a-z0-9.-]*$/.test(asset.filename)) throw new Error("asset filename is invalid")
  if (!Number.isInteger(asset.size) || asset.size < 1 || asset.size > MAX_ASSET_BYTES) throw new Error("asset exceeds the size limit")
  if (typeof asset.sha256 !== "string" || !/^[a-f0-9]{64}$/i.test(asset.sha256)) throw new Error("asset SHA-256 is invalid")
}

export async function installRelease({ manifest, platform, arch, binDir, fetchImpl = fetch, runSetup = defaultRunSetup }) {
  const asset = selectAsset(manifest, platform, arch)
  const response = await fetchImpl(asset.url, { redirect: "error" })
  if (!response.ok) throw new Error(`release download failed with HTTP ${response.status}`)
  const bytes = Buffer.from(await response.arrayBuffer())
  if (bytes.length !== asset.size) throw new Error("release download size does not match manifest")
  const actualChecksum = createHash("sha256").update(bytes).digest("hex")
  if (actualChecksum.toLowerCase() !== asset.sha256.toLowerCase()) throw new Error("release download checksum does not match manifest")

  await mkdir(binDir, { recursive: true, mode: 0o755 })
  const target = path.join(binDir, platform === "win32" ? "agent-doctor.exe" : "agent-doctor")
  const temporary = `${target}.new-${process.pid}`
  try {
    await writeFile(temporary, bytes, { mode: 0o755 })
    if (platform !== "win32") await chmod(temporary, 0o755)
    await rename(temporary, target)
  } finally {
    await rm(temporary, { force: true })
  }
  await runSetup(target)
  return target
}

async function defaultRunSetup(binaryPath) {
  await new Promise((resolve, reject) => {
    const child = spawn(binaryPath, ["setup"], { stdio: "inherit", windowsHide: false })
    child.once("error", reject)
    child.once("exit", (code) => code === 0 ? resolve() : reject(new Error(`agent-doctor setup exited with ${code}`)))
  })
}

async function fetchManifest(fetchImpl = fetch) {
  const url = `https://github.com/${OWNER}/${REPOSITORY}/releases/latest/download/release-manifest.json`
  const response = await fetchImpl(url, { redirect: "error" })
  if (!response.ok) throw new Error(`release manifest download failed with HTTP ${response.status}`)
  return response.json()
}

function defaultBinDirectory() {
  return path.join(os.homedir(), ".local", "bin")
}

async function main(args) {
  const dryRun = args.includes("--dry-run")
  const platform = process.platform
  const arch = process.arch
  const binDir = defaultBinDirectory()
  if (dryRun) {
    process.stdout.write(`Agent Doctor bootstrap dry run\nplatform: ${platform}/${arch}\nartifact: ${artifactName(platform, arch)}\ninstall path: ${path.join(binDir, platform === "win32" ? "agent-doctor.exe" : "agent-doctor")}\n`)
    return
  }
  const manifest = await fetchManifest()
  const installed = await installRelease({ manifest, platform, arch, binDir })
  process.stdout.write(`Agent Doctor installed: ${installed}\n`)
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main(process.argv.slice(2)).catch((error) => {
    process.stderr.write(`agent-doctor-install: ${error.message}\n`)
    process.exitCode = 1
  })
}
