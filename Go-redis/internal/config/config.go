package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Redis  RedisConfig  `mapstructure:"redis"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	NATS   NATSConfig   `mapstructure:"nats"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type RedisConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	MaxActive int    `mapstructure:"max_active"`
	MaxIdle   int    `mapstructure:"max_idle"`
}

type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

type NATSConfig struct {
	URL                    string `mapstructure:"url"`
	Stream                 string `mapstructure:"stream"`
	Subject                string `mapstructure:"subject"`
	Consumer               string `mapstructure:"consumer"`
	AckWaitSeconds         int    `mapstructure:"ack_wait_seconds"`
	MaxDeliver             int    `mapstructure:"max_deliver"`
	DuplicateWindowSeconds int    `mapstructure:"duplicate_window_seconds"`
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
