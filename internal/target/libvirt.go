// Package target generates VM definitions for the destination hypervisor.
package target

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/JustHumanz/az2qemu/internal/model"
)

var libvirtTmpl = template.Must(template.New("domain").Funcs(template.FuncMap{
	"letter": func(i int) string { return string(rune('a' + i)) },
}).Parse(`<domain type='kvm'>
  <name>{{.VM.Name}}</name>
  <memory unit='MiB'>{{.VM.MemoryMB}}</memory>
  <vcpu>{{.VM.CPUs}}</vcpu>
  <os{{if .UEFI}} firmware='efi'{{end}}>
    <type arch='x86_64' machine='q35'>hvm</type>
    <boot dev='hd'/>
  </os>
  <features><acpi/><apic/></features>
  <cpu mode='host-passthrough'/>
  <devices>
{{- range $i, $d := .VM.Disks}}
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='{{$d.Path}}'/>
      <target dev='vd{{letter $i}}' bus='virtio'/>
    </disk>
{{- end}}
{{- range .VM.NICs}}
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
{{- end}}
    <serial type='pty'><target port='0'/></serial>
    <console type='pty'><target type='serial' port='0'/></console>
    <graphics type='vnc' autoport='yes'/>
    <video><model type='virtio'/></video>
    <rng model='virtio'><backend model='random'>/dev/urandom</backend></rng>
  </devices>
</domain>
`))

// WriteLibvirtXML writes <vm>.xml into dir and returns its path.
func WriteLibvirtXML(vm *model.VM, dir string) (string, error) {
	out := filepath.Join(dir, vm.Name+".xml")
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()
	for i := range vm.Disks {
		abs, err := filepath.Abs(vm.Disks[i].Path)
		if err != nil {
			return "", err
		}
		vm.Disks[i].Path = abs
	}
	err = libvirtTmpl.Execute(f, struct {
		VM   *model.VM
		UEFI bool
	}{vm, vm.Firmware == model.UEFI})
	if err != nil {
		return "", fmt.Errorf("render libvirt xml: %w", err)
	}
	return out, nil
}
