package config

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

type Config struct {
	ServerPort string   `json:"port" yaml:"port"`
	Backends   []string `json:"backends" yaml:"backends"`
	RateLimit  `json:"rate_limit" yaml:"rate_limit"`
	Postgres   `json:"postgres" yaml:"postgres"`
}

type RateLimit struct {
	Capacity   int `json:"default_capacity" yaml:"default_capacity"`
	RefillRate int `json:"default_refill_rate" yaml:"default_refill_rate"`
}

type Postgres struct {
	Host     string `json:"host" yaml:"host"`
	Port     string `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Database string `json:"database" yaml:"database"`
}

func LoadConfig(filename string) Config {
	config := Config{}
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	if err = yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
	return config
}
