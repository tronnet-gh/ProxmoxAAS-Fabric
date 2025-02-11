package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

const MiB = 1024 * 1024

type Config struct {
	ListenPort int `json:"listenPort"`
	PVE        struct {
		Token struct {
			USER   string `json:"user"`
			REALM  string `json:"realm"`
			ID     string `json:"id"`
			Secret string `json:"uuid"`
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

// returns if a device pcie bus id is a super device or subsystem device
//
// subsystem devices always has the format xxxx:yy.z, whereas super devices have the format xxxx:yy
func DeviceBusIDIsSuperDevice(BusID string) bool {
	return !strings.ContainsRune(BusID, '.')
}

// splits a device pcie bus id into super device and subsystem device IDs if possible
func SplitDeviceBusID(BusID string) (string, string, error) {
	if DeviceBusIDIsSuperDevice(BusID) {
		return BusID, "", nil
	} else {
		x := strings.Split(BusID, ".")
		if len(x) != 2 {
			return "", "", fmt.Errorf("BusID: %s contained more than one '.'", BusID)
		} else {
			return x[0], x[1], nil
		}
	}
}
