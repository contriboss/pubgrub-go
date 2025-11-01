# PubGrub-Go

A comprehensive Go implementation of the PubGrub version solving algorithm with semantic versioning, version ranges, and enhanced error reporting.

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Algorithm](https://img.shields.io/badge/Algorithm-CDCL-green.svg)](.)

## Features

- ✨ **Semantic Versioning** - Full semver support with major.minor.patch ordering
- 📊 **Version Ranges** - Complex constraints like `>=1.0.0, <2.0.0 || >=3.0.0`
- 🔍 **Enhanced Error Messages** - Human-readable explanations of why resolution fails
- 🚀 **CDCL Solver** - Conflict-driven clause learning with unit propagation
- 🧪 **Well Tested** - Comprehensive test suite with strong coverage
- ⚡ **Production Ready** - Handles complex dependency graphs efficiently
- 🪵 **Structured Debug Logging** - Plug in `log/slog` via `WithLogger` for rich solver traces

## Origin

This is a derivative work based on the [tinyrange/tinyrange](https://github.com/tinyrange/tinyrange) `experimental/pubgrub` package (v0.2.6). The original implementation was removed from the tinyrange repository. This package preserves the original work and significantly extends it with:

- Semantic versioning support
- Complex version range constraints
- Enhanced error reporting with derivation trees
- Comprehensive test suite
- Production-ready CDCL solver

**Original Copyright:** Copyright 2024 The University of Queensland
**Enhancements:** Copyright 2025 Contriboss
**License:** Apache 2.0

## About PubGrub

PubGrub is a version solving algorithm designed for dependency resolution. This implementation features a **production-ready CDCL (Conflict-Driven Clause Learning) solver** with unit propagation, learned clauses, and intelligent backtracking for efficient dependency resolution. The solver includes optional incompatibility tracking for generating detailed, human-readable error messages with derivation trees.

## Installation

**Requirements:** Go 1.25 or later

```bash
go get github.com/contriboss/pubgrub-go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/contriboss/pubgrub-go"
)

func main() {
    // Create a root source with your initial requirements
    root := pubgrub.NewRootSource()
    root.AddPackage("mypackage", pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("1.0.0")})

    // Create an in-memory source for your package repository
    source := &pubgrub.InMemorySource{}
    source.AddPackage("mypackage", pubgrub.SimpleVersion("1.0.0"), []pubgrub.Term{
        pubgrub.NewTerm("dependency", pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("2.0.0")}),
    })
    source.AddPackage("dependency", pubgrub.SimpleVersion("2.0.0"), nil)

    // Create a solver and solve
    solver := pubgrub.NewSolver(root, source)
    solution, err := solver.Solve(root.Term())
    if err != nil {
        panic(err)
    }

    fmt.Println("Solution:", solution)
}
```

### Using New Features: Version Ranges + Better Errors

```go
package main

import (
    "fmt"
    "os"
    "github.com/contriboss/pubgrub-go"
)

func main() {
    // Use semantic versioning
    webVersion, _ := pubgrub.ParseSemanticVersion("1.0.0")

    // Create requirements with version ranges
    httpRange, _ := pubgrub.ParseVersionRange(">=2.0.0, <3.0.0")
    jsonRange, _ := pubgrub.ParseVersionRange(">=1.5.0")

    root := pubgrub.NewRootSource()
    root.AddPackage("web", pubgrub.EqualsCondition{Version: webVersion})

    source := &pubgrub.InMemorySource{}
    source.AddPackage("web", webVersion, []pubgrub.Term{
        pubgrub.NewTerm("http", pubgrub.NewVersionSetCondition(httpRange)),
        pubgrub.NewTerm("json", pubgrub.NewVersionSetCondition(jsonRange)),
    })

    http2, _ := pubgrub.ParseSemanticVersion("2.5.0")
    json15, _ := pubgrub.ParseSemanticVersion("1.5.0")
    source.AddPackage("http", http2, nil)
    source.AddPackage("json", json15, nil)

    // Enable enhanced error reporting
    solver := pubgrub.NewSolver(root, source).EnableIncompatibilityTracking()
    solution, err := solver.Solve(root.Term())

    if err != nil {
        if nsErr, ok := err.(*pubgrub.NoSolutionError); ok {
            fmt.Fprintln(os.Stderr, nsErr.Error())
        } else {
            fmt.Fprintln(os.Stderr, err)
        }
        os.Exit(1)
    }

    for _, nv := range solution {
        fmt.Printf("✓ %s: %s\n", nv.Name, nv.Version)
    }
}
```

### Debug Logging

```go
package main

import (
    "log/slog"
    "os"

    "github.com/contriboss/pubgrub-go"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    root := pubgrub.NewRootSource()
    source := &pubgrub.InMemorySource{}

    solver := pubgrub.NewSolverWithOptions(
        []pubgrub.Source{root, source},
        pubgrub.WithLogger(logger),
    )

    if _, err := solver.Solve(root.Term()); err != nil {
        logger.Error("resolution failed", "err", err)
    }
}
```

## Core Concepts

### Versions

```go
// Simple string-based versions (original)
v1 := pubgrub.SimpleVersion("1.0.0")

// Semantic versioning (new)
v2, _ := pubgrub.ParseSemanticVersion("1.2.3")
v3, _ := pubgrub.ParseSemanticVersion("2.0.0-alpha.1")
```

### Version Constraints

```go
// Exact match (original)
cond := pubgrub.EqualsCondition{Version: v}

// Version ranges (new)
set, _ := pubgrub.ParseVersionRange(">=1.0.0, <2.0.0")
cond := pubgrub.NewVersionSetCondition(set)

// Operators: >=, >, <=, <, ==, !=
// Compound: ">=1.0.0, <2.0.0" (AND)
// Union: ">=1.0.0 || >=3.0.0" (OR)
// Wildcard: "*" (any version)
```

### Custom Conditions

You can create custom version constraints by implementing the `Condition` interface and optionally the `VersionSetConverter` interface for CDCL solver support:

```go
// Custom caret constraint (like npm's ^1.2.3)
type CaretCondition struct {
    Base *SemanticVersion
}

func (cc CaretCondition) String() string {
    return fmt.Sprintf("^%s", cc.Base)
}

func (cc CaretCondition) Satisfies(ver Version) bool {
    sv, ok := ver.(*SemanticVersion)
    if !ok { return false }
    return sv.Major == cc.Base.Major && sv.Sort(cc.Base) >= 0
}

// Implement VersionSetConverter to enable CDCL solver support
func (cc CaretCondition) ToVersionSet() VersionSet {
    rangeStr := fmt.Sprintf(">=%d.%d.%d, <%d.0.0",
        cc.Base.Major, cc.Base.Minor, cc.Base.Patch,
        cc.Base.Major+1)
    set, _ := ParseVersionRange(rangeStr)
    return set
}

// Use it with the solver
base, _ := ParseSemanticVersion("1.2.0")
condition := CaretCondition{Base: base}
root.AddPackage("mylib", condition)
```

The `VersionSetConverter` interface enables your custom condition to participate in set operations (union, intersection, complement) required by the CDCL solver. Without it, custom conditions work for simple resolution but may fail in complex scenarios requiring conflict analysis.

### Error Reporting

```go
// Simple errors (default)
solver := pubgrub.NewSolver(root, source)
_, err := solver.Solve(root.Term())

// Enhanced errors (opt-in)
solver.EnableIncompatibilityTracking()
if nsErr, ok := err.(*pubgrub.NoSolutionError); ok {
    fmt.Println(nsErr.Error()) // Human-readable explanation
}
```

### Solver Configuration

You can tune the solver with functional options when constructing it, or update an existing instance:

```go
solver := pubgrub.NewSolverWithOptions(
    []pubgrub.Source{root, source},
    pubgrub.WithIncompatibilityTracking(true),
    pubgrub.WithMaxSteps(10_000), // 0 disables the watchdog
)

// Adjust configuration later if needed
solver.Configure(pubgrub.WithMaxSteps(0))
```

`WithIncompatibilityTracking` toggles derivation tree generation, while `WithMaxSteps` caps (or disables) the internal propagation watchdog used to detect runaway scenarios.

### Performance Optimization with Caching

For sources with expensive I/O operations (network, disk, database), wrap them with `CachedSource`:

```go
// Wrap an expensive source (e.g., HTTP API, database)
expensiveSource := &MyRegistrySource{} // implements Source
cached := pubgrub.NewCachedSource(expensiveSource)

solver := pubgrub.NewSolver(root, cached)
solution, _ := solver.Solve(root.Term())

// Check cache performance
stats := cached.GetCacheStats()
fmt.Printf("Cache hit rate: %.1f%%\n", stats.OverallHitRate * 100)
```

**When to use caching:**
- ✅ Network sources (package registries, APIs)
- ✅ Database or file system sources
- ✅ Multiple dependency resolutions with the same source
- ❌ InMemorySource (already fast, adds ~3-5% overhead)
- ❌ Single-shot resolutions

## API Reference

### Core Types
- **`Name`** - Package name identifier
- **`Version`** - Interface for version representation
- **`Condition`** - Interface for version constraints
- **`VersionSetConverter`** - Optional interface for custom conditions to enable CDCL solver support
- **`Term`** - Package name with constraint
- **`Source`** - Package version/dependency queries
- **`Solution`** - Resolved package versions
- **`VersionSet`** - Set of versions with operations

### Implementations
- **`SimpleVersion`** - String-based version (original)
- **`SemanticVersion`** - Full semver support (new)
- **`EqualsCondition`** - Exact match (original)
- **`VersionSetCondition`** - Version ranges (new)
- **`InMemorySource`** - In-memory repository
- **`CachedSource`** - Caching wrapper for expensive sources (new)
- **`CombinedSource`** - Multiple sources
- **`RootSource`** - Initial requirements

### Solver
- **`NewSolver(sources...)`** - Create solver with defaults
- **`NewSolverWithOptions([]Source, ...SolverOption)`** - Create solver with configuration options
- **`Solve(root)`** - Solve dependencies
- **`EnableIncompatibilityTracking()`** - Enable detailed errors
- **`Configure(...SolverOption)`** - Adjust options after construction
- **`GetIncompatibilities()`** - Get tracked conflicts

### Error Types
- **`ErrNoSolutionFound`** - Simple error (original)
- **`NoSolutionError`** - Enhanced error (new)
- **`ErrIterationLimit`** - Solver exceeded configured step limit
- **`DefaultReporter`** / **`CollapsedReporter`** - Error formatters (new)

## Examples

See runnable examples in test files:

```bash
go test -v -run Example
```

- `ExampleVersionSetCondition` - Using version ranges with solver
- `ExampleParseVersionRange` - Parsing and testing version ranges
- `ExampleSemanticVersion` - Semantic version parsing
- `ExampleNoSolutionError_defaultReporter` - Enhanced error messages
- `ExampleNoSolutionError_collapsedReporter` - Compact error format

## Documentation

- **[API Documentation](https://pkg.go.dev/github.com/contriboss/pubgrub-go)** - Full API reference (when published)
- See example tests for usage patterns (`go test -v -run Example`)

## Testing

```bash
# Run all tests
go test -v

# Run with coverage
go test -cover  # 82.7% coverage

# Run specific test suites
go test -v -run TestBackwardCompatibility
go test -v -run TestSemanticVersion
go test -v -run TestVersionRange
go test -v -run TestSolver

# Run benchmarks
go test -bench=. -benchmem
```

## Benchmarks

Performance characteristics on Apple M1 Max:

| Scenario | Time/op | Memory/op | Ops/sec |
|----------|---------|-----------|---------|
| Simple linear chain (4 packages) | 14.7μs | 17.3KB | 67,000 |
| Diamond dependency | 19.4μs | 22.5KB | 51,000 |
| Complex graph (10 packages) | 43.7μs | 48.7KB | 23,000 |
| Deep chain (20 levels) | 92.0μs | 98.8KB | 10,800 |
| Conflict detection | 14.9μs | 17.5KB | 67,000 |

**Key Insights:**
- ⚡ Most resolutions complete in **microseconds**
- 🎯 Conflict detection is as fast as successful resolution
- 📊 Incompatibility tracking adds **<1% overhead**
- 🚀 Scales linearly with dependency graph complexity

Run `go test -bench=. -benchmem` to see performance on your system.

## Performance & Status

- **Test Coverage:** Strong coverage with comprehensive test suite
- **Tests:** All passing
- **Algorithm:** CDCL (Conflict-Driven Clause Learning) with unit propagation
- **Memory:** Minimal; learned clauses and optional tracking add derivation trees
- **Caching:** Optional CachedSource wrapper for expensive I/O operations
- **Production Ready:** Yes, handles complex dependency graphs efficiently

## Contributing

Contributions welcome! Please:
1. Add tests for new features
2. Update documentation
3. Follow existing code style
4. Ensure all tests pass

## Attribution

This package is derived from the tinyrange project:
- **Original Repository:** https://github.com/tinyrange/tinyrange
- **Original Package:** experimental/pubgrub
- **Version:** v0.3.3 (Nov 1, 2025)
- **Original Copyright:** Copyright 2024 The University of Queensland
- **Original License:** Apache 2.0

Enhancements and extensions:
- **Semantic Versioning:** Copyright 2025 Contriboss
- **Version Ranges:** Copyright 2025 Contriboss
- **Error Reporting:** Copyright 2025 Contriboss
- **License:** Apache 2.0

## License

Apache License 2.0 - See [LICENSE](LICENSE) file for details.

This project maintains the same Apache 2.0 license as the original tinyrange implementation and includes proper attribution as required by the license terms.

## Credits

This implementation builds upon the following works:

1. **Weizenbaum, N.** (2018). *PubGrub: Next-Generation Version Solving*. Medium. https://medium.com/@nex3/pubgrub-2fb6470504f

2. **University of Queensland** (2024). *tinyrange/pubgrub* (v0.2.6). Original Go implementation. https://github.com/tinyrange/tinyrange

3. **Dart Team** (2018-present). *PubGrub Specification*. https://github.com/dart-lang/pub/blob/master/doc/solver.md

4. **pubgrub-rs Contributors** (2019-present). *pubgrub-rs*. Rust reference implementation providing CDCL patterns. https://github.com/pubgrub-rs/pubgrub
