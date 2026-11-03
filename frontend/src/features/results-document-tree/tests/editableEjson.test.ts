import { describe, it, expect } from 'vitest'
import {
  documentIdFilter,
  ejsonParseExpression,
  toEditableEjson,
  writeCount,
} from '../editableEjson'
import { hasAmbiguousEjsonField, hasReorderedKeys } from '../bsonValue'
import { hasDocumentId, isDocumentEditable } from '../useDocumentContextMenu'
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

describe('fields named __proto__', () => {
  const doc = JSON.parse('{"a":{"$numberInt":"1"},"__proto__":{"$numberLong":"7"}}')

  it('reach the editor text as a field', () => {
    expect(JSON.stringify(toEditableEjson(doc))).toBe('{"a":1,"__proto__":{"$numberLong":"7"}}')
  })

  it('stay in an _id filter', () => {
    const id = JSON.parse('{"a":1,"__proto__":{"b":1}}')

    expect(documentIdFilter(id)).toContain('__proto__')
    expect(hasDocumentId({ _id: id })).toBe(true)
  })
})

describe('$numberInt outside int32', () => {
  it.each(['9007199254740993', '2147483648', '-2147483649', '007', '-0'])(
    'is an ordinary object, not rounded: %s',
    ($numberInt) => {
      expect(toEditableEjson({ $numberInt })).toEqual({ $numberInt })
      expect(isDocumentEditable({ _id: 1, x: { $numberInt } })).toBe(false)
    },
  )

  it.each(['2147483647', '-2147483648', '0'])('is an int32 at %s', ($numberInt) => {
    expect(toEditableEjson({ $numberInt })).toBe(Number($numberInt))
    expect(isDocumentEditable({ _id: 1, x: { $numberInt } })).toBe(true)
  })
})

describe('documentIdFilter', () => {
  it('refuses a missing _id rather than matching any document', () => {
    expect(() => documentIdFilter(undefined)).toThrow()
  })
})

describe('writeCount', () => {
  it.each([
    [[{ matchedCount: 0 }], 0],
    [[{ matchedCount: 1 }], 1],
    [[{ matchedCount: { $numberInt: '0' } }], 0],
    [[{ other: 1 }], undefined],
    [[], undefined],
    [undefined, undefined],
  ])('reads %j', (documents, expected) => {
    expect(writeCount(documents, 'matchedCount')).toBe(expected)
  })
})

describe('hasAmbiguousEjsonField', () => {
  it.each(roundTrip)('accepts a real value: $name', ({ stored }) => {
    expect(hasAmbiguousEjsonField({ _id: 1, v: stored })).toBe(false)
  })

  it.each([
    [{ $binary: 'AQI=', $type: '00' }],
    [{ $uuid: '550e8400-e29b-41d4-a716-446655440000' }],
    [{ $oid: 5 }],
    [{ $oid: 'zz' }],
    [{ $timestamp: null }],
    [{ $date: 'nope' }],
    [{ $date: '2024-01-01', name: 'x' }],
    [{ $numberInt: 'db.c.drop()' }],
    [{ $regex: 'a', $options: 'i' }],
    [{ $scope: {} }],
  ])('flags an ordinary object: %j', (field) => {
    expect(hasAmbiguousEjsonField({ a: [{ b: field }] })).toBe(true)
  })

  it('leaves other $ fields alone', () => {
    expect(hasAmbiguousEjsonField({ $custom: 1, nested: { $other: 2 } })).toBe(false)
  })
})

describe('which documents can be edited', () => {
  it('turns Edit off for a document saving would change or reject', () => {
    expect(isDocumentEditable({ _id: 1, x: { $oid: 5 } })).toBe(false)
    expect(isDocumentEditable({ _id: 1, x: { $uuid: 'a' } })).toBe(false)
  })

  it('keeps Edit on for real values', () => {
    expect(isDocumentEditable({ _id: { $oid: '507f1f77bcf86cd799439011' }, x: { $code: 'y' } })).toBe(
      true,
    )
  })

  it('turns off targeting an _id whose field order may have been lost', () => {
    expect(hasReorderedKeys({ b: 1, '1': 2 })).toBe(true)
    expect(hasDocumentId({ _id: { b: 1, '1': 2 } })).toBe(false)
  })

  it('still targets an ordinary compound _id and a single integer key', () => {
    expect(hasDocumentId({ _id: { a: 1, b: 2 } })).toBe(true)
    expect(hasDocumentId({ _id: { '1': 2 } })).toBe(true)
  })
})
