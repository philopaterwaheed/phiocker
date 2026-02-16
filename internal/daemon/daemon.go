package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/philopaterwaheed/phiocker/internal/moods"
)

const (
	SocketPath = "/var/run/phiocker.sock"
	BasePath   = "/var/lib/phiocker"
)

type Daemon struct {
	listener net.Listener
	mu       sync.Mutex
}

func New() *Daemon {
	return &Daemon{}
}

func (d *Daemon) Start() error {
	if _, err := os.Stat(SocketPath); err == nil {
		if conn, err := net.Dial("unix", SocketPath); err == nil {
			conn.Close()
			return fmt.Errorf("daemon is already running on %s", SocketPath)
		}
		os.Remove(SocketPath)
	}

	ln, err := net.Listen("unix", SocketPath)
	if err != nil {
		return err
	}
	d.listener = ln
	defer ln.Close()

	fmt.Println("Daemon started, listening on", SocketPath)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}
		go d.handleConnection(conn)
	}
}

type Command struct {
	Type string   `json:"type"`
	Args []string `json:"args"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Output  string `json:"output"`
}

func (d *Daemon) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	var cmd Command
	if err := decoder.Decode(&cmd); err != nil {
		return
	}

	response := d.executeCommand(cmd)
	json.NewEncoder(conn).Encode(response)
}

func (d *Daemon) executeCommand(cmd Command) Response {
	d.mu.Lock()
	defer d.mu.Unlock()


	switch cmd.Type {
	case "run":
		// Capture stdout/stderr
		var outBuf bytes.Buffer
		var errBuf bytes.Buffer

		go moods.Run(cmd.Args, bytes.NewReader(nil), &outBuf, &errBuf)

		output := outBuf.String()
		errMsg := errBuf.String()

		if errMsg != "" {
			return Response{Status: "success", Output: output, Message: "Stderr: " + errMsg}
		}
		return Response{Status: "success", Output: output}

	case "list":
		// We can capture stdout of moods.ListContainers
		output := captureOutput(func() {
			moods.ListContainers(BasePath)
		})
		return Response{Status: "success", Output: output}

	case "create":
		if len(cmd.Args) < 1 {
			return Response{Status: "error", Message: "missing generator file"}
		}
		output := captureOutput(func() {
			moods.Create(cmd.Args[0], BasePath)
		})
		return Response{Status: "success", Output: output}

	case "delete":
		if len(cmd.Args) < 1 {
			return Response{Status: "error", Message: "missing args for delete"}
		}
		output := captureOutput(func() {
			if cmd.Args[0] == "all" {
				moods.DeleteAllContainers(BasePath)
			} else {
				moods.DeleteContainer(cmd.Args[0], BasePath)
			}
		})
		return Response{Status: "success", Output: output}

	default:
		return Response{Status: "error", Message: "unknown command"}
	}
}

func captureOutput(f func()) string {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	var buf strings.Builder
	_, _ = io.Copy(&buf, r)
	return buf.String()
}
