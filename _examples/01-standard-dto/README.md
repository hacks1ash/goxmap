# Example 01: Standard DTO Mapping

Maps a database model (`DBUser`) with nullable pointer fields to a clean JSON
DTO (`UserResponse`) with non-pointer fields.

## What it demonstrates

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
