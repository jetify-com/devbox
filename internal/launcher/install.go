// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package launcher

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.jetify.com/devbox/internal/fileutil"
)

// These URLs are variables so that tests can serve them locally.
var (
	// releasesURL serves the latest version of each channel.
	releasesURL = "https://releases.jetify.com/devbox"
	// downloadURL serves the release archives and checksums.
	downloadURL = "https://github.com/jetify-com/devbox/releases/download"
)

// httpClient's timeout is a backstop: callers set their own deadlines.
var httpClient = &http.Client{Timeout: 10 * time.Minute}

// FetchLatestVersion returns the latest devbox version from the release
// channel, which is the stable channel unless the channel config file names
// another URL.
func FetchLatestVersion(ctx context.Context) (string, error) {
	url := readConfig(channelFile())
	if url == "" {
		url = releasesURL + "/stable/version"
	}
	resp, err := get(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	version := strings.TrimSpace(string(data))
	if !validVersion(version) {
		return "", fmt.Errorf("%s returned an invalid version %q", url, version)
	}
	return normalizeVersion(version), nil
}

// Install downloads devbox version unless it's already installed, and returns
// the path to its binary. It prints progress to w.
func Install(ctx context.Context, version string, w io.Writer) (string, error) {
	if !validVersion(version) {
		return "", fmt.Errorf("invalid devbox version %q", version)
	}
	if installDir() == "" {
		return "", errors.New("can't install devbox: the home directory is unknown")
	}
	version = normalizeVersion(version)
	bin := BinaryPath(version)
	if isExecutable(bin) {
		return bin, nil
	}

	fmt.Fprintf(w, "Downloading devbox %s...\n", version)
	if err := download(ctx, version, filepath.Dir(bin)); err != nil {
		return "", fmt.Errorf("download devbox %s: %w", version, err)
	}
	fmt.Fprintf(w, "Downloaded devbox %s.\n", version)
	return bin, nil
}

// download installs the release archive for version into dir. It unpacks the
// archive into a temporary directory and then renames that to dir, so that
// dir only ever holds a complete binary.
func download(ctx context.Context, version, dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".download-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	archiveName := fmt.Sprintf("devbox_%s_%s.tar.gz", version, platform())
	wantSum, err := fetchChecksum(ctx, version, archiveName)
	if err != nil {
		return err
	}

	archivePath := filepath.Join(tmp, archiveName)
	gotSum, err := fetchFile(ctx, fmt.Sprintf("%s/%s/%s", downloadURL, version, archiveName), archivePath)
	if err != nil {
		return err
	}
	if gotSum != wantSum {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", archiveName, wantSum, gotSum)
	}

	unpacked := filepath.Join(tmp, "unpacked")
	if err := os.Mkdir(unpacked, 0o755); err != nil {
		return err
	}
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	err = fileutil.Untar(archive, unpacked)
	archive.Close()
	if err != nil {
		return fmt.Errorf("unpack %s: %w", archiveName, err)
	}
	if !isExecutable(filepath.Join(unpacked, "devbox")) {
		return fmt.Errorf("%s doesn't contain a devbox binary", archiveName)
	}

	// Remove whatever is left of an earlier failed install.
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.Rename(unpacked, dir); err != nil {
		// Another launcher may have installed the same version first.
		if isExecutable(filepath.Join(dir, "devbox")) {
			return nil
		}
		return err
	}
	return nil
}

// fetchChecksum returns the SHA-256 checksum of archiveName from the
// release's checksums.txt.
func fetchChecksum(ctx context.Context, version, archiveName string) (string, error) {
	url := fmt.Sprintf("%s/%s/checksums.txt", downloadURL, version)
	resp, err := get(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[1] == archiveName {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	return "", fmt.Errorf("%s has no checksum for %s", url, archiveName)
}

// fetchFile downloads url to path and returns the file's SHA-256 checksum.
func fetchFile(ctx context.Context, url, path string) (string, error) {
	resp, err := get(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, hash), resp.Body)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// get returns the response to a GET request for url, which the caller must
// close. It returns an error unless the status is 200 OK.
func get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return resp, nil
}
