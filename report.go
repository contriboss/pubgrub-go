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
	"strings"
)

// Reporter is an interface for formatting incompatibilities into error messages
type Reporter interface {
	// Report generates a human-readable error message from an incompatibility
	Report(incomp *Incompatibility) string
}

// DefaultReporter produces readable error messages with hierarchical structure
type DefaultReporter struct{}

// Report implements Reporter
func (r *DefaultReporter) Report(incomp *Incompatibility) string {
	if incomp == nil {
		return "no solution found"
	}

	var lines []string
	r.reportIncompatibility(incomp, &lines, 0, make(map[*Incompatibility]bool))
	return strings.Join(lines, "\n")
}

func (r *DefaultReporter) reportIncompatibility(incomp *Incompatibility, lines *[]string, depth int, visited map[*Incompatibility]bool) {
	if visited[incomp] {
		return
	}
	visited[incomp] = true

	indent := strings.Repeat("  ", depth)

	switch incomp.Kind {
	case KindNoVersions:
		if len(incomp.Terms) > 0 {
			*lines = append(*lines, fmt.Sprintf("%sNo versions of %s satisfy the constraint", indent, incomp.Terms[0]))
		}

	case KindFromDependency:
		if len(incomp.Terms) == 2 {
			// Terms are {P@v, not D@d}, unnegate the dependency for display
			dep := incomp.Terms[1]
			if !dep.Positive {
				dep = dep.Negate()
			}
			*lines = append(*lines, fmt.Sprintf("%sBecause %s %s depends on %s",
				indent, incomp.Package.Value(), incomp.Version, dep))
		}

	case KindConflict:
		if incomp.Cause1 != nil && incomp.Cause2 != nil {
			*lines = append(*lines, fmt.Sprintf("%sBecause:", indent))
			r.reportIncompatibility(incomp.Cause1, lines, depth+1, visited)
			*lines = append(*lines, fmt.Sprintf("%sand:", indent))
			r.reportIncompatibility(incomp.Cause2, lines, depth+1, visited)

			// Explain the result
			if len(incomp.Terms) == 0 {
				*lines = append(*lines, fmt.Sprintf("%sversion solving has failed.", indent))
			} else if len(incomp.Terms) == 1 {
				*lines = append(*lines, fmt.Sprintf("%s%s is forbidden.", indent, incomp.Terms[0]))
			} else {
				var termStrs []string
				for _, term := range incomp.Terms {
					termStrs = append(termStrs, term.String())
				}
				*lines = append(*lines, fmt.Sprintf("%sthese constraints conflict: %s",
					indent, strings.Join(termStrs, " and ")))
			}
		}

	default:
		*lines = append(*lines, fmt.Sprintf("%s%s", indent, incomp.String()))
	}
}

// CollapsedReporter produces a more compact error format
type CollapsedReporter struct{}

// Report implements Reporter with a collapsed format
func (r *CollapsedReporter) Report(incomp *Incompatibility) string {
	if incomp == nil {
		return "no solution found"
	}

	var lines []string
	r.collectLines(incomp, &lines, make(map[*Incompatibility]bool))

	if len(lines) == 0 {
		return "version solving failed"
	}

	// Join with "And because" for readability
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\nAnd because " + lines[i]
	}
	return result
}

func (r *CollapsedReporter) collectLines(incomp *Incompatibility, lines *[]string, visited map[*Incompatibility]bool) {
	if visited[incomp] {
		return
	}
	visited[incomp] = true

	switch incomp.Kind {
	case KindNoVersions:
		if len(incomp.Terms) > 0 {
			*lines = append(*lines, fmt.Sprintf("no versions of %s satisfy the constraint", incomp.Terms[0]))
		}

	case KindFromDependency:
		if len(incomp.Terms) == 2 {
			// Terms are {P@v, not D@d}, unnegate the dependency for display
			dep := incomp.Terms[1]
			if !dep.Positive {
				dep = dep.Negate()
			}
			*lines = append(*lines, fmt.Sprintf("%s %s depends on %s",
				incomp.Package.Value(), incomp.Version, dep))
		}

	case KindConflict:
		if incomp.Cause1 != nil && incomp.Cause2 != nil {
			// Recursively collect from causes
			r.collectLines(incomp.Cause1, lines, visited)
			r.collectLines(incomp.Cause2, lines, visited)

			// Add conclusion
			if len(incomp.Terms) == 1 {
				*lines = append(*lines, fmt.Sprintf("%s is forbidden", incomp.Terms[0]))
			} else if len(incomp.Terms) > 1 {
				var termStrs []string
				for _, term := range incomp.Terms {
					termStrs = append(termStrs, term.String())
				}
				*lines = append(*lines, fmt.Sprintf("these constraints conflict: %s",
					strings.Join(termStrs, " and ")))
			}
		}

	default:
		*lines = append(*lines, incomp.String())
	}
}
