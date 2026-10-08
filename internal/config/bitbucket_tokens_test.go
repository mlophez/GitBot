package config

import "testing"

// TestBitbucketTokenEnvName checks the naming convention: upper-cased workspace and
// repo, with characters not valid in env var names replaced by "_".
func TestBitbucketTokenEnvName(t *testing.T) {
	tests := []struct {
		workspace, repo, want string
	}{
		{"firmapro", "platform", "BITBUCKET_FIRMAPRO_PLATFORM_TOKEN"},
		{"firmapro", "kubeops-agent", "BITBUCKET_FIRMAPRO_KUBEOPS_AGENT_TOKEN"},
		{"FirmaPro", "my.repo_2", "BITBUCKET_FIRMAPRO_MY_REPO_2_TOKEN"},
	}
	for _, tt := range tests {
		if got := BitbucketTokenEnvName(tt.workspace, tt.repo); got != tt.want {
			t.Errorf("BitbucketTokenEnvName(%q, %q) = %q, want %q", tt.workspace, tt.repo, got, tt.want)
		}
	}
}

// TestBitbucketTokensFor checks resolution: a repository-specific variable wins,
// anything else (unknown repo, empty value, unrelated variables) falls back to the
// default token.
func TestBitbucketTokensFor(t *testing.T) {
	tokens := NewBitbucketTokens("default", []string{
		"BITBUCKET_FIRMAPRO_PLATFORM_TOKEN=platform-token",
		"BITBUCKET_FIRMAPRO_KUBEOPS_AGENT_TOKEN=agent-token",
		"BITBUCKET_FIRMAPRO_EMPTY_TOKEN=",
		"OTHER_FIRMAPRO_KUBERNETES_TOKEN=ignored",
		"MALFORMED",
	})

	tests := []struct {
		name, workspace, repo, want string
	}{
		{"repository token", "firmapro", "platform", "platform-token"},
		{"repository token with dash in slug", "firmapro", "kubeops-agent", "agent-token"},
		{"unknown repository falls back", "firmapro", "unknown", "default"},
		{"empty repository token falls back", "firmapro", "empty", "default"},
		{"unrelated prefix is ignored", "firmapro", "kubernetes", "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tokens.For(tt.workspace, tt.repo); got != tt.want {
				t.Errorf("For(%q, %q) = %q, want %q", tt.workspace, tt.repo, got, tt.want)
			}
		})
	}
}
