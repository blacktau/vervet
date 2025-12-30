import { padStart } from 'lodash'

export function parseHexColor(hex: string) {
  if (hex.length < 6) {
    return { r: 0, g: 0, b: 0 }
  }
  if (hex[0] === '#') {
    hex = hex.slice(1)
  }
  const bigint = parseInt(hex, 16)
  const r = (bigint >> 16) & 255
  const g = (bigint >> 8) & 255
  const b = bigint & 255
  return { r, g, b }
}

export type RGB = { r: number; g: number; b: number }
export type RGBA = RGB & { a: number }

export function hexGammaCorrection(rgb:RGB, gamma: number) {
  if (typeof rgb !== 'object') {
    return { r: 0, g: 0, b: 0 }
  }
  return {
    r: Math.max(0, Math.min(255, Math.round(rgb.r * gamma))),
    g: Math.max(0, Math.min(255, Math.round(rgb.g * gamma))),
    b: Math.max(0, Math.min(255, Math.round(rgb.b * gamma))),
  }
}

export function mixColors(rgba1: RGBA, rgba2: RGBA, weight = 0.5) {
  if (rgba1.a === undefined) {
    rgba1.a = 255
  }
  if (rgba2.a === undefined) {
    rgba2.a = 255
  }
  return {
    r: Math.floor(rgba1.r * (1 - weight) + rgba2.r * weight),
    g: Math.floor(rgba1.g * (1 - weight) + rgba2.g * weight),
    b: Math.floor(rgba1.b * (1 - weight) + rgba2.b * weight),
    a: Math.floor(rgba1.a * (1 - weight) + rgba2.a * weight),
  }
}

export function toHexColor(rgb: RGB) {
  return (
    '#' +
    padStart(rgb.r.toString(16), 2, '0') +
    padStart(rgb.g.toString(16), 2, '0') +
    padStart(rgb.b.toString(16), 2, '0')
  )
}
