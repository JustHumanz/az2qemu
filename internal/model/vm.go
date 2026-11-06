// Package model holds the provider-neutral VM description.
package model

type Firmware string

const (
	BIOS Firmware = "bios" // Hyper-V generation 1
	UEFI Firmware = "uefi" // Hyper-V generation 2
)

type OSType string

const (
	Linux   OSType = "linux"
	Windows OSType = "windows"
)

type Disk struct {
	Name string
	// Path is the local raw/VHD file once exported, later the converted image.
	Path   string
	SizeGB int
	IsOS   bool
}

type NIC struct {
	MAC string
}

type VM struct {
	Name     string
	OS       OSType
	Firmware Firmware
	CPUs     int
	MemoryMB int
	Disks    []Disk
	NICs     []NIC
}
