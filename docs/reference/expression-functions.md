# Expression Functions Reference

LinkFlow AI provides a powerful expression system for data transformation and dynamic values.

## Expression Syntax

Expressions are enclosed in double curly braces:

```
{{ expression }}
```

## Accessing Data

### Input Data

Access data from the previous node:

```javascript
{{ $input }}                    // Entire input object
{{ $input.fieldName }}          // Specific field
{{ $input.nested.field }}       // Nested field
{{ $input.array[0] }}           // Array element
{{ $input.array[0].name }}      // Array element property
```

### Node Data

Access data from specific nodes:

```javascript
{{ $node.nodeName.data }}           // Node output data
{{ $node.nodeName.data.field }}     // Specific field
{{ $node["Node Name"].data }}       // Node with spaces in name
```

### Context Variables

```javascript
{{ $executionId }}              // Current execution ID
{{ $workflowId }}               // Current workflow ID
{{ $nodeId }}                   // Current node ID
{{ $timestamp }}                // Current timestamp
{{ $env.VARIABLE_NAME }}        // Environment variable
```

### Loop Variables

Inside loop nodes:

```javascript
{{ $item }}                     // Current item
{{ $index }}                    // Current index (0-based)
{{ $first }}                    // Is first item (boolean)
{{ $last }}                     // Is last item (boolean)
```

## String Functions

### $uppercase(string)
Convert to uppercase.
```javascript
{{ $uppercase("hello") }}       // "HELLO"
{{ $uppercase($input.name) }}
```

### $lowercase(string)
Convert to lowercase.
```javascript
{{ $lowercase("HELLO") }}       // "hello"
```

### $capitalize(string)
Capitalize first letter.
```javascript
{{ $capitalize("hello world") }} // "Hello world"
```

### $trim(string)
Remove leading/trailing whitespace.
```javascript
{{ $trim("  hello  ") }}        // "hello"
```

### $split(string, separator)
Split string into array.
```javascript
{{ $split("a,b,c", ",") }}      // ["a", "b", "c"]
```

### $join(array, separator)
Join array into string.
```javascript
{{ $join(["a", "b", "c"], "-") }} // "a-b-c"
```

### $replace(string, search, replace)
Replace occurrences.
```javascript
{{ $replace("hello world", "world", "there") }} // "hello there"
```

### $substring(string, start, end?)
Extract substring.
```javascript
{{ $substring("hello", 0, 3) }} // "hel"
{{ $substring("hello", 2) }}    // "llo"
```

### $length(string|array)
Get length.
```javascript
{{ $length("hello") }}          // 5
{{ $length($input.items) }}     // array length
```

### $contains(string, search)
Check if contains substring.
```javascript
{{ $contains("hello world", "world") }} // true
```

### $startsWith(string, prefix)
Check if starts with prefix.
```javascript
{{ $startsWith("hello", "he") }} // true
```

### $endsWith(string, suffix)
Check if ends with suffix.
```javascript
{{ $endsWith("hello", "lo") }}  // true
```

### $padStart(string, length, char)
Pad string at start.
```javascript
{{ $padStart("5", 3, "0") }}    // "005"
```

### $padEnd(string, length, char)
Pad string at end.
```javascript
{{ $padEnd("5", 3, "0") }}      // "500"
```

## Number Functions

### $round(number, decimals?)
Round number.
```javascript
{{ $round(3.14159, 2) }}        // 3.14
{{ $round(3.5) }}               // 4
```

### $floor(number)
Round down.
```javascript
{{ $floor(3.9) }}               // 3
```

### $ceil(number)
Round up.
```javascript
{{ $ceil(3.1) }}                // 4
```

### $abs(number)
Absolute value.
```javascript
{{ $abs(-5) }}                  // 5
```

### $min(numbers...)
Minimum value.
```javascript
{{ $min(1, 2, 3) }}             // 1
{{ $min($input.values) }}       // min of array
```

### $max(numbers...)
Maximum value.
```javascript
{{ $max(1, 2, 3) }}             // 3
```

### $sum(array)
Sum of array.
```javascript
{{ $sum([1, 2, 3, 4]) }}        // 10
{{ $sum($input.amounts) }}
```

### $avg(array)
Average of array.
```javascript
{{ $avg([1, 2, 3, 4]) }}        // 2.5
```

### $random()
Random number 0-1.
```javascript
{{ $random() }}                 // 0.123456...
```

### $randomInt(min, max)
Random integer in range.
```javascript
{{ $randomInt(1, 100) }}        // 42
```

## Date/Time Functions

### $now()
Current timestamp (ISO 8601).
```javascript
{{ $now() }}                    // "2024-12-19T12:00:00.000Z"
```

### $today()
Current date (YYYY-MM-DD).
```javascript
{{ $today() }}                  // "2024-12-19"
```

### $timestamp()
Current Unix timestamp (ms).
```javascript
{{ $timestamp() }}              // 1703001600000
```

### $formatDate(date, format)
Format date string.
```javascript
{{ $formatDate($input.date, "YYYY-MM-DD") }}
{{ $formatDate($now(), "MMM D, YYYY") }}  // "Dec 19, 2024"
```

**Format Tokens:**
| Token | Description | Example |
|-------|-------------|---------|
| YYYY | 4-digit year | 2024 |
| YY | 2-digit year | 24 |
| MM | 2-digit month | 12 |
| M | Month | 12 |
| MMM | Short month | Dec |
| MMMM | Full month | December |
| DD | 2-digit day | 19 |
| D | Day | 19 |
| HH | 24-hour hour | 14 |
| hh | 12-hour hour | 02 |
| mm | Minutes | 30 |
| ss | Seconds | 45 |
| A | AM/PM | PM |

### $parseDate(string, format?)
Parse date string.
```javascript
{{ $parseDate("2024-12-19") }}
{{ $parseDate("12/19/2024", "MM/DD/YYYY") }}
```

### $addDays(date, days)
Add days to date.
```javascript
{{ $addDays($now(), 7) }}       // 7 days from now
{{ $addDays($input.date, -1) }} // Yesterday
```

### $addHours(date, hours)
Add hours to date.
```javascript
{{ $addHours($now(), 2) }}
```

### $diffDays(date1, date2)
Difference in days.
```javascript
{{ $diffDays($input.endDate, $input.startDate) }}
```

### $diffHours(date1, date2)
Difference in hours.
```javascript
{{ $diffHours($input.end, $input.start) }}
```

## Array Functions

### $first(array)
Get first element.
```javascript
{{ $first($input.items) }}
```

### $last(array)
Get last element.
```javascript
{{ $last($input.items) }}
```

### $nth(array, index)
Get element at index.
```javascript
{{ $nth($input.items, 2) }}     // Third element
```

### $slice(array, start, end?)
Extract portion of array.
```javascript
{{ $slice($input.items, 0, 5) }} // First 5 elements
```

### $reverse(array)
Reverse array.
```javascript
{{ $reverse([1, 2, 3]) }}       // [3, 2, 1]
```

### $sort(array, key?)
Sort array.
```javascript
{{ $sort([3, 1, 2]) }}          // [1, 2, 3]
{{ $sort($input.users, "name") }} // Sort by name
```

### $filter(array, expression)
Filter array.
```javascript
{{ $filter($input.items, "item.active == true") }}
```

### $map(array, expression)
Transform array.
```javascript
{{ $map($input.users, "item.name") }}  // Extract names
```

### $find(array, expression)
Find first matching element.
```javascript
{{ $find($input.users, "item.id == 123") }}
```

### $unique(array)
Remove duplicates.
```javascript
{{ $unique([1, 2, 2, 3]) }}     // [1, 2, 3]
```

### $flatten(array)
Flatten nested arrays.
```javascript
{{ $flatten([[1, 2], [3, 4]]) }} // [1, 2, 3, 4]
```

### $groupBy(array, key)
Group by property.
```javascript
{{ $groupBy($input.orders, "status") }}
```

## Object Functions

### $keys(object)
Get object keys.
```javascript
{{ $keys($input.data) }}        // ["key1", "key2", ...]
```

### $values(object)
Get object values.
```javascript
{{ $values($input.data) }}
```

### $entries(object)
Get key-value pairs.
```javascript
{{ $entries($input.data) }}     // [["key", "value"], ...]
```

### $merge(objects...)
Merge objects.
```javascript
{{ $merge($input.base, $input.override) }}
```

### $pick(object, keys)
Pick specific keys.
```javascript
{{ $pick($input.user, ["name", "email"]) }}
```

### $omit(object, keys)
Omit specific keys.
```javascript
{{ $omit($input.user, ["password", "secret"]) }}
```

### $get(object, path, default?)
Get nested value safely.
```javascript
{{ $get($input, "user.address.city", "Unknown") }}
```

### $set(object, path, value)
Set nested value.
```javascript
{{ $set($input, "user.verified", true) }}
```

## Type Functions

### $typeof(value)
Get type of value.
```javascript
{{ $typeof($input.value) }}     // "string", "number", "object", etc.
```

### $isNull(value)
Check if null/undefined.
```javascript
{{ $isNull($input.value) }}
```

### $isEmpty(value)
Check if empty.
```javascript
{{ $isEmpty($input.array) }}    // true if []
{{ $isEmpty($input.string) }}   // true if ""
{{ $isEmpty($input.object) }}   // true if {}
```

### $toNumber(value)
Convert to number.
```javascript
{{ $toNumber("42") }}           // 42
```

### $toString(value)
Convert to string.
```javascript
{{ $toString(42) }}             // "42"
```

### $toBoolean(value)
Convert to boolean.
```javascript
{{ $toBoolean("true") }}        // true
{{ $toBoolean(1) }}             // true
```

### $toJson(value)
Convert to JSON string.
```javascript
{{ $toJson($input.data) }}
```

### $fromJson(string)
Parse JSON string.
```javascript
{{ $fromJson($input.jsonString) }}
```

## Encoding Functions

### $base64Encode(string)
Base64 encode.
```javascript
{{ $base64Encode("hello") }}    // "aGVsbG8="
```

### $base64Decode(string)
Base64 decode.
```javascript
{{ $base64Decode("aGVsbG8=") }} // "hello"
```

### $urlEncode(string)
URL encode.
```javascript
{{ $urlEncode("hello world") }} // "hello%20world"
```

### $urlDecode(string)
URL decode.
```javascript
{{ $urlDecode("hello%20world") }} // "hello world"
```

### $md5(string)
MD5 hash.
```javascript
{{ $md5("hello") }}
```

### $sha256(string)
SHA-256 hash.
```javascript
{{ $sha256("hello") }}
```

## Conditional Functions

### $if(condition, trueValue, falseValue)
Conditional value.
```javascript
{{ $if($input.age >= 18, "adult", "minor") }}
```

### $coalesce(values...)
First non-null value.
```javascript
{{ $coalesce($input.nickname, $input.name, "Anonymous") }}
```

### $default(value, defaultValue)
Default if null/undefined.
```javascript
{{ $default($input.count, 0) }}
```

## Utility Functions

### $uuid()
Generate UUID.
```javascript
{{ $uuid() }}                   // "550e8400-e29b-41d4-a716-446655440000"
```

### $slug(string)
Generate URL slug.
```javascript
{{ $slug("Hello World!") }}     // "hello-world"
```

### $escape(string)
HTML escape.
```javascript
{{ $escape("<script>") }}       // "&lt;script&gt;"
```

### $unescape(string)
HTML unescape.
```javascript
{{ $unescape("&lt;script&gt;") }} // "<script>"
```

## Examples

### Complex Expression
```javascript
{{
  $if(
    $input.items.length > 0,
    $join($map($filter($input.items, "item.active"), "item.name"), ", "),
    "No active items"
  )
}}
```

### Nested Data Access
```javascript
{{
  $get($node.httpRequest.data, "response.data.users[0].email", "no-email@example.com")
}}
```

### Date Formatting
```javascript
{{
  "Report generated on " + $formatDate($now(), "MMMM D, YYYY") + " at " + $formatDate($now(), "h:mm A")
}}
```

## Next Steps

- [Node Types Reference](node-types.md)
- [Workflows API](../api/workflows.md)
- [Quick Start Guide](../getting-started/quickstart.md)
