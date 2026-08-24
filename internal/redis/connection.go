package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"
)

const (
	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
	defaultPoolSize     = 10
	defaultMinIdleConns = 3
	defaultMaxRetries   = 3
	defaultPingTimeout  = 5 * time.Second
)

func defaultOptions(conn types.Connection) (*redis.Options, error) {
	if conn.Host == "" {
		return nil, errors.New("host is required")
	}
	if conn.Port == 0 {
		return nil, errors.New("port is required")
	}

	return &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", conn.Host, conn.Port),
		Username:     conn.Username,
		Password:     conn.Password,
		DB:           conn.DB,
		DialTimeout:  defaultDialTimeout,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		PoolSize:     defaultPoolSize,
		MinIdleConns: defaultMinIdleConns,
		MaxRetries:   defaultMaxRetries,
	}, nil
}

func (c *Client) resolveCredentials(conn types.Connection) (types.Connection, error) {
	if conn.VaultPath == "" && conn.VaultUserKey == "" && conn.VaultPasswordKey == "" {
		return conn, nil
	}
	if conn.VaultPath == "" {
		return types.Connection{}, errors.New("Vault path is required when using Vault credential keys")
	}
	if conn.VaultUserKey == "" && conn.VaultPasswordKey == "" {
		return types.Connection{}, errors.New("at least one Vault username or password key is required")
	}

	selectors := make([]string, 0, 2)
	if conn.VaultUserKey != "" {
		selectors = append(selectors, conn.VaultUserKey)
	}
	if conn.VaultPasswordKey != "" {
		selectors = append(selectors, conn.VaultPasswordKey)
	}
	values, err := c.credentials.Resolve(c.ctx, conn.VaultPath, selectors...)
	if err != nil {
		return types.Connection{}, fmt.Errorf("resolve Redis credentials from Vault: %w", err)
	}
	if conn.VaultUserKey != "" {
		conn.Username = values[conn.VaultUserKey]
	}
	if conn.VaultPasswordKey != "" {
		conn.Password = values[conn.VaultPasswordKey]
	}
	return conn, nil
}

// cleanup closes existing connections before establishing a new one
func (c *Client) cleanup() {
	c.mu.Lock()
	_ = c.disconnectLocked()
	c.mu.Unlock()
}

// Connect establishes a connection to Redis
func (c *Client) Connect(conn types.Connection) error {
	c.cleanup()
	var err error
	conn, err = c.resolveCredentials(conn)
	if err != nil {
		return err
	}

	opts, optErr := defaultOptions(conn)
	if optErr != nil {
		return optErr
	}

	if conn.UseTLS {
		if conn.TLSConfig == nil {
			return fmt.Errorf("TLS requested but TLS configuration is missing")
		}
		tlsCfg, err := conn.TLSConfig.BuildTLSConfig()
		if err != nil {
			return err
		}
		opts.TLSConfig = tlsCfg
	}
	client := redis.NewClient(opts)

	c.mu.Lock()
	c.host = conn.Host
	c.port = conn.Port
	c.username = conn.Username
	c.password = conn.Password
	c.db = conn.DB
	c.client = client
	ctx := c.ctx
	c.mu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	_, err = client.Ping(pingCtx).Result()
	return err
}

// ConnectCluster establishes a connection to a Redis cluster
func (c *Client) ConnectCluster(addrs []string, conn types.Connection) error {
	c.cleanup()
	var err error
	conn, err = c.resolveCredentials(conn)
	if err != nil {
		return err
	}

	// Parse first address for display purposes and for the Dialer below.
	seedHost := "127.0.0.1"
	host := seedHost
	port := 6379
	if len(addrs) > 0 {
		host, port = parseAddr(addrs[0])
		seedHost = host
	}

	opts := &redis.ClusterOptions{
		Addrs:        addrs,
		Username:     conn.Username,
		Password:     conn.Password,
		DialTimeout:  defaultDialTimeout,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		PoolSize:     defaultPoolSize,
		MinIdleConns: defaultMinIdleConns,
		MaxRetries:   defaultMaxRetries,
		// Remap cluster node addresses to the seed host. Cluster nodes
		// (especially in Docker) advertise internal IPs that may not be
		// reachable from the client. Keep the port from each node but
		// route through the original host.
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, p, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			return net.DialTimeout(network, net.JoinHostPort(seedHost, p), defaultDialTimeout)
		},
	}

	if conn.UseTLS {
		if conn.TLSConfig == nil {
			return fmt.Errorf("TLS requested but TLS configuration is missing")
		}
		tlsCfg, err := conn.TLSConfig.BuildTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to build TLS config: %w", err)
		}
		opts.TLSConfig = tlsCfg
	}

	cluster := redis.NewClusterClient(opts)

	c.mu.Lock()
	c.isCluster = true
	c.username = conn.Username
	c.password = conn.Password
	c.host = host
	c.port = port
	c.cluster = cluster
	ctx := c.ctx
	c.mu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	_, err = cluster.Ping(pingCtx).Result()
	return err
}

func parseAddr(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		// No port separator or invalid format — treat whole string as host.
		return strings.TrimSpace(addr), 6379
	}
	p, err := strconv.Atoi(portStr)
	if err != nil {
		return host, 6379
	}
	return host, p
}

// Disconnect closes the Redis connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.disconnectLocked()
}

func (c *Client) disconnectLocked() error {
	var errs []error
	if c.cancelKeyspace != nil {
		c.cancelKeyspace()
		c.cancelKeyspace = nil
	}
	if c.keyspacePS != nil {
		errs = append(errs, c.keyspacePS.Close())
		c.keyspacePS = nil
	}
	if c.pubsub != nil {
		errs = append(errs, c.pubsub.Close())
		c.pubsub = nil
	}
	if c.cluster != nil {
		errs = append(errs, c.cluster.Close())
		c.cluster = nil
	}
	if c.client != nil {
		errs = append(errs, c.client.Close())
		c.client = nil
	}
	c.isCluster = false
	c.eventHandlers = nil
	return errors.Join(errs...)
}

// IsCluster returns whether connected to a cluster
func (c *Client) IsCluster() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isCluster
}

// SelectDB switches the database
func (c *Client) SelectDB(db int) error {
	c.mu.RLock()
	isCluster := c.isCluster
	client := c.client
	ctx := c.ctx
	c.mu.RUnlock()

	if isCluster {
		return fmt.Errorf("database selection not supported in cluster mode")
	}
	if client == nil {
		return fmt.Errorf("not connected")
	}
	if err := client.Do(ctx, "SELECT", db).Err(); err != nil {
		return err
	}
	c.mu.Lock()
	c.db = db
	c.mu.Unlock()
	return nil
}

// TestConnection tests a connection
func (c *Client) TestConnection(conn types.Connection) (time.Duration, error) {
	var err error
	conn, err = c.resolveCredentials(conn)
	if err != nil {
		return 0, err
	}
	opts, optErr := defaultOptions(conn)
	if optErr != nil {
		return 0, optErr
	}

	if conn.UseTLS {
		if conn.TLSConfig == nil {
			return 0, fmt.Errorf("TLS requested but TLS configuration is missing")
		}
		tlsCfg, err := conn.TLSConfig.BuildTLSConfig()
		if err != nil {
			return 0, err
		}
		opts.TLSConfig = tlsCfg
	}
	testClient := redis.NewClient(opts)
	defer testClient.Close()

	start := time.Now()
	ctx, cancel := context.WithTimeout(c.ctx, defaultPingTimeout)
	defer cancel()

	_, err = testClient.Ping(ctx).Result()
	return elapsedSince(start), err
}

// elapsedSince returns time since start, floored at 1ns for coarse Windows clocks.
func elapsedSince(start time.Time) time.Duration {
	return max(time.Since(start), time.Nanosecond)
}
