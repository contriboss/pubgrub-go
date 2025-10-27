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
)

// NoSolutionError is returned when version solving fails with detailed explanation
type NoSolutionError struct {
	// Incompatibility is the root cause of the failure
	Incompatibility *Incompatibility
	// Reporter is used to format the error message (defaults to DefaultReporter)
	Reporter Reporter
}

// Error implements the error interface
func (e *NoSolutionError) Error() string {
	if e.Incompatibility == nil {
		return "no solution found"
	}

	reporter := e.Reporter
	if reporter == nil {
		reporter = &DefaultReporter{}
	}

	return reporter.Report(e.Incompatibility)
}

// WithReporter returns a new error with a custom reporter
func (e *NoSolutionError) WithReporter(reporter Reporter) *NoSolutionError {
	return &NoSolutionError{
		Incompatibility: e.Incompatibility,
		Reporter:        reporter,
	}
}

// Unwrap returns the underlying error (for errors.Is/As compatibility)
func (e *NoSolutionError) Unwrap() error {
	return nil
}

// NewNoSolutionError creates a new NoSolutionError from an incompatibility
func NewNoSolutionError(incomp *Incompatibility) *NoSolutionError {
	return &NoSolutionError{
		Incompatibility: incomp,
		Reporter:        &DefaultReporter{},
	}
}

// VersionError represents an error related to version constraints
type VersionError struct {
	Package Name
	Message string
}

// Error implements the error interface
func (e *VersionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Package.Value(), e.Message)
	}
	return fmt.Sprintf("version error for package %s", e.Package.Value())
}

// DependencyError represents an error while fetching dependencies
type DependencyError struct {
	Package Name
	Version Version
	Err     error
}

// Error implements the error interface
func (e *DependencyError) Error() string {
	return fmt.Sprintf("failed to get dependencies for %s %s: %v", e.Package.Value(), e.Version, e.Err)
}

// Unwrap returns the underlying error
func (e *DependencyError) Unwrap() error {
	return e.Err
}

// PackageNotFoundError indicates that a package is absent from the source.
type PackageNotFoundError struct {
	Package Name
}

// Error implements the error interface.
func (e *PackageNotFoundError) Error() string {
	return fmt.Sprintf("package %s not found", e.Package.Value())
}

// PackageVersionNotFoundError indicates a specific version is unavailable.
type PackageVersionNotFoundError struct {
	Package Name
	Version Version
}

// Error implements the error interface.
func (e *PackageVersionNotFoundError) Error() string {
	return fmt.Sprintf("package %s version %s not found", e.Package.Value(), e.Version)
}

// ErrNoSolutionFound is a simple error returned when solving fails
// without incompatibility tracking. For detailed error messages with
// derivation trees, enable WithIncompatibilityTracking and use NoSolutionError.
//
// Example:
//
//	solver := NewSolver(root, source) // Tracking disabled by default
//	_, err := solver.Solve(root.Term())
//	if err != nil {
//	    if errors.Is(err, ErrNoSolutionFound{}) {
//	        // Handle no solution case
//	    }
//	}
type ErrNoSolutionFound struct {
	Term Term
}

// Error implements the error interface.
func (e ErrNoSolutionFound) Error() string {
	return fmt.Sprintf("no solution found for %s", e.Term)
}

// ErrIterationLimit is returned when the solver exceeds its maximum iteration count.
// This prevents infinite loops in pathological cases. Configure with WithMaxSteps(0)
// to disable the limit (not recommended for untrusted inputs).
//
// Example:
//
//	solver := NewSolverWithOptions(
//	    []Source{root, source},
//	    WithMaxSteps(1000),
//	)
//	_, err := solver.Solve(root.Term())
//	if iterErr, ok := err.(ErrIterationLimit); ok {
//	    log.Printf("Solver exceeded %d steps", iterErr.Steps)
//	}
type ErrIterationLimit struct {
	Steps int
}

// Error implements the error interface.
func (e ErrIterationLimit) Error() string {
	if e.Steps <= 0 {
		return "solver exceeded iteration limit"
	}
	return fmt.Sprintf("solver exceeded iteration limit after %d steps", e.Steps)
}

var (
	_ error = (*NoSolutionError)(nil)
	_ error = (*VersionError)(nil)
	_ error = (*DependencyError)(nil)
	_ error = (*PackageNotFoundError)(nil)
	_ error = (*PackageVersionNotFoundError)(nil)
	_ error = ErrNoSolutionFound{}
	_ error = ErrIterationLimit{}
)
