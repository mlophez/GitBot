package event

import "testing"

// TestEventResponseValidate checks the response's only own invariant: a response
// is either a help reply (Environments set) or an operation result (Summary set),
// never both at once. The mutually-exclusive-mode check never triggers for the
// responses the use case builds today; it guards against future corruption.
func TestEventResponseValidate(t *testing.T) {
	cases := []struct {
		name    string
		resp    EventResponse
		wantErr bool
	}{
		{"help only", EventResponse{Environments: []string{"dev"}}, false},
		{"operation only", EventResponse{Success: true, Summary: []EventAppStatus{{Name: "demo"}}}, false},
		{"empty", EventResponse{}, false},
		{"message only", EventResponse{Success: false, Message: "boom"}, false},
		{"mixed modes", EventResponse{Environments: []string{"dev"}, Summary: []EventAppStatus{{Name: "demo"}}}, true},
		{"empty environments slice with summary", EventResponse{Environments: []string{}, Summary: []EventAppStatus{{Name: "demo"}}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.resp.Validate()
			if c.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error")
			}
			if !c.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
