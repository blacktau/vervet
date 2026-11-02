import { describe, it, expect } from 'vitest'
import { documentIdFilter, ejsonParseExpression, toEditableEjson } from '../editableEjson'
import roundTrip from './fixtures/editRoundTrip.json'

// internal/queryengine/edit_round_trip_integration_test.go saves each "editor" value
// and checks the "stored" value comes back with the same BSON bytes.
describe('toEditableEjson', () => {
  it.each(roundTrip)('shows $name so it saves unchanged', ({ stored, editor }) => {
    expect(toEditableEjson(stored)).toEqual(editor)
  })

  it('leaves a document with a field named like a wrapper as a document', () => {
    const doc = { meta: { $date: '2024-01-01', name: 'x' }, n: { $numberInt: '1' } }

    expect(toEditableEjson(doc)).toEqual({ meta: { $date: '2024-01-01', name: 'x' }, n: 1 })
  })

  it('keeps a date beyond a JS Date as EJSON', () => {
    const value = { $date: { $numberLong: '99999999999999999' } }

    expect(toEditableEjson(value)).toEqual(value)
  })

  it('keeps a UUID with invalid base64 instead of throwing', () => {
    const value = { $binary: { base64: '!!!', subType: '04' } }

    expect(toEditableEjson(value)).toEqual(value)
  })
})

describe('script expressions', () => {
  it('passes text to EJSON.parse in strict mode', () => {
    expect(ejsonParseExpression('{"a":1}')).toBe('EJSON.parse("{\\"a\\":1}", { relaxed: false })')
  })

  it('cannot be broken out of by stored text', () => {
    const attack = '"), db.users.drop(), ("'
    const script = documentIdFilter(attack)

    expect(new Function('EJSON', `return ${script}`)({ parse: (text: string) => text })).toBe(
      JSON.stringify({ _id: attack }),
    )
  })

  it('filters on a string _id that looks like an ObjectId as a string', () => {
    expect(documentIdFilter('507f1f77bcf86cd799439011')).toBe(
      'EJSON.parse("{\\"_id\\":\\"507f1f77bcf86cd799439011\\"}", { relaxed: false })',
    )
  })
})
