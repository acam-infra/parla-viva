package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vivaui/viva/tools/winres"
)

func main() {
	exePath := flag.String("exe", "", "path to Windows .exe to patch")
	icoPath := flag.String("ico", "assets/icon.ico", "path to Windows .ico file")
	version := flag.String("version", "0.5.0", "product version")
	flag.Parse()

	if *exePath == "" {
		fmt.Fprintln(os.Stderr, "usage: parla-winres -exe <path.exe> -ico <icon.ico> -version 0.5.0")
		os.Exit(2)
	}
	exeBytes, err := os.ReadFile(*exePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read exe: %v\n", err)
		os.Exit(1)
	}
	icoBytes, err := os.ReadFile(*icoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read ico: %v\n", err)
		os.Exit(1)
	}
	patched, err := winres.Patch(exeBytes, icoBytes, winres.Info{
		Name:        "Parla Viva",
		Description: "Parla Viva Encrypted Messenger & Agent CLI",
		Version:     *version,
		Publisher:   "acam-infra",
		File:        "parla-viva.exe",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "winres patch: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*exePath, patched, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "write patched exe: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Embedded icon & version (%s) into %s (%d bytes)\n", *version, *exePath, len(patched))
}
