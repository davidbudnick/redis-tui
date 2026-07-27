# Static README screenshots — key browser, protobuf decode, key detail
Require ./bin/redis-tui

Set Shell "zsh"
Set FontSize 16
Set FontFamily "FiraCode Nerd Font Mono"
Set Theme "Dracula"
Set Width 1600
Set Height 960
Set TypingSpeed 0.03s
Set Padding 16

Hide
Type "export DEMO_HOME=$(mktemp -d) && mkdir -p $DEMO_HOME/.config/redis-tui && cp docs/demo-config.json $DEMO_HOME/.config/redis-tui/config.json"
Enter
Sleep 0.4s
# Force color: agent shells often export NO_COLOR=1 / TERM=dumb
Type "unset NO_COLOR FORCE_COLOR CLICOLOR; export COLORTERM=truecolor TERM=xterm-256color CLICOLOR_FORCE=1 FORCE_COLOR=1"
Enter
Sleep 0.2s
Type "HOME=$DEMO_HOME ./bin/redis-tui"
Enter
Sleep 1.5s
Show
Sleep 1s

# Connect to standalone
Enter
Sleep 2.5s

# --- Key browser: full list with mixed types ---
Sleep 1s
Screenshot docs/main.png

# --- Key browser: hash preview (filtered) ---
Type "/"
Sleep 0.2s
Type "user:"
Sleep 0.3s
Enter
Sleep 1.8s
Down
Sleep 0.8s
Screenshot docs/keys-preview.png

# --- Key browser: s2+protobuf preview ---
Type "/"
Sleep 0.2s
Backspace 20
Type "pb:s2"
Sleep 0.3s
Enter
Sleep 2s
Screenshot docs/protobuf.png

# --- Key detail: JSON value ---
Type "/"
Sleep 0.2s
Backspace 20
Type "json:user"
Sleep 0.3s
Enter
Sleep 1.5s
Enter
Sleep 2s
Screenshot docs/detail.png

Escape
Sleep 0.5s
Escape
Sleep 0.3s
