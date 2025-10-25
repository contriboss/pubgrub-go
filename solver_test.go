package pubgrub

import (
	"errors"
	"strings"
	"testing"
)

func TestSolverSimpleGraph(t *testing.T) {
	source := &InMemorySource{}

	v100, _ := ParseSemanticVersion("1.0.0")
	v110, _ := ParseSemanticVersion("1.1.0")
	b200, _ := ParseSemanticVersion("2.0.0")
	b210, _ := ParseSemanticVersion("2.1.0")

	range1x, _ := ParseVersionRange(">=1.0.0, <2.0.0")
	range2x, _ := ParseVersionRange(">=2.0.0")

	source.AddPackage(MakeName("A"), v100, nil)
	source.AddPackage(MakeName("A"), v110, []Term{
		NewTerm(MakeName("B"), NewVersionSetCondition(range2x)),
	})
	source.AddPackage(MakeName("B"), b200, nil)
	source.AddPackage(MakeName("B"), b210, nil)

	root := NewRootSource()
	root.AddPackage(MakeName("A"), NewVersionSetCondition(range1x))

	solver := NewSolver(root, source)
	solution, err := solver.Solve(root.Term())
	if err != nil {
		t.Fatalf("Solve returned error: %v", err)
	}

	check := func(name Name, want string) {
		ver, ok := solution.GetVersion(name)
		if !ok {
			t.Fatalf("expected %s in solution", name.Value())
		}
		if ver.String() != want {
			t.Fatalf("expected %s to be %s, got %s", name.Value(), want, ver.String())
		}
	}

	check(MakeName("A"), "1.1.0")
	check(MakeName("B"), "2.1.0")
}

func TestSolverConflictTracking(t *testing.T) {
	source := &InMemorySource{}
	source.AddPackage(MakeName("A"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})
	source.AddPackage(MakeName("B"), SimpleVersion("1.0.0"), nil)
	source.AddPackage(MakeName("B"), SimpleVersion("2.0.0"), nil)
	source.AddPackage(MakeName("C"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("2.0.0")}),
	})

	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	root.AddPackage(MakeName("C"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source).EnableIncompatibilityTracking()
	_, err := solver.Solve(root.Term())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	nsErr, ok := err.(*NoSolutionError)
	if !ok {
		t.Fatalf("expected *NoSolutionError, got %T", err)
	}

	if !strings.Contains(nsErr.Error(), "Because C 1.0.0 depends on B == 2.0.0") {
		t.Fatalf("unexpected error message: %v", nsErr.Error())
	}

	incomps := solver.GetIncompatibilities()
	if len(incomps) == 0 {
		t.Fatalf("expected tracked incompatibilities, got 0")
	}
}

func TestSolverConflictNoTracking(t *testing.T) {
	source := &InMemorySource{}
	source.AddPackage(MakeName("foo"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("bar"), EqualsCondition{Version: SimpleVersion("2.0.0")}),
	})
	source.AddPackage(MakeName("bar"), SimpleVersion("1.0.0"), nil)

	root := NewRootSource()
	root.AddPackage(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source)
	_, err := solver.Solve(root.Term())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if _, ok := err.(ErrNoSolutionFound); !ok {
		t.Fatalf("expected ErrNoSolutionFound, got %T", err)
	}
}

func TestSolverBacktrackingChoosesAlternateVersion(t *testing.T) {
	source := &InMemorySource{}

	a110, _ := ParseSemanticVersion("1.1.0")
	b100, _ := ParseSemanticVersion("1.0.0")
	b200, _ := ParseSemanticVersion("2.0.0")

	anyB, _ := ParseVersionRange(">=1.0.0")

	source.AddPackage(MakeName("A"), a110, []Term{
		NewTerm(MakeName("B"), NewVersionSetCondition(anyB)),
	})
	source.AddPackage(MakeName("B"), b100, nil)
	source.AddPackage(MakeName("B"), b200, []Term{
		NewTerm(MakeName("D"), EqualsCondition{Version: SimpleVersion("1.0.0")}),
	})

	root := NewRootSource()
	root.AddPackage(MakeName("A"), EqualsCondition{Version: a110})

	solver := NewSolver(root, source)
	solution, err := solver.Solve(root.Term())
	if err != nil {
		t.Fatalf("Solve returned error: %v", err)
	}

	ver, ok := solution.GetVersion(MakeName("B"))
	if !ok {
		t.Fatalf("expected B in solution")
	}
	if ver.String() != "1.0.0" {
		t.Fatalf("expected backtracking to select B 1.0.0, got %s", ver.String())
	}
}

func TestSolverOptionMaxSteps(t *testing.T) {
	root := NewRootSource()
	root.AddPackage(MakeName("ghost"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolverWithOptions([]Source{root}, WithMaxSteps(1))
	_, err := solver.Solve(root.Term())
	if err == nil {
		t.Fatalf("expected iteration limit error")
	}
	var limitErr ErrIterationLimit
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected ErrIterationLimit, got %T", err)
	}
}

func TestSolverCombinedSourcePrefersHighestVersion(t *testing.T) {
	sourceA := &InMemorySource{}
	sourceB := &InMemorySource{}

	v100, _ := ParseSemanticVersion("1.0.0")
	v120, _ := ParseSemanticVersion("1.2.0")
	rangeAny, _ := ParseVersionRange(">=1.0.0, <2.0.0")

	sourceA.AddPackage(MakeName("pkg"), v100, nil)
	sourceB.AddPackage(MakeName("pkg"), v120, nil)

	root := NewRootSource()
	root.AddPackage(MakeName("pkg"), NewVersionSetCondition(rangeAny))

	solver := NewSolver(root, sourceA, sourceB)
	solution, err := solver.Solve(root.Term())
	if err != nil {
		t.Fatalf("Solve returned error: %v", err)
	}

	ver, ok := solution.GetVersion(MakeName("pkg"))
	if !ok {
		t.Fatalf("expected pkg in solution")
	}
	if got := ver.String(); got != "1.2.0" {
		t.Fatalf("expected highest version 1.2.0, got %s", got)
	}
}

func TestSolverHandlesPrereleaseRanges(t *testing.T) {
	source := &InMemorySource{}

	preA, _ := ParseSemanticVersion("1.0.0-alpha.1")
	preB, _ := ParseSemanticVersion("1.0.0-beta.1")
	rangePre, _ := ParseVersionRange(">=1.0.0-alpha.1, <1.0.0")

	source.AddPackage(MakeName("lib"), preA, nil)
	source.AddPackage(MakeName("lib"), preB, nil)

	root := NewRootSource()
	root.AddPackage(MakeName("lib"), NewVersionSetCondition(rangePre))

	solver := NewSolver(root, source)
	solution, err := solver.Solve(root.Term())
	if err != nil {
		t.Fatalf("Solve returned error: %v", err)
	}

	ver, ok := solution.GetVersion(MakeName("lib"))
	if !ok {
		t.Fatalf("expected lib in solution")
	}
	if got := ver.String(); got != "1.0.0-beta.1" {
		t.Fatalf("expected prerelease selection 1.0.0-beta.1, got %s", got)
	}
}
