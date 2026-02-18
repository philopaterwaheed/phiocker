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
	"github.com/philopaterwaheed/phiocker/internal/utils"
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
	Mux     *AttachMux // I/O multiplexer for Docker-style attach
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
	decoder := json.NewDecoder(conn)
	var cmd Command
	if err := decoder.Decode(&cmd); err != nil {
		conn.Close()
		return
	}

	if cmd.Type == "attach" {
		d.handleAttach(conn, cmd)
		return
	}

	defer conn.Close()
	response := d.executeCommand(cmd)
	json.NewEncoder(conn).Encode(response)
}

func (d *Daemon) handleAttach(conn net.Conn, cmd Command) {
	defer conn.Close()

	if len(cmd.Args) < 1 {
		json.NewEncoder(conn).Encode(Response{Status: "error", Message: "missing container name"})
		return
	}

	name := cmd.Args[0]
	d.mu.Lock()
	rc, exists := d.containers[name]
	d.mu.Unlock()

	if !exists {
		json.NewEncoder(conn).Encode(Response{Status: "error", Message: fmt.Sprintf("container '%s' is not running", name)})
		return
	}

	// Apply terminal size
	if len(cmd.Args) >= 3 {
		rows, _ := strconv.Atoi(cmd.Args[1])
		cols, _ := strconv.Atoi(cmd.Args[2])
		if rows > 0 && cols > 0 {
			utils.SetPTYWinSize(rc.Process.PTYMaster, uint16(rows), uint16(cols))
		}
	}

	// Send success + PID, then switch to raw byte streaming
	json.NewEncoder(conn).Encode(Response{
		Status: "success",
		Output: strconv.Itoa(rc.PID),
	})

	// Block until client detaches or container exits
	rc.Mux.Attach(conn)
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
			Mux:     NewAttachMux(cp.PTYMaster),
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
		var listErr error
		var output string
		if len(cmd.Args) > 0 && cmd.Args[0] == "images" {
			// List images
			output = captureOutput(func() {
				listErr = moods.ListImages(BasePath)
			})
		} else {
			// List containers
			output = captureOutput(func() {
				listErr = moods.ListContainers(BasePath)
			})
		}
		if listErr != nil {
			return Response{Status: "error", Message: listErr.Error(), Output: output}
		}
		return Response{Status: "success", Output: output}

	case "create":
		if len(cmd.Args) < 1 {
			return Response{Status: "error", Message: "missing generator file"}
		}
		var createErr error
		output := captureOutput(func() {
			createErr = moods.Create(cmd.Args[0], BasePath)
		})
		if createErr != nil {
			return Response{Status: "error", Message: createErr.Error(), Output: output}
		}
		return Response{Status: "success", Output: output}

	case "delete":
		if len(cmd.Args) < 1 {
			return Response{Status: "error", Message: "missing args for delete"}
		}

		var deleteErr error
		var output string
		switch cmd.Args[0] {
		case "all":
			d.mu.Lock()
			defer d.mu.Unlock()
			if len(d.containers) > 0 {
				return Response{Status: "error", Message: "cannot delete all containers while some are still running"}
			}
			output = captureOutput(func() {
				deleteErr = moods.DeleteAllContainers(BasePath)
			})
		case "image":
			if len(cmd.Args) < 2 {
				return Response{Status: "error", Message: "missing image name or subcommand for delete image"}
			}
			switch cmd.Args[1] {
			case "all":
				output = captureOutput(func() {
					deleteErr = moods.DeleteAllImages(BasePath)
				})
			default:
				imageName := cmd.Args[1]
				output = captureOutput(func() {
					deleteErr = moods.DeleteImage(imageName, BasePath)
				})
			}
		default:
			// Delete specific container
			d.mu.Lock()
			defer d.mu.Unlock()
			if _, exists := d.containers[cmd.Args[0]]; exists {
				return Response{Status: "error", Message: fmt.Sprintf("cannot delete container '%s' while it is still running", cmd.Args[0])}
			}
			containerName := cmd.Args[0]
			output = captureOutput(func() {
				deleteErr = moods.DeleteContainer(containerName, BasePath)
			})
		}
		if deleteErr != nil {
			return Response{Status: "error", Message: deleteErr.Error(), Output: output}
		}
		return Response{Status: "success", Output: output}

	case "update":
		if len(cmd.Args) < 1 {
			return Response{Status: "error", Message: "missing args for update"}
		}

		var updateErr error
		var output string
		switch cmd.Args[0] {
		case "all":
			output = captureOutput(func() {
				updateErr = moods.UpdateAllImages(BasePath)
			})
		default:
			imageName := cmd.Args[0]
			output = captureOutput(func() {
				updateErr = moods.UpdateImage(imageName, BasePath)
			})
		}
		if updateErr != nil {
			return Response{Status: "error", Message: updateErr.Error(), Output: output}
		}
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
