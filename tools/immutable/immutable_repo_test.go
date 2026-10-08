package immutable_test

import (
	"fmt"
	"sort"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"

	"gitbot/tools/immutable"
)

// TestDomainFieldsAreOnlyAssignedByTheirMethods runs the immutable analyzer over
// every package of the module (test files included) and fails with the offending
// positions. It is the guard that keeps state transitions of the domain types in
// immutable.ProtectedTypes inside their own methods.
func TestDomainFieldsAreOnlyAssignedByTheirMethods(t *testing.T) {
	cfg := &packages.Config{
		// The test binary runs in tools/immutable; load the whole module from its root.
		Dir:   "../..",
		Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, "gitbot/...")
	if err != nil {
		t.Fatalf("loading packages: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no packages loaded")
	}

	var violations []string
	for _, pkg := range pkgs {
		// Skip packages that failed to type-check: without type information the
		// analyzer cannot tell which type a selector belongs to.
		if len(pkg.Errors) > 0 || pkg.TypesInfo == nil {
			continue
		}
		pass := &analysis.Pass{
			Analyzer:  immutable.Analyzer,
			Fset:      pkg.Fset,
			Files:     pkg.Syntax,
			Pkg:       pkg.Types,
			TypesInfo: pkg.TypesInfo,
			ResultOf:  map[*analysis.Analyzer]any{},
			Report: func(d analysis.Diagnostic) {
				violations = append(violations, fmt.Sprintf("%s: %s", pkg.Fset.Position(d.Pos), d.Message))
			},
		}
		if _, err := immutable.Analyzer.Run(pass); err != nil {
			t.Fatalf("running analyzer on %s: %v", pkg.PkgPath, err)
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		for _, v := range dedup(violations) {
			t.Error(v)
		}
	}
}

// dedup removes the duplicated diagnostics produced when the same file is loaded
// more than once (a package and its test variant share non-test files).
func dedup(items []string) []string {
	seen := make(map[string]bool, len(items))
	unique := make([]string, 0, len(items))
	for _, item := range items {
		if seen[item] {
			continue
		}
		seen[item] = true
		unique = append(unique, item)
	}
	return unique
}
