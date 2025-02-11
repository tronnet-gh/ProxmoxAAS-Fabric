package app

type Resource struct { // number of virtual cores (usually threads)
	Reserved int64
	Free     int64
	Total    int64
}

type Host struct {
	Name     string
	Cores    Resource
	Memory   Resource
	Swap     Resource
	Storage  map[string]Storage
	Hardware map[string]Device
}

type Storage struct{}

type QEMUInstance struct {
	Name     string
	Proctype string
	Cores    Resource
	Memory   Resource
	Drive    map[int]Volume
	Disk     map[int]Volume
	Net      map[int]Net
	Device   map[int]Device
}

type LXCInstance struct {
	Name     string
	Cores    Resource
	Memory   Resource
	Swap     Resource
	RootDisk Volume
	MP       map[int]Volume
	Net      map[int]Net
}

type Volume struct {
	Format string
	Path   string
	Size   string
	Used   string
}

type Net struct{}

type Device struct {
	BusID               string `json:"id"`
	DeviceName          string `json:"device_name"`
	VendorName          string `json:"vendor_name"`
	SubsystemDeviceName string `json:"subsystem_device_name"`
	SubsystemVendorName string `json:"subsystem_vendor_name"`
	Reserved            bool
}
