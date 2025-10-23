package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

// Load 从 YAML 文件加载配置。
func Load(file string) (Config, error) {
	var c Config
	b, err := os.ReadFile(file)
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

// MustLoad 从 YAML 文件加载配置（失败 panic）。
func MustLoad(file string) Config {
	c, err := Load(file)
	if err != nil {
		panic(err)
	}
	return c
}
