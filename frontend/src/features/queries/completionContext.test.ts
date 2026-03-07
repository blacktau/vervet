import { describe, it, expect } from 'vitest'
import { analyzeContext } from './completionContext'

describe('analyzeContext', () => {
  // Basic cases
  it('returns COLLECTION_NAME after db.', () => {
    const ctx = analyzeContext('db.')
    expect(ctx.type).toBe('COLLECTION_NAME')
  })

  it('returns COLLECTION_NAME with partial prefix', () => {
    const ctx = analyzeContext('db.us')
    expect(ctx.type).toBe('COLLECTION_NAME')
    expect(ctx.prefix).toBe('us')
  })

  it('returns METHOD_NAME after db.collection.', () => {
    const ctx = analyzeContext('db.users.')
    expect(ctx.type).toBe('METHOD_NAME')
    expect(ctx.collection).toBe('users')
  })

  it('returns METHOD_NAME with partial prefix', () => {
    const ctx = analyzeContext('db.users.fi')
    expect(ctx.type).toBe('METHOD_NAME')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('fi')
  })

  it('returns FIELD_NAME inside find filter', () => {
    const ctx = analyzeContext('db.users.find({ ')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
  })

  it('returns FIELD_NAME with partial prefix', () => {
    const ctx = analyzeContext('db.users.find({ na')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('na')
  })

  it('returns QUERY_OPERATOR after field:', () => {
    const ctx = analyzeContext('db.users.find({ name: ')
    expect(ctx.type).toBe('QUERY_OPERATOR')
  })

  it('returns KEYWORD at empty position', () => {
    const ctx = analyzeContext('')
    expect(ctx.type).toBe('KEYWORD')
  })

  it('returns KEYWORD with partial prefix', () => {
    const ctx = analyzeContext('d')
    expect(ctx.type).toBe('KEYWORD')
    expect(ctx.prefix).toBe('d')
  })

  it('returns AGG_STAGE inside aggregate', () => {
    const ctx = analyzeContext('db.orders.aggregate([ ')
    expect(ctx.type).toBe('AGG_STAGE')
  })

  // Edge cases: multi-line
  it('handles multi-line: db. on new line after variable', () => {
    const ctx = analyzeContext('const filter = { status: "active" }\ndb.')
    expect(ctx.type).toBe('COLLECTION_NAME')
  })

  it('handles multi-line: find filter on new line', () => {
    const ctx = analyzeContext('db.users.find({\n  ')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
  })

  it('handles multi-line: field value on new line', () => {
    const ctx = analyzeContext('db.users.find({\n  name: ')
    expect(ctx.type).toBe('QUERY_OPERATOR')
  })

  // Edge cases: after comma in filter
  it('returns FIELD_NAME after comma in filter', () => {
    const ctx = analyzeContext('db.users.find({ name: "x", ')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
  })

  it('returns FIELD_NAME with prefix after comma', () => {
    const ctx = analyzeContext('db.users.find({ name: "x", ag')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('ag')
  })

  // Edge cases: projection (second argument)
  it('returns FIELD_NAME in projection', () => {
    const ctx = analyzeContext('db.users.find({}, { ')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
  })

  // Edge cases: other methods
  it('returns FIELD_NAME inside updateOne filter', () => {
    const ctx = analyzeContext('db.users.updateOne({ ')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
  })

  it('returns FIELD_NAME inside deleteMany filter', () => {
    const ctx = analyzeContext('db.users.deleteMany({ ')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
  })

  // Edge cases: aggregate with existing stages
  it('returns AGG_STAGE after existing stage in aggregate', () => {
    const ctx = analyzeContext('db.orders.aggregate([{ $match: { status: "A" } }, ')
    expect(ctx.type).toBe('AGG_STAGE')
  })
})
