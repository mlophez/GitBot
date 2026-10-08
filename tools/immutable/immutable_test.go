package immutable_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"gitbot/tools/immutable"
)

// TestAnalyzer checks the rule against the golden files in testdata: package "a"
// declares the protected type and covers the assignments done from inside its own
// package, and package "b" covers the ones done from another package. The
// expected diagnostics are the "// want" comments in those files.
func TestAnalyzer(t *testing.T) {
	analyzer := immutable.NewAnalyzer("a.Aggregate")
	analysistest.Run(t, analysistest.TestData(), analyzer, "a", "b")
}
