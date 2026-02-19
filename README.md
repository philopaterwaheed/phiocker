# phiocker (philo's docker)

A lightweight Linux container runtime written in Go. It pulls OCI images from any container registry, creates isolated environments using Linux namespaces and cgroups v2, and manages their lifecycle through a Unix-socket daemon.

---

## How it works

When you run a container, the daemon re-executes the phiocker binary as a child process with three new namespaces (`CLONE_NEWUTS`, `CLONE_NEWPID`, `CLONE_NEWNS`). The child process `chroot`s into the container's rootfs, mounts `/proc`, then executes the configured command. A PTY pair is created so you can attach and detach interactively at any time. Resource limits are applied via a cgroup v2 leaf before the child starts.

The daemon listens on `/var/run/phiocker.sock`. The CLI detects whether the socket exists and either sends JSON commands to the daemon or shows an error.

---

## Requirements

- Linux kernel ≥ 5.14 (cgroup v2 + `UseCgroupFD`)
- Go 1.25+
- Root privileges

---

## Installation

```bash
# build and install binary to /usr/local/bin/phiocker
make build

# build + install + enable systemd service
make install-service

# remove service (prompts to also remove binary)
make uninstall-service
```

The systemd unit starts the daemon automatically on boot and restarts it on failure.

---

## Usage

Start the daemon before running any other command:

```bash
sudo phiocker daemon
```

| Command | Description |
|---|---|
| `phiocker create <file.json>` | Create a container from a generator file |
| `phiocker run <name>` | Start a container in the background |
| `phiocker attach <name>` | Attach to a running container's terminal |
| `phiocker stop <name>` | Send SIGTERM to a running container |
| `phiocker ps` | List running containers |
| `phiocker list` | List all containers (running or not) |
| `phiocker list images` | List downloaded images |
| `phiocker search <repo[:tag]> [limit]` | Search for images in a registry |
| `phiocker update <image>` | Re-pull a specific image |
| `phiocker update all` | Re-pull all images |
| `phiocker delete <name>` | Delete a container |
| `phiocker delete all` | Delete all containers |
| `phiocker delete image <name>` | Delete an image |
| `phiocker delete image all` | Delete all images |

While attached, press **Ctrl+P** then **Ctrl+Q** to detach without stopping the container.

---

## Generator file

Containers are defined by a JSON file passed to `phiocker create`.

```json
{
    "name": "my-container",
    "baseImage": "ubuntu:latest",
    "cmd": ["/bin/bash"],
    "workdir": "/root",
    "copy": [
        { "src": "app/", "dst": "/app" }
    ],
    "limits": {
        "cpuQuota":  50000,
        "cpuPeriod": 100000,
        "memory":    104857600,
        "pids":      20
    }
}
```

| Field | Required | Description |
|---|---|---|
| `name` | yes | Container name, used for all subsequent commands |
| `baseImage` | yes | Any OCI image reference (`image:tag`, registry prefix, etc.) |
| `cmd` | yes | Entrypoint and arguments run inside the container |
| `workdir` | no | Working directory inside the container (default `/`) |
| `copy` | no | Files or directories to copy from the host into the container |
| `limits.cpuQuota` | no | CPU quota in microseconds per period |
| `limits.cpuPeriod` | no | CPU period in microseconds (default kernel value if 0) |
| `limits.memory` | no | Memory limit in bytes |
| `limits.pids` | no | Maximum number of PIDs inside the container |

If `baseImage` is not already cached locally, it is downloaded automatically during `create`.

---

## Directory layout

```
/var/lib/phiocker/
├── images/
│   └── <image-name>/
│       └── rootfs/       # extracted OCI image layers
└── containers/
    └── <name>/
        ├── rootfs/       # copy of image rootfs for this container
        └── config.json   # generator file stored alongside the container
```

---

## Project layout

```
cmd/phiocker/main.go        entry point, CLI argument dispatch
internal/
  daemon/
    daemon.go               Unix socket server, command dispatch, container lifecycle
    attach.go               PTY I/O multiplexer (AttachMux)
  moods/
    types.go                ContainerConfig and Limits types
    create.go               Container creation (image pull, rootfs copy, file injection)
    run.go                  RunDetached — namespace + cgroup setup, PTY creation
    child.go                Child process entry: chroot, mount /proc, exec
    list.go / delete.go … remaining lifecycle operations
  download/download.go      OCI image pull and layer extraction
  client/client.go          CLI-side socket client
  utils/                    Directory helpers, file utilities, PTY helpers
```
