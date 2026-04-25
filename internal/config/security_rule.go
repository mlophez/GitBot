package config

// SecurityRule defines which users are allowed to perform which actions
// on files belonging to a specific repository.
type SecurityRule struct {
	Repository   string
	FilePatterns []string
	Actions      []string
	Users        []string
}
