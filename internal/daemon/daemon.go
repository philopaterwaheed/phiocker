package daemon

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/philopaterwaheed/phiocker/internal/moods"
)

const (
	SocketPath = "/var/run/phiocker.sock"
	BasePath   = "/var/lib/phiocker"
)

type RunningContainer struct {
	Name    string
	PID     int
	Started time.Time
	Process *moods.ContainerProcess
}

type Daemon struct {
	listener   net.Listener
	mu         sync.Mutex
	containers map[string]*RunningContainer
}

func New() *Daemon {
	return &Daemon{
		containers: make(map[string]*RunningContainer),
	}
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

	switch cmd.Type {
	case "run":
		if len(cmd.Args) < 1 {
			return Response{Status: "error", Message: "missing container name"}
		}
		name := cmd.Args[0]
		d.mu.Lock()
		defer d.mu.Unlock()

		if _, exists := d.containers[name]; exists {
			return Response{Status: "error", Message: fmt.Sprintf("container '%s' is already running", name)}
		}

		cp, err := moods.RunDetached(cmd.Args)
		if err != nil {
			return Response{Status: "error", Message: err.Error()}
		}

		rc := &RunningContainer{
			Name:    name,
			PID:     cp.PID(),
			Started: time.Now(),
			Process: cp,
		}
		d.containers[name] = rc

		go func(name string, cp *moods.ContainerProcess) {
			cp.Wait()
			d.mu.Lock()
			delete(d.containers, name)
			d.mu.Unlock()
		}(name, cp)

		return Response{
			Status: "success",
			Output: fmt.Sprintf("Container '%s' started (PID %d)\n", name, cp.PID()),
		}

	case "attach":
		if len(cmd.Args) < 1 {
			return Response{Status: "error", Message: "missing container name"}
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		name := cmd.Args[0]
		rc, exists := d.containers[name]
		if !exists {
			return Response{Status: "error", Message: fmt.Sprintf("container '%s' is not running", name)}
		}

		return Response{
			Status: "success",
			Output: strconv.Itoa(rc.PID),
		}

	case "ps":
		d.mu.Lock()
		defer d.mu.Unlock()
		if len(d.containers) == 0 {
			return Response{Status: "success", Output: "No running containers.\n"}
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%-20s %-10s %-20s\n", "NAME", "PID", "UPTIME"))
		for _, rc := range d.containers {
			uptime := time.Since(rc.Started).Truncate(time.Second)
			sb.WriteString(fmt.Sprintf("%-20s %-10d %-20s\n", rc.Name, rc.PID, uptime))
		}
		return Response{Status: "success", Output: sb.String()}

	case "stop":
		if len(cmd.Args) < 1 {
			return Response{Status: "error", Message: "missing container name"}
		}
		name := cmd.Args[0]
		d.mu.Lock()
		defer d.mu.Unlock()
		rc, exists := d.containers[name]
		if !exists {
			return Response{Status: "error", Message: fmt.Sprintf("container '%s' is not running", name)}
		}
		if err := rc.Process.Stop(); err != nil {
			return Response{Status: "error", Message: fmt.Sprintf("failed to stop container: %v", err)}
		}
		delete(d.containers, name)
		return Response{Status: "success", Output: fmt.Sprintf("Container '%s' stopped\n", name)}

	case "list":
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
