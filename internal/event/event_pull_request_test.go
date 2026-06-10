package event

import "testing"

// TestPullRequestValidate checks that a pull request must carry a positive id.
func TestPullRequestValidate(t *testing.T) {
	cases := []struct {
		name    string
		pr      PullRequest
		wantErr bool
	}{
		{"valid id", PullRequest{Id: 42}, false},
		{"zero id", PullRequest{Id: 0}, true},
		{"negative id", PullRequest{Id: -1}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.pr.Validate()
			if c.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error")
			}
			if !c.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
