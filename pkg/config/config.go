package config

import "fmt"

type AppConfig struct {
	SecretKey  string
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
