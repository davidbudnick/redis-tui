package ui

import (
	"log/slog"
	"strconv"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

// Key message handlers

func (m Model) handleKeysLoadedMsg(msg types.KeysLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		slog.Error("Failed to load keys", "error", msg.Err)
		m.StatusMsg = "Error: " + msg.Err.Error()
	} else {
		if m.KeyCursor == 0 {
			m.Keys = msg.Keys
			m.SelectedKeyIdx = 0
		} else {
			m.Keys = append(m.Keys, msg.Keys...)
		}
		m.KeyCursor = msg.Cursor
		m.TotalKeys = msg.TotalKeys
		// Load preview for selected key
		if len(m.Keys) > 0 && m.SelectedKeyIdx < len(m.Keys) {
			return m, m.Cmds.LoadKeyPreview(m.Keys[m.SelectedKeyIdx].Key)
		}
	}
	return m, nil
}

func (m Model) handleKeyValueLoadedMsg(msg types.KeyValueLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		slog.Error("Failed to load key value", "key", msg.Key, "error", msg.Err)
		m.StatusMsg = "Error: " + msg.Err.Error()
	} else {
		m.CurrentValue = msg.Value
		m.DetailRendered = buildDetailValueContent(msg.Value)
		m.DetailScroll = 0
		m.DetailCursor = 0
		// On-demand type resolution: fill in type if it was not fetched during scan,
		// or update when GetValue detected a subtype (e.g. string→hyperloglog)
		if m.CurrentKey != nil && msg.Value.Type != "" && m.CurrentKey.Type != msg.Value.Type {
			m.CurrentKey.Type = msg.Value.Type
		}
		m.Screen = types.ScreenKeyDetail
	}
	return m, nil
}

func (m Model) handleKeyPreviewLoadedMsg(msg types.KeyPreviewLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m, nil
	}
	// Only update if the key matches the currently selected key
	if len(m.Keys) > 0 && m.SelectedKeyIdx < len(m.Keys) && m.Keys[m.SelectedKeyIdx].Key == msg.Key {
		m.PreviewKey = msg.Key
		m.PreviewValue = msg.Value
		// Pre-format string/JSON/protobuf previews once — must not run per frame.
		m.PreviewRendered = ""
		switch msg.Value.Type {
		case types.KeyTypeString:
			m.PreviewRendered = formatPossibleJSON(msg.Value.StringValue)
		case types.KeyTypeJSON:
			m.PreviewRendered = formatPossibleJSON(msg.Value.JSONValue)
		case types.KeyTypeProtobuf:
			m.PreviewRendered = formatProtobufValue(msg.Value)
		}
		// On-demand type resolution: fill in type if it was not fetched during scan,
		// or update when GetValue detected a subtype (e.g. string→hyperloglog)
		if msg.Value.Type != "" && m.Keys[m.SelectedKeyIdx].Type != msg.Value.Type {
			m.Keys[m.SelectedKeyIdx].Type = msg.Value.Type
		}
	}
	return m, nil
}

func (m Model) handleKeyDeletedMsg(msg types.KeyDeletedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		for i, k := range m.Keys {
			if k.Key == msg.Key {
				m.Keys = append(m.Keys[:i], m.Keys[i+1:]...)
				break
			}
		}
		if m.SelectedKeyIdx >= len(m.Keys) && m.SelectedKeyIdx > 0 {
			m.SelectedKeyIdx--
		}
		m.CurrentKey = nil
		m.StatusMsg = "Key deleted"
		m.Screen = types.ScreenKeys
	} else {
		slog.Error("Failed to delete key", "key", msg.Key, "error", msg.Err)
		m.StatusMsg = "Error: " + msg.Err.Error()
	}
	return m, nil
}

func (m Model) handleKeySetMsg(msg types.KeySetMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		slog.Error("Failed to set key", "key", msg.Key, "error", msg.Err)
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "Key saved"
	m.Screen = types.ScreenKeys
	m.resetAddKeyInputs()
	return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
}

func (m Model) handleKeyRenamedMsg(msg types.KeyRenamedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
	} else {
		m.StatusMsg = "Key renamed to " + msg.NewKey
		if m.CurrentKey != nil {
			m.CurrentKey.Key = msg.NewKey
			for i, k := range m.Keys {
				if k.Key == msg.OldKey {
					m.Keys[i].Key = msg.NewKey
					break
				}
			}
		}
		m.Screen = types.ScreenKeyDetail
	}
	return m, nil
}

func (m Model) handleKeyCopiedMsg(msg types.KeyCopiedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "Key copied to " + msg.DestKey
	m.Screen = types.ScreenKeyDetail
	m.KeyCursor = 0
	return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
}

// Value message handlers

func (m Model) handleValueEditedMsg(msg types.ValueEditedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "Value updated"
	m.Screen = types.ScreenKeyDetail
	if m.CurrentKey != nil {
		return m, m.Cmds.LoadKeyValue(m.CurrentKey.Key)
	}
	return m, nil
}

func (m Model) handleItemAddedToCollectionMsg(msg types.ItemAddedToCollectionMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "Item added"
	m.Screen = types.ScreenKeyDetail
	m.resetAddCollectionInputs()
	if m.CurrentKey != nil {
		return m, m.Cmds.LoadKeyValue(m.CurrentKey.Key)
	}
	return m, nil
}

func (m Model) handleItemRemovedFromCollectionMsg(msg types.ItemRemovedFromCollectionMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "Item removed"
	if m.CurrentKey != nil {
		return m, m.Cmds.LoadKeyValue(m.CurrentKey.Key)
	}
	return m, nil
}

// TTL message handlers

func (m Model) handleTTLSetMsg(msg types.TTLSetMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "TTL updated"
	if m.CurrentKey != nil {
		m.CurrentKey.TTL = msg.TTL
		for i, k := range m.Keys {
			if k.Key == m.CurrentKey.Key {
				m.Keys[i].TTL = msg.TTL
				break
			}
		}
	}
	m.Screen = types.ScreenKeyDetail
	return m, nil
}

func (m Model) handleBatchTTLSetMsg(msg types.BatchTTLSetMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Batch TTL error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "Set TTL on " + strconv.Itoa(msg.Count) + " keys"
	m.Screen = types.ScreenKeys
	m.KeyCursor = 0
	return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
}
