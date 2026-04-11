package adapter

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
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

// Load reads environment variables from env.ini and returns a populated Config.
// Panics if the env file cannot be loaded or the Kubernetes clientset cannot be created.
func (l *EnvConfigLoader) Load() *types.Config {
	if err := godotenv.Load("env.ini"); err != nil {
		panic("Error loading env.ini: " + err.Error())
	}

	// YAML security rules loading (disabled — enable when config.yaml is wired up):
	// filepath := os.Getenv("CONFIG_FILE")
	// data, _ := os.ReadFile(filepath)
	// var cf configFile
	// yaml.Unmarshal(data, &cf)
	// cf.validate()
	// rules := cf.securityRules()

	return &types.Config{
		SecurityRules:        []types.SecurityRule{},
		HttpPort:             os.Getenv("HTTP_PORT"),
		BitbucketBearerToken: os.Getenv("BITBUCKET_BEARER_TOKEN"),
		ClusterName:          os.Getenv("CLUSTER_NAME"),
		ClientSet:            newKubernetesClient(),
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

// configFile is the schema for the optional config.yaml security rules file.
type configFile struct {
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
