package moods

import (
	"encoding/json"
	"io"
)

type CopySpec struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type Limits struct {
	CPUQuota  int `json:"cpuQuota,omitempty"`  // CPU time in microseconds per period
	CPUPeriod int `json:"cpuPeriod,omitempty"` // CPU period in microseconds
	Memory    int `json:"memory,omitempty"`    // Memory limit in bytes
	PIDs      int `json:"pids,omitempty"`      // Maximum number of PIDs
}

type ContainerConfig struct {
	Name      string     `json:"name"`
	Baseimage string     `json:"baseImage"`
	Cmd       []string   `json:"cmd,omitempty"`
	Workdir   string     `json:"workdir,omitempty"`
	Copy      []CopySpec `json:"copy,omitempty"`
	Limits    Limits     `json:"limits,omitempty"`
}

func LoadConfig(reader io.Reader) ContainerConfig {
	var config ContainerConfig
	err := json.NewDecoder(reader).Decode(&config)
	if err != nil {
		panic(err)
	}
	return config
}
