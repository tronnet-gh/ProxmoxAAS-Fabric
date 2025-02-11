package app

import (
	"fmt"
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

	cluster.Hosts = make(map[string]*Host)

	// get all nodes
	nodes, err := cluster.pve.Nodes()
	if err != nil {
		return err
	}
	// for each node:
	for _, hostName := range nodes {
		// rebuild node
		err := cluster.RebuildHost(hostName)
		if err != nil {
			return err
		}
	}

	return nil
}

// get a node in the cluster
func (cluster *Cluster) GetHost(hostName string) (*Host, error) {
	host_ch := make(chan *Host)
	err_ch := make(chan error)

	go func() {
		// aquire cluster lock
		cluster.lock.Lock()
		defer cluster.lock.Unlock()
		// get host
		host, ok := cluster.Hosts[hostName]
		if !ok {
			host_ch <- nil
			err_ch <- fmt.Errorf("%s not in cluster", hostName)
		}
		// aquire host lock to wait in case of a concurrent write
		host.lock.Lock()
		defer host.lock.Unlock()

		host_ch <- host
		err_ch <- nil
	}()

	host := <-host_ch
	err := <-err_ch
	return host, err
}

func (cluster *Cluster) RebuildHost(hostName string) error {
	host, err := cluster.pve.Node(hostName)
	if err != nil {
		return err
	}

	// aquire lock on host, release on return
	host.lock.Lock()
	defer host.lock.Unlock()

	cluster.Hosts[hostName] = host

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

func (host *Host) GetInstance(vmid uint) (*Instance, error) {
	instance_ch := make(chan *Instance)
	err_ch := make(chan error)

	go func() {
		// aquire host lock
		host.lock.Lock()
		defer host.lock.Unlock()
		// get instance
		instance, ok := host.Instances[vmid]
		if !ok {
			instance_ch <- nil
			err_ch <- fmt.Errorf("vmid %d not in host %s", vmid, host.Name)
		}
		// aquire instance lock to wait in case of a concurrent write
		instance.lock.Lock()
		defer instance.lock.Unlock()

		instance_ch <- instance
		err_ch <- nil
	}()

	instance := <-instance_ch
	err := <-err_ch
	return instance, err
}

func (host *Host) RebuildVM(vmid uint) error {
	instance, err := host.VirtualMachine(vmid)
	if err != nil {
		return err
	}

	// aquire lock on instance, release on return
	instance.lock.Lock()
	defer instance.lock.Unlock()

	host.Instances[vmid] = instance

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

func (host *Host) RebuildCT(vmid uint) error {
	instance, err := host.Container(vmid)
	if err != nil {
		return err
	}

	// aquire lock on instance, release on return
	instance.lock.Lock()
	defer instance.lock.Unlock()

	host.Instances[vmid] = instance

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

	volume, _, _, err := GetVolumeInfo(host, volumeDataString)
	if err != nil {
		return err
	}

	instance.Volumes[volid] = volume

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

	instance.Nets[uint(idnum)] = netinfo

	return nil
}

func (instance *Instance) RebuildDevice(host *Host, deviceid string) error {
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
		devices := []*Device{}
		for k, v := range host.Devices {
			if DeviceBusIDIsSubDevice(k, hostDeviceBusID) {
				v.Reserved = true
				devices = append(devices, v)
			}
		}
		instance.Devices[uint(instanceDeviceBusID)] = devices
	} else {
		devices := []*Device{}
		v := host.Devices[hostDeviceBusID]
		v.Reserved = true
		instance.Devices[uint(instanceDeviceBusID)] = devices
	}

	return nil
}
