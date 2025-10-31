package pubgrub

import (
	"testing"
)

// TestComplexRubyGemsScenario tests a more realistic scenario with multiple
// packages that all transitively depend on shared dependencies.
//
// This simulates what happens in a real Rails project where many gems
// depend on common utilities like rubyzip, and wrong version choices
// early in the search can lead to dead ends.
//
// Package structure:
// - root → [roo, rubyXL, caxlsx, another_gem]
// - All four packages use rubyzip with different constraints
// - PubGrub must choose rubyzip version that works for ALL of them
//
// This test will help identify if PubGrub's unit propagation and
// conflict learning are working efficiently.
func TestComplexRubyGemsScenario(t *testing.T) {
	source := NewMapSource()

	// Add rubyzip versions
	source.Add("rubyzip", "1.3.0", nil)
	source.Add("rubyzip", "2.3.0", nil)
	source.Add("rubyzip", "2.4.0", nil)
	source.Add("rubyzip", "2.4.1", nil)
	source.Add("rubyzip", "3.0.0", nil)
	source.Add("rubyzip", "3.1.0", nil)

	// Add roo versions - note that OLD versions require rubyzip >= 3.0
	source.Add("roo", "2.1.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 3.0.0, < 4.0.0"},
	})
	source.Add("roo", "2.5.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 3.0.0, < 4.0.0"},
	})
	source.Add("roo", "2.9.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 3.0.0, < 4.0.0"},
	})
	// Only 2.10.1 and 3.0.0 are compatible with rubyzip 2.x
	source.Add("roo", "2.10.1", []Dependency{
		{Name: "rubyzip", Constraint: ">= 1.3.0, < 3.0.0"},
	})
	source.Add("roo", "3.0.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 3.0.0, < 4.0.0"},
	})

	// Add rubyXL versions - all require rubyzip ~> 2.4
	source.Add("rubyXL", "3.4.14", []Dependency{
		{Name: "rubyzip", Constraint: ">= 2.4.0, < 3.0.0"},
	})
	source.Add("rubyXL", "3.4.25", []Dependency{
		{Name: "rubyzip", Constraint: ">= 2.4.0, < 3.0.0"},
	})
	source.Add("rubyXL", "3.4.34", []Dependency{
		{Name: "rubyzip", Constraint: ">= 2.4.0, < 3.0.0"},
	})

	// Add caxlsx which also depends on rubyzip
	source.Add("caxlsx", "3.3.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 1.6.0, < 3.0.0"},
	})
	source.Add("caxlsx", "4.0.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 2.3.0, < 4.0.0"},
	})

	// Add another gem that prefers older rubyzip
	source.Add("zip_tricks", "5.6.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 1.3.0, < 3.0.0"},
	})

	// Root depends on all four packages
	rootSource := NewRootSource()
	rootSource.AddPackage(MakeName("roo"), NewAnyVersionCondition())
	rootSource.AddPackage(MakeName("rubyXL"), NewAnyVersionCondition())
	rootSource.AddPackage(MakeName("caxlsx"), NewAnyVersionCondition())
	rootSource.AddPackage(MakeName("zip_tricks"), NewAnyVersionCondition())

	// Create solver
	solver := NewSolver(rootSource, source)

	// Solve
	solution, err := solver.Solve(rootSource.Term())
	if err != nil {
		t.Fatalf("Expected solution but got error: %v", err)
	}

	// Verify solution
	solutionMap := make(map[string]string)
	for _, pkg := range solution {
		if pkg.Name.Value() != "$$root" {
			solutionMap[pkg.Name.Value()] = pkg.Version.String()
		}
	}

	// The only valid solution should use rubyzip 2.4.x
	// because that's the intersection of all constraints:
	// - roo 2.10.1: >= 1.3.0, < 3.0.0
	// - rubyXL: >= 2.4.0, < 3.0.0
	// - caxlsx: depends on which version, but should work with 2.4.x
	// - zip_tricks: >= 1.3.0, < 3.0.0
	//
	// Intersection: >= 2.4.0, < 3.0.0 → rubyzip 2.4.1

	if solutionMap["roo"] != "2.10.1" {
		t.Errorf("Expected roo 2.10.1, got %s", solutionMap["roo"])
	}
	if solutionMap["rubyzip"] < "2.4.0" || solutionMap["rubyzip"] >= "3.0.0" {
		t.Errorf("Expected rubyzip in [2.4.0, 3.0.0), got %s", solutionMap["rubyzip"])
	}

	// Print solution for debugging
	t.Logf("Solution found:")
	for name, version := range solutionMap {
		t.Logf("  %s = %s", name, version)
	}
}
