package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// getScreenView returns the view for the current screen.
// Uses a switch instead of a map to avoid per-frame heap allocation.
func (m Model) getScreenView() string {
	switch m.Screen {
	case types.ScreenConnections:
		return m.viewConnections()
	case types.ScreenAddConnection:
		return m.viewAddConnection()
	case types.ScreenEditConnection:
		return m.viewEditConnection()
	case types.ScreenKeys:
		return m.viewKeys()
	case types.ScreenKeyDetail:
		return m.viewKeyDetail()
	case types.ScreenAddKey:
		return m.viewAddKey()
	case types.ScreenHelp:
		return m.viewHelp()
	case types.ScreenConfirmDelete:
		return m.viewConfirmDelete()
	case types.ScreenServerInfo:
		return m.viewServerInfo()
	case types.ScreenTTLEditor:
		return m.viewTTLEditor()
	case types.ScreenEditValue:
		return m.viewEditValue()
	case types.ScreenAddToCollection:
		return m.viewAddToCollection()
	case types.ScreenRemoveFromCollection:
		return m.viewRemoveFromCollection()
	case types.ScreenRenameKey:
		return m.viewRenameKey()
	case types.ScreenCopyKey:
		return m.viewCopyKey()
	case types.ScreenPubSub, types.ScreenPublishMessage:
		return m.viewPubSub()
	case types.ScreenSwitchDB:
		return m.viewSwitchDB()
	case types.ScreenSearchValues:
		return m.viewSearchValues()
	case types.ScreenExport:
		return m.viewExport()
	case types.ScreenImport:
		return m.viewImport()
	case types.ScreenSlowLog:
		return m.viewSlowLog()
	case types.ScreenLuaScript:
		return m.viewLuaScript()
	case types.ScreenTestConnection:
		return m.viewTestConnection()
	case types.ScreenLogs:
		return m.viewLogs()
	case types.ScreenBulkDelete:
		return m.viewBulkDelete()
	case types.ScreenBatchTTL:
		return m.viewBatchTTL()
	case types.ScreenFavorites:
		return m.viewFavorites()
	case types.ScreenRecentKeys:
		return m.viewRecentKeys()
	case types.ScreenTreeView:
		return m.viewTreeView()
	case types.ScreenRegexSearch:
		return m.viewRegexSearch()
	case types.ScreenFuzzySearch:
		return m.viewFuzzySearch()
	case types.ScreenClientList:
		return m.viewClientList()
	case types.ScreenMemoryStats:
		return m.viewMemoryStats()
	case types.ScreenClusterInfo:
		return m.viewClusterInfo()
	case types.ScreenCompareKeys:
		return m.viewCompareKeys()
	case types.ScreenTemplates:
		return m.viewTemplates()
	case types.ScreenValueHistory:
		return m.viewValueHistory()
	case types.ScreenKeyspaceEvents:
		return m.viewKeyspaceEvents()
	case types.ScreenJSONPath:
		return m.viewJSONPath()
	case types.ScreenExpiringKeys:
		return m.viewExpiringKeys()
	case types.ScreenLiveMetrics:
		return m.viewLiveMetrics()
	case types.ScreenPubSubChannels:
		return m.viewPubSubChannels()
	case types.ScreenRedisConfig:
		return m.viewRedisConfig()
	default:
		return ""
	}
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	if m.Width < 50 || m.Height < 15 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center,
			"Terminal too small.\nResize to at least 50x15.")
	}

	content := m.getScreenView()

	// Status bar
	status := m.getStatusBar()

	fullContent := content + "\n\n" + status

	// Only vertically center short modal/dialog screens
	vPos := lipgloss.Position(lipgloss.Top)
	switch m.Screen {
	case types.ScreenKeyDetail, types.ScreenConnections,
		types.ScreenAddConnection, types.ScreenEditConnection,
		types.ScreenConfirmDelete, types.ScreenTestConnection,
		types.ScreenHelp, types.ScreenAddKey, types.ScreenTTLEditor,
		types.ScreenRenameKey, types.ScreenCopyKey,
		types.ScreenSearchValues, types.ScreenRegexSearch,
		types.ScreenFuzzySearch, types.ScreenSwitchDB,
		types.ScreenBulkDelete, types.ScreenBatchTTL,
		types.ScreenExport, types.ScreenImport,
		types.ScreenCompareKeys, types.ScreenJSONPath,
		types.ScreenPublishMessage, types.ScreenLuaScript:
		vPos = lipgloss.Center
	}

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, vPos, fullContent,
		lipgloss.WithWhitespaceChars(" "))
}

func (m Model) getStatusBar() string {
	if m.Loading {
		return dimStyle.Render("Loading...")
	}
	if m.StatusMsg != "" {
		if strings.HasPrefix(m.StatusMsg, "Error") {
			return errorStyle.Render(m.StatusMsg)
		}
		return successStyle.Render(m.StatusMsg)
	}
	if m.UpdateAvailable != "" {
		return errorStyle.Render("Update available: " + m.UpdateAvailable + " — run: " + m.UpdateCmd)
	}
	return ""
}
