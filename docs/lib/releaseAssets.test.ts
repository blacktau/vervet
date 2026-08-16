import { describe, expect, it } from 'vitest'
import {
  classifyAsset,
  detectArch,
  detectPlatform,
  formatSize,
  pickBestAsset,
  type ReleaseAsset,
} from './releaseAssets'

const asset = (name: string): ReleaseAsset => ({
  name,
  browser_download_url: `https://example.test/${name}`,
  size: 1024 * 1024 * 42,
})

const release: ReleaseAsset[] = [
  'Vervet-linux-amd64.AppImage',
  'Vervet-linux-amd64.deb',
  'Vervet-linux-amd64.rpm',
  'Vervet-linux-amd64.src.rpm',
  'Vervet-macos-amd64.dmg',
  'Vervet-macos-arm64.dmg',
  'Vervet-windows-amd64-installer.exe',
  'Vervet-windows-amd64-portable.zip',
  'Vervet-windows-arm64-installer.exe',
  'Vervet-windows-arm64-portable.zip',
].map(asset)

describe('classifyAsset', () => {
  it('reads platform, arch and kind from the filename', () => {
    expect(classifyAsset(asset('Vervet-windows-arm64-installer.exe'))).toMatchObject({
      platform: 'windows',
      arch: 'arm64',
      kind: 'Installer',
      preferred: true,
    })
  })

  it('marks the AppImage as the preferred Linux build', () => {
    expect(classifyAsset(asset('Vervet-linux-amd64.AppImage')).preferred).toBe(true)
    expect(classifyAsset(asset('Vervet-linux-amd64.deb')).preferred).toBe(false)
  })

  it('never prefers a source rpm', () => {
    const classified = classifyAsset(asset('Vervet-linux-amd64.src.rpm'))
    expect(classified.preferred).toBe(false)
    expect(classified.kind).toBe('Source RPM')
  })

  it('leaves unrecognised assets unclassified', () => {
    expect(classifyAsset(asset('checksums.txt'))).toMatchObject({ platform: null, arch: null })
  })
})

describe('pickBestAsset', () => {
  it('matches platform and arch exactly', () => {
    expect(pickBestAsset(release, 'macos', 'arm64')?.name).toBe('Vervet-macos-arm64.dmg')
    expect(pickBestAsset(release, 'windows', 'amd64')?.name).toBe(
      'Vervet-windows-amd64-installer.exe',
    )
  })

  it('falls back to amd64 when the arch is unknown', () => {
    expect(pickBestAsset(release, 'macos', null)?.name).toBe('Vervet-macos-amd64.dmg')
  })

  it('falls back to amd64 when no build exists for the arch', () => {
    // There is no Linux arm64 build; offer the amd64 AppImage rather than nothing.
    expect(pickBestAsset(release, 'linux', 'arm64')?.name).toBe('Vervet-linux-amd64.AppImage')
  })

  it('returns null when the platform is unknown', () => {
    expect(pickBestAsset(release, null, null)).toBeNull()
  })

  it('returns null when the release has no assets', () => {
    expect(pickBestAsset([], 'linux', 'amd64')).toBeNull()
  })
})

describe('detectPlatform', () => {
  it.each([
    ['Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36', 'linux'],
    ['Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15', 'macos'],
    ['Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36', 'windows'],
    ['Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)', null],
    ['Mozilla/5.0 (Linux; Android 14; Pixel 8)', null],
  ])('reads %s as %s', (ua, expected) => {
    expect(detectPlatform(ua)).toBe(expected)
  })
})

describe('detectArch', () => {
  it('spots arm64 and aarch64', () => {
    expect(detectArch('Mozilla/5.0 (X11; Linux aarch64)')).toBe('arm64')
    expect(detectArch('Mozilla/5.0 (Windows NT 10.0; ARM64)')).toBe('arm64')
  })

  it('returns null when the UA says nothing useful', () => {
    // Apple silicon Macs report Intel; the caller falls back to amd64 and
    // still shows the full asset table.
    expect(detectArch('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)')).toBeNull()
  })
})

describe('formatSize', () => {
  it('renders MB to one decimal place', () => {
    expect(formatSize(1024 * 1024 * 42)).toBe('42.0 MB')
  })
})
