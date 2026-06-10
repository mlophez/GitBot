package event

import "testing"

// TestEventValidate checks the Event invariants: a repository and a recognized
// type are always required. A recognized PR event must also carry a valid pull
// request, but an EventTypeUnknown event (an irrelevant webhook the bot ignores)
// is accepted even without a pull request so it can be silently discarded later.
func TestEventValidate(t *testing.T) {
	validPR := PullRequest{Id: 7}
	cases := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name:    "valid commented event",
			event:   Event{Type: EventTypeCommented, Repository: "https://bitbucket.org/org/repo.git", PullRequest: validPR},
			wantErr: false,
		},
		{
			name:    "valid opened event",
			event:   Event{Type: EventTypeOpened, Repository: "https://bitbucket.org/org/repo.git", PullRequest: validPR},
			wantErr: false,
		},
		{
			name:    "unknown event without pull request is accepted",
			event:   Event{Type: EventTypeUnknown, Repository: "https://bitbucket.org/org/repo.git"},
			wantErr: false,
		},
		{
			name:    "missing repository",
			event:   Event{Type: EventTypeCommented, PullRequest: validPR},
			wantErr: true,
		},
		{
			name:    "type out of range high",
			event:   Event{Type: EventType(99), Repository: "https://bitbucket.org/org/repo.git", PullRequest: validPR},
			wantErr: true,
		},
		{
			name:    "type out of range low",
			event:   Event{Type: EventType(-5), Repository: "https://bitbucket.org/org/repo.git", PullRequest: validPR},
			wantErr: true,
		},
		{
			name:    "recognized event with invalid pull request",
			event:   Event{Type: EventTypeMerged, Repository: "https://bitbucket.org/org/repo.git", PullRequest: PullRequest{Id: 0}},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.event.Validate()
			if c.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error")
			}
			if !c.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
