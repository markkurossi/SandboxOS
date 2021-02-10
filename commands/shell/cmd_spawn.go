//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"syscall/js"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/errno"
	"github.com/markkurossi/blackbox-os/kernel/kmsg"
	"github.com/markkurossi/blackbox-os/kernel/process"
)

var (
	syscallSpawn  = js.Global().Get("syscallSpawn")
	syscallResult = js.Global().Get("syscallResult")
	uint8Array    = js.Global().Get("Uint8Array")
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "spawn",
		Cmd:  cmd_spawn,
	})
}

func cmd_spawn(p *process.Process, args []string) {
	var worker js.Value

	onSyscall := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			kmsg.Printf("syscall: invalid arguments: %v\n", args)
			return nil
		}
		event := args[0]

		var id int
		idVal := event.Get("id")
		if !idVal.IsNull() {
			id = idVal.Int()
		}

		switch event.Get("type").String() {
		case "write":
			fd := event.Get("fd").Int()
			dval := event.Get("data")
			offset := event.Get("offset").Int()
			length := event.Get("length").Int()

			data := make([]byte, dval.Length())
			js.CopyBytesToGo(data, dval)

			if offset < 0 || offset+length > len(data) {
				kmsg.Printf("syscall write: id=%d, fd=%d, offset=%d, length=%d",
					id, fd, offset, length)
				syscallResult.Invoke(worker, id, errno.EINVAL, 0)
				return nil
			}

			n, err := p.Stdout.Write(data[offset : offset+length])
			if err != nil {
				kmsg.Printf("syscall write: id=%d, fd=%d => %d %s:\n%s",
					id, fd, n, err, hex.Dump(data))
				syscallResult.Invoke(worker, id, err.Error(), n)
			} else {
				kmsg.Printf("syscall write: id=%d, fd=%d => %d:\n%s",
					id, fd, n, hex.Dump(data))
				syscallResult.Invoke(worker, id, nil, n)
			}

		case "read":
			fd := event.Get("fd").Int()
			length := event.Get("length").Int()

			_ = fd

			data := []byte("Hello, from kernel!")
			if len(data) > length {
				data = data[:length]
			}

			buf := uint8Array.New(len(data))
			js.CopyBytesToJS(buf, data)
			syscallResult.Invoke(worker, id, nil, len(data), buf)

		default:
			kmsg.Printf("syscall: type=%v\n", event.Get("type").String())
			syscallResult.Invoke(worker, id, errno.ENOSYS, 0)
		}

		return nil
	})

	resp, err := http.Get(fmt.Sprintf("%s/bin/echo.wasm", control.BaseURL))
	if err != nil {
		kmsg.Printf("syscall: HTTP error: %s", err)
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		kmsg.Printf("syscall: read error: %s", err)
		return
	}

	code := uint8Array.New(len(data))
	js.CopyBytesToJS(code, data)

	argv := []interface{}{
		onSyscall, code,
	}
	for _, arg := range args {
		argv = append(argv, arg)
	}

	worker = syscallSpawn.Invoke(argv...)
}