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

// RootSource provides a special source for initial dependency requirements.
// It creates a virtual "$$root" package that the solver uses as the starting
// point for dependency resolution.
//
// In PubGrub, the root package has a single version ("1") whose dependencies
// are the user's initial requirements. This design allows the solver to treat
// the root requirements uniformly with other package dependencies.
//
// Example:
//
//	root := NewRootSource()
//	root.AddPackage("lodash", EqualsCondition{Version: SimpleVersion("1.0.0")})
//	root.AddPackage("moment", EqualsCondition{Version: SimpleVersion("2.0.0")})
//	solver := NewSolver(root, otherSources...)
//	solution, _ := solver.Solve(root.Term())
type RootSource []Term

// GetVersions returns a single version for the root package only.
func (s RootSource) GetVersions(name Name) ([]Version, error) {
	rootName := MakeName("$$root")
	if name != rootName {
		return nil, &PackageNotFoundError{Package: name}
	}

	return []Version{SimpleVersion("1")}, nil
}

// GetDependencies returns the user's initial requirements for the root package.
func (s RootSource) GetDependencies(name Name, version Version) ([]Term, error) {
	rootName := MakeName("$$root")
	if name != rootName {
		return nil, &PackageNotFoundError{Package: name}
	}

	if version != SimpleVersion("1") {
		return nil, &PackageVersionNotFoundError{Package: name, Version: version}
	}

	return s, nil
}

// AddPackage adds a single requirement to the root source.
// Each requirement becomes a dependency of the virtual root package.
func (s *RootSource) AddPackage(name Name, condition Condition) {
	*s = append(*s, NewTerm(name, condition))
}

// Term returns the term representing the root package itself.
// This is the starting term passed to Solver.Solve().
func (s *RootSource) Term() Term {
	return NewTerm(MakeName("$$root"), EqualsCondition{SimpleVersion("1")})
}

// NewRootSource creates a new empty root source.
func NewRootSource() *RootSource {
	return &RootSource{}
}

var (
	_ Source = &RootSource{}
)
