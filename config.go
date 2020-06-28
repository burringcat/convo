package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
)

type DatabaseConfig struct {
	Type string `toml:"type"`
	Host string `toml:"host"`
	Path string `toml:"path"`
	Auth string `toml:"auth"`
}
func (dc *DatabaseConfig) DatabaseSrc() string {
	return fmt.Sprintf("%s@tcp(%s)%s", dc.Auth, dc.Host, dc.Path)
}
func (dc *DatabaseConfig) DatabaseUrl() string {
	return fmt.Sprintf("%s://%s", dc.Type, dc.DatabaseSrc())
}
type TabsConfig struct {
	Name string `toml:"name"`
	Nodes []string `toml:"nodes"`
}
type ConvoConfig struct {
	Debug bool `toml:"debug"`
	SecretKey string `toml:"secret_key"`
	Database DatabaseConfig `toml:"database"`
	Tabs map[string]TabsConfig `toml:"tabs"`

}
var Config ConvoConfig
var Debug bool
func ParseConfig() error {
	if _, err := toml.DecodeFile("config.toml", &Config); err != nil  {
		return err
	}
	Debug = Config.Debug
	return nil
}