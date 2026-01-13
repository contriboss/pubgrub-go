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
	"errors"
	"slices"
)

// CombinedSource aggregates multiple package sources into a single source.
// When querying for versions or dependencies, it tries each source in order
// and combines the results.
//
// This is useful for:
//   - Combining local and remote package sources
//   - Implementing package source fallbacks
//   - Testing with mixed source types
//
// Example:
//
//	local := &InMemorySource{}
//	remote := &RegistrySource{}
//	combined := CombinedSource{local, remote}
//	solver := NewSolver(root, combined)
type CombinedSource []Source

// GetVersions queries all sources and returns the combined set of versions
// in sorted order. Returns an error only if all sources fail with non-NotFound errors.
func (s CombinedSource) GetVersions(name Name) ([]Version, error) {
	unique := make(map[string]Version)
	var firstErr error
	var sawNotFound bool
	for _, source := range s {
		versions, err := source.GetVersions(name)
		if err != nil {
			var pkgErr *PackageNotFoundError
			if errors.As(err, &pkgErr) {
				sawNotFound = true
				continue
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, ver := range versions {
			key := ver.String()
			if _, ok := unique[key]; !ok {
				unique[key] = ver
			}
		}
	}

	if len(unique) > 0 {
		ret := make([]Version, 0, len(unique))
		for _, ver := range unique {
			ret = append(ret, ver)
		}

		// sort the versions
		slices.SortFunc(ret, func(a Version, b Version) int {
			return a.Sort(b)
		})

		return ret, nil
	}

	if firstErr != nil {
		return nil, firstErr
	}

	if sawNotFound {
		return nil, &PackageNotFoundError{Package: name}
	}

	return nil, &PackageNotFoundError{Package: name}
}

// GetDependencies queries sources in order and returns dependencies from the
// first source that has the specified package version.
func (s CombinedSource) GetDependencies(name Name, version Version) ([]Term, error) {
	var lastErr error
	for _, source := range s {
		deps, err := source.GetDependencies(name, version)
		if err != nil {
			var pkgErr *PackageNotFoundError
			var verErr *PackageVersionNotFoundError
			switch {
			case errors.As(err, &pkgErr):
				continue
			case errors.As(err, &verErr):
				continue
			default:
				if lastErr == nil {
					lastErr = err
				}
				continue
			}
		} else {
			return deps, nil
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, &PackageVersionNotFoundError{Package: name, Version: version}
}

var (
	_ Source = CombinedSource{}
)
