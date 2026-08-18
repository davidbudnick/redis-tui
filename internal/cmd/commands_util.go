package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

// detectedOS is the runtime OS, overridable in tests.
var (
	detectedOS = runtime.GOOS
)

// clipboardCmd returns the command name and args for the platform's clipboard
// utility. Returns ("", nil) if none is available.
func clipboardCmd() (string, []string) {
	switch detectedOS {
	case "darwin":
		return "pbcopy", nil
	case "windows":
		return "clip", nil
	default: // linux, freebsd, etc.
		if path, err := exec.LookPath("xclip"); err == nil {
			return path, []string{"-selection", "clipboard"}
		}
		if path, err := exec.LookPath("xsel"); err == nil {
			return path, []string{"--clipboard", "--input"}
		}
		return "", nil
	}
}

func (c *Commands) CheckVersion(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		if currentVersion == "" || currentVersion == "dev" {
			return types.UpdateAvailableMsg{}
		}

		latest, err := fetchLatestTag()
		if err != nil {
			return types.UpdateAvailableMsg{Err: err}
		}

		if strings.TrimPrefix(latest, "v") == strings.TrimPrefix(currentVersion, "v") {
			return types.UpdateAvailableMsg{}
		}

		upgradeCmd := detectUpgradeCmd()

		return types.UpdateAvailableMsg{
			LatestVersion: latest,
			UpgradeCmd:    upgradeCmd,
		}
	}
}

func (c *Commands) WatchKeyTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return types.WatchTickMsg{}
	})
}

func (c *Commands) CopyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		name, args := clipboardCmd()
		if name == "" {
			return types.ClipboardCopiedMsg{Content: content, Err: fmt.Errorf("no clipboard utility found (install pbcopy, xclip, or xsel)")}
		}
		cmd := exec.Command(name, args...) // #nosec G204 -- name/args from clipboardCmd() are hardcoded values
		cmd.Stdin = strings.NewReader(content)
		err := cmd.Run()
		return types.ClipboardCopiedMsg{Content: content, Err: err}
	}
}

// Version check helpers

const githubRepo = "davidbudnick/redis-tui"

var (
	versionHTTPClient = &http.Client{Timeout: 10 * time.Second}
	// githubReleaseURL is overridable in tests to cover the NewRequest error path.
	githubReleaseURL = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func fetchLatestTag() (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubReleaseURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := versionHTTPClient.Do(req) // #nosec G107 G704 - URL built from hardcoded GitHub API base
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in response")
	}

	return release.TagName, nil
}

// osExecutable indirection lets tests override executable path detection.
var osExecutable = os.Executable

func detectUpgradeCmd() string {
	execPath, err := osExecutable()
	if err != nil {
		return "redis-tui --update"
	}

	p := strings.ReplaceAll(execPath, `\`, "/")
	if strings.Contains(p, "/Cellar/") || strings.Contains(p, "/homebrew/") {
		return "brew upgrade redis-tui"
	}
	if strings.Contains(p, "/go/bin/") {
		return "go install github.com/davidbudnick/redis-tui@latest"
	}
	return "redis-tui --update"
}
