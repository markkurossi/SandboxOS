//
// kernel.go
//
// Copyright (c) 2018-2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"syscall/js"

	"github.com/markkurossi/backup/lib/crypto/identity"
	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/persistence"
	"github.com/markkurossi/blackbox-os/commands/shell"
	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/process"
	"github.com/markkurossi/blackbox-os/kernel/tty"
	"github.com/markkurossi/blackbox-os/lib/bbos"
)

var (
	console     = tty.NewConsole()
	IDs         []identity.PrivateKey
	FS          persistence.Accessor
	Zone        *zone.Zone
	locationURL = js.Global().Get("location").Get("href").String()
)

func main() {
	parseParams()

	console.Flush()
	log.SetOutput(console)
	err := runInit()
	if err != nil {
		fmt.Fprintf(console, "Init failed: %s\n", err)
	}

	fmt.Fprintf(console, "\nSystem halted.\n")
}

func runInit() error {
	// Load identities.
	id, err := identity.GetNull()
	if err != nil {
		return fmt.Errorf("Failed to load null identity: %s", err)
	}
	IDs = append(IDs, id)

	// Init filesystem.
	FS, err = persistence.NewHTTP(control.FSRoot)
	if err != nil {
		return fmt.Errorf("Failed to mount filesystem '%s': %s",
			control.FSRoot, err)
	}
	Zone, err = zone.Open(FS, control.FSZone, IDs)
	if err != nil {
		return fmt.Errorf("Failed to open filesystem zone '%s': %s",
			control.FSZone, err)
	}

	process, err := process.NewProcess(console, Zone)
	if err != nil {
		return fmt.Errorf("Failed to create init process: %s", err)
	}
	motd, err := bbos.Open(process, "/etc/motd")
	if err != nil {
		fmt.Fprintf(console, "Black Box OS\n\n")
	} else {
		io.Copy(process.Stdout, motd.Reader())
	}
	fmt.Fprintf(console, "\nType `help' for list of available commands.\n")

	return shell.Shell(process)
}

func parseParams() {
	url, err := url.Parse(locationURL)
	if err != nil {
		fmt.Fprintf(console, "Failed to parse location URL '%s': %s\n",
			locationURL, err)
	}
	url.RawQuery = ""
	url.Fragment = ""

	control.FSRoot = fmt.Sprintf("%sfs", url)
}
