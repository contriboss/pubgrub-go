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
//
// Originally from: github.com/tinyrange/tinyrange/experimental/pubgrub (v0.2.6)
// This is a derivative work based on the tinyrange pubgrub implementation.

// This is not a full implemention of Pubgrub yet. This is a simple solver that uses a depth first search to find a solution.
// The solver is not complete and does not handle all cases. It is a starting point for a full implementation.

package pubgrub

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"unique"
)

type ErrNoSolutionFound struct {
	Term Term
}

func (e ErrNoSolutionFound) Error() string {
	return fmt.Sprintf("no solution found for %s", e.Term)
}

var (
	_ error = ErrNoSolutionFound{}
)

// Name represents a package name using value interning for memory efficiency.
// Multiple instances of the same package name share the same underlying memory.
type Name = unique.Handle[string]

// MakeName creates an interned Name from a string.
// Equal strings will return the same Name value, enabling fast comparisons.
func MakeName(s string) Name {
	return unique.Make(s)
}

// EmptyName returns an empty name (interned empty string).
func EmptyName() Name {
	return unique.Make("")
}

type Version interface {
	String() string
	Sort(other Version) int
}

type Condition interface {
	String() string
	Satisfies(ver Version) bool
}

type EqualsCondition struct {
	Version Version
}

// String satisfies Condition.
func (c EqualsCondition) String() string {
	return fmt.Sprintf("== %s", c.Version)
}

// Satisfies satisfies Condition.
func (c EqualsCondition) Satisfies(ver Version) bool {
	return c.Version.String() == ver.String()
}

var (
	_ Condition = EqualsCondition{}
)

type Term struct {
	Name      Name
	Condition Condition
	Positive  bool
}

func (t Term) String() string {
	cond := "*"
	if t.Condition != nil {
		cond = t.Condition.String()
	}

	if t.Positive {
		if cond == "*" {
			return t.Name.Value()
		}
		return fmt.Sprintf("%s %s", t.Name.Value(), cond)
	}

	if cond == "*" {
		return fmt.Sprintf("not %s", t.Name.Value())
	}
	return fmt.Sprintf("not %s %s", t.Name.Value(), cond)
}

func NewTerm(name Name, condition Condition) Term {
	return Term{Name: name, Condition: condition, Positive: true}
}

func NewNegativeTerm(name Name, condition Condition) Term {
	return Term{Name: name, Condition: condition, Positive: false}
}

// Negate returns the logical negation of the term.
func (t Term) Negate() Term {
	return Term{
		Name:      t.Name,
		Condition: t.Condition,
		Positive:  !t.Positive,
	}
}

// IsPositive reports whether the term asserts a positive constraint.
func (t Term) IsPositive() bool {
	return t.Positive
}

// SatisfiedBy reports whether the provided version satisfies the term.
// A nil version indicates the package is not selected.
func (t Term) SatisfiedBy(ver Version) bool {
	if ver == nil {
		return !t.Positive
	}

	if t.Condition == nil {
		return t.Positive
	}

	satisfied := t.Condition.Satisfies(ver)
	if t.Positive {
		return satisfied
	}
	return !satisfied
}

type NameVersion struct {
	Name    Name
	Version Version
}

func (n NameVersion) String() string {
	return fmt.Sprintf("%s %s", n.Name.Value(), n.Version)
}

type Source interface {
	// GetVersions returns all versions of a package in a sorted order.
	GetVersions(name Name) ([]Version, error)
	GetDependencies(name Name, version Version) ([]Term, error)
}

type SimpleVersion string

// Sort implements Version.
func (v SimpleVersion) Sort(other Version) int {
	return strings.Compare(string(v), other.String())
}

// String satisfies Version.
func (v SimpleVersion) String() string {
	return string(v)
}

var (
	_ Version = SimpleVersion("")
)

type InMemorySource struct {
	Packages map[Name]map[Version][]Term
}

// GetVersions satisfies Source.
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

// GetDependencies satisfies Source.
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

// AddPackage adds a package to the source with a list of dependencies.
func (s *InMemorySource) AddPackage(name Name, version Version, deps []Term) {
	if s.Packages == nil {
		s.Packages = make(map[Name]map[Version][]Term)
	}

	if _, ok := s.Packages[name]; !ok {
		s.Packages[name] = make(map[Version][]Term)
	}

	s.Packages[name][version] = deps
}

type CombinedSource []Source

// GetVersions satisfies Source.
func (s CombinedSource) GetVersions(name Name) ([]Version, error) {
	var ret []Version
	var sawNotFound bool
	for _, source := range s {
		versions, err := source.GetVersions(name)
		if err != nil {
			var pkgErr *PackageNotFoundError
			if errors.As(err, &pkgErr) {
				sawNotFound = true
				continue
			}
			return nil, err
		}
		ret = append(ret, versions...)
	}

	if len(ret) == 0 {
		if sawNotFound {
			return nil, &PackageNotFoundError{Package: name}
		}
		return nil, &PackageNotFoundError{Package: name}
	}

	// sort the versions
	slices.SortFunc(ret, func(a Version, b Version) int {
		return a.Sort(b)
	})

	return ret, nil
}

// GetDependencies satisfies Source.
func (s CombinedSource) GetDependencies(name Name, version Version) ([]Term, error) {
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
				return nil, err
			}
		} else {
			return deps, nil
		}
	}

	return nil, &PackageVersionNotFoundError{Package: name, Version: version}
}

type RootSource []Term

// GetVersions satisfies Source.
func (s RootSource) GetVersions(name Name) ([]Version, error) {
	rootName := MakeName("$$root")
	if name != rootName {
		return nil, &PackageNotFoundError{Package: name}
	}

	return []Version{SimpleVersion("1")}, nil
}

// GetDependencies satisfies Source.
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

// AddPackage adds a single term to the source.
func (s *RootSource) AddPackage(name Name, condition Condition) {
	*s = append(*s, NewTerm(name, condition))
}

func (s *RootSource) Term() Term {
	return NewTerm(MakeName("$$root"), EqualsCondition{SimpleVersion("1")})
}

func NewRootSource() *RootSource {
	return &RootSource{}
}

var (
	_ Source = &InMemorySource{}
	_ Source = CombinedSource{}
)

type Solution []NameVersion

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
