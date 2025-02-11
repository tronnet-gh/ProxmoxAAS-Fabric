package app

type PVEBus int

const (
	IDE PVEBus = iota
	SATA
)

type PVEDrive struct{}

type PVEDisk struct{}

type PVENet struct{}

type PVEDevice struct{}

type QEMUInstance struct {
	Name     string
	Proctype string
	Cores    int16
	Memory   int32
	Drive    map[int]PVEDrive
	Disk     map[int]PVEDisk
	Net      map[int]PVENet
	Device   map[int]PVEDevice
}

type LXCInstance struct {
	Name     string
	Cores    int16
	Memory   int32
	Swap     int32
	RootDisk PVEDrive
	MP       map[int]PVEDisk
	Net      map[int]PVENet
}
