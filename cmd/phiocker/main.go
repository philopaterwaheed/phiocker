package main

import (
	"fmt"
	"os"

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
	fmt.Println("  run <container_name>        Run a container")
	fmt.Println("  create <generator_file>     Create a new container from generator file")
	fmt.Println("  download                    Download base images")
	fmt.Println("  delete <container_name>     Safely delete a specific container")
	fmt.Println("  delete all                  Safely delete all containers")
	fmt.Println("  delete list                 List all containers before deletion")
	fmt.Println("  list                        List all available containers")
	fmt.Println("  help, -h, --help            Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  phiocker create example.json")
	fmt.Println("  phiocker run my-container")
	fmt.Println("  phiocker list")
	fmt.Println("  phiocker delete my-container")
	fmt.Println("  phiocker delete all")
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	switch os.Args[1] {
	case "help", "-h", "--help":
		showHelp()
	case "run":
		if len(os.Args) < 3 {
			panic("usage: run <container_name>")
		}
		moods.Run()
	case "child":
		//Check are do at run
		moods.Child(os.Args[2], basePath)
	case "download":
		moods.Download(basePath)
	case "create":
		if len(os.Args) < 3 {
			panic("usage: create <generator_file>")
		}
		generatorFilePath := os.Args[2]
		moods.Create(generatorFilePath, basePath)
	case "delete":
		if len(os.Args) < 3 {
			panic("usage: delete <container_name> | delete all | delete list")
		}
		switch os.Args[2] {
		case "all":
			moods.DeleteAllContainers(basePath)
		case "list":
			moods.ListContainers(basePath)
		default:
			containerName := os.Args[2]
			moods.DeleteContainer(containerName, basePath)
		}
	case "list":
		moods.ListContainers(basePath)
	default:
		panic("unknown command")
	}
}
