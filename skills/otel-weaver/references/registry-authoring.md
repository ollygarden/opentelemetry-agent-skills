# Registry Authoring

Authoritative schema reference: [`schemas/semconv-syntax.v2.md`](https://github.com/open-telemetry/weaver/blob/main/schemas/semconv-syntax.v2.md) in the Weaver repo. Use this file as a quick map; do not duplicate field tables.

## Layout

```
telemetry/registry/
├── manifest.yaml          # required
├── attributes.yaml        # optional
├── metrics.yaml           # optional
├── spans.yaml             # optional
└── events.yaml            # optional
```

Files other than `manifest.yaml` are discovered by their `file_format: definition/2` header and the top-level array key (`attributes`, `metrics`, `spans`, ...).

## `manifest.yaml`

Minimal example:

```yaml
name: ecommerce
description: "Telemetry conventions for the ecommerce monolith"
semconv_version: 0.1.0
schema_url: https://example.com/schemas/ecommerce
```

`semconv_version` is yours to manage — bump it on any schema change. `schema_url` is what consumers see; pick a stable URL even if it does not yet resolve.

## Attributes

```yaml
file_format: definition/2

attributes:
  - key: ecommerce.user.id
    type: string
    stability: stable
    brief: "Identifier of the customer performing the operation."
    examples: ['user-9f3a', 'user-1c2d']

  - key: ecommerce.payment.method
    type:
      allow_custom_values: true
      members:
        - id: credit_card
          value: credit_card
          stability: stable
          brief: "Credit or debit card"
        - id: paypal
          value: paypal
          stability: stable
          brief: "PayPal account"
    stability: stable
    brief: "Payment method selected by the customer."
```

Notes:
- Required fields: `key`, `type`, `stability`, `brief`.
- Primitive `type` values: `string`, `int`, `double`, `boolean`, plus their `[]` array variants.
- Enum types use the `members` form. Set `allow_custom_values: false` to refuse anything outside the listed `id`s.
- Provide `examples` for non-enum strings; it improves generated docs and helps reviewers.

## Metrics

```yaml
file_format: definition/2

metrics:
  - name: ecommerce.orders.processing.duration
    instrument: histogram
    unit: s
    stability: stable
    brief: "End-to-end duration of processing a single order."
    attributes:
      - ref: ecommerce.payment.method
        requirement_level: required
      - ref: ecommerce.customer.tier
        requirement_level: recommended

  - name: ecommerce.orders.active
    instrument: updowncounter
    unit: "{order}"
    stability: stable
    brief: "Number of orders currently being processed."

  - name: ecommerce.products.inventory.value
    instrument: gauge
    unit: "USD"
    stability: stable
    brief: "Current value of products currently in stock, in USD."
```

Notes:
- `instrument` is one of `counter`, `updowncounter`, `histogram`, `gauge`.
- Counter names: drop the `.total` suffix.
- Duration histograms: use `s`. Bucket boundaries scale accordingly (e.g. `0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5`).
- `attributes:` here are by `ref` only. Declare attributes once in `attributes.yaml` and reference them across metrics, spans, and events.

## Spans

```yaml
file_format: definition/2

spans:
  - type: ecommerce.order.process
    kind: internal
    stability: stable
    name:
      note: "ecommerce.order.process"
    brief: "Process a single customer order end-to-end."
    attributes:
      - ref: ecommerce.user.id
        requirement_level: required
        sampling_relevant: true
      - ref: ecommerce.payment.method
        requirement_level: required
        sampling_relevant: true
      - ref: ecommerce.order.id
        requirement_level: recommended
```

Notes:
- Required fields: `type`, `kind`, `stability`, `brief`, and a structured `name: { note: "..." }`.
- `kind`: one of `client`, `server`, `producer`, `consumer`, `internal`.
- For internal business spans with static names, put the dotted type identifier in `name.note` and use the resolved `span.name` string at runtime.
- Upstream OTel HTTP/db/messaging spans use `{action} {target}` patterns derived from attributes — those should not appear in an org-local registry.
- `sampling_relevant: true` flags attributes that the SDK should make available at sampling time.

## What does NOT belong in your local registry

Boundary domains owned by upstream OTel semconv:
- `http`, `db`, `messaging`, `rpc`, `network`, `gen-ai`, ...

Reference the language SDK's semconv package for these (e.g. `go.opentelemetry.io/otel/semconv/v1.X.0`) until upstream is pulled in as a manifest dependency. Redeclaring them locally creates a maintenance burden and silently diverges from the ecosystem.

## Validation

Fast feedback loop:

```
weaver registry check -r ./telemetry/registry/
```

Expected stderr noise: `File format definition/2 is not yet stable`. This is normal as of 0.22.1.
