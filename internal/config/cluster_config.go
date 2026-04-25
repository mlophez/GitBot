package config

// ClusterAuth holds the authentication details for connecting to a cluster.
// Type "serviceaccount" means a local cluster accessed via in-cluster or kubeconfig credentials.
// Type "agent" means a remote cluster managed by another GitBot instance reachable at URL.
type ClusterAuth struct {
	Type                  string // "serviceaccount" for local, "agent" for remote
	URL                   string // Base URL for remote agent (only when Type == "agent")
	InsecureSkipTLSVerify bool   // Skip TLS certificate validation for the remote agent (use only in non-production environments)
}

// ClusterConfig describes a single cluster's connection parameters as defined in config.yaml.
type ClusterConfig struct {
	Name string
	Auth ClusterAuth
}
