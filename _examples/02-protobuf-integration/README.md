# Example 02: Protobuf Integration

Maps an internal domain model to a simulated Protobuf-generated struct using
`mapper:"bind:..."` tags for field name aliasing and pointer-based function signatures.

## What it demonstrates

- Pointer-based function signatures (`*User -> *ProtoUser`) for type-safe nil handling
- `bind` tags to map between different field naming conventions
- Bidirectional mapping (internal to proto and proto to internal)
- Getter-based access (`GetFullName()`) in the reverse direction for nil safety
- Nil-safe returns when mapping from nil pointers

## Run

```bash
go run .
```
