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

// VersionSetCondition implements Condition using a VersionSet.
// This enables using complex version constraints (ranges, unions, complements)
// as dependency conditions.
//
// Example:
//
//	// Create a version range condition
//	set, _ := ParseVersionRange(">=1.0.0, <2.0.0")
//	condition := NewVersionSetCondition(set)
//	term := NewTerm("lodash", condition)
type VersionSetCondition struct {
	Set VersionSet
}

// NewVersionSetCondition creates a new VersionSetCondition from a VersionSet.
func NewVersionSetCondition(set VersionSet) *VersionSetCondition {
	return &VersionSetCondition{Set: set}
}

// String returns a human-readable representation of the condition.
func (vsc *VersionSetCondition) String() string {
	if vsc == nil || vsc.Set == nil {
		return "*"
	}
	return vsc.Set.String()
}

// Satisfies returns true if the given version satisfies the condition.
func (vsc *VersionSetCondition) Satisfies(ver Version) bool {
	if vsc == nil || vsc.Set == nil {
		return true
	}
	return vsc.Set.Contains(ver)
}

// ToVersionSet implements VersionSetConverter, enabling CDCL solver support.
func (vsc *VersionSetCondition) ToVersionSet() VersionSet {
	if vsc == nil || vsc.Set == nil {
		return FullVersionSet()
	}
	return vsc.Set
}

// EmptyVersionSet returns a VersionSet that contains no versions.
// Useful for creating impossible constraints or complement operations.
func EmptyVersionSet() VersionSet {
	return &VersionIntervalSet{}
}

// FullVersionSet returns a VersionSet that contains all possible versions.
// Equivalent to "*" or "any version" constraint.
func FullVersionSet() VersionSet {
	return &VersionIntervalSet{
		intervals: []versionInterval{
			{
				lower: negativeInfinityBound(),
				upper: positiveInfinityBound(),
			},
		},
	}
}

// NewVersionRangeSet creates a VersionSet from lower and upper bounds.
// This helper allows custom Version implementations to create intervals
// without relying on ParseVersionRange which uses SemanticVersion.
//
// Example:
//
//	// Create range [1.0.0, 2.0.0)
//	lower, _ := ParseSemanticVersion("1.0.0")
//	upper, _ := ParseSemanticVersion("2.0.0")
//	set := NewVersionRangeSet(lower, true, upper, false)
func NewVersionRangeSet(lower Version, lowerInclusive bool, upper Version, upperInclusive bool) VersionSet {
	return intervalSetFromBounds(
		newLowerBound(lower, lowerInclusive),
		newUpperBound(upper, upperInclusive),
	)
}

// NewLowerBoundVersionSet creates a VersionSet with only a lower bound.
// Examples: ">= 1.0.0" (inclusive=true), "> 1.0.0" (inclusive=false)
//
// Example:
//
//	v, _ := ParseSemanticVersion("1.0.0")
//	set := NewLowerBoundVersionSet(v, true) // >=1.0.0
func NewLowerBoundVersionSet(version Version, inclusive bool) VersionSet {
	return intervalSetFromBounds(
		newLowerBound(version, inclusive),
		positiveInfinityBound(),
	)
}

// NewUpperBoundVersionSet creates a VersionSet with only an upper bound.
// Examples: "<= 2.0.0" (inclusive=true), "< 2.0.0" (inclusive=false)
//
// Example:
//
//	v, _ := ParseSemanticVersion("2.0.0")
//	set := NewUpperBoundVersionSet(v, false) // <2.0.0
func NewUpperBoundVersionSet(version Version, inclusive bool) VersionSet {
	return intervalSetFromBounds(
		negativeInfinityBound(),
		newUpperBound(version, inclusive),
	)
}

var (
	_ Condition           = (*VersionSetCondition)(nil)
	_ VersionSetConverter = (*VersionSetCondition)(nil)
)
