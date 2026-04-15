# Technical Design

This document describes the internal architecture of `goxmap`, a compile-time code generator that produces type-safe struct mapping functions for Go. It is intended for contributors and advanced users who want to understand the design decisions, data flow, and extension points of the tool.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Pipeline Stages](#pipeline-stages)
- [Stage 1: Type Loading and AST Analysis](#stage-1-type-loading-and-ast-analysis)
  - [Why `golang.org/x/tools/go/packages`](#why-golangorgxtoolsgopackages)
  - [Package Loading Modes](#package-loading-modes)
  - [The `PackageContext` Model](#the-packagecontext-model)
  - [Struct Field Extraction](#struct-field-extraction)
  - [Cross-Package Type Resolution](#cross-package-type-resolution)
  - [Getter Discovery](#getter-discovery)
- [Stage 2: Field Matching](#stage-2-field-matching)
  - [Same-Package Matching](#same-package-matching)
  - [Cross-Package Matching and the Resolution Hierarchy](#cross-package-matching-and-the-resolution-hierarchy)
  - [Bidirectional Pair Generation](#bidirectional-pair-generation)
- [Stage 3: Recursive Mapper Planning](#stage-3-recursive-mapper-planning)
  - [The Generation Queue](#the-generation-queue)
  - [Mapper Discovery and Reuse](#mapper-discovery-and-reuse)
  - [Circular Dependency Protection](#circular-dependency-protection)
- [Stage 4: Code Generation](#stage-4-code-generation)
  - [Template-Based Emission](#template-based-emission)
  - [Multi-Function Rendering](#multi-function-rendering)
  - [Cross-Package Rendering and Import Management](#cross-package-rendering-and-import-management)
  - [Getter-Based Access for Protobuf Compatibility](#getter-based-access-for-protobuf-compatibility)
  - [CLI Helper Functions](#cli-helper-functions)
- [Stage 2.5: Type Mismatch Resolution](#stage-25-type-mismatch-resolution)
  - [Numeric Coercion](#numeric-coercion)
  - [Converter Function Auto-Discovery](#converter-function-auto-discovery)
  - [Resolution Priority](#resolution-priority)
- [Pointer Conversion Logic](#pointer-conversion-logic)
  - [The Conversion Matrix](#the-conversion-matrix)
  - [Nested Struct Pointer Combinations](#nested-struct-pointer-combinations)
  - [Slice Element Pointer Combinations](#slice-element-pointer-combinations)
- [Tag System](#tag-system)
  - [Tag Grammar](#tag-grammar)
  - [Directive Reference](#directive-reference)
- [CLI and `go generate` Integration](#cli-and-go-generate-integration)
- [Design Decisions and Trade-offs](#design-decisions-and-trade-offs)

---

## Architecture Overview

`goxmap` follows a four-stage pipeline that transforms Go type metadata into formatted source code:

```
                     +-----------+
                     |  CLI /    |
                     | go:generate|
                     +-----+-----+
                           |
                           v
              +------------------------+
              |  Stage 1: Type Loading |
              |  (internal/loader)     |
              +----------+-------------+
                         |
                         v
              +------------------------+
              |  Stage 2: Field Match  |
              |  (internal/matcher)    |
              +----------+-------------+
                         |
                         v
              +------------------------+
              |  Stage 3: Recursive    |
              |  Mapper Planning       |
              |  (internal/generator)  |
              +----------+-------------+
                         |
                         v
              +------------------------+
              |  Stage 4: Code Gen     |
              |  & Formatting          |
              |  (internal/generator)  |
              +----------+-------------+
                         |
                         v
                  xxx_mapper_gen.go
```

Each stage has a single responsibility and communicates with the next through well-defined data structures: `StructInfo`, `MatchResult`/`CrossMatchResult`, and `funcEntry`. This separation ensures that changes to the matching algorithm, for instance, do not require changes to the code emitter.

---

## Pipeline Stages

## Stage 1: Type Loading and AST Analysis

**Package:** `internal/loader`

The loader is responsible for extracting complete type information from Go source code. It produces `StructInfo` values that describe a struct's fields, their types, struct tags, and type-level metadata such as whether a field is a named struct, a slice, or a pointer.

### Why `golang.org/x/tools/go/packages`

The tool uses `golang.org/x/tools/go/packages` rather than the lower-level `go/ast` + `go/parser` combination. This is a deliberate architectural choice driven by three requirements:

1. **Cross-package type resolution.** When a struct field references a type from another package (e.g., `proto.UserProto`), the loader must resolve that type through the module graph. `go/packages` integrates with the Go module system and performs full type-checking, providing a `*types.Package` with a populated scope. Raw AST parsing would require manually resolving imports, which is fragile and incomplete for types in external modules.

2. **Accurate type information.** The `go/types` package, populated by `go/packages`, provides the canonical representation of Go types after alias resolution, interface satisfaction, and method set computation. The loader needs this to correctly identify named struct types, compute method sets for getter discovery, and produce qualified type strings.

3. **Struct tag access.** The `*types.Struct` type provides access to field tags via `Tag(i)`, which is essential for parsing `json`, `mapper`, and other struct tags. This avoids the need to correlate AST-level tag literals with type-level field information.

### Package Loading Modes

The loader requests five capability flags when loading a package:

```go
packages.NeedName |       // Package name (e.g., "models")
packages.NeedTypes |      // Type-checked *types.Package
packages.NeedTypesInfo |  // Type information for identifiers
packages.NeedSyntax |     // Parsed AST (for future extension)
packages.NeedImports      // Imported packages (for cross-package resolution)
```

These flags are the minimum set required for the tool's current feature surface. `NeedSyntax` is included to support potential future features such as comment-based annotations, though it is not strictly required by the current implementation.

### The `PackageContext` Model

A loaded package is wrapped in a `PackageContext` struct, which serves as a handle for subsequent operations:

```go
type PackageContext struct {
    Pkg *packages.Package
}
```

The rationale for this wrapper is twofold:

- **Amortized loading cost.** `packages.Load` is expensive (it invokes `go list` and performs type-checking). By loading the package once and passing `PackageContext` to `LoadStructFromPkg`, `DiscoverMapperFuncs`, and `DiscoverGetters`, the tool avoids redundant loads when processing multiple types from the same package.

- **Encapsulation.** Callers operate on `PackageContext` rather than the `packages.Package` type directly, which allows the loader to change its internal representation without breaking the API.

Two entry points produce a `PackageContext`:

| Function | Input | Use Case |
|---|---|---|
| `LoadPackage(dir)` | Filesystem directory | Loading the local package being processed |
| `LoadExternalPackage(dir, importPath)` | Directory + import path | Loading an external dependency by its module path |

`LoadExternalPackage` sets the `Dir` field on the `packages.Config` to ensure module resolution happens from the correct working directory, then passes the import path (e.g., `github.com/org/repo/proto`) as the load pattern. This allows the Go toolchain to resolve the package through the module graph.

### Struct Field Extraction

`LoadStructFromPkg` performs the following steps:

1. **Scope lookup.** The type name is looked up in the package's type scope (`pkg.Types.Scope().Lookup(typeName)`). This returns a `types.Object` or nil.

2. **Named type assertion.** The object is asserted to `*types.Named` to ensure it is a named type (not an alias or built-in).

3. **Underlying struct extraction.** The named type's underlying type is asserted to `*types.Struct`. This is the canonical struct representation after all type resolution.

4. **Field iteration.** Each exported field is analyzed:
   - **Basic metadata:** Name, full type string (via `types.TypeString` with a qualifier that produces short names within the same package), JSON tag, and mapper tag directives.
   - **Pointer analysis:** If the field type is `*types.Pointer`, `IsPtr` is set and `ElemType` records the dereferenced type.
   - **Named struct detection:** After peeling the pointer (if any), the type is checked for `*types.Named` with a `*types.Struct` underlying type. This populates `IsNamedStruct` and `StructName`.
   - **Slice analysis:** If the (possibly dereferenced) type is `*types.Slice`, the element type undergoes the same pointer and named-struct analysis. This populates `IsSlice`, `SliceElemType`, `SliceElemIsPtr`, `SliceElemTypeName`, and `IsSliceElemStruct`.

The `analyzeFieldType` function encapsulates this analysis. Its layered approach (peel pointer, check named struct, check slice, check slice element pointer, check slice element named struct) produces a flat `StructField` that downstream stages can query without repeating type introspection.

### Cross-Package Type Resolution

Cross-package mapping requires loading two separate packages: the local (internal) package and the external package. These are loaded independently via `LoadPackage` and `LoadExternalPackage`, producing two distinct `PackageContext` values.

The qualifier function ensures type strings are contextual:

```go
func qualifier(pkg *types.Package) types.Qualifier {
    return func(other *types.Package) string {
        if pkg == other {
            return ""
        }
        return other.Name()
    }
}
```

Within the same package, types are unqualified (`Address`). Cross-package types are qualified with the package name (`proto.UserProto`). This ensures generated code uses the correct identifiers.

### Getter Discovery

`DiscoverGetters` scans the method set of a type for Protobuf-style getter methods. The algorithm:

1. Constructs the pointer type `*T` (since Protobuf getters are typically defined on pointer receivers).
2. Computes the method set via `types.NewMethodSet(ptrType)`.
3. Filters for methods matching: exported, prefix `Get`, no parameters (beyond receiver), exactly one return value.
4. Strips the `Get` prefix to derive the field name (e.g., `GetFullName` maps to field `FullName`).

The result is a `map[string]GetterInfo` keyed by field name, which is passed to the matcher for use in the `FromExternal` direction.

---

## Stage 2: Field Matching

**Package:** `internal/matcher`

The matcher produces `FieldPair` values that associate a source field with a destination field. Each pair carries enough metadata for the generator to emit the correct assignment expression.

### Same-Package Matching

The `Match` function implements a two-level resolution strategy:

1. **JSON tag match.** If the destination field has a `json` tag, the matcher looks for a source field with the same `json` tag value. This handles cases where Go field names differ but serialization names align.

2. **Field name match.** If no JSON match is found, the matcher falls back to matching by Go field name.

The matcher iterates over destination fields and attempts to find a source match. Unmatched destination fields are collected separately and reported as warnings by the CLI.

### Cross-Package Matching and the Resolution Hierarchy

`MatchCross` implements a four-level resolution hierarchy, evaluated in strict priority order:

| Priority | Tag | Semantics |
|---|---|---|
| 1 | `mapper:"bind:ExternalFieldName"` | Match the internal field to the external field with the exact Go name `ExternalFieldName`. |
| 2 | `mapper:"bind_json:external_json_key"` | Find the external field whose `json` tag value equals `external_json_key`. |
| 3 | `json:"key"` | Standard JSON tag matching (same as same-package matcher). |
| 4 | _(none)_ | Fall back to Go field name matching. |

The design principle is that more explicit bindings take precedence over implicit ones. `bind` is the most explicit (the developer names the exact target field), while field name matching is the least explicit (it relies on naming conventions).

Resolution stops at the first successful match. If `bind` matches, the JSON and name strategies are never evaluated. This prevents ambiguous matches when an internal field has both a `bind` tag and a `json` tag that would resolve to different external fields.

### Bidirectional Pair Generation

`MatchCross` produces a `CrossMatchResult` containing two `MatchResult` values:

- **`ToExternal`**: Pairs where `Src` is the internal field and `Dst` is the external field. These pairs drive the `MapInternalToExternal` function.
- **`FromExternal`**: Pairs where `Src` is the external field and `Dst` is the internal field. These pairs drive the `MapExternalToInternal` function.

The bidirectional pairs are generated in a single pass over the internal fields. For each matched field:

1. A `ToExternal` pair is created with `Src=internal, Dst=external`.
2. A `FromExternal` pair is created with `Src=external, Dst=internal`. If a getter exists for the external field (from `DiscoverGetters`), the pair is annotated with `UseGetter=true` and `GetterName`.

This single-pass approach ensures consistency: every field that is mapped in one direction is also mapped in the other direction (when bidirectional mode is enabled). Unmatched fields appear in both `ToExternal.Unmatched` and `FromExternal.Unmatched`.

---

## Stage 2.5: Type Mismatch Resolution

After field matching produces `FieldPair` values, a classification step inspects each pair for type mismatches. This happens in `classifyPair()` (called during `Match()`) and `resolveTypeMismatches()` (called by the CLI after matching).

### Ignored and Optional Fields

Before field matching, destination fields tagged with `mapper:"ignore"` are excluded entirely from the matching process. These fields produce no assignment and no warning, providing a mechanism to exclude fields from auto-mapping.

Destination fields tagged with `mapper:"optional"` participate in matching normally, but if unmatched, they are flagged in the `Unmatched` list without generating a CLI warning. The field receives the Go zero value by default.

This distinction allows developers to:
- Use `mapper:"ignore"` for internal/private fields that should never be set by the mapper
- Use `mapper:"optional"` for newly added fields where the zero value is acceptable

### Named Type Conversion (Enum Support)

When source and destination fields are different **named types** with the **same underlying type**, the pair is classified as `TypeCast`:

```go
type StatusA string
type StatusB string

// Generated: dst.Status = StatusB(src.Status)
```

The same-underlying-type detection uses the `go/types` API to compare underlying types precisely. This covers Go enum patterns (string-based enums, iota int-based enums, etc.) without requiring explicit `mapper:"func:..."` tags. Pointer combinations are handled with nil-safety closures, identical to numeric casts.

### Numeric Coercion

When both the source and destination element types (after pointer peeling) are Go numeric types, the pair undergoes narrowing detection:

**Non-narrowing casts** (safe conversions) are classified as `NumericCast` and the generator emits a simple Go type conversion:

```go
dst.Age = int32(src.Age)  // int to int32: safe (widening)
```

**Narrowing casts** (conversions that may lose data) are treated as `TypeMismatch` and require an explicit converter function:

```go
// int64 to int32: narrowing (smaller destination)
// int to float: cross-family (potential precision loss)
// uint to int: cross-sign (different interpretation)
```

The numeric type set includes `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `byte`, and `rune`.

Narrowing detection operates on two rules:

1. **Cross-family conversions** (signed ↔ unsigned, int ↔ float) are always narrowing and require an explicit converter.
2. **Same-family conversions** are narrowing when the destination has fewer bits than the source (e.g., `int64` → `int32`).

For pointer combinations of non-narrowing casts, the cast is wrapped in a nil-safe closure:

```go
// *int -> int32: safe widening
dst.Age = func() int32 {
    if src.Age != nil { return int32(*src.Age) }
    var zero int32; return zero
}()
```

Non-narrowing numeric casts compile to single machine instructions with zero heap allocations (8.9 ns/op in benchmarks). This is 319x faster than reflection-based alternatives, which must perform runtime type checks and `reflect.Value` boxing for each conversion.

### Converter Function Auto-Discovery

When element types differ and neither is a numeric-to-numeric conversion, the CLI attempts to find a converter function in the package scope:

1. **Compute the expected name.** `BaseTypeName()` extracts a capitalized base name: `time.Time` becomes `Time`, `string` becomes `String`, `*int32` becomes `Int32`. The expected function name is `Map<SrcBase>To<DstBase>` (e.g., `MapTimeToString`).

2. **Scope lookup.** `FindConverterFunc()` looks up the name in `pctx.Pkg.Types.Scope()` and verifies it is a `*types.Func` with exactly one parameter and one return value.

3. **Set the converter.** If found, `pair.ConverterFunc` is set and the generator emits a function call: `dst.F = MapTimeToString(src.F)`.

This auto-discovery eliminates the need for `mapper:"func:..."` tags in the common case where the converter follows the naming convention. The explicit tag still works and takes priority.

### Resolution Priority

When a matched field pair has different types, the following priority chain is evaluated during classification:

| Priority | Condition | Behavior |
|---|---|---|
| 1 | `mapper:"func:Fn"` tag present | Use `Fn` as the converter (skip further classification) |
| 2 | Both types are numeric and non-narrowing | Emit inline type cast (fast path) |
| 3 | Both types are named non-struct with same underlying | Emit named type cast (enum support) |
| 4 | Both types are numeric and narrowing | Mark as `TypeMismatch` (requires explicit converter) |
| 5 | Types differ but neither is numeric or same-underlying | Mark as `TypeMismatch` (requires explicit converter) |

After classification, unresolved `TypeMismatch` pairs enter the resolution phase:

| Priority | Condition | Behavior |
|---|---|---|
| 1 | `mapper:"func:Fn"` tag present | Use `Fn` as the converter |
| 2 | `Map<Src>To<Dst>` function exists in package | Use discovered converter |
| 3 | None of the above | Fatal error with suggestion |

The fatal error message includes the expected function name, a copy-paste-ready function signature, and (for narrowing conversions) a note that the conversion may lose data. This minimizes the developer's feedback loop and encourages explicit handling of risky conversions.

---

## Stage 3: Recursive Mapper Planning

**Package:** `internal/generator` (the `multiGenerator` type)

When the root mapper's field pairs include nested struct or slice-of-struct fields, the generator must ensure that sub-mappers exist for those types. This is orchestrated by the `multiGenerator`, which maintains a generation queue and several caches.

### The Generation Queue

The `multiGenerator` holds an ordered list of `funcEntry` values (`funcs []funcEntry`). Each entry describes a single mapper function to generate. The queue is populated depth-first: when the root entry is enqueued, its field pairs are inspected for nested dependencies, and those dependencies are enqueued before the root entry itself.

```
Enqueue(MapUserToUserDTO)
  ├── Inspect field "Address" → NeedsNestedMapper → ensureMapper(Address, AddressDTO)
  │   └── Enqueue(MapAddressToAddressDTO)
  │       └── (no nested dependencies)
  ├── Inspect field "Emails" → NeedsSliceMapper → ensureMapper(EmailInfo, EmailInfoDTO)
  │   └── Enqueue(MapEmailInfoToEmailInfoDTO)
  │       └── (no nested dependencies)
  └── Append MapUserToUserDTO to queue
```

The resulting queue is `[MapAddressToAddressDTO, MapEmailInfoToEmailInfoDTO, MapUserToUserDTO]`. Dependencies appear before their dependents, which is a natural consequence of the depth-first traversal, though the ordering is not strictly required since all functions are emitted into the same file.

### Mapper Discovery and Reuse

Before generating a sub-mapper, the generator checks two caches:

1. **`existing map[string]bool`**: Mapper functions already present in the target package, discovered by `loader.DiscoverMapperFuncs`. These are hand-written or previously generated functions that the tool should not overwrite.

2. **`generated map[string]bool`**: Mapper functions already enqueued in the current generation run.

If a function name appears in either cache, `ensureMapper` returns immediately without loading the nested types or creating a new entry. This has two implications:

- **Developer escape hatch.** A developer can write a custom `MapAddressToAddressDTO` function with complex logic (e.g., geocoding, normalization). The generator will call it from `MapUserToUserDTO` without attempting to generate its own version.

- **Deduplication.** If multiple parent types reference the same nested type, the sub-mapper is generated only once.

### Circular Dependency Protection

Mutually recursive types (e.g., `Parent` has `[]Child`, `Child` has `*Parent`) would cause infinite recursion during enqueue. The `inProgress map[string]bool` cache breaks cycles:

```go
func (g *multiGenerator) enqueue(entry funcEntry) error {
    key := entry.FuncName
    if g.generated[key] || g.existing[key] {
        return nil  // Already handled.
    }
    if g.inProgress[key] {
        return nil  // Cycle detected — break it.
    }

    g.inProgress[key] = true
    defer delete(g.inProgress, key)

    g.generated[key] = true

    // Process nested dependencies (may recursively call enqueue).
    for _, pair := range entry.Pairs {
        g.processFieldPair(pair)
    }

    g.funcs = append(g.funcs, entry)
    return nil
}
```

The key insight is that `generated` is set *before* processing children. When the cycle loops back to a function already in `generated`, the recursion terminates. The `inProgress` set provides an additional guard: if a function is currently being processed (its children are being enqueued), a re-entrant call returns immediately.

Both mapper functions in a cycle are still generated. The generated code compiles correctly because Go allows forward references within the same package: `MapParentToParentDTO` calls `MapChildToChildDTO` and vice versa, and both are emitted into the same file.

---

## Stage 4: Code Generation

**Package:** `internal/generator`

The generator transforms the planned function entries into formatted Go source code.

### Template-Based Emission

The single-function `Generate` path uses `text/template` for simple mapper functions:

```go
const fieldLevelTmpl = `// Code generated by goxmap; DO NOT EDIT.

package {{ .PackageName }}

// {{ .FuncName }} maps {{ .SrcType }} to {{ .DstType }}.
func {{ .FuncName }}(src {{ .SrcType }}) {{ .DstType }} {
	var dst {{ .DstType }}
{{- range .Assignments }}
	dst.{{ .DstField }} = {{ .Expression }}
{{- end }}
	return dst
}
`
```

Assignment expressions are pre-computed by `buildAssignment` and injected as opaque strings. This keeps the template declarative: it knows about function structure but not about pointer logic, nil checks, or nested mapper calls.

All generated output passes through `go/format.Source` before being written to disk. This ensures consistent formatting regardless of template whitespace and eliminates the need for manual indentation management in templates.

### Multi-Function Rendering

`GenerateMulti` takes a different approach from the template-based path. Because it emits multiple functions into a single file and some assignments (particularly slice mappings) are multi-line blocks, it uses `fmt.Fprintf` to write directly to a `bytes.Buffer`:

```go
func (g *multiGenerator) render() ([]byte, error) {
    var buf bytes.Buffer
    buf.WriteString("// Code generated by goxmap; DO NOT EDIT.\n\n")
    buf.WriteString("package " + g.pkgName + "\n\n")

    for i, entry := range g.funcs {
        body, _ := g.renderFunc(entry)
        buf.Write(body)
    }

    return format.Source(buf.Bytes())
}
```

The `renderFieldLevel` method handles field pairs in priority order:

1. **Custom mapper** (`pair.HasCustomMapper()`): Emits `dst.Field = CustomFunc(src.Field)`.
2. **Converter function** (`pair.ConverterFunc != ""`): Emits a converter call with pointer nil-safety. Covers both explicit `mapper:"func:..."` tags and auto-discovered `Map<Src>To<Dst>` functions.
3. **Numeric cast** (`pair.NumericCast`): Emits `DstType(src.Field)` with pointer nil-safety. Handles all four pointer combinations: `T->T`, `*T->T`, `T->*T`, `*T->*T`.
4. **Named type cast** (`pair.TypeCast`): Emits `CastType(src.Field)` with pointer nil-safety for enum support. Uses same nil-check pattern as numeric cast.
5. **Slice of named structs** (`pair.NeedsSliceMapper()`): Emits a multi-line block with nil check, `make`, and a `for` loop.
6. **Nested named struct** (`pair.NeedsNestedMapper()`): Emits a call to the sub-mapper, potentially wrapped in a nil-check closure for pointer combinations.
7. **Primitive types**: Emits direct assignment, deref, or addr-of expressions depending on pointer conversion.

### Cross-Package Rendering and Import Management

`GenerateCross` generates one or two functions for cross-package mapping. Unlike the multi-function renderer, it must emit an `import` block:

```go
buf.WriteString("import (\n")
buf.WriteString(fmt.Sprintf("\t%q\n", ccfg.ExternalPkgPath))
buf.WriteString(")\n\n")
```

The external type is qualified with the package name in the function signature:

```go
func MapUserToUserProto(src User) pb.UserProto { ... }
```

The `renderCrossFunc` helper handles both directions. A `useGetters` parameter controls whether `FieldPair.UseGetter` is honored. This parameter is `false` for the `ToExternal` direction (writing to external fields directly) and `true` for the `FromExternal` direction (reading from external fields via getters when available).

### Getter-Based Access for Protobuf Compatibility

Protobuf-generated Go structs use pointer receivers and accessor methods for nil safety. A Protobuf struct like:

```protobuf
message User {
    string full_name = 1;
}
```

Generates:

```go
type User struct {
    FullName string
}

func (x *User) GetFullName() string {
    if x != nil {
        return x.FullName
    }
    return ""
}
```

When mapping *from* such a struct, direct field access (`src.FullName`) works but bypasses the nil-safety guarantee. The generated getter (`src.GetFullName()`) is nil-safe by construction.

The tool detects these getters via `DiscoverGetters`, which scans the method set of the pointer type for methods matching the `Get<FieldName>` pattern with no parameters and a single return value. When a getter is found, the `FromExternal` pair is annotated, and the generator emits `src.GetFullName()` instead of `src.FullName`.

This design does not require the external struct to *be* a Protobuf type. Any struct with getter methods following this convention will benefit from the same treatment. The detection is purely structural.

### CLI Helper Functions

The `internal/cli/` package provides reusable helper functions called by the main command entry point. These helpers encapsulate logic for cross-cutting concerns:

| Function | Purpose |
|---|---|
| `ResolveTypeMismatches()` | Resolves unresolved type mismatches by searching for converter functions in package scope |
| `RunSamePackage()` | Orchestrates same-package mapping generation (matching, type resolution, multi-function code generation) |
| `RunCrossPackage()` | Orchestrates cross-package mapping generation (getter discovery, cross matching, bidirectional code generation) |
| `ToSnakeCase()` | Converts CamelCase type names to snake_case for output file naming (e.g., `UserDTO` → `user_dto_mapper_gen.go`) |

This separation allows the main command to focus on flag parsing and I/O, while delegating the core generation workflow to reusable, testable CLI helpers.

---

## Pointer Conversion Logic

Pointer mismatches between source and destination fields are common in real-world mapping scenarios. The tool handles all combinations automatically, prioritizing safety (no nil dereference panics) over brevity.

### The Conversion Matrix

The `FieldPair.Conversion()` method classifies each pair into one of three cases:

| Source | Destination | Conversion | Generated Code |
|---|---|---|---|
| `T` | `T` | `NoneConversion` | `dst.F = src.F` |
| `*T` | `*T` | `NoneConversion` | `dst.F = src.F` |
| `*T` | `T` | `DerefConversion` | Nil-checked dereference with zero-value fallback |
| `T` | `*T` | `AddrConversion` | Address-of via immediately-invoked function literal |

**`DerefConversion` (`*T` to `T`):**

```go
dst.F = func() string {
    if src.F != nil {
        return *src.F
    }
    var zero string
    return zero
}()
```

The immediately-invoked function literal (IIFL) pattern is used to keep the nil check and zero-value fallback as a single expression. This avoids introducing temporary variables into the enclosing scope and allows the assignment to remain a simple `dst.F = <expr>` statement.

The zero value is obtained via `var zero T`, which works for all types including structs, interfaces, and function types. Using a typed zero literal (e.g., `""` for string, `0` for int) would require type-specific logic with no behavioral benefit.

**`AddrConversion` (`T` to `*T`):**

```go
dst.F = func() *string {
    v := src.F
    return &v
}()
```

Go does not allow taking the address of a struct field access expression directly (`&src.F` is valid but `&(src.Method())` is not for getters). The IIFL pattern introduces a local variable whose address can be taken, and it works uniformly regardless of how the source expression is obtained.

### Nested Struct Pointer Combinations

When both source and destination are named structs, the pointer conversion interacts with the sub-mapper call. The generator handles four combinations:

| Source | Destination | Generated Code |
|---|---|---|
| `Struct` | `Struct` | `MapSrcToDst(src.F)` |
| `*Struct` | `Struct` | Nil check, dereference, map, zero-value fallback |
| `Struct` | `*Struct` | Map, take address of result |
| `*Struct` | `*Struct` | Nil check, dereference, map, take address, nil fallback |

The `*Struct` to `Struct` case:

```go
dst.F = func() DstType {
    if src.F != nil {
        return MapSrcToDst(*src.F)
    }
    var zero DstType
    return zero
}()
```

The `*Struct` to `*Struct` case:

```go
dst.F = func() *DstType {
    if src.F != nil {
        v := MapSrcToDst(*src.F)
        return &v
    }
    return nil
}()
```

### Slice Element Pointer Combinations

Slices add another dimension. The nil check is applied at the slice level (not the element level for value elements), and `make` pre-allocates the destination slice:

```go
if src.Emails != nil {
    dst.Emails = make([]EmailInfoDTO, len(src.Emails))
    for i, v := range src.Emails {
        dst.Emails[i] = MapEmailInfoToEmailInfoDTO(v)
    }
}
```

When slice elements are pointers, the same four-way matrix applies at the element level:

| Source Element | Destination Element | Loop Body |
|---|---|---|
| `T` | `T` | `dst.F[i] = MapFn(v)` |
| `*T` | `T` | Nil check on `v`, dereference, map |
| `T` | `*T` | Map, take address of result |
| `*T` | `*T` | Nil check on `v`, dereference, map, take address |

A nil source slice results in a nil destination slice (the `if` guard is not entered, and the zero value of a slice is `nil`). This is the idiomatic Go convention: nil and empty slices are semantically different, and the tool preserves this distinction.

---

## Tag System

The `mapper` struct tag is the primary configuration mechanism for field-level behavior. It uses a semicolon-delimited format to support multiple directives on a single field.

### Tag Grammar

```
mapper:"directive1:value1;directive2:value2"
```

Each directive is a `key:value` pair. The parser splits the tag value on `;`, trims whitespace, and attempts to match each part against known directive prefixes using `strings.CutPrefix`.

### Directive Reference

| Directive | Scope | Example | Semantics |
|---|---|---|---|
| `ignore` | Field | `mapper:"ignore"` | Exclude field from mapping entirely (no assignment, no warning). |
| `optional` | Field | `mapper:"optional"` | Suppress warning if no source match; field receives Go zero value. |
| `func:FnName` | Field | `mapper:"func:FormatTime"` | Use `FnName(src.Field)` instead of direct assignment. |
| `bind:ExtField` | Field | `mapper:"bind:UserId"` | Match this field to the external field named `ExtField` (Priority 1). |
| `bind_json:key` | Field | `mapper:"bind_json:user_id"` | Match this field to the external field whose `json` tag equals `key` (Priority 2). |
| `struct_func:FnName` | Struct | `mapper:"struct_func:CustomMap"` | Delegate the entire struct mapping to `FnName`. Applied via a blank identifier field: `_ struct{} \`mapper:"struct_func:CustomMap"\`` |

Directives can be combined:

```go
Name string `mapper:"bind:FullName;func:TrimName"`
```

This binds the field to `FullName` on the external struct and applies `TrimName` as the conversion function. The `bind` directive affects matching (Stage 2), while `func` affects code generation (Stage 4).

---

## CLI and `go generate` Integration

The root `main.go` is the user-facing entry point. It is designed to be invoked via `go generate`:

```go
//go:generate go run github.com/hacks1ash/goxmap -src User -dst UserDTO
```

Because the tool is invoked via `go run`, it does not require a pre-built binary. The Go toolchain compiles and caches the binary automatically.

**Flag summary:**

| Flag | Required | Default | Description |
|---|---|---|---|
| `-src` | Yes | | Source (or internal) struct type name |
| `-dst` | Yes | | Destination (or external) struct type name |
| `-func` | No | `Map<Src>To<Dst>` | Generated function name |
| `-dir` | No | `$GOFILE` dir or `.` | Package directory |
| `-output` | No | `<dst_snake>_mapper_gen.go` | Output file name |
| `-struct-func` | No | | Delegate entire mapping to this function |
| `-external-pkg` | No | | Import path for cross-package mode |
| `-bidi` | No | `false` | Generate bidirectional mappers (same-package or cross-package) |

The CLI operates in two modes:

1. **Same-package mode** (default): Loads a single package, matches fields, resolves type mismatches (numeric coercion, converter discovery), discovers existing mappers, and generates via `GenerateMulti`. When `-bidi` is set, also generates the reverse mapper.
2. **Cross-package mode** (when `-external-pkg` is set): Loads both packages, discovers getters on the external type, performs cross matching, and generates via `GenerateCross`.

---

## Design Decisions and Trade-offs

**Zero runtime dependencies.** The generated code uses only standard library types and function calls within the user's package. There is no runtime library to import. This means the generated code remains readable, debuggable, and free of version coupling with the generator.

**IIFLs over temporary variables.** The immediately-invoked function literal pattern for pointer conversions trades a small amount of runtime overhead (function call, stack frame) for cleaner generated code structure. Each assignment remains a single `dst.F = <expr>` statement, which simplifies the template and multi-function renderer. The Go compiler inlines these closures in practice, so the runtime cost is negligible.

**Depth-first generation order.** Sub-mappers are enqueued before their parent, which means the generated file reads bottom-up (leaf mappers first, root mapper last). This is a natural consequence of the recursive traversal and matches the dependency order, though Go does not require this ordering for compilation.

**Single output file.** All generated functions for a single `go:generate` invocation are written to one file. This simplifies cleanup (delete one file to remove all generated code) and avoids coordination between multiple output files. The trade-off is that deeply nested type hierarchies produce large generated files, but in practice the generated code is rarely read by humans.

**Convention over configuration for mapper names.** The `Map<Src>To<Dst>` naming convention is enforced for both discovery and generation. This eliminates the need for explicit registration or configuration files, at the cost of requiring developers to follow the convention for their hand-written mappers.
