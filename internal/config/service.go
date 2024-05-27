package config

import (
	"gitbot/internal/event"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kubeconfig = "/home/mlr/Documents/Code/gitbot/kubeconfig"
)

type Config struct {
	HttpPort             string
	SecurityRules        []event.SecurityRule
	BitbucketBearerToken string
	ClientSet            *kubernetes.Clientset
	// *Clusters
	// ** clientset
}

func init() {
	err := godotenv.Load("env.ini")
	if err != nil {
		panic("Error loading .env file")
	}
}

//func Get(key string, def ...string) string {
//	return os.Getenv(key)
//}

func Get(key string) string {
	return os.Getenv(key)
}

func Load() *Config {
	filepath := Get("CONFIG_FILE")

	//err := godotenv.Load("env.ini")
	//if err != nil {
	//	panic("Error loading .env file")
	//}

	data, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	var configfile ConfigFile
	err = yaml.Unmarshal(data, &configfile)
	if err != nil {
		panic(err)
	}

	// validate
	if err := configfile.validate(); err != nil {
		panic(err)
	}

	// Get security rules from configfile
	sg := configfile.seRules()

	return &Config{
		SecurityRules:        sg,
		HttpPort:             Get("HTTP_PORT"),
		BitbucketBearerToken: Get("BITBUCKET_BEARER_TOKEN"),
		ClientSet:            NewKubernetes(),
	}
}

func NewKubernetes() *kubernetes.Clientset {
	config, err := func() (*rest.Config, error) {
		_, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST")
		if exists {
			return rest.InClusterConfig()
		} else {
			return clientcmd.BuildConfigFromFlags("", kubeconfig)
		}
	}()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}
