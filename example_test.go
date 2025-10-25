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

package pubgrub_test

import (
	"fmt"

	"github.com/contriboss/pubgrub-go"
)

// ExampleVersionSetCondition demonstrates how to use version ranges with the pubgrub solver
func ExampleVersionSetCondition() {
	// Create an in-memory package source
	source := &pubgrub.InMemorySource{}

	// Add package versions with semantic versioning
	v100, _ := pubgrub.ParseSemanticVersion("1.0.0")
	v110, _ := pubgrub.ParseSemanticVersion("1.1.0")
	v200, _ := pubgrub.ParseSemanticVersion("2.0.0")
	v210, _ := pubgrub.ParseSemanticVersion("2.1.0")

	// Create version range conditions
	range1x, _ := pubgrub.ParseVersionRange(">=1.0.0, <2.0.0")
	range2x, _ := pubgrub.ParseVersionRange(">=2.0.0")

	// Package A has multiple versions
	source.AddPackage(pubgrub.MakeName("A"), v100, []pubgrub.Term{})
	source.AddPackage(pubgrub.MakeName("A"), v110, []pubgrub.Term{
		pubgrub.NewTerm(pubgrub.MakeName("B"), pubgrub.NewVersionSetCondition(range2x)),
	})

	// Package B has multiple versions
	source.AddPackage(pubgrub.MakeName("B"), v200, []pubgrub.Term{})
	source.AddPackage(pubgrub.MakeName("B"), v210, []pubgrub.Term{})

	// Create a root source that requires package A with version range
	root := pubgrub.NewRootSource()
	root.AddPackage(pubgrub.MakeName("A"), pubgrub.NewVersionSetCondition(range1x))

	// Create a solver and solve
	solver := pubgrub.NewSolver(root, source)
	solution, err := solver.Solve(root.Term())

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print the solution (sorted by name for consistency)
	for _, nv := range solution {
		if nv.Name != pubgrub.MakeName("$$root") {
			fmt.Printf("%s = %s\n", nv.Name.Value(), nv.Version)
		}
	}
	// Output:
	// A = 1.1.0
	// B = 2.1.0
}

// ExampleParseVersionRange demonstrates parsing various version range formats
func ExampleParseVersionRange() {
	// Simple range
	range1, _ := pubgrub.ParseVersionRange(">=1.0.0")
	fmt.Println("Range 1:", range1.String())

	// Compound range (AND)
	range2, _ := pubgrub.ParseVersionRange(">=1.0.0, <2.0.0")
	fmt.Println("Range 2:", range2.String())

	// Union range (OR)
	range3, _ := pubgrub.ParseVersionRange(">=1.0.0, <2.0.0 || >=3.0.0")
	fmt.Println("Range 3:", range3.String())

	// Test if a version is in the range
	v150, _ := pubgrub.ParseSemanticVersion("1.5.0")
	fmt.Println("1.5.0 in range2:", range2.Contains(v150))

	// Output:
	// Range 1: >=1.0.0
	// Range 2: >=1.0.0, <2.0.0
	// Range 3: >=1.0.0, <2.0.0 || >=3.0.0
	// 1.5.0 in range2: true
}

// ExampleSemanticVersion demonstrates semantic version parsing and comparison
func ExampleSemanticVersion() {
	// Parse semantic versions
	v1, _ := pubgrub.ParseSemanticVersion("1.2.3")
	v2, _ := pubgrub.ParseSemanticVersion("1.2.4")
	v3, _ := pubgrub.ParseSemanticVersion("2.0.0-alpha")

	// Compare versions
	fmt.Println("v1 < v2:", v1.Sort(v2) < 0)
	fmt.Println("v2 > v1:", v2.Sort(v1) > 0)
	fmt.Println("v3 (prerelease) < 2.0.0:", v3.Sort(pubgrub.NewSemanticVersion(2, 0, 0)) < 0)

	// Output:
	// v1 < v2: true
	// v2 > v1: true
	// v3 (prerelease) < 2.0.0: true
}
