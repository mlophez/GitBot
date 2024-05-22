package types

type SecurityRule struct {
	Repository   string
	FilePatterns []string
	Actions      []int
	Users        []string
}
