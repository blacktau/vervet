import * as systemProxy from 'wailsjs/go/api/SystemProxy'

type LogLevel = 'debug' | 'info' | 'warn' | 'error'

const originalConsole = {
  log: console.log,
  debug: console.debug,
  info: console.info,
  warn: console.warn,
  error: console.error,
}

function formatArg(arg: unknown): string {
  if (arg instanceof Error) {
    return arg.stack ?? `${arg.name}: ${arg.message}`
  }
  if (arg === null || arg === undefined) {
    return String(arg)
  }
  if (typeof arg === 'object') {
    try {
      return JSON.stringify(arg, Object.getOwnPropertyNames(arg))
    } catch {
      return String(arg)
    }
  }
  return String(arg)
}

function formatMessage(_level: LogLevel, args: unknown[]): string {
  return args.map(formatArg).join(' ')
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
