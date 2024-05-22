package config

import (
	"os"

	"github.com/joho/godotenv"
	//"gopkg.in/yaml.v3"
)

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

// func Load(filepath string) *Config {
// 	//err := godotenv.Load("env.ini")
// 	//if err != nil {
// 	//	panic("Error loading .env file")
// 	//}
//
// 	data, err := os.ReadFile(filepath)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	var configfile iYamlConfigFile
// 	err = yaml.Unmarshal(data, &configfile)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	// validate
// 	if err := configfile.validate(); err != nil {
// 		panic(err)
// 	}
//
// 	// Get security rules from configfile
// 	sg := configfile.seRules()
//
// 	return &Config{
// 		SecurityRules: sg,
// 	}
// }
