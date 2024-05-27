package config

import (
	"fmt"

	"gitbot/internal/event"
)

type ConfigFile struct {
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

func (c ConfigFile) validate() error {

	// Group has any user
	if err := c.hasEmptyGroup(); err != nil {
		return err
	}

	// Group in rule exists
	if err := c.hasRuleGroupThatExists(); err != nil {
		return err
	}

	// Check Actions lock, unlock
	if err := c.hasRuleCorrectActions(); err != nil {
		return err
	}

	if err := c.hasRuleFieldEmpty(); err != nil {
		return err
	}

	return nil
}

func (c ConfigFile) hasEmptyGroup() error {
	for _, g := range c.Security.Groups {
		if len(g.Users) == 0 {
			return fmt.Errorf("Group '%s' is empty", g.Name)
		}
	}

	return nil
}

func (c ConfigFile) hasRuleGroupThatExists() error {
	for _, rule := range c.Security.Rules {
		for _, group := range rule.GroupList {
			exists := false
			for _, g := range c.Security.Groups {
				if g.Name == group {
					exists = true
				}
			}
			if !exists {
				return fmt.Errorf("The group '%s' do not exists in config", group)
			}
		}
	}
	return nil
}

func (c ConfigFile) hasRuleCorrectActions() error {
	for _, rule := range c.Security.Rules {
		for _, action := range rule.ActionList {
			if !(action == "lock" || action == "unlock") {
				return fmt.Errorf("The action '%s' is not valid", action)
			}
		}
	}
	return nil
}

func (c ConfigFile) hasRuleFieldEmpty() error {
	for _, rule := range c.Security.Rules {
		if len(rule.ActionList) == 0 {
			return fmt.Errorf("The field action is empty")
		}
		if len(rule.FilePatternList) == 0 {
			return fmt.Errorf("The field filepattern is empty")
		}
		if len(rule.UserList)+len(rule.GroupList) == 0 {
			return fmt.Errorf("The rule dont have any user associate")
		}
	}
	return nil
}

func (c ConfigFile) seRules() []event.SecurityRule {
	var result []event.SecurityRule

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
		result = append(result, event.SecurityRule{
			Repository:   r.Respository,
			FilePatterns: r.FilePatternList,
			Actions:      r.ActionList,
			Users:        userList,
		})
	}

	return result
}
