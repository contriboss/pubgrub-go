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
	"errors"
	"fmt"
	"strings"
)

// partialSolution maintains the evolving solution during dependency resolution.
// It tracks assignments (decisions and derivations) organized by package name
// and decision level, supporting efficient backtracking and version set queries.
//
// The partial solution grows as the solver:
//  1. Makes decisions (selects package versions)
//  2. Propagates constraints (derives new constraints via unit propagation)
//  3. Backtracks (removes assignments when conflicts occur)
//
// Assignments are indexed both globally (for satisfier ordering) and per-package
// (for fast version set computation).
type partialSolution struct {
	assignments []*assignment          // All assignments in chronological order
	perPackage  map[Name][]*assignment // Assignments indexed by package name
	decisionLvl int                    // Current decision level
	nextIndex   int                    // Next assignment index
	root        Name                   // Root package name
}

// newPartialSolution creates a new empty partial solution for the given root package.
func newPartialSolution(root Name) *partialSolution {
	return &partialSolution{
		assignments: make([]*assignment, 0),
		perPackage:  make(map[Name][]*assignment),
		decisionLvl: 0,
		nextIndex:   0,
		root:        root,
	}
}

// newDecisionAssignment creates a new decision assignment for a package version.
func (ps *partialSolution) newDecisionAssignment(name Name, version Version, level int) *assignment {
	return &assignment{
		name:          name,
		term:          NewTerm(name, EqualsCondition{Version: version}),
		kind:          assignmentDecision,
		allowed:       (&VersionIntervalSet{}).Singleton(version),
		version:       version,
		decisionLevel: level,
		index:         ps.nextIndex,
	}
}

// append adds an assignment to the partial solution.
func (ps *partialSolution) append(assign *assignment) {
	ps.assignments = append(ps.assignments, assign)
	stack := ps.perPackage[assign.name]
	stack = append(stack, assign)
	ps.perPackage[assign.name] = stack
	ps.nextIndex++
}

// latest returns the most recent assignment for a package, or nil if none exists.
func (ps *partialSolution) latest(name Name) *assignment {
	stack := ps.perPackage[name]
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1]
}

// allowedSet computes the currently allowed version set for a package by
// intersecting all positive constraints and excluding forbidden sets.
func (ps *partialSolution) allowedSet(name Name) VersionSet {
	stack := ps.perPackage[name]
	full := FullVersionSet()
	if len(stack) == 0 {
		return full
	}

	current := full
	for _, assign := range stack {
		if assign.term.Positive {
			if assign.allowed != nil {
				current = current.Intersection(assign.allowed)
			}
		} else if assign.forbidden != nil {
			current = current.Intersection(assign.forbidden.Complement())
		}
	}
	return current
}

// hasAssignments returns true if there are any assignments for the package.
func (ps *partialSolution) hasAssignments(name Name) bool {
	return len(ps.perPackage[name]) > 0
}

// addDecision adds a version selection decision, incrementing the decision level.
func (ps *partialSolution) addDecision(name Name, version Version) *assignment {
	ps.decisionLvl++
	assign := ps.newDecisionAssignment(name, version, ps.decisionLvl)
	ps.append(assign)
	return assign
}

// seedRoot initializes the partial solution with the root package at decision level 0.
func (ps *partialSolution) seedRoot(name Name, version Version) *assignment {
	assign := ps.newDecisionAssignment(name, version, 0)
	ps.append(assign)
	return assign
}

var errNoAllowedVersions = errors.New("no versions satisfy constraints")

// addDerivation adds a constraint derived from unit propagation.
// Returns (assignment, changed, error) where changed indicates if the allowed set was tightened.
func (ps *partialSolution) addDerivation(term Term, cause *Incompatibility) (*assignment, bool, error) {
	currentAllowed := ps.allowedSet(term.Name)
	newAllowed, err := applyTermToAllowed(currentAllowed, term)
	if err != nil {
		return nil, false, err
	}
	if newAllowed.IsEmpty() {
		return nil, false, errNoAllowedVersions
	}

	assign := &assignment{
		name:          term.Name,
		term:          term,
		kind:          assignmentDerivation,
		cause:         cause,
		decisionLevel: ps.decisionLvl,
		index:         ps.nextIndex,
	}

	if term.Positive {
		assign.allowed = newAllowed
	} else {
		forbidden, ok := termForbiddenSet(term)
		if !ok {
			return nil, false, errors.New("unable to compute forbidden set for term")
		}
		assign.forbidden = forbidden
	}

	changed := !setsEqual(currentAllowed, newAllowed)
	ps.append(assign)

	if changed && term.Positive {
		return assign, true, nil
	}

	if changed && !term.Positive {
		// Record tightened allowance as positive assignment
		tightening := &assignment{
			name:          term.Name,
			term:          termFromAllowedSet(term.Name, newAllowed),
			kind:          assignmentDerivation,
			allowed:       newAllowed,
			cause:         cause,
			decisionLevel: ps.decisionLvl,
			index:         ps.nextIndex,
		}
		ps.append(tightening)
		return tightening, true, nil
	}

	return assign, changed, nil
}

// backtrack removes all assignments above the specified decision level.
// Used when the solver needs to undo decisions during conflict resolution.
func (ps *partialSolution) backtrack(level int) {
	if level < 0 {
		level = 0
	}

	for len(ps.assignments) > 0 {
		last := ps.assignments[len(ps.assignments)-1]
		if last.decisionLevel <= level {
			break
		}
		ps.assignments = ps.assignments[:len(ps.assignments)-1]
		stack := ps.perPackage[last.name]
		if len(stack) > 0 {
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				delete(ps.perPackage, last.name)
			} else {
				ps.perPackage[last.name] = stack
			}
		}
	}

	ps.decisionLvl = level
}

// isComplete returns true if every package (except root) has a decision assignment.
func (ps *partialSolution) isComplete() bool {
	for name, stack := range ps.perPackage {
		// Skip root assignment
		if name == ps.root {
			continue
		}

		hasDecision := false
		for _, assign := range stack {
			if assign.kind == assignmentDecision {
				hasDecision = true
				break
			}
		}
		if !hasDecision {
			return false
		}
	}
	return true
}

// nextDecisionCandidate finds the next package that needs a version decision.
// Returns the package name and true if found, or EmptyName and false if none.
func (ps *partialSolution) nextDecisionCandidate() (Name, bool) {
	seen := make(map[Name]bool)

	for _, assign := range ps.assignments {
		name := assign.name
		if name == ps.root {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true

		if !ps.hasDecision(name) {
			return name, true
		}
	}

	return EmptyName(), false
}

// hasDecision returns true if there's a decision assignment for the package.
func (ps *partialSolution) hasDecision(name Name) bool {
	stack := ps.perPackage[name]
	for _, assign := range stack {
		if assign.kind == assignmentDecision {
			return true
		}
	}
	return false
}

// satisfier finds the assignment that most recently satisfied a term in the incompatibility.
// Used during conflict resolution to identify which assignment to analyze.
func (ps *partialSolution) satisfier(inc *Incompatibility) *assignment {
	var selected *assignment
	maxIndex := -1

	for _, term := range inc.Terms {
		stack := ps.perPackage[term.Name]
		for i := len(stack) - 1; i >= 0; i-- {
			assign := stack[i]
			if termSatisfiedBy(term, assign) {
				if assign.index > maxIndex {
					selected = assign
					maxIndex = assign.index
				}
				break
			}
		}
	}

	return selected
}

// previousDecisionLevel finds the highest decision level among assignments
// satisfying the incompatibility, excluding the satisfier itself.
// Used to determine where to backtrack during conflict resolution.
func (ps *partialSolution) previousDecisionLevel(inc *Incompatibility, satisfier *assignment) int {
	level := 0

	for _, term := range inc.Terms {
		stack := ps.perPackage[term.Name]
		for i := len(stack) - 1; i >= 0; i-- {
			assign := stack[i]
			if assign == satisfier {
				continue
			}
			if termSatisfiedBy(term, assign) && assign.decisionLevel > level {
				level = assign.decisionLevel
			}
		}
	}

	return level
}

// buildSolution constructs the final solution from decision assignments.
// Returns a slice of package-version pairs representing the resolved dependencies.
func (ps *partialSolution) buildSolution() Solution {
	result := make([]NameVersion, 0)
	seen := make(map[Name]bool)

	for _, assign := range ps.assignments {
		if assign.kind != assignmentDecision {
			continue
		}
		if seen[assign.name] {
			continue
		}
		seen[assign.name] = true
		result = append(result, NameVersion{Name: assign.name, Version: assign.version})
	}

	return result
}

// snapshot returns a human-readable representation of the partial solution.
// Intended for debug logging to understand solver state during complex conflicts.
func (ps *partialSolution) snapshot() string {
	var b strings.Builder
	fmt.Fprintf(&b, "decision_level=%d next_index=%d assignments=%d\n", ps.decisionLvl, ps.nextIndex, len(ps.assignments))
	for _, assign := range ps.assignments {
		fmt.Fprintf(&b, "  %s\n", assign.describe())
	}
	return b.String()
}

// pendingPackages lists packages that have constraints but no decided version yet.
// Used for diagnostics when analysing package selection order.
func (ps *partialSolution) pendingPackages() []Name {
	pending := make([]Name, 0)
	seen := make(map[Name]bool)

	for _, assign := range ps.assignments {
		name := assign.name
		if name == ps.root || seen[name] {
			continue
		}
		seen[name] = true

		if !ps.hasDecision(name) {
			pending = append(pending, name)
		}
	}

	return pending
}

// termSatisfiedBy checks if an assignment satisfies a term in an incompatibility.
func termSatisfiedBy(term Term, assign *assignment) bool {
	if assign == nil {
		return false
	}

	if term.Positive {
		required, ok := termAllowedSet(term)
		if !ok {
			return false
		}
		if assign.term.Positive {
			if assign.allowed == nil {
				return false
			}
			return assign.allowed.IsSubset(required)
		}
		if assign.allowed != nil {
			return assign.allowed.IsSubset(required)
		}
		return false
	}

	forbidden, ok := termForbiddenSet(term)
	if !ok {
		return false
	}

	if assign.term.Positive {
		if assign.allowed == nil {
			return false
		}
		return assign.allowed.IsDisjoint(forbidden)
	}

	if assign.forbidden == nil {
		return false
	}
	return forbidden.IsSubset(assign.forbidden)
}
