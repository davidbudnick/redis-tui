package ui

import (
	"time"

	"github.com/davidbudnick/redis-tui/internal/cmd"
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/kujtimiihoxha/vimtea"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Resize vimtea editor if active
		if m.Screen == types.ScreenEditValue && m.VimEditor != nil {
			sized, _ := m.VimEditor.SetSize(msg.Width-4, msg.Height-10)
			m.VimEditor = sized.(vimtea.Editor)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case types.TickMsg:
		return m.handleTickMsg()

	// Connection messages
	case types.ConnectionsLoadedMsg:
		return m.handleConnectionsLoadedMsg(msg)
	case types.ConnectionAddedMsg:
		return m.handleConnectionAddedMsg(msg)
	case types.ConnectionUpdatedMsg:
		return m.handleConnectionUpdatedMsg(msg)
	case types.ConnectionDeletedMsg:
		return m.handleConnectionDeletedMsg(msg)
	case types.AutoConnectMsg:
		return m.handleAutoConnectMsg(msg)
	case types.ConnectedMsg:
		return m.handleConnectedMsg(msg)
	case types.DisconnectedMsg:
		return m.handleDisconnectedMsg()
	case types.ConnectionTestMsg:
		return m.handleConnectionTestMsg(msg)
	case types.GroupsLoadedMsg:
		return m.handleGroupsLoadedMsg(msg)

	// Key messages
	case types.KeysLoadedMsg:
		return m.handleKeysLoadedMsg(msg)
	case types.KeyValueLoadedMsg:
		return m.handleKeyValueLoadedMsg(msg)
	case types.KeyPreviewLoadedMsg:
		return m.handleKeyPreviewLoadedMsg(msg)
	case types.KeyDeletedMsg:
		return m.handleKeyDeletedMsg(msg)
	case types.KeySetMsg:
		return m.handleKeySetMsg(msg)
	case types.KeyRenamedMsg:
		return m.handleKeyRenamedMsg(msg)
	case types.KeyCopiedMsg:
		return m.handleKeyCopiedMsg(msg)

	// Value messages
	case types.ValueEditedMsg:
		return m.handleValueEditedMsg(msg)
	case types.ItemAddedToCollectionMsg:
		return m.handleItemAddedToCollectionMsg(msg)
	case types.ItemRemovedFromCollectionMsg:
		return m.handleItemRemovedFromCollectionMsg(msg)

	// TTL messages
	case types.TTLSetMsg:
		return m.handleTTLSetMsg(msg)
	case types.BatchTTLSetMsg:
		return m.handleBatchTTLSetMsg(msg)

	// Server messages
	case types.ServerInfoLoadedMsg:
		return m.handleServerInfoLoadedMsg(msg)
	case types.DBSwitchedMsg:
		return m.handleDBSwitchedMsg(msg)
	case types.FlushDBMsg:
		return m.handleFlushDBMsg(msg)
	case types.SlowLogLoadedMsg:
		return m.handleSlowLogLoadedMsg(msg)
	case types.ClientListLoadedMsg:
		return m.handleClientListLoadedMsg(msg)
	case types.MemoryStatsLoadedMsg:
		return m.handleMemoryStatsLoadedMsg(msg)
	case types.ClusterInfoLoadedMsg:
		return m.handleClusterInfoLoadedMsg(msg)
	case types.ClusterNodesLoadedMsg:
		return m.handleClusterNodesLoadedMsg(msg)
	case types.MemoryUsageMsg:
		return m.handleMemoryUsageMsg(msg)

	// Script and Pub/Sub messages
	case types.LuaScriptResultMsg:
		return m.handleLuaScriptResultMsg(msg)
	case types.PublishResultMsg:
		return m.handlePublishResultMsg(msg)
	case types.KeyspaceEventMsg:
		return m.handleKeyspaceEventMsg(msg)

	// Import/Export messages
	case types.ExportCompleteMsg:
		return m.handleExportCompleteMsg(msg)
	case types.ImportCompleteMsg:
		return m.handleImportCompleteMsg(msg)

	// Feature messages
	case types.BulkDeleteMsg:
		return m.handleBulkDeleteMsg(msg)
	case types.FavoritesLoadedMsg:
		return m.handleFavoritesLoadedMsg(msg)
	case types.FavoriteAddedMsg:
		return m.handleFavoriteAddedMsg(msg)
	case types.FavoriteRemovedMsg:
		return m.handleFavoriteRemovedMsg(msg)
	case types.RecentKeysLoadedMsg:
		return m.handleRecentKeysLoadedMsg(msg)
	case types.TemplatesLoadedMsg:
		return m.handleTemplatesLoadedMsg(msg)
	case types.ValueHistoryMsg:
		return m.handleValueHistoryMsg(msg)

	// Search messages
	case types.RegexSearchResultMsg:
		return m.handleRegexSearchResultMsg(msg)
	case types.FuzzySearchResultMsg:
		return m.handleFuzzySearchResultMsg(msg)
	case types.CompareKeysResultMsg:
		return m.handleCompareKeysResultMsg(msg)

	// Live Metrics
	case types.LiveMetricsMsg:
		return m.handleLiveMetricsMsg(msg)
	case types.LiveMetricsTickMsg:
		return m.handleLiveMetricsTickMsg()

	// Clipboard
	case types.ClipboardCopiedMsg:
		return m.handleClipboardCopiedMsg(msg)

	// Editor messages (from vimtea :w/:q commands)
	case types.EditorSaveMsg:
		if m.CurrentKey != nil {
			m.Loading = true
			return m, cmd.EditStringValueCmd(m.CurrentKey.Key, msg.Content)
		}
	case types.EditorQuitMsg:
		m.Screen = types.ScreenKeyDetail
		return m, nil

	default:
		// Pass unhandled messages to vimtea editor if active
		if m.Screen == types.ScreenEditValue && m.VimEditor != nil {
			updated, editorCmd := m.VimEditor.Update(msg)
			m.VimEditor = updated.(vimtea.Editor)
			return m, editorCmd
		}
	}
	return m, nil
}

// Tick handler
func (m Model) handleTickMsg() (tea.Model, tea.Cmd) {
	now := time.Now()
	if !m.LastTickTime.IsZero() {
		elapsed := now.Sub(m.LastTickTime)
		for i := range m.Keys {
			if m.Keys[i].TTL > 0 {
				m.Keys[i].TTL -= elapsed
				if m.Keys[i].TTL < 0 {
					m.Keys[i].TTL = 0
				}
			}
		}
		if m.CurrentKey != nil && m.CurrentKey.TTL > 0 {
			m.CurrentKey.TTL -= elapsed
			if m.CurrentKey.TTL < 0 {
				m.CurrentKey.TTL = 0
			}
		}
	}
	m.LastTickTime = now

	// Count expired keys first to avoid unnecessary allocations
	expiredCount := 0
	for _, k := range m.Keys {
		if k.TTL == 0 {
			expiredCount++
		}
	}

	if expiredCount > 0 {
		activeKeys := make([]types.RedisKey, 0, len(m.Keys)-expiredCount)
		for _, k := range m.Keys {
			if k.TTL != 0 {
				activeKeys = append(activeKeys, k)
			}
		}
		m.Keys = activeKeys
		if m.SelectedKeyIdx >= len(m.Keys) && m.SelectedKeyIdx > 0 {
			m.SelectedKeyIdx = len(m.Keys) - 1
		}
		if m.CurrentKey != nil && m.CurrentKey.TTL == 0 {
			m.CurrentKey = nil
			m.Screen = types.ScreenKeys
			m.StatusMsg = "Key expired"
		}
	}

	var cmds []tea.Cmd
	cmds = append(cmds, tickCmd())
	if m.WatchActive {
		cmds = append(cmds, cmd.WatchKeyTickCmd())
	}
	if m.LiveMetricsActive {
		cmds = append(cmds, cmd.LoadLiveMetricsCmd())
	}
	return m, tea.Batch(cmds...)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return types.TickMsg{}
	})
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.Screen == types.ScreenConnections {
			return m, tea.Quit
		}
		if m.Screen == types.ScreenKeys {
			return m, tea.Quit
		}
	case "?":
		if m.Screen != types.ScreenHelp && m.Screen != types.ScreenAddConnection &&
			m.Screen != types.ScreenEditConnection && m.Screen != types.ScreenAddKey &&
			m.Screen != types.ScreenTTLEditor {
			m.Screen = types.ScreenHelp
			return m, nil
		}
	}

	switch m.Screen {
	case types.ScreenConnections:
		return m.handleConnectionsScreen(msg)
	case types.ScreenAddConnection:
		return m.handleAddConnectionScreen(msg)
	case types.ScreenEditConnection:
		return m.handleEditConnectionScreen(msg)
	case types.ScreenKeys:
		return m.handleKeysScreen(msg)
	case types.ScreenKeyDetail:
		return m.handleKeyDetailScreen(msg)
	case types.ScreenAddKey:
		return m.handleAddKeyScreen(msg)
	case types.ScreenConfirmDelete:
		return m.handleConfirmDeleteScreen(msg)
	case types.ScreenTTLEditor:
		return m.handleTTLEditorScreen(msg)
	case types.ScreenHelp:
		return m.handleHelpScreen(msg)
	case types.ScreenServerInfo:
		return m.handleServerInfoScreen(msg)
	case types.ScreenEditValue:
		return m.handleEditValueScreen(msg)
	case types.ScreenAddToCollection:
		return m.handleAddToCollectionScreen(msg)
	case types.ScreenRemoveFromCollection:
		return m.handleRemoveFromCollectionScreen(msg)
	case types.ScreenRenameKey:
		return m.handleRenameKeyScreen(msg)
	case types.ScreenCopyKey:
		return m.handleCopyKeyScreen(msg)
	case types.ScreenPubSub:
		return m.handlePubSubScreen(msg)
	case types.ScreenPublishMessage:
		return m.handlePublishMessageScreen(msg)
	case types.ScreenSwitchDB:
		return m.handleSwitchDBScreen(msg)
	case types.ScreenSearchValues:
		return m.handleSearchValuesScreen(msg)
	case types.ScreenExport:
		return m.handleExportScreen(msg)
	case types.ScreenImport:
		return m.handleImportScreen(msg)
	case types.ScreenSlowLog:
		return m.handleSlowLogScreen(msg)
	case types.ScreenLuaScript:
		return m.handleLuaScriptScreen(msg)
	case types.ScreenTestConnection:
		return m.handleTestConnectionScreen(msg)
	case types.ScreenLogs:
		return m.handleLogsScreen(msg)
	case types.ScreenBulkDelete:
		return m.handleBulkDeleteScreen(msg)
	case types.ScreenBatchTTL:
		return m.handleBatchTTLScreen(msg)
	case types.ScreenFavorites:
		return m.handleFavoritesScreen(msg)
	case types.ScreenRecentKeys:
		return m.handleRecentKeysScreen(msg)
	case types.ScreenTreeView:
		return m.handleTreeViewScreen(msg)
	case types.ScreenRegexSearch:
		return m.handleRegexSearchScreen(msg)
	case types.ScreenFuzzySearch:
		return m.handleFuzzySearchScreen(msg)
	case types.ScreenClientList:
		return m.handleClientListScreen(msg)
	case types.ScreenMemoryStats:
		return m.handleMemoryStatsScreen(msg)
	case types.ScreenClusterInfo:
		return m.handleClusterInfoScreen(msg)
	case types.ScreenCompareKeys:
		return m.handleCompareKeysScreen(msg)
	case types.ScreenTemplates:
		return m.handleTemplatesScreen(msg)
	case types.ScreenValueHistory:
		return m.handleValueHistoryScreen(msg)
	case types.ScreenKeyspaceEvents:
		return m.handleKeyspaceEventsScreen(msg)
	case types.ScreenJSONPath:
		return m.handleJSONPathScreen(msg)
	case types.ScreenWatchKey:
		return m.handleWatchKeyScreen(msg)
	case types.ScreenConnectionGroups:
		return m.handleConnectionGroupsScreen(msg)
	case types.ScreenExpiringKeys:
		return m.handleExpiringKeysScreen(msg)
	case types.ScreenLiveMetrics:
		return m.handleLiveMetricsScreen(msg)
	}
	return m, nil
}
