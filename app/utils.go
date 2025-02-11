package app

import (
	"encoding/json"
	"log"
	"os"

	"github.com/luthermonson/go-proxmox"
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

func MarshallVirtualMachineConfig(v *proxmox.VirtualMachineConfig) {
	v.HostPCIs = make(map[string]string)
	v.HostPCIs["hostpci0"] = v.HostPCI0
	v.HostPCIs["hostpci1"] = v.HostPCI1
	v.HostPCIs["hostpci2"] = v.HostPCI2
	v.HostPCIs["hostpci3"] = v.HostPCI3
	v.HostPCIs["hostpci4"] = v.HostPCI4
	v.HostPCIs["hostpci5"] = v.HostPCI5
	v.HostPCIs["hostpci6"] = v.HostPCI6
	v.HostPCIs["hostpci7"] = v.HostPCI7
	v.HostPCIs["hostpci8"] = v.HostPCI8
	v.HostPCIs["hostpci9"] = v.HostPCI9
}
