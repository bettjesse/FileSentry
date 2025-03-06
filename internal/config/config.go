package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all rules.
type Config struct {
	Rules []Rule `yaml:"rules"`
}

// Rule represents one file processing rule.
type Rule struct {
	Name    string   `yaml:"name"`
	Watch   string   `yaml:"watch"`
	Filters []Filter `yaml:"filters"`
	Actions []Action `yaml:"actions"`
}

// Filter represents file filters (e.g., by extension).
type Filter struct {
	Extensions []string `yaml:"extension"`
}

// Action represents actions to be performed on matching files.
type Action struct {
	Move string `yaml:"move"`
}

// LoadRules reads the YAML file and returns the parsed rules.
func LoadRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML format: %w", err)
	}
	return cfg.Rules, nil
}
