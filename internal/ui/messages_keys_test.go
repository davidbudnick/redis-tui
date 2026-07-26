package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestHandleKeysLoadedMsg(t *testing.T) {
	t.Run("success cursor zero replaces keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "old"}}
		m.KeyCursor = 0
		msg := types.KeysLoadedMsg{Keys: []types.RedisKey{{Key: "a"}, {Key: "b"}}, Cursor: 100, TotalKeys: 2}
		result, cmd := m.handleKeysLoadedMsg(msg)
		model := result.(Model)
		if len(model.Keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(model.Keys))
		}
		if model.SelectedKeyIdx != 0 {
			t.Errorf("expected SelectedKeyIdx=0, got %d", model.SelectedKeyIdx)
		}
		if cmd == nil {
			t.Error("expected preview cmd")
		}
	})
	t.Run("success cursor non-zero appends", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "existing"}}
		m.KeyCursor = 50
		msg := types.KeysLoadedMsg{Keys: []types.RedisKey{{Key: "new"}}, Cursor: 0}
		result, _ := m.handleKeysLoadedMsg(msg)
		model := result.(Model)
		if len(model.Keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(model.Keys))
		}
	})
	t.Run("success empty keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeysLoadedMsg{Keys: []types.RedisKey{}}
		_, cmd := m.handleKeysLoadedMsg(msg)
		if cmd != nil {
			t.Error("expected nil cmd for empty keys")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeysLoadedMsg{Err: errors.New("boom")}
		result, _ := m.handleKeysLoadedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleKeyValueLoadedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
		msg := types.KeyValueLoadedMsg{Key: "foo", Value: types.RedisValue{Type: types.KeyTypeString, StringValue: "bar"}}
		result, _ := m.handleKeyValueLoadedMsg(msg)
		model := result.(Model)
		if model.Screen != types.ScreenKeyDetail {
			t.Errorf("expected ScreenKeyDetail, got %v", model.Screen)
		}
		if model.CurrentValue.StringValue != "bar" {
			t.Errorf("expected value set, got %q", model.CurrentValue.StringValue)
		}
	})
	t.Run("type resolution updates current key type", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
		msg := types.KeyValueLoadedMsg{Key: "foo", Value: types.RedisValue{Type: types.KeyTypeHyperLogLog}}
		result, _ := m.handleKeyValueLoadedMsg(msg)
		model := result.(Model)
		if model.CurrentKey.Type != types.KeyTypeHyperLogLog {
			t.Errorf("expected type updated, got %v", model.CurrentKey.Type)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeyValueLoadedMsg{Err: errors.New("boom")}
		result, _ := m.handleKeyValueLoadedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleKeyPreviewLoadedMsg(t *testing.T) {
	t.Run("success matching key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo", Type: types.KeyTypeString}}
		m.SelectedKeyIdx = 0
		msg := types.KeyPreviewLoadedMsg{Key: "foo", Value: types.RedisValue{Type: types.KeyTypeString, StringValue: "v"}}
		result, _ := m.handleKeyPreviewLoadedMsg(msg)
		model := result.(Model)
		if model.PreviewKey != "foo" {
			t.Errorf("expected PreviewKey=foo, got %q", model.PreviewKey)
		}
	})
	t.Run("type resolution for preview", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo", Type: types.KeyTypeString}}
		m.SelectedKeyIdx = 0
		msg := types.KeyPreviewLoadedMsg{Key: "foo", Value: types.RedisValue{Type: types.KeyTypeHyperLogLog}}
		result, _ := m.handleKeyPreviewLoadedMsg(msg)
		model := result.(Model)
		if model.Keys[0].Type != types.KeyTypeHyperLogLog {
			t.Errorf("expected type updated, got %v", model.Keys[0].Type)
		}
	})
	t.Run("error sets status for selected key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo"}}
		m.SelectedKeyIdx = 0
		m.PreviewKey = "foo"
		m.PreviewValue = types.RedisValue{StringValue: "stale"}
		m.PreviewRendered = "stale"
		msg := types.KeyPreviewLoadedMsg{Key: "foo", Err: errors.New("boom")}
		result, _ := m.handleKeyPreviewLoadedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
		if model.PreviewKey != "" {
			t.Errorf("expected cleared PreviewKey, got %q", model.PreviewKey)
		}
		if model.PreviewRendered != "" {
			t.Errorf("expected cleared PreviewRendered, got %q", model.PreviewRendered)
		}
	})
	t.Run("error ignored for unselected key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo"}}
		m.SelectedKeyIdx = 0
		m.PreviewKey = "foo"
		msg := types.KeyPreviewLoadedMsg{Key: "bar", Err: errors.New("boom")}
		result, _ := m.handleKeyPreviewLoadedMsg(msg)
		model := result.(Model)
		if model.StatusMsg != "" {
			t.Errorf("expected empty status, got %q", model.StatusMsg)
		}
		if model.PreviewKey != "foo" {
			t.Errorf("expected preview retained, got %q", model.PreviewKey)
		}
	})
	t.Run("key mismatch ignored", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo"}}
		m.SelectedKeyIdx = 0
		msg := types.KeyPreviewLoadedMsg{Key: "bar", Value: types.RedisValue{StringValue: "v"}}
		result, _ := m.handleKeyPreviewLoadedMsg(msg)
		model := result.(Model)
		if model.PreviewKey != "" {
			t.Errorf("expected no preview update, got %q", model.PreviewKey)
		}
	})
}

func TestHandleKeyDeletedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a"}, {Key: "b"}, {Key: "c"}}
		m.SelectedKeyIdx = 2
		m.CurrentKey = &types.RedisKey{Key: "c"}
		msg := types.KeyDeletedMsg{Key: "c"}
		result, _ := m.handleKeyDeletedMsg(msg)
		model := result.(Model)
		if len(model.Keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(model.Keys))
		}
		if model.SelectedKeyIdx != 1 {
			t.Errorf("expected SelectedKeyIdx=1, got %d", model.SelectedKeyIdx)
		}
		if model.CurrentKey != nil {
			t.Error("expected CurrentKey cleared")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeyDeletedMsg{Key: "c", Err: errors.New("boom")}
		result, _ := m.handleKeyDeletedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleKeySetMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenAddKey
		msg := types.KeySetMsg{Key: "foo"}
		result, cmd := m.handleKeySetMsg(msg)
		model := result.(Model)
		if model.Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", model.Screen)
		}
		if cmd == nil {
			t.Error("expected load keys cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeySetMsg{Err: errors.New("boom")}
		result, cmd := m.handleKeySetMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
		if cmd != nil {
			t.Error("expected nil cmd on error")
		}
	})
}

func TestHandleKeyRenamedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "old"}}
		m.CurrentKey = &types.RedisKey{Key: "old"}
		msg := types.KeyRenamedMsg{OldKey: "old", NewKey: "new"}
		result, _ := m.handleKeyRenamedMsg(msg)
		model := result.(Model)
		if model.CurrentKey.Key != "new" {
			t.Errorf("expected renamed, got %q", model.CurrentKey.Key)
		}
		if model.Keys[0].Key != "new" {
			t.Errorf("expected keys renamed, got %q", model.Keys[0].Key)
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeyRenamedMsg{Err: errors.New("boom")}
		result, _ := m.handleKeyRenamedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleKeyCopiedMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeyCopiedMsg{SourceKey: "a", DestKey: "b"}
		result, cmd := m.handleKeyCopiedMsg(msg)
		model := result.(Model)
		if !strings.Contains(model.StatusMsg, "b") {
			t.Errorf("unexpected status: %q", model.StatusMsg)
		}
		if cmd == nil {
			t.Error("expected load keys cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.KeyCopiedMsg{Err: errors.New("boom")}
		result, _ := m.handleKeyCopiedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleValueEditedMsg(t *testing.T) {
	t.Run("success with current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		msg := types.ValueEditedMsg{Key: "foo"}
		_, cmd := m.handleValueEditedMsg(msg)
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("success without current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ValueEditedMsg{}
		_, cmd := m.handleValueEditedMsg(msg)
		if cmd != nil {
			t.Error("expected nil cmd without current key")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ValueEditedMsg{Err: errors.New("boom")}
		result, _ := m.handleValueEditedMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleItemAddedToCollectionMsg(t *testing.T) {
	t.Run("success with current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		msg := types.ItemAddedToCollectionMsg{Key: "foo"}
		_, cmd := m.handleItemAddedToCollectionMsg(msg)
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("success without current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ItemAddedToCollectionMsg{}
		_, cmd := m.handleItemAddedToCollectionMsg(msg)
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ItemAddedToCollectionMsg{Err: errors.New("boom")}
		result, _ := m.handleItemAddedToCollectionMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleItemRemovedFromCollectionMsg(t *testing.T) {
	t.Run("success with current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		msg := types.ItemRemovedFromCollectionMsg{Key: "foo"}
		_, cmd := m.handleItemRemovedFromCollectionMsg(msg)
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("success without current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ItemRemovedFromCollectionMsg{}
		_, cmd := m.handleItemRemovedFromCollectionMsg(msg)
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.ItemRemovedFromCollectionMsg{Err: errors.New("boom")}
		result, _ := m.handleItemRemovedFromCollectionMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleTTLSetMsg(t *testing.T) {
	t.Run("success with current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.Keys = []types.RedisKey{{Key: "foo"}, {Key: "bar"}}
		msg := types.TTLSetMsg{Key: "foo", TTL: 30 * time.Second}
		result, _ := m.handleTTLSetMsg(msg)
		model := result.(Model)
		if model.CurrentKey.TTL != 30*time.Second {
			t.Errorf("expected TTL updated, got %v", model.CurrentKey.TTL)
		}
		if model.Keys[0].TTL != 30*time.Second {
			t.Errorf("expected list TTL updated, got %v", model.Keys[0].TTL)
		}
	})
	t.Run("success without current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.TTLSetMsg{Key: "foo"}
		_, _ = m.handleTTLSetMsg(msg)
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.TTLSetMsg{Err: errors.New("boom")}
		result, _ := m.handleTTLSetMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleBatchTTLSetMsg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.BatchTTLSetMsg{Count: 5}
		_, cmd := m.handleBatchTTLSetMsg(msg)
		if cmd == nil {
			t.Error("expected load keys cmd")
		}
	})
	t.Run("error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		msg := types.BatchTTLSetMsg{Err: errors.New("boom")}
		result, _ := m.handleBatchTTLSetMsg(msg)
		model := result.(Model)
		if !strings.HasPrefix(model.StatusMsg, "Batch TTL error:") {
			t.Errorf("expected error status, got %q", model.StatusMsg)
		}
	})
}

func TestHandleKeyPreviewLoadedMsg_CachedTypes(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Keys = []types.RedisKey{{Key: "s", Type: types.KeyTypeString}}
	m.SelectedKeyIdx = 0
	msg := types.KeyPreviewLoadedMsg{Key: "s", Value: types.RedisValue{Type: types.KeyTypeString, StringValue: `{"a":1}`}}
	got, _ := m.handleKeyPreviewLoadedMsg(msg)
	model := got.(Model)
	if model.PreviewRendered == "" {
		t.Error("expected PreviewRendered for string JSON")
	}
	msg = types.KeyPreviewLoadedMsg{Key: "s", Value: types.RedisValue{Type: types.KeyTypeJSON, JSONValue: `{"b":2}`}}
	got, _ = model.handleKeyPreviewLoadedMsg(msg)
	model = got.(Model)
	if model.PreviewRendered == "" {
		t.Error("expected PreviewRendered for JSON")
	}
	msg = types.KeyPreviewLoadedMsg{Key: "s", Value: types.RedisValue{Type: types.KeyTypeProtobuf, DecodedValue: "1: 1\n", DecodedFormat: "protobuf"}}
	got, _ = model.handleKeyPreviewLoadedMsg(msg)
	model = got.(Model)
	if model.PreviewRendered == "" || !strings.Contains(model.PreviewRendered, "protobuf") {
		t.Errorf("expected protobuf preview render, got %q", model.PreviewRendered)
	}
}
