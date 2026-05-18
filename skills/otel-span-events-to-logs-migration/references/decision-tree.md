# Decision Tree: Classifying Span Event Call Sites

For each `AddEvent` or `RecordException` call site, follow this tree to determine the migration action.

## Step 1: Is It an Exception?

If the call uses `RecordException` (or `AddEvent` with `exception.*` attributes):
- **Action: Migrate to log-based exception**
- Use the Logs API to emit an event with name `exception`
- Preserve `exception.type`, `exception.message`, `exception.stacktrace`
- The log record must carry the active span context

## Step 2: Is It a Timestamped Diagnostic Event?

If the event represents a discrete occurrence with its own timestamp that aids debugging:
- Examples: retry attempt, state transition, circuit breaker trip, cache miss, fallback activation
- **Action: Migrate to log-based event**
- Use the Logs API to emit an event with the same name and attributes
- The log record must carry the active span context

## Step 3: Is It Really Just Extra Span Detail?

If the event records additional details about the span that do not need their own timestamp:
- Examples: request/response body snippets, configuration values, resolved parameters, feature flag states
- **Action: Convert to span attributes**
- Move the data to attributes on the span itself
- Use semantic conventions where applicable
- See [semantic-conventions#2010](https://github.com/open-telemetry/semantic-conventions/issues/2010)

## Step 4: Is It Noise?

If the event provides no diagnostic value or duplicates information already on the span or in metrics:
- Examples: "processing started", "step 1 complete", happy-path confirmations
- **Action: Remove**
- Document the removal reason in the commit or PR

## Summary Table

| Pattern | Action | Target |
|---|---|---|
| `RecordException` / exception event | Migrate | Log-based exception via Logs API |
| Timestamped diagnostic event | Migrate | Log-based event via Logs API |
| Detail without own timestamp | Convert | Span attributes |
| Noise or duplication | Remove | Delete the call |
