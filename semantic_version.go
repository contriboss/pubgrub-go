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
	"strconv"
	"strings"
)

// SemanticVersion represents a semantic version (major.minor.patch[-prerelease][+build])
type SemanticVersion struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

// ParseSemanticVersion parses a semantic version string
// Supports formats like: "1.2.3", "1.2.3-alpha", "1.2.3-alpha.1", "1.2.3+build", "1.2.3-alpha+build"
func ParseSemanticVersion(s string) (*SemanticVersion, error) {
	sv := &SemanticVersion{}

	// Split by '+' to separate build metadata
	parts := strings.SplitN(s, "+", 2)
	if len(parts) == 2 {
		sv.Build = parts[1]
	}
	versionPart := parts[0]

	// Split by '-' to separate prerelease
	parts = strings.SplitN(versionPart, "-", 2)
	if len(parts) == 2 {
		sv.Prerelease = parts[1]
	}
	corePart := parts[0]

	// Parse major.minor.patch
	versionParts := strings.Split(corePart, ".")
	if len(versionParts) < 1 || len(versionParts) > 3 {
		return nil, fmt.Errorf("invalid version format: %s", s)
	}

	var err error
	sv.Major, err = strconv.Atoi(versionParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", versionParts[0])
	}

	if len(versionParts) > 1 {
		sv.Minor, err = strconv.Atoi(versionParts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid minor version: %s", versionParts[1])
		}
	}

	if len(versionParts) > 2 {
		sv.Patch, err = strconv.Atoi(versionParts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid patch version: %s", versionParts[2])
		}
	}

	return sv, nil
}

// String returns the string representation of the semantic version
func (sv *SemanticVersion) String() string {
	s := fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch)

	if sv.Prerelease != "" {
		s += "-" + sv.Prerelease
	}

	if sv.Build != "" {
		s += "+" + sv.Build
	}

	return s
}

// Sort implements Version.Sort
// Returns:
//
//	-1 if sv < other
//	 0 if sv == other
//	 1 if sv > other
//
// Comparison follows semantic versioning rules:
// 1. Compare major, minor, patch numerically
// 2. Pre-release versions have lower precedence than normal versions
// 3. Build metadata is ignored for comparison
func (sv *SemanticVersion) Sort(other Version) int {
	otherSV, ok := other.(*SemanticVersion)
	if !ok {
		// Fallback to string comparison if types don't match
		return strings.Compare(sv.String(), other.String())
	}

	// Compare major version
	if sv.Major != otherSV.Major {
		if sv.Major < otherSV.Major {
			return -1
		}
		return 1
	}

	// Compare minor version
	if sv.Minor != otherSV.Minor {
		if sv.Minor < otherSV.Minor {
			return -1
		}
		return 1
	}

	// Compare patch version
	if sv.Patch != otherSV.Patch {
		if sv.Patch < otherSV.Patch {
			return -1
		}
		return 1
	}

	// Compare prerelease versions
	// Per semver spec: a prerelease version has lower precedence than a normal version
	if sv.Prerelease == "" && otherSV.Prerelease == "" {
		return 0 // Both are release versions and equal
	}

	if sv.Prerelease == "" && otherSV.Prerelease != "" {
		return 1 // Release version is higher than prerelease
	}

	if sv.Prerelease != "" && otherSV.Prerelease == "" {
		return -1 // Prerelease is lower than release version
	}

	// Both have prerelease, compare lexically
	// This is a simplified comparison; full semver has more complex rules
	return comparePrereleaseIdentifiers(sv.Prerelease, otherSV.Prerelease)
}

// comparePrereleaseIdentifiers compares prerelease identifiers according to semver rules
func comparePrereleaseIdentifiers(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	minLen := len(aParts)
	if len(bParts) < minLen {
		minLen = len(bParts)
	}

	for i := 0; i < minLen; i++ {
		aPart := aParts[i]
		bPart := bParts[i]

		// Try to parse as integers
		aInt, aErr := strconv.Atoi(aPart)
		bInt, bErr := strconv.Atoi(bPart)

		if aErr == nil && bErr == nil {
			// Both are numeric
			if aInt != bInt {
				if aInt < bInt {
					return -1
				}
				return 1
			}
		} else if aErr == nil {
			// a is numeric, b is not - numeric has lower precedence
			return -1
		} else if bErr == nil {
			// b is numeric, a is not - numeric has lower precedence
			return 1
		} else {
			// Both are alphanumeric, compare lexically
			cmp := strings.Compare(aPart, bPart)
			if cmp != 0 {
				return cmp
			}
		}
	}

	// All compared parts are equal, shorter version has lower precedence
	if len(aParts) < len(bParts) {
		return -1
	} else if len(aParts) > len(bParts) {
		return 1
	}

	return 0
}

// NewSemanticVersion creates a new SemanticVersion with the given major, minor, and patch versions
func NewSemanticVersion(major, minor, patch int) *SemanticVersion {
	return &SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

// NewSemanticVersionWithPrerelease creates a new SemanticVersion with prerelease info
func NewSemanticVersionWithPrerelease(major, minor, patch int, prerelease string) *SemanticVersion {
	return &SemanticVersion{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}
}

// Verify interface compliance
var (
	_ Version = (*SemanticVersion)(nil)
)

// ParseVersionRange parses a version range string and returns a VersionSet.
// The syntax supports:
//   - Comparison operators: >=, >, <=, <, ==, !=
//   - Comma separated conjunctions (AND): ">=1.0.0, <2.0.0"
//   - Double-pipe disjunctions (OR): ">=1.0.0 || >=2.0.0"
//   - Wildcard "*" for any version
func ParseVersionRange(s string) (VersionSet, error) {
	s = strings.TrimSpace(s)

	if s == "" || s == "*" {
		return (&VersionIntervalSet{}).Full(), nil
	}

	orParts := strings.Split(s, "||")
	result := (&VersionIntervalSet{}).Empty()

	for _, orPart := range orParts {
		orPart = strings.TrimSpace(orPart)
		if orPart == "" {
			return nil, fmt.Errorf("invalid empty range in %q", s)
		}

		current := (&VersionIntervalSet{}).Full()
		andParts := strings.Split(orPart, ",")

		for _, andPart := range andParts {
			token := strings.TrimSpace(andPart)
			if token == "" {
				return nil, fmt.Errorf("invalid empty constraint in %q", orPart)
			}

			set, err := parseRangeExpression(token)
			if err != nil {
				return nil, err
			}

			current = current.Intersection(set)
			if current.IsEmpty() {
				break
			}
		}

		result = result.Union(current)
	}

	return result, nil
}

func parseRangeExpression(expr string) (VersionSet, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty range expression")
	}

	parseVersion := func(raw string) (Version, error) {
		if raw == "" {
			return nil, fmt.Errorf("missing version in range expression")
		}

		if sv, err := ParseSemanticVersion(raw); err == nil {
			return sv, nil
		}

		return SimpleVersion(raw), nil
	}

	operators := []struct {
		prefix  string
		builder func(Version) VersionSet
	}{
		{
			prefix: ">=",
			builder: func(v Version) VersionSet {
				return intervalSetFromBounds(newLowerBound(v, true), positiveInfinityBound())
			},
		},
		{
			prefix: ">",
			builder: func(v Version) VersionSet {
				return intervalSetFromBounds(newLowerBound(v, false), positiveInfinityBound())
			},
		},
		{
			prefix: "<=",
			builder: func(v Version) VersionSet {
				return intervalSetFromBounds(negativeInfinityBound(), newUpperBound(v, true))
			},
		},
		{
			prefix: "<",
			builder: func(v Version) VersionSet {
				return intervalSetFromBounds(negativeInfinityBound(), newUpperBound(v, false))
			},
		},
		{
			prefix: "==",
			builder: func(v Version) VersionSet {
				return intervalSetFromBounds(newLowerBound(v, true), newUpperBound(v, true))
			},
		},
		{
			prefix: "!=",
			builder: func(v Version) VersionSet {
				eq := intervalSetFromBounds(newLowerBound(v, true), newUpperBound(v, true))
				return eq.Complement()
			},
		},
		{
			prefix: "=",
			builder: func(v Version) VersionSet {
				return intervalSetFromBounds(newLowerBound(v, true), newUpperBound(v, true))
			},
		},
	}

	for _, op := range operators {
		if strings.HasPrefix(expr, op.prefix) {
			versionStr := strings.TrimSpace(expr[len(op.prefix):])
			version, err := parseVersion(versionStr)
			if err != nil {
				return nil, err
			}
			return op.builder(version), nil
		}
	}

	version, err := parseVersion(expr)
	if err != nil {
		return nil, err
	}
	return intervalSetFromBounds(newLowerBound(version, true), newUpperBound(version, true)), nil
}
