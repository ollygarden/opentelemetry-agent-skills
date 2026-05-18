# Migration Playbook: Hand-Maintained Constants → Weaver Registry

Use this playbook when an application has a hand-rolled file like `const.go`, `telemetry/constants.py`, or `Telemetry.java` declaring attribute keys, metric names, and span names — and you want to replace it with a generated, schema-backed package.

The walkthrough uses Go conventions; mappings to other languages are straightforward.

## Inventory the existing constants

For each constant in the legacy file, classify it:

| Kind                                 | Action                                                      |
|--------------------------------------|-------------------------------------------------------------|
| Org-local attribute (`ecommerce.*`)  | Move to `attributes.yaml`.                                  |
| Org-local metric name + unit         | Move to `metrics.yaml` with `instrument` and `unit`.        |
| Org-local span name                  | Move to `spans.yaml` with `kind` and `name.note`.           |
| HTTP/db/messaging/rpc/network/gen-ai | **Delete.** Replace call sites with the SDK semconv package. |
| Free-form log keys, config flags     | Out of scope for Weaver; leave them.                        |

This is the most common modeling mistake: do not redeclare upstream-domain attributes locally. Use the language SDK's semconv package directly.

## Author the registry

Build the four files:

- `telemetry/registry/manifest.yaml`
- `telemetry/registry/attributes.yaml`
- `telemetry/registry/metrics.yaml`
- `telemetry/registry/spans.yaml`

See `references/registry-authoring.md` for field reference and a working sample.

While moving entries, normalize:
- counter names: drop `.total`
- duration histograms: convert `ms` → `s` and rescale bucket boundaries (e.g. `0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5`)
- enum-shaped strings: declare them as enum types so consumers see allowed values

Validate as you go:

```
weaver registry check -r ./telemetry/registry/
```

## Author templates

Copy the Go templates from `references/template-authoring.md` into `telemetry/templates/registry/go/`. Adjust:
- `params.package_name` to your target package
- the output filenames (`attributes_gen.go`, `metrics_gen.go`, `spans_gen.go`) — keep the `_gen.go` suffix so it's grep-obvious that they are generated

## Generate and format

```
weaver registry generate \
  --registry ./telemetry/registry/ \
  --templates ./telemetry/templates/ \
  go \
  ./internal/telemetry/

gofmt -w ./internal/telemetry/
```

The generated `attributes_gen.go`, `metrics_gen.go`, `spans_gen.go` should compile cleanly alongside whatever non-generated code remains in the package (`setup.go`, helpers, etc.).

## Switch call sites

Replace references one constant at a time:

| Before                                    | After                                                  |
|-------------------------------------------|--------------------------------------------------------|
| `telemetry.AttrUserID`                    | `telemetry.AttrEcommerceUserId`                        |
| `"ecommerce.orders.processing.duration"`  | `telemetry.EcommerceOrdersProcessingDurationName`      |
| `"ecommerce.order.process"`               | `telemetry.SpanEcommerceOrderProcessName`              |
| `semconv.HTTPMethodKey` (already correct) | unchanged — keep using the SDK semconv package         |

For instrumentation hygiene as you do this, pair with the `manual-instrumentation` skill.

## Delete the legacy file

Once all call sites compile against the generated symbols, delete the hand-maintained `const.go`. Run the test suite. Inspect a sample trace and metric to confirm names and attributes still look right end-to-end.

## Wire CI

See `references/ci-integration.md`. Ship the workflow in the same PR as the migration so the contract is enforced from day one.

## Verify before declaring victory

- `weaver registry check` returns success (modulo the expected `definition/2` warning)
- `weaver registry generate` followed by the language formatter leaves `git diff --exit-code` clean
- the application builds and tests pass
- a sample trace shows org-local span names and attributes from the generated symbols
- a sample metric shows the renamed/rescaled units and no leftover `.total` suffix
