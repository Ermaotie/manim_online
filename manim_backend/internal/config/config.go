package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	OpenAI OpenAIConfig
	Manim  ManimConfig
	Redis  RedisConfig
	MySQL  MySQLConfig
}

type OpenAIConfig struct {
	APIKey  string
	BaseURL string
}

type ManimConfig struct {
	PythonPath    string
	MaxConcurrent int
	Timeout       int
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}
