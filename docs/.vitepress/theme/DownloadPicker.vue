<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  classifyAsset,
  detectArch,
  detectPlatform,
  formatSize,
  pickBestAsset,
  type ClassifiedAsset,
  type ReleaseAsset,
} from '../../lib/releaseAssets'

const RELEASES_URL = 'https://github.com/blacktau/vervet/releases/latest'
const API_URL = 'https://api.github.com/repos/blacktau/vervet/releases/latest'

const PLATFORM_LABELS: Record<string, string> = {
  linux: 'Linux',
  macos: 'macOS',
  windows: 'Windows',
}

const assets = ref<ReleaseAsset[]>([])
const version = ref('')
const failed = ref(false)
const loading = ref(true)

const platform = ref<ReturnType<typeof detectPlatform>>(null)
const arch = ref<ReturnType<typeof detectArch>>(null)

const best = computed(() => pickBestAsset(assets.value, platform.value, arch.value))

const grouped = computed(() => {
  const groups: { platform: string; label: string; items: ClassifiedAsset[] }[] = []
  for (const key of ['linux', 'macos', 'windows']) {
    const items = assets.value
      .map(classifyAsset)
      .filter((a) => a.platform === key && a.kind !== 'Other')
    if (items.length > 0) {
      groups.push({ platform: key, label: PLATFORM_LABELS[key], items })
    }
  }
  return groups
})

// ponytail: unauthenticated, uncached GitHub API call (60/hr/IP). If the
// limit ever bites, bake the asset URLs in at build time and trigger a docs
// rebuild from the release workflow.
onMounted(async () => {
  platform.value = detectPlatform(navigator.userAgent)
  arch.value = detectArch(navigator.userAgent)

  try {
    const response = await fetch(API_URL, { headers: { Accept: 'application/vnd.github+json' } })
    if (!response.ok) {
      failed.value = true
      return
    }
    const release = await response.json()
    assets.value = release.assets ?? []
    version.value = release.tag_name ?? ''
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="download-picker">
    <p v-if="loading" class="download-status">Looking up the latest release…</p>

    <p v-else-if="failed || assets.length === 0" class="download-status">
      <a class="download-primary" :href="RELEASES_URL">Download from GitHub Releases</a>
    </p>

    <template v-else>
      <a v-if="best" class="download-primary" :href="best.browser_download_url">
        Download for {{ PLATFORM_LABELS[best.platform!] }}
        <span class="download-meta">{{ version }} · {{ best.kind }} · {{ formatSize(best.size) }}</span>
      </a>
      <p v-else class="download-status">
        Pick the build for your system below, or
        <a :href="RELEASES_URL">browse all releases</a>.
      </p>

      <details class="download-all" :open="!best">
        <summary>All downloads{{ version ? ` (${version})` : '' }}</summary>
        <div v-for="group in grouped" :key="group.platform" class="download-group">
          <h4>{{ group.label }}</h4>
          <ul>
            <li v-for="item in group.items" :key="item.name">
              <a :href="item.browser_download_url">{{ item.name }}</a>
              <span class="download-meta">{{ item.kind }} · {{ item.arch }} · {{ formatSize(item.size) }}</span>
            </li>
          </ul>
        </div>
      </details>
    </template>
  </div>
</template>

<style scoped>
.download-picker {
  margin: 2rem 0;
}

.download-primary {
  display: inline-block;
  padding: 0.75rem 1.5rem;
  border-radius: 20px;
  background-color: var(--vp-c-brand-3);
  color: var(--vp-c-white);
  font-weight: 600;
  text-decoration: none;
}

.download-primary:hover {
  background-color: var(--vp-c-brand-2);
}

.download-meta {
  display: block;
  font-size: 0.8em;
  font-weight: 400;
  opacity: 0.85;
}

.download-status {
  color: var(--vp-c-text-2);
}

.download-all {
  margin-top: 1.5rem;
}

.download-group h4 {
  margin: 1rem 0 0.25rem;
}

.download-group ul {
  list-style: none;
  padding: 0;
}

.download-group li {
  margin: 0.5rem 0;
}
</style>
