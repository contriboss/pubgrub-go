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

// IncompatibilityKind represents the type/origin of an incompatibility
type IncompatibilityKind int

const (
	// KindNoVersions means no versions satisfy the constraint
	KindNoVersions IncompatibilityKind = iota
	// KindFromDependency means incompatibility from a package dependency
	KindFromDependency
	// KindConflict means derived from conflict resolution
	KindConflict
)

// Incompatibility represents a set of package requirements that cannot all be satisfied
type Incompatibility struct {
	// Terms that are incompatible
	Terms []Term
	// Kind of incompatibility
	Kind IncompatibilityKind
	// Cause1 and Cause2 are set for derived incompatibilities (Kind == KindConflict)
	Cause1 *Incompatibility
	Cause2 *Incompatibility
	// Package and Version for KindFromDependency
	Package Name
	Version Version
}

// NewIncompatibilityNoVersions creates an incompatibility for when no versions exist
func NewIncompatibilityNoVersions(term Term) *Incompatibility {
	return &Incompatibility{
		Terms: []Term{term},
		Kind:  KindNoVersions,
	}
}

// NewIncompatibilityFromDependency creates an incompatibility from a dependency
// Represents: package@version depends on dependency
// Per PubGrub spec: "foo ^1.0.0 depends on bar ^2.0.0" → {foo ^1.0.0, not bar ^2.0.0}
func NewIncompatibilityFromDependency(pkg Name, ver Version, dependency Term) *Incompatibility {
	base := NewTerm(pkg, EqualsCondition{Version: ver}) // Positive term for the package
	negatedDep := dependency.Negate()                   // Negate the dependency
	terms := []Term{base, negatedDep}
	return &Incompatibility{
		Terms:   terms,
		Kind:    KindFromDependency,
		Package: pkg,
		Version: ver,
	}
}

// NewIncompatibilityConflict creates a derived incompatibility from two causes
func NewIncompatibilityConflict(terms []Term, cause1, cause2 *Incompatibility) *Incompatibility {
	// Deduplicate terms by Name
	seen := make(map[Name]Term)
	deduped := make([]Term, 0, len(terms))
	for _, term := range terms {
		if _, ok := seen[term.Name]; ok {
			continue
		}
		seen[term.Name] = term
		deduped = append(deduped, term)
	}

	return &Incompatibility{
		Terms:  deduped,
		Kind:   KindConflict,
		Cause1: cause1,
		Cause2: cause2,
	}
}

// String returns a string representation of the incompatibility
func (inc *Incompatibility) String() string {
	if len(inc.Terms) == 0 {
		return "version solving failed"
	}

	if len(inc.Terms) == 1 {
		return fmt.Sprintf("%s is forbidden", inc.Terms[0])
	}

	// For dependency incompatibilities, display "Pkg ver depends on dependency"
	if inc.Kind == KindFromDependency && len(inc.Terms) == 2 {
		var dep Term
		for _, term := range inc.Terms {
			if term.Name != inc.Package {
				dep = term
				break
			}
		}
		if dep.Name == EmptyName() {
			dep = inc.Terms[1]
		}
		if !dep.Positive {
			dep = dep.Negate()
		}
		return fmt.Sprintf("%s %s depends on %s", inc.Package.Value(), inc.Version, dep)
	}

	var parts []string
	for _, term := range inc.Terms {
		parts = append(parts, term.String())
	}
	return fmt.Sprintf("%s are incompatible", strings.Join(parts, " and "))
}
