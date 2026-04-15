# Example 01: Standard DTO Mapping

Maps a database model (`*DBUser`) with nullable pointer fields to a pointer-based
JSON DTO (`*UserResponse`) with field-level nil-safe conversions.

## What it demonstrates

- Pointer-based function signatures for type-safe nil handling
- `*string` to `string` conversion with nil-safe dereference
- Zero-value fallback when a pointer is nil (`Phone` becomes `""`)
- Direct assignment for matching non-pointer fields

## Run

```bash
go run .
```

## Regenerate

```bash
go generate ./...
```
