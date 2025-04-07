package app

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

func (cluster *Cluster) Init(pve ProxmoxClient) {
	cluster.pve = pve
}

func (cluster *Cluster) Sync() error {
	// aquire lock on cluster, release on return
	cluster.lock.Lock()
	defer cluster.lock.Unlock()

	cluster.Nodes = make(map[string]*Node)

	// get all nodes
	nodes, err := cluster.pve.Nodes()
	if err != nil {
		return err
	}
	// for each node:
	for _, hostName := range nodes {
		// rebuild node
		err := cluster.RebuildHost(hostName)
		if err != nil { // if an error was encountered, continue and log the error
			log.Print(err.Error())
			continue
		}
	}

	return nil
}

// get a node in the cluster
func (cluster *Cluster) GetNode(hostName string) (*Node, error) {
	host_ch := make(chan *Node)
	err_ch := make(chan error)

	go func() {
		// aquire cluster lock
		cluster.lock.Lock()
		defer cluster.lock.Unlock()
		// get host
		host, ok := cluster.Nodes[hostName]
		if !ok {
			host_ch <- nil
			err_ch <- fmt.Errorf("%s not in cluster", hostName)
		} else {
			// aquire host lock to wait in case of a concurrent write
			host.lock.Lock()
			defer host.lock.Unlock()

			host_ch <- host
			err_ch <- nil
		}
	}()

	host := <-host_ch
	err := <-err_ch

	return host, err
}

func (cluster *Cluster) RebuildHost(hostName string) error {
	host, err := cluster.pve.Node(hostName)
	if err != nil { // host is probably down or otherwise unreachable
		return fmt.Errorf("error retrieving %s: %s, possibly down?", hostName, err.Error())
	}

	// aquire lock on host, release on return
	host.lock.Lock()
	defer host.lock.Unlock()

	cluster.Nodes[hostName] = host

	// get node's VMs
	vms, err := host.VirtualMachines()
	if err != nil {
		return err

	}
	for _, vmid := range vms {
		err := host.RebuildInstance(VM, vmid)
		if err != nil { // if an error was encountered, continue and log the error
			log.Print(err.Error())
			continue
		}
	}

	// get node's CTs
	cts, err := host.Containers()
	if err != nil {
		return err
	}
	for _, vmid := range cts {
		err := host.RebuildInstance(CT, vmid)
		if err != nil {
			return err
		}
	}

	// check node device reserved by iterating over each function, we will assume that a single reserved function means the device is also reserved
	for _, device := range host.Devices {
		reserved := false
		for _, function := range device.Functions {
			reserved = reserved || function.Reserved
		}
		device.Reserved = reserved
	}

	return nil
}

func (host *Node) GetInstance(vmid uint) (*Instance, error) {
	instance_ch := make(chan *Instance)
	err_ch := make(chan error)

	go func() {
		// aquire host lock
		host.lock.Lock()
		defer host.lock.Unlock()
		// get instance
		instance, ok := host.Instances[InstanceID(vmid)]
		if !ok {
			instance_ch <- nil
			err_ch <- fmt.Errorf("vmid %d not in host %s", vmid, host.Name)
		} else {
			// aquire instance lock to wait in case of a concurrent write
			instance.lock.Lock()
			defer instance.lock.Unlock()

			instance_ch <- instance
			err_ch <- nil
		}
	}()

	instance := <-instance_ch
	err := <-err_ch
	return instance, err
}

func (host *Node) RebuildInstance(instancetype InstanceType, vmid uint) error {
	var instance *Instance
	if instancetype == VM {
		var err error
		instance, err = host.VirtualMachine(vmid)
		if err != nil {
			return fmt.Errorf("error retrieving %d: %s, possibly down?", vmid, err.Error())
		}
	} else if instancetype == CT {
		var err error
		instance, err = host.Container(vmid)
		if err != nil {
			return fmt.Errorf("error retrieving %d: %s, possibly down?", vmid, err.Error())
		}

	}

	// aquire lock on instance, release on return
	instance.lock.Lock()
	defer instance.lock.Unlock()

	host.Instances[InstanceID(vmid)] = instance

	for volid := range instance.configDisks {
		instance.RebuildVolume(host, volid)
	}

	for netid := range instance.configNets {
		instance.RebuildNet(netid)
	}

	for deviceid := range instance.configHostPCIs {
		instance.RebuildDevice(host, deviceid)
	}

	return nil
}

func (instance *Instance) RebuildVolume(host *Node, volid string) error {
	volumeDataString := instance.configDisks[volid]

	volume, err := GetVolumeInfo(host, volumeDataString)
	if err != nil {
		return err
	}

	instance.Volumes[VolumeID(volid)] = volume

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

	instance.Nets[NetID(idnum)] = netinfo

	return nil
}

func (instance *Instance) RebuildDevice(host *Node, deviceid string) error {
	instanceDevice, ok := instance.configHostPCIs[deviceid]
	if !ok { // if device does not exist
		return fmt.Errorf("%s not found in devices", deviceid)
	}

	hostDeviceBusID := DeviceID(strings.Split(instanceDevice, ",")[0])

	idbid, err := strconv.ParseUint(strings.TrimPrefix(deviceid, "hostpci"), 10, 64)
	if err != nil {
		return err
	}
	instanceDeviceBusID := InstanceDeviceID(idbid)

	if DeviceBusIDIsSuperDevice(hostDeviceBusID) {
		instance.Devices[InstanceDeviceID(instanceDeviceBusID)] = host.Devices[DeviceID(hostDeviceBusID)]
		for _, function := range instance.Devices[InstanceDeviceID(instanceDeviceBusID)].Functions {
			function.Reserved = true
		}
	} else {
		// sub function assignment not supported yet
	}

	return nil
}
