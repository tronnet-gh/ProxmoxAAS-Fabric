package app

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

const MiB = 1024 * 1024

type Config struct {
	ListenPort int `json:"listenPort"`
	PVE        struct {
		URL   string `json:"url"`
		Token struct {
			USER   string `json:"user"`
			REALM  string `json:"realm"`
			ID     string `json:"id"`
			Secret string `json:"uuid"`
		}
	}
	ReloadInterval int `json:"rebuildInterval"`
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

// returns if a device pcie bus id is a super device or subsystem device
//
// subsystem devices always has the format xxxx:yy.z, whereas super devices have the format xxxx:yy
//
// returns true if BusID has format xxxx:yy
func DeviceBusIDIsSuperDevice(BusID DeviceID) bool {
	return !strings.ContainsRune(string(BusID), '.')
}
