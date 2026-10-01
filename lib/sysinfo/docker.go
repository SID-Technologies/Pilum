package sysinfo

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
)

// DockerResources is the CPU and memory available to the docker daemon.
// On Docker Desktop this is the Linux VM, which is usually smaller than the
// host (e.g. 8GB VM on a 16GB Mac), so docker builds must be sized from it.
type DockerResources struct {
	CPUs     int
	MemoryMB int
}

// CheckDocker verifies the docker CLI is installed and its daemon is reachable,
// returning the daemon's resources. `docker info` is the cheapest call that
// round-trips to the daemon; the CLI alone being on PATH says nothing about
// whether Docker Desktop is running.
func CheckDocker(timeout time.Duration) (DockerResources, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return DockerResources{}, errors.New("docker is required but was not found in PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", "info", "--format", "{{.NCPU}} {{.MemTotal}}").CombinedOutput()
	if ctx.Err() != nil {
		return DockerResources{}, errors.New("docker daemon did not respond within %s — is Docker running?", timeout)
	}
	if err != nil {
		return DockerResources{}, errors.New("docker daemon is not running — start Docker and retry (%s)",
			strings.TrimSpace(string(out)))
	}

	return parseDockerResources(string(out)), nil
}

// parseDockerResources parses "<NCPU> <MemTotal bytes>". Unparseable fields are
// left as 0 so callers fall back to host values.
func parseDockerResources(s string) DockerResources {
	var res DockerResources
	fields := strings.Fields(s)
	if len(fields) != 2 {
		return res
	}
	if cpus, err := strconv.Atoi(fields[0]); err == nil && cpus > 0 {
		res.CPUs = cpus
	}
	if mem, err := strconv.ParseInt(fields[1], 10, 64); err == nil && mem > 0 {
		res.MemoryMB = int(mem / (1024 * 1024))
	}
	return res
}
