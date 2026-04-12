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
    label: 'createIndexes',
    detail: '(specs) - creates multiple indexes for the collection',
    snippet: 'createIndexes([$1])$0',
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
  {
    label: 'stats',
    detail: '(scale?) - returns statistics about the collection',
    snippet: 'stats()$0',
  },
  {
    label: 'isCapped',
    detail: '() - returns true if the collection is a capped collection',
    snippet: 'isCapped()$0',
  },
  {
    label: 'dataSize',
    detail: '() - returns the uncompressed size of the collection in bytes',
    snippet: 'dataSize()$0',
  },
  {
    label: 'storageSize',
    detail: '() - returns the allocated storage size of the collection in bytes',
    snippet: 'storageSize()$0',
  },
  {
    label: 'totalIndexSize',
    detail: '() - returns the total size of all indexes on the collection in bytes',
    snippet: 'totalIndexSize()$0',
  },
  {
    label: 'totalSize',
    detail: '() - returns the total storage size of the collection and its indexes in bytes',
    snippet: 'totalSize()$0',
  },
  {
    label: 'getIndexes',
    detail: '() - lists the indexes on the collection (alias for listIndexes)',
    snippet: 'getIndexes()$0',
  },
  {
    label: 'count',
    detail: '(filter?) - counts matching documents (legacy; prefer countDocuments)',
    snippet: 'count({$1})$0',
  },
  {
    label: 'renameCollection',
    detail: '(newName, dropTarget?) - renames the collection',
    snippet: 'renameCollection("$1")$0',
  },
  {
    label: 'validate',
    detail: '(full?) - validates the collection',
    snippet: 'validate()$0',
  },
  {
    label: 'findAndModify',
    detail: '(spec) - finds and modifies a document (legacy; prefer findOneAnd*)',
    snippet: 'findAndModify({$1})$0',
  },
]

export const queryOperators = [
  // Comparison
  { label: '$eq', detail: 'Matches values equal to a value' },
  { label: '$ne', detail: 'Matches values not equal to a value' },
  { label: '$gt', detail: 'Matches values greater than a value' },
  { label: '$gte', detail: 'Matches values greater than or equal to a value' },
  { label: '$lt', detail: 'Matches values less than a value' },
  { label: '$lte', detail: 'Matches values less than or equal to a value' },
  { label: '$in', detail: 'Matches any value in an array' },
  { label: '$nin', detail: 'Matches none of the values in an array' },
  // Logical
  { label: '$and', detail: 'Joins clauses with logical AND' },
  { label: '$or', detail: 'Joins clauses with logical OR' },
  { label: '$nor', detail: 'Joins clauses with logical NOR' },
  { label: '$not', detail: 'Inverts a query expression' },
  // Element
  { label: '$exists', detail: 'Matches documents with the field' },
  { label: '$type', detail: 'Matches documents by BSON type' },
  // Evaluation
  { label: '$regex', detail: 'Matches by regular expression' },
  { label: '$expr', detail: 'Use aggregation expressions in queries' },
  { label: '$mod', detail: 'Matches values where field % divisor == remainder' },
  { label: '$text', detail: 'Full-text search on indexed fields' },
  { label: '$where', detail: 'Match with a JavaScript expression' },
  { label: '$jsonSchema', detail: 'Validate against a JSON Schema' },
  // Array
  { label: '$all', detail: 'Matches arrays containing all elements' },
  { label: '$elemMatch', detail: 'Matches array elements' },
  { label: '$size', detail: 'Matches arrays by size' },
  // Geospatial
  { label: '$geoWithin', detail: 'Matches within a GeoJSON geometry' },
  { label: '$geoIntersects', detail: 'Matches geometries that intersect' },
  { label: '$near', detail: 'Returns documents near a point' },
  { label: '$nearSphere', detail: 'Returns documents near a point on a sphere' },
  // Bitwise
  { label: '$bitsAllSet', detail: 'Matches where all bit positions are set' },
  { label: '$bitsAnySet', detail: 'Matches where any bit position is set' },
  { label: '$bitsAllClear', detail: 'Matches where all bit positions are clear' },
  { label: '$bitsAnyClear', detail: 'Matches where any bit position is clear' },
]

export const cursorMethods = [
  { label: 'limit', detail: '(n) - Limit number of results', snippet: 'limit($1)$0' },
  { label: 'skip', detail: '(n) - Skip first n results', snippet: 'skip($1)$0' },
  { label: 'sort', detail: '(sortSpec) - Sort results, e.g. { field: 1 }', snippet: 'sort({$1})$0' },
  { label: 'toArray', detail: '() - Convert cursor to array', snippet: 'toArray()$0' },
  { label: 'count', detail: '() - Count the results', snippet: 'count()$0' },
  { label: 'forEach', detail: '(fn) - Iterate with a callback', snippet: 'forEach($1)$0' },
  { label: 'pretty', detail: '() - Pretty print the output', snippet: 'pretty()$0' },
  { label: 'explain', detail: '(verbosity?) - Show query execution plan', snippet: 'explain()$0' },
  { label: 'hint', detail: '(index) - Force a specific index', snippet: 'hint($1)$0' },
  { label: 'batchSize', detail: '(n) - Set cursor batch size', snippet: 'batchSize($1)$0' },
  { label: 'maxTimeMS', detail: '(ms) - Set max execution time', snippet: 'maxTimeMS($1)$0' },
  { label: 'collation', detail: '(spec) - Set collation rules', snippet: 'collation({$1})$0' },
  { label: 'comment', detail: '(str) - Add a comment to the query', snippet: 'comment("$1")$0' },
  { label: 'map', detail: '(fn) - Transform each document', snippet: 'map($1)$0' },
  { label: 'hasNext', detail: '() - Check if cursor has more documents', snippet: 'hasNext()$0' },
  { label: 'next', detail: '() - Get next document from cursor', snippet: 'next()$0' },
]

export const aggStages = [
  { label: '$addFields', detail: 'Add new fields' },
  { label: '$bucket', detail: 'Categorize into buckets' },
  { label: '$bucketAuto', detail: 'Automatically bucket documents' },
  { label: '$changeStream', detail: 'Watch for changes' },
  { label: '$count', detail: 'Count documents' },
  { label: '$densify', detail: 'Fill gaps in time series or numeric data' },
  { label: '$documents', detail: 'Return literal documents' },
  { label: '$facet', detail: 'Multiple aggregation pipelines' },
  { label: '$fill', detail: 'Fill missing field values' },
  { label: '$geoNear', detail: 'Return documents by proximity to a point' },
  { label: '$graphLookup', detail: 'Recursive lookup across a collection' },
  { label: '$group', detail: 'Group documents by expression' },
  { label: '$limit', detail: 'Limit number of documents' },
  { label: '$lookup', detail: 'Join with another collection' },
  { label: '$match', detail: 'Filter documents' },
  { label: '$merge', detail: 'Merge results into collection' },
  { label: '$out', detail: 'Write results to collection' },
  { label: '$project', detail: 'Reshape documents' },
  { label: '$redact', detail: 'Restrict document content by access level' },
  { label: '$replaceRoot', detail: 'Replace root document' },
  { label: '$replaceWith', detail: 'Replace root document (alias)' },
  { label: '$sample', detail: 'Randomly select documents' },
  { label: '$search', detail: 'Atlas full-text search' },
  { label: '$searchMeta', detail: 'Atlas search metadata' },
  { label: '$set', detail: 'Add or overwrite fields' },
  { label: '$setWindowFields', detail: 'Window functions over sorted partitions' },
  { label: '$skip', detail: 'Skip documents' },
  { label: '$sort', detail: 'Sort documents' },
  { label: '$sortByCount', detail: 'Group and sort by count' },
  { label: '$unionWith', detail: 'Combine results from another collection' },
  { label: '$unset', detail: 'Remove fields' },
  { label: '$unwind', detail: 'Deconstruct array field' },
  { label: '$vectorSearch', detail: 'Atlas vector similarity search' },
]

export const aggExpressions = [
  // Accumulators
  { label: '$sum', detail: 'Sum of numeric values' },
  { label: '$avg', detail: 'Average of numeric values' },
  { label: '$min', detail: 'Minimum value' },
  { label: '$max', detail: 'Maximum value' },
  { label: '$first', detail: 'First value in a group' },
  { label: '$last', detail: 'Last value in a group' },
  { label: '$push', detail: 'Append values to an array' },
  { label: '$addToSet', detail: 'Append unique values to an array' },
  { label: '$count', detail: 'Count of documents in a group' },
  // Arithmetic
  { label: '$add', detail: 'Add numbers or dates' },
  { label: '$subtract', detail: 'Subtract numbers or dates' },
  { label: '$multiply', detail: 'Multiply numbers' },
  { label: '$divide', detail: 'Divide numbers' },
  { label: '$mod', detail: 'Remainder of division' },
  { label: '$abs', detail: 'Absolute value' },
  { label: '$ceil', detail: 'Round up to nearest integer' },
  { label: '$floor', detail: 'Round down to nearest integer' },
  { label: '$round', detail: 'Round to a specified decimal place' },
  // String
  { label: '$concat', detail: 'Concatenate strings' },
  { label: '$substr', detail: 'Substring by byte index' },
  { label: '$substrCP', detail: 'Substring by code point index' },
  { label: '$toLower', detail: 'Convert to lowercase' },
  { label: '$toUpper', detail: 'Convert to uppercase' },
  { label: '$trim', detail: 'Trim whitespace or characters' },
  { label: '$split', detail: 'Split string by delimiter' },
  { label: '$regexMatch', detail: 'Test string against regex' },
  { label: '$regexFind', detail: 'Find first regex match' },
  { label: '$replaceOne', detail: 'Replace first occurrence of a string' },
  { label: '$replaceAll', detail: 'Replace all occurrences of a string' },
  // Array
  { label: '$arrayElemAt', detail: 'Element at array index' },
  { label: '$filter', detail: 'Filter array by condition' },
  { label: '$map', detail: 'Transform each array element' },
  { label: '$reduce', detail: 'Reduce array to a single value' },
  { label: '$slice', detail: 'Subset of an array' },
  { label: '$size', detail: 'Number of elements in an array' },
  { label: '$concatArrays', detail: 'Concatenate arrays' },
  { label: '$in', detail: 'Check if value is in an array' },
  { label: '$isArray', detail: 'Check if value is an array' },
  { label: '$reverseArray', detail: 'Reverse an array' },
  { label: '$arrayToObject', detail: 'Convert key-value array to object' },
  // Date
  { label: '$year', detail: 'Year from a date' },
  { label: '$month', detail: 'Month from a date (1-12)' },
  { label: '$dayOfMonth', detail: 'Day of month from a date (1-31)' },
  { label: '$hour', detail: 'Hour from a date (0-23)' },
  { label: '$minute', detail: 'Minute from a date (0-59)' },
  { label: '$second', detail: 'Second from a date (0-59)' },
  { label: '$dateToString', detail: 'Format date as string' },
  { label: '$dateFromString', detail: 'Parse string to date' },
  { label: '$dateAdd', detail: 'Add a duration to a date' },
  { label: '$dateDiff', detail: 'Difference between two dates' },
  // Conditional
  { label: '$cond', detail: 'If-then-else expression' },
  { label: '$ifNull', detail: 'Coalesce null values' },
  { label: '$switch', detail: 'Multi-branch conditional' },
  // Type
  { label: '$type', detail: 'BSON type of a value' },
  { label: '$convert', detail: 'Convert value to a specified type' },
  { label: '$toInt', detail: 'Convert to integer' },
  { label: '$toString', detail: 'Convert to string' },
  { label: '$toDouble', detail: 'Convert to double' },
  { label: '$toBool', detail: 'Convert to boolean' },
  { label: '$toObjectId', detail: 'Convert to ObjectId' },
  { label: '$toDate', detail: 'Convert to date' },
  // Object
  { label: '$mergeObjects', detail: 'Merge objects into one' },
  { label: '$objectToArray', detail: 'Convert object to key-value array' },
  // Comparison
  { label: '$eq', detail: 'Equal comparison' },
  { label: '$ne', detail: 'Not equal comparison' },
  { label: '$gt', detail: 'Greater than comparison' },
  { label: '$gte', detail: 'Greater than or equal comparison' },
  { label: '$lt', detail: 'Less than comparison' },
  { label: '$lte', detail: 'Less than or equal comparison' },
  { label: '$cmp', detail: 'Compare two values (-1, 0, 1)' },
  // Literal
  { label: '$literal', detail: 'Return a value without parsing' },
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

export const ejsonMethods = [
  {
    label: 'stringify',
    detail: '(value, replacer?, space?) - Convert to Extended JSON string',
    snippet: 'stringify($1)$0',
  },
  {
    label: 'parse',
    detail: '(str) - Parse an Extended JSON string',
    snippet: "parse('$1')$0",
  },
  {
    label: 'serialize',
    detail: '(value) - Convert to Extended JSON object representation',
    snippet: 'serialize($1)$0',
  },
  {
    label: 'deserialize',
    detail: '(value) - Convert Extended JSON object to BSON types',
    snippet: 'deserialize($1)$0',
  },
]

export const dbMethods = [
  {
    label: 'runCommand',
    detail: '(command) - Run a database command',
    snippet: 'runCommand({$1})$0',
  },
  {
    label: 'adminCommand',
    detail: '(command) - Run an admin database command',
    snippet: 'adminCommand({$1})$0',
  },
  {
    label: 'getName',
    detail: '() - Returns the current database name',
    snippet: 'getName()$0',
  },
  {
    label: 'getCollection',
    detail: '(name) - Returns a collection object',
    snippet: "getCollection('$1')$0",
  },
  {
    label: 'getCollectionNames',
    detail: '(filter?) - List collection names in the database',
    snippet: 'getCollectionNames()$0',
  },
  {
    label: 'getCollectionInfos',
    detail: '(filter?) - List collection info objects',
    snippet: 'getCollectionInfos()$0',
  },
  {
    label: 'createCollection',
    detail: '(name) - Create a new collection',
    snippet: "createCollection('$1')$0",
  },
  {
    label: 'dropDatabase',
    detail: '() - Drop the current database',
    snippet: 'dropDatabase()$0',
  },
  {
    label: 'stats',
    detail: '() - Database statistics',
    snippet: 'stats()$0',
  },
  {
    label: 'version',
    detail: '() - MongoDB server version',
    snippet: 'version()$0',
  },
  {
    label: 'getSiblingDB',
    detail: '(name) - Switch to another database',
    snippet: "getSiblingDB('$1')$0",
  },
  {
    label: 'getMongo',
    detail: '() - Returns the connection object',
    snippet: 'getMongo()$0',
  },
  {
    label: 'aggregate',
    detail: '(pipeline) - Run a database-level aggregation',
    snippet: 'aggregate([$1])$0',
  },
]
