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

// Example demonstrating error reporting with derivation tree
func ExampleNoSolutionError_defaultReporter() {
	// Create a conflict scenario:
	// Package A v1.0 depends on B v1.0
	// Package C v1.0 depends on B v2.0
	// Root depends on both A and C
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

	// Enable incompatibility tracking for detailed errors
	solver := NewSolver(root, source).EnableIncompatibilityTracking()
	_, err := solver.Solve(root.Term())

	if err != nil {
		fmt.Println("Error:")
		fmt.Println(err.Error())
	}

	// Output:
	// Error:
	// Because:
	//   Because:
	//     Because:
	//       Because C 1.0.0 depends on B == 2.0.0
	//     and:
	//       Because A 1.0.0 depends on B == 1.0.0
	//     these constraints conflict: C == 1.0.0 and A == 1.0.0
	//   and:
	//     Because $$root 1 depends on C == 1.0.0
	//   these constraints conflict: A == 1.0.0 and $$root == 1
	// and:
	//   Because $$root 1 depends on A == 1.0.0
	// $$root == 1 is forbidden.
}

// Example demonstrating error reporting with collapsed reporter
func ExampleNoSolutionError_collapsedReporter() {
	// Same scenario but with collapsed reporter
	source := &InMemorySource{}
	source.AddPackage(MakeName("dropdown"), SimpleVersion("2.0.0"), []Term{
		NewTerm(MakeName("icons"), EqualsCondition{Version: SimpleVersion("2.0.0")}),
	})
	source.AddPackage(MakeName("icons"), SimpleVersion("1.0.0"), nil)
	// Note: icons 2.0.0 doesn't exist

	root := NewRootSource()
	root.AddPackage(MakeName("dropdown"), EqualsCondition{Version: SimpleVersion("2.0.0")})

	solver := NewSolver(root, source).EnableIncompatibilityTracking()
	_, err := solver.Solve(root.Term())

	if nsErr, ok := err.(*NoSolutionError); ok {
		// Use collapsed reporter for more compact output
		customErr := nsErr.WithReporter(&CollapsedReporter{})
		fmt.Println("Error:")
		fmt.Println(customErr.Error())
	}

	// Output:
	// Error:
	// no versions of icons == 2.0.0 satisfy the constraint
	// And because dropdown 2.0.0 depends on icons == 2.0.0
	// And because dropdown == 2.0.0 is forbidden
	// And because $$root 1 depends on dropdown == 2.0.0
	// And because $$root == 1 is forbidden
}

// Example demonstrating incompatibility tracking
func ExampleSolver_GetIncompatibilities() {
	source := &InMemorySource{}
	source.AddPackage(MakeName("foo"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("bar"), EqualsCondition{Version: SimpleVersion("2.0.0")}),
	})
	source.AddPackage(MakeName("bar"), SimpleVersion("1.0.0"), nil)

	root := NewRootSource()
	root.AddPackage(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	solver := NewSolver(root, source).EnableIncompatibilityTracking()
	_, err := solver.Solve(root.Term())

	if err != nil {
		fmt.Printf("Solving failed: %v\n", err)

		// Get all tracked incompatibilities
		incomps := solver.GetIncompatibilities()
		fmt.Printf("Tracked %d incompatibilities during solving\n", len(incomps))

		for i, incomp := range incomps {
			fmt.Printf("  [%d] %s (kind: %d)\n", i+1, incomp.String(), incomp.Kind)
		}
	}

	// Output:
	// Solving failed: Because:
	//   Because:
	//     No versions of bar == 2.0.0 satisfy the constraint
	//   and:
	//     Because foo 1.0.0 depends on bar == 2.0.0
	//   foo == 1.0.0 is forbidden.
	// and:
	//   Because $$root 1 depends on foo == 1.0.0
	// $$root == 1 is forbidden.
	// Tracked 4 incompatibilities during solving
	//   [1] $$root 1 depends on foo == 1.0.0 (kind: 1)
	//   [2] foo 1.0.0 depends on bar == 2.0.0 (kind: 1)
	//   [3] foo == 1.0.0 is forbidden (kind: 2)
	//   [4] foo == 1.0.0 is forbidden (kind: 2)
}

// Example showing backward compatibility without tracking
func ExampleSolver_withoutTracking() {
	source := &InMemorySource{}
	source.AddPackage(MakeName("foo"), SimpleVersion("1.0.0"), []Term{
		NewTerm(MakeName("bar"), EqualsCondition{Version: SimpleVersion("2.0.0")}),
	})
	source.AddPackage(MakeName("bar"), SimpleVersion("1.0.0"), nil)

	root := NewRootSource()
	root.AddPackage(MakeName("foo"), EqualsCondition{Version: SimpleVersion("1.0.0")})

	// Without tracking, get simple error (backward compatible)
	solver := NewSolver(root, source) // tracking disabled by default
	_, err := solver.Solve(root.Term())

	if err != nil {
		// Will be ErrNoSolutionFound, not NoSolutionError
		fmt.Printf("Error type: %T\n", err)
		// Error message will vary based on where solving fails
		if _, ok := err.(ErrNoSolutionFound); ok {
			fmt.Println("Got simple ErrNoSolutionFound (backward compatible)")
		}
	}

	// Output:
	// Error type: pubgrub.ErrNoSolutionFound
	// Got simple ErrNoSolutionFound (backward compatible)
}
