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

  it('does not flag "show dbs" alone', () => {
    expect(validate('show dbs')).toEqual([])
  })

  it('does not flag "show collections" alone', () => {
    expect(validate('show collections')).toEqual([])
  })

  it('does not flag "use foo" alone', () => {
    expect(validate('use foo')).toEqual([])
  })

  it('does not flag "it" alone', () => {
    expect(validate('it')).toEqual([])
  })

  it('tolerates trailing semicolons on shell sugar', () => {
    expect(validate('show dbs;')).toEqual([])
    expect(validate('use foo;')).toEqual([])
  })

  it('masks a shell-sugar line and only reports the error on the next line', () => {
    const source = 'show dbs\ndb.x.find({)'
    const markers = validate(source)
    expect(markers.length).toBeGreaterThan(0)
    for (const m of markers) {
      expect(m.startLineNumber).toBe(2)
    }
  })

  it('does not strip shell sugar when it appears inside an expression', () => {
    const markers = validate('const x = show dbs')
    expect(markers.length).toBeGreaterThan(0)
  })

  it('accepts top-level await', () => {
    expect(validate('await db.users.findOne({})')).toEqual([])
  })
})
