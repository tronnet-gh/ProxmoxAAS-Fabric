package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/luthermonson/go-proxmox"
)

type ProxmoxClient struct {
	client *proxmox.Client
}

type PVEDevice struct { // used only for requests to PVE
	ID                    string `json:"id"`
	Device_Name           string `json:"device_name"`
	Vendor_Name           string `json:"vendor_name"`
	Subsystem_Device_Name string `json:"subsystem_device_name"`
	Subsystem_Vendor_Name string `json:"subsystem_vendor_name"`
}

func NewClient(url string, token string, secret string) ProxmoxClient {
	HTTPClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client := proxmox.NewClient(url,
		proxmox.WithHTTPClient(&HTTPClient),
		proxmox.WithAPIToken(token, secret),
	)

	return ProxmoxClient{client: client}
}

// Gets and returns the PVE API version
func (pve ProxmoxClient) Version() (proxmox.Version, error) {
	version, err := pve.client.Version(context.Background())
	if err != nil {
		return *version, err
	}
	return *version, err
}

// Gets all Nodes names
func (pve ProxmoxClient) Nodes() ([]string, error) {
	nodes, err := pve.client.Nodes(context.Background())
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, node := range nodes {
		names = append(names, node.Node)
	}

	return names, nil
}

// Gets a Node's resources but does not recursively expand instances
func (pve ProxmoxClient) Node(nodeName string) (*Node, error) {
	host := Node{}
	host.Devices = make(map[DeviceBus]*Device)
	host.Instances = make(map[InstanceID]*Instance)

	node, err := pve.client.Node(context.Background(), nodeName)
	if err != nil {
		return &host, err
	}

	devices := []PVEDevice{}
	err = pve.client.Get(context.Background(), fmt.Sprintf("/nodes/%s/hardware/pci", nodeName), &devices)
	if err != nil {
		return &host, err
	}

	for _, device := range devices {
		x := strings.Split(device.ID, ".")
		if len(x) != 2 { // this should always be true, but skip if not
			continue
		}
		deviceid := DeviceBus(x[0])
		functionid := FunctionID(x[1])
		if _, ok := host.Devices[deviceid]; !ok {
			host.Devices[deviceid] = &Device{
				Device_Bus:  deviceid,
				Device_Name: device.Device_Name,
				Vendor_Name: device.Vendor_Name,
				Functions:   make(map[FunctionID]*Function),
			}
		}
		host.Devices[deviceid].Functions[functionid] = &Function{
			Function_ID:   functionid,
			Function_Name: device.Subsystem_Device_Name,
			Vendor_Name:   device.Subsystem_Vendor_Name,
			Reserved:      false,
		}
	}

	host.Name = node.Name
	host.Cores = uint64(node.CPUInfo.CPUs)
	host.Memory = uint64(node.Memory.Total)
	host.Swap = uint64(node.Swap.Total)
	host.pvenode = node

	return &host, err
}

// Get all VM IDs on specified host
func (host *Node) VirtualMachines() ([]uint, error) {
	vms, err := host.pvenode.VirtualMachines(context.Background())
	if err != nil {
		return nil, err
	}
	ids := []uint{}
	for _, vm := range vms {
		ids = append(ids, uint(vm.VMID))
	}
	return ids, nil
}

// Get a VM's CPU, Memory but does not recursively link Devices, Disks, Drives, Nets
func (host *Node) VirtualMachine(VMID uint) (*Instance, error) {
	instance := Instance{}
	vm, err := host.pvenode.VirtualMachine(context.Background(), int(VMID))
	if err != nil {
		return &instance, err
	}

	config := vm.VirtualMachineConfig
	instance.configHostPCIs = config.MergeHostPCIs()
	instance.configNets = config.MergeNets()
	instance.configDisks = MergeVMDisksAndUnused(config)
	instance.configBoot = config.Boot

	instance.pveconfig = config
	instance.Type = VM

	instance.Name = vm.Name
	instance.Proctype = vm.VirtualMachineConfig.CPU
	instance.Cores = uint64(vm.VirtualMachineConfig.Cores)
	instance.Memory = uint64(vm.VirtualMachineConfig.Memory) * MiB
	instance.Volumes = make(map[VolumeID]*Volume)
	instance.Nets = make(map[NetID]*Net)
	instance.Devices = make(map[DeviceID]*Device)

	return &instance, nil
}

func MergeVMDisksAndUnused(vmc *proxmox.VirtualMachineConfig) map[string]string {
	mergedDisks := vmc.MergeDisks()
	for k, v := range vmc.MergeUnuseds() {
		mergedDisks[k] = v
	}
	return mergedDisks
}

// Get all CT IDs on specified host
func (host *Node) Containers() ([]uint, error) {
	cts, err := host.pvenode.Containers(context.Background())
	if err != nil {
		return nil, err
	}
	ids := []uint{}
	for _, ct := range cts {
		ids = append(ids, uint(ct.VMID))
	}
	return ids, nil
}

// Get a CT's CPU, Memory, Swap but does not recursively link Devices, Disks, Drives, Nets
func (host *Node) Container(VMID uint) (*Instance, error) {
	instance := Instance{}
	ct, err := host.pvenode.Container(context.Background(), int(VMID))
	if err != nil {
		return &instance, err
	}

	config := ct.ContainerConfig
	instance.configHostPCIs = make(map[string]string)
	instance.configNets = config.MergeNets()
	instance.configDisks = MergeCTDisksAndUnused(config)

	instance.pveconfig = config
	instance.Type = CT

	instance.Name = ct.Name
	instance.Cores = uint64(ct.ContainerConfig.Cores)
	instance.Memory = uint64(ct.ContainerConfig.Memory) * MiB
	instance.Swap = uint64(ct.ContainerConfig.Swap) * MiB
	instance.Volumes = make(map[VolumeID]*Volume)
	instance.Nets = make(map[NetID]*Net)

	return &instance, nil
}

func MergeCTDisksAndUnused(cc *proxmox.ContainerConfig) map[string]string {
	mergedDisks := make(map[string]string)
	for k, v := range cc.MergeUnuseds() {
		mergedDisks[k] = v
	}
	for k, v := range cc.MergeMps() {
		mergedDisks[k] = v
	}
	mergedDisks["rootfs"] = cc.RootFS
	return mergedDisks
}

// get volume format, size, volumeid, and storageid from instance volume data string (eg: local:100/vm-100-disk-0.raw ... )
func GetVolumeInfo(host *Node, volume string) (*Volume, error) {
	volumeData := Volume{}

	storageID := strings.Split(volume, ":")[0]
	volumeID := strings.Split(volume, ",")[0]
	storage, err := host.pvenode.Storage(context.Background(), storageID)
	if err != nil {
		return &volumeData, nil
	}

	content, err := storage.GetContent(context.Background())
	if err != nil {
		return &volumeData, nil
	}

	for _, c := range content {
		if c.Volid == volumeID {
			volumeData.Storage = storageID
			volumeData.Format = c.Format
			volumeData.Size = uint64(c.Size)
			volumeData.File = volumeID
		}
	}

	return &volumeData, nil
}

func GetNetInfo(net string) (*Net, error) {
	n := Net{}

	for _, val := range strings.Split(net, ",") {
		if strings.HasPrefix(val, "rate=") {
			rate, err := strconv.ParseUint(strings.TrimPrefix(val, "rate="), 10, 64)
			if err != nil {
				return &n, err
			}
			n.Rate = rate
		} else if strings.HasPrefix(val, "tag=") {
			vlan, err := strconv.ParseUint(strings.TrimPrefix(val, "tag="), 10, 64)
			if err != nil {
				return &n, err
			}
			n.VLAN = vlan
		}
	}

	n.Value = net

	return &n, nil
}
