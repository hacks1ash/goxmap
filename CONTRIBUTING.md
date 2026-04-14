# Contributing to goxmap

Thank you for your interest in contributing. This document covers everything you need to get started.

## Prerequisites

- Go 1.22 or later (latest stable recommended)
- Git

## Development Setup

```bash
# Clone the repository
git clone https://github.com/hacks1ash/goxmap.git
cd model-mapper

# Download dependencies
go mod download

# Verify everything builds
go vet ./...

# Run the full test suite
go test -v ./...
```

## Project Structure

```
cmd/mapper-gen/          CLI entry point
internal/
  loader/                Type loading via golang.org/x/tools/go/packages
  matcher/               Field matching algorithms
  generator/             Code generation via text/template and fmt.Fprintf
_examples/               Self-contained runnable examples
benchmarks/              Performance benchmarks
docs/                    Technical documentation
testdata/                Integration test fixtures
```

## Running Tests

```bash
# Unit and integration tests
go test ./...

# With race detector
go test -race ./...

# Verbose output
go test -v ./...

# Benchmarks
go test -bench=. -benchmem ./benchmarks/
```

## Making Changes

### Before You Start

1. Check existing [issues](https://github.com/hacks1ash/goxmap/issues) to see if the work is already planned or in progress.
2. For non-trivial changes, open an issue first to discuss the approach.

### Development Workflow

1. Fork the repository and create a branch from `main`.
2. Make your changes.
3. Add or update tests. All new functionality must have test coverage.
4. Run `go vet ./...` and `go test ./...` to ensure nothing is broken.
5. Run `go generate ./...` if you changed any generated code or generation logic, and commit the results.
6. Write a clear commit message describing what changed and why.
7. Open a pull request against `main`.

### Code Style

- Follow standard Go conventions. Run `gofmt` (or let your editor do it).
- Keep functions focused. If a function does two things, split it.
- Exported functions and types need doc comments. Internal helpers do not need them if the name is self-explanatory.
- Error messages should be lowercase and not end with punctuation, following Go convention.
- Prefer table-driven tests.

### Commit Messages

Use a concise summary line (under 72 characters) followed by a blank line and an optional body explaining the motivation:

```
matcher: support bind_json tag for cross-package field resolution

The bind_json directive allows internal structs to reference external
fields by their json tag value, which is necessary for Protobuf and
OpenAPI-generated structs where field names don't follow Go conventions.
```

Prefix the summary with the package name when the change is scoped to a single package.

### What Makes a Good Pull Request

- **Focused.** One logical change per PR. A bug fix and a feature should be separate PRs.
- **Tested.** New behavior has test coverage. Bug fixes have a regression test.
- **Documented.** If the change affects user-facing behavior, update the README or relevant docs.
- **Backwards compatible.** Breaking changes to the tag syntax or CLI flags require discussion first.

## Architecture

See [docs/TECHNICAL_DESIGN.md](docs/TECHNICAL_DESIGN.md) for a detailed description of the four-stage pipeline (load, match, plan, generate), the recursive mapper queue, pointer conversion logic, and cross-package resolution.

The pipeline has four stages, each with a clear responsibility and a well-defined data boundary:

```
Loader (internal/loader)     StructInfo, GetterInfo
   |
Matcher (internal/matcher)   FieldPair, MatchResult, CrossMatchResult
   |
Planner (internal/generator) funcEntry queue (multiGenerator)
   |
Emitter (internal/generator) formatted Go source code
```

## Adding a New Field Resolver

The matcher currently resolves fields using a 4-level priority chain: `bind` > `bind_json` > `json` > name. If you want to add support for a new resolution strategy (e.g., an OpenAPI resolver, a YAML tag resolver, or a GraphQL alias), follow this pattern.

### Step 1: Add the tag to the loader

In `internal/loader/loader.go`, add a new field to `StructField`:

```go
type StructField struct {
    // ... existing fields ...
    OpenAPIRef string // Value from `mapper:"openapi:operationId"` tag
}
```

Add a parser function and call it from `LoadStructFromPkg`:

```go
func parseMapperOpenAPITag(rawTag string) string {
    tag := reflect.StructTag(rawTag)
    mapperVal, ok := tag.Lookup("mapper")
    if !ok {
        return ""
    }
    for _, part := range strings.Split(mapperVal, ";") {
        part = strings.TrimSpace(part)
        if after, found := strings.CutPrefix(part, "openapi:"); found {
            return after
        }
    }
    return ""
}
```

The semicolon-delimited format ensures your new directive coexists with existing ones (e.g., `mapper:"bind:Foo;openapi:bar"`).

### Step 2: Add the resolution step to the matcher

In `internal/matcher/matcher.go`, insert your resolution step at the correct priority level in `MatchCross`:

```go
// Priority 2.5: openapi tag - find external field by operation ID.
if !matched && inf.OpenAPIRef != "" {
    if ef, ok := extByOpenAPI[inf.OpenAPIRef]; ok {
        extField = ef
        matched = true
    }
}
```

You will also need to build the lookup map from external fields. If your resolver uses a custom annotation on the external side, build it during the external field iteration loop at the top of `MatchCross`.

### Step 3: Add tests

Add test cases to:
- `internal/loader/loader_test.go` -- verify the tag is parsed correctly from testdata structs
- `internal/matcher/matcher_test.go` -- verify the resolution priority (new resolver should not override higher-priority resolvers)
- `internal/generator/generator_test.go` -- verify the generated code uses the correct field access

Add testdata structs to `internal/loader/testdata/models.go`.

### Step 4: Update documentation

- Add the new tag to the "Tag Reference" table in `README.md`
- Add the new tag to the "Directive Reference" table in `docs/TECHNICAL_DESIGN.md`
- If the resolver requires a new CLI flag, add it to `cmd/mapper-gen/main.go` and the CLI Reference section

### Design Principles for Resolvers

- **Higher-priority resolvers win.** If a field has both `bind` and your new tag, `bind` takes precedence. Insert your resolver at the appropriate level.
- **Resolution is single-pass.** The matcher iterates over internal fields exactly once. Do not add a second pass.
- **The loader is the only package that touches `go/types`.** Your resolver should work with the flat `StructField` type, not with raw `types.Type` values.
- **Tags are parsed from the `mapper` tag only.** Do not read from other struct tags (e.g., `openapi:"..."` as a separate tag). The `mapper` tag with semicolon-delimited directives is the single source of truth.

## Releasing

Releases follow [semantic versioning](https://semver.org/). Tags are created on `main` and trigger the CI pipeline. There is no separate release branch.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
