package pubgrub

import "fmt"

func termAllowedSet(term Term) (VersionSet, bool) {
	if !term.Positive {
		return nil, false
	}

	switch cond := term.Condition.(type) {
	case nil:
		return (&VersionIntervalSet{}).Full(), true
	case EqualsCondition:
		return (&VersionIntervalSet{}).Singleton(cond.Version), true
	case *EqualsCondition:
		if cond == nil {
			return (&VersionIntervalSet{}).Full(), true
		}
		return (&VersionIntervalSet{}).Singleton(cond.Version), true
	case *VersionSetCondition:
		if cond == nil || cond.Set == nil {
			return (&VersionIntervalSet{}).Full(), true
		}
		return cond.Set, true
	default:
		return nil, false
	}
}

func termForbiddenSet(term Term) (VersionSet, bool) {
	if term.Positive {
		return nil, false
	}

	switch cond := term.Condition.(type) {
	case nil:
		return (&VersionIntervalSet{}).Full(), true
	case EqualsCondition:
		return (&VersionIntervalSet{}).Singleton(cond.Version), true
	case *EqualsCondition:
		if cond == nil {
			return (&VersionIntervalSet{}).Full(), true
		}
		return (&VersionIntervalSet{}).Singleton(cond.Version), true
	case *VersionSetCondition:
		if cond == nil || cond.Set == nil {
			return (&VersionIntervalSet{}).Full(), true
		}
		return cond.Set, true
	default:
		return nil, false
	}
}

func applyTermToAllowed(current VersionSet, term Term) (VersionSet, error) {
	if current == nil {
		current = (&VersionIntervalSet{}).Full()
	}

	if term.Positive {
		allowed, ok := termAllowedSet(term)
		if !ok {
			return nil, fmt.Errorf("term %s does not support positive conversion", term)
		}
		return current.Intersection(allowed), nil
	}

	forbidden, ok := termForbiddenSet(term)
	if !ok {
		return nil, fmt.Errorf("term %s does not support negative conversion", term)
	}
	return current.Intersection(forbidden.Complement()), nil
}

func termFromAllowedSet(name Name, set VersionSet) Term {
	if set == nil {
		set = (&VersionIntervalSet{}).Full()
	}

	if version, ok := singletonVersionFromSet(set); ok {
		return Term{
			Name:      name,
			Condition: EqualsCondition{Version: version},
			Positive:  true,
		}
	}

	return Term{
		Name:      name,
		Condition: NewVersionSetCondition(set),
		Positive:  true,
	}
}

func termFromForbiddenSet(name Name, set VersionSet) Term {
	if set == nil {
		set = (&VersionIntervalSet{}).Full()
	}

	return Term{
		Name:      name,
		Condition: NewVersionSetCondition(set),
		Positive:  false,
	}
}

func setsEqual(a, b VersionSet) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.IsSubset(b) && b.IsSubset(a)
}
