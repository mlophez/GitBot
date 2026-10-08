package config

import (
	"strings"
)

const (
	bitbucketTokenPrefix = "BITBUCKET_"
	bitbucketTokenSuffix = "_TOKEN"
)

// BitbucketTokens resolves the bearer token to use against the Bitbucket API for a
// given repository. Repository access tokens are scoped to a single repository, so
// each repository the bot talks to may need its own token, supplied as the env var
// BITBUCKET_<WORKSPACE>_<REPO>_TOKEN (see BitbucketTokenEnvName). Repositories
// without a dedicated token fall back to Default (BITBUCKET_BEARER_TOKEN).
type BitbucketTokens struct {
	Default string            // fallback token (BITBUCKET_BEARER_TOKEN)
	byEnv   map[string]string // repository-specific tokens keyed by env var name
}

// NewBitbucketTokens builds a BitbucketTokens from the fallback token and an
// environment in os.Environ() "KEY=value" form. Every non-empty
// BITBUCKET_*_TOKEN variable is kept as a candidate repository token.
func NewBitbucketTokens(defaultToken string, environ []string) BitbucketTokens {
	byEnv := make(map[string]string)
	for _, kv := range environ {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || value == "" {
			continue
		}
		if strings.HasPrefix(key, bitbucketTokenPrefix) && strings.HasSuffix(key, bitbucketTokenSuffix) {
			byEnv[key] = value
		}
	}
	return BitbucketTokens{Default: defaultToken, byEnv: byEnv}
}

// For returns the token for workspace/repo: the value of
// BITBUCKET_<WORKSPACE>_<REPO>_TOKEN when set, otherwise Default.
func (t BitbucketTokens) For(workspace, repo string) string {
	if token, ok := t.byEnv[BitbucketTokenEnvName(workspace, repo)]; ok {
		return token
	}
	return t.Default
}

// EnvNames returns the names of the repository-specific token variables found,
// for logging at startup. Values are never exposed.
func (t BitbucketTokens) EnvNames() []string {
	names := make([]string, 0, len(t.byEnv))
	for name := range t.byEnv {
		names = append(names, name)
	}
	return names
}

// BitbucketTokenEnvName returns the env var name holding the token of
// workspace/repo: BITBUCKET_<WORKSPACE>_<REPO>_TOKEN, upper-cased and with every
// character that is not a letter or digit (e.g. "-" or ".") replaced by "_",
// since those are not valid in environment variable names.
// Example: firmapro/kubeops-agent → BITBUCKET_FIRMAPRO_KUBEOPS_AGENT_TOKEN.
func BitbucketTokenEnvName(workspace, repo string) string {
	return bitbucketTokenPrefix + envSegment(workspace) + "_" + envSegment(repo) + bitbucketTokenSuffix
}

// envSegment upper-cases s and replaces any character outside [A-Z0-9] with "_".
func envSegment(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, s)
}
