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

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

// VersionIntervalSet implements VersionSet using sorted, disjoint intervals.
// This representation efficiently handles common version constraints like ranges
// and unions.
//
// Intervals are stored in normalized form: sorted, non-empty, non-overlapping,
// and with no adjacent intervals that could be merged. This ensures efficient
// set operations and canonical string representations.
//
// Example:
//
//	set := &VersionIntervalSet{}
//	set1 := ParseVersionRange(">=1.0.0, <2.0.0")
//	set2 := ParseVersionRange(">=1.5.0, <3.0.0")
//	union := set1.Union(set2) // >=1.0.0, <3.0.0
type VersionIntervalSet struct {
	intervals []versionInterval
}

// newVersionIntervalSet creates a new VersionIntervalSet from intervals.
// The intervals are automatically normalized (sorted, merged, deduplicated).
func newVersionIntervalSet(intervals []versionInterval) *VersionIntervalSet {
	normalized := normalizeIntervals(intervals)
	return &VersionIntervalSet{intervals: normalized}
}

// intervalSetFromBounds creates a VersionSet from single lower and upper bounds.
func intervalSetFromBounds(lower, upper versionBound) VersionSet {
	if interval, ok := newInterval(lower, upper); ok {
		return newVersionIntervalSet([]versionInterval{interval})
	}
	return &VersionIntervalSet{}
}

// cloneIntervals creates a copy of the intervals slice for safe mutation.
func (s *VersionIntervalSet) cloneIntervals() []versionInterval {
	if len(s.intervals) == 0 {
		return nil
	}
	cloned := make([]versionInterval, len(s.intervals))
	copy(cloned, s.intervals)
	return cloned
}

// Empty returns a VersionSet containing no versions.
func (s *VersionIntervalSet) Empty() VersionSet {
	return &VersionIntervalSet{}
}

// Full returns a VersionSet containing all possible versions.
func (s *VersionIntervalSet) Full() VersionSet {
	return &VersionIntervalSet{
		intervals: []versionInterval{
			{
				lower: negativeInfinityBound(),
				upper: positiveInfinityBound(),
			},
		},
	}
}

// Singleton returns a VersionSet containing exactly one version.
func (s *VersionIntervalSet) Singleton(version Version) VersionSet {
	if version == nil {
		return &VersionIntervalSet{}
	}
	if interval, ok := newInterval(
		newLowerBound(version, true),
		newUpperBound(version, true),
	); ok {
		return &VersionIntervalSet{intervals: []versionInterval{interval}}
	}
	return &VersionIntervalSet{}
}

// Union returns the set of versions in either this set or the other.
func (s *VersionIntervalSet) Union(other VersionSet) VersionSet {
	o := asIntervalSet(other)
	intervals := s.cloneIntervals()
	intervals = append(intervals, o.intervals...)
	return newVersionIntervalSet(intervals)
}

// Intersection returns the set of versions in both this set and the other.
func (s *VersionIntervalSet) Intersection(other VersionSet) VersionSet {
	o := asIntervalSet(other)
	if len(s.intervals) == 0 || len(o.intervals) == 0 {
		return &VersionIntervalSet{}
	}

	result := make([]versionInterval, 0, len(s.intervals))
	i, j := 0, 0
	for i < len(s.intervals) && j < len(o.intervals) {
		if interval, ok := intersectInterval(s.intervals[i], o.intervals[j]); ok {
			result = append(result, interval)
		}

		if compareUpper(s.intervals[i].upper, o.intervals[j].upper) < 0 {
			i++
		} else {
			j++
		}
	}

	return newVersionIntervalSet(result)
}

// intersectInterval computes the intersection of two intervals.
func intersectInterval(a, b versionInterval) (versionInterval, bool) {
	return newInterval(
		max(a.lower, b.lower, compareLower), // Higher lower bound
		min(a.upper, b.upper, compareUpper), // Lower upper bound
	)
}

// Complement returns the set of versions NOT in this set.
func (s *VersionIntervalSet) Complement() VersionSet {
	if len(s.intervals) == 0 {
		return s.Full()
	}

	gaps := make([]versionInterval, 0, len(s.intervals)+1)
	currentLower := negativeInfinityBound()

	for _, interval := range s.intervals {
		gapUpper := interval.complementUpperBound()
		if gap, ok := newInterval(currentLower, gapUpper); ok {
			gaps = append(gaps, gap)
		}
		currentLower = interval.complementLowerBound()
	}

	if tail, ok := newInterval(currentLower, positiveInfinityBound()); ok {
		gaps = append(gaps, tail)
	}

	return newVersionIntervalSet(gaps)
}

// Contains tests if a specific version is in the set.
func (s *VersionIntervalSet) Contains(version Version) bool {
	for _, interval := range s.intervals {
		if interval.contains(version) {
			return true
		}
	}
	return false
}

// IsEmpty returns true if the set contains no versions.
func (s *VersionIntervalSet) IsEmpty() bool {
	return len(s.intervals) == 0
}

// IsSubset returns true if all versions in this set are also in the other set.
func (s *VersionIntervalSet) IsSubset(other VersionSet) bool {
	if len(s.intervals) == 0 {
		return true
	}

	o := asIntervalSet(other)
	if len(o.intervals) == 0 {
		return false
	}

	i, j := 0, 0
	for i < len(s.intervals) {
		if j >= len(o.intervals) {
			return false
		}

		if o.intervals[j].covers(s.intervals[i]) {
			i++
			continue
		}

		if upperLessThanLower(o.intervals[j].upper, s.intervals[i].lower) {
			j++
			continue
		}

		return false
	}

	return true
}

// IsDisjoint returns true if this set and the other set have no versions in common.
func (s *VersionIntervalSet) IsDisjoint(other VersionSet) bool {
	if len(s.intervals) == 0 {
		return true
	}

	o := asIntervalSet(other)
	if len(o.intervals) == 0 {
		return true
	}

	i, j := 0, 0
	for i < len(s.intervals) && j < len(o.intervals) {
		if s.intervals[i].overlaps(o.intervals[j]) {
			return false
		}

		if compareUpper(s.intervals[i].upper, o.intervals[j].upper) < 0 {
			i++
		} else {
			j++
		}
	}

	return true
}

// Intervals returns an iterator over the internal version intervals.
// This enables using range-over-function syntax:
//
//	for interval := range versionSet.Intervals() {
//	    fmt.Printf("Range: %v to %v\n", interval.lower, interval.upper)
//	}
func (s *VersionIntervalSet) Intervals() iter.Seq[versionInterval] {
	return slices.Values(s.intervals)
}

// String returns a human-readable representation of the set.
// Empty sets display as "∅", full sets as "*", and intervals use standard operators.
func (s *VersionIntervalSet) String() string {
	if len(s.intervals) == 0 {
		return "∅"
	}

	if len(s.intervals) == 1 {
		return intervalToString(s.intervals[0])
	}

	parts := make([]string, len(s.intervals))
	for i, interval := range s.intervals {
		parts[i] = intervalToString(interval)
	}
	return strings.Join(parts, " || ")
}

// intervalToString converts a single interval to its string representation.
func intervalToString(interval versionInterval) string {
	if interval.lower.isNegInfinity() && interval.upper.isPosInfinity() {
		return "*"
	}

	if interval.lower.isFinite() && interval.upper.isFinite() {
		if interval.lower.version.Sort(interval.upper.version) == 0 &&
			interval.lower.inclusive && interval.upper.inclusive {
			return fmt.Sprintf("==%s", interval.lower.version)
		}
	}

	var parts []string

	if interval.lower.isFinite() {
		if interval.lower.inclusive {
			parts = append(parts, fmt.Sprintf(">=%s", interval.lower.version))
		} else {
			parts = append(parts, fmt.Sprintf(">%s", interval.lower.version))
		}
	}

	if interval.upper.isFinite() {
		if interval.upper.inclusive {
			parts = append(parts, fmt.Sprintf("<=%s", interval.upper.version))
		} else {
			parts = append(parts, fmt.Sprintf("<%s", interval.upper.version))
		}
	}

	if len(parts) == 0 {
		return "*"
	}

	return strings.Join(parts, ", ")
}

// asIntervalSet converts a VersionSet to VersionIntervalSet or panics.
// This is used internally for type assertion with a helpful error message.
func asIntervalSet(set VersionSet) *VersionIntervalSet {
	if set == nil {
		return &VersionIntervalSet{}
	}

	if iv, ok := set.(*VersionIntervalSet); ok {
		return iv
	}

	// Fallback: if the set behaves as empty/full, use that knowledge.
	if set.IsEmpty() {
		return &VersionIntervalSet{}
	}

	panic("unsupported VersionSet implementation")
}

// singletonVersionFromSet extracts a single version if the set contains exactly one.
// Returns (version, true) if singleton, (nil, false) otherwise.
func singletonVersionFromSet(set VersionSet) (Version, bool) {
	iv, ok := set.(*VersionIntervalSet)
	if !ok || len(iv.intervals) != 1 {
		return nil, false
	}

	interval := iv.intervals[0]
	if !interval.lower.isFinite() || !interval.upper.isFinite() {
		return nil, false
	}

	if interval.lower.version.Sort(interval.upper.version) != 0 {
		return nil, false
	}

	if !interval.lower.inclusive || !interval.upper.inclusive {
		return nil, false
	}

	return interval.lower.version, true
}

var (
	_ VersionSet = (*VersionIntervalSet)(nil)
)
