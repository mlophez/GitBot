package app

import (
	"errors"
	"testing"
)

func TestShouldUnlock(t *testing.T) {
	cases := []struct {
		name  string
		state PullRequestState
		want  bool
	}{
		{"open keeps lock", PullRequestStateOpen, false},
		{"unknown keeps lock", PullRequestStateUnknown, false},
		{"merged releases", PullRequestStateMerged, true},
		{"declined releases", PullRequestStateDeclined, true},
		{"superseded releases", PullRequestStateSuperseded, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldUnlock(c.state); got != c.want {
				t.Errorf("shouldUnlock(%v) = %v, want %v", c.state, got, c.want)
			}
		})
	}
}

// fakeManager is an in-memory AppManager that records unlocked apps.
type fakeManager struct {
	apps      []Application
	listErr   error
	unlockErr error
	unlocked  []string
}

func (f *fakeManager) List() ([]Application, error) { return f.apps, f.listErr }
func (f *fakeManager) Lock(_ Application, _ string, _ int, _ bool) error {
	return nil
}
func (f *fakeManager) Unlock(a Application) error {
	if f.unlockErr != nil {
		return f.unlockErr
	}
	f.unlocked = append(f.unlocked, a.Name)
	return nil
}

// fakeChecker returns a canned state (or error) per pull request id.
type fakeChecker struct {
	states map[int]PullRequestState
	errs   map[int]error
}

func (f fakeChecker) GetPullRequestState(_ string, prId int) (PullRequestState, error) {
	if err := f.errs[prId]; err != nil {
		return PullRequestStateUnknown, err
	}
	return f.states[prId], nil
}

func TestRepairLocks(t *testing.T) {
	mgr := &fakeManager{
		apps: []Application{
			{Name: "open-pr", Locked: true, PullRequestId: 1, Branch: "feat", LastBranch: "main"},
			{Name: "merged-pr", Locked: true, PullRequestId: 2, Branch: "feat", LastBranch: "main"},
			{Name: "declined-pr", Locked: true, PullRequestId: 3, Branch: "feat", LastBranch: "main"},
			{Name: "api-error", Locked: true, PullRequestId: 4, Branch: "feat", LastBranch: "main"},
			{Name: "not-locked", Locked: false, PullRequestId: -1},
			// Inconsistent: marked locked but branch unchanged -> Sanitize clears it, skipped.
			{Name: "inconsistent", Locked: true, PullRequestId: 5, Branch: "main", LastBranch: "main"},
		},
	}
	checker := fakeChecker{
		states: map[int]PullRequestState{
			1: PullRequestStateOpen,
			2: PullRequestStateMerged,
			3: PullRequestStateDeclined,
		},
		errs: map[int]error{
			4: errors.New("boom"),
		},
	}

	res, err := RepairLocks(mgr, checker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Checked != 4 {
		t.Errorf("Checked = %d, want 4 (open, merged, declined, api-error)", res.Checked)
	}
	if res.Unlocked != 2 {
		t.Errorf("Unlocked = %d, want 2 (merged, declined)", res.Unlocked)
	}
	if res.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2 (open, api-error)", res.Skipped)
	}
	if res.Failed != 0 {
		t.Errorf("Failed = %d, want 0", res.Failed)
	}

	want := map[string]bool{"merged-pr": true, "declined-pr": true}
	if len(mgr.unlocked) != 2 {
		t.Fatalf("unlocked = %v, want exactly merged-pr and declined-pr", mgr.unlocked)
	}
	for _, name := range mgr.unlocked {
		if !want[name] {
			t.Errorf("unexpected unlock of %q", name)
		}
	}
}

func TestRepairLocks_ListError(t *testing.T) {
	mgr := &fakeManager{listErr: errors.New("cannot list")}
	if _, err := RepairLocks(mgr, fakeChecker{}); err == nil {
		t.Fatal("expected error when List fails")
	}
}

func TestRepairLocks_UnlockError(t *testing.T) {
	mgr := &fakeManager{
		apps: []Application{
			{Name: "merged-pr", Locked: true, PullRequestId: 2, Branch: "feat", LastBranch: "main"},
		},
		unlockErr: errors.New("patch failed"),
	}
	checker := fakeChecker{states: map[int]PullRequestState{2: PullRequestStateMerged}}

	res, err := RepairLocks(mgr, checker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Failed != 1 || res.Unlocked != 0 {
		t.Errorf("got Failed=%d Unlocked=%d, want Failed=1 Unlocked=0", res.Failed, res.Unlocked)
	}
}
