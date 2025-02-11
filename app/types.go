package app

import (
	"sync"

	"github.com/luthermonson/go-proxmox"
)

type Cluster struct {
	lock  sync.Mutex
	pve   ProxmoxClient
	Hosts map[string]*Host
}

type Host struct {
	lock      sync.Mutex
	Name      string
	Cores     uint64
	Memory    uint64
	Swap      uint64
	Devices   map[string]*Device
	Instances map[uint]*Instance
	pvenode   *proxmox.Node
}

type InstanceType string

const (
	VM InstanceType = "VM"
	CT InstanceType = "CT"
)

type Instance struct {
	lock           sync.Mutex
	Type           InstanceType
	Name           string
	Proctype       string
	Cores          uint64
	Memory         uint64
	Swap           uint64
	Volumes        map[string]*Volume
	Nets           map[uint]*Net
	Devices        map[uint][]*Device
	pveconfig      interface{}
	configDisks    map[string]string
	configNets     map[string]string
	configHostPCIs map[string]string
}

type Volume struct {
	Path   string
	Format string
	Size   uint64
	Volid  string
}

type Net struct {
	Rate uint64
	VLAN uint64
}

type Device struct {
	BusID               string `json:"id"`
	DeviceName          string `json:"device_name"`
	VendorName          string `json:"vendor_name"`
	SubsystemDeviceName string `json:"subsystem_device_name"`
	SubsystemVendorName string `json:"subsystem_vendor_name"`
	Reserved            bool
}
