package pubgrub

import (
	"fmt"
	"testing"
)

// Benchmark scenarios for CDCL solver performance testing

// BenchmarkSimpleLinearChain tests a simple linear dependency chain
// A -> B -> C -> D
func BenchmarkSimpleLinearChain(b *testing.B) {
	source := &InMemorySource{}

	// Create linear chain: A depends on B, B depends on C, C depends on D
	source.AddPackage(MakeName("A"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("B"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("C"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("C"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("D"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("D"), SimpleVersion("1.0.0"), nil)

	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source)

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkDiamondDependency tests classic diamond dependency
//
//	  A
//	 / \
//	B   C
//	 \ /
//	  D
func BenchmarkDiamondDependency(b *testing.B) {
	source := &InMemorySource{}

	v100 := SimpleVersion("1.0.0")

	// A depends on both B and C
	source.AddPackage(MakeName("A"), v100, []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("C"), EqualsCondition{Version: v100}),
	})
	// Both B and C depend on D
	source.AddPackage(MakeName("B"), v100, []Term{
		NewTerm(MakeName("D"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("C"), v100, []Term{
		NewTerm(MakeName("D"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("D"), v100, nil)

	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: v100})

	solver := NewSolver(root, source)

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkMultipleVersions tests version selection with multiple available versions
func BenchmarkMultipleVersions(b *testing.B) {
	source := &InMemorySource{}

	// Package A with 10 versions
	for i := 1; i <= 10; i++ {
		ver := SimpleVersion(fmt.Sprintf("1.0.%d", i))
		deps := []Term{}
		if i > 1 {
			// Each version depends on B
			deps = append(deps, NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("1.0.0")}))
		}
		source.AddPackage(MakeName("A"), ver, deps)
	}
	source.AddPackage(MakeName("B"), SimpleVersion("1.0.0"), nil)

	// Request latest version (solver should pick 1.0.10)
	root := NewRootSource()
	vrange, _ := ParseVersionRange(">=1.0.0")
	root.AddPackage(MakeName("A"), NewVersionSetCondition(vrange))

	solver := NewSolver(root, source)

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkComplexGraph tests a more realistic dependency graph
// 10 packages with cross-dependencies
func BenchmarkComplexGraph(b *testing.B) {
	source := &InMemorySource{}

	v100 := SimpleVersion("1.0.0")

	// Create a web of dependencies
	source.AddPackage(MakeName("web"), v100, []Term{
		NewTerm(MakeName("http"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("json"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("template"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("http"), v100, []Term{
		NewTerm(MakeName("net"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("crypto"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("json"), v100, []Term{
		NewTerm(MakeName("encoding"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("template"), v100, []Term{
		NewTerm(MakeName("text"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("html"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("net"), v100, nil)
	source.AddPackage(MakeName("crypto"), v100, []Term{
		NewTerm(MakeName("math"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("encoding"), v100, nil)
	source.AddPackage(MakeName("text"), v100, nil)
	source.AddPackage(MakeName("html"), v100, []Term{
		NewTerm(MakeName("text"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("math"), v100, nil)

	root := NewRootSource()
	root.AddPackage(MakeName("web"), EqualsCondition{Version: v100})

	solver := NewSolver(root, source)

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkBacktracking tests scenario requiring backtracking
// A depends on B>=2.0, C depends on B<2.0, D has versions compatible with both
func BenchmarkBacktracking(b *testing.B) {
	source := &InMemorySource{}

	v100, _ := ParseSemanticVersion("1.0.0")
	v200, _ := ParseSemanticVersion("2.0.0")
	v210, _ := ParseSemanticVersion("2.1.0")

	rangeGte2, _ := ParseVersionRange(">=2.0.0")
	rangeLt2, _ := ParseVersionRange("<2.0.0")

	source.AddPackage(MakeName("A"), v100, []Term{
		NewTerm(MakeName("B"), NewVersionSetCondition(rangeGte2)),
	})
	source.AddPackage(MakeName("C"), v100, []Term{
		NewTerm(MakeName("B"), NewVersionSetCondition(rangeLt2)),
	})

	// B has multiple versions
	source.AddPackage(MakeName("B"), v100, nil)
	source.AddPackage(MakeName("B"), v200, nil)
	source.AddPackage(MakeName("B"), v210, nil)

	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: v100})

	solver := NewSolver(root, source)

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkConflictDetection tests CDCL's conflict detection and learning
// Incompatible version requirements that should fail quickly
func BenchmarkConflictDetection(b *testing.B) {
	source := &InMemorySource{}

	source.AddPackage(MakeName("A"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("B"), SimpleVersion("1.0.0"), nil)
	source.AddPackage(MakeName("B"), SimpleVersion("2.0.0"), nil)
	source.AddPackage(MakeName("C"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("2.0.0")}),
	})

	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	root.AddPackage(MakeName("C"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source)

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err == nil {
			b.Fatal("expected conflict but got solution")
		}
	}
}

// BenchmarkWithTracking measures overhead of incompatibility tracking
func BenchmarkWithTracking(b *testing.B) {
	source := &InMemorySource{}

	source.AddPackage(MakeName("A"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("B"), SimpleVersion("1.0.0"), nil)
	source.AddPackage(MakeName("C"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("2.0.0")}),
	})

	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	root.AddPackage(MakeName("C"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source).EnableIncompatibilityTracking()

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err == nil {
			b.Fatal("expected conflict")
		}
	}
}

// BenchmarkDeepDependencyChain tests performance with deep chains
func BenchmarkDeepDependencyChain(b *testing.B) {
	source := &InMemorySource{}

	depth := 20

	// Create chain: pkg0 -> pkg1 -> pkg2 -> ... -> pkg19
	for i := 0; i < depth; i++ {
		pkg := fmt.Sprintf("pkg%d", i)
		ver := SimpleVersion("1.0.0")

		var deps []Term
		if i < depth-1 {
			nextPkg := fmt.Sprintf("pkg%d", i+1)
			deps = []Term{
				NewTerm(MakeName(nextPkg), EqualsCondition{Version: ver}),
			}
		}
		source.AddPackage(MakeName(pkg), ver, deps)
	}

	root := NewRootSource()
	root.AddPackage(MakeName("pkg0"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source)

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkWideGraph tests many dependencies at one level
func BenchmarkWideGraph(b *testing.B) {
	source := &InMemorySource{}

	width := 20
	ver := SimpleVersion("1.0.0")

	// Root depends on pkg0, pkg1, ..., pkg19
	deps := make([]Term, width)
	for i := 0; i < width; i++ {
		pkg := fmt.Sprintf("pkg%d", i)
		deps[i] = NewTerm(MakeName(pkg), EqualsCondition{Version: ver})
		source.AddPackage(MakeName(pkg), ver, nil)
	}

	source.AddPackage(MakeName("root"), ver, deps)

	root := NewRootSource()
	root.AddPackage(MakeName("root"), EqualsCondition{Version: ver})

	solver := NewSolver(root, source)

	b.ResetTimer()
	for b.Loop() {
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkCached_SimpleLinearChain tests caching on simple linear chain
func BenchmarkCached_SimpleLinearChain(b *testing.B) {
	source := &InMemorySource{}

	// Create linear chain: A depends on B, B depends on C, C depends on D
	source.AddPackage(MakeName("A"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("B"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("C"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("C"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("D"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("D"), SimpleVersion("1.0.0"), nil)

	cached := NewCachedSource(source)
	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, cached)

	b.ResetTimer()
	for b.Loop() {
		cached.ClearCache() // Clear cache between iterations for fair comparison
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkCached_ComplexGraph tests caching on complex graph
func BenchmarkCached_ComplexGraph(b *testing.B) {
	source := &InMemorySource{}

	v100 := SimpleVersion("1.0.0")

	// Create a web of dependencies
	source.AddPackage(MakeName("web"), v100, []Term{
		NewTerm(MakeName("http"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("json"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("template"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("http"), v100, []Term{
		NewTerm(MakeName("net"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("crypto"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("json"), v100, []Term{
		NewTerm(MakeName("encoding"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("template"), v100, []Term{
		NewTerm(MakeName("text"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("html"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("net"), v100, nil)
	source.AddPackage(MakeName("crypto"), v100, []Term{
		NewTerm(MakeName("math"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("encoding"), v100, nil)
	source.AddPackage(MakeName("text"), v100, nil)
	source.AddPackage(MakeName("html"), v100, []Term{
		NewTerm(MakeName("text"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("math"), v100, nil)

	cached := NewCachedSource(source)
	root := NewRootSource()
	root.AddPackage(MakeName("web"), EqualsCondition{Version: v100})

	solver := NewSolver(root, cached)

	b.ResetTimer()
	for b.Loop() {
		cached.ClearCache()
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkCached_DeepChain tests caching on deep dependency chain
func BenchmarkCached_DeepChain(b *testing.B) {
	source := &InMemorySource{}

	depth := 20

	// Create chain: pkg0 -> pkg1 -> pkg2 -> ... -> pkg19
	for i := 0; i < depth; i++ {
		pkg := fmt.Sprintf("pkg%d", i)
		ver := SimpleVersion("1.0.0")

		var deps []Term
		if i < depth-1 {
			nextPkg := fmt.Sprintf("pkg%d", i+1)
			deps = []Term{
				NewTerm(MakeName(nextPkg), EqualsCondition{Version: ver}),
			}
		}
		source.AddPackage(MakeName(pkg), ver, deps)
	}

	cached := NewCachedSource(source)
	root := NewRootSource()
	root.AddPackage(MakeName("pkg0"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, cached)

	b.ResetTimer()
	for b.Loop() {
		cached.ClearCache()
		_, err := solver.Solve(root.Term())
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkCacheReuse tests cache benefits across multiple solves
// This demonstrates the real-world benefit of caching when running
// multiple dependency resolutions without clearing the cache
func BenchmarkCacheReuse(b *testing.B) {
	source := &InMemorySource{}
	v100 := SimpleVersion("1.0.0")

	// Create a shared dependency graph
	source.AddPackage(MakeName("web"), v100, []Term{
		NewTerm(MakeName("http"), EqualsCondition{Version: v100}),
		NewTerm(MakeName("json"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("http"), v100, []Term{
		NewTerm(MakeName("net"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("json"), v100, []Term{
		NewTerm(MakeName("encoding"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("net"), v100, nil)
	source.AddPackage(MakeName("encoding"), v100, nil)

	// Create multiple root requirements that share dependencies
	source.AddPackage(MakeName("app1"), v100, []Term{
		NewTerm(MakeName("web"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("app2"), v100, []Term{
		NewTerm(MakeName("http"), EqualsCondition{Version: v100}),
	})
	source.AddPackage(MakeName("app3"), v100, []Term{
		NewTerm(MakeName("json"), EqualsCondition{Version: v100}),
	})

	cached := NewCachedSource(source)

	b.Run("WithCache", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			// Solve app1 (caches web, http, json, net, encoding)
			root1 := NewRootSource()
			root1.AddPackage(MakeName("app1"), EqualsCondition{Version: v100})
			solver1 := NewSolver(root1, cached)
			_, _ = solver1.Solve(root1.Term())

			// Solve app2 (reuses cached http, net)
			root2 := NewRootSource()
			root2.AddPackage(MakeName("app2"), EqualsCondition{Version: v100})
			solver2 := NewSolver(root2, cached)
			_, _ = solver2.Solve(root2.Term())

			// Solve app3 (reuses cached json, encoding)
			root3 := NewRootSource()
			root3.AddPackage(MakeName("app3"), EqualsCondition{Version: v100})
			solver3 := NewSolver(root3, cached)
			_, _ = solver3.Solve(root3.Term())
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			// Solve app1
			root1 := NewRootSource()
			root1.AddPackage(MakeName("app1"), EqualsCondition{Version: v100})
			solver1 := NewSolver(root1, source)
			_, _ = solver1.Solve(root1.Term())

			// Solve app2
			root2 := NewRootSource()
			root2.AddPackage(MakeName("app2"), EqualsCondition{Version: v100})
			solver2 := NewSolver(root2, source)
			_, _ = solver2.Solve(root2.Term())

			// Solve app3
			root3 := NewRootSource()
			root3.AddPackage(MakeName("app3"), EqualsCondition{Version: v100})
			solver3 := NewSolver(root3, source)
			_, _ = solver3.Solve(root3.Term())
		}
	})
}
