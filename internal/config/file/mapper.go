package config

func (c iYamlConfigFile) seRules() []SecurityRule {
	var result []SecurityRule

	for _, r := range c.Security.Rules {
		// Add users from group to userlist
		userList := r.UserList
		for _, group := range r.GroupList {
			for _, g := range c.Security.Groups {
				if group != g.Name {
					continue
				}
				userList = append(userList, g.Users...)
			}
		}

		// create final securityrule
		result = append(result, SecurityRule{
			Repository:      r.Respository,
			FilePatternList: r.FilePatternList,
			ActionList:      r.ActionList,
			UserList:        userList,
		})
	}

	return result
}
