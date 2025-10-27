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

// EqualsCondition represents an exact version match constraint.
// This is the most basic condition type, requiring a version to exactly
// match the specified version.
//
// For more flexible constraints like version ranges, use VersionSetCondition
// with ParseVersionRange.
//
// Example:
//
//	cond := EqualsCondition{Version: SimpleVersion("1.0.0")}
//	fmt.Println(cond.Satisfies(SimpleVersion("1.0.0"))) // true
//	fmt.Println(cond.Satisfies(SimpleVersion("1.0.1"))) // false
type EqualsCondition struct {
	Version Version
}

// String returns a human-readable representation of the condition.
func (c EqualsCondition) String() string {
	return fmt.Sprintf("== %s", c.Version)
}

// Satisfies returns true if the given version exactly matches the constraint.
func (c EqualsCondition) Satisfies(ver Version) bool {
	return c.Version.String() == ver.String()
}

var (
	_ Condition = EqualsCondition{}
)
