package pubgrub

import "testing"

func mustParseVersionRange(t *testing.T, s string) VersionSet {
	set, err := ParseVersionRange(s)
	if err != nil {
		t.Fatalf("ParseVersionRange(%q): %v", s, err)
	}
	return set
}

func mustSemver(t *testing.T, s string) Version {
	v, err := ParseSemanticVersion(s)
	if err != nil {
		t.Fatalf("ParseSemanticVersion(%q): %v", s, err)
	}
	return v
}

func TestParseVersionRangeContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rangeExpr string
		version   string
		expect    bool
	}{
		{">=1.0.0", "1.0.0", true},
		{">=1.0.0", "0.9.9", false},
		{">=1.0.0, <2.0.0", "1.5.0", true},
		{">=1.0.0, <2.0.0", "2.0.0", false},
		{"==1.5.0", "1.5.0", true},
		{"==1.5.0", "1.5.1", false},
		{"!=1.5.0", "1.5.0", false},
		{"!=1.5.0", "1.6.0", true},
		{">=1.0.0, <2.0.0 || >=3.0.0", "3.2.0", true},
		{">=1.0.0, <2.0.0 || >=3.0.0", "2.5.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.rangeExpr+" contains "+tt.version, func(t *testing.T) {
			set := mustParseVersionRange(t, tt.rangeExpr)
			ver := mustSemver(t, tt.version)
			if got := set.Contains(ver); got != tt.expect {
				t.Fatalf("Contains(%s) = %v, want %v", tt.version, got, tt.expect)
			}
		})
	}
}

func TestVersionSetOperations(t *testing.T) {
	t.Parallel()

	setA := mustParseVersionRange(t, ">=1.0.0, <2.0.0")
	setB := mustParseVersionRange(t, ">=1.5.0, <3.0.0")

	intersection := setA.Intersection(setB)
	if intersection.IsEmpty() {
		t.Fatal("expected intersection to be non-empty")
	}

	if !intersection.Contains(mustSemver(t, "1.7.0")) {
		t.Fatal("expected intersection to contain 1.7.0")
	}

	if intersection.Contains(mustSemver(t, "2.5.0")) {
		t.Fatal("did not expect intersection to contain 2.5.0")
	}

	union := setA.Union(setB)
	if !union.Contains(mustSemver(t, "2.5.0")) {
		t.Fatal("expected union to contain 2.5.0")
	}
}

func TestVersionSetComplement(t *testing.T) {
	t.Parallel()

	set := mustParseVersionRange(t, ">=1.0.0, <2.0.0")
	comp := set.Complement()

	if comp.Contains(mustSemver(t, "1.5.0")) {
		t.Fatal("complement should not contain 1.5.0")
	}

	if !comp.Contains(mustSemver(t, "2.5.0")) {
		t.Fatal("complement should contain 2.5.0")
	}
}

func TestVersionSetCondition(t *testing.T) {
	t.Parallel()

	set := mustParseVersionRange(t, ">=1.0.0, <2.0.0")
	cond := NewVersionSetCondition(set)

	if !cond.Satisfies(mustSemver(t, "1.2.3")) {
		t.Fatal("condition should satisfy 1.2.3")
	}

	if cond.Satisfies(mustSemver(t, "2.1.0")) {
		t.Fatal("condition should not satisfy 2.1.0")
	}
}

func TestVersionSetString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"*", "*"},
		{">=1.0.0", ">=1.0.0"},
		{">=1.0.0, <2.0.0", ">=1.0.0, <2.0.0"},
		{">=1.0.0, <2.0.0 || >=3.0.0", ">=1.0.0, <2.0.0 || >=3.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			set := mustParseVersionRange(t, tt.input)
			if got := set.String(); got != tt.expected {
				t.Fatalf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseVersionRangeErrors(t *testing.T) {
	t.Parallel()

	// Test cases that should return errors
	tests := []string{">=1.0.0,", "|| >=1.0.0"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseVersionRange(input); err == nil {
				t.Fatalf("expected error for input %q", input)
			}
		})
	}
}

func TestVersionSetIsSubset(t *testing.T) {
	t.Parallel()

	setA := mustParseVersionRange(t, ">=1.5.0, <1.8.0")
	setB := mustParseVersionRange(t, ">=1.0.0, <2.0.0")
	setC := mustParseVersionRange(t, ">=2.0.0, <3.0.0")

	if !setA.IsSubset(setB) {
		t.Fatal("setA should be subset of setB")
	}

	if setB.IsSubset(setA) {
		t.Fatal("setB should not be subset of setA")
	}

	if setA.IsSubset(setC) {
		t.Fatal("setA should not be subset of setC (disjoint)")
	}

	empty := EmptyVersionSet()
	if !empty.IsSubset(setA) {
		t.Fatal("empty set should be subset of any set")
	}
}

func TestVersionSetIsDisjoint(t *testing.T) {
	t.Parallel()

	setA := mustParseVersionRange(t, ">=1.0.0, <2.0.0")
	setB := mustParseVersionRange(t, ">=2.0.0, <3.0.0")
	setC := mustParseVersionRange(t, ">=1.5.0, <2.5.0")

	if !setA.IsDisjoint(setB) {
		t.Fatal("setA and setB should be disjoint")
	}

	if setA.IsDisjoint(setC) {
		t.Fatal("setA and setC should overlap")
	}

	empty := EmptyVersionSet()
	if !empty.IsDisjoint(setA) {
		t.Fatal("empty set should be disjoint with any set")
	}
}

func TestVersionSetSingleton(t *testing.T) {
	t.Parallel()

	set := EmptyVersionSet()
	v := mustSemver(t, "1.2.3")

	singleton := set.Singleton(v)
	if !singleton.Contains(v) {
		t.Fatal("singleton should contain the version")
	}

	if singleton.Contains(mustSemver(t, "1.2.4")) {
		t.Fatal("singleton should not contain other versions")
	}

	if !singleton.IsEmpty() && singleton.String() != "==1.2.3" {
		t.Fatalf("singleton string should be ==1.2.3, got %q", singleton.String())
	}
}

func TestEmptyAndFullVersionSet(t *testing.T) {
	t.Parallel()

	empty := EmptyVersionSet()
	if !empty.IsEmpty() {
		t.Fatal("EmptyVersionSet should be empty")
	}

	full := FullVersionSet()
	if full.IsEmpty() {
		t.Fatal("FullVersionSet should not be empty")
	}

	v := mustSemver(t, "1.2.3")
	if empty.Contains(v) {
		t.Fatal("empty set should not contain any version")
	}

	if !full.Contains(v) {
		t.Fatal("full set should contain any version")
	}

	if full.String() != "*" {
		t.Fatalf("full set string should be *, got %q", full.String())
	}
}

func TestVersionSetConditionNil(t *testing.T) {
	t.Parallel()

	var cond *VersionSetCondition
	if cond.String() != "*" {
		t.Fatalf("nil condition string should be *, got %q", cond.String())
	}

	if !cond.Satisfies(mustSemver(t, "1.2.3")) {
		t.Fatal("nil condition should satisfy any version")
	}
}
