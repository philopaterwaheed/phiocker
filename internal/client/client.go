package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/philopaterwaheed/phiocker/internal/daemon"
)

func SendCommand(cmdType string, args []string) {
	conn, err := net.Dial("unix", daemon.SocketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to daemon: %v\nIs the daemon running?\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	cmd := daemon.Command{
		Type: cmdType,
		Args: args,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		panic(err)
	}

	var resp daemon.Response
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		if err == io.EOF {
			return
		}
		panic(err)
	}

	if resp.Status == "error" {
		fmt.Println("Error:", resp.Message)
		os.Exit(1)
	}

	fmt.Print(resp.Output)
	if resp.Message != "" {
		fmt.Println(resp.Message)
	}
}

func AttachContainer(containerName string) {
	conn, err := net.Dial("unix", daemon.SocketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to daemon: %v\nIs the daemon running?\n", err)
		os.Exit(1)
	}

	cmd := daemon.Command{
		Type: "attach",
		Args: []string{containerName},
	}

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		panic(err)
	}

	var resp daemon.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		if err == io.EOF {
			fmt.Fprintln(os.Stderr, "Error: unexpected EOF from daemon")
			os.Exit(1)
		}
		panic(err)
	}
	conn.Close()

	if resp.Status == "error" {
		fmt.Println("Error:", resp.Message)
		os.Exit(1)
	}

	pid := strings.TrimSpace(resp.Output)

	// Use nsenter to enter the container's namespaces and get a shell
	nsenter := exec.Command("nsenter",
		"-t", pid,
		"-m", "-p", "-u",
		"-r", "-w",
		"--", "/bin/sh",
	)
	nsenter.Stdin = os.Stdin
	nsenter.Stdout = os.Stdout
	nsenter.Stderr = os.Stderr

	fmt.Printf("Attaching to container '%s' (PID %s). Use 'exit' or Ctrl+D to detach.\n", containerName, pid)

	if err := nsenter.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error attaching to container: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Detached from container.")
}
