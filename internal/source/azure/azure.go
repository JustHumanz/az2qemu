// Package azure exports Azure VMs using the az and azcopy CLIs.
package azure

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/JustHumanz/az2qemu/internal/model"
	"github.com/JustHumanz/az2qemu/internal/run"
)

type Azure struct {
	ResourceGroup string
	VMName        string
	Subscription  string

	snapshots []string // names created by Export
	grants    []string
}

func New(rg, vm, sub string) (*Azure, error) {
	if err := run.Require("az", "azcopy"); err != nil {
		return nil, err
	}
	return &Azure{ResourceGroup: rg, VMName: vm, Subscription: sub}, nil
}

func (a *Azure) az(args ...string) (string, error) {
	if a.Subscription != "" {
		args = append(args, "--subscription", a.Subscription)
	}
	return run.Output("az", args...)
}

type vmJSON struct {
	Location        string `json:"location"`
	HardwareProfile struct {
		VMSize string `json:"vmSize"`
	} `json:"hardwareProfile"`
	StorageProfile struct {
		OSDisk struct {
			OSType      string `json:"osType"`
			Name        string `json:"name"`
			DiskSizeGB  int    `json:"diskSizeGb"`
			ManagedDisk *struct {
				ID string `json:"id"`
			} `json:"managedDisk"`
			EncryptionSettings *struct {
				Enabled bool `json:"enabled"`
			} `json:"encryptionSettings"`
		} `json:"osDisk"`
		DataDisks []struct {
			Name        string `json:"name"`
			DiskSizeGB  int    `json:"diskSizeGb"`
			ManagedDisk *struct {
				ID string `json:"id"`
			} `json:"managedDisk"`
		} `json:"dataDisks"`
	} `json:"storageProfile"`
	NetworkProfile struct {
		NetworkInterfaces []struct {
			ID string `json:"id"`
		} `json:"networkInterfaces"`
	} `json:"networkProfile"`
}

type sizeJSON struct {
	Name     string `json:"name"`
	Cores    int    `json:"numberOfCores"`
	MemoryMB int    `json:"memoryInMb"`
}

type diskJSON struct {
	HyperVGeneration string `json:"hyperVGeneration"`
	Encryption       struct {
		Type string `json:"type"`
	} `json:"encryption"`
	EncryptionSettingsCollection *struct {
		Enabled bool `json:"enabled"`
	} `json:"encryptionSettingsCollection"`
}

func (a *Azure) Describe() (*model.VM, error) {
	out, err := a.az("vm", "show", "-g", a.ResourceGroup, "-n", a.VMName, "-o", "json")
	if err != nil {
		return nil, err
	}
	var v vmJSON
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return nil, err
	}
	if v.StorageProfile.OSDisk.ManagedDisk == nil {
		return nil, fmt.Errorf("unmanaged (blob) OS disks are not supported yet")
	}

	vm := &model.VM{Name: a.VMName, Firmware: model.BIOS}
	switch strings.ToLower(v.StorageProfile.OSDisk.OSType) {
	case "windows":
		vm.OS = model.Windows
	default:
		vm.OS = model.Linux
	}

	sizes, err := a.az("vm", "list-sizes", "-l", v.Location, "-o", "json")
	if err != nil {
		return nil, err
	}
	var all []sizeJSON
	if err := json.Unmarshal([]byte(sizes), &all); err != nil {
		return nil, err
	}
	for _, s := range all {
		if s.Name == v.HardwareProfile.VMSize {
			vm.CPUs, vm.MemoryMB = s.Cores, s.MemoryMB
		}
	}
	if vm.CPUs == 0 {
		return nil, fmt.Errorf("could not resolve size %q", v.HardwareProfile.VMSize)
	}

	osd := v.StorageProfile.OSDisk
	vm.Disks = append(vm.Disks, model.Disk{Name: osd.Name, SizeGB: osd.DiskSizeGB, IsOS: true})
	for _, d := range v.StorageProfile.DataDisks {
		if d.ManagedDisk == nil {
			return nil, fmt.Errorf("unmanaged data disk %q is not supported yet", d.Name)
		}
		vm.Disks = append(vm.Disks, model.Disk{Name: d.Name, SizeGB: d.DiskSizeGB})
	}

	for _, d := range vm.Disks {
		dj, err := a.showDisk(d.Name)
		if err != nil {
			return nil, err
		}
		if t := dj.Encryption.Type; strings.Contains(t, "ConfidentialVm") {
			return nil, fmt.Errorf("disk %q uses confidential VM encryption; not migratable", d.Name)
		}
		if dj.EncryptionSettingsCollection != nil && dj.EncryptionSettingsCollection.Enabled {
			return nil, fmt.Errorf("disk %q uses Azure Disk Encryption; decrypt it first", d.Name)
		}
		if d.IsOS && strings.EqualFold(dj.HyperVGeneration, "V2") {
			vm.Firmware = model.UEFI
		}
	}

	for range v.NetworkProfile.NetworkInterfaces {
		vm.NICs = append(vm.NICs, model.NIC{})
	}
	return vm, nil
}

func (a *Azure) showDisk(name string) (*diskJSON, error) {
	out, err := a.az("disk", "show", "-g", a.ResourceGroup, "-n", name, "-o", "json")
	if err != nil {
		return nil, err
	}
	var d diskJSON
	return &d, json.Unmarshal([]byte(out), &d)
}

func (a *Azure) Export(vm *model.VM, dir string) error {
	stamp := time.Now().Unix()
	for i := range vm.Disks {
		d := &vm.Disks[i]
		snap := fmt.Sprintf("az2qemu-%s-%d", d.Name, stamp)
		if len(snap) > 80 {
			snap = snap[:80]
		}
		fmt.Printf("snapshotting %s -> %s\n", d.Name, snap)
		if _, err := a.az("snapshot", "create", "-g", a.ResourceGroup, "-n", snap,
			"--source", d.Name, "-o", "none"); err != nil {
			return err
		}
		a.snapshots = append(a.snapshots, snap)

		sas, err := a.az("snapshot", "grant-access", "-g", a.ResourceGroup, "-n", snap,
			"--duration-in-seconds", "7200", "--access-level", "Read",
			"--query", "accessSas", "-o", "tsv")
		if err != nil {
			return err
		}
		a.grants = append(a.grants, snap)

		dst := filepath.Join(dir, d.Name+".vhd")
		fmt.Printf("downloading %s\n", dst)
		if err := run.Stream("azcopy", "copy", sas, dst, "--log-level", "ERROR"); err != nil {
			return err
		}
		d.Path = dst
	}
	return nil
}

func (a *Azure) Cleanup() error {
	var firstErr error
	keep := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, s := range a.grants {
		_, err := a.az("snapshot", "revoke-access", "-g", a.ResourceGroup, "-n", s, "-o", "none")
		keep(err)
	}
	for _, s := range a.snapshots {
		_, err := a.az("snapshot", "delete", "-g", a.ResourceGroup, "-n", s, "-o", "none")
		keep(err)
	}
	a.grants, a.snapshots = nil, nil
	return firstErr
}
