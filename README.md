# phiocker (philo's docker)

phiocker is a small Linux container runtime written in Go. It pulls OCI images, expands them into a local root filesystem, and runs containers with Linux namespaces, cgroup v2 limits, a PTY-backed attach flow, and a Unix-socket daemon.

## What it does

- Pulls OCI images from container registries.
- Creates per-container root filesystems under `/var/lib/phiocker`.
- Starts containers with new UTS, PID, mount, and network namespaces.
- Applies CPU, memory, and PID limits through cgroup v2.
- Configures a bridge network and NAT for outbound connectivity.
- Lets you attach and detach from running containers through the daemon.

## Runtime model

The `phiocker daemon` process listens on `/var/run/phiocker.sock` and tracks running containers in memory. Most lifecycle commands use that socket when it is available.

When a container starts, phiocker re-executes itself as a `child` process, then:

- opens a PTY pair for interactive attach support
- creates new namespaces for UTS, PID, mount, and networking
- places the process in a cgroup v2 leaf
- prepares a root filesystem copied from the selected image
- bind-mounts a minimal set of `/dev` devices
- `chroot`s into the container rootfs, mounts `/proc`, and runs the configured command

While attached, press `Ctrl+P` then `Ctrl+Q` to detach without stopping the container.

## Requirements

- Linux only
- Root privileges
- cgroup v2 enabled
- Go 1.25+
- System tools available on the host: `ip`, `iptables`, `nsenter`, `cp`

The runtime stores state in `/var/lib/phiocker` and expects to create `/var/run/phiocker.sock`.

## Installation

### Build locally

```bash
sudo make build
```

If you only want a local binary without installing it system-wide:

```bash
go build -o phiocker ./cmd/phiocker
```

### Install as a systemd service

```bash
make install-service
```
### Remove the service

```bash
make uninstall-service
```

## Quick start

### 1. Start the daemon

Run it directly:

```bash
sudo phiocker daemon
```

Or install the systemd service and let systemd manage it.

### 2. Create a generator file

Example:

```json
{
  "name": "demo",
  "baseImage": "ubuntu:latest",
  "cmd": ["/bin/sh"],
  "workdir": "/root",
  "copy": [
    {
      "src": "./app",
      "dst": "/app"
    }
  ],
  "limits": {
    "cpuQuota": 50000,
    "cpuPeriod": 100000,
    "memory": 104857600,
    "pids": 20
  }
}
```

### 3. Create and run a container

```bash
sudo phiocker create ./container.json
sudo phiocker run demo
sudo phiocker attach demo
```

### 4. Stop or inspect it

```bash
sudo phiocker ps
sudo phiocker stop demo
sudo phiocker list
sudo phiocker list images
```

## Command reference

### Daemon-backed commands

| Command | Description |
|---|---|
| `phiocker daemon` | Start the daemon |
| `phiocker create <generator_file>` | Create a container from a generator file |
| `phiocker run <container_name>` | Start a created container |
| `phiocker attach <container_name>` | Attach to a running container |
| `phiocker stop <container_name>` | Stop a running container |
| `phiocker ps` | List running containers tracked by the daemon |
| `phiocker list` | List all created containers on disk |
| `phiocker list images` | List cached images on disk |
| `phiocker delete <container_name>` | Delete one container |
| `phiocker delete all` | Delete all containers |
| `phiocker delete image <image_name>` | Delete one cached image |
| `phiocker delete image all` | Delete all cached images |
| `phiocker update <image_name>` | Re-pull one cached image |
| `phiocker update all` | Re-pull all cached images |

### Local commands

These commands run without the daemon:

| Command | Description |
|---|---|
| `phiocker help` | Show CLI help |
| `phiocker search <repository> [limit]` | List tags for a repository |
| `phiocker search <repository:tag>` | Show details for a specific image reference |
| `phiocker download <image>` | Pull and extract an image into the local cache |
| `phiocker list` | Works locally when the daemon is not running |
| `phiocker list images` | Works locally when the daemon is not running |

Notes:

- `create` automatically downloads the base image if it is not already cached.
- `search ubuntu 20` lists up to 20 tags for `ubuntu`.

## Generator file reference

| Field | Required | Description |
|---|---|---|
| `name` | yes | Container name used by subsequent commands |
| `baseImage` | yes | OCI image reference to use as the container base |
| `cmd` | yes | Command and arguments executed inside the container |
| `workdir` | no | Working directory inside the container; defaults to `/` |
| `copy` | no | Files or directories copied into the container before runtime |
| `limits.cpuQuota` | no | CPU quota in microseconds |
| `limits.cpuPeriod` | no | CPU period in microseconds |
| `limits.memory` | no | Memory limit in bytes |
| `limits.pids` | no | Maximum process count |

## Storage layout

```text
/var/lib/phiocker/
├── containers/
│   └── <name>/
│       ├── config.json
│       └── rootfs/
└── images/
    └── <image-ref>/
        └── rootfs/
```
The daemon socket lives at `/var/run/phiocker.sock`.

## Networking

phiocker creates a Linux bridge named `phiocker0` with subnet `172.20.0.0/16`. Each running container gets:

- a veth pair connected to the bridge
- an `eth0` address in that subnet
- a default route through `172.20.0.1`
- a generated `/etc/resolv.conf` based on host DNS settings

Outbound traffic is enabled through an `iptables` masquerade rule.

## Project layout

```text
cmd/phiocker/main.go
internal/
  client/      CLI to daemon communication
  daemon/      Unix socket server and attach handling
  download/    OCI pull and extraction logic
  moods/       container lifecycle and config handling
  network/     bridge, veth, routing, and DNS setup
  utils/       filesystem and PTY helpers
```

## Limitations

- Running-container state is kept in daemon memory, so `ps` only shows containers started by the current daemon instance.
- The runtime is Linux-specific and expects root access.
- Networking depends on host tools such as `ip`, `iptables`, and `nsenter` being installed.
- No port farwarding yet
