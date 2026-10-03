# OTTL Functions Catalog

Editor and converter reference for collector-contrib **v0.162.0**. Editors mutate telemetry; converters return values for use in expressions. See the upstream `pkg/ottl/ottlfuncs/README.md` for the authoritative source; since v0.162 the lambda converters are documented in `pkg/ottl/xottl/ottlfuncs/README.md` and `ProfileID` in `pkg/ottl/contexts/xprofile/README.md`.

## Contents

- [Transform-only functions](#transform-processor-only-functions)
- [Editors](#editors-data-manipulation)
- [String](#string-converters), [type conversion](#type-conversion), and [type checking](#type-checking)
- [Lambda converters](#lambda-converters-v0157-alpha), [patterns](#pattern-matching), and [parsing](#data-parsing)
- [Collections](#collections), [date/time](#datetime), [encoding](#hashing--encoding), and [OpenTelemetry-specific](#opentelemetry-specific)
- [Safe function patterns](#function-patterns)

## Transform-processor-only functions

The `transform` processor adds the following functions to the common OTTL
catalog. They are not generally available in other OTTL-consuming components.
The contexts and signatures below are pinned to the released
[v0.162.0 transform processor source](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.162.0/processor/transformprocessor/README.md#supported-functions).

| Context | Signature | Behavior and limits |
|---------|-----------|---------------------|
| `metric` | `convert_sum_to_gauge()` | Sum to Gauge; other metric types are unchanged. The conversion can violate Gauge semantics. |
| `metric` | `convert_gauge_to_sum(aggregation_temporality, is_monotonic)` | Gauge to Sum; temporality is `"delta"` or `"cumulative"`; other metric types are unchanged. The caller owns the resulting Sum semantics. |
| `metric` | `extract_avg_metric(suffix?)` | Creates a Gauge of sum/count per datapoint from Histogram, ExponentialHistogram, or Summary; default suffix `_avg`; skips datapoints with an absent sum or zero count. Added in v0.162. |
| `metric` | `extract_count_metric(is_monotonic, suffix?)` | Creates a Sum from Histogram, ExponentialHistogram, or Summary counts; default suffix `_count`; only creates output when datapoints exist. |
| `metric` | `extract_percentile_metric(percentile, suffix?)` | Creates a Gauge from Histogram or ExponentialHistogram buckets; `0 < percentile < 100`; default suffix `_p{percentile}`; the result is an interpolated estimate. |
| `metric` | `extract_sum_metric(is_monotonic, suffix?)` | Creates a Sum from Histogram, ExponentialHistogram, or Summary sums; default suffix `_sum`; skips datapoints whose sum is absent. |
| `datapoint` | `convert_summary_count_val_to_sum(aggregation_temporality, is_monotonic, suffix?)` | Creates a Sum from a Summary count; temporality is `"delta"` or `"cumulative"`; default suffix `_count`. |
| `metric` | `convert_summary_quantile_val_to_gauge(attribute_key?, suffix?)` | Creates one Gauge datapoint per Summary quantile; defaults are attribute `quantile` and suffix `.quantiles`. |
| `datapoint` | `convert_summary_sum_val_to_sum(aggregation_temporality, is_monotonic, suffix?)` | Creates a Sum from a Summary sum; temporality is `"delta"` or `"cumulative"`; default suffix `_sum`. |
| `metric` | `copy_metric(name?, description?, unit?)` | Appends a full copy, optionally overriding metadata. The copy runs through later metric statements, so guard it with a condition that excludes the copy. |
| `metric` | `scale_metric(factor, unit?)` | Scales Gauge, Sum, Histogram, and Summary values; optionally changes the unit. |
| `metric` | `aggregate_on_attributes(function, attributes?)` | Aggregates Sum, Gauge, Histogram, or ExponentialHistogram datapoints. Functions are `sum`, `max`, `min`, `mean`, `median`, or `count`; histogram types support only `sum`. Omitted attributes retain all keys, while `[]` drops all keys. |
| `metric` | `convert_exponential_histogram_to_histogram(distribution, explicit_bounds)` | Converts ExponentialHistogram to explicit Histogram using `upper`, `midpoint`, `uniform`, or `random`; bounds must be non-empty and strictly increasing. Since v0.162 it converts positive, negative, and zero buckets and preserves the total count. This lossy conversion is not specified by OpenTelemetry. |
| `metric` | `aggregate_on_attribute_value(function, attribute, values, new_value)` | Aggregates selected attribute values for Sum, Gauge, Histogram, or ExponentialHistogram datapoints; histogram types support only `sum`. |
| `datapoint` | `merge_histogram_buckets(target_value, method?)` | Explicit Histograms only. Default `remove_explicit_bound` removes a matching bound; `limit_buckets` requires a positive integer target and reduces resolution. Other metric types are unchanged. |
| `log` | `ParseCEF(target)` | Parses Common Event Format into a map, including an optional syslog prefix and string-valued extensions; malformed or empty input errors. |
| `log` | `ParseCLF(target, format?)` | Parses CLF (`"clf"`, default) or NCSA combined (`"combined"`) text into a map; malformed or empty input errors. |
| `log` | `ParseELF(target)` | Parses a complete W3C Extended Log Format block into directive metadata, fields, and entries; requires a `#Version` directive and `#Fields` before data. Added in v0.158. |
| `log` | `ParseLEEF(target)` | Parses LEEF 1.0/2.0 into a map; malformed or empty input errors; attribute values remain strings. |
| `span` | `set_semconv_span_name(semconv_version, original_span_name_attribute?)` | Derives low-cardinality HTTP, RPC, messaging, or database span names. v0.162 accepts semantic-convention versions 1.37.0 through 1.43.0; unrelated spans are unchanged. |

For full behavior, examples, and edge cases, follow the tag-pinned
[metrics](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.162.0/processor/transformprocessor/README.md#convert_sum_to_gauge),
[logs](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.162.0/processor/transformprocessor/README.md#parsecef), and
[traces](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.162.0/processor/transformprocessor/README.md#set_semconv_span_name)
function sections. The registrations that constrain the contexts are also tag-pinned:
[metric/datapoint](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.162.0/processor/transformprocessor/internal/metrics/functions.go),
[log](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.162.0/processor/transformprocessor/internal/logs/functions.go), and
[span](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.162.0/processor/transformprocessor/internal/traces/functions.go).

## Editors (data manipulation)

Editors are lowercase. Every OTTL statement contains exactly one.

### `set`
```ottl
set(target, value)
set(span.attributes["env"], "production")
set(log.body, Concat([log.severity_text, ": ", log.body.string], ""))
```

`set` passes `nil` directly to the target: a map or slice path is cleared, a `pcommon.Value` (for
example a missing attribute key) is set to an empty value, and scalar or struct-typed targets return
an error. This is the `ottl.set.allowNil` behavior, beta (on by default) in v0.161 and stable in
v0.162; through v0.160 a nil value was a no-op. Guard missing sources with `where source != nil`
when the target must stay unchanged.

### `clear` (v0.160+)
```ottl
clear(target)
clear(span.attributes["http.request.header.authorization"])
```

Sets the target to the empty value for its current type: for example, `""`, `0`, `false`, an empty
`pcommon.Map`, `pcommon.Slice`, or `pcommon.Value`, or a zero TraceID/SpanID/time. For Go maps,
slices, pointers, byte slices, and a current `nil` value, it passes `nil`; whether the path setter
accepts that value is target-specific and may produce an evaluation error.

### `append`
```ottl
append(target, value)
append(target, values = ["tag1", "tag2"])
append(span.attributes["tags"], "processed")
```

If `target` is a scalar, it is converted to a slice first.

### `delete_key` / `delete_matching_keys` / `delete_index`
```ottl
delete_key(span.attributes, "internal.debug")
delete_matching_keys(span.attributes, "(?i).*password.*")
delete_index(log.attributes["tags"], 0)        # v0.145+, one slice index
delete_index(log.attributes["tags"], 0, 3)     # delete indices 0, 1, 2
```

### `keep_keys` / `keep_matching_keys`
```ottl
keep_keys(span.attributes, ["http.method", "http.status_code", "http.route"])
keep_matching_keys(span.attributes, "(?i)^http\\..*")
```

### `flatten`
```ottl
flatten(target)
flatten(target, prefix, depth, resolveConflicts?)   # v0.139+
flatten(log.body)
flatten(span.attributes, "nested", 2)
flatten(log.attributes, resolveConflicts = true)
```

### `limit`
```ottl
limit(target, count, priority_keys[])
limit(span.attributes, 10, ["http.method", "http.status_code"])
```

### `merge_maps`
```ottl
merge_maps(target, source, "insert" | "update" | "upsert")
merge_maps(span.attributes, ParseJSON(span.attributes["meta"]), "upsert")
```

- `insert` — only set keys not already in target
- `update` — only overwrite keys already in target
- `upsert` — both

### `truncate_all`
```ottl
truncate_all(target, max_length)
truncate_all(target, max_length, utf8_safe = false)   # v0.148+
truncate_all(target, max_length, truncation_marker = "(truncated)") # v0.157+
truncate_all(span.attributes, 256)
```

UTF-8 safe by default since v0.148 — truncates at character boundaries, so the result may be slightly shorter than `max_length`. Pass `utf8_safe = false` for the previous byte-level behavior.

The optional `truncation_marker` is included inside `max_length`, so the complete result never exceeds the limit. A marker longer than the limit is an error.

### `stringify_all`
```ottl
stringify_all(target)
stringify_all(log.attributes)
stringify_all(resource.attributes)
```

Converts every non-string value in a map to its string representation. Numbers and booleans become scalar strings, bytes become base64, maps and slices become JSON strings, and empty values become `""`. Added in collector-contrib v0.155.0.

### Dynamic indexing of converter results

Converter results can be indexed with expressions that evaluate to a string or integer, not only literals:

```ottl
Split(log.body.string, ",")[log.attributes["index"]]
ParseJSON(log.body.string)[log.attributes["field_name"]]
```

This was added in collector-contrib v0.155.0. On older collectors, keep converter-result indexes literal, such as `[0]` or `["field"]`.

### `replace_match` / `replace_all_matches`
```ottl
replace_match(target, pattern, replacement, function?, format?)
replace_all_matches(target, pattern, replacement, function?, format?)
replace_match(span.name, "GET ", "")
replace_all_matches(span.attributes, "/user/*/id/*", "/user/{userId}/id/{id}")
```

Glob patterns (not regex). Use `replace_pattern`/`replace_all_patterns` for regex.

### `replace_pattern` / `replace_all_patterns`
```ottl
replace_pattern(target, regex, replacement, function?, format?)
replace_all_patterns(target, "key" | "value", regex, replacement, function?, format?)

# Strip query string
replace_pattern(attributes["http.url"], "\\?.*$", "")

# Redact a token query parameter (YAML form; in a raw OTTL string use ${1})
replace_pattern(attributes["http.url"], "([?&]token=)[^&]+", "$${1}REDACTED")

# Map-wide patterns over keys or values
replace_all_patterns(span.attributes, "value", "\\d{16}", "[REDACTED]")
replace_all_patterns(span.attributes, "key",   "^kube_", "k8s.")
```

**Backreferences in YAML.** OTTL uses `${1}`, `${2}`, … The collector's YAML loader treats `$` as an env-var marker, so a single `$` in YAML produces nothing in the OTTL string. Write `$${1}` in YAML to emit `${1}` in OTTL. Writing `"$1REDACTED"` produces a literal `$1REDACTED` — silent failure.

**RE2 repeat-count limit.** Patterns like `^(.{1024}).*` fail to compile with "invalid repeat count". For length-based truncation, use `Substring` + `Len` instead:

```ottl
set(attributes["db.statement"], Substring(attributes["db.statement"], 0, 1024))
    where attributes["db.statement"] != nil
      and Len(attributes["db.statement"]) > 1024
```

---

## String converters

### `Concat`, `Split`, `Substring`
```ottl
Concat(["user", span.attributes["user.id"], "action"], "-")  # "user-123-action" when user.id == "123"
Split(span.name, "/")                         # ["", "api", "v1", "users"]
Split(span.name, "/")[1]                      # "api"
Substring(span.span_id.string, 0, 8)          # first 8 chars
Substring(target, start, length, utf8_safe?)  # byte offsets; utf8_safe defaults false; added v0.156
Concat(Split(span.attributes["request.id"], "-"), "")   # v0.161+: values may be a slice-valued path or converter
```

With `utf8_safe=true`, a start inside a multi-byte character advances to the next UTF-8 boundary and an end inside one backs up to the previous boundary. The result stays valid UTF-8 but can be shorter than `length` bytes.

### Case
```ottl
ToLowerCase(s) / ToUpperCase(s)
ToCamelCase(s) / ToSnakeCase(s)
ConvertCase(s, "lower" | "upper" | "camel" | "snake")
```

### Trim
```ottl
Trim(s, cutset?)
TrimPrefix(s, prefix)        # v0.124+
TrimSuffix(s, suffix)        # v0.124+
```

### Prefix/suffix tests
```ottl
HasPrefix(s, prefix)         # bool
HasSuffix(s, suffix)         # bool
where HasPrefix(span.name, "internal.")
```

### `Format`
```ottl
Format("user=%s req=%d", [span.attributes["user.id"], span.attributes["request.count"]])
```

---

## Type conversion

```ottl
String(v)                # any -> string
Int(v)                   # any -> int64
Double(v)                # any -> float64
Bool(v)                  # v0.143+; loose coercion (see below)
ParseInt(s, base)        # ParseInt("ff", 16) -> 255
Hex(bytes)               # bytes -> hex string
```

**`Bool` coercion is loose, not Pythonic.** Booleans pass through; any non-zero integer or double is `true`; zero is `false`; strings use Go boolean parsing (`1`, `t`, `T`, `TRUE`, `true`, `True`, and false equivalents). Invalid strings and unsupported types error, while `nil` returns `nil`.

---

## Type checking

```ottl
IsString(v) / IsInt(v) / IsDouble(v) / IsBool(v)
IsList(v) / IsMap(v)
IsEmpty(v)                                               # nil, zero value, or empty collection
IsInCIDR(ip_string, ["10.0.0.0/8", "192.168.0.0/16"])   # v0.146+
```

`IsInCIDR` returns `false` for invalid IP strings — useful in conditions: `where IsInCIDR(attributes["client.ip"], ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"])`.

---

## Lambda converters (v0.157+, alpha)

These converters require the alpha `ottl.functions.enableLambda` feature gate.
For slices, two-argument lambdas receive `(index, value)`; for maps they receive
`(key, value)`. Use `_` for an ignored parameter.

```ottl
All(source, (k, v) => predicate)                 # every element; true for empty input
Any(source, (k, v) => predicate)                 # at least one; false for empty input
Filter(source, (k, v) => predicate)              # filtered slice or map
Find(source, (k, v) => predicate, mapper?)        # first value, mapped value, or nil
MapEach(source, (k, v) => mapped_value)           # transform values
MapKeys(map, (k, v) => mapped_key)                # mapped_key must be a string
Reduce(source, seed, (acc, k, v) => next_acc)     # fold; empty input returns seed
When(() => condition, true_value, false_value)    # select one value
```

Map iteration order is not stable, so order-dependent `Reduce` expressions are
not deterministic for maps. If `MapKeys` creates duplicate keys, which value is
retained is unspecified.

---

## Pattern matching

### `IsMatch`
```ottl
IsMatch(target, pattern)     # bool
where IsMatch(span.name, "^GET /api/.*")
where not IsMatch(span.name, ".*health.*")
```

### `ExtractPatterns`
```ottl
ExtractPatterns(target, "(?P<user>\\w+)/(?P<action>\\w+)")
# Returns map keyed by the named groups.
```

Named capture groups required — unnamed groups are not surfaced as map keys.

### `ExtractGrokPatterns` (v0.130+)
```ottl
ExtractGrokPatterns(log.body, "%{IP:client_ip} %{WORD:method} %{URIPATHPARAM:path}")
ExtractGrokPatterns(log.body, "%{URI}", true)                         # named only
ExtractGrokPatterns(log.body, "%{MYPATTERN}", true,
                   ["MYPATTERN=%{IP}:%{INT}"])                         # custom defs
```

Built-in pattern library covers HTTP, syslog, paths, IPs, dates, etc. Cheaper than expressing the same thing as a raw regex.

---

## Data parsing

```ottl
ParseJSON(s)                                  # JSON string -> map | list
ParseCSV(s, headers, delimiter?, headerDelim?, mode?)
ParseKeyValue(s, kv_delim?, pair_delim?)      # ParseKeyValue("a=1&b=2", "=", "&")
ParseXML(s) / ParseSimplifiedXML(s)
ParseSeverity(value, mapping)                 # map values -> SEVERITY_NUMBER_*
URL(s)                                        # v0.127+; { url.scheme, url.domain, url.port, url.path, … }
UserAgent(s)                                  # v0.134+; { user_agent.name, .version, os.name, os.version, … }
```

`ParseJSON`, `IsString`, and `IsMatch` are available in v0.156 and do not require the v0.157
`ottl.functions.enableLambda` feature gate.

```ottl
# Conditional JSON parsing
set(log.attributes, ParseJSON(log.body.string))
    where IsString(log.body) and IsMatch(log.body.string, "(?s)^\\s*\\{.*\\}\\s*$")

# URL parsing
set(span.attributes["http.host"], URL(span.attributes["http.url"])["url.domain"])
```

### Fallback values
```ottl
Coalesce([span.attributes["user.id"], resource.attributes["user.id"], "fallback"])
```

---

## XML

```ottl
GetXML(target, xpath)                         # select elements
InsertXML(target, xpath, value)
RemoveXML(target, xpath)
ConvertAttributesToElementsXML(target, xpath?)
ConvertTextToElementsXML(target, xpath?, elementName?)
```

---

## Collections

```ottl
Len(target)                                   # works for string, list, map
Keys(map) / Values(map)                       # -> list
ContainsValue(list, value)                    # bool
Index(target, value)                          # v0.126+; -1 if not found
Sort(list, "asc" | "desc")                    # v0.125+
SliceToMap(list, [keyPath]?, [valuePath]?)    # v0.128+; turn objects into a map
```

```ottl
SliceToMap(resource.attributes["items"], ["name"])                     # key by .name
SliceToMap(resource.attributes["items"], ["name"], ["value"])          # explicit value
```

---

## Date/time

```ottl
Now()                                         # current time
Time(s, format, location?, locale?)           # locale added v0.136
FormatTime(t, format)
Duration(s)                                   # "1h30m" -> time.Duration
TruncateTime(t, duration)
Unix(seconds, nanoseconds?)
UnixSeconds(t) / UnixMilli(t) / UnixMicro(t) / UnixNano(t)
Year(t) / Month(t) / Day(t) / Hour(t) / Minute(t) / Second(t) / Nanosecond(t) / Weekday(t)
Hours(d) / Minutes(d) / Seconds(d) / Milliseconds(d) / Microseconds(d) / Nanoseconds(d)
```

---

## Hashing & encoding

```ottl
SHA256(s) / SHA512(s)
SHA1(s) / MD5(s)                              # cryptographically weak; avoid for security
FNV(s)                                        # int64
Murmur3Hash(s) / Murmur3Hash128(s)            # v0.129+; hexadecimal string
XXH3(s) / XXH128(s)                           # v0.135+; hexadecimal string
Decode(value, encoding)                       # v0.141+; "base64", "base64-raw", "base64-url", "base64-raw-url", IANA charsets
Base64Encode(s, variant?)                     # v0.147+; default base64
Hex(bytes)
```

---

## OpenTelemetry-specific

```ottl
SpanID(bytes | hex_string)                    # hex strings accepted v0.142+
TraceID(bytes | hex_string)                   # same
ProfileID(bytes | hex_string)                 # v0.124+
IsRootSpan()                                  # span context only
IsValidLuhn(s)                                # credit card checksum
CommunityID(srcIP, srcPort, dstIP, dstPort, protocol?, seed?)   # v0.131+; network flow hash
ToKeyValueString(map, delim?, pair_delim?, sort?)
```

---

## Identifier generation

```ottl
UUID()
UUIDv7()                                      # v0.138+; time-ordered
```

`UUIDv7` is time-ordered; choose the identifier form according to the consuming system's requirements.

---

## Math

```ottl
Log(value)                                    # natural logarithm
```

OTTL's basic arithmetic operators (`+`, `-`, `*`, `/`) cover most needs; `Log` is the rare named math converter.

---

## Function patterns

### Safe type conversion
```ottl
set(span.attributes["count_int"], Int(span.attributes["count"]))
    where IsString(span.attributes["count"])
      and IsMatch(span.attributes["count"], "^\\d+$")
```

### Chained string operations
```ottl
set(span.attributes["normalized"],
    ToLowerCase(Trim(span.attributes["value"], " ")))
```

### Conditional parsing into cache
```ottl
set(log.cache["parsed"], ParseJSON(log.body.string))
    where IsString(log.body) and IsMatch(log.body.string, "(?s)^\\s*\\{.*\\}\\s*$")
```

Then read from `log.cache["parsed"]` in subsequent statements without paying the parse cost again.

### Hash-based sampling
```ottl
set(span.attributes["sampled"], true)
    where IsMatch(XXH3(span.trace_id.string), "^(0[0-9a-f]|1[0-9])")  # ~10.2%
```
