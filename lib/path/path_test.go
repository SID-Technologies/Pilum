package path_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sid-technologies/pilum/lib/path"
	"github.com/stretchr/testify/require"
)

// resolvePath resolves symlinks in a path for comparison on macOS
func resolvePath(t *testing.T, p string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return resolved
}

func TestProjectConfigList(t *testing.T) {
	t.Parallel()

	// Verify the list of project config files
	require.Contains(t, path.ProjectConfig, "package.json")
	require.Contains(t, path.ProjectConfig, "go.mod")
	require.Contains(t, path.ProjectConfig, "Cargo.toml")
	require.Contains(t, path.ProjectConfig, ".gitignore")
	require.Contains(t, path.ProjectConfig, "tsconfig.json")
	require.Contains(t, path.ProjectConfig, "cdk.json")
}

func TestFindProjectRootWithGoMod(t *testing.T) {
	// Not parallel - changes working directory

	// Create a temp directory structure
	tmpDir := t.TempDir()

	// Create go.mod in root
	goModPath := filepath.Join(tmpDir, "go.mod")
	err := os.WriteFile(goModPath, []byte("module test"), 0644)
	require.NoError(t, err)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "cmd", "app")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Find project root
	root, err := path.FindProjectRoot()

	require.NoError(t, err)
	require.Equal(t, resolvePath(t, tmpDir), root)
}

func TestFindProjectRootWithPackageJson(t *testing.T) {
	// Not parallel - changes working directory

	tmpDir := t.TempDir()

	// Create package.json in root
	packageJsonPath := filepath.Join(tmpDir, "package.json")
	err := os.WriteFile(packageJsonPath, []byte("{}"), 0644)
	require.NoError(t, err)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "src", "components")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Find project root
	root, err := path.FindProjectRoot()

	require.NoError(t, err)
	require.Equal(t, resolvePath(t, tmpDir), root)
}

func TestFindProjectRootWithCargoToml(t *testing.T) {
	// Not parallel - changes working directory

	tmpDir := t.TempDir()

	// Create Cargo.toml in root
	cargoPath := filepath.Join(tmpDir, "Cargo.toml")
	err := os.WriteFile(cargoPath, []byte("[package]"), 0644)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to root
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Find project root
	root, err := path.FindProjectRoot()

	require.NoError(t, err)
	require.Equal(t, resolvePath(t, tmpDir), root)
}

func TestFindProjectRootWithGitignore(t *testing.T) {
	// Not parallel - changes working directory

	tmpDir := t.TempDir()

	// Create .gitignore in root
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("node_modules/"), 0644)
	require.NoError(t, err)

	// Create nested subdirectory
	subDir := filepath.Join(tmpDir, "a", "b", "c")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to deep subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Find project root
	root, err := path.FindProjectRoot()

	require.NoError(t, err)
	require.Equal(t, resolvePath(t, tmpDir), root)
}

func TestFindProjectRootNoConfigFound(t *testing.T) {
	// Not parallel - changes working directory

	tmpDir := t.TempDir()

	// Don't create any config files

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to empty directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Find project root should fail
	_, err = path.FindProjectRoot()

	require.Error(t, err)
	require.Contains(t, err.Error(), "no project configuration found")
}

func TestFindProjectRootIgnoresDirectories(t *testing.T) {
	// Not parallel - changes working directory

	tmpDir := t.TempDir()

	// Create a directory named "go.mod" (not a file)
	goModDir := filepath.Join(tmpDir, "go.mod")
	err := os.Mkdir(goModDir, 0755)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Find project root should fail (directory named go.mod doesn't count)
	_, err = path.FindProjectRoot()

	require.Error(t, err)
	require.Contains(t, err.Error(), "no project configuration found")
}

func TestFindProjectRootInCurrentDirectory(t *testing.T) {
	// Not parallel - changes working directory

	tmpDir := t.TempDir()

	// Create go.mod in current directory
	goModPath := filepath.Join(tmpDir, "go.mod")
	err := os.WriteFile(goModPath, []byte("module test"), 0644)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to directory with go.mod
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Find project root should return current directory
	root, err := path.FindProjectRoot()

	require.NoError(t, err)
	require.Equal(t, resolvePath(t, tmpDir), root)
}

func TestFindProjectRootMultipleConfigs(t *testing.T) {
	// Not parallel - changes working directory

	tmpDir := t.TempDir()

	// Create multiple config files
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644)
	require.NoError(t, err)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "sub")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Find project root
	root, err := path.FindProjectRoot()

	require.NoError(t, err)
	require.Equal(t, resolvePath(t, tmpDir), root)
}
