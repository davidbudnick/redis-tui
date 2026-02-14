# Redis TUI Manager

[![CI](https://github.com/davidbudnick/redis-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/davidbudnick/redis-tui/actions/workflows/ci.yml)
[![Release](https://github.com/davidbudnick/redis-tui/actions/workflows/release.yml/badge.svg)](https://github.com/davidbudnick/redis-tui/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A feature-rich terminal UI for managing Redis databases, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). Browse, edit, and monitor your Redis keys without leaving the terminal.

![Main Screenshot](docs/main.gif)

## Quick Install

```bash
# Native install — recommended (macOS and Linux)
curl -fsSL https://raw.githubusercontent.com/davidbudnick/redis-tui/main/install.sh | bash

# Homebrew (macOS and Linux)
brew tap davidbudnick/homebrew-tap
brew install --cask redis-tui

# Go (requires Go 1.26+)
go install github.com/davidbudnick/redis-tui@latest
```

> **Pre-built binaries** — [Download from GitHub Releases](https://github.com/davidbudnick/redis-tui/releases)

## Screenshots
### Key Browser with Preview
![Main](docs/main.png)

### Live Metrics Dashboard
![Metrics](docs/metrics.png)

## Features

### Browsing and Editing
- **Key browser** with pattern filtering, regex, and fuzzy search
- **All data types** — strings, lists, sets, sorted sets, hashes, and streams
- **Inline editing** with VIM keybindings for strings and collections
- **Tree view** for hierarchical key navigation
- **Favorites and recent keys** for quick access
- **Key templates** for creating keys from predefined structures
- **Value history** — view and restore previous values
- **JSON syntax highlighting** and JSON path queries

### Connections and Security
- **CLI quick connect** — pass `--host`, `--port`, `--password`, etc. to connect without a config file
- **Connection manager** — save and switch between multiple Redis instances
- **TLS/SSL** encryption support
- **SSH tunneling** for secure remote access
- **Connection groups** to organize instances
- **Database switching** between Redis databases (0-15)
- **Cluster support** — connect to any cluster node and press `K` to view all nodes, their roles (master/replica), slot ranges, and link state; cluster metrics in the live dashboard

### Monitoring and Operations
- **Live metrics dashboard** — real-time ops/sec, memory, CPU, network I/O, hit rate, and client count with scrolling ASCII charts; cluster node count display
- **Server info** — version, mode, OS, uptime, memory, and connected clients
- **Memory stats** — detailed usage breakdown and top keys by memory consumption
- **Slow log** — view slow query entries with execution time and command details
- **Client list** — view all connected Redis clients with address, age, and command info
- **Watch mode** — monitor key values for changes in real-time with configurable interval
- **Keyspace events** — subscribe to keyspace notifications (set, del, expire, etc.)
- **Export/Import** — JSON-based key backup and restore
- **Bulk operations** — pattern-based delete and batch TTL across multiple keys
- **Pub/Sub** — publish messages to channels and view active channels
- **Lua scripting** — execute Lua scripts directly against the server
- **Themes** — switch between color themes

## Installation

### Native Install (Recommended)

The install script auto-detects your OS and architecture, downloads the latest release, verifies the checksum, and installs the binary to `/usr/local/bin` (override with `INSTALL_DIR`):

```bash
curl -fsSL https://raw.githubusercontent.com/davidbudnick/redis-tui/main/install.sh | bash

# Custom install directory
INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/davidbudnick/redis-tui/main/install.sh | bash
```

### Homebrew

See [Quick Install](#quick-install) above.

### From Source

```bash
# Clone the repository
git clone https://github.com/davidbudnick/redis-tui.git
cd redis-tui

# Build
make build

# Install to GOPATH/bin
make install
```

### Pre-built Binaries

Download the latest release from the [Releases](https://github.com/davidbudnick/redis-tui/releases) page. Pre-built binaries are available for macOS, Linux, and Windows with no Go installation required.

### Using Go Install

> **Note:** Requires Go 1.26 or later.

```bash
go install github.com/davidbudnick/redis-tui@latest
```

## Usage

```bash
# Launch the interactive connection manager
redis-tui

# Quick connect to a Redis server
redis-tui --host localhost

# Connect with password and specific database
redis-tui -h redis.example.com -p 6380 -a mypassword -n 2

# Connect to a cluster node
redis-tui --host redis.example.com --port 6380 --cluster

# Connect with TLS
redis-tui --host redis.example.com --tls --tls-ca /path/to/ca.pem
```

When `--host` is provided the TUI connects automatically on startup. Without flags the interactive connection manager is shown.

Press `?` inside the app to view the full help screen.

### CLI Flags

| Flag | Short | Description | Default |
| --- | --- | --- | --- |
| `--host` | `-h` | Redis server hostname | |
| `--port` | `-p` | Redis server port | 6379 |
| `--password` | `-a` | Redis password | |
| `--db` | `-n` | Database number (0-15) | 0 |
| `--name` | | Connection display name | `host:port` |
| `--cluster` | | Enable cluster mode | false |
| `--tls` | | Enable TLS/SSL | false |
| `--tls-cert` | | TLS client certificate file | |
| `--tls-key` | | TLS client private key file | |
| `--tls-ca` | | TLS CA certificate file | |
| `--tls-skip-verify` | | Skip TLS certificate verification | false |
| `--version` | | Print version and exit | |

Short flags (`-h`, `-p`, `-a`, `-n`) follow [redis-cli](https://redis.io/docs/latest/develop/connect/cli/) conventions.

### Uninstall

```bash
# Native install
rm -f /usr/local/bin/redis-tui

# Homebrew
brew uninstall --cask redis-tui

# Go
rm -f $(go env GOPATH)/bin/redis-tui
```

<details>
<summary>Keyboard Shortcuts</summary>

### Global

| Key | Action | Key | Action |
| --- | --- | --- | --- |
| `q` | Quit / Go back | `Ctrl+U/D` | Page up/down |
| `?` | Show help | `g/G` | Go to top/bottom |
| `j/k` | Navigate up/down | `home/end` | Go to top/bottom |
| `Ctrl+C` | Force quit | | |

### Connections Screen

| Key | Action | Key | Action |
| --- | --- | --- | --- |
| `Enter` | Connect to selected | `d/delete/backspace` | Delete connection |
| `a/n` | Add new connection | `r` | Refresh list |
| `e` | Edit connection | `Ctrl+T` | Test connection |

### Keys Screen

| Key | Action | Key | Action |
| --- | --- | --- | --- |
| `Enter` | View key details | `O` | View logs |
| `a/n` | Add new key | `B` | Bulk delete |
| `d/delete/backspace` | Delete key | `T` | Batch set TTL |
| `r` | Refresh keys | `F` | View favorites |
| `l` | Load more keys | `W` | Tree view |
| `/` | Filter by pattern | `Ctrl+R` | Regex search |
| `s/S` | Sort / Toggle direction | `Ctrl+F` | Fuzzy search |
| `v` | Search by value | `Ctrl+H` | Recent keys |
| `e` | Export to JSON | `Ctrl+L` | Client list |
| `I` | Import from JSON | `Ctrl+E` | Toggle keyspace events |
| `i` | Server info | `Ctrl+X` | View expiring keys |
| `D` | Switch database | `m` | Live metrics dashboard |
| `f` | Flush database | `M` | Memory stats |
| `p` | Pub/Sub publish | `K` | Cluster info |
| `L` | View slow log | `=` | Compare keys |
| `E` | Execute Lua script | `P` | Key templates |

### Key Detail Screen

| Key | Action | Key | Action |
| --- | --- | --- | --- |
| `e` | Edit value (string) | `r` | Refresh value |
| `a` | Add to collection | `f` | Toggle favorite |
| `x` | Remove from collection | `w` | Watch for changes |
| `t` | Set TTL | `h` | View value history |
| `R` | Rename key | `y` | Copy to clipboard |
| `c` | Copy key | `J` | JSON path query |
| `d/delete` | Delete key | `j/k` | Navigate collection items |
| `esc/backspace` | Go back to keys list | | |

</details>

## Docker Compose Examples

Need a Redis instance to try redis-tui? Docker Compose files are included under [`examples/`](examples/README.md).

```bash
# Standalone Redis on port 6379
docker compose -f examples/standalone/docker-compose.yml up -d
redis-tui -h localhost

# 6-node cluster (3 masters + 3 replicas) on ports 6380-6385
docker compose -f examples/cluster/docker-compose.yml up -d
redis-tui -h localhost -p 6380 --cluster
```

## Configuration

Configuration is stored in `~/.config/redis-tui/config.json`.

### Example Configuration

```json
{
  "connections": [
    {
      "id": 1,
      "name": "Standalone",
      "host": "localhost",
      "port": 6379,
      "db": 0,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": 2,
      "name": "Cluster",
      "host": "localhost",
      "port": 6380,
      "db": 0,
      "use_cluster": true,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ],
  "groups": [
    {
      "name": "local",
      "color": "#50fa7b",
      "connections": [1, 2]
    }
  ],
  "favorites": [
    {
      "connection_id": 1,
      "connection": "Standalone",
      "key": "app:config",
      "label": "App Settings",
      "added_at": "2025-01-15T10:30:00Z"
    }
  ],
  "recent_keys": [
    {
      "connection_id": 1,
      "key": "session:abc123",
      "type": "hash",
      "accessed_at": "2025-01-20T14:00:00Z"
    }
  ],
  "templates": [
    {
      "name": "Session",
      "description": "User session data",
      "key_pattern": "session:{user_id}",
      "type": "hash",
      "default_ttl": 86400000000000,
      "fields": {
        "token": "",
        "created_at": "",
        "user_agent": ""
      }
    },
    {
      "name": "Cache",
      "description": "Cached data with TTL",
      "key_pattern": "cache:{resource}:{id}",
      "type": "string",
      "default_ttl": 3600000000000
    },
    {
      "name": "Rate Limit",
      "description": "Rate limiting counter",
      "key_pattern": "ratelimit:{ip}:{endpoint}",
      "type": "string",
      "default_ttl": 60000000000,
      "default_value": "0"
    },
    {
      "name": "Queue",
      "description": "Job queue",
      "key_pattern": "queue:{name}",
      "type": "list"
    },
    {
      "name": "Leaderboard",
      "description": "Sorted leaderboard",
      "key_pattern": "leaderboard:{game}",
      "type": "zset"
    }
  ],
  "value_history": [
    {
      "key": "app:config",
      "value": "old-value",
      "timestamp": "2025-01-18T09:00:00Z",
      "action": "edit"
    }
  ],
  "key_bindings": {
    "up": "k",
    "down": "j",
    "left": "h",
    "right": "l",
    "page_up": "ctrl+u",
    "page_down": "ctrl+d",
    "top": "g",
    "bottom": "G",
    "select": "enter",
    "back": "esc",
    "quit": "q",
    "help": "?",
    "refresh": "r",
    "delete": "d",
    "add": "a",
    "edit": "e",
    "copy": "c",
    "rename": "R",
    "search": "/",
    "filter": "f",
    "favorite": "F",
    "watch": "w",
    "export": "E",
    "import": "I",
    "server_info": "i",
    "slow_log": "L",
    "lua_script": "X",
    "pubsub": "p",
    "switch_db": "D",
    "ttl": "t",
    "bulk_delete": "B",
    "tree_view": "T",
    "memory_stats": "M",
    "client_list": "C",
    "cluster_info": "K",
    "compare_keys": "=",
    "json_path": "J",
    "copy_clipboard": "y",
    "logs": "O",
    "themes": "ctrl+t",
    "recent_keys": "H",
    "favorites": "ctrl+f",
    "value_history": "u"
  },
  "tree_separator": ":",
  "max_recent_keys": 20,
  "max_value_history": 50,
  "watch_interval_ms": 1000
}
```

> **Note:** Passwords and SSH passphrases are never saved to the config file. They are stripped before serialization for security. The config file is written with `0600` permissions (owner read/write only).

### Connection Options

| Option | Description |
| --- | --- |
| `name` | Display name for the connection |
| `host` | Redis server hostname or IP |
| `port` | Redis server port (default: 6379) |
| `password` | Redis password (never saved to disk) |
| `db` | Redis database number (0-15) |
| `group` | Connection group name (optional) |
| `color` | Display color for the connection (optional) |
| `use_tls` | Enable TLS/SSL connection |
| `tls_config.cert_file` | Client certificate file path |
| `tls_config.key_file` | Client key file path |
| `tls_config.ca_file` | CA certificate file path |
| `tls_config.insecure_skip_verify` | Skip TLS certificate verification |
| `tls_config.server_name` | TLS server name for verification |
| `use_ssh` | Enable SSH tunneling |
| `ssh_config.host` | SSH server hostname |
| `ssh_config.port` | SSH server port |
| `ssh_config.user` | SSH username |
| `ssh_config.private_key_path` | Path to SSH private key file |
| `use_cluster` | Enable Redis cluster mode |

### Custom Keybindings

Keybindings can be customized in the configuration file under the `key_bindings` section. All navigation and action keys can be remapped to your preference. See the full example above for the complete list of customizable bindings and their default values.

## Requirements

- Go 1.26 or later (for building from source or `go install`)
- A terminal that supports 256 colors
- Redis server 4.0 or later

## Supported Platforms

- macOS (Intel and Apple Silicon)
- Linux (amd64, arm64)
- Windows (amd64)

## Development

```bash
# Install development dependencies
make dev-deps

# Run the application
make run

# Run tests
make test

# Run tests with coverage
make test-cover

# Run linter
make lint

# Format code
make fmt

# Build the application
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean

# Create a release with goreleaser
make release

# Create a snapshot release (no publish)
make snapshot
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling library
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [go-redis](https://github.com/redis/go-redis) - Redis client

## Keywords

redis, redis-cli, redis-client, redis-tui, redis-gui, redis-manager, terminal, tui, cli, go, golang, database, key-value, cache, devops, sysadmin
