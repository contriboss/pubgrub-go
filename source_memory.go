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

import "slices"

// InMemorySource provides an in-memory implementation of Source for testing
// and simple use cases. It stores all package versions and dependencies in
// memory without any I/O operations.
//
// This is the simplest source implementation and is useful for:
//   - Testing dependency resolution scenarios
//   - Building example dependency graphs
//   - Prototyping before implementing a real package source
//
// For production use cases with network or database access, consider wrapping
// your source with CachedSource for performance.
//
// Example:
//
//	source := &InMemorySource{}
//	source.AddPackage("lodash", SimpleVersion("1.0.0"), []Term{
//	    NewTerm("core-js", EqualsCondition{Version: SimpleVersion("2.0.0")}),
//	})
//	source.AddPackage("core-js", SimpleVersion("2.0.0"), nil)
type InMemorySource struct {
	Packages map[Name]map[Version][]Term
}

// GetVersions returns all available versions of a package in sorted order.
func (s *InMemorySource) GetVersions(name Name) ([]Version, error) {
	versions, ok := s.Packages[name]
	if !ok {
		return nil, &PackageNotFoundError{Package: name}
	}

	var result []Version
	for v := range versions {
		result = append(result, v)
	}

	// sort the versions
	slices.SortFunc(result, func(a Version, b Version) int {
		return a.Sort(b)
	})

	return result, nil
}

// GetDependencies returns the dependency terms for a specific package version.
func (s *InMemorySource) GetDependencies(name Name, version Version) ([]Term, error) {
	versions, ok := s.Packages[name]
	if !ok {
		return nil, &PackageNotFoundError{Package: name}
	}

	if _, ok := versions[version]; !ok {
		return nil, &PackageVersionNotFoundError{Package: name, Version: version}
	}

	return s.Packages[name][version], nil
}

// AddPackage adds a package version with its dependencies to the source.
// If the package map is nil, it will be initialized automatically.
func (s *InMemorySource) AddPackage(name Name, version Version, deps []Term) {
	if s.Packages == nil {
		s.Packages = make(map[Name]map[Version][]Term)
	}

	if _, ok := s.Packages[name]; !ok {
		s.Packages[name] = make(map[Version][]Term)
	}

	s.Packages[name][version] = deps
}

var (
	_ Source = &InMemorySource{}
)
