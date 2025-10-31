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

import "errors"

// solverState maintains all mutable state during CDCL-based dependency resolution.
// It coordinates between:
//   - The partial solution (current assignments and decisions)
//   - Incompatibilities (learned conflicts and dependency constraints)
//   - Unit propagation queue (packages needing constraint propagation)
//
// The solver state implements the core CDCL algorithm:
//  1. Make decisions (pick package versions)
//  2. Propagate constraints (unit propagation)
//  3. Detect conflicts (incompatibility satisfaction)
//  4. Analyze conflicts (conflict resolution)
//  5. Learn clauses (add derived incompatibilities)
//  6. Backtrack (undo decisions to earlier state)
type solverState struct {
	source            Source                      // Package version and dependency source
	options           SolverOptions               // Solver configuration
	partial           *partialSolution            // Current partial solution
	incompatibilities map[Name][]*Incompatibility // Incompatibilities indexed by package
	learned           []*Incompatibility          // Learned incompatibilities (for error reporting)
	queue             []Name                      // Unit propagation queue
	queued            map[Name]bool               // Tracks which packages are queued
}

// newSolverState creates a new solver state for the given source and root package.
func newSolverState(source Source, options SolverOptions, root Name) *solverState {
	return &solverState{
		source:            source,
		options:           options,
		partial:           newPartialSolution(root),
		incompatibilities: make(map[Name][]*Incompatibility),
		learned:           make([]*Incompatibility, 0),
		queue:             make([]Name, 0),
		queued:            make(map[Name]bool),
	}
}

// enqueue adds a package to the unit propagation queue if not already queued.
func (st *solverState) enqueue(name Name) {
	if st.queued[name] {
		return
	}
	st.queue = append(st.queue, name)
	st.queued[name] = true
}

// dequeue removes and returns the next package from the propagation queue.
func (st *solverState) dequeue() (Name, bool) {
	if len(st.queue) == 0 {
		return EmptyName(), false
	}
	name := st.queue[0]
	st.queue = st.queue[1:]
	delete(st.queued, name)
	return name, true
}

// addIncompatibility registers an incompatibility for all involved packages.
// If tracking is enabled, also adds it to the learned clauses list.
func (st *solverState) addIncompatibility(incomp *Incompatibility) {
	for _, term := range incomp.Terms {
		st.incompatibilities[term.Name] = append(st.incompatibilities[term.Name], incomp)
	}
	if st.options.TrackIncompatibilities {
		st.learned = append(st.learned, incomp)
	}
}

// markAssigned is called when a package receives an assignment.
// Currently a no-op, but provides extension point for future optimizations.
func (st *solverState) markAssigned(name Name) {
}

func (st *solverState) debug(msg string, args ...any) {
	if st.options.Logger == nil {
		return
	}
	st.options.Logger.Debug(msg, args...)
}

func (st *solverState) traceAssignment(event string, assign *assignment) {
	if st.options.Logger == nil || assign == nil {
		return
	}
	st.options.Logger.Debug("assignment",
		"event", event,
		"package", assign.name.Value(),
		"detail", assign.describe(),
	)
}

// propagate performs unit propagation starting from a package.
// Returns a conflict incompatibility if one is detected, or nil if propagation succeeds.
//
// Unit propagation iteratively:
//  1. Dequeues a package from the propagation queue
//  2. Checks all incompatibilities involving that package
//  3. If an incompatibility is "almost satisfied" (one unsatisfied term),
//     derives the negation of that term as a new constraint
//  4. Enqueues newly constrained packages for further propagation
func (st *solverState) propagate(start Name) (*Incompatibility, error) {
	if start != EmptyName() {
		st.enqueue(start)
	}

	for {
		pkg, ok := st.dequeue()
		if !ok {
			return nil, nil
		}

		for _, inc := range st.incompatibilities[pkg] {
			relation, unsatisfied, err := st.evaluateIncompatibility(inc)
			if err != nil {
				return nil, err
			}

			switch relation {
			case relationSatisfied:
				st.debug("conflict detected during propagation",
					"package", pkg.Value(),
					"incompatibility", inc.String(),
				)
				return inc, nil
			case relationAlmostSatisfied:
				if unsatisfied == nil {
					continue
				}
				derived := unsatisfied.Negate()
				st.debug("unit propagation",
					"package", pkg.Value(),
					"incompatibility", inc.String(),
					"derived_term", derived.String(),
				)
				assign, changed, err := st.partial.addDerivation(derived, inc)
				if errors.Is(err, errNoAllowedVersions) {
					return inc, nil
				}
				if err != nil {
					return nil, err
				}
				if assign != nil {
					st.traceAssignment("derivation", assign)
					st.markAssigned(assign.name)
				}
				if changed && assign != nil {
					st.debug("enqueueing package after derivation",
						"package", assign.name.Value(),
						"term", assign.term.String(),
					)
					st.enqueue(assign.name)
				}
			}
		}
	}
}

// incompatibilityRelation describes the relationship between an incompatibility
// and the current partial solution.
type incompatibilityRelation int

const (
	relationSatisfied       incompatibilityRelation = iota // All terms satisfied (conflict!)
	relationAlmostSatisfied                                // All but one term satisfied (unit propagation)
	relationContradicted                                   // At least one term contradicted (incompatibility inapplicable)
	relationInconclusive                                   // Multiple terms unsatisfied (wait for more decisions)
)

// evaluateIncompatibility determines the relationship between an incompatibility
// and the current partial solution.
func (st *solverState) evaluateIncompatibility(inc *Incompatibility) (incompatibilityRelation, *Term, error) {
	var unsatisfied *Term

	for _, term := range inc.Terms {
		allowed := st.partial.allowedSet(term.Name)
		rel, err := relationForTerm(term, allowed, st.partial.hasAssignments(term.Name))
		if err != nil {
			return relationInconclusive, nil, err
		}

		switch rel {
		case relationContradicted:
			return relationContradicted, nil, nil
		case relationSatisfied:
			continue
		case relationInconclusive:
			if unsatisfied != nil {
				return relationInconclusive, nil, nil
			}
			temp := term
			unsatisfied = &temp
		}
	}

	if unsatisfied == nil {
		return relationSatisfied, nil, nil
	}
	return relationAlmostSatisfied, unsatisfied, nil
}

// relationForTerm determines the relationship between a single term and the
// current allowed version set for its package.
func relationForTerm(term Term, allowed VersionSet, hasAssignment bool) (incompatibilityRelation, error) {
	if allowed == nil {
		allowed = FullVersionSet()
	}

	if term.Positive {
		required, ok := termAllowedSet(term)
		if !ok {
			return relationInconclusive, nil
		}
		if allowed.IsSubset(required) {
			if hasAssignment {
				return relationSatisfied, nil
			}
			return relationInconclusive, nil
		}
		if allowed.IsDisjoint(required) {
			return relationContradicted, nil
		}
		return relationInconclusive, nil
	}

	forbidden, ok := termForbiddenSet(term)
	if !ok {
		return relationInconclusive, nil
	}

	if allowed.IsDisjoint(forbidden) {
		return relationSatisfied, nil
	}
	if allowed.IsSubset(forbidden) {
		if hasAssignment {
			return relationContradicted, nil
		}
		return relationInconclusive, nil
	}
	return relationInconclusive, nil
}

// resolveIncompatibility performs conflict resolution by merging two incompatibilities.
// This is the core of CDCL's learned clause generation.
//
// Given:
//   - conflict: An incompatibility satisfied by the current solution
//   - cause: The incompatibility that caused a specific assignment
//   - pkg: The package whose assignment we're resolving
//
// Returns a new incompatibility with:
//   - All terms from conflict except pkg's term
//   - All terms from cause except pkg's term
//   - Merged terms where both incompatibilities constrain the same package
func resolveIncompatibility(conflict, cause *Incompatibility, pkg Name) *Incompatibility {
	terms := make(map[Name]Term)

	for _, term := range conflict.Terms {
		if term.Name == pkg {
			continue
		}
		terms[term.Name] = term
	}

	for _, term := range cause.Terms {
		if term.Name == pkg {
			continue
		}
		if existing, ok := terms[term.Name]; ok {
			if merged, ok := mergeTerms(existing, term); ok {
				terms[term.Name] = merged
				continue
			}
		}
		terms[term.Name] = term
	}

	// Build slice with deterministic order to ensure consistent output
	merged := make([]Term, 0, len(terms))

	// First add terms from conflict in their original order
	for _, term := range conflict.Terms {
		if term.Name == pkg {
			continue
		}
		if t, ok := terms[term.Name]; ok {
			merged = append(merged, t)
			delete(terms, term.Name)
		}
	}

	// Then add any remaining terms from cause
	for _, term := range cause.Terms {
		if term.Name == pkg {
			continue
		}
		if t, ok := terms[term.Name]; ok {
			merged = append(merged, t)
			delete(terms, term.Name)
		}
	}

	return NewIncompatibilityConflict(merged, conflict, cause)
}

// mergeTerms combines two terms for the same package during conflict resolution.
// For positive terms, takes intersection of version sets.
// For negative terms, takes union of forbidden sets.
func mergeTerms(a, b Term) (Term, bool) {
	if a.Name != b.Name {
		return Term{}, false
	}

	switch {
	case a.Positive && b.Positive:
		setA, okA := termAllowedSet(a)
		setB, okB := termAllowedSet(b)
		if !okA || !okB {
			return Term{}, false
		}
		return termFromAllowedSet(a.Name, setA.Intersection(setB)), true
	case !a.Positive && !b.Positive:
		forbA, okA := termForbiddenSet(a)
		forbB, okB := termForbiddenSet(b)
		if !okA || !okB {
			return Term{}, false
		}
		return termFromForbiddenSet(a.Name, forbA.Union(forbB)), true
	default:
		return Term{}, false
	}
}

// registerDependencies adds incompatibilities for a package version's dependencies.
// Returns a conflict incompatibility if constraint application fails.
func (st *solverState) registerDependencies(pkg Name, version Version, deps []Term) (*Incompatibility, error) {
	for _, dep := range deps {
		incomp := NewIncompatibilityFromDependency(pkg, version, dep)
		st.addIncompatibility(incomp)
		conflict, err := st.applyConstraint(dep, incomp)
		if err != nil {
			return nil, err
		}
		if conflict != nil {
			return conflict, nil
		}
	}
	return nil, nil
}

// applyConstraint applies a dependency constraint to the partial solution.
// Returns a conflict incompatibility if the constraint cannot be satisfied.
func (st *solverState) applyConstraint(term Term, cause *Incompatibility) (*Incompatibility, error) {
	assign, _, err := st.partial.addDerivation(term, cause)
	if errors.Is(err, errNoAllowedVersions) {
		causeDesc := "<nil>"
		if cause != nil {
			causeDesc = cause.String()
		}
		st.debug("constraint left no allowed versions",
			"term", term.String(),
			"cause", causeDesc,
		)
		base := NewIncompatibilityNoVersions(term)
		if cause != nil {
			terms := make([]Term, 0, len(cause.Terms)+len(base.Terms))
			terms = append(terms, cause.Terms...)
			terms = append(terms, base.Terms...)
			return NewIncompatibilityConflict(terms, base, cause), nil
		}
		return base, nil
	}
	if err != nil {
		return nil, err
	}
	if assign != nil {
		st.traceAssignment("dependency-constraint", assign)
		st.markAssigned(assign.name)
		st.enqueue(assign.name)
	}
	return nil, nil
}

// pickVersion selects the best available version for a package from the source.
// Returns the version if found, or (nil, false) if no suitable version exists.
//
// Selection strategy:
//  1. Get all available versions from the source
//  2. Filter to versions matching current constraints
//  3. Return the highest version (versions are pre-sorted by source)
func (st *solverState) pickVersion(name Name) (Version, bool, error) {
	allowed := st.partial.allowedSet(name)
	if allowed == nil || allowed.IsEmpty() {
		return nil, false, nil
	}

	versions, err := st.source.GetVersions(name)
	if err != nil {
		var pkgErr *PackageNotFoundError
		var verErr *PackageVersionNotFoundError
		if errors.As(err, &pkgErr) || errors.As(err, &verErr) {
			return nil, false, nil
		}
		return nil, false, err
	}
	for i := len(versions) - 1; i >= 0; i-- {
		ver := versions[i]
		if allowed.Contains(ver) {
			return ver, true, nil
		}
	}

	return nil, false, nil
}

// resolveConflict performs conflict analysis and backtracking via CDCL.
// Returns:
//   - (nil, pkg, nil) to continue solving with backtracking to decision level for pkg
//   - (nil, EmptyName, error) if the conflict is unsolvable (root-level conflict)
//
// The algorithm:
//  1. Find the satisfier (most recent assignment satisfying the conflict)
//  2. If satisfier is a root-level decision, the problem is unsolvable
//  3. If satisfier is a decision at a higher level than other satisfying assignments,
//     backtrack to the previous decision level and learn the conflict
//  4. If satisfier is a derivation, resolve it with its cause and continue
func (st *solverState) resolveConflict(conflict *Incompatibility) (*Incompatibility, Name, error) {
	for {
		satisfier := st.partial.satisfier(conflict)
		if satisfier == nil {
			return nil, EmptyName(), NewNoSolutionError(conflict)
		}

		prevLevel := st.partial.previousDecisionLevel(conflict, satisfier)
		st.debug("conflict analysis iteration",
			"conflict", conflict.String(),
			"satisfier", satisfier.describe(),
			"satisfier_level", satisfier.decisionLevel,
			"previous_level", prevLevel,
		)

		if satisfier.decisionLevel == 0 && satisfier.isDecision() {
			return nil, EmptyName(), NewNoSolutionError(conflict)
		}

		if satisfier.isDecision() && prevLevel < satisfier.decisionLevel {
			st.partial.backtrack(prevLevel)
			if st.options.Logger != nil {
				st.options.Logger.Debug("backtracked after conflict",
					"pivot", satisfier.name.Value(),
					"target_level", prevLevel,
					"learned", conflict.String(),
					"state", st.partial.snapshot(),
				)
			}
			st.addIncompatibility(conflict)
			return nil, satisfier.name, nil
		}

		if satisfier.cause == nil {
			return nil, EmptyName(), errors.New("derived assignment missing cause")
		}

		st.debug("resolving with cause",
			"pivot", satisfier.name.Value(),
			"cause", satisfier.cause.String(),
		)
		conflict = resolveIncompatibility(conflict, satisfier.cause, satisfier.name)
		st.debug("derived new conflict",
			"pivot", satisfier.name.Value(),
			"conflict", conflict.String(),
		)
	}
}
