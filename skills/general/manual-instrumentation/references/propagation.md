# Propagation

Preserve trace continuity across boundaries.

## Rules

At inbound boundaries:
- extract incoming trace context
- start the server or consumer span from that extracted context

At outbound boundaries:
- inject trace context into headers, metadata, or message properties

Across internal calls:
- preserve the active context instead of starting disconnected traces

## Baggage

Use baggage intentionally.

Good uses:
- bounded business context that must cross service boundaries
- values that help downstream decisions or correlation

Avoid:
- large payloads
- sensitive data unless explicitly approved and protected
- turning baggage into a generic key-value dump

## Default Preference

Prefer W3C Trace Context and W3C Baggage unless a legacy environment requires otherwise.
