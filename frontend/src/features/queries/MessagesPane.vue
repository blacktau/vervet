<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NIcon, NTooltip, useThemeVars } from 'naive-ui'
import {
  InformationCircleIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
  ClipboardDocumentIcon,
} from '@heroicons/vue/24/outline'
import type { LogLevel, LogMessage, LogMessageQuery, MessageFilter } from './queryStore'
import { filterMessages, formatLogLine } from './messageFormatters'

const props = defineProps<{
  messages: LogMessage[]
  filter: MessageFilter
  fontSize: number
  fontFamily?: string
  onJumpToQuery: (q: LogMessageQuery) => void
}>()

const { t } = useI18n()
const themeVars = useThemeVars()
const notifier = useNotification()

const visibleMessages = computed(() => filterMessages(props.messages, props.filter))

const containerRef = ref<HTMLElement | null>(null)
const stickToBottom = ref(true)

function onScroll() {
  const el = containerRef.value
  if (!el) {
    return
  }
  const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
  stickToBottom.value = distanceFromBottom < 32
}

watch(
  () => props.messages.length,
  async () => {
    if (!stickToBottom.value) {
      return
    }
    await nextTick()
    const el = containerRef.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  },
)

const iconForLevel: Record<LogLevel, typeof InformationCircleIcon> = {
  info: InformationCircleIcon,
  warning: ExclamationTriangleIcon,
  error: XCircleIcon,
}

function colorForLevel(level: LogLevel): string {
  if (level === 'error') {
    return themeVars.value.errorColor
  }
  if (level === 'warning') {
    return themeVars.value.warningColor
  }
  return themeVars.value.infoColor
}

async function copyMessage(m: LogMessage) {
  try {
    await navigator.clipboard.writeText(formatLogLine(m))
    notifier.success({ content: t('query.messages.copied'), duration: 1500 })
  } catch (e) {
    notifier.error({ content: String(e), duration: 3000 })
  }
}

function onRowClick(m: LogMessage) {
  if (m.query) {
    props.onJumpToQuery(m.query)
  }
}

const containerStyle = computed(() => {
  const style: Record<string, string> = {
    fontSize: `${props.fontSize}px`,
  }
  if (props.fontFamily) {
    style.fontFamily = `"${props.fontFamily}", monospace`
  }
  return style
})
</script>

<template>
  <div ref="containerRef" class="messages-pane" :style="containerStyle" @scroll="onScroll">
    <div v-if="messages.length === 0" class="empty-state">
      {{ t('query.messages.noMessages') }}
    </div>
    <div v-else-if="visibleMessages.length === 0" class="empty-state">
      {{ t('query.messages.noFilteredMessages') }}
    </div>
    <div
      v-for="m in visibleMessages"
      :key="m.id"
      class="row"
      :class="{ 'row--clickable': !!m.query }"
      :title="m.query ? t('query.messages.jumpToQuery') : undefined"
      @click="onRowClick(m)">
      <n-icon
        class="row-icon"
        :component="iconForLevel[m.level]"
        :style="{ color: colorForLevel(m.level) }" />
      <span class="row-timestamp">{{ m.timestamp }}</span>
      <span class="row-text" :style="{ color: colorForLevel(m.level) }">{{ m.text }}</span>
      <n-tooltip>
        <template #trigger>
          <n-button
            class="row-copy"
            quaternary
            size="tiny"
            @click.stop="copyMessage(m)">
            <template #icon>
              <n-icon :component="ClipboardDocumentIcon" />
            </template>
          </n-button>
        </template>
        {{ t('query.messages.copy') }}
      </n-tooltip>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.messages-pane {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 4px 0;
  font-family: monospace;
  -webkit-user-select: text;
  user-select: text;
}

.empty-state {
  padding: 16px;
  text-align: center;
  color: var(--n-text-color-3);
  font-size: 13px;
}

.row {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 2px 8px;
  line-height: 1.5;
  border-radius: 2px;

  &:hover {
    background-color: var(--n-hover-color);
    .row-copy {
      opacity: 1;
    }
  }

  &--clickable {
    cursor: pointer;
  }
}

.row-icon {
  flex-shrink: 0;
  margin-top: 2px;
  font-size: 14px;
}

.row-timestamp {
  flex-shrink: 0;
  color: var(--n-text-color-3);
}

.row-text {
  flex: 1;
  white-space: pre-wrap;
  word-break: break-word;
}

.row-copy {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}
</style>
