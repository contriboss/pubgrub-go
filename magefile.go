//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/sh"
)

// CI runs all validation checks (tests, linting, coverage, benchmarks)
func CI() error {
	fmt.Println("Running full CI validation...")

	if err := Test(); err != nil {
		return err
	}

	if err := Lint(); err != nil {
		return err
	}

	if err := Benchmark(); err != nil {
		return err
	}

	fmt.Println("All CI checks passed!")
	return nil
}

// Test runs all tests with race detection and coverage
func Test() error {
	fmt.Println("Running tests with race detection...")
	if err := sh.RunV("go", "test", "-v", "-race", "-coverprofile=coverage.out", "./..."); err != nil {
		return err
	}

	fmt.Println("Generating coverage report...")
	return sh.RunV("go", "tool", "cover", "-func=coverage.out")
}

// Lint runs golangci-lint
func Lint() error {
	fmt.Println("Running linter...")
	return sh.RunV("golangci-lint", "run", "--timeout=5m")
}

// Benchmark runs all benchmarks
func Benchmark() error {
	fmt.Println("Running benchmarks...")
	return sh.RunV("go", "test", "-bench=.", "-benchmem")
}

// Clean removes build artifacts and coverage files
func Clean() error {
	fmt.Println("Cleaning build artifacts...")
	os.Remove("coverage.out")
	return nil
}

// Fmt formats all Go code
func Fmt() error {
	fmt.Println("Formatting code...")
	return sh.RunV("go", "fmt", "./...")
}

// Vet runs go vet
func Vet() error {
	fmt.Println("Running go vet...")
	return sh.RunV("go", "vet", "./...")
}

// ModTidy tidies go.mod
func ModTidy() error {
	fmt.Println("Tidying go.mod...")
	return sh.RunV("go", "mod", "tidy")
}
