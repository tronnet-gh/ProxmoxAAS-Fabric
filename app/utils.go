package app

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	ListenPort int `json:"listenPort"`
	PVE        struct {
		Token struct {
			ID     string `json:"id"`
			Secret string `json:"secret"`
		}
	}
}

func GetConfig(configPath string) Config {
	content, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal("Error when opening config file: ", err)
	}
	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatal("Error during parsing config file: ", err)
	}
	return config
}
