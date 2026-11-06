package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JustHumanz/az2qemu/internal/pipeline"
	"github.com/JustHumanz/az2qemu/internal/source/azure"
)

func main() {
	rg := flag.String("resource-group", "", "Azure resource group (required)")
	vm := flag.String("vm", "", "Azure VM name (required)")
	sub := flag.String("subscription", "", "Azure subscription (optional)")
	out := flag.String("out", "./out", "output directory")
	skipAdapt := flag.Bool("skip-adapt", false, "skip guest adaptation (virt-customize)")
	keepVHD := flag.Bool("keep-vhd", false, "keep downloaded VHD files")
	flag.Parse()

	if *rg == "" || *vm == "" {
		flag.Usage()
		os.Exit(2)
	}
	src, err := azure.New(*rg, *vm, *sub)
	if err != nil {
		fatal(err)
	}
	if err := pipeline.Migrate(src, pipeline.Options{OutDir: *out, SkipAdapt: *skipAdapt, KeepVHD: *keepVHD}); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
