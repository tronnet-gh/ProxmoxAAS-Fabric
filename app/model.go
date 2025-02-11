package app

import (
	"fmt"
	"strconv"
	"strings"
)

type Cluster struct {
	pve   ProxmoxClient
	Hosts map[string]*Host
	//Instance map[uint]*Instance
}

func (cluster *Cluster) Init(pve ProxmoxClient) {
	cluster.pve = pve
}

func (cluster *Cluster) Rebuild() error {
	cluster.Hosts = make(map[string]*Host)
	//cluster.Instance = make(map[uint]*Instance)

	// get all nodes
	nodes, err := cluster.pve.Nodes()
	if err != nil {
		return err
	}
	// for each node:
	for _, hostName := range nodes {
		// rebuild node
		err := cluster.RebuildNode(hostName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cluster *Cluster) RebuildNode(hostName string) error {
	host, err := cluster.pve.Node(hostName)
	if err != nil {
		return err
	}
	cluster.Hosts[hostName] = &host

	// get node's VMs
	vms, err := host.VirtualMachines()
	if err != nil {
		return err
	}
	for _, vmid := range vms {
		err := host.RebuildVM(vmid)
		if err != nil {
			return err
		}
	}

	// get node's CTs
	cts, err := host.Containers()
	if err != nil {
		return err
	}
	for _, vmid := range cts {
		err := host.RebuildCT(vmid)
		if err != nil {
			return err
		}
	}

	return nil
}

func (host *Host) RebuildVM(vmid uint) error {
	instance, err := host.VirtualMachine(vmid)
	if err != nil {
		return err
	}

	host.Instance[vmid] = &instance

	for volid := range instance.configDisks {
		instance.RebuildVolume(host, volid)
	}

	for netid := range instance.configNets {
		instance.RebuildNet(netid)
	}

	for deviceid := range instance.configHostPCIs {
		instance.RebuildDevice(*host, deviceid)
	}

	return nil
}

func (host *Host) RebuildCT(vmid uint) error {
	instance, err := host.Container(vmid)
	if err != nil {
		return err
	}

	host.Instance[vmid] = &instance

	for volid := range instance.configDisks {
		instance.RebuildVolume(host, volid)
	}

	for netid := range instance.configNets {
		instance.RebuildNet(netid)
	}

	return nil
}

func (instance *Instance) RebuildVolume(host *Host, volid string) error {
	volumeDataString := instance.configDisks[volid]

	volume, _, _, err := GetVolumeInfo(*host, volumeDataString)
	if err != nil {
		return err
	}

	instance.Volume[volid] = &volume

	return nil
}

func (instance *Instance) RebuildNet(netid string) error {
	net := instance.configNets[netid]
	idnum, err := strconv.ParseUint(strings.TrimPrefix(netid, "net"), 10, 64)
	if err != nil {
		return err
	}

	netinfo, err := GetNetInfo(net)
	if err != nil {
		return nil
	}

	instance.Net[uint(idnum)] = &netinfo

	return nil
}

func (instance *Instance) RebuildDevice(host Host, deviceid string) error {
	instanceDevice, ok := instance.configHostPCIs[deviceid]
	if !ok { // if device does not exist
		return fmt.Errorf("%s not found in devices", deviceid)
	}

	hostDeviceBusID := strings.Split(instanceDevice, ",")[0]

	instanceDeviceBusID, err := strconv.ParseUint(strings.TrimPrefix(deviceid, "hostpci"), 10, 64)
	if err != nil {
		return err
	}

	if DeviceBusIDIsSuperDevice(hostDeviceBusID) {
		hostSuperDevice := host.Hardware[hostDeviceBusID]
		subDevices := []*HostDevice{}
		for _, v := range hostSuperDevice.Devices {
			subDevices = append(subDevices, v)
		}
		instance.Device[uint(instanceDeviceBusID)] = &InstanceDevice{
			Device: subDevices,
			PCIE:   strings.Contains(instanceDevice, "pcie=1"),
		}
	} else {
		_, hostSubdeviceBusID, err := SplitDeviceBusID(hostDeviceBusID)
		if err != nil {
			return err
		}
		instance.Device[uint(instanceDeviceBusID)] = &InstanceDevice{
			Device: []*HostDevice{
				host.Hardware[hostDeviceBusID].Devices[hostSubdeviceBusID],
			},
			PCIE: strings.Contains(instanceDevice, "pcie=1"),
		}
	}

	return nil
}

func (cluster Cluster) String() string {
	r := ""
	for _, host := range cluster.Hosts {
		r += host.String()
	}
	return r
}

func (host Host) String() string {
	r := fmt.Sprintf("%s\n\tCores:\t%s\n\tMemory:\t%s\n\tSwap:\t%s\n", host.Name, host.Cores, host.Memory, host.Swap)

	r += "\tHardware:\n"

	for _, superdevice := range host.Hardware {
		r += fmt.Sprintf("%s\n", superdevice)
	}

	r += "\tInstances:\n"

	for vmid, vm := range host.Instance {
		r += fmt.Sprintf("\t\t%d: %s\n", vmid, vm)
	}

	return r
}

func (r Resource) String() string {
	return fmt.Sprintf("Totl: %d, Rsrv: %d, Free: %d", r.Total, r.Reserved, r.Free)
}

func (superdevice HostSuperDevice) String() string {
	s := fmt.Sprintf("\t\t%s: %s %s -> ", superdevice.BusID, superdevice.VendorName, superdevice.DeviceName)
	numunused := 0
	for _, device := range superdevice.Devices {
		if device.Reserved {
			s += fmt.Sprintf("%s:(Rsrv %t, %s %s: %s %s)", device.SubID, device.Reserved, superdevice.VendorName, device.SubVendorName, superdevice.DeviceName, device.SubDeviceName)
		} else {
			numunused++
		}
	}
	s += fmt.Sprintf("+%d unreserved subdevices", numunused)
	return s
}

func (i Instance) String() string {
	if i.Type == VM {
		r := fmt.Sprintf("VM, Name: %s, Proctype: %s, Cores: %d, Memory: %d\n", i.Name, i.Proctype, i.Cores, i.Memory)
		for k, v := range i.Volume {
			r += fmt.Sprintf("\t\t\t%s: %s\n", k, v)
		}
		for k, v := range i.Net {
			r += fmt.Sprintf("\t\t\tnet%d: %s\n", k, v)
		}
		for k, v := range i.Device {
			r += fmt.Sprintf("\t\t\thostpci%d: %s\n", k, v)
		}
		return r
	} else {
		r := fmt.Sprintf("CT, Name: %s, Cores: %d, Memory: %d, Swap: %d\n", i.Name, i.Cores, i.Memory, i.Swap)
		for k, v := range i.Volume {
			r += fmt.Sprintf("\t\t\t%s: %s\n", k, v)
		}
		for k, v := range i.Net {
			r += fmt.Sprintf("\t\t\tnet%d: %s\n", k, v)
		}
		return r
	}
}

func (v Volume) String() string {
	return fmt.Sprintf("id: %s, format: %s, size: %d", v.Volid, v.Format, v.Size)
}

func (n Net) String() string {
	return fmt.Sprintf("rate: %d, vlan: %d", n.Rate, n.VLAN)
}

func (d InstanceDevice) String() string {
	r := ""
	for _, v := range d.Device {
		r += fmt.Sprintf("%s:%s ", v.SubVendorName, v.SubDeviceName)
	}
	return r
}
