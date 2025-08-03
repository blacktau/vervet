import * as systemProxy from 'wailsjs/go/api/SystemProxy'

type LogLevel = 'debug' | 'info' | 'warn' | 'error'

const originalConsole = {
  log: console.log,
  debug: console.debug,
  info: console.info,
  warn: console.warn,
  error: console.error,
}

function formatMessage(level: LogLevel, args: unknown[]): string {
  const message = args
    .map((arg) => {
      if (typeof arg === 'object') {
        try {
          return JSON.stringify(arg)
        } catch {
          return String(arg)
        }
      }
      return String(arg)
    })
    .join(' ')
  return `${message}`
}

function sendToBackend(level: LogLevel, message: string) {
  systemProxy.Log(level, message)
}

function createProxy(method: keyof typeof originalConsole, level: LogLevel) {
  return (...args: unknown[]) => {
    originalConsole[method]?.(...args)
    const message = formatMessage(level, args)
    sendToBackend(level, message)
  }
}

const enableLogging = true

if (typeof window !== 'undefined' && enableLogging) {
  console.log = createProxy('log', 'info')
  console.debug = createProxy('debug', 'debug')
  console.info = createProxy('info', 'info')
  console.warn = createProxy('warn', 'warn')
  console.error = createProxy('error', 'error')
}
