// Package source defines the interface for cloud/hypervisor providers.
package source

import "github.com/JustHumanz/az2qemu/internal/model"

type Source interface {
	// Describe returns the VM metadata.
	Describe() (*model.VM, error)
	// Export downloads every disk of the VM into dir as a VHD and fills Disk.Path.
	Export(vm *model.VM, dir string) error
	// Cleanup removes temporary cloud resources (snapshots, SAS grants).
	Cleanup() error
}
