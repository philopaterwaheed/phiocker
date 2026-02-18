package moods

import (
	"encoding/json"
	"io"
)

type CopySpec struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type ContainerConfig struct {
	Name      string     `json:"name"`
	Baseimage string     `json:"baseImage"`
	Cmd       []string   `json:"cmd,omitempty"`
	Workdir   string     `json:"workdir,omitempty"`
	Copy      []CopySpec `json:"copy,omitempty"`
}

func LoadConfig(reader io.Reader) ContainerConfig {
	var config ContainerConfig
	err := json.NewDecoder(reader).Decode(&config)
	if err != nil {
		panic(err)
	}
	return config
}

