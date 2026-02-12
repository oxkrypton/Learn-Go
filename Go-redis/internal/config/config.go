package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Redis  RedisConfig  `mapstructure:"redis"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type RedisConfig struct {
	Host     string `mapstruct:"host"`
	Port     int    `mapstruct:"port"`
	DB       int    `mapstruct:"db"`
	PoolSize int    `mapstruct:"pool_size"`
}

var GlobalConfig *Config

func InitConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	GlobalConfig = &Config{}
	err = viper.Unmarshal(GlobalConfig)
	if err != nil {
		return err
	}
	return nil
}
