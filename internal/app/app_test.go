package app

import "testing"

// TestApplicationValidate checks the minimal domain invariants of an Application:
// it must carry a name and a repository. Other fields (branch, lock state) are not
// enforced here so that apps hydrated from ArgoCD with legitimate gaps are not rejected.
func TestApplicationValidate(t *testing.T) {
	cases := []struct {
		name    string
		app     Application
		wantErr bool
	}{
		{"valid", Application{Name: "demo", Repository: "https://bitbucket.org/org/repo.git"}, false},
		{"missing name", Application{Repository: "https://bitbucket.org/org/repo.git"}, true},
		{"missing repository", Application{Name: "demo"}, true},
		{"missing both", Application{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.app.Validate()
			if c.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error")
			}
			if !c.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
