# `attributes`: advanced use-cases

## Scope with include / exclude

Limit the actions to a subset so you don't touch unrelated telemetry. `include` is evaluated first; `exclude` then trims that set.

```yaml
processors:
  attributes/scoped:
    include:
      match_type: strict
      services: ["auth-service"]
    exclude:
      match_type: strict
      attributes:
        - key: test_request
          value: true
    actions:
      - key: sensitive_data
        action: delete
      - key: user.email
        action: hash
```

Only spans from `auth-service` are considered, and test requests within that set are skipped. For metrics, only `metric_names` and `resources` are valid match fields — see [Configuration](configuration.md#valid-fields-per-signal).

## Hash PII instead of deleting

Hashing keeps a stable, joinable token without exposing the raw value. Hash by key or by regex `pattern`:

```yaml
actions:
  - key: user.email
    action: hash
  - pattern: "^pii_.*"
    action: hash
```

## Extract structured fields from a string

`extract` runs a regex with **named** capture groups against a string attribute and creates one attribute per group. The source value is left in place.

```yaml
actions:
  - key: http.url
    pattern: '^(?P<http_scheme>https?):\/\/(?P<http_host>[^\/]+)(?P<http_path>.*)'
    action: extract
  # http.url = "https://api.example.com/v1/users"
  # -> http_scheme="https", http_host="api.example.com", http_path="/v1/users"
```

## Convert types

Coerce an attribute to `int`, `double`, or `string` for downstream compatibility:

```yaml
actions:
  - key: http.status_code
    action: convert
    converted_type: int
  - key: response_time_ms
    action: convert
    converted_type: double
```

## Ordered-action pipelines

Actions run top to bottom, so you can stage a transform: copy through a temporary key, then clean up.

```yaml
actions:
  - key: operation        # 1. backfill a default
    value: default
    action: insert
  - key: svc.operation    # 2. copy to the standardized key
    from_attribute: operation
    action: upsert
  - key: operation        # 3. drop the old key
    action: delete
```

## Inject request context

Pull values from transport metadata or an authenticator (requires `include_metadata: true` on the receiver, or an authenticator extension — see [Configuration](configuration.md#context-values)).

```yaml
actions:
  - key: client.ip
    from_context: client.address
    action: insert
  - key: enduser.id
    from_context: auth.subject
    action: insert
```

## Named instances

Configure the same type more than once with `type/name` and reference each in the pipeline that needs it:

```yaml
processors:
  attributes/redact:
    actions:
      - key: user.email
        action: hash
  attributes/enrich:
    actions:
      - key: region
        value: us-east-1
        action: upsert

service:
  pipelines:
    traces:
      processors: [attributes/redact, attributes/enrich]
```

## Combining with `resource` and `filter`

- Pair with **`resource`** when you also need to edit resource-level attributes (`attributes` cannot — it only sees span/log/datapoint attributes).
- Run **before `filter`** so the attribute you want to filter on exists, or has been normalized, by the time `filter`'s condition is evaluated.

```yaml
processors:
  attributes:
    actions:
      - key: env
        from_attribute: deployment.env
        action: upsert
  filter:
    traces:
      span:
        - 'attributes["env"] == "test"'   # drop test spans normalized above

service:
  pipelines:
    traces:
      processors: [attributes, filter]
```
