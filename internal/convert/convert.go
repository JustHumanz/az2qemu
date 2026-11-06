// Package convert turns exported VHDs into qcow2 images.
package convert

import (
	"path/filepath"
	"strings"

	"github.com/JustHumanz/az2qemu/internal/model"
	"github.com/JustHumanz/az2qemu/internal/run"
)

// Disks converts every disk to qcow2 in dir and updates Disk.Path.
func Disks(vm *model.VM, dir string) error {
	if err := run.Require("qemu-img"); err != nil {
		return err
	}
	for i := range vm.Disks {
		d := &vm.Disks[i]
		dst := filepath.Join(dir, strings.TrimSuffix(filepath.Base(d.Path), ".vhd")+".qcow2")
		// Azure exports are fixed VHDs; "vpc" is QEMU's name for the format.
		if err := run.Stream("qemu-img", "convert", "-p", "-f", "vpc",
			"-O", "qcow2", d.Path, dst); err != nil {
			return err
		}
		d.Path = dst
	}
	return nil
}
