package config

import (
	"log"

	"github.com/spf13/viper"
)

type HttpserverConfig struct {
	Host                 string `mapstructure:"HTTP_SERVER_HOST"`
	Port                 int    `mapstructure:"HTTP_SERVER_PORT"`
	GracePeriod          int    `mapstructure:"HTTP_SERVER_GRACE_PERIOD"`
	RequestTimeoutPeriod int    `mapstructure:"HTTP_SERVER_REQUEST_TIMEOUT_PERIOD"`
}

func initHttpserverConfig() *HttpserverConfig {
	httpserverConfig := &HttpserverConfig{}

	if err := viper.Unmarshal(&httpserverConfig); err != nil {
		log.Fatalf("error mapping http server config: %v", err)
	}

	return httpserverConfig
}
