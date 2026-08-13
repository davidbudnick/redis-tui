package ui

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/davidbudnick/redis-tui/internal/ui/editor"

	tea "charm.land/bubbletea/v2"
)

func createVimEditor(content string, width, height int, fileName string) *editor.Model {
	return editor.New(content, width, height, fileName)
}

func (m *Model) debounceKeyPreview() tea.Cmd {
	if len(m.Keys) == 0 || m.SelectedKeyIdx < 0 || m.SelectedKeyIdx >= len(m.Keys) {
		return nil
	}
	m.PreviewSeq++
	seq := m.PreviewSeq
	key := m.Keys[m.SelectedKeyIdx].Key
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return types.PreviewDebounceMsg{Seq: seq, Key: key}
	})
}

func (m Model) handleKeysScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs.PatternInput.Focused() {
		switch msg.String() {
		case "enter":
			pattern := m.Inputs.PatternInput.Value()
			if pattern != "" && !strings.ContainsAny(pattern, "*?[]") {
				pattern = "*" + pattern + "*"
			}
			m.KeyPattern = pattern
			m.Inputs.PatternInput.Blur()
			m.KeyCursor = 0
			m.Loading = true
			return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
		case "esc":
			m.Inputs.PatternInput.Blur()
			m.Inputs.PatternInput.SetValue("")
			m.KeyPattern = ""
			m.KeyCursor = 0
			m.SearchSeq++
			m.Loading = true
			return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
		default:
			var inputCmd tea.Cmd
			m.Inputs.PatternInput, inputCmd = m.Inputs.PatternInput.Update(msg)
			m.SearchSeq++
			seq := m.SearchSeq
			debounceCmd := tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
				return types.SearchDebounceMsg{Seq: seq}
			})
			return m, tea.Batch(inputCmd, debounceCmd)
		}
	}

	switch msg.String() {
	case "up", "k":
		if m.SelectedKeyIdx > 0 {
			m.SelectedKeyIdx--
			return m, m.debounceKeyPreview()
		}
	case "down", "j":
		if len(m.Keys) > 0 && m.SelectedKeyIdx < len(m.Keys)-1 {
			m.SelectedKeyIdx++
			return m, m.debounceKeyPreview()
		}
	case "pgup", "ctrl+u":
		m.SelectedKeyIdx -= 10
		if m.SelectedKeyIdx < 0 {
			m.SelectedKeyIdx = 0
		}
		return m, m.debounceKeyPreview()
	case "pgdown", "ctrl+d":
		if len(m.Keys) == 0 {
			return m, nil
		}
		m.SelectedKeyIdx += 10
		if m.SelectedKeyIdx >= len(m.Keys) {
			m.SelectedKeyIdx = len(m.Keys) - 1
		}
		return m, m.debounceKeyPreview()
	case "home", "g":
		m.SelectedKeyIdx = 0
		return m, m.debounceKeyPreview()
	case "end", "G":
		if len(m.Keys) > 0 {
			m.SelectedKeyIdx = len(m.Keys) - 1
			return m, m.debounceKeyPreview()
		}
	case "enter":
		if len(m.Keys) > 0 && m.SelectedKeyIdx < len(m.Keys) {
			key := m.Keys[m.SelectedKeyIdx]
			m.CurrentKey = &key
			m.Loading = true
			m.StatusMsg = ""
			m.SelectedItemIdx = 0
			return m, tea.Batch(m.Cmds.LoadKeyValue(key.Key), m.Cmds.GetMemoryUsage(key.Key))
		}
	case "a", "n":
		m.Screen = types.ScreenAddKey
		m.resetAddKeyInputs()
	case "d", "delete", "backspace":
		if len(m.Keys) > 0 && m.SelectedKeyIdx < len(m.Keys) {
			m.ConfirmType = "key"
			m.ConfirmData = m.Keys[m.SelectedKeyIdx]
			m.Screen = types.ScreenConfirmDelete
		}
	case "r":
		m.Loading = true
		m.KeyCursor = 0
		return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
	case "l":
		if m.KeyCursor > 0 {
			m.Loading = true
			return m, m.Cmds.LoadKeys(m.KeyPattern, m.KeyCursor, m.ScanSize)
		}
	case "i":
		return m, m.Cmds.LoadServerInfo()
	case "/":
		m.Inputs.PatternInput.Focus()
	case "f":
		m.ConfirmType = "flushdb"
		m.Screen = types.ScreenConfirmDelete
	case "s":
		m.sortKeys()
	case "S":
		m.SortAsc = !m.SortAsc
		m.sortKeys()
	case "v":
		m.Inputs.SearchValueInput.SetValue("")
		m.Inputs.SearchValueInput.Focus()
		m.Screen = types.ScreenSearchValues
	case "e":
		m.Screen = types.ScreenExport
		m.Inputs.ExportInput.Focus()
	case "I":
		m.Screen = types.ScreenImport
		m.Inputs.ImportInput.Focus()
	case "p":
		m.Loading = true
		return m, m.Cmds.GetPubSubChannels("*")
	case "L":
		m.Loading = true
		return m, m.Cmds.GetSlowLog(20)
	case "E":
		m.Inputs.LuaScriptInput.SetValue("")
		m.Inputs.LuaScriptInput.Focus()
		m.LuaResult = ""
		m.Screen = types.ScreenLuaScript
	case "D":
		m.Inputs.DBSwitchInput.SetValue("")
		m.Inputs.DBSwitchInput.Focus()
		m.Screen = types.ScreenSwitchDB
	case "O":
		m.LogCursor = 0
		m.ShowingLogDetail = false
		m.Screen = types.ScreenLogs
	case "B":
		m.Inputs.BulkDeleteInput.SetValue("")
		m.Inputs.BulkDeleteInput.Focus()
		m.BulkDeletePreview = nil
		m.Screen = types.ScreenBulkDelete
	case "T":
		m.Inputs.BatchTTLInput.SetValue("")
		m.Inputs.BatchTTLPattern.SetValue("")
		m.Inputs.BatchTTLInput.Focus()
		m.Screen = types.ScreenBatchTTL
	case "F":
		connID := int64(0)
		if m.CurrentConn != nil {
			connID = m.CurrentConn.ID
		}
		m.Screen = types.ScreenFavorites
		return m, m.Cmds.LoadFavorites(connID)
	case "ctrl+r":
		m.Inputs.RegexSearchInput.SetValue("")
		m.Inputs.RegexSearchInput.Focus()
		m.Screen = types.ScreenRegexSearch
	case "ctrl+f":
		m.Inputs.FuzzySearchInput.SetValue("")
		m.Inputs.FuzzySearchInput.Focus()
		m.Screen = types.ScreenFuzzySearch
	case "ctrl+l":
		m.Loading = true
		return m, m.Cmds.GetClientList()
	case "m":
		m.LiveMetricsActive = true
		m.Loading = true
		return m, m.Cmds.LoadLiveMetrics()
	case "M":
		m.Loading = true
		return m, m.Cmds.GetMemoryStats()
	case "C":
		m.Loading = true
		return m, m.Cmds.GetClusterInfo()
	case "K":
		m.Inputs.CompareKey1Input.SetValue("")
		m.Inputs.CompareKey2Input.SetValue("")
		m.Inputs.CompareKey1Input.Focus()
		m.CompareFocusIdx = 0
		m.Screen = types.ScreenCompareKeys
	case "P":
		return m, m.Cmds.LoadTemplates()
	case "ctrl+h":
		connID := int64(0)
		if m.CurrentConn != nil {
			connID = m.CurrentConn.ID
		}
		m.Screen = types.ScreenRecentKeys
		return m, m.Cmds.LoadRecentKeys(connID)
	case "ctrl+e":
		m.KeyspaceSubActive = !m.KeyspaceSubActive
		if m.KeyspaceSubActive {
			var sendFunc func(tea.Msg)
			if m.SendFunc != nil {
				sendFunc = *m.SendFunc
			}
			return m, m.Cmds.SubscribeKeyspace("*", sendFunc)
		}
		m.StatusMsg = "Keyspace events disabled"
	case "W":
		m.TreeSeparator = ":"
		m.Screen = types.ScreenTreeView
		m.Loading = true
		return m, m.Cmds.LoadKeyPrefixes(m.TreeSeparator, 3)
	case "y":
		if len(m.Keys) > 0 && m.SelectedKeyIdx < len(m.Keys) {
			return m, m.Cmds.CopyToClipboard(m.Keys[m.SelectedKeyIdx].Key)
		}
	case "ctrl+g":
		m.Loading = true
		return m, m.Cmds.LoadRedisConfig("*")
	case "ctrl+x":
		var expiring []types.RedisKey
		for _, k := range m.Keys {
			if k.TTL > 0 && k.TTL.Seconds() < float64(m.ExpiryThreshold) {
				expiring = append(expiring, k)
			}
		}
		m.ExpiringKeys = expiring
		m.Screen = types.ScreenExpiringKeys
	case "esc":
		if m.KeyPattern != "" {
			m.Inputs.PatternInput.SetValue("")
			m.KeyPattern = ""
			m.KeyCursor = 0
			m.SearchSeq++
			m.Loading = true
			return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
		}
		m.Screen = types.ScreenConnections
	}
	return m, nil
}

func (m *Model) sortKeys() {
	switch m.SortBy {
	case "name":
		m.SortBy = "type"
	case "type":
		m.SortBy = "ttl"
	case "ttl":
		m.SortBy = "name"
	default:
		m.SortBy = "name"
	}

	sort.Slice(m.Keys, func(i, j int) bool {
		var less bool
		switch m.SortBy {
		case "name":
			less = m.Keys[i].Key < m.Keys[j].Key
		case "type":
			less = string(m.Keys[i].Type) < string(m.Keys[j].Type)
		case "ttl":
			less = m.Keys[i].TTL < m.Keys[j].TTL
		}
		if m.SortAsc {
			return less
		}
		return !less
	})
}

func (m Model) handleKeyDetailScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "d", "delete":
		if m.CurrentKey != nil {
			m.ConfirmType = "key"
			m.ConfirmData = *m.CurrentKey
			m.Screen = types.ScreenConfirmDelete
		}
	case "t":
		if m.CurrentKey != nil {
			m.Inputs.TTLInput.SetValue("")
			m.Inputs.TTLInput.Focus()
			m.Screen = types.ScreenTTLEditor
		}
	case "r":
		if m.CurrentKey != nil {
			m.Loading = true
			m.DetailScroll = 0
			m.DetailCursor = 0
			return m, tea.Batch(m.Cmds.LoadKeyValue(m.CurrentKey.Key), m.Cmds.GetMemoryUsage(m.CurrentKey.Key))
		}
	case "e":
		if m.CurrentKey != nil && (m.CurrentKey.Type == types.KeyTypeString || m.CurrentKey.Type == types.KeyTypeJSON) {
			content := m.CurrentValue.StringValue
			fileName := ""
			if m.CurrentKey.Type == types.KeyTypeJSON {
				content = m.CurrentValue.JSONValue
				fileName = "value.json"
			}
			if trimmed := strings.TrimSpace(content); len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
				var buf bytes.Buffer
				if err := json.Indent(&buf, []byte(trimmed), "", "  "); err == nil {
					content = buf.String()
					if fileName == "" {
						fileName = "value.json"
					}
				}
			}
			m.VimEditor = createVimEditor(content, m.Width-4, m.Height-10, fileName)
			m.Screen = types.ScreenEditValue
		}
	case "a":
		if m.CurrentKey != nil && m.CurrentKey.Type != types.KeyTypeString && m.CurrentKey.Type != types.KeyTypeJSON &&
			m.CurrentKey.Type != types.KeyTypeProtobuf {
			m.resetAddCollectionInputs()
			m.Screen = types.ScreenAddToCollection
		}
	case "x":
		if m.CurrentKey != nil && m.CurrentKey.Type != types.KeyTypeString && m.CurrentKey.Type != types.KeyTypeJSON &&
			m.CurrentKey.Type != types.KeyTypeHyperLogLog && m.CurrentKey.Type != types.KeyTypeBitmap &&
			m.CurrentKey.Type != types.KeyTypeProtobuf {
			m.SelectedItemIdx = 0
			m.Screen = types.ScreenRemoveFromCollection
		}
	case "R":
		if m.CurrentKey != nil {
			m.Inputs.RenameInput.SetValue(m.CurrentKey.Key)
			m.Inputs.RenameInput.Focus()
			m.Screen = types.ScreenRenameKey
		}
	case "c":
		if m.CurrentKey != nil {
			m.Inputs.CopyInput.SetValue(m.CurrentKey.Key + "_copy")
			m.Inputs.CopyInput.Focus()
			m.Screen = types.ScreenCopyKey
		}
	case "f":
		if m.CurrentKey != nil {
			connID := int64(0)
			if m.CurrentConn != nil {
				connID = m.CurrentConn.ID
			}
			if m.CurrentKey.IsFavorite {
				return m, m.Cmds.RemoveFavorite(connID, m.CurrentKey.Key)
			}
			return m, m.Cmds.AddFavorite(connID, m.CurrentKey.Key, m.CurrentConn.Name)
		}
	case "w":
		if m.CurrentKey != nil {
			if m.WatchActive && m.WatchKey == m.CurrentKey.Key {
				m.WatchActive = false
				m.StatusMsg = "Watch stopped"
			} else {
				m.WatchActive = true
				m.WatchKey = m.CurrentKey.Key
				m.WatchValue = m.CurrentValue.StringValue
				m.WatchLastUpdate = time.Now()
				m.StatusMsg = "Watching key for changes..."
				return m, m.Cmds.WatchKeyTick()
			}
		}
	case "h":
		if m.CurrentKey != nil {
			return m, m.Cmds.LoadValueHistory(m.CurrentKey.Key)
		}
	case "y":
		if m.CurrentKey != nil {
			clip := m.CurrentValue.StringValue
			if m.CurrentKey.Type == types.KeyTypeProtobuf && m.CurrentValue.DecodedValue != "" {
				clip = m.CurrentValue.DecodedValue
			}
			return m, m.Cmds.CopyToClipboard(clip)
		}
	case "J":
		if m.CurrentKey != nil && m.CurrentKey.Type == types.KeyTypeString {
			m.Inputs.JSONPathInput.SetValue("")
			m.Inputs.JSONPathInput.Focus()
			m.Screen = types.ScreenJSONPath
		}
	case "up", "k":
		if m.DetailCursor > 0 {
			m.DetailCursor--
			m.syncDetailScroll()
		} else if m.SelectedItemIdx > 0 {
			m.SelectedItemIdx--
		}
	case "down", "j":
		if m.DetailCursor+1 < m.detailContentLines() {
			m.DetailCursor++
			m.syncDetailScroll()
		}
	case "pgup", "ctrl+u":
		m.DetailCursor -= 10
		if m.DetailCursor < 0 {
			m.DetailCursor = 0
		}
		m.syncDetailScroll()
	case "pgdown", "ctrl+d":
		m.DetailCursor += 10
		if last := m.detailLastLine(); m.DetailCursor > last {
			m.DetailCursor = last
		}
		m.syncDetailScroll()
	case "home", "g":
		m.DetailCursor = 0
		m.syncDetailScroll()
	case "end", "G":
		m.DetailCursor = m.detailLastLine()
		m.syncDetailScroll()
	case "esc", "backspace":
		m.Screen = types.ScreenKeys
		m.CurrentKey = nil
		m.SelectedItemIdx = 0
		m.DetailScroll = 0
		m.DetailCursor = 0
		m.WatchActive = false
	}
	return m, nil
}

// syncDetailScroll keeps DetailScroll aligned so DetailCursor stays visible.
func (m *Model) syncDetailScroll() {
	m.DetailCursor, m.DetailScroll = ensureDetailCursorVisible(
		m.DetailCursor,
		m.DetailScroll,
		m.detailContentLines(),
		detailMaxVisible(m.Height),
	)
}

func (m Model) getCollectionLength() int {
	switch m.CurrentValue.Type {
	case types.KeyTypeList:
		return len(m.CurrentValue.ListValue)
	case types.KeyTypeSet:
		return len(m.CurrentValue.SetValue)
	case types.KeyTypeZSet:
		return len(m.CurrentValue.ZSetValue)
	case types.KeyTypeHash:
		return len(m.CurrentValue.HashValue)
	case types.KeyTypeStream:
		return len(m.CurrentValue.StreamValue)
	case types.KeyTypeGeo:
		return len(m.CurrentValue.GeoValue)
	case types.KeyTypeHyperLogLog, types.KeyTypeJSON, types.KeyTypeBitmap, types.KeyTypeProtobuf:
		return 0
	default:
		return 0
	}
}
