$ErrorActionPreference = 'Stop'
$version = if ($env:AGENT_DOCTOR_VERSION) { $env:AGENT_DOCTOR_VERSION } else { '1.0.0' }
$arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq 'Arm64') { 'arm64' } else { 'amd64' }
$archive = "agent-doctor_${version}_windows_${arch}.zip"
$base = "https://github.com/18534516725/Agent-Doctor/releases/download/v${version}"
$temporary = Join-Path ([System.IO.Path]::GetTempPath()) ("agent-doctor-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $temporary | Out-Null
try {
  Invoke-WebRequest -Uri "$base/$archive" -OutFile (Join-Path $temporary $archive) -UseBasicParsing
  Invoke-WebRequest -Uri "$base/SHA256SUMS.txt" -OutFile (Join-Path $temporary 'SHA256SUMS.txt') -UseBasicParsing
  $expectedLine = Get-Content (Join-Path $temporary 'SHA256SUMS.txt') | Where-Object { $_ -match [regex]::Escape($archive) }
  if (-not $expectedLine) { throw 'Checksum entry missing' }
  $expected = ($expectedLine -split '\s+')[0].ToLowerInvariant()
  $actual = (Get-FileHash (Join-Path $temporary $archive) -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($expected -ne $actual) { throw 'SHA-256 mismatch' }
  Expand-Archive (Join-Path $temporary $archive) -DestinationPath $temporary
  $install = Join-Path $env:LOCALAPPDATA 'AgentDoctor\bin'
  New-Item -ItemType Directory -Force -Path $install | Out-Null
  Copy-Item (Join-Path $temporary 'agent-doctor.exe') (Join-Path $install 'agent-doctor.exe') -Force
  & (Join-Path $install 'agent-doctor.exe') setup --yes --json
  Write-Host "Installed. Add $install to PATH, then run: agent-doctor start --no-open"
} finally {
  Remove-Item -Recurse -Force $temporary -ErrorAction SilentlyContinue
}
