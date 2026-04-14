# Example 03: Custom Converters

Maps a domain `Event` to an API `EventDTO` using custom converter functions
for `time.Time` formatting and tag normalization.

## What it demonstrates

- `mapper:"func:FormatTime"` to convert `time.Time` to RFC3339 string
- `mapper:"func:NormalizeTags"` to apply domain-specific string transformation
- Custom functions are plain Go functions in the same package, no special interface required

## Run

```bash
go run .
```

## Regenerate

```bash
go generate ./...
```
