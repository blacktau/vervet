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

  // getCollection string completions
  it('returns COLLECTION_NAME_STRING inside getCollection single quotes', () => {
    const ctx = analyzeContext("db.getCollection('")
    expect(ctx.type).toBe('COLLECTION_NAME_STRING')
    expect(ctx.insideQuotes).toBe(true)
    expect(ctx.prefix).toBe('')
  })

  it('returns COLLECTION_NAME_STRING inside getCollection double quotes', () => {
    const ctx = analyzeContext('db.getCollection("')
    expect(ctx.type).toBe('COLLECTION_NAME_STRING')
    expect(ctx.insideQuotes).toBe(true)
  })

  it('returns COLLECTION_NAME_STRING with prefix', () => {
    const ctx = analyzeContext("db.getCollection('lis")
    expect(ctx.type).toBe('COLLECTION_NAME_STRING')
    expect(ctx.prefix).toBe('lis')
    expect(ctx.insideQuotes).toBe(true)
  })

  // Quoted field names
  it('returns FIELD_NAME inside quoted key position', () => {
    const ctx = analyzeContext('db.users.find({ "')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
    expect(ctx.insideQuotes).toBe(true)
    expect(ctx.prefix).toBe('')
  })

  it('returns FIELD_NAME with prefix inside quoted key', () => {
    const ctx = analyzeContext('db.users.find({ "addr')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('addr')
    expect(ctx.insideQuotes).toBe(true)
  })

  it('returns QUERY_OPERATOR after quoted field key with colon', () => {
    const ctx = analyzeContext('db.users.find({ "address.country": ')
    expect(ctx.type).toBe('QUERY_OPERATOR')
  })

  it('returns FIELD_NAME not insideQuotes for unquoted field', () => {
    const ctx = analyzeContext('db.users.find({ na')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.insideQuotes).toBe(false)
    expect(ctx.prefix).toBe('na')
  })

  // Quoted field after comma
  it('returns FIELD_NAME inside quoted key after comma', () => {
    const ctx = analyzeContext('db.users.find({ "name": "x", "')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
    expect(ctx.insideQuotes).toBe(true)
  })

  // Dotted field paths inside quotes
  it('returns FIELD_NAME with dotted prefix inside quoted key', () => {
    const ctx = analyzeContext('db.users.find({ "address.')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('address.')
    expect(ctx.insideQuotes).toBe(true)
  })

  it('returns FIELD_NAME with dotted prefix and partial child', () => {
    const ctx = analyzeContext('db.users.find({ "address.cou')
    expect(ctx.type).toBe('FIELD_NAME')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('address.cou')
    expect(ctx.insideQuotes).toBe(true)
  })

  it('returns QUERY_OPERATOR after dotted quoted field with colon', () => {
    const ctx = analyzeContext('db.users.find({ "address.country": ')
    expect(ctx.type).toBe('QUERY_OPERATOR')
  })

  // Nested operator objects
  it('returns QUERY_OPERATOR inside nested operator object after {', () => {
    const ctx = analyzeContext('db.users.find({ "age": { ')
    expect(ctx.type).toBe('QUERY_OPERATOR')
    expect(ctx.prefix).toBe('')
  })

  it('returns QUERY_OPERATOR with $ prefix inside nested operator object', () => {
    const ctx = analyzeContext('db.users.find({ "age": { $')
    expect(ctx.type).toBe('QUERY_OPERATOR')
    expect(ctx.prefix).toBe('$')
  })

  it('returns QUERY_OPERATOR with partial operator inside nested object', () => {
    const ctx = analyzeContext('db.users.find({ "age": { $gt')
    expect(ctx.type).toBe('QUERY_OPERATOR')
    expect(ctx.prefix).toBe('$gt')
  })

  it('returns QUERY_OPERATOR inside nested object with unquoted field', () => {
    const ctx = analyzeContext('db.users.find({ age: { $n')
    expect(ctx.type).toBe('QUERY_OPERATOR')
    expect(ctx.prefix).toBe('$n')
  })

  // Cursor method chaining
  it('returns CURSOR_METHOD after find().', () => {
    const ctx = analyzeContext('db.users.find({}).')
    expect(ctx.type).toBe('CURSOR_METHOD')
    expect(ctx.prefix).toBe('')
  })

  it('returns CURSOR_METHOD with partial prefix after find().li', () => {
    const ctx = analyzeContext('db.users.find({}).li')
    expect(ctx.type).toBe('CURSOR_METHOD')
    expect(ctx.prefix).toBe('li')
  })

  it('returns CURSOR_METHOD after chained limit().', () => {
    const ctx = analyzeContext('db.users.find({}).limit(10).')
    expect(ctx.type).toBe('CURSOR_METHOD')
    expect(ctx.prefix).toBe('')
  })

  it('returns CURSOR_METHOD after chained sort().', () => {
    const ctx = analyzeContext('db.users.find({}).sort({ name: 1 }).')
    expect(ctx.type).toBe('CURSOR_METHOD')
    expect(ctx.prefix).toBe('')
  })

  it('returns UPDATE_OPERATOR for updateOne update doc with prefix start', () => {
    const ctx = analyzeContext('db.users.updateOne({ "name": "joe" }, { $')
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('$')
  })

  it('returns UPDATE_OPERATOR for updateOne in clean update doc', () => {
    const ctx = analyzeContext('db.users.updateOne({ "name": "joe" }, {')
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('')
  })

  it('returns UPDATE_OPERATOR for updateOne with partial prefix', () => {
    const ctx = analyzeContext('db.users.updateOne({ "name": "joe" }, { $s')
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('$s')
  })

  it('returns UPDATE_OPERATOR for updateMany update doc with prefix start', () => {
    const ctx = analyzeContext(
      'db.employees.updateMany({ "salary": { $lt: 100000 }, raiseApplied: { $ne: true } }, { $',
    )
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('employees')
    expect(ctx.prefix).toBe('$')
  })

  it('returns UPDATE_OPERATOR for updateMany in clean update doc', () => {
    const ctx = analyzeContext(
      'db.employees.updateMany({ "salary": { $lt: 100000 }, raiseApplied: { $ne: true } }, {',
    )
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('employees')
    expect(ctx.prefix).toBe('')
  })

  it('returns UPDATE_OPERATOR for updateMany with partial prefix', () => {
    const ctx = analyzeContext('db.users.findOneAndUpdate({ "name": "joe" }, { $s')
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('$s')
  })

  it('returns UPDATE_OPERATOR for findOneAndUpdate update doc with prefix start', () => {
    const ctx = analyzeContext('db.users.findOneAndUpdate({ "name": "joe" }, { $')
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('$')
  })

  it('returns UPDATE_OPERATOR for updateMany in clean update doc', () => {
    const ctx = analyzeContext('db.users.findOneAndUpdate({ "name": "joe" }, {')
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('users')
    expect(ctx.prefix).toBe('')
  })

  it('returns UPDATE_OPERATOR for updateMany with partial prefix', () => {
    const ctx = analyzeContext(
      'db.employees.updateMany({ "salary": { $lt: 100000 }, raiseApplied: { $ne: true } }, { $i',
    )
    expect(ctx.type).toBe('UPDATE_OPERATOR')
    expect(ctx.collection).toBe('employees')
    expect(ctx.prefix).toBe('$i')
  })
})
