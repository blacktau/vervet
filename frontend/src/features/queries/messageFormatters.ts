import type { LogMessage, MessageFilter } from './queryStore'

export function filterMessages(messages: LogMessage[], filter: MessageFilter): LogMessage[] {
  return messages.filter((m) => filter[m.level])
}

export function formatLogLine(m: LogMessage): string {
  return `${m.timestamp} [${m.level.toUpperCase()}] ${m.text}`
}
