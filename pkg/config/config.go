package config

import "fmt"

type AppConfig struct {
	SecretKey  string
	DBEngine   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	Workers    int
}

func (conf *AppConfig) DBUri() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True", conf.DBUser, conf.DBPassword, conf.DBHost, conf.DBPort, conf.DBName)
}

func (conf *AppConfig) PostgresUri() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", conf.DBHost, conf.DBPort, conf.DBUser, conf.DBPassword, conf.DBName)
}

func (conf *AppConfig) IsPostgreSQL() bool {
	return conf.DBEngine == "postgresql" || conf.DBEngine == "postgres"
}
