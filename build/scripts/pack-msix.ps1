<#
.SYNOPSIS
  Stages MSIX layout and packs Vervet.msix.

.PARAMETER ExePath
  Path to the Vervet.exe built with -X vervet/internal/buildinfo.channel=msstore.

.PARAMETER Version
  4-part MSIX version, e.g. "2026.4.4.0".

.PARAMETER PackageIdentity
  Manifest Identity Name, e.g. "Blacktau.Vervet".

.PARAMETER PublisherCN
  Manifest Publisher CN string, e.g. "CN=ABCD1234-1234-...".

.PARAMETER PublisherDisplayName
  Human-readable publisher, e.g. "Blacktau".

.PARAMETER OutputPath
  Path to write the resulting .msix.
#>
param(
  [Parameter(Mandatory=$true)][string]$ExePath,
  [Parameter(Mandatory=$true)][string]$Version,
  [Parameter(Mandatory=$true)][string]$PackageIdentity,
  [Parameter(Mandatory=$true)][string]$PublisherCN,
  [Parameter(Mandatory=$true)][string]$PublisherDisplayName,
  [Parameter(Mandatory=$true)][string]$OutputPath
)

$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path "$PSScriptRoot\..\..").Path
$msixSrc  = Join-Path $repoRoot 'build\windows\msix'
$stagingRoot = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { [System.IO.Path]::GetTempPath() }
$staging  = Join-Path $stagingRoot 'msix-staging'

if (Test-Path $staging) { Remove-Item -Recurse -Force $staging }
New-Item -ItemType Directory -Path $staging | Out-Null
New-Item -ItemType Directory -Path (Join-Path $staging 'Assets') | Out-Null

# Copy executable
Copy-Item $ExePath (Join-Path $staging 'Vervet.exe')

# Copy assets
Copy-Item (Join-Path $msixSrc 'Assets\*.png') (Join-Path $staging 'Assets\')

# Substitute manifest tokens
$manifestSrc = Join-Path $msixSrc 'Package.appxmanifest.template'
$manifestDst = Join-Path $staging 'AppxManifest.xml'
$content = Get-Content $manifestSrc -Raw
$content = $content.Replace('@@PACKAGE_IDENTITY_NAME@@', $PackageIdentity)
$content = $content.Replace('@@PUBLISHER_CN@@', $PublisherCN)
$content = $content.Replace('@@PUBLISHER_DISPLAY_NAME@@', $PublisherDisplayName)
$content = $content.Replace('@@MSIX_VERSION@@', $Version)
Set-Content -Path $manifestDst -Value $content -Encoding UTF8

# Locate makeappx.exe (Windows SDK)
$makeappx = (Get-ChildItem 'C:\Program Files (x86)\Windows Kits\10\bin' -Recurse -Filter makeappx.exe -ErrorAction SilentlyContinue |
             Where-Object { $_.FullName -match '\\x64\\makeappx\.exe$' } |
             Sort-Object FullName -Descending |
             Select-Object -First 1).FullName

if (-not $makeappx) { throw 'makeappx.exe not found in Windows SDK' }

# Pack
& $makeappx pack /d $staging /p $OutputPath /nv
if ($LASTEXITCODE -ne 0) { throw "makeappx pack failed (exit $LASTEXITCODE)" }

Write-Host "Packed: $OutputPath"
