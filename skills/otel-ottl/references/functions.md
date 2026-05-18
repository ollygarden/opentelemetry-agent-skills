# OTTL Functions Catalog

Editor and converter reference for collector-contrib **v0.149.0**. Editors mutate telemetry; converters return values for use in expressions. See the upstream `pkg/ottl/ottlfuncs/README.md` for the authoritative source.

## Editors (data manipulation)

Editors are lowercase. Every OTTL statement contains exactly one.

### `set`
```ottl
set(target, value)
set(span.attributes["env"], "production")
set(log.body, Concat([log.severity_text, ": ", log.body.string], ""))
```

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
delete_index(log.attributes["tags"], 0)        # v0.145+, slice index
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
truncate_all(span.attributes, 256)
```

UTF-8 safe by default since v0.148 — truncates at character boundaries, so the result may be slightly shorter than `max_length`. Pass `utf8_safe = false` for the previous byte-level behavior.

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
Concat(["user", user_id, "action"], "-")     # "user-123-action"
Split(path, "/")                              # ["", "api", "v1", "users"]
Split(path, "/")[1]                           # "api"
Substring(span.span_id.string, 0, 8)          # first 8 chars
```

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

### `Replace` (literal, non-regex)
```ottl
Replace(target, old, new, count?)
Replace(log.body.string, "ERROR", "ERR", 1)
```

### `Format`
```ottl
Format("user=%s req=%d", [user, count])      # printf-style
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

**`Bool` coercion is loose, not Pythonic.** `Bool("true")`, `Bool("1")`, `Bool(1)` → `true`; `Bool("false")`, `Bool("0")`, `Bool(0)`, `Bool(0.0)` → `false`. Other values error. Don't assume `Bool("yes")` or `Bool("anything-non-empty")` returns true.

---

## Type checking

```ottl
IsString(v) / IsInt(v) / IsDouble(v) / IsBool(v)
IsList(v) / IsMap(v)
IsInCIDR(ip_string, ["10.0.0.0/8", "192.168.0.0/16"])   # v0.146+
```

`IsInCIDR` returns `false` for invalid IP strings — useful in conditions: `where IsInCIDR(attributes["client.ip"], ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"])`.

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
ParseKeyValue(s, pair_delim?, kv_delim?)      # ParseKeyValue("a=1&b=2", "&", "=")
ParseXML(s) / ParseSimplifiedXML(s)
ParseSeverity(value, mapping)                 # map values -> SEVERITY_NUMBER_*
URL(s)                                        # v0.127+; { scheme, domain, port, path, query, fragment, … }
UserAgent(s)                                  # v0.134+; { user_agent.name, .version, os.name, os.version, … }
```

```ottl
# Conditional JSON parsing
set(log.attributes, ParseJSON(log.body.string))
    where IsString(log.body) and IsMatch(log.body.string, "^\\s*\\{.*\\}\\s*$")

# URL parsing
set(span.attributes["http.host"], URL(span.attributes["http.url"])["domain"])
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
SHA256(v) / SHA512(v)
SHA1(v) / MD5(v)                              # cryptographically weak; avoid for security
FNV(v)                                        # int64
Murmur3Hash(v) / Murmur3Hash128(v)            # v0.129+
XXH3(v) / XXH128(v)                           # v0.135+; very fast
Decode(value, encoding)                       # v0.141+; "base64", "base64-raw", "base64-url", "base64-raw-url", IANA charsets
Base64Encode(v)                               # v0.147+
Base64Decode(v)                               # DEPRECATED — use Decode(v, "base64")
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

`UUIDv7` is preferable to `UUID()` when the resulting value is used as a primary key or sort key, because v7 is monotonic-ish and storage-friendly.

---

## Math

```ottl
Log(value)                                    # natural logarithm
```

OTTL's basic arithmetic operators (`+`, `-`, `*`, `/`) cover most needs; `Log` is the rare named math converter.

---

## Utility

```ottl
UUID() / UUIDv7()
```

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
    where IsString(log.body) and IsMatch(log.body.string, "^\\s*\\{.*\\}\\s*$")
```

Then read from `log.cache["parsed"]` in subsequent statements without paying the parse cost again.

### Hash-based sampling
```ottl
set(span.attributes["sampled"], true)
    where FNV(span.trace_id.string) % 100 < 10        # 10%
# Murmur3Hash or XXH3 work too and are faster on long inputs.
```
