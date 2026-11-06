package target

import (
	"os"
	"strings"
	"testing"

	"github.com/JustHumanz/az2qemu/internal/model"
)

func TestWriteLibvirtXML(t *testing.T) {
	vm := &model.VM{
		Name: "vm1", Firmware: model.UEFI, CPUs: 2, MemoryMB: 4096,
		Disks: []model.Disk{{Name: "os", Path: "/tmp/os.qcow2"}, {Name: "d1", Path: "/tmp/d1.qcow2"}},
		NICs:  []model.NIC{{}},
	}
	p, err := WriteLibvirtXML(vm, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	for _, want := range []string{"firmware='efi'", "dev='vda'", "dev='vdb'", "<vcpu>2</vcpu>", "model type='virtio'"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}
