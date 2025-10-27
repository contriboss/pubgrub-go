package pubgrub_test

import (
	"fmt"
	"testing"

	"github.com/contriboss/pubgrub-go"
)

// CaretCondition implements a caret (^) version constraint like npm's semver.
// ^1.2.3 means >=1.2.3 <2.0.0 (compatible within same major version).
type CaretCondition struct {
	Base *pubgrub.SemanticVersion
}

func (cc CaretCondition) String() string {
	return fmt.Sprintf("^%s", cc.Base)
}

func (cc CaretCondition) Satisfies(ver pubgrub.Version) bool {
	sv, ok := ver.(*pubgrub.SemanticVersion)
	if !ok {
		return false
	}
	// Must be same major version and >= base
	return sv.Major == cc.Base.Major && sv.Sort(cc.Base) >= 0
}

// ToVersionSet converts the caret constraint to a VersionSet.
// This enables the condition to work with the CDCL solver.
func (cc CaretCondition) ToVersionSet() pubgrub.VersionSet {
	// ^1.2.3 becomes >=1.2.3, <2.0.0
	rangeStr := fmt.Sprintf(">=%d.%d.%d, <%d.0.0",
		cc.Base.Major, cc.Base.Minor, cc.Base.Patch,
		cc.Base.Major+1)
	set, _ := pubgrub.ParseVersionRange(rangeStr)
	return set
}

// Verify CaretCondition implements both interfaces
var (
	_ pubgrub.Condition           = CaretCondition{}
	_ pubgrub.VersionSetConverter = CaretCondition{}
)

func TestCustomCondition_CaretWithSolver(t *testing.T) {
	// Create versions
	v1_0_0, _ := pubgrub.ParseSemanticVersion("1.0.0")
	v1_2_0, _ := pubgrub.ParseSemanticVersion("1.2.0")
	v1_2_3, _ := pubgrub.ParseSemanticVersion("1.2.3")
	v1_5_0, _ := pubgrub.ParseSemanticVersion("1.5.0")
	v2_0_0, _ := pubgrub.ParseSemanticVersion("2.0.0")

	// Setup source with multiple versions
	source := &pubgrub.InMemorySource{}
	source.AddPackage(pubgrub.MakeName("lib"), v1_0_0, nil)
	source.AddPackage(pubgrub.MakeName("lib"), v1_2_0, nil)
	source.AddPackage(pubgrub.MakeName("lib"), v1_2_3, nil)
	source.AddPackage(pubgrub.MakeName("lib"), v1_5_0, nil)
	source.AddPackage(pubgrub.MakeName("lib"), v2_0_0, nil)

	// Use custom CaretCondition: ^1.2.3 means >=1.2.3 <2.0.0
	caretCond := CaretCondition{Base: v1_2_3}

	root := pubgrub.NewRootSource()
	root.AddPackage(pubgrub.MakeName("lib"), caretCond)

	// Solve with custom condition
	solver := pubgrub.NewSolver(root, source)
	solution, err := solver.Solve(root.Term())

	if err != nil {
		t.Fatalf("solver failed: %v", err)
	}

	// Should pick highest version satisfying ^1.2.3 (which is 1.5.0)
	libVer, ok := solution.GetVersion(pubgrub.MakeName("lib"))
	if !ok {
		t.Fatal("lib not found in solution")
	}

	if libVer.String() != "1.5.0" {
		t.Errorf("expected solver to pick 1.5.0, got %s", libVer)
	}
}

func TestCustomCondition_CaretSatisfies(t *testing.T) {
	base, _ := pubgrub.ParseSemanticVersion("1.2.3")
	caret := CaretCondition{Base: base}

	tests := []struct {
		version string
		want    bool
	}{
		{"1.2.3", true},  // exact match
		{"1.2.4", true},  // patch bump
		{"1.3.0", true},  // minor bump
		{"1.5.0", true},  // higher minor
		{"2.0.0", false}, // major bump - not satisfied
		{"1.2.2", false}, // lower patch
		{"1.1.0", false}, // lower minor
		{"0.9.9", false}, // lower major
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			ver, err := pubgrub.ParseSemanticVersion(tt.version)
			if err != nil {
				t.Fatalf("failed to parse version: %v", err)
			}

			got := caret.Satisfies(ver)
			if got != tt.want {
				t.Errorf("Satisfies(%s) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestCustomCondition_ToVersionSet(t *testing.T) {
	base, _ := pubgrub.ParseSemanticVersion("1.2.3")
	caret := CaretCondition{Base: base}

	set := caret.ToVersionSet()

	// Should contain versions in [1.2.3, 2.0.0)
	testCases := []struct {
		version string
		want    bool
	}{
		{"1.2.3", true},
		{"1.5.0", true},
		{"1.9.9", true},
		{"2.0.0", false},
		{"1.2.2", false},
		{"0.9.9", false},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			ver, _ := pubgrub.ParseSemanticVersion(tc.version)
			got := set.Contains(ver)
			if got != tc.want {
				t.Errorf("set.Contains(%s) = %v, want %v", tc.version, got, tc.want)
			}
		})
	}
}

// Example showing how to use a custom condition with the solver
func ExampleVersionSetConverter() {
	// Define a custom caret condition ^1.2.0
	base, _ := pubgrub.ParseSemanticVersion("1.2.0")
	caretCondition := CaretCondition{Base: base}

	// Create a source with several versions
	source := &pubgrub.InMemorySource{}
	v1_2_0, _ := pubgrub.ParseSemanticVersion("1.2.0")
	v1_3_0, _ := pubgrub.ParseSemanticVersion("1.3.0")
	v2_0_0, _ := pubgrub.ParseSemanticVersion("2.0.0")

	source.AddPackage(pubgrub.MakeName("mylib"), v1_2_0, nil)
	source.AddPackage(pubgrub.MakeName("mylib"), v1_3_0, nil)
	source.AddPackage(pubgrub.MakeName("mylib"), v2_0_0, nil)

	// Use the custom condition
	root := pubgrub.NewRootSource()
	root.AddPackage(pubgrub.MakeName("mylib"), caretCondition)

	// Solve
	solver := pubgrub.NewSolver(root, source)
	solution, _ := solver.Solve(root.Term())

	// The solver picks the highest compatible version (1.3.0, not 2.0.0)
	for _, nv := range solution {
		if nv.Name.Value() == "mylib" {
			fmt.Printf("Selected version: %s\n", nv.Version)
		}
	}
	// Output: Selected version: 1.3.0
}
