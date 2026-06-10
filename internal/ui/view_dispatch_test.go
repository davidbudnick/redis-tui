package ui

import (
	"strings"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestView(t *testing.T) {
	t.Run("too small", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 10
		m.Height = 5
		out := m.View().Content
		if !strings.Contains(out, "too small") {
			t.Errorf("expected 'too small' in output")
		}
	})
	t.Run("normal path connections", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "normal", m.View().Content)
	})
}

func TestGetStatusBar(t *testing.T) {
	t.Run("loading", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Loading = true
		out := m.getStatusBar()
		if !strings.Contains(out, "Loading") {
			t.Error("expected loading")
		}
	})
	t.Run("error status", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = "Error: something"
		assertNonEmpty(t, "err", m.getStatusBar())
	})
	t.Run("success status", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = "OK"
		assertNonEmpty(t, "ok", m.getStatusBar())
	})
	t.Run("update available", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.UpdateAvailable = "v2.0.0"
		m.UpdateCmd = "brew upgrade"
		out := m.getStatusBar()
		if !strings.Contains(out, "v2.0.0") {
			t.Error("expected update version")
		}
	})
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		out := m.getStatusBar()
		if out != "" {
			t.Errorf("expected empty, got %q", out)
		}
	})
}

func TestGetScreenView_AllScreens(t *testing.T) {
	// Smoke test every screen's dispatch.
	screens := []types.Screen{
		types.ScreenConnections, types.ScreenAddConnection, types.ScreenEditConnection,
		types.ScreenKeys, types.ScreenKeyDetail, types.ScreenAddKey,
		types.ScreenHelp, types.ScreenConfirmDelete, types.ScreenServerInfo,
		types.ScreenTTLEditor, types.ScreenEditValue, types.ScreenAddToCollection,
		types.ScreenRemoveFromCollection, types.ScreenRenameKey, types.ScreenCopyKey,
		types.ScreenPubSub, types.ScreenPublishMessage, types.ScreenSwitchDB,
		types.ScreenSearchValues, types.ScreenExport, types.ScreenImport,
		types.ScreenSlowLog, types.ScreenLuaScript, types.ScreenTestConnection,
		types.ScreenLogs, types.ScreenBulkDelete, types.ScreenBatchTTL,
		types.ScreenFavorites, types.ScreenRecentKeys, types.ScreenTreeView,
		types.ScreenRegexSearch, types.ScreenFuzzySearch, types.ScreenClientList,
		types.ScreenMemoryStats, types.ScreenClusterInfo, types.ScreenCompareKeys,
		types.ScreenTemplates, types.ScreenValueHistory, types.ScreenKeyspaceEvents,
		types.ScreenJSONPath, types.ScreenExpiringKeys, types.ScreenLiveMetrics,
		types.ScreenPubSubChannels, types.ScreenRedisConfig,
	}
	for _, s := range screens {
		t.Run(s.String(), func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.Screen = s
			// Provide safe defaults for screens that dereference CurrentKey or similar.
			m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
			_ = m.getScreenView()
			// Also exercise full View() which wraps getScreenView in layout.
			_ = m.View().Content
		})
	}
	t.Run("unknown screen returns empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = 9999 // out of range
		if out := m.getScreenView(); out != "" {
			t.Errorf("expected empty, got %q", out)
		}
	})
}
