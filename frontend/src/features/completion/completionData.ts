export const mongoMethods = [
  { label: 'find', detail: '(filter?, projection?) - Query documents', snippet: 'find({$1})$0' },
  { label: 'findOne', detail: '(filter?) - Find a single document', snippet: 'findOne({$1})$0' },
  { label: 'insertOne', detail: '(document) - Insert a document', snippet: 'insertOne({$1})$0' },
  {
    label: 'insertMany',
    detail: '(documents) - Insert multiple documents',
    snippet: 'insertMany([$1])$0',
  },
  {
    label: 'updateOne',
    detail: '(filter, update) - Update a document',
    snippet: 'updateOne({$1}, {$2})$0',
  },
  {
    label: 'updateMany',
    detail: '(filter, update) - Update multiple documents',
    snippet: 'updateMany({$1}, {$2})$0',
  },
  { label: 'deleteOne', detail: '(filter) - Delete a document', snippet: 'deleteOne({$1})$0' },
  {
    label: 'deleteMany',
    detail: '(filter) - Delete multiple documents',
    snippet: 'deleteMany({$1})$0',
  },
  {
    label: 'replaceOne',
    detail: '(filter, replacement) - Replace a document',
    snippet: 'replaceOne({$1}, {$2})$0',
  },
  {
    label: 'countDocuments',
    detail: '(filter?) - Count matching documents',
    snippet: 'countDocuments({$1})$0',
  },
  {
    label: 'aggregate',
    detail: '(pipeline) - Run aggregation pipeline',
    snippet: 'aggregate([$1])$0',
  },
  {
    label: 'distinct',
    detail: '(field, filter?) - find distinct values for a field',
    snippet: 'distinct("$1")$0',
  },
  {
    label: 'findOneAndDelete',
    detail: '(filter) - find and delete a single document',
    snippet: 'findOneAndDelete({$1})$0',
  },
  {
    label: 'findOneAndReplace',
    detail: '(filter, replacement) - find and replace a single document',
    snippet: 'findOneAndReplace({$1}, {$2})$0',
  },
  {
    label: 'findOneAndUpdate',
    detail: '(filter, update) - find and update a single document',
    snippet: 'findOneAndUpdate({$1}, {$2})$0',
  },
  {
    label: 'estimatedDocumentCount',
    detail: '() - gets estimated number of documents from metadata',
    snippet: 'estimatedDocumentCount()$0',
  },
  {
    label: 'drop',
    detail: '() - removes the collection from the database',
    snippet: 'drop()$0',
  },
  {
    label: 'createIndex',
    detail: '(keys, options?) - creates an index for the collection',
    snippet: 'createIndex({$1})$0',
  },
  {
    label: 'dropIndex',
    detail: '(name) - deletes an index from the collection',
    snippet: 'dropIndex($1)$0',
  },
  {
    label: 'dropIndexes',
    detail: '() - deletes all indexes from the collection',
    snippet: 'dropIndexes()$0',
  },
  {
    label: 'listIndexes',
    detail: '() - lists the indexes on the collection',
    snippet: 'listIndexes()$0',
  },
  {
    label: 'bulkWrite',
    detail: '(operations) - executes multiple write operations',
    snippet: 'bulkWrite([$1])$0',
  },
]

export const queryOperators = [
  { label: '$eq', detail: 'Matches values equal to a value' },
  { label: '$ne', detail: 'Matches values not equal to a value' },
  { label: '$gt', detail: 'Matches values greater than a value' },
  { label: '$gte', detail: 'Matches values greater than or equal to a value' },
  { label: '$lt', detail: 'Matches values less than a value' },
  { label: '$lte', detail: 'Matches values less than or equal to a value' },
  { label: '$in', detail: 'Matches any value in an array' },
  { label: '$nin', detail: 'Matches none of the values in an array' },
  { label: '$exists', detail: 'Matches documents with the field' },
  { label: '$type', detail: 'Matches documents by BSON type' },
  { label: '$regex', detail: 'Matches by regular expression' },
  { label: '$not', detail: 'Inverts a query expression' },
  { label: '$and', detail: 'Joins clauses with logical AND' },
  { label: '$or', detail: 'Joins clauses with logical OR' },
  { label: '$nor', detail: 'Joins clauses with logical NOR' },
  { label: '$elemMatch', detail: 'Matches array elements' },
  { label: '$size', detail: 'Matches arrays by size' },
  { label: '$all', detail: 'Matches arrays containing all elements' },
]

export const cursorMethods = [
  { label: 'limit', detail: '(n) - Limit number of results' },
  { label: 'skip', detail: '(n) - Skip first n results' },
  { label: 'sort', detail: '(sortSpec) - Sort results, e.g. { field: 1 }' },
  { label: 'toArray', detail: '() - Convert cursor to array' },
  { label: 'count', detail: '() - Count the results' },
  { label: 'forEach', detail: '(fn) - Iterate with a callback' },
  { label: 'pretty', detail: '() - Pretty print the output' },
  { label: 'explain', detail: '(verbosity?) - Show query execution plan' },
  { label: 'hint', detail: '(index) - Force a specific index' },
  { label: 'batchSize', detail: '(n) - Set cursor batch size' },
  { label: 'maxTimeMS', detail: '(ms) - Set max execution time' },
  { label: 'collation', detail: '(spec) - Set collation rules' },
  { label: 'comment', detail: '(str) - Add a comment to the query' },
  { label: 'map', detail: '(fn) - Transform each document' },
  { label: 'hasNext', detail: '() - Check if cursor has more documents' },
  { label: 'next', detail: '() - Get next document from cursor' },
]

export const aggStages = [
  { label: '$match', detail: 'Filter documents' },
  { label: '$group', detail: 'Group documents by expression' },
  { label: '$project', detail: 'Reshape documents' },
  { label: '$sort', detail: 'Sort documents' },
  { label: '$limit', detail: 'Limit number of documents' },
  { label: '$skip', detail: 'Skip documents' },
  { label: '$unwind', detail: 'Deconstruct array field' },
  { label: '$lookup', detail: 'Join with another collection' },
  { label: '$addFields', detail: 'Add new fields' },
  { label: '$set', detail: 'Add or overwrite fields' },
  { label: '$unset', detail: 'Remove fields' },
  { label: '$replaceRoot', detail: 'Replace root document' },
  { label: '$count', detail: 'Count documents' },
  { label: '$out', detail: 'Write results to collection' },
  { label: '$merge', detail: 'Merge results into collection' },
  { label: '$facet', detail: 'Multiple aggregation pipelines' },
  { label: '$bucket', detail: 'Categorize into buckets' },
  { label: '$sample', detail: 'Randomly select documents' },
]

export const updateOperators = [
  // fields
  { label: '$currentDate', detail: 'Set a field to the current date' },
  { label: '$inc', detail: 'Increment a field by a value' },
  { label: '$min', detail: 'Update field to supplied value if less than existing.' },
  { label: '$max', detail: 'Update field to supplied value if greater than existing.' },
  { label: '$mul', detail: 'Multiply a field by a value' },
  { label: '$rename', detail: 'Rename a field' },
  { label: '$set', detail: 'Set a field to a value' },
  { label: '$setOnInsert', detail: "Set a field only if it's new" },
  { label: '$unset', detail: 'Remove a field' },
  // arrays
  { label: '$addToSet', detail: "Add a value to an array if it doesn't exist" },
  { label: '$pop', detail: 'Remove the last or first element from an array' },
  { label: '$pull', detail: 'Remove a value from an array' },
  { label: '$push', detail: 'Push a value to an array' },
  { label: '$pullAll', detail: 'Removes all matching values from an array' },
  // modifiers
  {
    label: '$each',
    detail: 'Modifies $push and $addToSet to append multiple items for array updates',
  },
  { label: '$position', detail: 'Specify insertion index for $push' },
  { label: '$slice', detail: 'Limit the array size after $push' },
  { label: '$sort', detail: 'Sort the array after pushing' },
  // bitwise
  { label: '$bit', detail: 'Performs bitwise AND, OR and XOR updates of integer values' },
]
