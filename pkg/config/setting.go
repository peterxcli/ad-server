package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Setting struct {
	HttpServer HttpServerConfig `mapstructure:"http-server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Secret     SecretConfig     `mapstructure:"secret"`
}

type HttpServerConfig struct {
	Hostname string `mapstructure:"hostname"`
	Port     string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type SecretConfig struct {
	AccessSecret  string `mapstructure:"access_secret"`
	RefreshSecret string `mapstructure:"refresh_secret"`
}

func NewSetting(filename string) (*Setting, error) {
	config := &Setting{}
	viper.SetConfigFile(filename)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error loading config file: %s", err.Error())
	}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}
	return config, nil
}
