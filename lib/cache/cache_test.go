package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadReturnsEmptyCacheWhenFileDoesNotExist(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	c, err := Load(tmpDir)

	require.NoError(t, err)
	require.NotNil(t, c)
	require.Empty(t, c)
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	c := make(Cache)
	Update(c, "api", "abc123", "v1.0.0")
	Update(c, "worker", "def456", "v1.0.0")

	err := Save(tmpDir, c)
	require.NoError(t, err)

	loaded, err := Load(tmpDir)
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	require.Equal(t, "abc123", loaded["api"].Hash)
	require.Equal(t, "v1.0.0", loaded["api"].Tag)
	require.Equal(t, "def456", loaded["worker"].Hash)
}

func TestHashServiceDirProducesStableHash(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "myservice")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "main.go"), []byte("package main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "go.mod"), []byte("module example"), 0644))

	hash1, err := HashServiceDir(svcDir, tmpDir)
	require.NoError(t, err)
	require.NotEmpty(t, hash1)

	hash2, err := HashServiceDir(svcDir, tmpDir)
	require.NoError(t, err)

	require.Equal(t, hash1, hash2)
}

func TestHashServiceDirChangesOnFileModification(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "myservice")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "main.go"), []byte("package main"), 0644))

	hash1, err := HashServiceDir(svcDir, tmpDir)
	require.NoError(t, err)

	// Modify a file
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "main.go"), []byte("package main\nfunc init() {}"), 0644))

	hash2, err := HashServiceDir(svcDir, tmpDir)
	require.NoError(t, err)

	require.NotEqual(t, hash1, hash2)
}

func TestHashServiceDirChangesOnNewFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "myservice")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "main.go"), []byte("package main"), 0644))

	hash1, err := HashServiceDir(svcDir, tmpDir)
	require.NoError(t, err)

	// Add a new file
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "util.go"), []byte("package main"), 0644))

	hash2, err := HashServiceDir(svcDir, tmpDir)
	require.NoError(t, err)

	require.NotEqual(t, hash1, hash2)
}

func TestIsCached_Match(t *testing.T) {
	t.Parallel()

	c := make(Cache)
	Update(c, "api", "abc123", "v1.0.0")

	require.True(t, IsCached(c, "api", "abc123", "v1.0.0"))
}

func TestIsCached_DifferentHash(t *testing.T) {
	t.Parallel()

	c := make(Cache)
	Update(c, "api", "abc123", "v1.0.0")

	require.False(t, IsCached(c, "api", "different", "v1.0.0"))
}

func TestIsCached_DifferentTag(t *testing.T) {
	t.Parallel()

	c := make(Cache)
	Update(c, "api", "abc123", "v1.0.0")

	require.False(t, IsCached(c, "api", "abc123", "v2.0.0"))
}

func TestIsCached_NotPresent(t *testing.T) {
	t.Parallel()

	c := make(Cache)

	require.False(t, IsCached(c, "api", "abc123", "v1.0.0"))
}

func TestLoadCorruptedCacheReturnsEmpty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, cacheDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(FilePath(tmpDir), []byte("not json"), 0o644))

	c, err := Load(tmpDir)

	require.NoError(t, err)
	require.NotNil(t, c)
	require.Empty(t, c)
}

func TestFilePath(t *testing.T) {
	t.Parallel()

	fp := FilePath("/my/project")
	require.Equal(t, filepath.Join("/my/project", ".pilum", "build-cache.json"), fp)
}

func TestHashServiceDirEmptyDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	hash, err := HashServiceDir(svcDir, tmpDir)
	require.NoError(t, err)
	require.Empty(t, hash) // Empty dir produces empty hash
}
