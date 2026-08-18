package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var githubAPIBase = "https://api.github.com"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Overridable in tests.
var (
	osExecutable  = os.Executable
	osMkdirTemp   = os.MkdirTemp
	osUserHomeDir = os.UserHomeDir
	ioCopy        = io.Copy
	osOpenFile    = os.OpenFile
	osCreateTemp  = os.CreateTemp
	osRename      = os.Rename
)

const githubRepo = "davidbudnick/redis-tui"

// maxDownloadSize is the maximum allowed download size (256 MB).
const maxDownloadSize = 256 << 20

// maxBinarySize is the maximum allowed extracted binary size (128 MB).
const maxBinarySize = 128 << 20

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func runUpdate(currentVersion string) error {
	if currentVersion == "dev" || !isSemver(currentVersion) {
		return fmt.Errorf("cannot self-update a development build (version=%q); use the install script instead:\n  curl -fsSL https://raw.githubusercontent.com/davidbudnick/redis-tui/main/install.sh | bash", currentVersion)
	}

	execPath, err := osExecutable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	if isHomebrew(execPath) {
		return fmt.Errorf("this binary was installed via Homebrew; please update with:\n  brew upgrade redis-tui")
	}

	if err := checkWriteAccess(execPath); err != nil {
		home, homeErr := osUserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("cannot write to %s and could not determine home directory: %w", execPath, homeErr)
		}
		localBin := filepath.Join(home, ".local", "bin")
		if mkErr := os.MkdirAll(localBin, 0750); mkErr != nil {
			return fmt.Errorf("cannot write to %s and could not create %s: %w", execPath, localBin, mkErr)
		}
		execPath = filepath.Join(localBin, "redis-tui")
		fmt.Printf("No write access to current location, installing to %s\n", execPath)
	}

	latest, err := fetchLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to fetch latest version: %w", err)
	}

	if strings.TrimPrefix(latest, "v") == strings.TrimPrefix(currentVersion, "v") {
		fmt.Printf("Already up to date (v%s).\n", strings.TrimPrefix(currentVersion, "v"))
		return nil
	}

	ver := strings.TrimPrefix(latest, "v")
	archive := archiveName(ver, runtime.GOOS, runtime.GOARCH)
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", githubRepo, latest)
	archiveURL := baseURL + "/" + archive
	checksumURL := baseURL + "/checksums.txt"

	tmpDir, err := osMkdirTemp("", "redis-tui-update-*")
	if err != nil {
		return fmt.Errorf("could not create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archive)
	checksumPath := filepath.Join(tmpDir, "checksums.txt")

	fmt.Printf("Downloading redis-tui v%s...\n", ver)

	if err := downloadFile(archiveURL, archivePath); err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}
	if err := downloadFile(checksumURL, checksumPath); err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	if err := verifyChecksum(archivePath, checksumPath, archive); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	newBinaryPath := filepath.Join(tmpDir, "redis-tui")
	if err := extractBinary(archivePath, newBinaryPath); err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}

	if err := replaceBinary(execPath, newBinaryPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("Successfully updated to v%s.\n", ver)
	return nil
}

func fetchLatestVersion() (string, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, githubRepo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("could not create request: %w", err)
	}

	resp, err := httpClient.Do(req) // #nosec G704 - URL built from hardcoded GitHub API base
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in response")
	}

	return release.TagName, nil
}

func archiveName(ver, goos, goarch string) string {
	osName := strings.ToUpper(goos[:1]) + goos[1:]
	arch := goarch
	if goarch == "amd64" {
		arch = "x86_64"
	}

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("redis-tui_%s_%s_%s.%s", ver, osName, arch, ext)
}

func downloadFile(rawURL, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	resp, err := httpClient.Do(req) // #nosec G704 - URL built from hardcoded GitHub release base
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	cleanPath := filepath.Clean(destPath)
	f, err := os.Create(cleanPath)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, io.LimitReader(resp.Body, maxDownloadSize)); err != nil {
		return fmt.Errorf("could not write file: %w", err)
	}

	return nil
}

func verifyChecksum(archivePath, checksumPath, archiveFilename string) error {
	cleanChecksumPath := filepath.Clean(checksumPath)
	data, err := os.ReadFile(cleanChecksumPath) // #nosec G304 - path constructed from os.MkdirTemp
	if err != nil {
		return fmt.Errorf("could not read checksums file: %w", err)
	}

	var expectedHash string
	for line := range strings.SplitSeq(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == archiveFilename {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("no checksum found for %s", archiveFilename)
	}

	cleanArchivePath := filepath.Clean(archivePath)
	f, err := os.Open(cleanArchivePath) // #nosec G304 - path constructed from os.MkdirTemp
	if err != nil {
		return fmt.Errorf("could not open archive: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := ioCopy(h, f); err != nil {
		return fmt.Errorf("could not hash archive: %w", err)
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

func extractBinary(archivePath, destPath string) error {
	cleanArchivePath := filepath.Clean(archivePath)
	f, err := os.Open(cleanArchivePath) // #nosec G304 - path constructed from os.MkdirTemp
	if err != nil {
		return fmt.Errorf("could not open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("could not open gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read tar entry: %w", err)
		}

		if filepath.Base(hdr.Name) == "redis-tui" && hdr.Typeflag == tar.TypeReg {
			cleanDestPath := filepath.Clean(destPath)
			out, err := osOpenFile(cleanDestPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700) // #nosec G302 - binary must be executable
			if err != nil {
				return fmt.Errorf("could not create binary: %w", err)
			}

			if _, err := io.Copy(out, io.LimitReader(tr, maxBinarySize)); err != nil {
				_ = out.Close() // #nosec G104 - best-effort close on write error
				return fmt.Errorf("could not write binary: %w", err)
			}

			return out.Close()
		}
	}

	return fmt.Errorf("binary not found in archive")
}

func replaceBinary(currentPath, newPath string) error {
	oldPath := currentPath + ".old"

	// Back up existing binary if it exists
	hasBackup := true
	if err := os.Rename(currentPath, oldPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("could not back up current binary: %w", err)
		}
		hasBackup = false
	}

	if err := installBinary(newPath, currentPath); err != nil {
		if hasBackup {
			_ = os.Rename(oldPath, currentPath)
		}
		return fmt.Errorf("could not install new binary: %w", err)
	}

	if hasBackup {
		_ = os.Remove(oldPath)
	}
	return nil
}

// installBinary moves src onto dest, copying when rename cannot cross devices.
func installBinary(src, dest string) error {
	if err := osRename(src, dest); err == nil {
		return nil
	} else if !isCrossDevice(err) {
		return err
	}
	return copyFile(src, dest)
}

// isCrossDevice reports whether err is a cross-device rename failure.
func isCrossDevice(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}

// copyFile copies src to dest with executable permissions.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open new binary: %w", err)
	}
	defer in.Close()

	out, err := osOpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700) // #nosec G302 - binary must be executable
	if err != nil {
		return fmt.Errorf("could not create destination: %w", err)
	}

	if _, err := ioCopy(out, in); err != nil {
		_ = out.Close() // #nosec G104 - best-effort close on write error
		_ = os.Remove(dest)
		return fmt.Errorf("could not write destination: %w", err)
	}

	return out.Close()
}

func isSemver(s string) bool {
	s = strings.TrimPrefix(s, "v")
	matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+`, s)
	return matched
}

func isHomebrew(path string) bool {
	return strings.Contains(path, "/Cellar/") || strings.Contains(path, "/homebrew/")
}

func checkWriteAccess(path string) error {
	dir := filepath.Dir(path)
	tmp, err := osCreateTemp(dir, ".redis-tui-write-check-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Remove(filepath.Clean(name)) // #nosec G703 - name from os.CreateTemp in a known directory
}
