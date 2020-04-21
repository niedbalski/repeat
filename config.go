package main

import (
	"fmt"
	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Collections map[string]Collection `yaml:"collections"`
}

type Collection struct {
	Command   string `yaml:"command"`
	RunEvery  string `yaml:"run-every" default:"0s"`
	Timeout   string `yaml:"timeout" default:"0s"`
	RunOnce   bool   `yaml:"run-once" default:"false"`
	Script    string `yaml:"script"`
	ExitCodes string `yaml:"exit-codes" default:"any"`
}

func NewConfigFromFile(path string) (*Config, error) {
	config := new(Config)
	readConfig, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(readConfig, config)
	if err != nil {
		return nil, err
	}
	for name, collection := range config.Collections {
		if err := defaults.Set(&collection); err != nil {
			return nil, err
		}
		if collection.Command != "" && collection.Script != "" {
			return nil, fmt.Errorf("command or script stanzas are mutually exclusive")
		}

		config.Collections[name] = collection
	}

	return config, nil
}
