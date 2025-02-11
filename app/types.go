package app

import "github.com/luthermonson/go-proxmox"

type Resource struct {
	Reserved uint64
	Free     uint64
	Total    uint64
}

type Host struct {
	Name     string
	Cores    Resource
	Memory   Resource
	Swap     Resource
	Hardware map[string]*HostSuperDevice
	Instance map[uint]*Instance
	node     *proxmox.Node
}

type InstanceType bool

const (
	VM InstanceType = true
	CT InstanceType = false
)

type Instance struct {
	Type           InstanceType
	Name           string
	Proctype       string
	Cores          uint64
	Memory         uint64
	Swap           uint64
	Volume         map[string]*Volume
	Net            map[uint]*Net
	Device         map[uint]*InstanceDevice
	config         interface{}
	configDisks    map[string]string
	configNets     map[string]string
	configHostPCIs map[string]string
	proxmox.ContainerInterface
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

type InstanceDevice struct {
	Device []*HostDevice
	PCIE   bool
}

type HostSuperDevice struct {
	BusID      string
	DeviceName string
	VendorName string
	Devices    map[string]*HostDevice
}

type HostDevice struct {
	SubID         string
	SubDeviceName string
	SubVendorName string
	Reserved      bool
}
