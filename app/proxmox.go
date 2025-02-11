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

func NewClient(token string, secret string) ProxmoxClient {
	HTTPClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client := proxmox.NewClient("https://pve.tronnet.net/api2/json",
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
func (pve ProxmoxClient) Node(nodeName string) (Host, error) {
	host := Host{}
	host.Devices = make(map[string]*Device)
	host.Instances = make(map[uint]*Instance)

	node, err := pve.client.Node(context.Background(), nodeName)
	if err != nil {
		return host, err
	}

	devices := []Device{}
	err = pve.client.Get(context.Background(), fmt.Sprintf("/nodes/%s/hardware/pci", nodeName), &devices)
	if err != nil {
		return host, err
	}

	for _, device := range devices {
		host.Devices[device.BusID] = &device
	}

	host.Name = node.Name
	host.Cores.Total = uint64(node.CPUInfo.CPUs)
	host.Memory.Total = uint64(node.Memory.Total)
	host.Swap.Total = uint64(node.Swap.Total)
	host.node = node

	return host, err
}

// Get all VM IDs on specified host
func (host Host) VirtualMachines() ([]uint, error) {
	vms, err := host.node.VirtualMachines(context.Background())
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
func (host Host) VirtualMachine(VMID uint) (Instance, error) {
	instance := Instance{}
	vm, err := host.node.VirtualMachine(context.Background(), int(VMID))
	if err != nil {
		return instance, err
	}

	config := vm.VirtualMachineConfig
	instance.configHostPCIs = config.MergeHostPCIs()
	instance.configNets = config.MergeNets()
	instance.configDisks = MergeVMDisksAndUnused(config)

	instance.config = config
	instance.Type = VM

	instance.Name = vm.Name
	instance.Proctype = vm.VirtualMachineConfig.CPU
	instance.Cores = uint64(vm.VirtualMachineConfig.Cores)
	instance.Memory = uint64(vm.VirtualMachineConfig.Memory) * MiB
	instance.Volumes = make(map[string]*Volume)
	instance.Nets = make(map[uint]*Net)
	instance.Devices = make(map[uint][]*Device)

	return instance, nil
}

func MergeVMDisksAndUnused(vmc *proxmox.VirtualMachineConfig) map[string]string {
	mergedDisks := vmc.MergeDisks()
	for k, v := range vmc.MergeUnuseds() {
		mergedDisks[k] = v
	}
	return mergedDisks
}

// Get all CT IDs on specified host
func (host Host) Containers() ([]uint, error) {
	cts, err := host.node.Containers(context.Background())
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
func (host Host) Container(VMID uint) (Instance, error) {
	instance := Instance{}
	ct, err := host.node.Container(context.Background(), int(VMID))
	if err != nil {
		return instance, err
	}

	config := ct.ContainerConfig
	instance.configHostPCIs = make(map[string]string)
	instance.configNets = config.MergeNets()
	instance.configDisks = MergeCTDisksAndUnused(config)

	instance.config = config
	instance.Type = CT

	instance.Name = ct.Name
	instance.Cores = uint64(ct.ContainerConfig.Cores)
	instance.Memory = uint64(ct.ContainerConfig.Memory) * MiB
	instance.Swap = uint64(ct.ContainerConfig.Swap) * MiB
	instance.Volumes = make(map[string]*Volume)
	instance.Nets = make(map[uint]*Net)

	return instance, nil
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

// get volume fornmat, size, volumeid, and storageid from instance volume data string (eg: local:100/vm-100-disk-0.raw ... )
func GetVolumeInfo(host Host, volume string) (Volume, string, string, error) {
	volumeData := Volume{}

	storageID := strings.Split(volume, ":")[0]
	volumeID := strings.Split(volume, ",")[0]
	storage, err := host.node.Storage(context.Background(), storageID)
	if err != nil {
		return volumeData, volumeID, storageID, nil
	}

	content, err := storage.GetContent(context.Background())
	if err != nil {
		return volumeData, volumeID, storageID, nil
	}

	for _, c := range content {
		if c.Volid == volumeID {
			volumeData.Format = c.Format
			volumeData.Size = uint64(c.Size)
			volumeData.Volid = volumeID
		}
	}

	return volumeData, volumeID, storageID, nil
}

func GetNetInfo(net string) (Net, error) {
	n := Net{}

	for _, val := range strings.Split(net, ",") {
		if strings.HasPrefix(val, "rate=") {
			rate, err := strconv.ParseUint(strings.TrimPrefix(val, "rate="), 10, 64)
			if err != nil {
				return n, err
			}
			n.Rate = rate
		} else if strings.HasPrefix(val, "tag=") {
			vlan, err := strconv.ParseUint(strings.TrimPrefix(val, "tag="), 10, 64)
			if err != nil {
				return n, err
			}
			n.VLAN = vlan
		}
	}

	return n, nil
}
