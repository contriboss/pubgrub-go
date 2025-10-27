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

// assignmentKind distinguishes between decision and derivation assignments.
// Decision assignments are explicit choices made by the solver (version selections).
// Derivation assignments are constraints derived from incompatibilities via unit propagation.
type assignmentKind int

const (
	assignmentDecision   assignmentKind = iota // Explicit version selection
	assignmentDerivation                       // Constraint derived from propagation
)

// assignment represents a single constraint or decision in the partial solution.
// Assignments form the building blocks of the CDCL solver's state, tracking:
//   - Explicit decisions (version selections)
//   - Derived constraints (from unit propagation)
//   - Version sets (allowed/forbidden ranges)
//   - Causality (which incompatibility caused this derivation)
//   - Decision levels (for backtracking)
type assignment struct {
	name          Name             // Package name
	term          Term             // The constraint term
	kind          assignmentKind   // Decision or derivation
	allowed       VersionSet       // Allowed version set (positive terms)
	forbidden     VersionSet       // Forbidden version set (negative terms)
	version       Version          // Selected version (for decisions)
	cause         *Incompatibility // Incompatibility that caused this (for derivations)
	decisionLevel int              // Decision level for backtracking
	index         int              // Assignment index for satisfier ordering
}

// isDecision returns true if this assignment is an explicit version selection
// rather than a derived constraint.
func (a *assignment) isDecision() bool {
	return a.kind == assignmentDecision
}
