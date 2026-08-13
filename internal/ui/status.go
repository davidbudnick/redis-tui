package ui

import (
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

const (
	statusFlashDuration = 2 * time.Second
	statusErrorDuration = 4 * time.Second
)

func (m Model) handleStatusClear(msg types.StatusClearMsg) (tea.Model, tea.Cmd) {
	if msg.ID == m.statusID {
		m.StatusMsg = ""
	}
	return m, nil
}

func (m *Model) scheduleStatusClear(prev string) tea.Cmd {
	if m.StatusMsg == prev {
		return nil
	}
	m.statusID++
	if m.StatusMsg == "" || isStickyStatus(m.StatusMsg) {
		return nil
	}
	return statusClearAfter(m.statusID, statusDuration(m.StatusMsg))
}

func statusClearAfter(id int, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return types.StatusClearMsg{ID: id}
	})
}

func statusDuration(msg string) time.Duration {
	if isErrorStatus(msg) {
		return statusErrorDuration
	}
	return statusFlashDuration
}

func isStickyStatus(msg string) bool {
	switch msg {
	case "Connecting...", "Watching key for changes...":
		return true
	default:
		return false
	}
}

func isErrorStatus(msg string) bool {
	if strings.HasPrefix(msg, "Error") {
		return true
	}
	if strings.Contains(msg, "failed") {
		return true
	}
	if strings.Contains(strings.ToLower(msg), "error:") {
		return true
	}
	if strings.HasPrefix(msg, "Invalid ") {
		return true
	}
	return false
}
