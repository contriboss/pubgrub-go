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

// VersionSet represents a set of versions that can be used in version constraints.
// Implementations must be immutable – all operations return new instances.
//
// VersionSet enables algebraic operations on version constraints, supporting:
//   - Union: combining multiple version ranges
//   - Intersection: finding common versions between constraints
//   - Complement: inverting version constraints
//   - Subset/Disjoint testing: analyzing constraint relationships
//
// The primary implementation is VersionIntervalSet, which efficiently represents
// version ranges as sorted, non-overlapping intervals.
//
// Example usage:
//
//	// Parse a version range from string
//	set1, _ := ParseVersionRange(">=1.0.0, <2.0.0")
//	set2, _ := ParseVersionRange(">=1.5.0, <3.0.0")
//
//	// Combine constraints
//	union := set1.Union(set2)        // >=1.0.0, <3.0.0
//	intersection := set1.Intersection(set2) // >=1.5.0, <2.0.0
//	complement := set1.Complement()   // <1.0.0 || >=2.0.0
type VersionSet interface {
	// Empty returns a VersionSet containing no versions.
	Empty() VersionSet

	// Full returns a VersionSet containing all possible versions.
	Full() VersionSet

	// Singleton returns a VersionSet containing exactly one version.
	Singleton(version Version) VersionSet

	// Union returns the set of versions in either this set or the other.
	Union(other VersionSet) VersionSet

	// Intersection returns the set of versions in both this set and the other.
	Intersection(other VersionSet) VersionSet

	// Complement returns the set of versions NOT in this set.
	Complement() VersionSet

	// Contains tests if a specific version is in the set.
	Contains(version Version) bool

	// IsEmpty returns true if the set contains no versions.
	IsEmpty() bool

	// IsSubset returns true if all versions in this set are also in the other set.
	IsSubset(other VersionSet) bool

	// IsDisjoint returns true if this set and the other set have no versions in common.
	IsDisjoint(other VersionSet) bool

	// String returns a human-readable representation of the set.
	String() string
}
