package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/philopaterwaheed/phiocker/internal/client"
	"github.com/philopaterwaheed/phiocker/internal/daemon"
	"github.com/philopaterwaheed/phiocker/internal/moods"
)

const basePath = "/var/lib/phiocker"

func showHelp() {
	fmt.Println("phiocker - A simple container management tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  phiocker <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  daemon                      Start the daemon")
	fmt.Println("  run <container_name>        Run a container (detached)")
	fmt.Println("  attach <container_name>     Attach to a running container (Ctrl+P, Ctrl+Q to detach)")
	fmt.Println("  stop <container_name>       Stop a running container")
	fmt.Println("  ps                          List running containers")
	fmt.Println("  create <generator_file>     Create a new container from generator file")
	fmt.Println("  download                    Download base images")
	fmt.Println("  search <repository> [limit] Search for downloadable images in a repository (optional limit)")
	fmt.Println("  delete <container_name>     Safely delete a specific container")
	fmt.Println("  delete all                  Safely delete all containers")
	fmt.Println("  delete list                 List all containers before deletion")
	fmt.Println("  delete image <image_name>   Safely delete a specific image")
	fmt.Println("  delete image all            Safely delete all images")
	fmt.Println("  delete image list           List all images before deletion")
	fmt.Println("  list                        List all available containers")
	fmt.Println("  list images                 List all available images")
	fmt.Println("  help, -h, --help            Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  phiocker create example.json")
	fmt.Println("  phiocker run my-container")
	fmt.Println("  phiocker attach my-container")
	fmt.Println("  phiocker stop my-container")
	fmt.Println("  phiocker ps")
	fmt.Println("  phiocker list")
	fmt.Println("  phiocker list images")
	fmt.Println("  phiocker search ubuntu")
	fmt.Println("  phiocker search nginx:1.21")
	fmt.Println("  phiocker delete my-container")
	fmt.Println("  phiocker delete all")
	fmt.Println("  phiocker delete image ubuntu")
	fmt.Println("  phiocker delete image all")
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	if os.Args[1] == "daemon" {
		d := daemon.New()
		if err := d.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check if daemon socket exists to decide mode
	useDaemon := false
	if _, err := os.Stat(daemon.SocketPath); err == nil {
		useDaemon = true
	}

	if os.Args[1] == "child" {
		moods.Child(os.Args[2], basePath)
		return
	}

	if useDaemon {
		// Client mode
		switch os.Args[1] {
		case "help", "-h", "--help":
			showHelp()
		case "run":
			if len(os.Args) < 3 {
				panic("usage: run <container_name>")
			}
			client.SendCommand("run", os.Args[2:])
		case "create":
			if len(os.Args) < 3 {
				panic("usage: create <generator_file>")
			}
			client.SendCommand("create", os.Args[2:])
		case "attach":
			if len(os.Args) < 3 {
				panic("usage: attach <container_name>")
			}
			client.AttachContainer(os.Args[2])
		case "ps":
			client.SendCommand("ps", nil)
		case "stop":
			if len(os.Args) < 3 {
				panic("usage: stop <container_name>")
			}
			client.SendCommand("stop", os.Args[2:])
		case "list":
			if len(os.Args) >= 3 && os.Args[2] == "images" {
				client.SendCommand("list", os.Args[2:])
			} else {
				client.SendCommand("list", nil)
			}
		case "delete":
			client.SendCommand("delete", os.Args[2:])
		default:
			// Fallback or error?
			if os.Args[1] == "download" || os.Args[1] == "search" {
				goto LocalMode
			}
			panic("unknown command or not supported via daemon yet")
		}
		return
	}

// For command that don't need to run on the back
LocalMode:
	switch os.Args[1] {
	case "help", "-h", "--help":
		showHelp()
	case "download":
		moods.Download(basePath)
	case "search":
		if len(os.Args) < 3 {
			panic("usage: search <repository> [limit]")
		}
		limit := 50
		if len(os.Args) >= 4 {
			if parsedLimit, err := strconv.Atoi(os.Args[3]); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}
		moods.Search(os.Args[2], limit)
	// TODO: make the daemon run it but without effecting it if running
	case "delete":
		if len(os.Args) < 3 {
			panic("usage: delete <container_name> | delete all | delete list | delete image <image_name> | delete image all | delete image list")
		}
		switch os.Args[2] {
		case "all":
			moods.DeleteAllContainers(basePath)
		case "list":
			moods.ListContainers(basePath)
		case "image":
			if len(os.Args) < 4 {
				panic("usage: delete image <image_name> | delete image all | delete image list")
			}
			switch os.Args[3] {
			case "all":
				moods.DeleteAllImages(basePath)
			case "list":
				moods.ListImages(basePath)
			default:
				imageName := os.Args[3]
				moods.DeleteImage(imageName, basePath)
			}
		default:
			containerName := os.Args[2]
			moods.DeleteContainer(containerName, basePath)
		}
	case "list":
		if len(os.Args) >= 3 && os.Args[2] == "images" {
			moods.ListImages(basePath)
		} else {
			moods.ListContainers(basePath)
		}
	default:
		panic("unknown command")
	}
}
