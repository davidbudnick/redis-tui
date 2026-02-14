package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/davidbudnick/redis-tui/internal/cmd"
	"github.com/davidbudnick/redis-tui/internal/db"
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/davidbudnick/redis-tui/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	opts := parseCLIFlags()

	// Minimal setup before starting UI
	var logs []string

	// Start the UI immediately for perceived speed
	m := ui.NewModel()
	m.Logs = &logs

	// If CLI connection flags were provided, set up auto-connect
	if opts != nil {
		m.CLIConnection = opts
	}

	sendFunc := func(msg tea.Msg) {}
	m.SendFunc = &sendFunc

	// Initialize logger in background (non-blocking)
	logWriter := types.LogWriter{Logs: &logs}
	handler := slog.NewJSONHandler(logWriter, nil)
	slog.SetDefault(slog.New(handler))

	// Load config synchronously for now to ensure it's available for connection operations
	config, err := initConfig()
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
	cmd.Config = config

	p := tea.NewProgram(m, tea.WithAltScreen())
	*m.SendFunc = p.Send
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func parseCLIFlags() *types.Connection {
	conn, showVersion, err := parseFlags(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}
	if showVersion {
		fmt.Printf("redis-tui version %s\n", version)
		os.Exit(0)
	}
	return conn
}

// parseFlags parses the given args into a Connection. Returns nil when no
// --host is provided (interactive mode). showVersion is true when --version
// was requested. Returns an error if flag parsing fails.
func parseFlags(args []string) (conn *types.Connection, showVersion bool, err error) {
	fs := flag.NewFlagSet("redis-tui", flag.ContinueOnError)

	host := fs.String("host", "", "Redis server hostname (required for quick-connect mode)")
	port := fs.Int("port", 6379, "Redis server port")
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
		fmt.Fprintf(os.Stderr, "      --name string       Connection display name\n")
		fmt.Fprintf(os.Stderr, "      --cluster           Enable cluster mode\n")
		fmt.Fprintf(os.Stderr, "      --tls               Enable TLS/SSL\n")
		fmt.Fprintf(os.Stderr, "      --tls-cert string   TLS client certificate file\n")
		fmt.Fprintf(os.Stderr, "      --tls-key string    TLS client private key file\n")
		fmt.Fprintf(os.Stderr, "      --tls-ca string     TLS CA certificate file\n")
		fmt.Fprintf(os.Stderr, "      --tls-skip-verify   Skip TLS certificate verification\n")
		fmt.Fprintf(os.Stderr, "      --version           Print version and exit\n")
	}

	if err := fs.Parse(args); err != nil {
		return nil, false, err
	}

	if *version {
		return nil, true, nil
	}

	// If no host flag provided, return nil (normal interactive mode)
	if *host == "" {
		return nil, false, nil
	}

	conn = &types.Connection{
		Host:       *host,
		Port:       *port,
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

	return conn, false, nil
}

func initConfig() (*db.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}

	configDir := filepath.Join(homeDir, ".redis")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return nil, err
	}

	return db.NewConfig(filepath.Join(configDir, "config.json"))
}
