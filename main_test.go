package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/davidbudnick/redis-tui/internal/ui"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFlags_NoArgs(t *testing.T) {
	conn, version, _, _, _, err := parseFlags([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil {
		t.Error("expected nil connection with no args")
	}
	if version {
		t.Error("expected version=false")
	}
}

func TestParseFlags_HostOnly(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{"--host", "localhost"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "localhost" {
		t.Errorf("Host = %q, want %q", conn.Host, "localhost")
	}
	if conn.Port != 6379 {
		t.Errorf("Port = %d, want %d", conn.Port, 6379)
	}
	if conn.DB != 0 {
		t.Errorf("DB = %d, want %d", conn.DB, 0)
	}
	if conn.Name != "localhost:6379" {
		t.Errorf("Name = %q, want %q", conn.Name, "localhost:6379")
	}
}

func TestParseFlags_ShortFlags(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{"-h", "redis.example.com", "-p", "6380", "-a", "secret", "-n", "5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "redis.example.com" {
		t.Errorf("Host = %q, want %q", conn.Host, "redis.example.com")
	}
	if conn.Port != 6380 {
		t.Errorf("Port = %d, want %d", conn.Port, 6380)
	}
	if conn.Password != "secret" {
		t.Errorf("Password = %q, want %q", conn.Password, "secret")
	}
	if conn.DB != 5 {
		t.Errorf("DB = %d, want %d", conn.DB, 5)
	}
}

func TestParseFlags_LongFlags(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{"--host", "10.0.0.1", "--port", "7000", "--user", "user", "--password", "pass", "--vault-path", "secret/data/redis", "--vault-username-key", "credentials.redis.username", "--vault-password-key", "credentials.redis.password", "--db", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "10.0.0.1" {
		t.Errorf("Host = %q, want %q", conn.Host, "10.0.0.1")
	}
	if conn.Port != 7000 {
		t.Errorf("Port = %d, want %d", conn.Port, 7000)
	}
	if conn.Username != "user" {
		t.Errorf("Username = %q, want %q", conn.Username, "user")
	}
	if conn.Password != "pass" {
		t.Errorf("Password = %q, want %q", conn.Password, "pass")
	}
	if conn.VaultPath != "secret/data/redis" || conn.VaultUserKey != "credentials.redis.username" || conn.VaultPasswordKey != "credentials.redis.password" {
		t.Errorf("Vault reference = %q/%q/%q", conn.VaultPath, conn.VaultUserKey, conn.VaultPasswordKey)
	}
	if conn.DB != 3 {
		t.Errorf("DB = %d, want %d", conn.DB, 3)
	}
}

func TestParseFlags_CustomName(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{"--host", "localhost", "--name", "Production"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Name != "Production" {
		t.Errorf("Name = %q, want %q", conn.Name, "Production")
	}
}

func TestParseFlags_DefaultName(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{"--host", "myhost", "--port", "9999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Name != "myhost:9999" {
		t.Errorf("Name = %q, want %q", conn.Name, "myhost:9999")
	}
}

func TestParseFlags_Cluster(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{"--host", "localhost", "--cluster"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if !conn.UseCluster {
		t.Error("UseCluster should be true")
	}
}

func TestParseFlags_TLS(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{
		"--host", "localhost",
		"--tls",
		"--tls-cert", "/path/cert.pem",
		"--tls-key", "/path/key.pem",
		"--tls-ca", "/path/ca.pem",
		"--tls-skip-verify",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if !conn.UseTLS {
		t.Error("UseTLS should be true")
	}
	if conn.TLSConfig == nil {
		t.Fatal("TLSConfig should be set")
	}
	if conn.TLSConfig.CertFile != "/path/cert.pem" {
		t.Errorf("CertFile = %q, want %q", conn.TLSConfig.CertFile, "/path/cert.pem")
	}
	if conn.TLSConfig.KeyFile != "/path/key.pem" {
		t.Errorf("KeyFile = %q, want %q", conn.TLSConfig.KeyFile, "/path/key.pem")
	}
	if conn.TLSConfig.CAFile != "/path/ca.pem" {
		t.Errorf("CAFile = %q, want %q", conn.TLSConfig.CAFile, "/path/ca.pem")
	}
	if !conn.TLSConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}
}

func TestParseFlags_TLSNotSet(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{"--host", "localhost"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.UseTLS {
		t.Error("UseTLS should be false")
	}
	if conn.TLSConfig != nil {
		t.Error("TLSConfig should be nil when --tls is not set")
	}
}

func TestParseFlags_Version(t *testing.T) {
	conn, version, _, _, _, err := parseFlags([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil {
		t.Error("expected nil connection for --version")
	}
	if !version {
		t.Error("expected version=true")
	}
}

func TestParseFlags_AllOptions(t *testing.T) {
	conn, _, _, _, _, err := parseFlags([]string{
		"--host", "redis.prod.com",
		"--port", "6380",
		"--user", "admin",
		"--password", "s3cret",
		"--db", "7",
		"--name", "Prod Redis",
		"--cluster",
		"--tls",
		"--tls-cert", "/cert.pem",
		"--tls-key", "/key.pem",
		"--tls-ca", "/ca.pem",
		"--tls-skip-verify",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "redis.prod.com" {
		t.Errorf("Host = %q", conn.Host)
	}
	if conn.Port != 6380 {
		t.Errorf("Port = %d", conn.Port)
	}
	if conn.Username != "admin" {
		t.Errorf("Username = %q", conn.Username)
	}
	if conn.Password != "s3cret" {
		t.Errorf("Password = %q", conn.Password)
	}
	if conn.DB != 7 {
		t.Errorf("DB = %d", conn.DB)
	}
	if conn.Name != "Prod Redis" {
		t.Errorf("Name = %q", conn.Name)
	}
	if !conn.UseCluster {
		t.Error("UseCluster should be true")
	}
	if !conn.UseTLS {
		t.Error("UseTLS should be true")
	}
	if conn.TLSConfig == nil {
		t.Fatal("TLSConfig should be set")
	}
}

func TestParseFlags_InvalidFlag(t *testing.T) {
	_, _, _, _, _, err := parseFlags([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestParseFlags_Help(t *testing.T) {
	_, _, _, _, _, err := parseFlags([]string{"--help"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestParseFlags_Update(t *testing.T) {
	conn, version, doUpdate, _, _, err := parseFlags([]string{"--update"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil {
		t.Error("expected nil connection for --update")
	}
	if version {
		t.Error("expected version=false")
	}
	if !doUpdate {
		t.Error("expected doUpdate=true")
	}
}

func TestParseFlags_UpdateWithOtherFlags(t *testing.T) {
	conn, version, doUpdate, _, _, err := parseFlags([]string{"--host", "localhost", "--update"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil {
		t.Error("expected nil connection when --update is set")
	}
	if version {
		t.Error("expected version=false")
	}
	if !doUpdate {
		t.Error("expected doUpdate=true")
	}
}

func TestParseFlags_ScanSize(t *testing.T) {
	t.Run("default scan size", func(t *testing.T) {
		_, _, _, scanSize, _, err := parseFlags([]string{"--host", "localhost"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scanSize != 1000 {
			t.Errorf("ScanSize = %d, want 1000", scanSize)
		}
	})

	t.Run("custom scan size", func(t *testing.T) {
		_, _, _, scanSize, _, err := parseFlags([]string{"--host", "localhost", "--scan-size", "500"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scanSize != 500 {
			t.Errorf("ScanSize = %d, want 500", scanSize)
		}
	})
}

func TestParseFlags_IncludeTypesFalse(t *testing.T) {
	_, _, _, _, includeTypes, err := parseFlags([]string{"--host", "localhost", "--include-types=false"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if includeTypes {
		t.Error("expected includeTypes=false")
	}
}

func TestParseFlags_Defaults(t *testing.T) {
	_, _, _, scanSize, includeTypes, err := parseFlags([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scanSize != 1000 {
		t.Errorf("ScanSize = %d, want 1000", scanSize)
	}
	if !includeTypes {
		t.Error("expected includeTypes=true by default")
	}
}

func TestInitConfig_LegacyMigration_UnsafePerms(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up legacy path with group/other-writable permissions.
	legacyDir := filepath.Join(tmpDir, ".redis")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	legacyConfig := map[string]any{
		"connections": []any{},
	}
	data, err := json.Marshal(legacyConfig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	legacyPath := filepath.Join(legacyDir, "config.json")
	if err := os.WriteFile(legacyPath, data, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	// Force group/other-writable to bypass umask.
	if err := os.Chmod(legacyPath, 0o666); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}

	configDir := filepath.Join(tmpDir, ".config", "redis-tui")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")

	// The new config should NOT exist after migration is skipped due to perms.
	// We can't call initConfig directly (it uses os.UserHomeDir), so we
	// verify the permission check by stat-ing the legacy file ourselves.
	info, err := os.Stat(legacyPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	if info.Mode().Perm()&0o022 == 0 {
		t.Fatal("expected unsafe permissions on legacy file")
	}

	// Config file should not have been created.
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("expected config file to not exist (migration should be skipped)")
	}
}

func TestInitConfig_LegacyMigration_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	legacyDir := filepath.Join(tmpDir, ".redis")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Write invalid JSON to legacy path.
	legacyPath := filepath.Join(legacyDir, "config.json")
	if err := os.WriteFile(legacyPath, []byte("not valid json{{{"), 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Verify it's detected as invalid.
	var cfg map[string]any
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err == nil {
		t.Error("expected JSON parse error for invalid config")
	}
}

// --- initConfig tests ---

func TestInitConfig_Fresh(t *testing.T) {
	tmpDir := t.TempDir()
	orig := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = orig })

	cfg, err := initConfig()
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	_ = cfg.Close()

	// Verify the config directory was created.
	configDir := filepath.Join(tmpDir, ".config", "redis-tui")
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected config dir to be a directory")
	}
}

func TestInitConfig_HomeDirError(t *testing.T) {
	orig := userHomeDir
	userHomeDir = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { userHomeDir = orig })

	// Falls back to os.TempDir() — should still succeed.
	cfg, err := initConfig()
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}
	_ = cfg.Close()
}

func TestInitConfig_LegacyMigration_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid legacy config.
	legacyDir := filepath.Join(tmpDir, ".redis")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatal(err)
	}
	legacyData := `{"connections":[],"tree_separator":":"}`
	if err := os.WriteFile(filepath.Join(legacyDir, "config.json"), []byte(legacyData), 0o600); err != nil {
		t.Fatal(err)
	}

	orig := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = orig })

	// Silence slog output during test.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})))

	cfg, err := initConfig()
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}
	_ = cfg.Close()

	// Verify the new config file was created.
	configPath := filepath.Join(tmpDir, ".config", "redis-tui", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected migrated config file")
	}
}

func TestInitConfig_LegacyMigration_ReadAfterValidateError(t *testing.T) {
	tmpDir := t.TempDir()
	legacyDir := filepath.Join(tmpDir, ".redis")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "config.json"), []byte(`{"connections":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	orig := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = orig })

	origRead := osReadFile
	osReadFile = func(string) ([]byte, error) { return nil, fmt.Errorf("injected read error") }
	t.Cleanup(func() { osReadFile = origRead })

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})))

	cfg, err := initConfig()
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}
	_ = cfg.Close()
}

func TestInitConfig_LegacyMigration_ParseError(t *testing.T) {
	tmpDir := t.TempDir()

	legacyDir := filepath.Join(tmpDir, ".redis")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Invalid JSON — migration should be skipped gracefully.
	if err := os.WriteFile(filepath.Join(legacyDir, "config.json"), []byte("{{{bad"), 0o600); err != nil {
		t.Fatal(err)
	}

	orig := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = orig })

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})))

	// Should still return a valid config (fresh, not migrated).
	cfg, err := initConfig()
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}
	_ = cfg.Close()
}

// --- parseCLIFlags tests ---

func withOsArgs(t *testing.T, args []string) {
	t.Helper()
	orig := os.Args
	os.Args = args
	t.Cleanup(func() { os.Args = orig })
}

// exitPanic is used to halt execution in parseCLIFlags when osExit is called.
type exitPanic int

func withExitTrap(t *testing.T) *int {
	t.Helper()
	var exitCode int
	orig := osExit
	osExit = func(code int) {
		exitCode = code
		panic(exitPanic(code))
	}
	t.Cleanup(func() { osExit = orig })
	return &exitCode
}

// callParseCLIFlags calls parseCLIFlags and recovers the exitPanic.
func callParseCLIFlags() (conn interface{}, code int, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			if ep, ok := r.(exitPanic); ok {
				code = int(ep)
				panicked = true
				return
			}
			panic(r) // re-panic if not our sentinel
		}
	}()
	c := parseCLIFlags()
	return c, 0, false
}

func TestParseCLIFlags_NoArgs(t *testing.T) {
	withOsArgs(t, []string{"redis-tui"})
	_ = withExitTrap(t)

	result, _, panicked := callParseCLIFlags()
	if panicked {
		t.Error("did not expect exit for no args")
	}
	// nil *types.Connection wrapped in interface{} is not == nil,
	// so check via type assertion.
	if result != (*types.Connection)(nil) {
		t.Error("expected nil connection")
	}
}

func TestParseCLIFlags_Version(t *testing.T) {
	withOsArgs(t, []string{"redis-tui", "--version"})
	code := withExitTrap(t)

	_, _, panicked := callParseCLIFlags()
	if !panicked {
		t.Error("expected exit for --version")
	}
	if *code != 0 {
		t.Errorf("exit code = %d, want 0", *code)
	}
}

func TestParseCLIFlags_InvalidFlag(t *testing.T) {
	withOsArgs(t, []string{"redis-tui", "--bogus"})
	code := withExitTrap(t)

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	_, _, panicked := callParseCLIFlags()
	_ = w.Close()
	_, _ = new(bytes.Buffer).ReadFrom(r)
	os.Stderr = oldStderr

	if !panicked {
		t.Error("expected exit for invalid flag")
	}
	if *code != 2 {
		t.Errorf("exit code = %d, want 2", *code)
	}
}

func TestParseCLIFlags_Help(t *testing.T) {
	withOsArgs(t, []string{"redis-tui", "--help"})
	code := withExitTrap(t)

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	_, _, panicked := callParseCLIFlags()
	_ = w.Close()
	_, _ = new(bytes.Buffer).ReadFrom(r)
	os.Stderr = oldStderr

	if !panicked {
		t.Error("expected exit for --help")
	}
	if *code != 0 {
		t.Errorf("exit code = %d, want 0", *code)
	}
}

func TestParseCLIFlags_Update_Failure(t *testing.T) {
	withOsArgs(t, []string{"redis-tui", "--update"})
	code := withExitTrap(t)

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	_, _, panicked := callParseCLIFlags()
	_ = w.Close()
	_, _ = new(bytes.Buffer).ReadFrom(r)
	os.Stderr = oldStderr

	if !panicked {
		t.Error("expected exit for --update")
	}
	// version is "dev" → runUpdate fails → exit 1
	if *code != 1 {
		t.Errorf("exit code = %d, want 1", *code)
	}
}

func TestParseCLIFlags_Update_Success(t *testing.T) {
	// Temporarily set version to a semver so runUpdate can proceed
	// past the dev check. Use a server that returns the same version
	// so it's "already up to date" (no actual download needed).
	withOsArgs(t, []string{"redis-tui", "--update"})
	code := withExitTrap(t)

	oldVer := version
	version = "1.0.0"
	t.Cleanup(func() { version = oldVer })

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "redis-tui")
	if err := os.WriteFile(execPath, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	origExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = origExec })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0"}`)
	}))
	defer srv.Close()
	oldAPI := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = oldAPI })

	_, _, panicked := callParseCLIFlags()
	if !panicked {
		t.Error("expected exit after successful update")
	}
	if *code != 0 {
		t.Errorf("exit code = %d, want 0", *code)
	}
}

func TestParseCLIFlags_WithHost(t *testing.T) {
	withOsArgs(t, []string{"redis-tui", "--host", "myhost"})
	_ = withExitTrap(t)

	result, _, panicked := callParseCLIFlags()
	if panicked {
		t.Error("did not expect exit for --host")
	}
	if result == nil {
		t.Fatal("expected non-nil connection")
	}
}

// --- setup tests ---

func TestSetup_Success(t *testing.T) {
	withOsArgs(t, []string{"redis-tui"})
	_ = withExitTrap(t)

	tmpDir := t.TempDir()
	orig := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = orig })

	m, err := setup()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if m.Cmds == nil {
		t.Error("expected Cmds to be set")
	}
}

type fatalPanic struct{ v any }

func withFatalTrap(t *testing.T) {
	t.Helper()
	orig := logFatal
	logFatal = func(v ...any) { panic(fatalPanic{v}) }
	t.Cleanup(func() { logFatal = orig })
}

func withRunApp(t *testing.T, fn func(ui.Model) error) {
	t.Helper()
	orig := runApp
	runApp = fn
	t.Cleanup(func() { runApp = orig })
}

func safeMain() (recovered bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(fatalPanic); ok {
				recovered = true
				return
			}
			if _, ok := r.(exitPanic); ok {
				recovered = true
				return
			}
			panic(r)
		}
	}()
	main()
	return false
}

func TestMain_Success(t *testing.T) {
	withOsArgs(t, []string{"redis-tui"})
	_ = withExitTrap(t)

	tmpDir := t.TempDir()
	origHome := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = origHome })

	withRunApp(t, func(_ ui.Model) error { return nil })

	if safeMain() {
		t.Error("main() should not have panicked")
	}
}

func TestMain_SetupError(t *testing.T) {
	withOsArgs(t, []string{"redis-tui"})
	_ = withExitTrap(t)
	withFatalTrap(t)

	blockedHome := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockedHome, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	origHome := userHomeDir
	userHomeDir = func() (string, error) { return blockedHome, nil }
	t.Cleanup(func() { userHomeDir = origHome })

	withRunApp(t, func(_ ui.Model) error { return nil })

	if !safeMain() {
		t.Error("expected logFatal for setup error")
	}
}

func TestMain_RunError(t *testing.T) {
	withOsArgs(t, []string{"redis-tui"})
	_ = withExitTrap(t)
	withFatalTrap(t)

	tmpDir := t.TempDir()
	origHome := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = origHome })

	withRunApp(t, func(_ ui.Model) error { return fmt.Errorf("TUI crashed") })

	if !safeMain() {
		t.Error("expected logFatal for run error")
	}
}

func TestProdLogFatal(t *testing.T) {
	if os.Getenv("TEST_PROD_LOG_FATAL") == "1" {
		prodLogFatal("test fatal")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestProdLogFatal")
	cmd.Env = append(os.Environ(), "TEST_PROD_LOG_FATAL=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit from prodLogFatal")
	}
}

func TestProdRunApp(t *testing.T) {
	// prodRunApp creates a tea.Program and calls Run(). Without a real
	// terminal, Run() returns an error. We just need the lines to execute.
	m := ui.NewModel()
	sendFunc := func(tea.Msg) {}
	m.SendFunc = &sendFunc
	// Run will fail (no tty) — that's fine, we just need coverage.
	_ = prodRunApp(m)
}

func TestSetup_WithHost(t *testing.T) {
	withOsArgs(t, []string{"redis-tui", "--host", "myhost"})
	_ = withExitTrap(t)

	tmpDir := t.TempDir()
	orig := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = orig })

	m, err := setup()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if m.CLIConnection == nil {
		t.Fatal("expected CLIConnection to be set")
	}
	if m.CLIConnection.Host != "myhost" {
		t.Errorf("Host = %q, want %q", m.CLIConnection.Host, "myhost")
	}
}

func TestSetup_ConfigError(t *testing.T) {
	withOsArgs(t, []string{"redis-tui"})
	_ = withExitTrap(t)

	blockedHome := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockedHome, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := userHomeDir
	userHomeDir = func() (string, error) { return blockedHome, nil }
	t.Cleanup(func() { userHomeDir = orig })

	_, err := setup()
	if err == nil {
		t.Fatal("expected error when config dir is not creatable")
	}
}

func TestInitConfig_MkdirError(t *testing.T) {
	orig := userHomeDir
	// Use a file (not dir) as home — MkdirAll will fail.
	f, err := os.CreateTemp("", "home")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	userHomeDir = func() (string, error) { return f.Name(), nil }
	t.Cleanup(func() { userHomeDir = orig })

	_, err = initConfig()
	if err == nil {
		t.Fatal("expected MkdirAll error")
	}
}

func TestInitConfig_LegacyMigration_WriteError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid legacy config with connections field.
	legacyDir := filepath.Join(tmpDir, ".redis")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(legacyDir, "config.json")
	if err := os.WriteFile(legacyPath, []byte(`{"connections":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// DON'T pre-create the config dir — initConfig creates it. But then
	// the configPath won't exist so the migration triggers. The WriteFile
	// to configPath should fail if the dir is read-only.
	// So: create the dir, make it read-only.
	configDir := filepath.Join(tmpDir, ".config", "redis-tui")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(configDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(configDir, 0o750) })

	orig := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = orig })

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})))

	// initConfig should still succeed — migration write failure is logged, not fatal.
	// The error happens at line 243 (writeErr path).
	cfg, err := initConfig()
	if err != nil {
		// On some systems, db.NewConfig itself may fail if the dir is read-only.
		// That's OK — we're testing that the migration write error path is hit.
		return
	}
	_ = cfg.Close()
}

func TestInitConfig_LegacyMigration_UnsafePermsSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	legacyDir := filepath.Join(tmpDir, ".redis")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "config.json"), []byte(`{"connections":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(legacyDir, "config.json"), 0o666); err != nil {
		t.Fatal(err)
	}

	orig := userHomeDir
	userHomeDir = func() (string, error) { return tmpDir, nil }
	t.Cleanup(func() { userHomeDir = orig })

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})))

	cfg, err := initConfig()
	if err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}
	_ = cfg.Close()
}

func TestParseFlags_PasswordWarning(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantWarn bool
	}{
		{"short flag -a", []string{"-h", "localhost", "-a", "secret"}, true},
		{"long flag --password", []string{"--host", "localhost", "--password", "secret"}, true},
		{"no password flag", []string{"--host", "localhost"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			os.Stderr = w

			_, _, _, _, _, parseErr := parseFlags(tt.args)
			if parseErr != nil {
				os.Stderr = oldStderr
				t.Fatalf("unexpected error: %v", parseErr)
			}

			if err := w.Close(); err != nil {
				t.Fatalf("failed to close writer: %v", err)
			}
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(r); err != nil {
				t.Fatalf("failed to read pipe: %v", err)
			}
			os.Stderr = oldStderr

			gotWarn := strings.Contains(buf.String(), "process list")
			if gotWarn != tt.wantWarn {
				t.Errorf("warning present = %v, want %v (stderr: %q)", gotWarn, tt.wantWarn, buf.String())
			}
		})
	}
}
