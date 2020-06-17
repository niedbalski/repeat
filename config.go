package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/creasty/defaults"
	log "github.com/sirupsen/logrus"
	"github.com/utahta/go-openuri"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"strings"
)

type MapValueField struct {
	Name  string `yaml:"name"`
	Type  string `yaml:"type" default:""`
	Index string `yaml:"idx"`
}

type MapValue struct {
	Separator       string          `yaml:"separator" default:","`
	AppendTimestamp bool            `yaml:"add-timestamp" default:"false"`
	Fields          []MapValueField `yaml:"fields"`
	IgnoreRegex     string          `yaml:"ignore-regex,omitempty"`
}

type DBConfig struct {
	MapValues MapValue `yaml:"map-values,omitempty"`
	Name      string   `yaml:"name,omitempty"`
}

type Collection struct {
	Command   string   `yaml:"command"`
	RunEvery  string   `yaml:"run-every" default:"0s"`
	Timeout   string   `yaml:"timeout" default:"0s"`
	RunOnce   bool     `yaml:"run-once" default:"false"`
	Script    string   `yaml:"script"`
	ExitCodes string   `yaml:"exit-codes" default:"any"`
	Store     string   `yaml:"store" default:"file"`
	Database  DBConfig `yaml:"database"`
}

func (c *Collection) SetDefaults() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	if c.Command != "" && c.Script != "" {
		return fmt.Errorf("command or script stanzas are mutually exclusive")
	}
	return nil
}

type Config struct {
	Collections map[string]Collection `yaml:"collections"`
	Imports     []string              `yaml:"import,omitempty"`
}

func fetchImport(importURL string) ([]byte, error) {
	log.Debugf("Importing item: %s", importURL)

	parsed, err := url.Parse(importURL)
	if err != nil {
		return nil, err
	}
	importChecksum := ""
	if strings.Contains(parsed.Fragment, "md5sum") {
		splitted := strings.Split(parsed.Fragment, "=")
		// if its a file, use the prefix before the sum as the filepath.
		if parsed.Scheme == "" {
			importURL = strings.Split(importURL, "#")[0]
		}
		importChecksum = splitted[1]
	}

	o, err := openuri.Open(importURL)
	if err != nil {
		return nil, err
	}
	defer o.Close()

	b, _ := ioutil.ReadAll(o)
	if importChecksum != "" {
		sum := md5.Sum(b)
		hash := hex.EncodeToString(sum[:])

		if importChecksum != hash {
			return nil, fmt.Errorf("md5sum of %s - sum: %s differs from expected: %s", importURL, hash,
				importChecksum)
		}
	}

	return b, nil
}

var LoadedImports map[string]bool

func init() {
	LoadedImports = make(map[string]bool)
}

func LoadConfig(config *Config, urlFetcher func(url string) ([]byte, error)) error {
	for _, importUrl := range config.Imports {
		if _, ok := LoadedImports[importUrl]; ok {
			log.Warnf("item: %s already imported, skipping", importUrl)
			continue
		}

		fetched, err := urlFetcher(importUrl)
		if err != nil {
			return err
		}
		var importedConfig Config

		err = yaml.Unmarshal(fetched, &importedConfig)
		if err != nil {
			return err
		}

		err = LoadConfig(&importedConfig, urlFetcher)
		if err != nil {
			return err
		}

		for name, collection := range importedConfig.Collections {
			if _, ok := config.Collections[name]; ok {
				log.Warnf("collection with name %s already exists, not added", name)
				continue
			}
			if err := collection.SetDefaults(); err != nil {
				return err
			}
			config.Collections[name] = collection
		}

		LoadedImports[importUrl] = true
	}

	for name, collection := range config.Collections {
		if err := collection.SetDefaults(); err != nil {
			return err
		}
		log.Infof("collection: %s added", name)
		config.Collections[name] = collection
	}

	return nil
}

func NewConfigFromFile(path string) (*Config, error) {
	var config Config

	readConfig, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(readConfig, &config)
	if err != nil {
		return nil, err
	}

	if err := LoadConfig(&config, fetchImport); err != nil {
		return nil, err
	}

	return &config, nil
}
