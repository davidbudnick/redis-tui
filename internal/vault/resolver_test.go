package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
)

func TestIsKVV2Metadata(t *testing.T) {
	valid := map[string]any{
		"created_time":  "2026-08-21T19:00:00.123456789Z",
		"deletion_time": "",
		"destroyed":     false,
		"version":       json.Number("1"),
	}
	tests := []struct {
		name     string
		metadata map[string]any
		want     bool
	}{
		{name: "valid", metadata: valid, want: true},
		{name: "valid deletion time", metadata: map[string]any{"created_time": "2026-08-21T19:00:00Z", "deletion_time": "2026-08-22T19:00:00Z", "destroyed": true, "version": json.Number("2")}, want: true},
		{name: "missing created time", metadata: map[string]any{"deletion_time": "", "destroyed": false, "version": json.Number("1")}},
		{name: "invalid created time", metadata: map[string]any{"created_time": "yesterday", "deletion_time": "", "destroyed": false, "version": json.Number("1")}},
		{name: "missing deletion time", metadata: map[string]any{"created_time": "2026-08-21T19:00:00Z", "destroyed": false, "version": json.Number("1")}},
		{name: "invalid deletion time", metadata: map[string]any{"created_time": "2026-08-21T19:00:00Z", "deletion_time": "tomorrow", "destroyed": false, "version": json.Number("1")}},
		{name: "missing destroyed", metadata: map[string]any{"created_time": "2026-08-21T19:00:00Z", "deletion_time": "", "version": json.Number("1")}},
		{name: "invalid version type", metadata: map[string]any{"created_time": "2026-08-21T19:00:00Z", "deletion_time": "", "destroyed": false, "version": 1}},
		{name: "non-integer version", metadata: map[string]any{"created_time": "2026-08-21T19:00:00Z", "deletion_time": "", "destroyed": false, "version": json.Number("1.5")}},
		{name: "zero version", metadata: map[string]any{"created_time": "2026-08-21T19:00:00Z", "deletion_time": "", "destroyed": false, "version": json.Number("0")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isKVV2Metadata(tt.metadata); got != tt.want {
				t.Errorf("isKVV2Metadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolverResolve(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		status      int
		path        string
		key         string
		want        string
		wantErr     string
		wantAPIPath string
	}{
		{name: "KV v2", response: `{"data":{"data":{"password":"vault-secret"},"metadata":{"created_time":"2026-08-21T19:00:00.123456789Z","deletion_time":"","destroyed":false,"version":1}}}`, status: http.StatusOK, path: "/secret/data/redis", key: "password", want: "vault-secret", wantAPIPath: "/v1/secret/data/redis"},
		{name: "KV v1", response: `{"data":{"password":"vault-secret"}}`, status: http.StatusOK, path: "secret/redis", key: "password", want: "vault-secret", wantAPIPath: "/v1/secret/redis"},
		{name: "KV v1 data field", response: `{"data":{"password":"vault-secret","data":{"other":"value"}}}`, status: http.StatusOK, path: "secret/redis", key: "password", want: "vault-secret", wantAPIPath: "/v1/secret/redis"},
		{name: "KV v1 data and metadata fields", response: `{"data":{"password":"vault-secret","data":{"password":"wrong-secret"},"metadata":{"version":1}}}`, status: http.StatusOK, path: "secret/redis", key: "password", want: "vault-secret", wantAPIPath: "/v1/secret/redis"},
		{name: "nested key", response: `{"data":{"credentials":{"VALKEY":{"password":"vault-secret"}}}}`, status: http.StatusOK, path: "secret/redis", key: "credentials.VALKEY.password", want: "vault-secret", wantAPIPath: "/v1/secret/redis"},
		{name: "missing secret", status: http.StatusNotFound, path: "secret/missing", key: "password", wantErr: "was not found", wantAPIPath: "/v1/secret/missing"},
		{name: "missing key", response: `{"data":{"username":"redis"}}`, status: http.StatusOK, path: "secret/redis", key: "password", wantErr: `does not contain key "password"`, wantAPIPath: "/v1/secret/redis"},
		{name: "nested key through scalar", response: `{"data":{"password":"vault-secret"}}`, status: http.StatusOK, path: "secret/redis", key: "password.value", wantErr: `does not contain key "password.value"`, wantAPIPath: "/v1/secret/redis"},
		{name: "non-string key", response: `{"data":{"password":123}}`, status: http.StatusOK, path: "secret/redis", key: "password", wantErr: "is not a string", wantAPIPath: "/v1/secret/redis"},
		{name: "Vault error", response: `{"errors":["denied"]}`, status: http.StatusForbidden, path: "secret/redis", key: "password", wantErr: "read Vault secret", wantAPIPath: "/v1/secret/redis"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.wantAPIPath {
					t.Errorf("request path = %q, want %q", r.URL.Path, tt.wantAPIPath)
				}
				if r.Header.Get("X-Vault-Token") != "test-token" {
					t.Errorf("X-Vault-Token = %q", r.Header.Get("X-Vault-Token"))
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				if _, err := fmt.Fprint(w, tt.response); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			t.Cleanup(server.Close)
			t.Setenv("VAULT_ADDR", server.URL)
			t.Setenv("VAULT_TOKEN", "test-token")
			t.Setenv("VAULT_MAX_RETRIES", "0")

			got, err := NewResolver().Resolve(context.Background(), tt.path, tt.key)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Resolve() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if got[tt.key] != tt.want {
				t.Errorf("Resolve() = %q, want %q", got[tt.key], tt.want)
			}
		})
	}
}

func TestResolverClientConfigurationError(t *testing.T) {
	t.Setenv("VAULT_MAX_RETRIES", "invalid")
	_, err := NewResolver().Resolve(context.Background(), "secret/redis", "password")
	if err == nil || !strings.Contains(err.Error(), "create Vault client") {
		t.Fatalf("Resolve() error = %v, want client configuration error", err)
	}
}

func TestResolverResolveMultipleNestedKeys(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprint(w, `{"data":{"credentials":{"VALKEY":{"username":"redis-user","password":"redis-password"}}}}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("VAULT_ADDR", server.URL)
	t.Setenv("VAULT_TOKEN", "test-token")
	t.Setenv("VAULT_MAX_RETRIES", "0")

	usernameKey := "credentials.VALKEY.username"
	passwordKey := "credentials.VALKEY.password"
	values, err := NewResolver().Resolve(context.Background(), "eng-ebs/example", usernameKey, passwordKey)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if values[usernameKey] != "redis-user" || values[passwordKey] != "redis-password" {
		t.Errorf("Resolve() = %#v", values)
	}
	if requests != 1 {
		t.Errorf("Vault request count = %d, want 1", requests)
	}
}

func TestNewVaultClientTokenHelper(t *testing.T) {
	t.Run("internal token helper", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("VAULT_TOKEN", "")
		t.Setenv("VAULT_CONFIG_PATH", filepath.Join(home, "missing-config"))
		homedir.Reset()
		t.Cleanup(homedir.Reset)
		if err := os.WriteFile(filepath.Join(home, ".vault-token"), []byte("helper-token\n"), 0o600); err != nil {
			t.Fatalf("write token file: %v", err)
		}

		client, err := newVaultClient()
		if err != nil {
			t.Fatalf("newVaultClient() error = %v", err)
		}
		if client.Token() != "helper-token" {
			t.Errorf("Token() = %q, want helper token", client.Token())
		}
	})

	t.Run("token helper configuration error", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "vault-config")
		if err := os.WriteFile(configPath, []byte("invalid = true\n"), 0o600); err != nil {
			t.Fatalf("write Vault config: %v", err)
		}
		t.Setenv("VAULT_TOKEN", "")
		t.Setenv("VAULT_CONFIG_PATH", configPath)

		if _, err := newVaultClient(); err == nil || !strings.Contains(err.Error(), "load Vault token helper") {
			t.Fatalf("newVaultClient() error = %v", err)
		}
	})

	t.Run("token helper read error", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "vault-config")
		config := "token_helper = " + strconv.Quote(dir) + "\n"
		if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
			t.Fatalf("write Vault config: %v", err)
		}
		t.Setenv("VAULT_TOKEN", "")
		t.Setenv("VAULT_CONFIG_PATH", configPath)

		if _, err := newVaultClient(); err == nil || !strings.Contains(err.Error(), "read Vault token helper") {
			t.Fatalf("newVaultClient() error = %v", err)
		}
	})
}
