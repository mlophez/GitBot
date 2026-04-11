package types

import "k8s.io/client-go/kubernetes"

// Config holds the runtime configuration for the server.
type Config struct {
	HttpPort             string
	SecurityRules        []SecurityRule
	BitbucketBearerToken string
	ClientSet            *kubernetes.Clientset
	ClusterName          string
}

// ConfigLoader abstracts loading the server configuration from an external source.
// Implemented by adapter.EnvConfigLoader for env-var + optional YAML based configuration.
type ConfigLoader interface {
	Load() *Config
}
