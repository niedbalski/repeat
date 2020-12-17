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
	"sort"
	"strings"
)

type MapValueField struct {
	Name  string `yaml:"name"`
	Type  string `yaml:"type" default:""`
	Index int    `yaml:"field-index"`
}

func (field *MapValueField) Format(values []string) string {
	switch field.Type {
	case "string":
		return "'" + values[field.Index] + "'"
	default:
		return values[field.Index]
	}
}

type MapValue struct {
	Separator       string          `yaml:"field-separator" default:","`
	AppendTimestamp bool            `yaml:"add-timestamp" default:"true"`
	Fields          []MapValueField `yaml:"fields"`
	IgnoreRegex     string          `yaml:"ignore-regex,omitempty"`
}

func (mv *MapValue) SortFieldsByIndex() {
	sort.Slice(mv.Fields, func(i, j int) bool {
		return mv.Fields[i].Index < mv.Fields[j].Index
	})
}

type DBConfig struct {
	MapValues MapValue `yaml:"map-values,omitempty"`
	Name      string   `yaml:"name,omitempty"`
}

func (c *DBConfig) SetDefaults() error {
	c.MapValues.SortFieldsByIndex()
	for i, f := range c.MapValues.Fields {
		for j, ff := range c.MapValues.Fields {
			if i != j && f.Name == ff.Name {
				return fmt.Errorf("duplicate field name: %s - idx: %d and idx: %d, please rename one of them",
					f.Name, i, j)
			}
		}
	}
	return nil
}

type Collection struct {
	Command   string   `yaml:"command"`
	RunEvery  string   `yaml:"run-every" default:"0s"`
	Timeout   string   `yaml:"timeout" default:"0s"`
	BatchSize int      `yaml:"batch-size" default:"1"`
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

	if len(c.Database.MapValues.Fields) > 0 {
		if err := c.Database.SetDefaults(); err != nil {
			return err
		}
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
			if config.Collections == nil {
				config.Collections = make(map[string]Collection)
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
			return fmt.Errorf("cannot set defaults on collection: %s , reason: %s", name, err)
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
