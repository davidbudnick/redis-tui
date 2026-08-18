package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/davidbudnick/redis-tui/internal/cmd"
	"github.com/davidbudnick/redis-tui/internal/db"
	"github.com/davidbudnick/redis-tui/internal/redis"
	"github.com/davidbudnick/redis-tui/internal/service"
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/davidbudnick/redis-tui/internal/ui"

	tea "charm.land/bubbletea/v2"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Overridable in tests.
var (
	osExit   = os.Exit
	logFatal = prodLogFatal
	runApp   = prodRunApp
)

func prodLogFatal(v ...any) { log.Fatal(v...) }

func prodRunApp(m ui.Model) error {
	p := tea.NewProgram(m)
	*m.SendFunc = p.Send
	_, err := p.Run()
	return err
}

func main() {
	m, err := setup()
	if err != nil {
		logFatal(err)
		return
	}
	if err := runApp(m); err != nil {
		logFatal(err)
	}
}

func setup() (ui.Model, error) {
	opts := parseCLIFlags()

	logWriter := types.NewLogWriter()

	m := ui.NewModel()
	m.Logs = logWriter

	if opts != nil {
		m.CLIConnection = opts
	}

	sendFunc := func(msg tea.Msg) {}
	m.SendFunc = &sendFunc

	handler := slog.NewJSONHandler(logWriter, nil)
	slog.SetDefault(slog.New(handler))

	config, err := initConfig()
	if err != nil {
		return m, fmt.Errorf("failed to initialize config: %w", err)
	}

	redisClient := redis.NewClient()
	redisClient.SetIncludeTypes(cmd.GetIncludeTypes())
	container := &service.Container{Config: config, Redis: redisClient}
	m.Cmds = cmd.NewCommandsFromContainer(container)
	m.ScanSize = cmd.GetScanSize()
	m.Version = version

	return m, nil
}

func parseCLIFlags() *types.Connection {
	conn, showVersion, doUpdate, scanSize, includeTypes, err := parseFlags(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			osExit(0)
		}
		osExit(2)
	}
	if showVersion {
		fmt.Printf("redis-tui %s (commit: %s, built: %s)\n", version, commit, date)
		osExit(0)
	}
	if doUpdate {
		if err := runUpdate(version); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			osExit(1)
		}
		osExit(0)
	}
	cmd.SetScanSize(scanSize)
	cmd.SetIncludeTypes(includeTypes)
	return conn
}

// parseFlags parses the given args into a Connection. Returns nil when no
// --host is provided (interactive mode). showVersion is true when --version
// was requested. Returns an error if flag parsing fails.
func parseFlags(args []string) (conn *types.Connection, showVersion bool, doUpdate bool, scanSize int64, includeTypes bool, err error) {
	fs := flag.NewFlagSet("redis-tui", flag.ContinueOnError)

	host := fs.String("host", "", "Redis server hostname (required for quick-connect mode)")
	port := fs.Int("port", 6379, "Redis server port")
	username := fs.String("user", "", "Redis username (for ACL-enabled servers)")
	password := fs.String("password", "", "Redis password")
	dbNum := fs.Int("db", 0, "Redis database number (0-15)")
	name := fs.String("name", "", "Connection display name")
	cluster := fs.Bool("cluster", false, "Enable cluster mode")
	tls := fs.Bool("tls", false, "Enable TLS/SSL")
	tlsCert := fs.String("tls-cert", "", "TLS client certificate file")
	tlsKey := fs.String("tls-key", "", "TLS client private key file")
	tlsCA := fs.String("tls-ca", "", "TLS CA certificate file")
	tlsSkipVerify := fs.Bool("tls-skip-verify", false, "Skip TLS certificate verification")
	version := fs.Bool("version", false, "Print version and exit")
	update := fs.Bool("update", false, "Update to the latest version")
	scanSizeFlag := fs.Int64("scan-size", 1000, "Redis SCAN COUNT hint (batch size for key scanning)")
	includeTypesFlag := fs.Bool("include-types", true, "Fetch key types during scan (set false to skip)")

	// Short aliases
	fs.StringVar(host, "h", "", "Redis server hostname (shorthand)")
	fs.IntVar(port, "p", 6379, "Redis server port (shorthand)")
	fs.StringVar(password, "a", "", "Redis password (shorthand)")
	fs.IntVar(dbNum, "n", 0, "Redis database number (shorthand)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: redis-tui [flags]\n\n")
		fmt.Fprintf(os.Stderr, "A terminal UI for Redis.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fmt.Fprintf(os.Stderr, "  -h, --host string       Redis server hostname (required for quick-connect)\n")
		fmt.Fprintf(os.Stderr, "  -p, --port int          Redis server port (default 6379)\n")
		fmt.Fprintf(os.Stderr, "  -a, --password string   Redis password\n")
		fmt.Fprintf(os.Stderr, "  -n, --db int            Redis database number, 0-15 (default 0)\n")
		fmt.Fprintf(os.Stderr, "      --user string       Redis username (for ACL-enabled servers)\n")
		fmt.Fprintf(os.Stderr, "      --name string       Connection display name\n")
		fmt.Fprintf(os.Stderr, "      --cluster           Enable cluster mode\n")
		fmt.Fprintf(os.Stderr, "      --tls               Enable TLS/SSL\n")
		fmt.Fprintf(os.Stderr, "      --tls-cert string   TLS client certificate file\n")
		fmt.Fprintf(os.Stderr, "      --tls-key string    TLS client private key file\n")
		fmt.Fprintf(os.Stderr, "      --tls-ca string     TLS CA certificate file\n")
		fmt.Fprintf(os.Stderr, "      --tls-skip-verify   Skip TLS certificate verification\n")
		fmt.Fprintf(os.Stderr, "      --scan-size int     Redis SCAN COUNT hint (default 1000)\n")
		fmt.Fprintf(os.Stderr, "      --include-types     Fetch key types during scan (default true)\n")
		fmt.Fprintf(os.Stderr, "      --version           Print version and exit\n")
		fmt.Fprintf(os.Stderr, "      --update            Update to the latest version\n")
	}

	if err := fs.Parse(args); err != nil {
		return nil, false, false, 0, false, err
	}

	if *version {
		return nil, true, false, *scanSizeFlag, *includeTypesFlag, nil
	}

	if *update {
		return nil, false, true, *scanSizeFlag, *includeTypesFlag, nil
	}

	// If no host flag provided, return nil (normal interactive mode)
	if *host == "" {
		return nil, false, false, *scanSizeFlag, *includeTypesFlag, nil
	}

	// Warn if password was passed on the command line — it is visible via ps(1).
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "password" || f.Name == "a" {
			fmt.Fprintln(os.Stderr, "Warning: Using a password via -a/--password exposes it in the process list. Consider using the interactive connection form instead.")
		}
	})

	conn = &types.Connection{
		Host:       *host,
		Port:       *port,
		Username:   *username,
		Password:   *password,
		DB:         *dbNum,
		UseCluster: *cluster,
	}

	if *name != "" {
		conn.Name = *name
	} else {
		conn.Name = fmt.Sprintf("%s:%d", *host, *port)
	}

	if *tls {
		conn.UseTLS = true
		conn.TLSConfig = &types.TLSConfig{
			CertFile:           *tlsCert,
			KeyFile:            *tlsKey,
			CAFile:             *tlsCA,
			InsecureSkipVerify: *tlsSkipVerify,
		}
	}

	return conn, false, false, *scanSizeFlag, *includeTypesFlag, nil
}

// Overridable in tests.
var (
	userHomeDir = os.UserHomeDir
	osReadFile  = os.ReadFile
)

func initConfig() (*db.Config, error) {
	homeDir, err := userHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}

	configDir := filepath.Join(homeDir, ".config", "redis-tui")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.json")

	// Migrate from legacy config path (~/.redis/config.json) if new config doesn't exist.
	// Parse through NewConfig+save() instead of raw byte copy so the JSON is
	// validated and passwords are stripped before writing to the new location.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		legacyPath := filepath.Join(homeDir, ".redis", "config.json")
		if info, legacyStatErr := os.Stat(legacyPath); legacyStatErr == nil {
			// Refuse to migrate files writable by others (mode & 0o022).
			if runtime.GOOS != "windows" && info.Mode().Perm()&0o022 != 0 {
				slog.Warn("Legacy config has unsafe permissions, skipping migration", "path", legacyPath, "mode", info.Mode().Perm())
			} else {
				legacyCfg, loadErr := db.NewConfig(legacyPath)
				if loadErr != nil {
					slog.Warn("Failed to parse legacy config for migration", "path", legacyPath, "error", loadErr)
				} else {
					_ = legacyCfg.Close()
					// Re-read the validated, password-stripped output that NewConfig wrote.
					validatedData, readErr := osReadFile(legacyPath) // #nosec G304 -- path from homeDir + hardcoded strings
					if readErr != nil {
						slog.Warn("Failed to read validated legacy config", "path", legacyPath, "error", readErr)
					} else if writeErr := os.WriteFile(configPath, validatedData, 0o600); writeErr != nil {
						slog.Warn("Failed to write migrated config", "from", legacyPath, "to", configPath, "error", writeErr)
					} else {
						slog.Info("Migrated config from legacy path", "from", legacyPath, "to", configPath)
					}
				}
			}
		}
	}

	return db.NewConfig(configPath)
}
