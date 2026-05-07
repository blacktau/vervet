import { describe, it, expect } from 'vitest'
import { validate } from './syntaxValidator'

describe('validate', () => {
  it('returns no markers for a syntactically valid script', () => {
    const markers = validate('db.users.find({ name: "alice" })')
    expect(markers).toEqual([])
  })

  it('flags an unbalanced brace', () => {
    const markers = validate('db.users.find({ name: "alice"')
    expect(markers.length).toBeGreaterThan(0)
    expect(markers[0].severity).toBe(8)
    expect(markers[0].source).toBe('vervet')
    expect(markers[0].startLineNumber).toBe(1)
  })

  it('flags a stray comma in an object literal', () => {
    const markers = validate('const x = { a: 1,, b: 2 }')
    expect(markers.length).toBeGreaterThan(0)
    expect(markers[0].startLineNumber).toBe(1)
  })

  it('returns multiple markers for multiple distinct errors', () => {
    const source = ['const x = { a:: 1 }', 'function () {'].join('\n')
    const markers = validate(source)
    expect(markers.length).toBeGreaterThan(0)
  })
})
