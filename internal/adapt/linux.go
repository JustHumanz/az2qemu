// Package adapt prepares a guest to boot under QEMU/KVM.
package adapt

import (
	"fmt"

	"github.com/JustHumanz/az2qemu/internal/model"
	"github.com/JustHumanz/az2qemu/internal/run"
)

// Guest runs offline adaptation on the OS disk (the first disk).
func Guest(vm *model.VM) error {
	switch vm.OS {
	case model.Linux:
		return linux(vm)
	default:
		return fmt.Errorf("guest adaptation for %s is not implemented yet", vm.OS)
	}
}

// linux uses virt-customize (libguestfs) to remove Azure-specific pieces and
// rebuild the initramfs with virtio drivers.
func linux(vm *model.VM) error {
	if err := run.Require("virt-customize"); err != nil {
		return err
	}
	os := vm.Disks[0].Path
	// Best-effort commands: distro differences make some of them no-ops.
	script := `
set -u
(command -v dracut >/dev/null && dracut -f --add-drivers "virtio_blk virtio_scsi virtio_net virtio_pci" --regenerate-all) || true
if command -v update-initramfs >/dev/null; then
  for m in virtio_blk virtio_scsi virtio_net virtio_pci; do grep -qx "$m" /etc/initramfs-tools/modules || echo "$m" >> /etc/initramfs-tools/modules; done
  update-initramfs -u -k all
fi
rm -f /etc/udev/rules.d/70-persistent-net.rules
rm -f /etc/cloud/cloud.cfg.d/90-azure.cfg /etc/cloud/cloud.cfg.d/10-azure-kvp.cfg
cat > /etc/cloud/cloud.cfg.d/99-az2qemu.cfg <<'EOF'
datasource_list: [ NoCloud, ConfigDrive, None ]
EOF
`
	return run.Stream("virt-customize", "-a", os,
		"--run-command", "touch /.autorelabel || true",
		"--run-command", script)
}
