package pubgrub

import (
	"fmt"
	"testing"
)

// TestRubyGemsRooRubyXLConflict tests a real-world scenario from Ruby gems
// where PubGrub should find a solution but currently fails.
//
// The scenario:
// - Root depends on: roo (any) and rubyXL (any)
// - roo has versions: 2.1.0, 2.10.1, 3.0.0
//   - roo 2.1.0 depends on rubyzip >= 3.0.0, < 4.0.0
//   - roo 2.10.1 depends on rubyzip >= 1.3.0, < 3.0.0  (compatible!)
//   - roo 3.0.0 depends on rubyzip >= 3.0.0, < 4.0.0
//
// - rubyXL has versions: 3.4.14, 3.4.34
//   - rubyXL 3.4.14 depends on rubyzip ~> 2.4 (>= 2.4.0, < 3.0.0)
//   - rubyXL 3.4.34 depends on rubyzip ~> 2.4 (>= 2.4.0, < 3.0.0)
//
// - rubyzip has versions: 2.3.0, 2.4.0, 2.4.1, 3.0.0
//
// EXPECTED SOLUTION:
// - roo 2.10.1 (requires rubyzip >= 1.3.0, < 3.0.0)
// - rubyXL 3.4.34 (requires rubyzip >= 2.4.0, < 3.0.0)
// - rubyzip 2.4.1 (satisfies both: >= 2.4.0 AND < 3.0.0)
//
// CURRENT BUG:
// PubGrub explores roo 2.1.0 first (due to search order), learns it conflicts
// with rubyXL, then excludes ALL versions without trying roo 2.10.1.
func TestRubyGemsRooRubyXLConflict(t *testing.T) {
	// Create mock source with Ruby gem versions
	source := NewMapSource()

	// Add rubyzip versions (dependency)
	source.Add("rubyzip", "2.3.0", nil)
	source.Add("rubyzip", "2.4.0", nil)
	source.Add("rubyzip", "2.4.1", nil)
	source.Add("rubyzip", "3.0.0", nil)

	// Add roo versions with their rubyzip constraints
	source.Add("roo", "2.1.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 3.0.0, < 4.0.0"},
	})
	source.Add("roo", "2.10.1", []Dependency{
		{Name: "rubyzip", Constraint: ">= 1.3.0, < 3.0.0"}, // Compatible with rubyXL!
	})
	source.Add("roo", "3.0.0", []Dependency{
		{Name: "rubyzip", Constraint: ">= 3.0.0, < 4.0.0"},
	})

	// Add rubyXL versions with their rubyzip constraints
	source.Add("rubyXL", "3.4.14", []Dependency{
		{Name: "rubyzip", Constraint: ">= 2.4.0, < 3.0.0"}, // ~> 2.4 in Ruby
	})
	source.Add("rubyXL", "3.4.34", []Dependency{
		{Name: "rubyzip", Constraint: ">= 2.4.0, < 3.0.0"}, // ~> 2.4 in Ruby
	})

	// Root depends on both roo and rubyXL (any version)
	rootSource := NewRootSource()
	rootSource.AddPackage(MakeName("roo"), NewAnyVersionCondition())
	rootSource.AddPackage(MakeName("rubyXL"), NewAnyVersionCondition())

	// Create solver with both root and map sources
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

	// Check we got the expected versions
	if solutionMap["roo"] != "2.10.1" {
		t.Errorf("Expected roo 2.10.1, got %s", solutionMap["roo"])
	}
	if solutionMap["rubyXL"] != "3.4.34" {
		t.Errorf("Expected rubyXL 3.4.34, got %s", solutionMap["rubyXL"])
	}
	if solutionMap["rubyzip"] != "2.4.1" {
		t.Errorf("Expected rubyzip 2.4.1, got %s", solutionMap["rubyzip"])
	}

	// Print solution for debugging
	t.Logf("Solution found:")
	for name, version := range solutionMap {
		t.Logf("  %s = %s", name, version)
	}
}

// NewAnyVersionCondition creates a condition that accepts any version
func NewAnyVersionCondition() Condition {
	return NewVersionSetCondition(FullVersionSet())
}

// MapSource is a simple in-memory source for testing
type MapSource struct {
	packages map[string][]packageVersion
}

type packageVersion struct {
	version string
	deps    []Dependency
}

type Dependency struct {
	Name       string
	Constraint string
}

func NewMapSource() *MapSource {
	return &MapSource{
		packages: make(map[string][]packageVersion),
	}
}

func (m *MapSource) Add(name, version string, deps []Dependency) {
	m.packages[name] = append(m.packages[name], packageVersion{
		version: version,
		deps:    deps,
	})
}

func (m *MapSource) GetVersions(name Name) ([]Version, error) {
	pkgName := name.Value()
	versions := m.packages[pkgName]
	if len(versions) == 0 {
		return nil, &PackageNotFoundError{Package: name}
	}

	result := make([]Version, 0, len(versions))
	for _, pv := range versions {
		result = append(result, SimpleVersion(pv.version))
	}
	return result, nil
}

func (m *MapSource) GetDependencies(name Name, version Version) ([]Term, error) {
	pkgName := name.Value()
	versionStr := version.String()

	versions := m.packages[pkgName]
	for _, pv := range versions {
		if pv.version == versionStr {
			var terms []Term
			for _, dep := range pv.deps {
				// Parse the constraint into a Condition
				condition, err := parseConstraint(dep.Constraint)
				if err != nil {
					return nil, fmt.Errorf("failed to parse constraint %q: %w", dep.Constraint, err)
				}
				terms = append(terms, NewTerm(MakeName(dep.Name), condition))
			}
			return terms, nil
		}
	}

	return nil, &PackageVersionNotFoundError{
		Package: name,
		Version: version,
	}
}

// parseConstraint converts constraint strings like ">= 1.3.0, < 3.0.0" into Conditions
func parseConstraint(constraint string) (Condition, error) {
	if constraint == "" || constraint == "*" {
		return NewAnyVersionCondition(), nil
	}

	// Use ParseVersionRange to handle compound constraints
	versionSet, err := ParseVersionRange(constraint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse constraint %q: %w", constraint, err)
	}

	return NewVersionSetCondition(versionSet), nil
}
