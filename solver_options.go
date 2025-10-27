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

import "log/slog"

// SolverOptions configures the behavior of the dependency solver.
//
// Options control:
//   - Incompatibility tracking for enhanced error reporting
//   - Maximum iteration limits to prevent infinite loops
//   - Debug logging for solver diagnostics
type SolverOptions struct {
	// TrackIncompatibilities enables collecting learned clauses for error reporting.
	// When enabled, NoSolutionError will include a detailed derivation tree.
	// When disabled, returns simple ErrNoSolutionFound.
	TrackIncompatibilities bool

	// MaxSteps limits the number of solver iterations.
	// Set to 0 to disable the limit (not recommended for untrusted inputs).
	// Default: 100000
	MaxSteps int

	// Logger enables debug logging of solver operations.
	// When nil, no logging is performed.
	Logger *slog.Logger
}

// SolverOption is a functional option for configuring the solver.
type SolverOption func(*SolverOptions)

const defaultMaxSteps = 100000

// defaultSolverOptions returns the default solver configuration.
func defaultSolverOptions() SolverOptions {
	return SolverOptions{
		TrackIncompatibilities: false,
		MaxSteps:               defaultMaxSteps,
	}
}

// WithIncompatibilityTracking enables or disables incompatibility tracking.
// When enabled, the solver collects learned clauses and provides detailed
// error messages with derivation trees.
//
// Example:
//
//	solver := NewSolverWithOptions(
//	    []Source{root, source},
//	    WithIncompatibilityTracking(true),
//	)
func WithIncompatibilityTracking(enabled bool) SolverOption {
	return func(opts *SolverOptions) {
		opts.TrackIncompatibilities = enabled
	}
}

// WithMaxSteps sets the maximum number of solver iterations.
// Use 0 to disable the limit (allows unbounded execution).
//
// The iteration limit prevents infinite loops in pathological cases.
// Most real-world dependency graphs resolve in thousands of steps.
//
// Example:
//
//	solver := NewSolverWithOptions(
//	    []Source{root, source},
//	    WithMaxSteps(10000), // Limit to 10k iterations
//	)
func WithMaxSteps(steps int) SolverOption {
	return func(opts *SolverOptions) {
		if steps <= 0 {
			opts.MaxSteps = 0
		} else {
			opts.MaxSteps = steps
		}
	}
}

// WithLogger sets a structured logger for solver diagnostics.
// The logger receives debug messages during solving, useful for understanding
// the solver's decision-making process.
//
// Example:
//
//	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
//	solver := NewSolverWithOptions(
//	    []Source{root, source},
//	    WithLogger(logger),
//	)
func WithLogger(logger *slog.Logger) SolverOption {
	return func(opts *SolverOptions) {
		opts.Logger = logger
	}
}
