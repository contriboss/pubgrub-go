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

import "fmt"

// Term represents a dependency constraint, either positive or negative.
// A positive term (e.g., "lodash >=1.0.0") asserts that a package must satisfy
// the condition. A negative term (e.g., "not lodash ==1.5.0") excludes versions
// that match the condition.
//
// Terms are the building blocks of dependency resolution, combining package
// names with version constraints and polarity.
type Term struct {
	Name      Name
	Condition Condition
	Positive  bool
}

// String returns a human-readable representation of the term.
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

// NewTerm creates a positive term requiring the package to satisfy the condition.
func NewTerm(name Name, condition Condition) Term {
	return Term{Name: name, Condition: condition, Positive: true}
}

// NewNegativeTerm creates a negative term excluding versions matching the condition.
func NewNegativeTerm(name Name, condition Condition) Term {
	return Term{Name: name, Condition: condition, Positive: false}
}

// Negate returns the logical negation of the term.
// A positive term becomes negative and vice versa.
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
//
// For positive terms, returns true if the version matches the condition.
// For negative terms, returns true if the version does NOT match the condition.
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
