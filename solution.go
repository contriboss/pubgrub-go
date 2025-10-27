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
	"iter"
)

// NameVersion represents a resolved package with its selected version.
// This is the fundamental unit of a dependency resolution solution.
type NameVersion struct {
	Name    Name
	Version Version
}

// String returns a human-readable representation of the package-version pair.
func (n NameVersion) String() string {
	return fmt.Sprintf("%s %s", n.Name.Value(), n.Version)
}

// Solution represents the complete set of resolved package versions.
// A solution maps package names to their selected versions, ensuring
// all dependency constraints are satisfied.
//
// Example:
//
//	solution, err := solver.Solve(root.Term())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for pkg := range solution.All() {
//	    fmt.Printf("%s: %s\n", pkg.Name.Value(), pkg.Version)
//	}
type Solution []NameVersion

// GetVersion retrieves the resolved version for a given package name.
// Returns the version and true if found, or nil and false if the package
// is not in the solution.
func (s Solution) GetVersion(name Name) (Version, bool) {
	for _, nv := range s {
		if nv.Name == name {
			return nv.Version, true
		}
	}

	return nil, false
}

// All returns an iterator over all package-version pairs in the solution.
// This enables using range-over-function syntax:
//
//	for pkg := range solution.All() {
//	    fmt.Printf("%s: %s\n", pkg.Name, pkg.Version)
//	}
func (s Solution) All() iter.Seq[NameVersion] {
	return func(yield func(NameVersion) bool) {
		for _, nv := range s {
			if !yield(nv) {
				return
			}
		}
	}
}
