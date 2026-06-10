# Redis TUI Demo
Output "docs/main.gif"

Require ./bin/redis-tui

Set Shell "zsh"
Set FontSize 18
Set FontFamily "SFMono Nerd Font"
Set Theme "Dracula"
Set Width 1920
Set Height 1080
Set TypingSpeed 0.04s
Set Padding 20

# Clean up leftover key, seed an isolated demo config (standalone + cluster only), and launch
Hide
Type "docker exec standalone-redis-1 redis-cli DEL 'hello world'"
Enter
Sleep 1s
Type "export DEMO_HOME=$(mktemp -d) && mkdir -p $DEMO_HOME/.config/redis-tui && cp docs/demo-config.json $DEMO_HOME/.config/redis-tui/config.json"
Enter
Sleep 0.5s
Type "HOME=$DEMO_HOME ./bin/redis-tui"
Enter
Sleep 2s
Show
Sleep 4s

# Connect to standalone (first connection, already selected)
Enter
Sleep 2.5s

# Filter to show JSON keys
Type "/"
Sleep 0.3s
Type "json"
Sleep 0.5s
Enter
Sleep 2s

# View the first JSON key detail
Enter
Sleep 3s

# Back to keys list
Escape
Sleep 1s

# Clear the filter
Type "/"
Sleep 0.3s
Backspace 4
Sleep 0.3s
Enter
Sleep 2s

# Add a new key
Type "a"
Sleep 1s

# Type key name
Type "hello world"
Sleep 0.3s

# Move to value field
Tab
Sleep 0.3s

# Type the value
Type "Hello from Redis TUI!"
Sleep 0.5s

# Create the key
Enter
Sleep 2s

# Back to connections screen
Escape
Sleep 1.5s

# Navigate down to cluster connection
Down
Sleep 0.5s

# Connect to cluster
Enter
Sleep 3s
