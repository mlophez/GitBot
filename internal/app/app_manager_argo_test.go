package app

import "testing"

// makeArgoApp builds a minimal argoApp with the given sync/health status so the
// status-derivation helpers can be exercised without a Kubernetes backend.
func makeArgoApp(sync, health, healthMsg string) argoApp {
	var a argoApp
	a.Metadata.Name = "demo"
	a.Status.Sync.Status = sync
	a.Status.Health.Status = health
	a.Status.Health.Message = healthMsg
	return a
}

// TestDeriveStatus verifies that deriveStatus collapses sync and health into a single
// display value, giving precedence to an unhealthy health status over any sync status.
func TestDeriveStatus(t *testing.T) {
	cases := []struct {
		name       string
		sync       string
		health     string
		wantStatus string
	}{
		{"healthy and synced", "Synced", "Healthy", "Healthy"},
		{"unhealthy takes precedence over synced", "Synced", "Degraded", "Degraded"},
		{"unhealthy takes precedence over outofsync", "OutOfSync", "Missing", "Missing"},
		{"healthy but outofsync", "OutOfSync", "Healthy", "OutOfSync"},
		{"progressing", "Synced", "Progressing", "Progressing"},
		{"no status at all", "", "", "Unknown"},
		{"only sync present", "Synced", "", "Synced"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := toApplication(makeArgoApp(c.sync, c.health, ""))
			if got.Status != c.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, c.wantStatus)
			}
		})
	}
}

// TestDeriveStatusMessage_HealthyHasNoMessage confirms that a healthy, synced app
// produces an empty StatusMessage even when ArgoCD reports a health detail string.
func TestDeriveStatusMessage_HealthyHasNoMessage(t *testing.T) {
	got := toApplication(makeArgoApp("Synced", "Healthy", "all good"))
	if got.StatusMessage != "" {
		t.Errorf("StatusMessage = %q, want empty for healthy app", got.StatusMessage)
	}
}

// TestDeriveStatusMessage_FallsBackToHealthMessage checks that when no condition or
// failed-operation message is present, the health message is used as a fallback.
func TestDeriveStatusMessage_FallsBackToHealthMessage(t *testing.T) {
	got := toApplication(makeArgoApp("OutOfSync", "Degraded", "pod crashloop"))
	if got.StatusMessage != "pod crashloop" {
		t.Errorf("StatusMessage = %q, want %q", got.StatusMessage, "pod crashloop")
	}
}

// TestDeriveStatusMessage_PrefersErrorCondition checks that a status condition whose
// type contains "error" takes priority over the health message.
func TestDeriveStatusMessage_PrefersErrorCondition(t *testing.T) {
	a := makeArgoApp("OutOfSync", "Degraded", "health detail")
	a.Status.Conditions = []struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}{
		{Type: "ComparisonError", Message: "manifest generation failed"},
	}
	got := toApplication(a)
	if got.StatusMessage != "manifest generation failed" {
		t.Errorf("StatusMessage = %q, want error condition message", got.StatusMessage)
	}
}

// TestDeriveStatusMessage_PrefersFailedOperationOverHealth checks that a failed
// ArgoCD sync operation message takes priority over the health message.
func TestDeriveStatusMessage_PrefersFailedOperationOverHealth(t *testing.T) {
	a := makeArgoApp("OutOfSync", "Degraded", "health detail")
	a.Status.OperationState.Phase = "Failed"
	a.Status.OperationState.Message = "sync operation failed"
	got := toApplication(a)
	if got.StatusMessage != "sync operation failed" {
		t.Errorf("StatusMessage = %q, want failed-operation message", got.StatusMessage)
	}
}
