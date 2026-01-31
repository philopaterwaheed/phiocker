package main

import (
	"github.com/philopaterwaheed/phiocker/internal/moods"
	"os"
)

const basePath = "/var/lib/phiocker"

func main() {
	if len(os.Args) < 2 {
		//Todo: show help message
		panic("usage: run <command>")
	}

	switch os.Args[1] {
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
	default:
		panic("unknown command")
	}
}

