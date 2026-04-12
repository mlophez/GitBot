package adapters

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"gitbot/internal/types"
)

const kubeconfig = "/home/mlr/Documents/Code/gitbot/kubeconfig"

// EnvConfigLoader implements types.ConfigLoader.
// It loads configuration from environment variables (via env.ini) and an optional YAML config file.
type EnvConfigLoader struct{}

// NewEnvConfigLoader creates an EnvConfigLoader ready to use.
func NewEnvConfigLoader() *EnvConfigLoader {
	return &EnvConfigLoader{}
}

// Load reads environment variables from env.ini, parses the optional config.yaml
// for cluster definitions and security rules, and returns a populated Config.
// Panics if the env file cannot be loaded or the Kubernetes clientset cannot be created.
func (l *EnvConfigLoader) Load() *types.Config {
	if err := godotenv.Load("env.ini"); err != nil {
		panic("Error loading env.ini: " + err.Error())
	}

	var clusters []types.ClusterConfig
	var rules []types.SecurityRule

	if filepath := os.Getenv("CONFIG_FILE"); filepath != "" {
		data, err := os.ReadFile(filepath)
		if err != nil {
			slog.Warn("Could not read config file, skipping", "path", filepath, "error", err)
		} else {
			var cf configFile
			if err := yaml.Unmarshal(data, &cf); err != nil {
				slog.Warn("Could not parse config file, skipping", "path", filepath, "error", err)
			} else {
				if err := cf.validate(); err != nil {
					slog.Warn("Config file validation failed, skipping security rules", "error", err)
				} else {
					rules = cf.securityRules()
				}
				clusters = cf.clusterConfigs()
			}
		}
	}

	return &types.Config{
		SecurityRules:         rules,
		HttpPort:              os.Getenv("HTTP_PORT"),
		ContextRoot:           os.Getenv("CONTEXT_ROOT"),
		WebhookToken:          os.Getenv("WEBHOOK_TOKEN"),
		BitbucketBearerToken:  os.Getenv("BITBUCKET_BEARER_TOKEN"),
		ClusterName:           os.Getenv("CLUSTER_NAME"),
		ClientSet:             newKubernetesClient(),
		Clusters:              clusters,
		BotKubernetesUsername: os.Getenv("BOT_KUBERNETES_USERNAME"),
	}
}

func newKubernetesClient() *kubernetes.Clientset {
	cfg, err := func() (*rest.Config, error) {
		if _, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST"); exists {
			return rest.InClusterConfig()
		}
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}()
	if err != nil {
		panic(err)
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	return cs
}

// ── YAML config file (security rules) ────────────────────────────────────────

// configFile is the schema for the optional config.yaml file.
// It holds cluster definitions and security rules.
type configFile struct {
	Clusters []struct {
		Name string `yaml:"name"`
		Auth struct {
			Type                string `yaml:"type"`
			URL                 string `yaml:"url"`
			InsecureSkipTLSVerify bool   `yaml:"insecure-skip-tls-verify"`
		} `yaml:"auth"`
	} `yaml:"clusters"`
	Security struct {
		Groups []struct {
			Name  string   `yaml:"name"`
			Users []string `yaml:"users"`
		} `yaml:"groups"`
		Rules []struct {
			Repository      string   `yaml:"repository"`
			FilePatternList []string `yaml:"filepattern"`
			ActionList      []string `yaml:"action"`
			GroupList       []string `yaml:"group"`
			UserList        []string `yaml:"user"`
		} `yaml:"rules"`
	} `yaml:"security"`
}

// clusterConfigs converts the YAML cluster definitions into domain ClusterConfig values.
func (c configFile) clusterConfigs() []types.ClusterConfig {
	var result []types.ClusterConfig
	for _, cl := range c.Clusters {
		result = append(result, types.ClusterConfig{
			Name: cl.Name,
			Auth: types.ClusterAuth{
				Type:                cl.Auth.Type,
				URL:                 cl.Auth.URL,
				InsecureSkipTLSVerify: cl.Auth.InsecureSkipTLSVerify,
			},
		})
	}
	return result
}

func (c configFile) validate() error {
	for _, g := range c.Security.Groups {
		if len(g.Users) == 0 {
			return fmt.Errorf("group '%s' is empty", g.Name)
		}
	}
	for _, rule := range c.Security.Rules {
		for _, group := range rule.GroupList {
			found := false
			for _, g := range c.Security.Groups {
				if g.Name == group {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("group '%s' in rule does not exist in config", group)
			}
		}
		for _, action := range rule.ActionList {
			if action != "lock" && action != "unlock" {
				return fmt.Errorf("action '%s' is not valid", action)
			}
		}
		if len(rule.ActionList) == 0 {
			return fmt.Errorf("rule has no actions")
		}
		if len(rule.FilePatternList) == 0 {
			return fmt.Errorf("rule has no filepattern")
		}
		if len(rule.UserList)+len(rule.GroupList) == 0 {
			return fmt.Errorf("rule has no users or groups")
		}
	}
	return nil
}

func (c configFile) securityRules() []types.SecurityRule {
	var result []types.SecurityRule
	for _, r := range c.Security.Rules {
		users := append([]string{}, r.UserList...)
		for _, groupName := range r.GroupList {
			for _, g := range c.Security.Groups {
				if g.Name == groupName {
					users = append(users, g.Users...)
				}
			}
		}
		result = append(result, types.SecurityRule{
			Repository:   r.Repository,
			FilePatterns: r.FilePatternList,
			Actions:      r.ActionList,
			Users:        users,
		})
	}
	return result
}
