// Copyright 2024 The University of Queensland
// Copyright 2025 Contriboss
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pubgrub

// Version represents a package version in the dependency resolution system.
// Implementations must provide string representation and comparison.
//
// The PubGrub algorithm is version-type agnostic - any type can be used as
// long as it implements this interface. Built-in implementations include:
//   - SimpleVersion: Lexicographic string comparison
//   - SemanticVersion: Full semver with major.minor.patch ordering
//
// Example custom version:
//
//	type DateVersion time.Time
//
//	func (dv DateVersion) String() string {
//	    return time.Time(dv).Format("2006-01-02")
//	}
//
//	func (dv DateVersion) Sort(other Version) int {
//	    otherDate, ok := other.(DateVersion)
//	    if !ok {
//	        return strings.Compare(dv.String(), other.String())
//	    }
//	    return time.Time(dv).Compare(time.Time(otherDate))
//	}
type Version interface {
	// String returns a human-readable representation of the version.
	String() string

	// Sort compares this version to another.
	// Returns:
	//   - negative if this version < other
	//   - zero if this version == other
	//   - positive if this version > other
	Sort(other Version) int
}

// Condition represents a constraint on package versions.
// Basic conditions like equality are built-in, but custom conditions
// can be implemented by satisfying this interface.
//
// Built-in implementations:
//   - EqualsCondition: Exact version match
//   - VersionSetCondition: Version range constraints
//
// Example custom condition:
//
//	type MinVersionCondition struct {
//	    MinVersion Version
//	}
//
//	func (mvc MinVersionCondition) String() string {
//	    return fmt.Sprintf(">=%s", mvc.MinVersion)
//	}
//
//	func (mvc MinVersionCondition) Satisfies(ver Version) bool {
//	    return ver.Sort(mvc.MinVersion) >= 0
//	}
type Condition interface {
	// String returns a human-readable representation of the condition.
	String() string

	// Satisfies returns true if the given version meets the condition.
	Satisfies(ver Version) bool
}

// VersionSetConverter is an optional interface that Condition implementations
// can provide to enable conversion to VersionSet for use with the CDCL solver.
//
// The CDCL solver needs to perform set operations (intersection, union, complement)
// on version constraints. Conditions that implement this interface can participate
// in these operations, enabling them to work with unit propagation and conflict
// resolution.
//
// Built-in conditions (EqualsCondition, VersionSetCondition) are already handled
// by the solver. Custom condition types should implement this interface to enable
// solver support.
//
// Example custom condition:
//
//	type SemverCaretCondition struct {
//	    Base *SemanticVersion
//	}
//
//	func (sc SemverCaretCondition) String() string {
//	    return fmt.Sprintf("^%s", sc.Base)
//	}
//
//	func (sc SemverCaretCondition) Satisfies(ver Version) bool {
//	    sv, ok := ver.(*SemanticVersion)
//	    if !ok {
//	        return false
//	    }
//	    return sv.Major == sc.Base.Major &&
//	           sv.Sort(sc.Base) >= 0 &&
//	           sv.Major == sc.Base.Major
//	}
//
//	func (sc SemverCaretCondition) ToVersionSet() VersionSet {
//	    // Convert ^1.2.3 to >=1.2.3 <2.0.0
//	    upper := &SemanticVersion{Major: sc.Base.Major + 1}
//	    return NewVersionRangeSet(sc.Base, true, upper, false)
//	}
type VersionSetConverter interface {
	// ToVersionSet converts the condition to a VersionSet for algebraic operations.
	ToVersionSet() VersionSet
}

// Source provides access to package versions and their dependencies.
// Implementations can fetch from in-memory stores, network registries,
// file systems, or any other package source.
//
// Built-in implementations:
//   - InMemorySource: Simple in-memory storage for testing
//   - CombinedSource: Aggregates multiple sources
//   - RootSource: Special source for initial requirements
//   - CachedSource: Wraps a source with caching for performance
//
// Example custom source:
//
//	type RegistrySource struct {
//	    BaseURL string
//	    Client  *http.Client
//	}
//
//	func (rs *RegistrySource) GetVersions(name Name) ([]Version, error) {
//	    resp, err := rs.Client.Get(rs.BaseURL + "/packages/" + name.Value() + "/versions")
//	    // ... parse response ...
//	}
//
//	func (rs *RegistrySource) GetDependencies(name Name, version Version) ([]Term, error) {
//	    resp, err := rs.Client.Get(rs.BaseURL + "/packages/" + name.Value() + "/" + version.String())
//	    // ... parse response ...
//	}
type Source interface {
	// GetVersions returns all versions of a package in sorted order.
	// Versions should be sorted from lowest to highest, as the solver
	// selects from the highest available version.
	GetVersions(name Name) ([]Version, error)

	// GetDependencies returns the dependency terms for a specific package version.
	GetDependencies(name Name, version Version) ([]Term, error)
}
