# Example 02: Protobuf Integration

Maps an internal domain model to a simulated Protobuf-generated struct using
`mapper:"bind:..."` tags for field name aliasing.

## What it demonstrates

- `bind` tags to map between different field naming conventions
- Bidirectional mapping (internal to proto and proto to internal)
- Getter-based access (`GetFullName()`) in the reverse direction for nil safety
- Zero-value safety when mapping from an uninitialized Protobuf message

## Run

```bash
go run .
```
