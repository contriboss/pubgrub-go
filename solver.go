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

// Solver implements the PubGrub dependency resolution algorithm with CDCL.
//
// The solver uses Conflict-Driven Clause Learning (CDCL) to efficiently
// find valid package version assignments that satisfy all dependencies
// and constraints. It maintains learned incompatibilities to avoid
// repeating failed resolution attempts.
//
// Basic usage:
//
//	root := NewRootSource()
//	root.AddPackage("myapp", EqualsCondition{Version: SimpleVersion("1.0.0")})
//
//	source := &InMemorySource{}
//	// ... populate source with packages ...
//
//	solver := NewSolver(root, source)
//	solution, err := solver.Solve(root.Term())
//
// With options:
//
//	solver := NewSolverWithOptions(
//	    []Source{root, source},
//	    WithIncompatibilityTracking(true),
//	    WithMaxSteps(10000),
//	)
type Solver struct {
	Source  Source
	options SolverOptions

	learned []*Incompatibility
}

// NewSolver creates a new solver with default options from multiple sources.
// The sources are combined into a single CombinedSource that tries each source in order.
//
// Example:
//
//	root := NewRootSource()
//	source := &InMemorySource{}
//	solver := NewSolver(root, source)
func NewSolver(sources ...Source) *Solver {
	return NewSolverWithOptions(sources)
}

func NewSolverWithOptions(sources []Source, opts ...SolverOption) *Solver {
	options := defaultSolverOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return &Solver{
		Source:  CombinedSource(sources),
		options: options,
		learned: nil,
	}
}

func (s *Solver) Configure(opts ...SolverOption) *Solver {
	for _, opt := range opts {
		if opt != nil {
			opt(&s.options)
		}
	}
	return s
}

func (s *Solver) EnableIncompatibilityTracking() *Solver {
	return s.Configure(WithIncompatibilityTracking(true))
}

func (s *Solver) DisableIncompatibilityTracking() *Solver {
	return s.Configure(WithIncompatibilityTracking(false))
}

func (s *Solver) GetIncompatibilities() []*Incompatibility {
	return s.learned
}

func (s *Solver) ClearIncompatibilities() {
	clear(s.learned)
	s.learned = s.learned[:0]
}

func (s *Solver) Solve(root Term) (Solution, error) {
	if s.options.Logger != nil {
		s.options.Logger.Debug("starting solver", "root", root)
	}

	state := newSolverState(s.Source, s.options, root.Name)

	version, err := extractDecisionVersion(root)
	if err != nil {
		return nil, err
	}

	assign := state.partial.seedRoot(root.Name, version)
	state.markAssigned(root.Name)

	if s.options.Logger != nil {
		s.options.Logger.Debug("seeded root", "package", root.Name, "version", version)
	}

	deps, err := s.Source.GetDependencies(root.Name, version)
	if err != nil {
		return nil, &DependencyError{Package: root.Name, Version: version, Err: err}
	}

	var conflict *Incompatibility
	if depConflict, err := state.registerDependencies(root.Name, version, deps); err != nil {
		return nil, &DependencyError{Package: root.Name, Version: version, Err: err}
	} else if depConflict != nil {
		conflict = depConflict
	}

	state.enqueue(assign.name)

	var propagateSeed Name

	for steps := 0; ; steps++ {
		if s.options.MaxSteps > 0 && steps >= s.options.MaxSteps {
			return nil, ErrIterationLimit{Steps: s.options.MaxSteps}
		}

		if conflict != nil {
			if s.options.Logger != nil {
				s.options.Logger.Debug("resolving conflict", "step", steps, "conflict", conflict)
			}
			_, pivot, err := state.resolveConflict(conflict)
			if err != nil {
				if ns, ok := err.(*NoSolutionError); ok {
					return s.fail(state, ns.Incompatibility)
				}
				return nil, err
			}
			conflict = nil
			if pivot != EmptyName() {
				propagateSeed = pivot
			}
			continue
		}

		seed := propagateSeed
		propagateSeed = EmptyName()
		propConflict, err := state.propagate(seed)
		if err != nil {
			return nil, err
		}
		if propConflict != nil {
			conflict = propConflict
			continue
		}

		if state.partial.isComplete() {
			return state.partial.buildSolution(), nil
		}

		nextPkg, ok := state.partial.nextDecisionCandidate()
		if !ok {
			if s.options.Logger != nil {
				s.options.Logger.Debug("solution found", "step", steps)
			}
			return state.partial.buildSolution(), nil
		}

		if s.options.Logger != nil {
			s.options.Logger.Debug("selecting package", "step", steps, "package", nextPkg)
		}

		ver, found, err := state.pickVersion(nextPkg)
		if err != nil {
			return nil, err
		}
		if !found {
			allowed := state.partial.allowedSet(nextPkg)
			conflict = NewIncompatibilityNoVersions(termFromAllowedSet(nextPkg, allowed))

			if support := state.partial.latest(nextPkg); support != nil && support.cause != nil {
				conflict = resolveIncompatibility(conflict, support.cause, nextPkg)
			}
			state.addIncompatibility(conflict)
			continue
		}

		if s.options.Logger != nil {
			s.options.Logger.Debug("making decision", "step", steps, "package", nextPkg, "version", ver)
		}

		assign := state.partial.addDecision(nextPkg, ver)
		state.markAssigned(assign.name)

		deps, err := s.Source.GetDependencies(nextPkg, ver)
		if err != nil {
			return nil, &DependencyError{Package: nextPkg, Version: ver, Err: err}
		}

		if depConflict, err := state.registerDependencies(nextPkg, ver, deps); err != nil {
			return nil, &DependencyError{Package: nextPkg, Version: ver, Err: err}
		} else if depConflict != nil {
			conflict = depConflict
			continue
		}

		state.enqueue(assign.name)
	}
}

func extractDecisionVersion(root Term) (Version, error) {
	if !root.Positive {
		return nil, &VersionError{Package: root.Name, Message: "root term must be positive"}
	}

	switch cond := root.Condition.(type) {
	case EqualsCondition:
		return cond.Version, nil
	case *EqualsCondition:
		if cond == nil {
			return nil, &VersionError{Package: root.Name, Message: "root condition missing version"}
		}
		return cond.Version, nil
	default:
		return nil, &VersionError{Package: root.Name, Message: "root must pin exact version"}
	}
}

func (s *Solver) fail(state *solverState, incomp *Incompatibility) (Solution, error) {
	if s.options.TrackIncompatibilities {
		if state != nil {
			s.learned = append([]*Incompatibility{}, state.learned...)
		}
		if incomp == nil {
			term := fallbackTerm(nil)
			incomp = NewIncompatibilityNoVersions(term)
		}
		return nil, NewNoSolutionError(incomp)
	}

	term := fallbackTerm(incomp)
	return nil, ErrNoSolutionFound{Term: term}
}

func fallbackTerm(incomp *Incompatibility) Term {
	if incomp == nil || len(incomp.Terms) == 0 {
		return NewTerm(MakeName("$$root"), nil)
	}
	term := incomp.Terms[0]
	if !term.Positive {
		term = term.Negate()
	}
	return term
}
