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

import "slices"

// versionInterval represents a contiguous range of versions between lower and upper bounds.
// Intervals are half-open or closed depending on the inclusivity of their bounds.
//
// Examples:
//   - [1.0.0, 2.0.0) represents >=1.0.0, <2.0.0
//   - (1.0.0, 2.0.0] represents >1.0.0, <=2.0.0
//   - [1.0.0, ∞) represents >=1.0.0
//
// Intervals form the building blocks of VersionIntervalSet.
type versionInterval struct {
	lower versionBound
	upper versionBound
}

// newInterval creates a version interval from bounds, returning false if the interval is empty.
func newInterval(lower, upper versionBound) (versionInterval, bool) {
	interval := versionInterval{lower: lower, upper: upper}
	if interval.isEmpty() {
		return versionInterval{}, false
	}
	return interval, true
}

// isEmpty returns true if the interval contains no versions.
// This happens when the upper bound is less than the lower bound,
// or when both bounds are the same but at least one is exclusive.
func (iv versionInterval) isEmpty() bool {
	if iv.lower.isPosInfinity() || iv.upper.isNegInfinity() {
		return true
	}

	if iv.lower.isNegInfinity() && iv.upper.isPosInfinity() {
		return false
	}

	if iv.lower.isNegInfinity() {
		return false
	}

	if iv.upper.isPosInfinity() {
		return false
	}

	cmp := iv.lower.version.Sort(iv.upper.version)
	switch {
	case cmp < 0:
		return false
	case cmp > 0:
		return true
	default:
		return !iv.lower.inclusive || !iv.upper.inclusive
	}
}

// contains returns true if the given version falls within this interval.
func (iv versionInterval) contains(version Version) bool {
	if version == nil {
		return false
	}

	if !iv.lower.isNegInfinity() {
		if cmp := version.Sort(iv.lower.version); cmp < 0 {
			return false
		} else if cmp == 0 && !iv.lower.inclusive {
			return false
		}
	}

	if !iv.upper.isPosInfinity() {
		if cmp := version.Sort(iv.upper.version); cmp > 0 {
			return false
		} else if cmp == 0 && !iv.upper.inclusive {
			return false
		}
	}

	return true
}

// upperLessThanLower returns true if upper bound is strictly less than lower bound.
// Used to detect gaps between intervals.
func upperLessThanLower(upper versionBound, lower versionBound) bool {
	switch {
	case upper.isNegInfinity():
		return !lower.isNegInfinity()
	case lower.isPosInfinity():
		return !upper.isPosInfinity()
	case upper.isPosInfinity():
		return false
	case lower.isNegInfinity():
		return false
	}

	cmp := upper.version.Sort(lower.version)
	if cmp < 0 {
		return true
	}
	if cmp > 0 {
		return false
	}
	return !upper.inclusive || !lower.inclusive
}

// overlaps returns true if this interval has any versions in common with other.
func (iv versionInterval) overlaps(other versionInterval) bool {
	if upperLessThanLower(iv.upper, other.lower) {
		return false
	}
	if upperLessThanLower(other.upper, iv.lower) {
		return false
	}
	return true
}

// touches returns true if this interval overlaps or is adjacent to other.
// Adjacent intervals can be merged without creating a gap.
func (iv versionInterval) touches(other versionInterval) bool {
	return !upperLessThanLower(iv.upper, other.lower) &&
		!upperLessThanLower(other.upper, iv.lower)
}

// merge combines two intervals into a single interval spanning both.
func (iv versionInterval) merge(other versionInterval) versionInterval {
	return versionInterval{
		lower: min(iv.lower, other.lower, compareLower),
		upper: max(iv.upper, other.upper, compareUpper),
	}
}

// min returns the minimum of two values using a comparison function
func min[T any](a, b T, compare func(T, T) int) T {
	if compare(a, b) <= 0 {
		return a
	}
	return b
}

// max returns the maximum of two values using a comparison function
func max[T any](a, b T, compare func(T, T) int) T {
	if compare(a, b) >= 0 {
		return a
	}
	return b
}

// covers returns true if this interval completely contains other.
func (iv versionInterval) covers(other versionInterval) bool {
	if compareLower(iv.lower, other.lower) > 0 {
		return false
	}
	if compareUpper(iv.upper, other.upper) < 0 {
		return false
	}
	return true
}

// complementLowerBound returns the lower bound for the complement interval above this interval.
func (iv versionInterval) complementLowerBound() versionBound {
	switch iv.upper.infinite {
	case boundPositiveInfinity:
		return positiveInfinityBound()
	case boundNegativeInfinity:
		return negativeInfinityBound()
	default:
		return versionBound{
			version:   iv.upper.version,
			inclusive: !iv.upper.inclusive,
			infinite:  boundFinite,
		}
	}
}

// complementUpperBound returns the upper bound for the complement interval below this interval.
func (iv versionInterval) complementUpperBound() versionBound {
	switch iv.lower.infinite {
	case boundNegativeInfinity:
		return negativeInfinityBound()
	case boundPositiveInfinity:
		return positiveInfinityBound()
	default:
		return versionBound{
			version:   iv.lower.version,
			inclusive: !iv.lower.inclusive,
			infinite:  boundFinite,
		}
	}
}

// normalizeIntervals canonicalizes a slice of intervals by:
//  1. Removing empty intervals
//  2. Sorting by lower bound
//  3. Merging overlapping or adjacent intervals
//
// This ensures intervals are disjoint and sorted, enabling efficient set operations.
func normalizeIntervals(intervals []versionInterval) []versionInterval {
	filtered := intervals[:0]
	for _, iv := range intervals {
		if !iv.isEmpty() {
			filtered = append(filtered, iv)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	// Sort by lower bound.
	slices.SortFunc(filtered, func(a, b versionInterval) int {
		return compareLower(a.lower, b.lower)
	})

	merged := filtered[:1]
	for i := 1; i < len(filtered); i++ {
		last := &merged[len(merged)-1]
		current := filtered[i]
		if last.touches(current) {
			*last = last.merge(current)
		} else {
			merged = append(merged, current)
		}
	}

	out := make([]versionInterval, len(merged))
	copy(out, merged)
	return out
}
