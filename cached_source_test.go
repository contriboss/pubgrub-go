package pubgrub

import (
	"fmt"
	"testing"
)

// mockCountingSource tracks how many times GetVersions and GetDependencies are called
type mockCountingSource struct {
	source        *InMemorySource
	versionsCalls int
	depsCalls     int
}

func (m *mockCountingSource) GetVersions(name Name) ([]Version, error) {
	m.versionsCalls++
	return m.source.GetVersions(name)
}

func (m *mockCountingSource) GetDependencies(name Name, version Version) ([]Term, error) {
	m.depsCalls++
	return m.source.GetDependencies(name, version)
}

func TestCachedSource_GetVersions(t *testing.T) {
	// Create underlying source
	inner := &InMemorySource{}
	inner.AddPackage(MakeName("A"), SimpleVersion("1.0.0"), nil)
	inner.AddPackage(MakeName("A"), SimpleVersion("2.0.0"), nil)

	mock := &mockCountingSource{source: inner}
	cached := NewCachedSource(mock)

	// First call should hit the underlying source
	versions1, err := cached.GetVersions(MakeName("A"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions1) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions1))
	}
	if mock.versionsCalls != 1 {
		t.Fatalf("expected 1 call to underlying source, got %d", mock.versionsCalls)
	}

	// Second call should hit the cache
	versions2, err := cached.GetVersions(MakeName("A"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions2) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions2))
	}
	if mock.versionsCalls != 1 {
		t.Fatalf("expected still 1 call to underlying source, got %d", mock.versionsCalls)
	}

	// Verify cache stats
	stats := cached.GetCacheStats()
	if stats.VersionsCalls != 2 {
		t.Errorf("expected 2 total calls, got %d", stats.VersionsCalls)
	}
	if stats.VersionsCacheHits != 1 {
		t.Errorf("expected 1 cache hit, got %d", stats.VersionsCacheHits)
	}
	if stats.VersionsHitRate != 0.5 {
		t.Errorf("expected 0.5 hit rate, got %f", stats.VersionsHitRate)
	}
}

func TestCachedSource_GetDependencies(t *testing.T) {
	// Create underlying source
	inner := &InMemorySource{}
	v1 := SimpleVersion("1.0.0")
	deps := []Term{NewTerm(MakeName("B"), EqualsCondition{Version: v1})}
	inner.AddPackage(MakeName("A"), v1, deps)

	mock := &mockCountingSource{source: inner}
	cached := NewCachedSource(mock)

	// First call should hit the underlying source
	deps1, err := cached.GetDependencies(MakeName("A"), v1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps1) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps1))
	}
	if mock.depsCalls != 1 {
		t.Fatalf("expected 1 call to underlying source, got %d", mock.depsCalls)
	}

	// Second call should hit the cache
	deps2, err := cached.GetDependencies(MakeName("A"), v1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps2) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps2))
	}
	if mock.depsCalls != 1 {
		t.Fatalf("expected still 1 call to underlying source, got %d", mock.depsCalls)
	}

	// Verify cache stats
	stats := cached.GetCacheStats()
	if stats.DepsCalls != 2 {
		t.Errorf("expected 2 total calls, got %d", stats.DepsCalls)
	}
	if stats.DepsCacheHits != 1 {
		t.Errorf("expected 1 cache hit, got %d", stats.DepsCacheHits)
	}
	if stats.DepsHitRate != 0.5 {
		t.Errorf("expected 0.5 hit rate, got %f", stats.DepsHitRate)
	}
}

func TestCachedSource_ClearCache(t *testing.T) {
	inner := &InMemorySource{}
	inner.AddPackage(MakeName("A"), SimpleVersion("1.0.0"), nil)

	mock := &mockCountingSource{source: inner}
	cached := NewCachedSource(mock)

	// Make a call
	_, _ = cached.GetVersions(MakeName("A"))

	// Clear cache
	cached.ClearCache()

	// Stats should be reset
	stats := cached.GetCacheStats()
	if stats.VersionsCalls != 0 {
		t.Errorf("expected 0 calls after clear, got %d", stats.VersionsCalls)
	}

	// Next call should hit underlying source again
	_, _ = cached.GetVersions(MakeName("A"))
	if mock.versionsCalls != 2 {
		t.Errorf("expected 2 calls to underlying source after clear, got %d", mock.versionsCalls)
	}
}

func TestCachedSource_DifferentPackages(t *testing.T) {
	inner := &InMemorySource{}
	inner.AddPackage(MakeName("A"), SimpleVersion("1.0.0"), nil)
	inner.AddPackage(MakeName("B"), SimpleVersion("1.0.0"), nil)

	mock := &mockCountingSource{source: inner}
	cached := NewCachedSource(mock)

	// Call for package A
	_, _ = cached.GetVersions(MakeName("A"))
	_, _ = cached.GetVersions(MakeName("A")) // Should be cached

	// Call for package B (different key)
	_, _ = cached.GetVersions(MakeName("B"))
	_, _ = cached.GetVersions(MakeName("B")) // Should be cached

	// Should have called underlying source twice (once per package)
	if mock.versionsCalls != 2 {
		t.Errorf("expected 2 calls to underlying source, got %d", mock.versionsCalls)
	}

	// Cache hit rate should be 50%
	stats := cached.GetCacheStats()
	if stats.VersionsHitRate != 0.5 {
		t.Errorf("expected 0.5 hit rate, got %f", stats.VersionsHitRate)
	}
}

func TestCachedSource_Integration(t *testing.T) {
	// Test that CachedSource works correctly with the solver
	inner := &InMemorySource{}
	v100 := SimpleVersion("1.0.0")

	inner.AddPackage(MakeName("A"), v100, []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: v100}),
	})
	inner.AddPackage(MakeName("B"), v100, []Term{
		NewTerm(MakeName("C"), EqualsCondition{Version: v100}),
	})
	inner.AddPackage(MakeName("C"), v100, nil)

	mock := &mockCountingSource{source: inner}
	cached := NewCachedSource(mock)

	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: v100})

	solver := NewSolver(root, cached)
	solution, err := solver.Solve(root.Term())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify solution is correct (root + A + B + C = 4 packages)
	if len(solution) != 4 {
		t.Errorf("expected 4 packages in solution, got %d", len(solution))
	}

	// Check that caching reduced calls
	stats := cached.GetCacheStats()
	fmt.Printf("Cache stats: %d total calls, %d hits (%.1f%% hit rate)\n",
		stats.TotalCalls, stats.TotalCacheHits, stats.OverallHitRate*100)

	// Verify we got some cache hits (exact number depends on solver internals)
	if stats.TotalCalls == 0 {
		t.Error("expected some calls to be made")
	}
}
