package lambdadeploy

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
)

// zipEpoch is every entry's modification time, so the same files always
// make the same zip. Lambda compares code by SHA-256: an unchanged build
// then publishes no new version.
var zipEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// Package returns a zip of source: source itself when it's a .zip file,
// otherwise a zip of the directory's contents in a temp file, which cleanup
// removes.
func Package(source string) (string, func(), error) {
	noop := func() {}
	info, err := os.Stat(source)
	if err != nil {
		return "", noop, errors.Wrap(err, "lambda artifact %s (set lambda.artifact to your build output)", source)
	}
	if !info.IsDir() {
		if !strings.HasSuffix(strings.ToLower(source), ".zip") {
			return "", noop, errors.New("lambda artifact %s is a file but not a .zip; point lambda.artifact at a .zip or a directory", source)
		}
		return source, noop, nil
	}

	out, err := os.CreateTemp("", "pilum-lambda-*.zip")
	if err != nil {
		return "", noop, errors.Wrap(err, "creating zip")
	}
	cleanup := func() { _ = os.Remove(out.Name()) }

	err = writeZip(out, source)
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		cleanup()
		return "", noop, errors.Wrap(err, "zipping %s", source)
	}
	return out.Name(), cleanup, nil
}

// writeZip writes dir's files, in lexical order, with their modes (a Go
// function's bootstrap must stay executable) and a fixed timestamp.
func writeZip(w io.Writer, dir string) error {
	zw := zip.NewWriter(w)
	count := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil // sockets, devices; symlinks Lambda wouldn't follow
		}

		header := &zip.FileHeader{Name: filepath.ToSlash(rel), Method: zip.Deflate, Modified: zipEpoch}
		header.SetMode(info.Mode())
		entry, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(path) //nolint:gosec // Walking the service's own build output
		if err != nil {
			return err
		}
		_, err = io.Copy(entry, f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		count++
		return closeErr
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("%s has no files; did the build write somewhere else?", dir)
	}
	return zw.Close()
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // The zip this deploy wrote or was given
	if err != nil {
		return "", errors.Wrap(err, "reading %s", path)
	}
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", errors.Wrap(err, "hashing %s", path)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
