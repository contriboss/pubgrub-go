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

// versionBound represents either a lower or upper bound of a version interval.
// Bounds can be finite (with a specific version), or infinite (unbounded).
//
// The `infinite` field uses sentinel values:
//   - boundNegativeInfinity (-1): represents -∞ (no lower limit)
//   - boundFinite (0): represents a specific version
//   - boundPositiveInfinity (1): represents +∞ (no upper limit)
//
// The `inclusive` field determines whether the bound includes the version itself.
// For example, ">=1.0.0" has inclusive=true, while ">1.0.0" has inclusive=false.
type versionBound struct {
	version   Version
	inclusive bool
	infinite  int
}

const (
	boundNegativeInfinity = -1
	boundFinite           = 0
	boundPositiveInfinity = 1
)

// newLowerBound creates a lower bound from a version.
// A nil version creates an unbounded lower limit (-∞).
func newLowerBound(version Version, inclusive bool) versionBound {
	if version == nil {
		return versionBound{infinite: boundNegativeInfinity, inclusive: true}
	}
	return versionBound{version: version, inclusive: inclusive}
}

// newUpperBound creates an upper bound from a version.
// A nil version creates an unbounded upper limit (+∞).
func newUpperBound(version Version, inclusive bool) versionBound {
	if version == nil {
		return versionBound{infinite: boundPositiveInfinity, inclusive: true}
	}
	return versionBound{version: version, inclusive: inclusive}
}

// negativeInfinityBound returns a bound representing -∞.
func negativeInfinityBound() versionBound {
	return versionBound{infinite: boundNegativeInfinity, inclusive: true}
}

// positiveInfinityBound returns a bound representing +∞.
func positiveInfinityBound() versionBound {
	return versionBound{infinite: boundPositiveInfinity, inclusive: true}
}

// isNegInfinity returns true if this bound represents -∞.
func (b versionBound) isNegInfinity() bool {
	return b.infinite == boundNegativeInfinity
}

// isPosInfinity returns true if this bound represents +∞.
func (b versionBound) isPosInfinity() bool {
	return b.infinite == boundPositiveInfinity
}

// isFinite returns true if this bound represents a specific version.
func (b versionBound) isFinite() bool {
	return b.infinite == boundFinite
}

// compareLower compares two lower bounds.
// Returns negative if a < b, zero if equal, positive if a > b.
// For lower bounds: inclusive comes before exclusive when versions are equal.
func compareLower(a, b versionBound) int {
	switch {
	case a.infinite == boundNegativeInfinity && b.infinite == boundNegativeInfinity:
		return 0
	case a.infinite == boundNegativeInfinity:
		return -1
	case b.infinite == boundNegativeInfinity:
		return 1
	case a.infinite == boundPositiveInfinity && b.infinite == boundPositiveInfinity:
		return 0
	case a.infinite == boundPositiveInfinity:
		return 1
	case b.infinite == boundPositiveInfinity:
		return -1
	default:
		if cmp := a.version.Sort(b.version); cmp != 0 {
			return cmp
		}
		// For lower bounds: inclusive comes before exclusive
		if a.inclusive == b.inclusive {
			return 0
		}
		if a.inclusive {
			return -1
		}
		return 1
	}
}

// compareUpper compares two upper bounds.
// Returns negative if a < b, zero if equal, positive if a > b.
// For upper bounds: inclusive comes after exclusive when versions are equal.
func compareUpper(a, b versionBound) int {
	switch {
	case a.infinite == boundPositiveInfinity && b.infinite == boundPositiveInfinity:
		return 0
	case a.infinite == boundPositiveInfinity:
		return 1
	case b.infinite == boundPositiveInfinity:
		return -1
	case a.infinite == boundNegativeInfinity && b.infinite == boundNegativeInfinity:
		return 0
	case a.infinite == boundNegativeInfinity:
		return -1
	case b.infinite == boundNegativeInfinity:
		return 1
	default:
		if cmp := a.version.Sort(b.version); cmp != 0 {
			return cmp
		}
		// For upper bounds: inclusive comes after exclusive
		if a.inclusive == b.inclusive {
			return 0
		}
		if a.inclusive {
			return 1
		}
		return -1
	}
}
