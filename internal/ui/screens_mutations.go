package ui

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

// addKeyFieldCount returns the number of focusable fields for the current add key type.
// Types that need a third input (zset, hash, stream) have 3 fields; others have 2.
func (m Model) addKeyFieldCount() int {
	switch m.AddKeyType {
	case types.KeyTypeZSet, types.KeyTypeHash, types.KeyTypeStream, types.KeyTypeGeo:
		return 3
	default:
		return 2
	}
}

func (m Model) handleAddKeyScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	fieldCount := m.addKeyFieldCount()
	switch msg.String() {
	case "tab":
		m.AddKeyInputs[m.AddKeyFocusIdx].Blur()
		m.AddKeyFocusIdx = (m.AddKeyFocusIdx + 1) % fieldCount
		m.AddKeyInputs[m.AddKeyFocusIdx].Focus()
	case "shift+tab":
		m.AddKeyInputs[m.AddKeyFocusIdx].Blur()
		m.AddKeyFocusIdx--
		if m.AddKeyFocusIdx < 0 {
			m.AddKeyFocusIdx = fieldCount - 1
		}
		m.AddKeyInputs[m.AddKeyFocusIdx].Focus()
	case "ctrl+t":
		typeOrder := []types.KeyType{
			types.KeyTypeString, types.KeyTypeList, types.KeyTypeSet,
			types.KeyTypeZSet, types.KeyTypeHash, types.KeyTypeStream,
			types.KeyTypeJSON, types.KeyTypeHyperLogLog, types.KeyTypeBitmap, types.KeyTypeGeo,
		}
		for i, t := range typeOrder {
			if t == m.AddKeyType {
				m.AddKeyType = typeOrder[(i+1)%len(typeOrder)]
				// Reset focus if it's beyond the new field count
				newFieldCount := m.addKeyFieldCount()
				if m.AddKeyFocusIdx >= newFieldCount {
					m.AddKeyInputs[m.AddKeyFocusIdx].Blur()
					m.AddKeyFocusIdx = newFieldCount - 1
					m.AddKeyInputs[m.AddKeyFocusIdx].Focus()
				}
				break
			}
		}
	case "enter":
		if m.AddKeyInputs[0].Value() != "" {
			m.Loading = true
			extra := ""
			if fieldCount == 3 {
				extra = m.AddKeyInputs[2].Value()
			}
			return m, m.Cmds.CreateKey(
				m.AddKeyInputs[0].Value(),
				m.AddKeyType,
				m.AddKeyInputs[1].Value(),
				extra,
				0,
			)
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.resetAddKeyInputs()
	default:
		var cmds []tea.Cmd
		for i := range fieldCount {
			var inputCmd tea.Cmd
			m.AddKeyInputs[i], inputCmd = m.AddKeyInputs[i].Update(msg)
			cmds = append(cmds, inputCmd)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) handleConfirmDeleteScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.Loading = true
		switch m.ConfirmType {
		case "connection":
			if conn, ok := m.ConfirmData.(types.Connection); ok {
				return m, m.Cmds.DeleteConnection(conn.ID)
			}
		case "key":
			if key, ok := m.ConfirmData.(types.RedisKey); ok {
				return m, m.Cmds.DeleteKey(key.Key)
			}
		case "flushdb":
			return m, m.Cmds.FlushDB()
		}
	case "n", "N", "esc":
		switch m.ConfirmType {
		case "connection":
			m.Screen = types.ScreenConnections
		case "key":
			if m.CurrentKey != nil {
				m.Screen = types.ScreenKeyDetail
			} else {
				m.Screen = types.ScreenKeys
			}
		case "flushdb":
			m.Screen = types.ScreenKeys
		}
	}
	return m, nil
}

func (m Model) handleTTLEditorScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.CurrentKey != nil {
			ttlSecs, err := strconv.Atoi(m.Inputs.TTLInput.Value())
			if err != nil {
				m.StatusMsg = "Invalid TTL: must be an integer (seconds)"
				return m, nil
			}
			ttl := time.Duration(ttlSecs) * time.Second
			m.Loading = true
			return m, m.Cmds.SetTTL(m.CurrentKey.Key, ttl)
		}
	case "esc":
		m.Screen = types.ScreenKeyDetail
		m.Inputs.TTLInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.TTLInput, inputCmd = m.Inputs.TTLInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleEditValueScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		if m.CurrentKey != nil {
			m.Loading = true
			content := m.VimEditor.Value()
			if m.CurrentKey.Type == types.KeyTypeJSON {
				return m, m.Cmds.EditJSONValue(m.CurrentKey.Key, content)
			}
			return m, m.Cmds.EditStringValue(m.CurrentKey.Key, content)
		}
	case "ctrl+q":
		m.Screen = types.ScreenKeyDetail
		return m, nil
	}

	if m.VimEditor != nil {
		updated, editorCmd := m.VimEditor.Update(msg)
		m.VimEditor = updated
		return m, editorCmd
	}
	return m, nil
}

func (m Model) handleAddToCollectionScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.AddCollectionInput[m.AddCollFocusIdx].Blur()
		m.AddCollFocusIdx = (m.AddCollFocusIdx + 1) % len(m.AddCollectionInput)
		m.AddCollectionInput[m.AddCollFocusIdx].Focus()
	case "shift+tab":
		m.AddCollectionInput[m.AddCollFocusIdx].Blur()
		m.AddCollFocusIdx--
		if m.AddCollFocusIdx < 0 {
			m.AddCollFocusIdx = len(m.AddCollectionInput) - 1
		}
		m.AddCollectionInput[m.AddCollFocusIdx].Focus()
	case "enter":
		if m.CurrentKey != nil && m.AddCollectionInput[0].Value() != "" {
			m.Loading = true
			value := m.AddCollectionInput[0].Value()
			extra := m.AddCollectionInput[1].Value()

			switch m.CurrentKey.Type {
			case types.KeyTypeList:
				return m, m.Cmds.AddToList(m.CurrentKey.Key, value)
			case types.KeyTypeSet:
				return m, m.Cmds.AddToSet(m.CurrentKey.Key, value)
			case types.KeyTypeZSet:
				score := 0.0
				if extra != "" {
					var parseErr error
					score, parseErr = strconv.ParseFloat(extra, 64)
					if parseErr != nil {
						m.StatusMsg = "Invalid score: must be a number"
						m.Loading = false
						return m, nil
					}
				}
				return m, m.Cmds.AddToZSet(m.CurrentKey.Key, score, value)
			case types.KeyTypeHash:
				if extra == "" {
					extra = "value"
				}
				return m, m.Cmds.AddToHash(m.CurrentKey.Key, value, extra)
			case types.KeyTypeStream:
				fields := map[string]any{value: extra}
				return m, m.Cmds.AddToStream(m.CurrentKey.Key, fields)
			case types.KeyTypeHyperLogLog:
				return m, m.Cmds.AddToHLL(m.CurrentKey.Key, value)
			case types.KeyTypeBitmap:
				offset := int64(0)
				if value != "" {
					var parseErr error
					offset, parseErr = strconv.ParseInt(value, 10, 64)
					if parseErr != nil {
						m.StatusMsg = "Invalid offset: must be an integer"
						m.Loading = false
						return m, nil
					}
				}
				return m, m.Cmds.SetBit(m.CurrentKey.Key, offset, 1)
			case types.KeyTypeGeo:
				lon, lat := 0.0, 0.0
				if extra != "" {
					parts := strings.SplitN(extra, ",", 2)
					if len(parts) == 2 {
						var parseErr error
						lon, parseErr = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
						if parseErr != nil {
							m.StatusMsg = "Invalid longitude: must be a number"
							m.Loading = false
							return m, nil
						}
						lat, parseErr = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
						if parseErr != nil {
							m.StatusMsg = "Invalid latitude: must be a number"
							m.Loading = false
							return m, nil
						}
					}
				}
				return m, m.Cmds.AddToGeo(m.CurrentKey.Key, lon, lat, value)
			}
		}
	case "esc":
		m.Screen = types.ScreenKeyDetail
		m.resetAddCollectionInputs()
	default:
		var cmds []tea.Cmd
		for i := range m.AddCollectionInput {
			var inputCmd tea.Cmd
			m.AddCollectionInput[i], inputCmd = m.AddCollectionInput[i].Update(msg)
			cmds = append(cmds, inputCmd)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) handleRemoveFromCollectionScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedItemIdx > 0 {
			m.SelectedItemIdx--
		}
	case "down", "j":
		maxIdx := m.getCollectionLength() - 1
		if m.SelectedItemIdx < maxIdx {
			m.SelectedItemIdx++
		}
	case "enter", "d":
		if m.CurrentKey != nil {
			m.Loading = true
			switch m.CurrentKey.Type {
			case types.KeyTypeList:
				if m.SelectedItemIdx < len(m.CurrentValue.ListValue) {
					return m, m.Cmds.RemoveFromList(m.CurrentKey.Key, m.CurrentValue.ListValue[m.SelectedItemIdx])
				}
			case types.KeyTypeSet:
				if m.SelectedItemIdx < len(m.CurrentValue.SetValue) {
					return m, m.Cmds.RemoveFromSet(m.CurrentKey.Key, m.CurrentValue.SetValue[m.SelectedItemIdx])
				}
			case types.KeyTypeZSet:
				if m.SelectedItemIdx < len(m.CurrentValue.ZSetValue) {
					return m, m.Cmds.RemoveFromZSet(m.CurrentKey.Key, m.CurrentValue.ZSetValue[m.SelectedItemIdx].Member)
				}
			case types.KeyTypeHash:
				keys := make([]string, 0, len(m.CurrentValue.HashValue))
				for k := range m.CurrentValue.HashValue {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				if m.SelectedItemIdx < len(keys) {
					return m, m.Cmds.RemoveFromHash(m.CurrentKey.Key, keys[m.SelectedItemIdx])
				}
			case types.KeyTypeStream:
				if m.SelectedItemIdx < len(m.CurrentValue.StreamValue) {
					return m, m.Cmds.RemoveFromStream(m.CurrentKey.Key, m.CurrentValue.StreamValue[m.SelectedItemIdx].ID)
				}
			case types.KeyTypeGeo:
				if m.SelectedItemIdx < len(m.CurrentValue.GeoValue) {
					return m, m.Cmds.RemoveFromZSet(m.CurrentKey.Key, m.CurrentValue.GeoValue[m.SelectedItemIdx].Name)
				}
			}
		}
	case "esc":
		m.Screen = types.ScreenKeyDetail
	}
	return m, nil
}

func (m Model) handleRenameKeyScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.CurrentKey != nil && m.Inputs.RenameInput.Value() != "" && m.Inputs.RenameInput.Value() != m.CurrentKey.Key {
			m.Loading = true
			return m, m.Cmds.RenameKey(m.CurrentKey.Key, m.Inputs.RenameInput.Value())
		}
	case "esc":
		m.Screen = types.ScreenKeyDetail
		m.Inputs.RenameInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.RenameInput, inputCmd = m.Inputs.RenameInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleCopyKeyScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.CurrentKey != nil && m.Inputs.CopyInput.Value() != "" {
			m.Loading = true
			return m, m.Cmds.CopyKey(m.CurrentKey.Key, m.Inputs.CopyInput.Value(), false)
		}
	case "esc":
		m.Screen = types.ScreenKeyDetail
		m.Inputs.CopyInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.CopyInput, inputCmd = m.Inputs.CopyInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleBulkDeleteScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Inputs.BulkDeleteInput.Value() != "" {
			m.Loading = true
			return m, m.Cmds.BulkDelete(m.Inputs.BulkDeleteInput.Value())
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.BulkDeleteInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.BulkDeleteInput, inputCmd = m.Inputs.BulkDeleteInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleBatchTTLScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.Inputs.BatchTTLInput.Focused() {
			m.Inputs.BatchTTLInput.Blur()
			m.Inputs.BatchTTLPattern.Focus()
		} else {
			m.Inputs.BatchTTLPattern.Blur()
			m.Inputs.BatchTTLInput.Focus()
		}
	case "enter":
		if m.Inputs.BatchTTLInput.Value() != "" && m.Inputs.BatchTTLPattern.Value() != "" {
			ttlSecs, err := strconv.Atoi(m.Inputs.BatchTTLInput.Value())
			if err == nil {
				m.Loading = true
				ttl := time.Duration(ttlSecs) * time.Second
				return m, m.Cmds.BatchSetTTL(m.Inputs.BatchTTLPattern.Value(), ttl)
			}
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.BatchTTLInput.Blur()
		m.Inputs.BatchTTLPattern.Blur()
	default:
		if m.Inputs.BatchTTLInput.Focused() {
			var inputCmd tea.Cmd
			m.Inputs.BatchTTLInput, inputCmd = m.Inputs.BatchTTLInput.Update(msg)
			return m, inputCmd
		}
		var inputCmd tea.Cmd
		m.Inputs.BatchTTLPattern, inputCmd = m.Inputs.BatchTTLPattern.Update(msg)
		return m, inputCmd
	}
	return m, nil
}
