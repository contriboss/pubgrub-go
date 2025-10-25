package pubgrub

import "fmt"

// CachedSource wraps a Source and caches GetVersions and GetDependencies calls
// to improve performance when the same queries are made repeatedly.
//
// WHEN TO USE:
// CachedSource is most beneficial for:
// - Sources with expensive network I/O (package registries, APIs)
// - Sources with disk I/O or database queries
// - Running multiple dependency resolutions without recreating the source
// - Build systems that resolve dependencies repeatedly
//
// WHEN NOT TO USE:
// CachedSource adds ~3-5% overhead for:
// - InMemorySource (already fast, no repeated queries in CDCL solver)
// - Simple, single-shot dependency resolutions
// - Sources where queries are naturally cached upstream
//
// The cache is maintained for the lifetime of the CachedSource instance and
// assumes that version lists and dependencies are immutable during solving.
type CachedSource struct {
	source Source

	// Cache for GetVersions results
	versionsCache     map[Name][]Version
	versionsCalls     int
	versionsCacheHits int

	// Cache for GetDependencies results
	depsCache     map[string][]Term
	depsCalls     int
	depsCacheHits int
}

// NewCachedSource creates a new caching wrapper around the given source.
func NewCachedSource(source Source) *CachedSource {
	return &CachedSource{
		source:        source,
		versionsCache: make(map[Name][]Version),
		depsCache:     make(map[string][]Term),
	}
}

// GetVersions returns all available versions for a package, caching the result.
func (c *CachedSource) GetVersions(name Name) ([]Version, error) {
	c.versionsCalls++

	// Check cache first
	if versions, ok := c.versionsCache[name]; ok {
		c.versionsCacheHits++
		return versions, nil
	}

	// Cache miss - fetch from underlying source
	versions, err := c.source.GetVersions(name)
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.versionsCache[name] = versions
	return versions, nil
}

// GetDependencies returns dependencies for a specific package version, caching the result.
func (c *CachedSource) GetDependencies(name Name, version Version) ([]Term, error) {
	c.depsCalls++

	// Create cache key from name and version
	key := fmt.Sprintf("%s@%s", name.Value(), version)

	// Check cache first
	if deps, ok := c.depsCache[key]; ok {
		c.depsCacheHits++
		return deps, nil
	}

	// Cache miss - fetch from underlying source
	deps, err := c.source.GetDependencies(name, version)
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.depsCache[key] = deps
	return deps, nil
}

// CacheStats returns statistics about cache performance.
type CacheStats struct {
	VersionsCalls     int
	VersionsCacheHits int
	VersionsHitRate   float64

	DepsCalls     int
	DepsCacheHits int
	DepsHitRate   float64

	TotalCalls     int
	TotalCacheHits int
	OverallHitRate float64
}

// GetCacheStats returns cache performance statistics.
func (c *CachedSource) GetCacheStats() CacheStats {
	stats := CacheStats{
		VersionsCalls:     c.versionsCalls,
		VersionsCacheHits: c.versionsCacheHits,
		DepsCalls:         c.depsCalls,
		DepsCacheHits:     c.depsCacheHits,
		TotalCalls:        c.versionsCalls + c.depsCalls,
		TotalCacheHits:    c.versionsCacheHits + c.depsCacheHits,
	}

	if stats.VersionsCalls > 0 {
		stats.VersionsHitRate = float64(stats.VersionsCacheHits) / float64(stats.VersionsCalls)
	}

	if stats.DepsCalls > 0 {
		stats.DepsHitRate = float64(stats.DepsCacheHits) / float64(stats.DepsCalls)
	}

	if stats.TotalCalls > 0 {
		stats.OverallHitRate = float64(stats.TotalCacheHits) / float64(stats.TotalCalls)
	}

	return stats
}

// ClearCache clears all cached data while preserving the underlying source.
func (c *CachedSource) ClearCache() {
	c.versionsCache = make(map[Name][]Version)
	c.depsCache = make(map[string][]Term)
	c.versionsCalls = 0
	c.versionsCacheHits = 0
	c.depsCalls = 0
	c.depsCacheHits = 0
}
