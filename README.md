# az2qemu

Migrate Azure VMs to QEMU/KVM (libvirt).

Pipeline: Azure snapshot -> azcopy download (VHD) -> `qemu-img` to qcow2 -> `virt-customize` guest adaptation -> libvirt XML.

## Usage

    az2qemu -resource-group RG -vm NAME -out ./out
    virsh define ./out/NAME.xml

Requires `az`, `azcopy`, `qemu-img`, and `virt-customize` (libguestfs).

## Status

MVP: Linux, managed disks, Gen1 (BIOS) and Gen2 (UEFI). Not yet supported: Windows, unmanaged disks, ADE/confidential disks, NIC/MAC mapping, boot validation.
