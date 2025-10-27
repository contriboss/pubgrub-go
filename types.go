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
type Version interface {
	String() string
	Sort(other Version) int
}

// Condition represents a constraint on package versions.
// Basic conditions like equality are built-in, but custom conditions
// can be implemented by satisfying this interface.
type Condition interface {
	String() string
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
//	    Base SemanticVersion
//	}
//
//	func (sc SemverCaretCondition) String() string {
//	    return fmt.Sprintf("^%s", sc.Base)
//	}
//
//	func (sc SemverCaretCondition) Satisfies(ver Version) bool {
//	    sv, ok := ver.(SemanticVersion)
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
//	    upper := SemanticVersion{Major: sc.Base.Major + 1}
//	    set := &VersionIntervalSet{}
//	    return set.Interval(sc.Base, true, upper, false)
//	}
type VersionSetConverter interface {
	ToVersionSet() VersionSet
}

// Source provides access to package versions and their dependencies.
// Implementations can fetch from in-memory stores, network registries,
// file systems, or any other package source.
type Source interface {
	// GetVersions returns all versions of a package in a sorted order.
	GetVersions(name Name) ([]Version, error)

	// GetDependencies returns the dependency terms for a specific package version.
	GetDependencies(name Name, version Version) ([]Term, error)
}
