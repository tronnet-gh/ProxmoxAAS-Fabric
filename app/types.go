package app

import (
	"sync"

	"github.com/luthermonson/go-proxmox"
)

type Cluster struct {
	lock  sync.Mutex
	pve   ProxmoxClient
	Nodes map[string]*Node
}

type Node struct {
	lock      sync.Mutex
	Name      string                   `json:"name"`
	Cores     uint64                   `json:"cores"`
	Memory    uint64                   `json:"memory"`
	Swap      uint64                   `json:"swap"`
	Devices   map[DeviceID]*Device     `json:"devices"`
	Instances map[InstanceID]*Instance `json:"instances"`
	pvenode   *proxmox.Node
}

type InstanceID uint64
type InstanceType string

const (
	VM InstanceType = "VM"
	CT InstanceType = "CT"
)

type Instance struct {
	lock           sync.Mutex
	Type           InstanceType                 `json:"type"`
	Name           string                       `json:"name"`
	Proctype       string                       `json:"cpu"`
	Cores          uint64                       `json:"cores"`
	Memory         uint64                       `json:"memory"`
	Swap           uint64                       `json:"swap"`
	Volumes        map[VolumeID]*Volume         `json:"volumes"`
	Nets           map[NetID]*Net               `json:"nets"`
	Devices        map[InstanceDeviceID]*Device `json:"devices"`
	pveconfig      any
	configDisks    map[string]string
	configNets     map[string]string
	configHostPCIs map[string]string
}

var VolumeTypes = []string{
	"sata",
	"scsi",
	"ide",
	"rootfs",
	"mp",
	"unused",
}

type VolumeID string
type Volume struct {
	Type    string `json:"type"`
	Storage string `json:"storage"`
	Format  string `json:"format"`
	Size    uint64 `json:"size"`
	File    string `json:"file"`
}

type NetID uint64
type Net struct {
	Value string `json:"value"`
	Rate  uint64 `json:"rate"`
	VLAN  uint64 `json:"vlan"`
}

type PVEDevice struct {
	ID                    string `json:"id"`
	Device_Name           string `json:"device_name"`
	Vendor_Name           string `json:"vendor_name"`
	Subsystem_Device_Name string `json:"subsystem_device_name"`
	Subsystem_Vendor_Name string `json:"subsystem_vendor_name"`
}

type DeviceID string
type InstanceDeviceID uint64
type Device struct {
	Device_ID   DeviceID                 `json:"device_id"`
	Device_Name string                   `json:"device_name"`
	Vendor_Name string                   `json:"vendor_name"`
	Functions   map[FunctionID]*Function `json:"functions"`
	Reserved    bool                     `json:"reserved"`
	Value       string
}

type FunctionID string
type Function struct {
	Function_ID   FunctionID `json:"function_id"`
	Function_Name string     `json:"subsystem_device_name"`
	Vendor_Name   string     `json:"subsystem_vendor_name"`
	Reserved      bool       `json:"reserved"`
}
