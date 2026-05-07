import { describe, it, expect } from 'vitest'
import { validate } from './syntaxValidator'

describe('validate', () => {
  it('returns no markers for a syntactically valid script', () => {
    const markers = validate('db.users.find({ name: "alice" })')
    expect(markers).toEqual([])
  })
})
