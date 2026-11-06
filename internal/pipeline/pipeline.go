// Package pipeline wires source -> convert -> adapt -> target.
package pipeline

import (
	"fmt"
	"os"

	"github.com/JustHumanz/az2qemu/internal/adapt"
	"github.com/JustHumanz/az2qemu/internal/convert"
	"github.com/JustHumanz/az2qemu/internal/source"
	"github.com/JustHumanz/az2qemu/internal/target"
)

type Options struct {
	OutDir    string
	SkipAdapt bool
	KeepVHD   bool
}

func Migrate(src source.Source, o Options) error {
	if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
		return err
	}
	vm, err := src.Describe()
	if err != nil {
		return fmt.Errorf("describe: %w", err)
	}
	fmt.Printf("VM %s: %d vCPU, %d MiB, %d disk(s), firmware=%s, os=%s\n",
		vm.Name, vm.CPUs, vm.MemoryMB, len(vm.Disks), vm.Firmware, vm.OS)

	defer func() {
		if err := src.Cleanup(); err != nil {
			fmt.Fprintln(os.Stderr, "warning: cleanup failed:", err)
		}
	}()

	if err := src.Export(vm, o.OutDir); err != nil {
		return fmt.Errorf("export: %w", err)
	}
	vhds := make([]string, len(vm.Disks))
	for i, d := range vm.Disks {
		vhds[i] = d.Path
	}
	if err := convert.Disks(vm, o.OutDir); err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	if !o.KeepVHD {
		for _, p := range vhds {
			os.Remove(p)
		}
	}
	if !o.SkipAdapt {
		if err := adapt.Guest(vm); err != nil {
			return fmt.Errorf("adapt: %w", err)
		}
	}
	xml, err := target.WriteLibvirtXML(vm, o.OutDir)
	if err != nil {
		return err
	}
	fmt.Printf("done. Import with: virsh define %s\n", xml)
	return nil
}
