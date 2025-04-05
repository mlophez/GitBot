package types

type GitHook struct {
	hooktype          HookType
  provider Provider
	repository    Repository
	comment Comment
	pullRequestID int
}

