// Package config holds the runtime configuration of kubeops-agent.
// It exposes the Config struct, the ConfigLoader interface, and the
// EnvConfigLoader implementation that reads env vars + an optional YAML file.
package config

import "k8s.io/client-go/kubernetes"

// Config holds the runtime configuration for the server.
type Config struct {
	HttpPort              string
	ContextRoot           string // Optional URL path prefix for all routes (CONTEXT_ROOT env var)
	WebhookToken          string // Shared secret for validating incoming webhooks (WEBHOOK_TOKEN env var)
	APIToken              string // Bearer token required on all API requests (API_TOKEN env var)
	SecurityRules         []SecurityRule
	BitbucketBearerToken  string
	ClientSet             *kubernetes.Clientset
	ClusterName           string
	Clusters              []ClusterConfig // Parsed from config.yaml clusters section
	BotKubernetesUsername string          // Kubernetes username of the bot service account (BOT_KUBERNETES_USERNAME)
	BitbucketBotUUID      string          // Bitbucket UUID of the bot account used to post comments (BITBUCKET_BOT_UUID)
}
