param(
  [string]$Agent = "codex",
  [string]$InstallDir = "$HOME\AppData\Local\Programs\assemble-flow",
  [string]$SkillDir = "",
  [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

$Repo = "SSYCloud/AssembleFlow"
$ApiBase = "https://api.github.com/repos/$Repo"

function Resolve-SkillDir {
  param([string]$AgentName, [string]$Override)
  if ($Override) { return $Override }
  switch ($AgentName) {
    "codex" { return "$HOME\.codex\skills\assemble-flow" }
    "claude" { return "$HOME\.claude\skills\assemble-flow" }
    default { throw "unsupported agent: $AgentName" }
  }
}

function Resolve-Tag {
  param([string]$Requested)
  if ($Requested -ne "latest") { return $Requested }
  $resp = Invoke-RestMethod -Uri "$ApiBase/releases/latest" -Headers @{ Accept = "application/vnd.github+json"; "User-Agent" = "assemble-flow-installer" }
  if (-not $resp.tag_name) { throw "failed to resolve latest release tag" }
  return [string]$resp.tag_name
}

function Get-ChecksumMap {
  param([string]$ChecksumsPath)
  $map = @{}
  Get-Content $ChecksumsPath | ForEach-Object {
    if ($_ -match '^([0-9a-fA-F]+)\s+(.+)$') {
      $map[$matches[2]] = $matches[1].ToLowerInvariant()
    }
  }
  return $map
}

function Assert-Checksum {
  param(
    [string]$AssetName,
    [string]$FilePath,
    [hashtable]$ChecksumMap
  )
  if (-not $ChecksumMap.ContainsKey($AssetName)) { return }
  $actual = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLowerInvariant()
  $expected = $ChecksumMap[$AssetName]
  if ($actual -ne $expected) {
    throw "checksum mismatch for $AssetName"
  }
}

$arch = switch ($env:PROCESSOR_ARCHITECTURE.ToLowerInvariant()) {
  "amd64" { "amd64" }
  "arm64" { "arm64" }
  default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$tag = Resolve-Tag -Requested $Version
$cliAsset = "assemble-flow-windows-$arch.zip"
$skillsAsset = "assemble-flow-skills.zip"
$checksumsAsset = "checksums.txt"
$baseUrl = "https://github.com/$Repo/releases/download/$tag"

$tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ("AssembleFlow-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmpDir | Out-Null
try {
  $cliZip = Join-Path $tmpDir $cliAsset
  $skillsZip = Join-Path $tmpDir $skillsAsset
  $checksumsPath = Join-Path $tmpDir $checksumsAsset

  Write-Host "AssembleFlow installer"
  Write-Host "repo: $Repo"
  Write-Host "version: $tag"
  Write-Host "agent: $Agent"
  Write-Host "install dir: $InstallDir"
  Write-Host "skill dir: $(Resolve-SkillDir -AgentName $Agent -Override $SkillDir)"
  Write-Host ""

  Invoke-WebRequest -Uri "$baseUrl/$cliAsset" -OutFile $cliZip
  Invoke-WebRequest -Uri "$baseUrl/$checksumsAsset" -OutFile $checksumsPath
  $checksumMap = Get-ChecksumMap -ChecksumsPath $checksumsPath
  Assert-Checksum -AssetName $cliAsset -FilePath $cliZip -ChecksumMap $checksumMap

  $cliExtract = Join-Path $tmpDir "cli"
  Expand-Archive -LiteralPath $cliZip -DestinationPath $cliExtract -Force
  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  Copy-Item -LiteralPath (Join-Path $cliExtract "assemble-flow.exe") -Destination (Join-Path $InstallDir "assemble-flow.exe") -Force

  Invoke-WebRequest -Uri "$baseUrl/$skillsAsset" -OutFile $skillsZip
  Assert-Checksum -AssetName $skillsAsset -FilePath $skillsZip -ChecksumMap $checksumMap

  $skillsExtract = Join-Path $tmpDir "skills"
  Expand-Archive -LiteralPath $skillsZip -DestinationPath $skillsExtract -Force
  $finalSkillDir = Resolve-SkillDir -AgentName $Agent -Override $SkillDir
  New-Item -ItemType Directory -Force -Path $finalSkillDir | Out-Null
  Copy-Item -LiteralPath (Join-Path $skillsExtract "skills\$Agent\assemble-flow\SKILL.md") -Destination (Join-Path $finalSkillDir "SKILL.md") -Force

  Write-Host "installed:"
  Write-Host "  $(Join-Path $InstallDir 'assemble-flow.exe')"
  Write-Host "  $(Join-Path (Resolve-SkillDir -AgentName $Agent -Override $SkillDir) 'SKILL.md')"
  Write-Host ""
  Write-Host "next:"
  Write-Host "  Add $InstallDir to PATH if needed"
  Write-Host "  `$env:BATCHJOB_SERVER='https://batchjob-test.shengsuanyun.com/batch'"
  Write-Host "  `$env:BATCHJOB_TOKEN='your-token'"
  Write-Host "  assemble-flow doctor"
}
finally {
  if (Test-Path $tmpDir) {
    Remove-Item -LiteralPath $tmpDir -Recurse -Force
  }
}
