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

// CheckBuildxCache reports whether the active buildx builder can export a
// layer cache to a registry. When it can't, reason says why. The default
// "docker" driver only can with the containerd image store; a
// docker-container, remote or kubernetes builder always can.
func CheckBuildxCache(timeout time.Duration) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	inspect, err := exec.CommandContext(ctx, "docker", "buildx", "inspect").CombinedOutput()
	if err != nil {
		return false, "docker buildx is not available"
	}

	driverStatus, err := exec.CommandContext(ctx, "docker", "info", "--format", "{{json .DriverStatus}}").Output()
	if err != nil {
		// Can't tell whether the containerd store is on; assume it isn't.
		driverStatus = nil
	}

	return buildxCacheSupport(string(inspect), string(driverStatus))
}

// buildxCacheSupport decides from `docker buildx inspect` output and the
// daemon's driver status whether the builder can export a registry cache.
func buildxCacheSupport(inspect, driverStatus string) (bool, string) {
	driver := ""
	for line := range strings.SplitSeq(inspect, "\n") {
		value, found := strings.CutPrefix(strings.TrimSpace(line), "Driver:")
		if found {
			driver = strings.TrimSpace(value)
			break
		}
	}

	switch driver {
	case "docker-container", "remote", "kubernetes":
		return true, ""
	case "docker":
		if strings.Contains(driverStatus, "io.containerd.snapshotter") {
			return true, ""
		}
		return false, "the active buildx builder uses the \"docker\" driver, which can't export a registry cache"
	case "":
		return false, "could not determine the active buildx builder's driver"
	default:
		return false, "the active buildx builder uses the \"" + driver + "\" driver, which can't export a registry cache"
	}
}
