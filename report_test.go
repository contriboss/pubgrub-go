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
	"strings"
	"testing"
)

func TestIncompatibilityNoVersions(t *testing.T) {
	term := NewTerm(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	incomp := NewIncompatibilityNoVersions(term)

	if incomp.Kind != KindNoVersions {
		t.Error("Expected KindNoVersions")
	}
	if len(incomp.Terms) != 1 {
		t.Errorf("Expected 1 term, got %d", len(incomp.Terms))
	}

	str := incomp.String()
	if !strings.Contains(str, "foo") {
		t.Errorf("Expected string to contain 'foo', got: %s", str)
	}
}

func TestIncompatibilityFromDependency(t *testing.T) {
	dep := NewTerm(MakeName("bar"), EqualsCondition{Version: SimpleVersion("2.0.0")})
	incomp := NewIncompatibilityFromDependency(MakeName("foo"), SimpleVersion("1.0.0"), dep)

	if incomp.Kind != KindFromDependency {
		t.Error("Expected KindFromDependency")
	}
	if len(incomp.Terms) != 2 {
		t.Errorf("Expected 2 terms, got %d", len(incomp.Terms))
	}
	if incomp.Package != MakeName("foo") {
		t.Errorf("Expected package 'foo', got %s", incomp.Package.Value())
	}

	str := incomp.String()
	if !strings.Contains(str, "foo") || !strings.Contains(str, "bar") {
		t.Errorf("Expected string to contain both packages, got: %s", str)
	}
}

func TestIncompatibilityConflict(t *testing.T) {
	term1 := NewTerm(MakeName("A"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	incomp1 := NewIncompatibilityNoVersions(term1)

	term2 := NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("2.0.0")})
	incomp2 := NewIncompatibilityNoVersions(term2)

	conflict := NewIncompatibilityConflict([]Term{}, incomp1, incomp2)

	if conflict.Kind != KindConflict {
		t.Error("Expected KindConflict")
	}
	if conflict.Cause1 != incomp1 || conflict.Cause2 != incomp2 {
		t.Error("Causes don't match")
	}
}

func TestDefaultReporter_NoVersions(t *testing.T) {
	reporter := &DefaultReporter{}
	term := NewTerm(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	incomp := NewIncompatibilityNoVersions(term)

	result := reporter.Report(incomp)
	t.Logf("Output: %s", result)

	if !strings.Contains(result, "foo") {
		t.Errorf("Expected output to mention 'foo', got: %s", result)
	}
	if !strings.Contains(result, "No versions") {
		t.Errorf("Expected output to mention 'No versions', got: %s", result)
	}
}

func TestDefaultReporter_FromDependency(t *testing.T) {
	reporter := &DefaultReporter{}
	dep := NewTerm(MakeName("bar"), EqualsCondition{Version: SimpleVersion("2.0.0")})
	incomp := NewIncompatibilityFromDependency(MakeName("foo"), SimpleVersion("1.0.0"), dep)

	result := reporter.Report(incomp)
	t.Logf("Output: %s", result)

	if !strings.Contains(result, "foo") || !strings.Contains(result, "bar") {
		t.Errorf("Expected output to mention both packages, got: %s", result)
	}
	if !strings.Contains(result, "depends") {
		t.Errorf("Expected output to mention 'depends', got: %s", result)
	}
}

func TestDefaultReporter_Conflict(t *testing.T) {
	reporter := &DefaultReporter{}

	// Create a conflict scenario
	dep1 := NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("2.0.0")})
	incomp1 := NewIncompatibilityFromDependency(MakeName("A"), SimpleVersion("1.0.0"), dep1)

	dep2 := NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	incomp2 := NewIncompatibilityFromDependency(MakeName("C"), SimpleVersion("1.0.0"), dep2)

	conflict := NewIncompatibilityConflict([]Term{}, incomp1, incomp2)

	result := reporter.Report(conflict)
	t.Logf("Output:\n%s", result)

	if !strings.Contains(result, "Because") {
		t.Errorf("Expected output to contain 'Because', got: %s", result)
	}
}

func TestCollapsedReporter_NoVersions(t *testing.T) {
	reporter := &CollapsedReporter{}
	term := NewTerm(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	incomp := NewIncompatibilityNoVersions(term)

	result := reporter.Report(incomp)
	t.Logf("Output: %s", result)

	if !strings.Contains(result, "foo") {
		t.Errorf("Expected output to mention 'foo', got: %s", result)
	}
}

func TestCollapsedReporter_Conflict(t *testing.T) {
	reporter := &CollapsedReporter{}

	dep1 := NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("2.0.0")})
	incomp1 := NewIncompatibilityFromDependency(MakeName("A"), SimpleVersion("1.0.0"), dep1)

	dep2 := NewTerm(MakeName("B"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	incomp2 := NewIncompatibilityFromDependency(MakeName("C"), SimpleVersion("1.0.0"), dep2)

	conflict := NewIncompatibilityConflict([]Term{}, incomp1, incomp2)

	result := reporter.Report(conflict)
	t.Logf("Output:\n%s", result)

	if result == "" {
		t.Error("Expected non-empty output")
	}
}

func TestNoSolutionError_Basic(t *testing.T) {
	term := NewTerm(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	incomp := NewIncompatibilityNoVersions(term)
	err := NewNoSolutionError(incomp)

	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
	if !strings.Contains(err.Error(), "foo") {
		t.Errorf("Expected error to mention foo, got: %s", err.Error())
	}
}

func TestNoSolutionError_WithReporter(t *testing.T) {
	term := NewTerm(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	incomp := NewIncompatibilityNoVersions(term)
	err := NewNoSolutionError(incomp)
	customErr := err.WithReporter(&CollapsedReporter{})

	if customErr.Reporter == nil {
		t.Error("Custom reporter should be set")
	}
	if _, ok := customErr.Reporter.(*CollapsedReporter); !ok {
		t.Error("Reporter should be CollapsedReporter")
	}
}

func TestNoSolutionError_Nil(t *testing.T) {
	err := &NoSolutionError{Incompatibility: nil}
	if err.Error() != "no solution found" {
		t.Errorf("Expected 'no solution found', got: %s", err.Error())
	}
}

func TestVersionError(t *testing.T) {
	err := &VersionError{
		Package: MakeName("foo"),
		Message: "test error",
	}

	if !strings.Contains(err.Error(), "foo") {
		t.Errorf("Expected error to contain package name, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "test error") {
		t.Errorf("Expected error to contain message, got: %s", err.Error())
	}
}

func TestDependencyError(t *testing.T) {
	innerErr := &VersionError{
		Package: MakeName("bar"),
		Message: "inner error",
	}
	err := &DependencyError{
		Package: MakeName("foo"),
		Version: SimpleVersion("1.0.0"),
		Err:     innerErr,
	}

	if !strings.Contains(err.Error(), "foo") {
		t.Errorf("Expected error to contain package name, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "1.0.0") {
		t.Errorf("Expected error to contain version, got: %s", err.Error())
	}

	if err.Unwrap() != innerErr {
		t.Error("Unwrap should return inner error")
	}
}

func TestSolverIncompatibilityTracking(t *testing.T) {
	// Create a scenario that will fail
	source := &InMemorySource{}
	source.AddPackage(MakeName("foo"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("bar"), EqualsCondition{Version: SimpleVersion("2.0.0")}),
	})
	source.AddPackage(MakeName("bar"), SimpleVersion("1.0.0"), nil)
	// Note: bar 2.0.0 doesn't exist, creating a conflict

	root := NewRootSource()
	root.AddPackage(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source).EnableIncompatibilityTracking()
	_, err := solver.Solve(root.Term())

	if err == nil {
		t.Fatal("Expected solving to fail")
	}

	// Check if we got a NoSolutionError
	if noSolErr, ok := err.(*NoSolutionError); ok {
		errMsg := noSolErr.Error()
		t.Logf("Error message:\n%s", errMsg)

		// The error should mention the packages involved
		if !strings.Contains(errMsg, "bar") {
			t.Errorf("Error should mention bar, got: %s", errMsg)
		}
	} else {
		t.Logf("Got ErrNoSolutionFound (expected with tracking enabled): %v", err)
		t.Error("Expected NoSolutionError when tracking is enabled")
	}

	// Check incompatibilities were tracked
	incomps := solver.GetIncompatibilities()
	if len(incomps) == 0 {
		t.Error("Expected incompatibilities to be tracked")
	}

	solver.ClearIncompatibilities()
	if len(solver.GetIncompatibilities()) != 0 {
		t.Error("Expected incompatibilities to be cleared")
	}
}

func TestComplexConflictScenario(t *testing.T) {
	// Setup: A depends on B 1.0.0, C depends on B 2.0.0, root depends on A and C
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
		t.Fatal("Expected solving to fail due to conflict")
	}

	errMsg := err.Error()
	t.Logf("Error message:\n%s", errMsg)

	// The error should be informative
	if strings.Contains(errMsg, "no solution found for") {
		// Old error format
		t.Log("Got simple error format (tracking may not have captured full context)")
	} else {
		// New error format should be more detailed
		if !strings.Contains(errMsg, "B") {
			t.Errorf("Expected error to mention B (the conflicting package), got: %s", errMsg)
		}
	}
}

func TestReporterInterfaces(t *testing.T) {
	// Verify both reporters implement the Reporter interface
	var _ Reporter = (*DefaultReporter)(nil)
	var _ Reporter = (*CollapsedReporter)(nil)
}

func TestSolverWithoutTracking(t *testing.T) {
	// Verify backward compatibility - solver works without tracking
	source := &InMemorySource{}
	source.AddPackage(MakeName("foo"), SimpleVersion("1.0.0"), nil)

	root := NewRootSource()
	root.AddPackage(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source) // No tracking enabled
	solution, err := solver.Solve(root.Term())

	if err != nil {
		t.Fatalf("Expected successful solve, got: %v", err)
	}

	if len(solution) == 0 {
		t.Error("Expected non-empty solution")
	}

	// No incompatibilities should be tracked
	if len(solver.GetIncompatibilities()) != 0 {
		t.Error("Expected no incompatibilities without tracking")
	}
}

func TestErrorMethods(t *testing.T) {
	t.Parallel()

	t.Run("ErrNoSolutionFound.Error()", func(t *testing.T) {
		term := NewTerm(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})
		err := ErrNoSolutionFound{Term: term}
		msg := err.Error()
		if !strings.Contains(msg, "no solution found") {
			t.Errorf("expected 'no solution found' in error, got %q", msg)
		}
	})

	t.Run("PackageNotFoundError.Error()", func(t *testing.T) {
		err := PackageNotFoundError{
			Package: MakeName("foo"),
		}
		msg := err.Error()
		if !strings.Contains(msg, "not found") {
			t.Errorf("expected 'not found' in error, got %q", msg)
		}
	})

	t.Run("PackageVersionNotFoundError.Error()", func(t *testing.T) {
		err := PackageVersionNotFoundError{
			Package: MakeName("foo"),
			Version: SimpleVersion("1.0.0"),
		}
		msg := err.Error()
		if !strings.Contains(msg, "not found") {
			t.Errorf("expected 'not found' in error, got %q", msg)
		}
	})

	t.Run("NoSolutionError.Unwrap()", func(t *testing.T) {
		term := NewTerm(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})
		incomp := NewIncompatibilityNoVersions(term)
		nsErr := NewNoSolutionError(incomp)

		// Test Unwrap returns nil (no wrapped error)
		if unwrapped := nsErr.Unwrap(); unwrapped != nil {
			t.Errorf("expected nil from Unwrap, got %v", unwrapped)
		}
	})
}

func TestNewSemanticVersionWithPrerelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		major, minor, patch int
		prerelease          string
	}{
		{1, 2, 3, "alpha"},
		{1, 2, 3, "alpha.1"},
		{1, 2, 3, "beta.2"},
		{1, 2, 3, ""},
	}

	for _, tt := range tests {
		v := NewSemanticVersionWithPrerelease(tt.major, tt.minor, tt.patch, tt.prerelease)
		if v == nil {
			t.Errorf("NewSemanticVersionWithPrerelease(%d, %d, %d, %q) returned nil",
				tt.major, tt.minor, tt.patch, tt.prerelease)
			continue
		}

		if v.Major != tt.major || v.Minor != tt.minor || v.Patch != tt.patch {
			t.Errorf("expected %d.%d.%d, got %d.%d.%d",
				tt.major, tt.minor, tt.patch, v.Major, v.Minor, v.Patch)
		}
		if v.Prerelease != tt.prerelease {
			t.Errorf("expected prerelease %q, got %q", tt.prerelease, v.Prerelease)
		}
	}
}

func TestDisableIncompatibilityTracking(t *testing.T) {
	t.Parallel()

	source := &InMemorySource{}
	source.AddPackage(MakeName("foo"), SimpleVersion("1.0.0"), nil)

	root := NewRootSource()
	root.AddPackage(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source)
	solver.EnableIncompatibilityTracking()
	solver.DisableIncompatibilityTracking()

	// Should work normally even after disabling
	solution, err := solver.Solve(root.Term())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(solution) == 0 {
		t.Fatal("expected non-empty solution")
	}
}

func TestDefaultReporter_Nil(t *testing.T) {
	t.Parallel()

	reporter := &DefaultReporter{}
	msg := reporter.Report(nil)
	if msg != "no solution found" {
		t.Errorf("expected 'no solution found', got %q", msg)
	}
}

func TestCollapsedReporter_Nil(t *testing.T) {
	t.Parallel()

	reporter := &CollapsedReporter{}
	msg := reporter.Report(nil)
	if msg != "no solution found" {
		t.Errorf("expected 'no solution found', got %q", msg)
	}
}

func TestConflictWithSingleTerm(t *testing.T) {
	t.Parallel()

	term1 := NewTerm(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})
	term2 := NewTerm(MakeName("bar"), EqualsCondition{Version: SimpleVersion("2.0.0")})

	cause1 := NewIncompatibilityNoVersions(term1)
	cause2 := NewIncompatibilityNoVersions(term2)

	// Conflict with single term
	conflict := NewIncompatibilityConflict([]Term{term1}, cause1, cause2)

	reporter := &DefaultReporter{}
	msg := reporter.Report(conflict)
	if !strings.Contains(msg, "is forbidden") {
		t.Errorf("expected 'is forbidden' in message, got %q", msg)
	}

	reporter2 := &CollapsedReporter{}
	msg2 := reporter2.Report(conflict)
	if !strings.Contains(msg2, "is forbidden") {
		t.Errorf("expected 'is forbidden' in collapsed message, got %q", msg2)
	}
}
