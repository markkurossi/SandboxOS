//
// shell.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"syscall/js"
	"time"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/network"
	"github.com/markkurossi/blackbox-os/kernel/process"
)

type Builtin struct {
	Name string
	Cmd  func(p *process.Process, args []string)
}

var builtin []Builtin

func cmd_help(p *process.Process, args []string) {
	fmt.Fprintf(p.Stdout, "Available commands are:\n")
	for _, cmd := range builtin {
		fmt.Fprintf(p.Stdout, "  %s\n", cmd.Name)
	}
}

func cmd_ssh(p *process.Process, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(p.Stdout, "Usage: ssh [user@]host[:port]\n")
		return
	}
	addr := "localhost:2252"
	conn, err := network.DialTimeout(addr, 5*time.Second)
	if err != nil {
		fmt.Fprintf(p.Stderr, "Dial failed: %s\n", err)
		return
	}
	if true {
		go func() {
			var buf [1024]byte
			for {
				n, err := conn.Read(buf[:])
				if err != nil {
					return
				}
				fmt.Fprintf(p.Stdout, "conn:\n%s", hex.Dump(buf[:n]))

			}
			fmt.Fprintf(p.Stdout, "Connection to %s closed\n", addr)
			conn.Close()
		}()
	}
}

func init() {
	builtin = []Builtin{
		Builtin{
			Name: "alert",
			Cmd: func(p *process.Process, args []string) {
				if len(args) < 2 {
					fmt.Fprintf(p.Stdout, "Usage: alert msg\n")
					return
				}
				js.Global().Get("alert").Invoke(strings.Join(args[1:], " "))
			},
		},
		Builtin{
			Name: "halt",
			Cmd: func(p *process.Process, args []string) {
				fmt.Fprintf(p.Stdout, "System shutting down...\n")
				control.Halt()
			},
		},
		Builtin{
			Name: "help",
			Cmd:  cmd_help,
		},
		Builtin{
			Name: "ssh",
			Cmd:  cmd_ssh,
		},
	}
}

func readLine(in io.Reader) []string {
	var buf [1024]byte
	var line string

	for {
		n, _ := in.Read(buf[:])
		if n == 0 {
			break
		}
		line += string(buf[:n])
		if buf[n-1] == '\n' {
			break
		}
	}
	return strings.Split(strings.TrimSpace(line), " ")
}

func Shell(p *process.Process) {
	for control.HasPower {
		fmt.Fprintf(p.Stdout, "root $ ")
		args := readLine(p.Stdin)
		if len(args) == 0 || len(args[0]) == 0 {
			continue
		}

		var found bool

		for _, cmd := range builtin {
			if args[0] == cmd.Name {
				cmd.Cmd(p, args)
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(p.Stderr, "Unknown command '%s'\n", args[0])
		}
	}
}
