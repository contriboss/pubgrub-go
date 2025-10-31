package pubgrub

import "testing"

func TestPartialSolutionPreviousDecisionLevel(t *testing.T) {
	root := MakeName("root")
	ps := newPartialSolution(root)
	rootVersion := SimpleVersion("1.0.0")
	ps.seedRoot(root, rootVersion)

	a := MakeName("a")
	aVersion := SimpleVersion("1.0.0")
	ps.addDecision(a, aVersion)

	b := MakeName("b")
	bVersion := SimpleVersion("1.0.0")
	assignB := ps.addDecision(b, bVersion)

	inc := &Incompatibility{
		Terms: []Term{
			NewTerm(a, EqualsCondition{Version: aVersion}),
			NewTerm(b, EqualsCondition{Version: bVersion}),
		},
		Kind: KindConflict,
	}

	satisfier := ps.satisfier(inc)
	if satisfier == nil {
		t.Fatalf("expected satisfier, got nil")
	}
	if satisfier != assignB {
		t.Fatalf("expected satisfier to be assignment for %s, got %s", b.Value(), satisfier.name.Value())
	}

	prev := ps.previousDecisionLevel(inc, satisfier)
	if prev != 1 {
		t.Fatalf("expected previous decision level 1, got %d", prev)
	}
}
