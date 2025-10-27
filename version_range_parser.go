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
	"strings"
)

// ParseVersionRange parses a version range string and returns a VersionSet.
//
// Supported syntax:
//   - Comparison operators: >=, >, <=, <, ==, !=, =
//   - Comma-separated conjunctions (AND): ">=1.0.0, <2.0.0"
//   - Double-pipe disjunctions (OR): ">=1.0.0 || >=2.0.0"
//   - Wildcard "*" for any version
//
// Examples:
//
//	ParseVersionRange(">=1.0.0, <2.0.0")     // [1.0.0, 2.0.0)
//	ParseVersionRange(">=1.0.0 || >=3.0.0")  // >=1.0.0 OR >=3.0.0
//	ParseVersionRange("*")                   // Any version
//	ParseVersionRange("==1.5.0")             // Exactly 1.5.0
//	ParseVersionRange("!=1.5.0")             // Not 1.5.0
//
// The parser tries to interpret versions as SemanticVersion first,
// falling back to SimpleVersion if parsing fails. This allows mixing
// version types within a constraint string.
func ParseVersionRange(s string) (VersionSet, error) {
	s = strings.TrimSpace(s)

	if s == "" || s == "*" {
		return (&VersionIntervalSet{}).Full(), nil
	}

	// Split by OR operator (||)
	orParts := strings.Split(s, "||")
	result := (&VersionIntervalSet{}).Empty()

	for _, orPart := range orParts {
		orPart = strings.TrimSpace(orPart)
		if orPart == "" {
			return nil, fmt.Errorf("invalid empty range in %q", s)
		}

		// Start with full set for this OR branch
		current := (&VersionIntervalSet{}).Full()

		// Split by AND operator (,)
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

// parseRangeExpression parses a single range expression like ">=1.0.0" or "!=2.0.0"
func parseRangeExpression(expr string) (VersionSet, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty range expression")
	}

	// Helper to parse version string (try SemanticVersion first, fallback to SimpleVersion)
	parseVersion := func(raw string) (Version, error) {
		if raw == "" {
			return nil, fmt.Errorf("missing version in range expression")
		}

		if sv, err := ParseSemanticVersion(raw); err == nil {
			return sv, nil
		}

		return SimpleVersion(raw), nil
	}

	// Define operators and their VersionSet builders
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

	// Try each operator in order
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

	// No operator found, treat as exact version match
	version, err := parseVersion(expr)
	if err != nil {
		return nil, err
	}
	return intervalSetFromBounds(newLowerBound(version, true), newUpperBound(version, true)), nil
}
