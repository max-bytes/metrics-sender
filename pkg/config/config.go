package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

func LoadConfig(configFile string) (*Configuration, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Configuration
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

type ConfigurationInflux struct {
	URL      string `yaml:"url"`
	Database string `yaml:"database"`
	GZip     bool   `yaml:"gzip"`
}

type Configuration struct {
	SourceFolder string              `yaml:"sourceFolder"`
	LogLevel     string              `yaml:"logLevel"`
	LogFile      string              `yaml:"logFile"`
	Influx       ConfigurationInflux `yaml:"influx"`
}
