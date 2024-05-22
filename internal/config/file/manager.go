package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ConfigManager struct{}

func Load(filepath string) *Config {
	//err := godotenv.Load("env.ini")
	//if err != nil {
	//	panic("Error loading .env file")
	//}

	data, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	var configfile iYamlConfigFile
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
		SecurityRules: sg,
	}
}
