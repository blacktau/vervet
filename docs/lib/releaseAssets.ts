export type Platform = 'linux' | 'macos' | 'windows'
export type Arch = 'amd64' | 'arm64'

export interface ReleaseAsset {
  name: string
  browser_download_url: string
  size: number
}

export interface ClassifiedAsset extends ReleaseAsset {
  platform: Platform | null
  arch: Arch | null
  kind: string
  preferred: boolean
}

// Matched in order: the first entry whose suffix matches wins, so
// `.src.rpm` must be tested before `.rpm`.
const KINDS: { suffix: string; kind: string; preferred: boolean }[] = [
  { suffix: '.src.rpm', kind: 'Source RPM', preferred: false },
  { suffix: '.AppImage', kind: 'AppImage', preferred: true },
  { suffix: '.deb', kind: 'Debian package', preferred: false },
  { suffix: '.rpm', kind: 'RPM package', preferred: false },
  { suffix: '.dmg', kind: 'Disk image', preferred: true },
  { suffix: '-installer.exe', kind: 'Installer', preferred: true },
  { suffix: '-portable.zip', kind: 'Portable ZIP', preferred: false },
]

export function classifyAsset(asset: ReleaseAsset): ClassifiedAsset {
  const name = asset.name.toLowerCase()

  let platform: Platform | null = null
  if (name.includes('linux')) {
    platform = 'linux'
  } else if (name.includes('macos') || name.includes('darwin')) {
    platform = 'macos'
  } else if (name.includes('windows')) {
    platform = 'windows'
  }

  let arch: Arch | null = null
  if (name.includes('arm64')) {
    arch = 'arm64'
  } else if (name.includes('amd64')) {
    arch = 'amd64'
  }

  const match = KINDS.find((k) => name.endsWith(k.suffix.toLowerCase()))

  return {
    ...asset,
    platform,
    arch,
    kind: match ? match.kind : 'Other',
    preferred: match ? match.preferred && platform !== null : false,
  }
}

export function pickBestAsset(
  assets: ReleaseAsset[],
  platform: Platform | null,
  arch: Arch | null,
): ClassifiedAsset | null {
  if (platform === null) {
    return null
  }

  const candidates = assets.map(classifyAsset).filter((a) => a.platform === platform && a.preferred)
  if (candidates.length === 0) {
    return null
  }

  // Exact arch first; otherwise amd64, which every platform ships.
  const wanted = arch ?? 'amd64'
  return candidates.find((a) => a.arch === wanted) ?? candidates.find((a) => a.arch === 'amd64') ?? null
}

export function detectPlatform(userAgent: string): Platform | null {
  const ua = userAgent.toLowerCase()

  // Mobile first: an Android UA also contains "linux".
  if (ua.includes('android') || ua.includes('iphone') || ua.includes('ipad')) {
    return null
  }
  if (ua.includes('windows')) {
    return 'windows'
  }
  if (ua.includes('mac os x') || ua.includes('macintosh')) {
    return 'macos'
  }
  if (ua.includes('linux') || ua.includes('x11')) {
    return 'linux'
  }
  return null
}

export function detectArch(userAgent: string): Arch | null {
  const ua = userAgent.toLowerCase()
  if (ua.includes('arm64') || ua.includes('aarch64')) {
    return 'arm64'
  }
  return null
}

export function archFromUserAgentData(architecture: string | undefined): Arch | null {
  if (architecture === 'arm') {
    return 'arm64'
  }
  if (architecture === 'x86') {
    return 'amd64'
  }
  return null
}

export function formatSize(bytes: number): string {
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
