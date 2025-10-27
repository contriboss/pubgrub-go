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

import "strings"

// SimpleVersion provides a basic string-based version implementation.
// Versions are compared lexicographically using string comparison.
//
// For semantic versioning support, use SemanticVersion instead.
//
// Example:
//
//	v1 := SimpleVersion("1.0.0")
//	v2 := SimpleVersion("2.0.0")
//	fmt.Println(v1.Sort(v2)) // prints negative number (v1 < v2)
type SimpleVersion string

// Sort implements Version by performing lexicographic string comparison.
// Returns:
//   - negative if v < other
//   - zero if v == other
//   - positive if v > other
func (v SimpleVersion) Sort(other Version) int {
	return strings.Compare(string(v), other.String())
}

// String returns the string representation of the version.
func (v SimpleVersion) String() string {
	return string(v)
}

var (
	_ Version = SimpleVersion("")
)
