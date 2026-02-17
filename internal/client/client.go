package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/philopaterwaheed/phiocker/internal/daemon"
	"golang.org/x/sys/unix"
)

var errDetached = errors.New("detached")

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

	// Get the client terminal size to apply to the container PTY
	rows, cols := getTermSize()

	cmd := daemon.Command{
		Type: "attach",
		Args: []string{containerName, strconv.Itoa(rows), strconv.Itoa(cols)},
	}

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending command: %v\n", err)
		os.Exit(1)
	}

	// Read the JSON response; use a decoder and then handle any buffered bytes
	decoder := json.NewDecoder(conn)
	var resp daemon.Response
	if err := decoder.Decode(&resp); err != nil {
		if err == io.EOF {
			fmt.Fprintln(os.Stderr, "Error: unexpected EOF from daemon")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	if resp.Status == "error" {
		fmt.Println("Error:", resp.Message)
		os.Exit(1)
	}

	pid := strings.TrimSpace(resp.Output)

	// Build a reader that includes any bytes the JSON decoder buffered past the response
	connReader := io.MultiReader(decoder.Buffered(), conn)

	//Keystrokes are forwarded immediately
	oldState, rawErr := makeRaw(int(os.Stdin.Fd()))
	if rawErr == nil {
		defer restoreTerminal(int(os.Stdin.Fd()), oldState)
	}

	fmt.Fprintf(os.Stdout, "Attached to container '%s' (PID %s). Use Ctrl+P, Ctrl+Q to detach.\r\n", containerName, pid)

	done := make(chan error, 2)

	// Container output → client stdout
	go func() {
		// Reapeatedly read
		_, err := io.Copy(os.Stdout, connReader)
		done <- err
	}()

	// Client stdin → container (with Ctrl+P, Ctrl+Q detach detection)
	go func() {
		done <- copyWithDetach(conn)
	}()

	result := <-done
	conn.Close()

	if errors.Is(result, errDetached) {
		fmt.Fprint(os.Stdout, "\r\nDetached from container.\r\n")
	} else {
		fmt.Fprint(os.Stdout, "\r\nConnection to container closed.\r\n")
	}
}

// copyWithDetach reads from stdin byte-by-byte.
// It detects the Docker-style Ctrl+P, Ctrl+Q escape sequence to detach.
func copyWithDetach(conn net.Conn) error {
	buf := make([]byte, 1)
	var prevCtrlP bool

	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			b := buf[0]

			if prevCtrlP {
				if b == 0x11 { // Ctrl+Q after Ctrl+P → detach
					return errDetached
				}
				// Not Ctrl+Q: send the buffered Ctrl+P first
				if _, werr := conn.Write([]byte{0x10}); werr != nil {
					return werr
				}
				prevCtrlP = false
			}

			if b == 0x10 { // Ctrl+P → buffer it
				prevCtrlP = true
				continue
			}

			if _, werr := conn.Write(buf[:1]); werr != nil {
				return werr
			}
		}
		if err != nil {
			// Flush the pending Ctrl+P if stdin closed
			if prevCtrlP {
				conn.Write([]byte{0x10})
			}
			return err
		}
	}
}

// --- terminal helpers (using golang.org/x/sys/unix) ---

// makeRaw puts the terminal in raw mode and returns the previous state.
func makeRaw(fd int) (*unix.Termios, error) {
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	oldState := *termios

	// cfmakeraw equivalent
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		return nil, err
	}

	return &oldState, nil
}

// restoreTerminal restores the terminal to a previous state.
func restoreTerminal(fd int, state *unix.Termios) {
	unix.IoctlSetTermios(fd, unix.TCSETS, state)
}

// getTermSize returns the current terminal dimensions (rows, cols).
func getTermSize() (int, int) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 24, 80 // sensible defaults
	}
	return int(ws.Row), int(ws.Col)
}
