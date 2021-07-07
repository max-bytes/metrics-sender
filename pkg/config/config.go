package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

func LoadConfig(configFile string) (*Configuration, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// defaults
	var cfg Configuration = Configuration{
		ProcessIntervalSeconds: 5,
		RereadFolderSeconds:    180,
	}
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
	SourceFolder           string              `yaml:"sourceFolder"`
	ProcessIntervalSeconds time.Duration       `yaml:"processIntervalSeconds"`
	RereadFolderSeconds    time.Duration       `yaml:"rereadFolderSeconds"`
	LogLevel               string              `yaml:"logLevel"`
	LogFile                string              `yaml:"logFile"`
	Influx                 ConfigurationInflux `yaml:"influx"`
}
