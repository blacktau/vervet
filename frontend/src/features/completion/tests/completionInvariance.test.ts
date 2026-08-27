import { describe, it, expect } from 'vitest'
import { analyzeContext } from '../completionContext'

// A generated matrix rather than hand-picked cases. Two invariants:
//
//  1. Prepending an unrelated earlier statement must not change the context at
//     the caret. The rules scan backwards, so an earlier `aggregate(`,
//     `updateOne(` or `getCollection(` used to leak into the caret's statement
//     and silently swap the suggestion list (issue #297 was one instance).
//  2. Formatting a statement across several lines must not change its context.
//
// Every context type appears in `carets`; add a row there when adding one.

const carets: [string, string][] = [
  ['COLLECTION_NAME', 'db.'],
  ['COLLECTION_NAME/prefix', 'db.us'],
  ['COLLECTION_NAME_STRING', 'db.getCollection("'],
  ['METHOD_NAME', 'db.users.'],
  ['METHOD_NAME/gc', 'db.getCollection("users").'],
  ['METHOD_NAME/prefix', 'db.users.up'],
  ['CURSOR_METHOD', 'db.users.find({}).'],
  ['CURSOR_METHOD/gc', 'db.getCollection("users").find({}).'],
  ['CURSOR_METHOD/chain', 'db.users.find({}).sort({ a: 1 }).'],
  ['FIELD_NAME', 'db.users.find({ '],
  ['FIELD_NAME/gc', 'db.getCollection("users").find({ '],
  ['FIELD_NAME/quoted', 'db.users.find({ "'],
  ['FIELD_NAME/second', 'db.users.find({ a: 1, '],
  ['FIELD_NAME/proj', 'db.users.find({}, { '],
  ['QUERY_OPERATOR', 'db.users.find({ age: { '],
  ['QUERY_OPERATOR/gc', 'db.getCollection("users").find({ age: { '],
  ['QUERY_OPERATOR/quotedkey', 'db.users.find({ "address.city": { '],
  ['UPDATE_OPERATOR', 'db.users.updateOne({}, { '],
  ['UPDATE_OPERATOR/many', 'db.users.updateMany({ a: 1 }, { '],
  ['UPDATE_OPERATOR/faou', 'db.users.findOneAndUpdate({}, { '],
  ['AGG_STAGE', 'db.users.aggregate([{ '],
  ['AGG_STAGE/gc', 'db.getCollection("users").aggregate([{ '],
  ['AGG_STAGE/empty', 'db.users.aggregate(['],
  ['AGG_STAGE/second', 'db.users.aggregate([{ $match: { a: 1 } }, { '],
  ['AGG_EXPRESSION', 'db.users.aggregate([{ $group: { t: { '],
  ['USE_DATABASE', 'use '],
  ['USE_DATABASE/prefix', 'use sh'],
  ['EJSON_METHOD', 'EJSON.'],
  ['KEYWORD', ''],
  ['KEYWORD/prefix', 'E'],
]

const prefixes: [string, string][] = [
  ['none', ''],
  ['plain-stmt', 'db.orders.find({})\n'],
  ['gc-stmt', 'db.getCollection("orders").find({})\n'],
  ['two-gc-stmts', 'db.getCollection("a").find({})\ndb.getCollection("b").find({})\n'],
  ['gc-no-call', 'db.getCollection("orders")\n'],
  ['comment', '// db.getCollection("orders").updateOne({}, {})\n'],
  ['comment-unbalanced', '// db.orders.aggregate([{ $match: {\n'],
  ['use-stmt', 'use shop\n'],
  ['semicolons', 'db.orders.find({});\ndb.getCollection("x").drop();\n'],
  ['blank-lines', 'db.getCollection("orders").find({})\n\n\n'],
  ['multiline-query', 'db.getCollection("orders").find({\n  total: 1\n})\n'],
  ['agg-above', 'db.orders.aggregate([{ $match: { a: 1 } }])\n'],
  ['multiline-agg-above', 'db.orders.aggregate([\n  { $group: { _id: "$a", n: { $sum: 1 } } }\n])\n'],
  ['update-above', 'db.orders.updateMany({}, { $set: { a: 1 } })\n'],
  ['var-assign', 'const x = db.getCollection("orders").find({})\n'],
  ['string-with-brackets', 'db.orders.find({ note: "a) { [ $match" })\n'],
  ['string-with-comment', 'db.orders.find({ url: "http://x/y" })\n'],
  ['ejson-above', 'EJSON.stringify({ a: 1 })\n'],
  ['semicolon-same-line', 'db.orders.drop(); '],
]

// Formatting a statement across lines must not change its context.
const multiline: [string, string, string][] = [
  ['find-field', 'db.users.find({ ', 'db.users.find({\n  '],
  ['find-field-gc', 'db.getCollection("users").find({ ', 'db.getCollection("users").find(\n  {\n    '],
  ['query-op', 'db.users.find({ age: { ', 'db.users.find({\n  age: {\n    '],
  ['update-op', 'db.users.updateOne({}, { ', 'db.users.updateOne(\n  {},\n  {\n    '],
  ['agg-stage', 'db.users.aggregate([{ ', 'db.users.aggregate([\n  {\n    '],
  ['agg-expr', 'db.users.aggregate([{ $group: { t: { ', 'db.users.aggregate([\n  {\n    $group: {\n      t: {\n        '],
  ['method', 'db.users.find({}).', 'db.users\n  .find({})\n  .'],
]

describe('context is invariant to earlier statements', () => {
  for (const [name, caret] of carets) {
    const base = analyzeContext(caret)

    for (const [prefixName, prefix] of prefixes) {
      it(`${name} after ${prefixName}`, () => {
        expect(analyzeContext(prefix + caret)).toEqual(base)
      })
    }
  }
})

describe('context is invariant to line formatting', () => {
  for (const [name, oneLine, multiLine] of multiline) {
    it(name, () => {
      expect(analyzeContext(multiLine)).toEqual(analyzeContext(oneLine))
    })
  }
})
