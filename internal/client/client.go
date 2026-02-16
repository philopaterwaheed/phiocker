package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

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
