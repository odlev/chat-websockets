// Package config is a nice package
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Websocket `mapstructure:"websocket"`
}

type Websocket struct {
	Address string `mapstructure:"address"`
	HTMLAddress string `mapstructure:"html_address"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read in config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cfg: %w", err)
	}

	return &cfg, nil
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}
