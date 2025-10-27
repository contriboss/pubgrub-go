package pubgrub

import "errors"

type assignmentKind int

const (
	assignmentDecision assignmentKind = iota
	assignmentDerivation
)

type assignment struct {
	name          Name
	term          Term
	kind          assignmentKind
	allowed       VersionSet
	forbidden     VersionSet
	version       Version
	cause         *Incompatibility
	decisionLevel int
	index         int
}

func (a *assignment) isDecision() bool {
	return a.kind == assignmentDecision
}

type partialSolution struct {
	assignments []*assignment
	perPackage  map[Name][]*assignment
	decisionLvl int
	nextIndex   int
	root        Name
}

func newPartialSolution(root Name) *partialSolution {
	return &partialSolution{
		assignments: make([]*assignment, 0),
		perPackage:  make(map[Name][]*assignment),
		decisionLvl: 0,
		nextIndex:   0,
		root:        root,
	}
}

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

func (ps *partialSolution) append(assign *assignment) {
	ps.assignments = append(ps.assignments, assign)
	stack := ps.perPackage[assign.name]
	stack = append(stack, assign)
	ps.perPackage[assign.name] = stack
	ps.nextIndex++
}

func (ps *partialSolution) latest(name Name) *assignment {
	stack := ps.perPackage[name]
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1]
}

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

func (ps *partialSolution) hasAssignments(name Name) bool {
	return len(ps.perPackage[name]) > 0
}

func (ps *partialSolution) addDecision(name Name, version Version) *assignment {
	ps.decisionLvl++
	assign := ps.newDecisionAssignment(name, version, ps.decisionLvl)
	ps.append(assign)
	return assign
}

func (ps *partialSolution) seedRoot(name Name, version Version) *assignment {
	assign := ps.newDecisionAssignment(name, version, 0)
	ps.append(assign)
	return assign
}

var errNoAllowedVersions = errors.New("no versions satisfy constraints")

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
		// record tightened allowance as positive assignment
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

func (ps *partialSolution) hasDecision(name Name) bool {
	stack := ps.perPackage[name]
	for _, assign := range stack {
		if assign.kind == assignmentDecision {
			return true
		}
	}
	return false
}

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
			return assign.allowed.IsSubset(required) || assign.allowed.IsDisjoint(required)
		}
		if assign.allowed != nil {
			return assign.allowed.IsSubset(required) || assign.allowed.IsDisjoint(required)
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
		return assign.allowed.IsDisjoint(forbidden) || assign.allowed.IsSubset(forbidden)
	}

	if assign.forbidden == nil {
		return false
	}
	return forbidden.IsSubset(assign.forbidden)
}

type solverState struct {
	source            Source
	options           SolverOptions
	partial           *partialSolution
	incompatibilities map[Name][]*Incompatibility
	learned           []*Incompatibility
	queue             []Name
	queued            map[Name]bool
}

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

func (st *solverState) enqueue(name Name) {
	if st.queued[name] {
		return
	}
	st.queue = append(st.queue, name)
	st.queued[name] = true
}

func (st *solverState) dequeue() (Name, bool) {
	if len(st.queue) == 0 {
		return EmptyName(), false
	}
	name := st.queue[0]
	st.queue = st.queue[1:]
	delete(st.queued, name)
	return name, true
}

func (st *solverState) addIncompatibility(incomp *Incompatibility) {
	for _, term := range incomp.Terms {
		st.incompatibilities[term.Name] = append(st.incompatibilities[term.Name], incomp)
	}
	if st.options.TrackIncompatibilities {
		st.learned = append(st.learned, incomp)
	}
}

func (st *solverState) markAssigned(name Name) {
}

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
				return inc, nil
			case relationAlmostSatisfied:
				if unsatisfied == nil {
					continue
				}
				assign, changed, err := st.partial.addDerivation(unsatisfied.Negate(), inc)
				if errors.Is(err, errNoAllowedVersions) {
					return inc, nil
				}
				if err != nil {
					return nil, err
				}
				if assign != nil {
					st.markAssigned(assign.name)
				}
				if changed && assign != nil {
					st.enqueue(assign.name)
				}
			}
		}
	}
}

type incompatibilityRelation int

const (
	relationSatisfied incompatibilityRelation = iota
	relationAlmostSatisfied
	relationContradicted
	relationInconclusive
)

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

func (st *solverState) applyConstraint(term Term, cause *Incompatibility) (*Incompatibility, error) {
	assign, _, err := st.partial.addDerivation(term, cause)
	if errors.Is(err, errNoAllowedVersions) {
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
		st.markAssigned(assign.name)
		st.enqueue(assign.name)
	}
	return nil, nil
}

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

func (st *solverState) resolveConflict(conflict *Incompatibility) (*Incompatibility, Name, error) {
	for {
		satisfier := st.partial.satisfier(conflict)
		if satisfier == nil {
			return nil, EmptyName(), NewNoSolutionError(conflict)
		}

		prevLevel := st.partial.previousDecisionLevel(conflict, satisfier)

		if satisfier.decisionLevel == 0 && satisfier.isDecision() {
			return nil, EmptyName(), NewNoSolutionError(conflict)
		}

		if satisfier.isDecision() && prevLevel < satisfier.decisionLevel {
			st.partial.backtrack(prevLevel)
			st.addIncompatibility(conflict)
			return nil, satisfier.name, nil
		}

		if satisfier.cause == nil {
			return nil, EmptyName(), errors.New("derived assignment missing cause")
		}

		conflict = resolveIncompatibility(conflict, satisfier.cause, satisfier.name)
	}
}
