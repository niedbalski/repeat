package main

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

var MockConfigImport = `
import:
  - https://raw.githubusercontent.com/niedbalski/repeat/master/example_metrics.yaml#md5sum=6c5b5d8fafd343d5cf452a7660ad9dd1
collections:
  testing:
    command: ps aux
  process_list:
    command: ps auxh
    run-every: 2s
    exit-codes: any
`

var MockConfigNoImport = `
collections:
  testing:
    command: ps aux
  process_list:
    command: ps auxh
    run-every: 2s
    exit-codes: any
`

var MockConfigNoImportNewCollections = `
collections:
  felipestrolo:
    command: ps aux
  talks:
    command: ps auxh
    run-every: 2s
    exit-codes: any
`

func TestLoadConfigMatchingCollections(t *testing.T) {
	var config Config
	_ = yaml.Unmarshal([]byte(MockConfigImport), &config)

	err := LoadConfig(&config, func(url string) ([]byte, error) {
		return []byte(MockConfigNoImport), nil
	})

	assert.Nil(t, err)
	assert.NotNil(t, config)
	assert.Len(t, config.Collections, 2)
}

func TestLoadConfigMultipleImports(t *testing.T) {
	var config Config
	_ = yaml.Unmarshal([]byte(MockConfigImport), &config)

	LoadedImports = make(map[string]bool)
	err := LoadConfig(&config, func(url string) ([]byte, error) {
		return []byte(MockConfigNoImportNewCollections), nil
	})

	assert.Nil(t, err)
	assert.NotNil(t, config)
	assert.Len(t, config.Collections, 4)
}
