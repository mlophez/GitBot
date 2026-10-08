package preview

import "testing"

// TestPullRequestValidate checks the minimal invariants: a positive number and
// both branch names are required to build a preview.
func TestPullRequestValidate(t *testing.T) {
	noNumber := openPR(0)
	noBranch := openPR(1)
	noBranch.Branch = ""
	noTarget := openPR(1)
	noTarget.TargetBranch = ""

	tests := []struct {
		name    string
		in      PullRequest
		wantErr bool
	}{
		{"valid", openPR(1), false},
		{"non-positive number", noNumber, true},
		{"missing source branch", noBranch, true},
		{"missing target branch", noTarget, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.in.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
