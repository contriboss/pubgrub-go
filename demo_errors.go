//go:build ignore

package main

import (
	"fmt"
	"github.com/contriboss/pubgrub-go"
)

func main() {
	fmt.Println("=== PubGrub Error Reporting Demo ===\n")

	// Scenario 1: No versions available
	fmt.Println("Scenario 1: Package not found")
	fmt.Println("--------------------------------")
	source1 := &pubgrub.InMemorySource{}
	source1.AddPackage(MakeName("app"), pubgrub.SimpleVersion("1.0.0"), []pubgrub.Term{
		pubgrub.NewTerm(MakeName("missing-dep"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("1.0.0")}),
	})

	root1 := pubgrub.NewRootSource()
	root1.AddPackage(MakeName("app"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("1.0.0")})

	solver1 := pubgrub.NewSolver(root1, source1).EnableIncompatibilityTracking()
	_, err1 := solver1.Solve(root1.Term())
	if err1 != nil {
		fmt.Printf("Error:\n%s\n\n", err1)
	}

	// Scenario 2: Version conflict
	fmt.Println("Scenario 2: Version conflict")
	fmt.Println("----------------------------")
	source2 := &pubgrub.InMemorySource{}
	source2.AddPackage(MakeName("dropdown"), pubgrub.SimpleVersion("2.0.0"), []pubgrub.Term{
		pubgrub.NewTerm(MakeName("icons"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("2.0.0")}),
	})
	source2.AddPackage(MakeName("menu"), pubgrub.SimpleVersion("1.1.0"), []pubgrub.Term{
		pubgrub.NewTerm(MakeName("dropdown"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("2.0.0")}),
	})
	source2.AddPackage(MakeName("icons"), pubgrub.SimpleVersion("1.0.0"), nil)

	root2 := pubgrub.NewRootSource()
	root2.AddPackage(MakeName("menu"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("1.1.0")})
	root2.AddPackage(MakeName("icons"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("1.0.0")})

	solver2 := pubgrub.NewSolver(root2, source2).EnableIncompatibilityTracking()
	_, err2 := solver2.Solve(root2.Term())
	if err2 != nil {
		fmt.Printf("Error with DefaultReporter:\n%s\n\n", err2)

		// Try with CollapsedReporter
		if nsErr, ok := err2.(*pubgrub.NoSolutionError); ok {
			collapsedErr := nsErr.WithReporter(&pubgrub.CollapsedReporter{})
			fmt.Printf("Same error with CollapsedReporter:\n%s\n\n", collapsedErr)
		}
	}

	// Scenario 3: Successful resolution
	fmt.Println("Scenario 3: Successful resolution")
	fmt.Println("----------------------------------")
	source3 := &pubgrub.InMemorySource{}
	source3.AddPackage(MakeName("web"), pubgrub.SimpleVersion("1.0.0"), []pubgrub.Term{
		pubgrub.NewTerm(MakeName("http"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("2.0.0")}),
	})
	source3.AddPackage(MakeName("http"), pubgrub.SimpleVersion("2.0.0"), []pubgrub.Term{
		pubgrub.NewTerm(MakeName("json"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("1.5.0")}),
	})
	source3.AddPackage(MakeName("json"), pubgrub.SimpleVersion("1.5.0"), nil)

	root3 := pubgrub.NewRootSource()
	root3.AddPackage(MakeName("web"), pubgrub.EqualsCondition{Version: pubgrub.SimpleVersion("1.0.0")})

	solver3 := pubgrub.NewSolver(root3, source3).EnableIncompatibilityTracking()
	solution, err3 := solver3.Solve(root3.Term())
	if err3 == nil {
		fmt.Println("Solution found:")
		for _, nv := range solution {
			if nv.Name != pubgrub.MakeName("$$root") {
				fmt.Printf("  - %s %s\n", nv.Name.Value(), nv.Version)
			}
		}
	} else {
		fmt.Printf("Error: %s\n", err3)
	}
}
