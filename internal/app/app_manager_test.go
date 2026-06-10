package app

import "testing"

// TestKeepValidApps verifies that mass-hydration helpers drop invalid applications
// instead of failing the whole listing: invalid apps are filtered out and the valid
// ones are preserved in order.
func TestKeepValidApps(t *testing.T) {
	in := []Application{
		{Name: "good-1", Repository: "https://bitbucket.org/org/a.git"},
		{Name: "", Repository: "https://bitbucket.org/org/b.git"}, // invalid: no name
		{Name: "good-2", Repository: "https://bitbucket.org/org/c.git"},
		{Name: "no-repo"}, // invalid: no repository
	}

	got := keepValidApps(in)

	if len(got) != 2 {
		t.Fatalf("keepValidApps returned %d apps, want 2", len(got))
	}
	if got[0].Name != "good-1" || got[1].Name != "good-2" {
		t.Errorf("keepValidApps kept %q, %q; want good-1, good-2", got[0].Name, got[1].Name)
	}
}
