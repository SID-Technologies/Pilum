// Package cache provides hash-based build caching to skip unchanged services.
// Cache state is stored at .pilum/build-cache.json and maps service names to
// content hashes of their source directories.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
)

const (
	cacheDir  = ".pilum"
	cacheFile = "build-cache.json"
)

// Entry holds the cached state for a single service build.
type Entry struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Tag       string    `json:"tag"`
}

// Cache maps service names to their cached build entries.
type Cache map[string]Entry

// FilePath returns the cache file path under the given project root.
func FilePath(projectRoot string) string {
	return filepath.Join(projectRoot, cacheDir, cacheFile)
}

// Load reads the build cache from disk. Returns an empty cache if the file
// doesn't exist.
func Load(projectRoot string) (Cache, error) {
	fp := FilePath(projectRoot)

	data, err := os.ReadFile(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return make(Cache), nil
		}
		return nil, errors.Wrap(err, "reading cache file")
	}

	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		// Corrupted cache — return empty
		return make(Cache), nil
	}

	return c, nil
}

// Save writes the build cache to disk.
func Save(projectRoot string, c Cache) error {
	dir := filepath.Join(projectRoot, cacheDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return errors.Wrap(err, "creating %s directory", cacheDir)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshaling cache")
	}

	fp := FilePath(projectRoot)
	if err := os.WriteFile(fp, data, 0o600); err != nil {
		return errors.Wrap(err, "writing cache file")
	}

	return nil
}

// HashServiceDir computes a content hash for a service directory. It uses
// git ls-files to respect .gitignore, falling back to filepath.Walk if not
// in a git repo. The hash is a SHA-256 of sorted "path\x00content" pairs.
func HashServiceDir(serviceDir, projectRoot string) (string, error) {
	// Resolve relative service dirs to absolute so filepath.Rel works correctly
	// when gitTrackedFiles joins paths with the absolute projectRoot.
	if !filepath.IsAbs(serviceDir) {
		abs, err := filepath.Abs(serviceDir)
		if err == nil {
			serviceDir = abs
		}
	}

	files, err := gitTrackedFiles(serviceDir, projectRoot)
	if err != nil {
		// Fallback to filesystem walk
		files, err = walkFiles(serviceDir)
		if err != nil {
			return "", errors.Wrap(err, "walking service directory")
		}
	}

	if len(files) == 0 {
		return "", nil
	}

	sort.Strings(files)

	h := sha256.New()
	for _, f := range files {
		absPath := filepath.Join(serviceDir, f)
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue // skip files we can't read
		}
		// Write "relativePath\x00content" to hasher
		_, _ = io.WriteString(h, f)
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(content)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// IsCached returns true if the service has a cache entry matching the current
// hash and tag.
func IsCached(c Cache, serviceName, currentHash, tag string) bool {
	entry, exists := c[serviceName]
	if !exists {
		return false
	}
	if entry.Hash != currentHash {
		return false
	}
	if entry.Tag != tag {
		return false
	}
	return true
}

// Update sets the cache entry for a service.
func Update(c Cache, serviceName, hash, tag string) {
	c[serviceName] = Entry{
		Hash:      hash,
		Timestamp: time.Now(),
		Tag:       tag,
	}
}

// gitTrackedFiles returns the list of files tracked by git (plus untracked
// non-ignored files) relative to serviceDir.
func gitTrackedFiles(serviceDir, projectRoot string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", serviceDir)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Make paths relative to serviceDir
		rel, err := filepath.Rel(serviceDir, filepath.Join(projectRoot, line))
		if err != nil {
			continue
		}
		// Skip files outside serviceDir
		if strings.HasPrefix(rel, "..") {
			continue
		}
		files = append(files, rel)
	}

	return files, nil
}

// walkFiles returns all files in a directory, excluding common build artifacts.
func walkFiles(dir string) ([]string, error) {
	excludeDirs := map[string]bool{
		"node_modules": true,
		".git":         true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		"__pycache__":  true,
		".pilum":       true,
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			if excludeDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
