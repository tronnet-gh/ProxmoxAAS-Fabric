package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
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

	println(token, secret)

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

// Gets and returns a Node's CPU, memory, swap, and Hardware (PCI) resources
func (pve ProxmoxClient) Node(nodeName string) (Host, error) {
	host := Host{}
	host.Hardware = make(map[string]PVEDevice)

	node, err := pve.client.Node(context.Background(), nodeName)
	if err != nil {
		return host, err
	}

	devices := []PVEDevice{}
	err = pve.client.Get(context.Background(), fmt.Sprintf("/nodes/%s/hardware/pci", nodeName), &devices)
	if err != nil {
		return host, err
	}

	vms, err := node.VirtualMachines(context.Background())
	if err != nil {
		return host, err
	}

	cts, err := node.Containers(context.Background())
	if err != nil {
		return host, err
	}

	// temporary helper which maps supersystem devices to each contained subsystem
	// eg 0000:00:05 -> [0000:00:05.0, 0000:00:05.1, 0000:00:05.2, 0000:00:05.3, ...]
	DeviceSubsystemMap := make(map[string][]string)
	for _, device := range devices {
		host.Hardware[device.BusID] = device
		SupersystemID := strings.Split(device.BusID, ".")[0]
		DeviceSubsystemMap[SupersystemID] = append(DeviceSubsystemMap[SupersystemID], device.BusID)
	}

	host.Name = node.Name
	host.Cores.Total = int64(node.CPUInfo.CPUs)
	host.Memory.Total = int64(node.Memory.Total)
	host.Swap.Total = int64(node.Swap.Total)

	for _, vm := range vms {
		vm, err := node.VirtualMachine(context.Background(), int(vm.VMID))
		if err != nil {
			return host, err
		}

		host.Cores.Reserved += int64(vm.VirtualMachineConfig.Cores)
		host.Memory.Reserved += int64(vm.VirtualMachineConfig.Memory * MiB)

		MarshallVirtualMachineConfig(vm.VirtualMachineConfig)

		for _, v := range vm.VirtualMachineConfig.HostPCIs {
			HostPCIBusID := strings.Split(v, ",")[0]
			if device, ok := host.Hardware[HostPCIBusID]; ok { // is a specific subsystem of device
				device.Reserved = true
				host.Hardware[HostPCIBusID] = device
			} else if SubsystemBusIDs, ok := DeviceSubsystemMap[HostPCIBusID]; ok { // is a supersystem device containing multiple subsystems
				for _, SubsystemBusID := range SubsystemBusIDs {
					device := host.Hardware[SubsystemBusID]
					device.Reserved = true
					host.Hardware[SubsystemBusID] = device
				}
			}
		}
	}

	for _, ct := range cts {
		ct, err := node.Container(context.Background(), int(ct.VMID))
		if err != nil {
			return host, err
		}

		host.Cores.Reserved += int64(ct.ContainerConfig.Cores)
		host.Memory.Reserved += int64(ct.ContainerConfig.Memory * MiB)
		host.Swap.Reserved += int64(ct.ContainerConfig.Swap * MiB)
	}

	host.Cores.Free = host.Cores.Total - host.Cores.Reserved
	host.Memory.Free = host.Memory.Total - host.Memory.Reserved
	host.Swap.Free = host.Swap.Total - host.Swap.Reserved

	return host, err
}
