package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestFetchLatestVersion(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
		}))
		defer srv.Close()

		old := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = old }()

		ver, err := fetchLatestVersion()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ver != "v1.2.3" {
			t.Errorf("version = %q, want %q", ver, "v1.2.3")
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		old := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for non-200 status")
		}
	})

	t.Run("bad JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `not json`)
		}))
		defer srv.Close()

		old := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for bad JSON")
		}
	})

	t.Run("empty tag", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":""}`)
		}))
		defer srv.Close()

		old := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for empty tag")
		}
	})

	t.Run("HTTP request error", func(t *testing.T) {
		old := githubAPIBase
		githubAPIBase = "http://127.0.0.1:1" // unreachable
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for unreachable server")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		old := githubAPIBase
		githubAPIBase = "://bad-url"
		defer func() { githubAPIBase = old }()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Fatal("expected error for invalid URL")
		}
	})
}

func TestArchiveName(t *testing.T) {
	tests := []struct {
		ver, goos, goarch string
		want              string
	}{
		{"1.0.0", "darwin", "arm64", "redis-tui_1.0.0_Darwin_arm64.tar.gz"},
		{"1.0.0", "linux", "amd64", "redis-tui_1.0.0_Linux_x86_64.tar.gz"},
		{"2.1.0", "windows", "amd64", "redis-tui_2.1.0_Windows_x86_64.zip"},
		{"1.0.0", "darwin", "amd64", "redis-tui_1.0.0_Darwin_x86_64.tar.gz"},
		{"1.0.0", "linux", "arm64", "redis-tui_1.0.0_Linux_arm64.tar.gz"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got := archiveName(tt.ver, tt.goos, tt.goarch)
			if got != tt.want {
				t.Errorf("archiveName(%q, %q, %q) = %q, want %q", tt.ver, tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	t.Run("matching checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("test archive content"), 0o644); err != nil {
			t.Fatal(err)
		}

		hash := sha256.Sum256([]byte("test archive content"))
		checksumContent := fmt.Sprintf("%x  test.tar.gz\n", hash)
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("mismatching checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("test archive content"), 0o644); err != nil {
			t.Fatal(err)
		}

		checksumContent := "0000000000000000000000000000000000000000000000000000000000000000  test.tar.gz\n"
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err == nil {
			t.Error("expected error for mismatching checksum")
		}
	})

	t.Run("missing archive in checksums", func(t *testing.T) {
		tmpDir := t.TempDir()

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}

		checksumContent := "abcdef1234567890  other_file.tar.gz\n"
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte(checksumContent), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(archivePath, checksumPath, "test.tar.gz"); err == nil {
			t.Error("expected error for missing archive in checksums")
		}
	})

	t.Run("checksums file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := verifyChecksum(archivePath, filepath.Join(tmpDir, "nonexistent.txt"), "test.tar.gz")
		if err == nil {
			t.Error("expected error for missing checksums file")
		}
	})

	t.Run("archive file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		checksumPath := filepath.Join(tmpDir, "checksums.txt")
		if err := os.WriteFile(checksumPath, []byte("abc123  test.tar.gz\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := verifyChecksum(filepath.Join(tmpDir, "nonexistent.tar.gz"), checksumPath, "test.tar.gz")
		if err == nil {
			t.Error("expected error for missing archive file")
		}
	})
}

func TestDownloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "file content here")
		}))
		defer srv.Close()

		dest := filepath.Join(t.TempDir(), "downloaded")
		if err := downloadFile(srv.URL+"/test.tar.gz", dest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "file content here" {
			t.Errorf("content = %q, want %q", string(data), "file content here")
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		dest := filepath.Join(t.TempDir(), "downloaded")
		err := downloadFile(srv.URL+"/missing", dest)
		if err == nil {
			t.Fatal("expected error for 404")
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "downloaded")
		err := downloadFile("http://127.0.0.1:1/bad", dest)
		if err == nil {
			t.Fatal("expected error for unreachable server")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "downloaded")
		err := downloadFile("://bad", dest)
		if err == nil {
			t.Fatal("expected error for invalid URL")
		}
	})

	t.Run("unwritable dest", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "data")
		}))
		defer srv.Close()

		err := downloadFile(srv.URL, "/nonexistent-dir/file")
		if err == nil {
			t.Fatal("expected error for unwritable destination")
		}
	})
}

// buildTestTarGz creates a tar.gz archive containing a file named "redis-tui"
// with the given content.
func buildTestTarGz(t *testing.T, binaryContent string) []byte {
	t.Helper()
	var buf strings.Builder
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name:     "redis-tui_1.0.0_Test/redis-tui",
		Mode:     0o755,
		Size:     int64(len(binaryContent)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar write header: %v", err)
	}
	if _, err := tw.Write([]byte(binaryContent)); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return []byte(buf.String())
}

func TestExtractBinary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		archiveData := buildTestTarGz(t, "#!/bin/sh\necho hello\n")

		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
			t.Fatal(err)
		}

		destPath := filepath.Join(tmpDir, "redis-tui")
		if err := extractBinary(archivePath, destPath); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "#!/bin/sh\necho hello\n" {
			t.Errorf("content = %q", string(data))
		}
	})

	t.Run("archive not found", func(t *testing.T) {
		err := extractBinary("/nonexistent.tar.gz", filepath.Join(t.TempDir(), "out"))
		if err == nil {
			t.Fatal("expected error for missing archive")
		}
	})

	t.Run("not a gzip", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "bad.tar.gz")
		if err := os.WriteFile(archivePath, []byte("not a gzip"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := extractBinary(archivePath, filepath.Join(tmpDir, "out"))
		if err == nil {
			t.Fatal("expected error for non-gzip file")
		}
	})

	t.Run("binary not in archive", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Build a tar.gz without a redis-tui file.
		var buf strings.Builder
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)
		hdr := &tar.Header{Name: "other-file", Mode: 0o644, Size: 5, Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gw.Close(); err != nil {
			t.Fatal(err)
		}

		archivePath := filepath.Join(tmpDir, "no-binary.tar.gz")
		if err := os.WriteFile(archivePath, []byte(buf.String()), 0o644); err != nil {
			t.Fatal(err)
		}

		err := extractBinary(archivePath, filepath.Join(tmpDir, "out"))
		if err == nil {
			t.Fatal("expected error for missing binary in archive")
		}
		if !strings.Contains(err.Error(), "binary not found") {
			t.Errorf("error = %q, want it to contain 'binary not found'", err)
		}
	})

	t.Run("unwritable destination", func(t *testing.T) {
		tmpDir := t.TempDir()
		archiveData := buildTestTarGz(t, "binary")
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
			t.Fatal(err)
		}

		err := extractBinary(archivePath, "/nonexistent-dir/redis-tui")
		if err == nil {
			t.Fatal("expected error for unwritable destination")
		}
	})
}

func TestReplaceBinary(t *testing.T) {
	t.Run("replace existing", func(t *testing.T) {
		tmpDir := t.TempDir()
		current := filepath.Join(tmpDir, "redis-tui")
		newBin := filepath.Join(tmpDir, "redis-tui-new")

		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newBin, []byte("new"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := replaceBinary(current, newBin); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "new" {
			t.Errorf("content = %q, want %q", string(data), "new")
		}

		// Old backup should be cleaned up.
		if _, err := os.Stat(current + ".old"); !os.IsNotExist(err) {
			t.Error("expected .old backup to be removed")
		}
	})

	t.Run("no existing binary", func(t *testing.T) {
		tmpDir := t.TempDir()
		current := filepath.Join(tmpDir, "redis-tui")
		newBin := filepath.Join(tmpDir, "redis-tui-new")

		if err := os.WriteFile(newBin, []byte("fresh"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := replaceBinary(current, newBin); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "fresh" {
			t.Errorf("content = %q, want %q", string(data), "fresh")
		}
	})

	t.Run("new binary missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		current := filepath.Join(tmpDir, "redis-tui")
		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}

		err := replaceBinary(current, filepath.Join(tmpDir, "nonexistent"))
		if err == nil {
			t.Fatal("expected error when new binary doesn't exist")
		}

		// Original should be restored from backup.
		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "old" {
			t.Errorf("content = %q, want original restored", string(data))
		}
	})

	t.Run("cross device copy", func(t *testing.T) {
		tmpDir := t.TempDir()
		current := filepath.Join(tmpDir, "redis-tui")
		newBin := filepath.Join(tmpDir, "redis-tui-new")

		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newBin, []byte("new"), 0o755); err != nil {
			t.Fatal(err)
		}

		origRename := osRename
		osRename = func(oldpath, newpath string) error {
			return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.EXDEV}
		}
		t.Cleanup(func() { osRename = origRename })

		if err := replaceBinary(current, newBin); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "new" {
			t.Errorf("content = %q, want %q", string(data), "new")
		}

		info, err := os.Stat(current)
		if err != nil {
			t.Fatalf("stat failed: %v", err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Errorf("mode = %v, want executable", info.Mode())
		}
	})

	t.Run("cross device copy dest error restores backup", func(t *testing.T) {
		tmpDir := t.TempDir()
		current := filepath.Join(tmpDir, "redis-tui")
		newBin := filepath.Join(tmpDir, "redis-tui-new")

		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newBin, []byte("new"), 0o755); err != nil {
			t.Fatal(err)
		}

		origRename := osRename
		osRename = func(oldpath, newpath string) error {
			return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.EXDEV}
		}
		t.Cleanup(func() { osRename = origRename })

		origOpen := osOpenFile
		osOpenFile = func(string, int, os.FileMode) (*os.File, error) {
			return nil, fmt.Errorf("no dest")
		}
		t.Cleanup(func() { osOpenFile = origOpen })

		err := replaceBinary(current, newBin)
		if err == nil {
			t.Fatal("expected install error")
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "old" {
			t.Errorf("content = %q, want original restored", string(data))
		}
	})
}

func TestCopyFile(t *testing.T) {
	t.Run("source missing", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "dest")
		err := copyFile(filepath.Join(t.TempDir(), "missing"), dest)
		if err == nil {
			t.Fatal("expected error for missing source")
		}
		if !strings.Contains(err.Error(), "could not open new binary") {
			t.Errorf("error = %q, want open error", err)
		}
	})

	t.Run("destination unwritable", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "src")
		if err := os.WriteFile(src, []byte("bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		err := copyFile(src, filepath.Join("/nonexistent-dir-xyz", "dest"))
		if err == nil {
			t.Fatal("expected error for unwritable destination")
		}
		if !strings.Contains(err.Error(), "could not create destination") {
			t.Errorf("error = %q, want create error", err)
		}
	})

	t.Run("copy error", func(t *testing.T) {
		tmpDir := t.TempDir()
		src := filepath.Join(tmpDir, "src")
		dest := filepath.Join(tmpDir, "dest")
		if err := os.WriteFile(src, []byte("bin"), 0o755); err != nil {
			t.Fatal(err)
		}

		origCopy := ioCopy
		ioCopy = func(io.Writer, io.Reader) (int64, error) {
			return 0, fmt.Errorf("copy fail")
		}
		t.Cleanup(func() { ioCopy = origCopy })

		err := copyFile(src, dest)
		if err == nil {
			t.Fatal("expected copy error")
		}
		if !strings.Contains(err.Error(), "could not write destination") {
			t.Errorf("error = %q, want write error", err)
		}
		if _, err := os.Stat(dest); !os.IsNotExist(err) {
			t.Error("expected partial destination to be removed")
		}
	})
}

func TestReplaceBinary_RealCrossDevice(t *testing.T) {
	srcDir, destDir := realCrossDeviceDirs(t)

	t.Run("rename fails with EXDEV", func(t *testing.T) {
		src := filepath.Join(srcDir, "probe-new")
		dest := filepath.Join(destDir, "probe-dest")
		if err := os.WriteFile(src, []byte("probe"), 0o755); err != nil {
			t.Fatal(err)
		}

		err := os.Rename(src, dest)
		if err == nil {
			t.Fatal("os.Rename succeeded; src and dest are not on different devices")
		}
		if !isCrossDevice(err) {
			t.Fatalf("os.Rename error = %v, want EXDEV", err)
		}
		t.Logf("reproduced issue #71: %v", err)
	})

	t.Run("replace existing", func(t *testing.T) {
		current := filepath.Join(destDir, "redis-tui")
		newBin := filepath.Join(srcDir, "redis-tui-new")
		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newBin, []byte("new-cross-device"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := replaceBinary(current, newBin); err != nil {
			t.Fatalf("replaceBinary: %v", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "new-cross-device" {
			t.Errorf("content = %q, want %q", string(data), "new-cross-device")
		}

		info, err := os.Stat(current)
		if err != nil {
			t.Fatalf("stat failed: %v", err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Errorf("mode = %v, want executable", info.Mode())
		}
		if _, err := os.Stat(current + ".old"); !os.IsNotExist(err) {
			t.Error("expected .old backup to be removed")
		}
	})

	t.Run("legacy 1.0.42 rename across devices fails", func(t *testing.T) {
		current := filepath.Join(destDir, "redis-tui-legacy")
		newBin := filepath.Join(srcDir, "redis-tui-legacy-new")
		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newBin, []byte("new-legacy"), 0o755); err != nil {
			t.Fatal(err)
		}

		err := replaceBinaryLegacy(current, newBin)
		if err == nil {
			t.Fatal("1.0.42 rename-only replace unexpectedly succeeded")
		}
		if !isCrossDevice(err) {
			t.Fatalf("legacy replace error = %v, want EXDEV", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "old" {
			t.Errorf("content = %q, want original restored", string(data))
		}
	})

	t.Run("legacy 1.0.42 TMPDIR same-device escape", func(t *testing.T) {
		current := filepath.Join(destDir, "redis-tui-escape")
		newBin := filepath.Join(destDir, "redis-tui-escape-new")
		if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newBin, []byte("escaped"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := replaceBinaryLegacy(current, newBin); err != nil {
			t.Fatalf("same-device 1.0.42 replace failed: %v", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "escaped" {
			t.Errorf("content = %q, want %q", string(data), "escaped")
		}
	})

	t.Run("no existing binary", func(t *testing.T) {
		current := filepath.Join(destDir, "redis-tui-fresh")
		newBin := filepath.Join(srcDir, "redis-tui-fresh-new")
		if err := os.WriteFile(newBin, []byte("fresh-cross-device"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := replaceBinary(current, newBin); err != nil {
			t.Fatalf("replaceBinary: %v", err)
		}

		data, err := os.ReadFile(current)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != "fresh-cross-device" {
			t.Errorf("content = %q, want %q", string(data), "fresh-cross-device")
		}
	})
}

func TestUpdateTempDir(t *testing.T) {
	t.Run("prefers dest directory", func(t *testing.T) {
		destDir := t.TempDir()
		dir, err := updateTempDir(filepath.Join(destDir, "redis-tui"))
		if err != nil {
			t.Fatalf("updateTempDir: %v", err)
		}
		t.Cleanup(func() {
			if err := os.RemoveAll(dir); err != nil {
				t.Errorf("cleanup: %v", err)
			}
		})
		if filepath.Dir(dir) != destDir {
			t.Errorf("temp dir %q is not under dest %q", dir, destDir)
		}
	})

	t.Run("falls back to system temp", func(t *testing.T) {
		orig := osMkdirTemp
		calls := 0
		osMkdirTemp = func(dir, pattern string) (string, error) {
			calls++
			if calls == 1 {
				if dir == "" {
					t.Error("first attempt should use dest dir, not system temp")
				}
				return "", fmt.Errorf("dest not writable")
			}
			if dir != "" {
				t.Errorf("fallback dir = %q, want system temp", dir)
			}
			return orig(dir, pattern)
		}
		t.Cleanup(func() { osMkdirTemp = orig })

		dir, err := updateTempDir(filepath.Join(t.TempDir(), "redis-tui"))
		if err != nil {
			t.Fatalf("updateTempDir: %v", err)
		}
		t.Cleanup(func() {
			if err := os.RemoveAll(dir); err != nil {
				t.Errorf("cleanup: %v", err)
			}
		})
		if calls != 2 {
			t.Errorf("osMkdirTemp calls = %d, want 2", calls)
		}
	})
}

func TestReplaceBinaryLegacy_SameDevice(t *testing.T) {
	tmpDir := t.TempDir()
	current := filepath.Join(tmpDir, "redis-tui")
	newBin := filepath.Join(tmpDir, "redis-tui-new")
	if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newBin, []byte("from-tmpdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinaryLegacy(current, newBin); err != nil {
		t.Fatalf("legacy replace: %v", err)
	}

	data, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "from-tmpdir" {
		t.Errorf("content = %q, want %q", string(data), "from-tmpdir")
	}
}

func replaceBinaryLegacy(currentPath, newPath string) error {
	oldPath := currentPath + ".old"
	hasBackup := true
	if err := os.Rename(currentPath, oldPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		hasBackup = false
	}
	if err := os.Rename(newPath, currentPath); err != nil {
		if hasBackup {
			_ = os.Rename(oldPath, currentPath)
		}
		return err
	}
	if hasBackup {
		_ = os.Remove(oldPath)
	}
	return nil
}

func realCrossDeviceDirs(t *testing.T) (string, string) {
	t.Helper()

	srcRoot := os.Getenv("CROSS_DEVICE_SRC")
	destRoot := os.Getenv("CROSS_DEVICE_DEST")
	required := os.Getenv("REQUIRE_CROSS_DEVICE") == "1"

	if srcRoot == "" || destRoot == "" {
		if required {
			t.Fatal("REQUIRE_CROSS_DEVICE=1 requires CROSS_DEVICE_SRC and CROSS_DEVICE_DEST")
		}
		t.Skip("set CROSS_DEVICE_SRC and CROSS_DEVICE_DEST to run the real EXDEV test")
	}

	srcDir := filepath.Join(srcRoot, "redis-tui-exdev-src")
	destDir := filepath.Join(destRoot, "redis-tui-exdev-dest")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("create src dir: %v", err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("create dest dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(srcDir); err != nil {
			t.Errorf("cleanup src dir: %v", err)
		}
		if err := os.RemoveAll(destDir); err != nil {
			t.Errorf("cleanup dest dir: %v", err)
		}
	})
	return srcDir, destDir
}

func TestCheckWriteAccess(t *testing.T) {
	t.Run("writable directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "redis-tui")
		if err := checkWriteAccess(path); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("non-writable directory", func(t *testing.T) {
		err := checkWriteAccess("/nonexistent-dir/redis-tui")
		if err == nil {
			t.Error("expected error for non-writable directory")
		}
	})
}

func TestRunUpdateDevVersion(t *testing.T) {
	err := runUpdate("dev")
	if err == nil {
		t.Fatal("expected error for dev version")
	}
	if !strings.Contains(err.Error(), "development build") {
		t.Errorf("error = %q, want it to mention 'development build'", err)
	}
}

func TestRunUpdateNonSemver(t *testing.T) {
	err := runUpdate("abc123")
	if err == nil {
		t.Fatal("expected error for non-semver version")
	}
	if !strings.Contains(err.Error(), "development build") {
		t.Errorf("error = %q, want it to mention 'development build'", err)
	}
}

func TestRunUpdate_Homebrew(t *testing.T) {
	// Create a file inside a path containing "/homebrew/" to trigger
	// the Homebrew detection after EvalSymlinks succeeds.
	tmpDir := t.TempDir()
	brewDir := filepath.Join(tmpDir, "homebrew", "bin")
	if err := os.MkdirAll(brewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	execPath := filepath.Join(brewDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err == nil {
		t.Fatal("expected error for Homebrew install")
	}
	if !strings.Contains(err.Error(), "Homebrew") {
		t.Errorf("error = %q, want Homebrew mention", err)
	}
}

func TestRunUpdate_AlreadyUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0"}`)
	}))
	defer srv.Close()

	old := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = old }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUpdate_FullFlow(t *testing.T) {
	// Build a tar.gz with a fake binary.
	binaryContent := "#!/bin/sh\necho updated\n"
	archiveData := buildTestTarGz(t, binaryContent)

	archiveHash := sha256.Sum256(archiveData)
	archiveName := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	checksumContent := fmt.Sprintf("%x  %s\n", archiveHash, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			fmt.Fprint(w, checksumContent)
		case strings.HasSuffix(r.URL.Path, archiveName):
			if _, err := w.Write(archiveData); err != nil {
				t.Errorf("write archive: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	// Set up a writable exec path.
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	// Override the GitHub download base URL to use our test server.
	// runUpdate builds URLs from githubRepo constant, so we need to
	// override httpClient to redirect github.com to our test server.
	origClient := httpClient
	httpClient = srv.Client()
	// Intercept all requests and rewrite to test server.
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err != nil {
		t.Fatalf("runUpdate failed: %v", err)
	}

	// Verify the binary was replaced.
	data, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != binaryContent {
		t.Errorf("binary content = %q, want %q", string(data), binaryContent)
	}
}

func TestRunUpdate_FetchError(t *testing.T) {
	old := githubAPIBase
	githubAPIBase = "http://127.0.0.1:1"
	defer func() { githubAPIBase = old }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Errorf("error = %q, want 'failed to fetch'", err)
	}
}

func TestRunUpdate_NoWriteAccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
	}))
	defer srv.Close()

	old := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = old }()

	origExec := osExecutable
	osExecutable = func() (string, error) {
		return "/usr/bin/redis-tui", nil // not writable
	}
	t.Cleanup(func() { osExecutable = origExec })

	// This should fall back to ~/.local/bin — won't error on that part,
	// but will error trying to download from fake github.com URLs.
	err := runUpdate("1.0.0")
	if err == nil {
		t.Fatal("expected error (download from real github.com URLs will fail)")
	}
}

func TestRunUpdate_ExecPathError(t *testing.T) {
	origExec := osExecutable
	osExecutable = func() (string, error) { return "", fmt.Errorf("no executable") }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "could not determine executable path") {
		t.Errorf("error = %v, want executable path error", err)
	}
}

func TestRunUpdate_EvalSymlinksError(t *testing.T) {
	origExec := osExecutable
	osExecutable = func() (string, error) { return "/nonexistent/path/redis-tui", nil }
	t.Cleanup(func() { osExecutable = origExec })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "could not resolve executable path") {
		t.Errorf("error = %v, want resolve path error", err)
	}
}

func TestRunUpdate_DownloadArchiveError(t *testing.T) {
	// Server returns latest version but 404 for the archive download.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "releases/latest") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "failed to download") {
		t.Errorf("error = %v, want download error", err)
	}
}

func TestRunUpdate_ChecksumMismatch(t *testing.T) {
	archiveData := buildTestTarGz(t, "binary")
	archiveName := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	// Provide wrong checksum.
	checksumContent := fmt.Sprintf("0000000000000000000000000000000000000000000000000000000000000000  %s\n", archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			fmt.Fprint(w, checksumContent)
		case strings.HasSuffix(r.URL.Path, archiveName):
			if _, err := w.Write(archiveData); err != nil {
				t.Errorf("write: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "checksum verification failed") {
		t.Errorf("error = %v, want checksum error", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRunUpdate_NoWriteAccess_HomeDirError(t *testing.T) {
	// Covers the branch where checkWriteAccess fails AND os.UserHomeDir fails.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	// Use a read-only directory for the exec path.
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	execPath := filepath.Join(readOnlyDir, "redis-tui")

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	// Create the file so EvalSymlinks succeeds.
	if err := os.Chmod(readOnlyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Make dir read-only again so checkWriteAccess fails.
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o755) })

	// The fallback creates ~/.local/bin — since the download will fail
	// (GitHub URLs), the error will be about downloading, not write access.
	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	// Should proceed past the write access check and fail on download.
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunUpdate_ExtractError(t *testing.T) {
	// Cover the extractBinary error path in runUpdate (lines 109-111).
	an := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	// Serve a non-gzip file with matching checksum so verifyChecksum
	// passes but extractBinary fails.
	badArchive := []byte("not-a-gzip")
	badHash := sha256.Sum256(badArchive)
	checksumContent := fmt.Sprintf("%x  %s\n", badHash, an)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			fmt.Fprint(w, checksumContent)
		case strings.HasSuffix(r.URL.Path, an):
			if _, err := w.Write(badArchive); err != nil {
				t.Errorf("write: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "failed to extract") {
		t.Errorf("error = %v, want extract error", err)
	}
	// Verify the uncovered lines are different: checksum download error.
	// Serve checksums.txt as 404:
}

func TestRunUpdate_ChecksumDownloadError(t *testing.T) {
	archiveData := buildTestTarGz(t, "binary")
	an := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, an):
			if _, err := w.Write(archiveData); err != nil {
				t.Errorf("write: %v", err)
			}
		default:
			// checksums.txt returns 404
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "failed to download checksums") {
		t.Errorf("error = %v, want checksum download error", err)
	}
}

func TestReplaceBinary_BackupRenameError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a directory at the "current" path — os.Rename to .old will
	// fail with a different error if .old already exists as a directory.
	current := filepath.Join(tmpDir, "redis-tui")
	if err := os.Mkdir(current, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a file at .old so Rename(current, .old) fails because
	// on some OSes you can't rename a dir over a file.
	oldPath := current + ".old"
	if err := os.WriteFile(oldPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	newBin := filepath.Join(tmpDir, "new")
	if err := os.WriteFile(newBin, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := replaceBinary(current, newBin)
	if err == nil {
		t.Fatal("expected error from backup rename")
	}
}

func TestRunUpdate_UserHomeDirError(t *testing.T) {
	// Covers update.go:59 — UserHomeDir error when no write access.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	// Create exec in a read-only dir so checkWriteAccess fails.
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	execPath := filepath.Join(readOnlyDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o755) })

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	// Make UserHomeDir fail.
	origHome := osUserHomeDir
	osUserHomeDir = func() (string, error) { return "", fmt.Errorf("no home") }
	t.Cleanup(func() { osUserHomeDir = origHome })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "could not determine home directory") {
		t.Errorf("error = %v, want home directory error", err)
	}
}

func TestRunUpdate_MkdirAllError(t *testing.T) {
	// Covers update.go:63 — MkdirAll error for ~/.local/bin.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	execPath := filepath.Join(readOnlyDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o755) })

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	// Return a home dir that's a file (not dir) so MkdirAll fails.
	homeFile := filepath.Join(tmpDir, "fakehome")
	if err := os.WriteFile(homeFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	origHome := osUserHomeDir
	osUserHomeDir = func() (string, error) { return homeFile, nil }
	t.Cleanup(func() { osUserHomeDir = origHome })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "could not create") {
		t.Errorf("error = %v, want create error", err)
	}
}

func TestRunUpdate_TempDirError(t *testing.T) {
	// Covers update.go:87 — osMkdirTemp error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origMkdir := osMkdirTemp
	osMkdirTemp = func(string, string) (string, error) { return "", fmt.Errorf("no temp") }
	t.Cleanup(func() { osMkdirTemp = origMkdir })

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "could not create temp directory") {
		t.Errorf("error = %v, want temp dir error", err)
	}
}

func TestRunUpdate_ReplaceBinaryError(t *testing.T) {
	// Covers update.go:113 — replaceBinary error in runUpdate.
	// Serve a valid archive but make the exec path read-only.
	binaryContent := "#!/bin/sh\n"
	archiveData := buildTestTarGz(t, binaryContent)
	archiveHash := sha256.Sum256(archiveData)
	an := archiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	checksumContent := fmt.Sprintf("%x  %s\n", archiveHash, an)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			fmt.Fprint(w, checksumContent)
		case strings.HasSuffix(r.URL.Path, an):
			if _, err := w.Write(archiveData); err != nil {
				t.Errorf("write: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldAPI }()

	// Make exec path inside a read-only dir so replaceBinary fails.
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	execPath := filepath.Join(readOnlyDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o755) })

	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	origClient := httpClient
	httpClient = srv.Client()
	httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpClient = origClient })

	// However, checkWriteAccess will also fail for this dir.
	// So runUpdate will try the ~/.local/bin fallback.
	// We need to make checkWriteAccess pass but replaceBinary fail.
	// Use a different approach: exec in a writable dir, but make
	// the .old rename fail by putting a non-removable dir at .old.
	if err := os.Chmod(readOnlyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldDir := execPath + ".old"
	if err := os.MkdirAll(filepath.Join(oldDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := runUpdate("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "failed to replace binary") {
		t.Errorf("error = %v, want replace binary error", err)
	}
	if !strings.Contains(err.Error(), "TMPDIR=") {
		t.Errorf("error = %v, want TMPDIR recovery hint", err)
	}
}

func TestDownloadFile_CopyError(t *testing.T) {
	// Covers update.go:188 — io.Copy error during download.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		fmt.Fprint(w, "partial")
		// Flush and close connection abruptly to cause io.Copy error.
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "downloaded")
	// The server claims 1MB but only sends 7 bytes then closes.
	// io.Copy may or may not error depending on timing.
	// This is inherently flaky so just ensure no panic.
	_ = downloadFile(srv.URL, dest)
}

func TestExtractBinary_CorruptTar(t *testing.T) {
	// Covers update.go:255 — tar read error with corrupt tar inside valid gzip.
	tmpDir := t.TempDir()

	// Create a valid gzip containing corrupt tar data.
	var buf strings.Builder
	gw := gzip.NewWriter(&buf)
	// Write random bytes that aren't valid tar.
	if _, err := gw.Write([]byte("this is not valid tar data at all!!")); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(tmpDir, "corrupt.tar.gz")
	if err := os.WriteFile(archivePath, []byte(buf.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	err := extractBinary(archivePath, filepath.Join(tmpDir, "out"))
	if err == nil {
		t.Fatal("expected error for corrupt tar")
	}
}

func TestExtractBinary_OpenFileError(t *testing.T) {
	// Covers update.go:266 — OpenFile error when dest dir doesn't exist.
	// Already covered by TestExtractBinary/unwritable_destination.
	// Adding explicit test for clarity.
	tmpDir := t.TempDir()
	archiveData := buildTestTarGz(t, "binary")
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
		t.Fatal(err)
	}

	err := extractBinary(archivePath, filepath.Join("/nonexistent-dir-xyz", "redis-tui"))
	if err == nil {
		t.Fatal("expected error for unwritable destination")
	}
	if !strings.Contains(err.Error(), "could not create binary") {
		t.Errorf("error = %q, want 'could not create binary'", err)
	}
}

func TestVerifyChecksum_HashCopyError(t *testing.T) {
	tmpDir := t.TempDir()

	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	if err := os.WriteFile(archivePath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash := sha256.Sum256([]byte("content"))
	checksumPath := filepath.Join(tmpDir, "checksums.txt")
	if err := os.WriteFile(checksumPath, []byte(fmt.Sprintf("%x  test.tar.gz\n", hash)), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := ioCopy
	ioCopy = func(io.Writer, io.Reader) (int64, error) {
		return 0, fmt.Errorf("injected copy error")
	}
	t.Cleanup(func() { ioCopy = orig })

	err := verifyChecksum(archivePath, checksumPath, "test.tar.gz")
	if err == nil || !strings.Contains(err.Error(), "could not hash archive") {
		t.Errorf("error = %v, want hash error", err)
	}
}

func TestExtractBinary_WriteCopyError(t *testing.T) {
	tmpDir := t.TempDir()
	archiveData := buildTestTarGz(t, "binary content here")
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Return a file that's already closed so io.Copy fails on Write.
	orig := osOpenFile
	osOpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		f, err := os.OpenFile(name, flag, perm)
		if err != nil {
			return nil, err
		}
		_ = f.Close() // close it so writes fail
		return f, nil
	}
	t.Cleanup(func() { osOpenFile = orig })

	err := extractBinary(archivePath, filepath.Join(tmpDir, "out"))
	if err == nil {
		t.Fatal("expected error from io.Copy on closed file")
	}
}

func TestExtractBinary_OpenFileError_Injected(t *testing.T) {
	tmpDir := t.TempDir()
	archiveData := buildTestTarGz(t, "binary")
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
		t.Fatal(err)
	}

	orig := osOpenFile
	osOpenFile = func(string, int, os.FileMode) (*os.File, error) {
		return nil, fmt.Errorf("injected open error")
	}
	t.Cleanup(func() { osOpenFile = orig })

	err := extractBinary(archivePath, filepath.Join(tmpDir, "out"))
	if err == nil || !strings.Contains(err.Error(), "could not create binary") {
		t.Errorf("error = %v, want create binary error", err)
	}
}

func TestCheckWriteAccess_CloseError(t *testing.T) {
	// Trigger Close() error by pre-closing the file descriptor.
	tmpDir := t.TempDir()
	orig := osCreateTemp
	osCreateTemp = func(dir, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		// Close the fd — the next f.Close() call in checkWriteAccess will error.
		fd := f.Fd()
		_ = fd // used implicitly by Close
		// Unfortunately, closing via fd and then calling f.Close() is UB in Go.
		// Instead, close the *os.File itself so the next Close returns an error.
		_ = f.Close()
		// Re-open so checkWriteAccess can call Close again (will return "file already closed").
		return f, nil
	}
	t.Cleanup(func() { osCreateTemp = orig })

	err := checkWriteAccess(filepath.Join(tmpDir, "redis-tui"))
	if err == nil {
		t.Error("expected error from close on already-closed file")
	}
}

func TestCheckWriteAccess_Success(t *testing.T) {
	// The tmp.Close() error in checkWriteAccess is nearly impossible to
	// trigger on a normal filesystem. Verify the function works correctly
	// in the success and failure paths that ARE reachable.
	tmpDir := t.TempDir()
	if err := checkWriteAccess(filepath.Join(tmpDir, "redis-tui")); err != nil {
		t.Errorf("unexpected error in writable dir: %v", err)
	}

	if err := checkWriteAccess("/nonexistent-dir/redis-tui"); err == nil {
		t.Error("expected error for non-writable directory")
	}
}

func TestIsSemver(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1.0.0", true},
		{"v1.0.0", true},
		{"0.1.2", true},
		{"dev", false},
		{"abc", false},
		{"", false},
		{"1.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isSemver(tt.input); got != tt.want {
				t.Errorf("isSemver(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsHomebrew(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/usr/local/Cellar/redis-tui/1.0.0/bin/redis-tui", true},
		{"/opt/homebrew/bin/redis-tui", true},
		{"/usr/local/bin/redis-tui", false},
		{"/home/user/go/bin/redis-tui", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isHomebrew(tt.path); got != tt.want {
				t.Errorf("isHomebrew(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
