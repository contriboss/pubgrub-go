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

import "unique"

// Name represents a package name using value interning for memory efficiency.
// Multiple instances of the same package name share the same underlying memory.
//
// Name uses Go's unique.Handle for efficient string interning, enabling:
//   - Fast equality comparisons (pointer comparison instead of string comparison)
//   - Reduced memory usage when the same package names appear frequently
//   - Safe concurrent access (interning is thread-safe)
type Name = unique.Handle[string]

// MakeName creates an interned Name from a string.
// Equal strings will return the same Name value, enabling fast comparisons.
//
// Example:
//
//	pkg1 := MakeName("lodash")
//	pkg2 := MakeName("lodash")
//	// pkg1 == pkg2 (fast pointer comparison)
func MakeName(s string) Name {
	return unique.Make(s)
}

// EmptyName returns an empty name (interned empty string).
// Useful for creating placeholder or root package names.
func EmptyName() Name {
	return unique.Make("")
}
