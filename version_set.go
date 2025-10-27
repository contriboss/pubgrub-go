package pubgrub

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

// VersionSet represents a set of versions that can be used in version constraints.
// Implementations must be immutable – all operations return new instances.
type VersionSet interface {
	Empty() VersionSet
	Full() VersionSet
	Singleton(version Version) VersionSet
	Union(other VersionSet) VersionSet
	Intersection(other VersionSet) VersionSet
	Complement() VersionSet
	Contains(version Version) bool
	IsEmpty() bool
	IsSubset(other VersionSet) bool
	IsDisjoint(other VersionSet) bool
	String() string
}

const (
	boundNegativeInfinity = -1
	boundFinite           = 0
	boundPositiveInfinity = 1
)

type versionBound struct {
	version   Version
	inclusive bool
	infinite  int
}

func newLowerBound(version Version, inclusive bool) versionBound {
	if version == nil {
		return versionBound{infinite: boundNegativeInfinity, inclusive: true}
	}
	return versionBound{version: version, inclusive: inclusive}
}

func newUpperBound(version Version, inclusive bool) versionBound {
	if version == nil {
		return versionBound{infinite: boundPositiveInfinity, inclusive: true}
	}
	return versionBound{version: version, inclusive: inclusive}
}

func negativeInfinityBound() versionBound {
	return versionBound{infinite: boundNegativeInfinity, inclusive: true}
}

func positiveInfinityBound() versionBound {
	return versionBound{infinite: boundPositiveInfinity, inclusive: true}
}

func (b versionBound) isNegInfinity() bool {
	return b.infinite == boundNegativeInfinity
}

func (b versionBound) isPosInfinity() bool {
	return b.infinite == boundPositiveInfinity
}

func (b versionBound) isFinite() bool {
	return b.infinite == boundFinite
}

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

type versionInterval struct {
	lower versionBound
	upper versionBound
}

func newInterval(lower, upper versionBound) (versionInterval, bool) {
	interval := versionInterval{lower: lower, upper: upper}
	if interval.isEmpty() {
		return versionInterval{}, false
	}
	return interval, true
}

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

func (iv versionInterval) overlaps(other versionInterval) bool {
	if upperLessThanLower(iv.upper, other.lower) {
		return false
	}
	if upperLessThanLower(other.upper, iv.lower) {
		return false
	}
	return true
}

func (iv versionInterval) touches(other versionInterval) bool {
	return !upperLessThanLower(iv.upper, other.lower) &&
		!upperLessThanLower(other.upper, iv.lower)
}

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

func (iv versionInterval) covers(other versionInterval) bool {
	if compareLower(iv.lower, other.lower) > 0 {
		return false
	}
	if compareUpper(iv.upper, other.upper) < 0 {
		return false
	}
	return true
}

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

type VersionIntervalSet struct {
	intervals []versionInterval
}

func newVersionIntervalSet(intervals []versionInterval) *VersionIntervalSet {
	normalized := normalizeIntervals(intervals)
	return &VersionIntervalSet{intervals: normalized}
}

func intervalSetFromBounds(lower, upper versionBound) VersionSet {
	if interval, ok := newInterval(lower, upper); ok {
		return newVersionIntervalSet([]versionInterval{interval})
	}
	return &VersionIntervalSet{}
}

func (s *VersionIntervalSet) cloneIntervals() []versionInterval {
	if len(s.intervals) == 0 {
		return nil
	}
	cloned := make([]versionInterval, len(s.intervals))
	copy(cloned, s.intervals)
	return cloned
}

func (s *VersionIntervalSet) Empty() VersionSet {
	return &VersionIntervalSet{}
}

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

func (s *VersionIntervalSet) Union(other VersionSet) VersionSet {
	o := asIntervalSet(other)
	intervals := s.cloneIntervals()
	intervals = append(intervals, o.intervals...)
	return newVersionIntervalSet(intervals)
}

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

func intersectInterval(a, b versionInterval) (versionInterval, bool) {
	return newInterval(
		max(a.lower, b.lower, compareLower), // Higher lower bound
		min(a.upper, b.upper, compareUpper), // Lower upper bound
	)
}

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

func (s *VersionIntervalSet) Contains(version Version) bool {
	for _, interval := range s.intervals {
		if interval.contains(version) {
			return true
		}
	}
	return false
}

func (s *VersionIntervalSet) IsEmpty() bool {
	return len(s.intervals) == 0
}

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

// VersionSetCondition implements Condition using a VersionSet.
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

// EmptyVersionSet returns a VersionSet that contains no versions.
func EmptyVersionSet() VersionSet {
	return &VersionIntervalSet{}
}

// FullVersionSet returns a VersionSet that contains all possible versions.
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
func NewVersionRangeSet(lower Version, lowerInclusive bool, upper Version, upperInclusive bool) VersionSet {
	return intervalSetFromBounds(
		newLowerBound(lower, lowerInclusive),
		newUpperBound(upper, upperInclusive),
	)
}

// NewLowerBoundVersionSet creates a VersionSet with only a lower bound.
// Examples: ">= 1.0.0" (inclusive=true), "> 1.0.0" (inclusive=false)
func NewLowerBoundVersionSet(version Version, inclusive bool) VersionSet {
	return intervalSetFromBounds(
		newLowerBound(version, inclusive),
		positiveInfinityBound(),
	)
}

// NewUpperBoundVersionSet creates a VersionSet with only an upper bound.
// Examples: "<= 2.0.0" (inclusive=true), "< 2.0.0" (inclusive=false)
func NewUpperBoundVersionSet(version Version, inclusive bool) VersionSet {
	return intervalSetFromBounds(
		negativeInfinityBound(),
		newUpperBound(version, inclusive),
	)
}

// Ensure interface compliance.
var (
	_ VersionSet = (*VersionIntervalSet)(nil)
	_ Condition  = (*VersionSetCondition)(nil)
)
