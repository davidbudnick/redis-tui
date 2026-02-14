package types

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"
)

// Connection stores Redis connection details
type Connection struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Host      string     `json:"host"`
	Port      int        `json:"port"`
	Password  string     `json:"password,omitempty"` // #nosec G117 -- stored in local user config
	DB        int        `json:"db"`
	Group     string     `json:"group,omitempty"`
	Color     string     `json:"color,omitempty"`
	UseSSH    bool       `json:"use_ssh,omitempty"`
	SSHConfig *SSHConfig `json:"ssh_config,omitempty"`
	UseTLS     bool       `json:"use_tls,omitempty"`
	TLSConfig  *TLSConfig `json:"tls_config,omitempty"`
	UseCluster bool       `json:"use_cluster,omitempty"`
	Created   time.Time  `json:"created_at"`
	Updated   time.Time  `json:"updated_at"`
}

// SSHConfig stores SSH tunnel configuration
type SSHConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Password       string `json:"password,omitempty"` // #nosec G117 -- stored in local user config
	PrivateKeyPath string `json:"private_key_path,omitempty"`
	Passphrase     string `json:"passphrase,omitempty"`
}

// TLSConfig stores TLS/SSL configuration
type TLSConfig struct {
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	CAFile             string `json:"ca_file,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	ServerName         string `json:"server_name,omitempty"`
}

// BuildTLSConfig creates a *tls.Config from the stored TLS parameters.
func (t *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	cfg := &tls.Config{
		InsecureSkipVerify: t.InsecureSkipVerify, // #nosec G402 -- user-configured
		ServerName:         t.ServerName,
	}

	if t.CertFile != "" && t.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	if t.CAFile != "" {
		caCert, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}

// ConnectionGroup organizes connections
type ConnectionGroup struct {
	Name        string  `json:"name"`
	Color       string  `json:"color,omitempty"`
	Connections []int64 `json:"connections"`
	Collapsed   bool    `json:"collapsed,omitempty"`
}

// Favorite stores a favorited key
type Favorite struct {
	ConnectionID int64     `json:"connection_id"`
	Connection   string    `json:"connection"` // Connection name for display
	Key          string    `json:"key"`
	Label        string    `json:"label,omitempty"`
	AddedAt      time.Time `json:"added_at"`
}

// RecentKey tracks recently accessed keys
type RecentKey struct {
	ConnectionID int64     `json:"connection_id"`
	Key          string    `json:"key"`
	Type         KeyType   `json:"type"`
	AccessedAt   time.Time `json:"accessed_at"`
}

// KeyTemplate is a template for creating new keys
type KeyTemplate struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	KeyPattern   string            `json:"key_pattern"`
	Pattern      string            `json:"pattern"` // Alias for KeyPattern
	Type         KeyType           `json:"type"`
	KeyType      KeyType           `json:"key_type"` // Alias for Type
	DefaultTTL   time.Duration     `json:"default_ttl,omitempty"`
	DefaultValue string            `json:"default_value,omitempty"`
	Fields       map[string]string `json:"fields,omitempty"` // For hash/stream
}
