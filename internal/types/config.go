package types

import "k8s.io/client-go/kubernetes"

// ClusterAuth holds the authentication details for connecting to a cluster.
// Type "serviceaccount" means a local cluster accessed via in-cluster or kubeconfig credentials.
// Type "agent" means a remote cluster managed by another GitBot instance reachable at URL.
type ClusterAuth struct {
	Type                string // "serviceaccount" for local, "agent" for remote
	URL                 string // Base URL for remote agent (only when Type == "agent")
	InsecureSkipTLSVerify bool // Skip TLS certificate validation for the remote agent (use only in non-production environments)
}

// ClusterConfig describes a single cluster's connection parameters as defined in config.yaml.
type ClusterConfig struct {
	Name string
	Auth ClusterAuth
}

// Config holds the runtime configuration for the server.
type Config struct {
	HttpPort              string
	SecurityRules         []SecurityRule
	BitbucketBearerToken  string
	ClientSet             *kubernetes.Clientset
	ClusterName           string
	Clusters              []ClusterConfig // Parsed from config.yaml clusters section
	BotKubernetesUsername string          // Kubernetes username of the bot service account (BOT_KUBERNETES_USERNAME)
}

// ConfigLoader abstracts loading the server configuration from an external source.
// Implemented by adapter.EnvConfigLoader for env-var + optional YAML based configuration.
type ConfigLoader interface {
	Load() *Config
}
