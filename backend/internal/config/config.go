package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	App        *AppConfig
	HttpServer *HttpserverConfig
	Postgres   *PostgresConfig
	Logger     *LoggerConfig
	Upload     *UploadConfig
}

func parseConfigPath() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return filepath.Join(wd)
}

func InitConfig() *Config {
	configPath := parseConfigPath()
	viper.AddConfigPath(configPath)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	return &Config{
		App:        initAppConfig(),
		HttpServer: initHttpserverConfig(),
		Postgres:   initPostgresConfig(),
		Logger:     initLoggerConfig(),
		Upload:     initUploadConfig(),
	}
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Postgres.Host, c.Postgres.Port, c.Postgres.Username, c.Postgres.Password, c.Postgres.DbName, c.Postgres.Sslmode,
	)
}
