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

package pubgrub_test

import (
	"testing"

	"github.com/contriboss/pubgrub-go"
)

func TestSemanticVersionParsing(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"1.2.3", false},
		{"1.2.3-alpha", false},
		{"1.2.3-alpha.1", false},
		{"1.2.3+build.123", false},
		{"1.2.3-alpha+build", false},
		{"2.0.0", false},
		{"0.1.0", false},
		{"invalid", true},
		{"1.2.3.4", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := pubgrub.ParseSemanticVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSemanticVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSemanticVersionComparison(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0 vs 1.0.0", "1.0.0", "1.0.0", 0},
		{"1.0.0 vs 2.0.0", "1.0.0", "2.0.0", -1},
		{"2.0.0 vs 1.0.0", "2.0.0", "1.0.0", 1},
		{"1.2.3 vs 1.2.4", "1.2.3", "1.2.4", -1},
		{"1.2.4 vs 1.2.3", "1.2.4", "1.2.3", 1},
		{"1.2.0 vs 1.3.0", "1.2.0", "1.3.0", -1},
		{"1.3.0 vs 1.2.0", "1.3.0", "1.2.0", 1},
		{"1.0.0 vs 1.0.0-alpha", "1.0.0", "1.0.0-alpha", 1},
		{"1.0.0-alpha vs 1.0.0", "1.0.0-alpha", "1.0.0", -1},
		{"1.0.0-alpha vs 1.0.0-beta", "1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-beta vs 1.0.0-alpha", "1.0.0-beta", "1.0.0-alpha", 1},
		{"1.0.0-alpha.1 vs 1.0.0-alpha.2", "1.0.0-alpha.1", "1.0.0-alpha.2", -1},
		{"1.0.0-1 vs 1.0.0-2", "1.0.0-1", "1.0.0-2", -1},
		{"1.0.0-alpha vs 1.0.0-alpha.1", "1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"1.0.0+build1 vs 1.0.0+build2", "1.0.0+build1", "1.0.0+build2", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, err := pubgrub.ParseSemanticVersion(tt.v1)
			if err != nil {
				t.Fatalf("Failed to parse v1 %q: %v", tt.v1, err)
			}

			v2, err := pubgrub.ParseSemanticVersion(tt.v2)
			if err != nil {
				t.Fatalf("Failed to parse v2 %q: %v", tt.v2, err)
			}

			result := v1.Sort(v2)
			if (result < 0 && tt.expected >= 0) || (result > 0 && tt.expected <= 0) || (result == 0 && tt.expected != 0) {
				t.Errorf("Sort(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestRangeContains(t *testing.T) {
	tests := []struct {
		name     string
		rangeStr string
		version  string
		want     bool
	}{
		{">= 1.0.0 with 1.0.0", ">=1.0.0", "1.0.0", true},
		{">= 1.0.0 with 1.5.0", ">=1.0.0", "1.5.0", true},
		{">= 1.0.0 with 0.9.0", ">=1.0.0", "0.9.0", false},
		{"> 1.0.0 with 1.0.0", ">1.0.0", "1.0.0", false},
		{"> 1.0.0 with 1.0.1", ">1.0.0", "1.0.1", true},
		{"> 1.0.0 with 0.9.0", ">1.0.0", "0.9.0", false},
		{"<= 2.0.0 with 2.0.0", "<=2.0.0", "2.0.0", true},
		{"<= 2.0.0 with 1.5.0", "<=2.0.0", "1.5.0", true},
		{"<= 2.0.0 with 2.1.0", "<=2.0.0", "2.1.0", false},
		{"< 2.0.0 with 2.0.0", "<2.0.0", "2.0.0", false},
		{"< 2.0.0 with 1.9.0", "<2.0.0", "1.9.0", true},
		{"< 2.0.0 with 2.1.0", "<2.0.0", "2.1.0", false},
		{"== 1.5.0 with 1.5.0", "==1.5.0", "1.5.0", true},
		{"== 1.5.0 with 1.5.1", "==1.5.0", "1.5.1", false},
		{"!= 1.5.0 with 1.5.0", "!=1.5.0", "1.5.0", false},
		{"!= 1.5.0 with 1.5.1", "!=1.5.0", "1.5.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versionSet, err := pubgrub.ParseVersionRange(tt.rangeStr)
			if err != nil {
				t.Fatalf("Failed to parse range %q: %v", tt.rangeStr, err)
			}

			version, err := pubgrub.ParseSemanticVersion(tt.version)
			if err != nil {
				t.Fatalf("Failed to parse version %q: %v", tt.version, err)
			}

			got := versionSet.Contains(version)
			if got != tt.want {
				t.Errorf("Range %q contains %q = %v, want %v", tt.rangeStr, tt.version, got, tt.want)
			}
		})
	}
}

func TestRangeSetContains(t *testing.T) {
	tests := []struct {
		name     string
		rangeStr string
		version  string
		want     bool
	}{
		{">=1.0.0 and <2.0.0 with 1.5.0", ">=1.0.0, <2.0.0", "1.5.0", true},
		{">=1.0.0 and <2.0.0 with 0.9.0", ">=1.0.0, <2.0.0", "0.9.0", false},
		{">=1.0.0 and <2.0.0 with 2.0.0", ">=1.0.0, <2.0.0", "2.0.0", false},
		{"empty range set with any version", "", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rangeStr == "" {
				// Empty range set test
				versionSet := pubgrub.EmptyVersionSet()
				version, _ := pubgrub.ParseSemanticVersion(tt.version)
				got := versionSet.Contains(version)
				if got != tt.want {
					t.Errorf("Empty set contains %q = %v, want %v", tt.version, got, tt.want)
				}
				return
			}

			versionSet, err := pubgrub.ParseVersionRange(tt.rangeStr)
			if err != nil {
				t.Fatalf("Failed to parse range %q: %v", tt.rangeStr, err)
			}

			version, err := pubgrub.ParseSemanticVersion(tt.version)
			if err != nil {
				t.Fatalf("Failed to parse version %q: %v", tt.version, err)
			}

			got := versionSet.Contains(version)
			if got != tt.want {
				t.Errorf("Range %q contains %q = %v, want %v", tt.rangeStr, tt.version, got, tt.want)
			}
		})
	}
}

func TestUnionSetContains(t *testing.T) {
	tests := []struct {
		name     string
		rangeStr string
		version  string
		want     bool
	}{
		{">=1.0.0, <2.0.0 || >=3.0.0 with 1.5.0", ">=1.0.0, <2.0.0 || >=3.0.0", "1.5.0", true},
		{">=1.0.0, <2.0.0 || >=3.0.0 with 3.5.0", ">=1.0.0, <2.0.0 || >=3.0.0", "3.5.0", true},
		{">=1.0.0, <2.0.0 || >=3.0.0 with 2.5.0", ">=1.0.0, <2.0.0 || >=3.0.0", "2.5.0", false},
		{"empty union set", "*", "1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versionSet, err := pubgrub.ParseVersionRange(tt.rangeStr)
			if err != nil {
				t.Fatalf("Failed to parse range %q: %v", tt.rangeStr, err)
			}

			version, err := pubgrub.ParseSemanticVersion(tt.version)
			if err != nil {
				t.Fatalf("Failed to parse version %q: %v", tt.version, err)
			}

			got := versionSet.Contains(version)
			if got != tt.want {
				t.Errorf("Range %q contains %q = %v, want %v", tt.rangeStr, tt.version, got, tt.want)
			}
		})
	}
}

func TestUnionSetOperations(t *testing.T) {
	t.Run("Union", func(t *testing.T) {
		vs1, _ := pubgrub.ParseVersionRange(">=1.0.0, <2.0.0")
		vs2, _ := pubgrub.ParseVersionRange(">=1.5.0, <3.0.0")
		union := vs1.Union(vs2)

		v1, _ := pubgrub.ParseSemanticVersion("1.2.0")
		v2, _ := pubgrub.ParseSemanticVersion("2.5.0")

		if !union.Contains(v1) {
			t.Error("Union should contain 1.2.0")
		}
		if !union.Contains(v2) {
			t.Error("Union should contain 2.5.0")
		}
	})

	t.Run("Intersection", func(t *testing.T) {
		vs1, _ := pubgrub.ParseVersionRange(">=1.0.0, <3.0.0")
		vs2, _ := pubgrub.ParseVersionRange(">=2.0.0, <4.0.0")
		intersection := vs1.Intersection(vs2)

		v1, _ := pubgrub.ParseSemanticVersion("1.5.0")
		v2, _ := pubgrub.ParseSemanticVersion("2.5.0")
		v3, _ := pubgrub.ParseSemanticVersion("3.5.0")

		if intersection.Contains(v1) {
			t.Error("Intersection should not contain 1.5.0")
		}
		if !intersection.Contains(v2) {
			t.Error("Intersection should contain 2.5.0")
		}
		if intersection.Contains(v3) {
			t.Error("Intersection should not contain 3.5.0")
		}
	})

	t.Run("Complement", func(t *testing.T) {
		vs, _ := pubgrub.ParseVersionRange(">=1.0.0, <2.0.0")
		complement := vs.Complement()

		v1, _ := pubgrub.ParseSemanticVersion("0.5.0")
		v2, _ := pubgrub.ParseSemanticVersion("1.5.0")
		v3, _ := pubgrub.ParseSemanticVersion("2.5.0")

		if !complement.Contains(v1) {
			t.Error("Complement should contain 0.5.0")
		}
		if complement.Contains(v2) {
			t.Error("Complement should not contain 1.5.0")
		}
		if !complement.Contains(v3) {
			t.Error("Complement should contain 2.5.0")
		}
	})

	t.Run("IsEmpty", func(t *testing.T) {
		empty := pubgrub.EmptyVersionSet()
		if !empty.IsEmpty() {
			t.Error("EmptyVersionSet should be empty")
		}

		vs, _ := pubgrub.ParseVersionRange(">=1.0.0")
		if vs.IsEmpty() {
			t.Error("Non-empty version set should not be empty")
		}
	})
}

func TestParseVersionRange(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		testVersion string
		want        bool
	}{
		{">=1.0.0 with 1.0.0", ">=1.0.0", "1.0.0", true},
		{">=1.0.0 with 0.9.9", ">=1.0.0", "0.9.9", false},
		{">=1.0.0, <2.0.0 with 1.5.0", ">=1.0.0, <2.0.0", "1.5.0", true},
		{">=1.0.0, <2.0.0 with 2.0.0", ">=1.0.0, <2.0.0", "2.0.0", false},
		{">=1.0.0, <2.0.0 || >=3.0.0 with 1.5.0", ">=1.0.0, <2.0.0 || >=3.0.0", "1.5.0", true},
		{">=1.0.0, <2.0.0 || >=3.0.0 with 3.5.0", ">=1.0.0, <2.0.0 || >=3.0.0", "3.5.0", true},
		{">=1.0.0, <2.0.0 || >=3.0.0 with 2.5.0", ">=1.0.0, <2.0.0 || >=3.0.0", "2.5.0", false},
		{"* with 1.0.0", "*", "1.0.0", true},
		{"* with 999.0.0", "*", "999.0.0", true},
		{"==1.5.0 with 1.5.0", "==1.5.0", "1.5.0", true},
		{"==1.5.0 with 1.5.1", "==1.5.0", "1.5.1", false},
		{">1.0.0, <=2.0.0 with 1.5.0", ">1.0.0, <=2.0.0", "1.5.0", true},
		{">1.0.0, <=2.0.0 with 1.0.0", ">1.0.0, <=2.0.0", "1.0.0", false},
		{">1.0.0, <=2.0.0 with 2.0.0", ">1.0.0, <=2.0.0", "2.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versionSet, err := pubgrub.ParseVersionRange(tt.input)
			if err != nil {
				t.Fatalf("ParseVersionRange(%q) failed: %v", tt.input, err)
			}

			version, err := pubgrub.ParseSemanticVersion(tt.testVersion)
			if err != nil {
				t.Fatalf("ParseSemanticVersion(%q) failed: %v", tt.testVersion, err)
			}

			got := versionSet.Contains(version)
			if got != tt.want {
				t.Errorf("ParseVersionRange(%q).Contains(%q) = %v, want %v", tt.input, tt.testVersion, got, tt.want)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("Prerelease versions in ranges", func(t *testing.T) {
		vs, _ := pubgrub.ParseVersionRange(">=1.0.0, <2.0.0")
		v1, _ := pubgrub.ParseSemanticVersion("1.0.0-alpha")
		v2, _ := pubgrub.ParseSemanticVersion("1.5.0")

		// According to SemVer, 1.0.0-alpha < 1.0.0, so it's NOT in >=1.0.0
		if vs.Contains(v1) {
			t.Error("Should NOT contain prerelease 1.0.0-alpha (it's < 1.0.0)")
		}
		if !vs.Contains(v2) {
			t.Error("Should contain regular 1.5.0")
		}
	})

	t.Run("Zero versions", func(t *testing.T) {
		v, err := pubgrub.ParseSemanticVersion("0.0.0")
		if err != nil {
			t.Fatalf("Failed to parse 0.0.0: %v", err)
		}
		if v.String() != "0.0.0" {
			t.Errorf("Expected 0.0.0, got %s", v.String())
		}
	})

	t.Run("Large version numbers", func(t *testing.T) {
		v, err := pubgrub.ParseSemanticVersion("999.999.999")
		if err != nil {
			t.Fatalf("Failed to parse large version: %v", err)
		}
		if v.String() != "999.999.999" {
			t.Errorf("Expected 999.999.999, got %s", v.String())
		}
	})
}
