package config

type iYamlConfigFile struct {
	Security struct {
		Groups []struct {
			Name  string   `yaml:"name"`
			Users []string `yaml:"users"`
		} `yaml:"groups"`
		Rules []struct {
			Respository     string   `yaml:"repository"`
			FilePatternList []string `yaml:"filepattern"`
			ActionList      []string `yaml:"action"`
			GroupList       []string `yaml:"group"`
			UserList        []string `yaml:"user"`
		} `yaml:"rules"`
	} `yaml:"security"`
}
