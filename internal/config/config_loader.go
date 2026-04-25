package config

// ConfigLoader abstracts loading the server configuration from an external source.
// Implemented by EnvConfigLoader for env-var + optional YAML based configuration.
type ConfigLoader interface {
	Load() *Config
}
