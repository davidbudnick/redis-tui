package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestHandleHelpScreen(t *testing.T) {
	tests := []struct {
		name    string
		hasConn bool
		want    types.Screen
	}{
		{"with conn returns to keys", true, types.ScreenKeys},
		{"no conn returns to connections", false, types.ScreenConnections},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _, _ := newTestModel(t)
			if tt.hasConn {
				m.CurrentConn = &types.Connection{ID: 1}
			}
			result, _ := m.handleHelpScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
			if result.(Model).Screen != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result.(Model).Screen)
			}
		})
	}
	t.Run("enter and ? also exit", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleHelpScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		_, _ = m.handleHelpScreen(keyMsg('?'))
	})
	t.Run("other key no-op", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleHelpScreen(keyMsg('x'))
	})
}

func TestHandleServerInfoScreen(t *testing.T) {
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleServerInfoScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", result.(Model).Screen)
		}
	})
	t.Run("enter exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleServerInfoScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleServerInfoScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("other no-op", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleServerInfoScreen(keyMsg('x'))
	})
}

func TestHandlePubSubScreen(t *testing.T) {
	t.Run("tab advances focus", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handlePubSubScreen(tea.KeyPressMsg{Code: tea.KeyTab})
		if result.(Model).PubSubFocusIdx != 1 {
			t.Errorf("expected 1, got %d", result.(Model).PubSubFocusIdx)
		}
	})
	t.Run("enter publishes when valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.PubSubInput[0].SetValue("ch")
		m.PubSubInput[1].SetValue("msg")
		_, cmd := m.handlePubSubScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected publish cmd")
		}
	})
	t.Run("enter no-op empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handlePubSubScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handlePubSubScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).Screen != types.ScreenPubSubChannels {
			t.Errorf("expected ScreenPubSubChannels, got %v", result.(Model).Screen)
		}
	})
	t.Run("default updates inputs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handlePubSubScreen(keyMsg('a'))
	})
}

func TestHandlePublishMessageScreen(t *testing.T) {
	m, _, _ := newTestModel(t)
	_, _ = m.handlePublishMessageScreen(tea.KeyPressMsg{Code: tea.KeyTab})
}

func TestHandleSwitchDBScreen(t *testing.T) {
	t.Run("enter valid db", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.DBSwitchInput.SetValue("5")
		_, cmd := m.handleSwitchDBScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected switch cmd")
		}
	})
	t.Run("enter invalid db", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.DBSwitchInput.SetValue("not-a-number")
		result, _ := m.handleSwitchDBScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if result.(Model).StatusMsg == "" {
			t.Error("expected error status")
		}
	})
	t.Run("enter out of range", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.DBSwitchInput.SetValue("20")
		result, _ := m.handleSwitchDBScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if result.(Model).StatusMsg == "" {
			t.Error("expected error status")
		}
	})
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleSwitchDBScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", result.(Model).Screen)
		}
	})
	t.Run("default input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleSwitchDBScreen(keyMsg('5'))
	})
}

func TestHandleExportScreen(t *testing.T) {
	t.Run("enter exports with pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.ExportInput.SetValue("out.json")
		m.KeyPattern = "user:*"
		_, cmd := m.handleExportScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected export cmd")
		}
	})
	t.Run("enter exports empty pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.ExportInput.SetValue("out.json")
		_, cmd := m.handleExportScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected export cmd")
		}
	})
	t.Run("enter empty filename", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleExportScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleExportScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", result.(Model).Screen)
		}
	})
	t.Run("default input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleExportScreen(keyMsg('a'))
	})
}

func TestHandleImportScreen(t *testing.T) {
	t.Run("enter imports", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.ImportInput.SetValue("in.json")
		_, cmd := m.handleImportScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected import cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleImportScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleImportScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
	t.Run("default input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleImportScreen(keyMsg('a'))
	})
}

func TestHandleSlowLogScreen(t *testing.T) {
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleSlowLogScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
	t.Run("enter exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleSlowLogScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleSlowLogScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
}

func TestHandleLuaScriptScreen(t *testing.T) {
	t.Run("enter runs script", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.LuaScriptInput.SetValue("return 1")
		_, cmd := m.handleLuaScriptScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected eval cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleLuaScriptScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleLuaScriptScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
	t.Run("default input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleLuaScriptScreen(keyMsg('a'))
	})
}

func TestHandleLogsScreen(t *testing.T) {
	t.Run("showing detail esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ShowingLogDetail = true
		result, _ := m.handleLogsScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).ShowingLogDetail {
			t.Error("expected ShowingLogDetail=false")
		}
	})
	t.Run("showing detail enter", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ShowingLogDetail = true
		_, _ = m.handleLogsScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleLogsScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", result.(Model).Screen)
		}
	})
	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LogCursor = 1
		result, _ := m.handleLogsScreen(keyMsg('k'))
		if result.(Model).LogCursor != 0 {
			t.Errorf("expected 0, got %d", result.(Model).LogCursor)
		}
	})
	t.Run("down with no logs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleLogsScreen(keyMsg('j'))
	})
	t.Run("enter with logs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Logs = types.NewLogWriter()
		if _, err := m.Logs.Write([]byte("hello\n")); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		result, _ := m.handleLogsScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if !result.(Model).ShowingLogDetail {
			t.Error("expected ShowingLogDetail=true")
		}
	})
	t.Run("enter no logs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleLogsScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("g home", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LogCursor = 3
		result, _ := m.handleLogsScreen(keyMsg('g'))
		if result.(Model).LogCursor != 0 {
			t.Errorf("expected 0, got %d", result.(Model).LogCursor)
		}
	})
	t.Run("G end with logs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Logs = types.NewLogWriter()
		if _, err := m.Logs.Write([]byte("a\nb\n")); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		_, _ = m.handleLogsScreen(keyMsg('G'))
	})
	t.Run("G end empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleLogsScreen(keyMsg('G'))
	})
}

func TestHandleClientListScreen(t *testing.T) {
	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SelectedClientIdx = 1
		_, _ = m.handleClientListScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ClientList = []types.ClientInfo{{ID: 1}, {ID: 2}}
		_, _ = m.handleClientListScreen(keyMsg('j'))
	})
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleClientListScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleClientListScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleMemoryStatsScreen(t *testing.T) {
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleMemoryStatsScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleMemoryStatsScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleClusterInfoScreen(t *testing.T) {
	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SelectedNodeIdx = 1
		_, _ = m.handleClusterInfoScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ClusterNodes = []types.ClusterNode{{ID: "n1"}, {ID: "n2"}}
		_, _ = m.handleClusterInfoScreen(keyMsg('j'))
	})
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleClusterInfoScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleClusterInfoScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandlePubSubChannelsScreen(t *testing.T) {
	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SelectedChannelIdx = 1
		_, _ = m.handlePubSubChannelsScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.PubSubChannels = []types.PubSubChannel{{Name: "a"}, {Name: "b"}}
		_, _ = m.handlePubSubChannelsScreen(keyMsg('j'))
	})
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handlePubSubChannelsScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("p publish", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handlePubSubChannelsScreen(keyMsg('p'))
		if result.(Model).Screen != types.ScreenPubSub {
			t.Errorf("expected ScreenPubSub, got %v", result.(Model).Screen)
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handlePubSubChannelsScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleRedisConfigScreen(t *testing.T) {
	t.Run("editing enter sets config", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.EditingConfigParam = "maxmemory"
		m.Inputs.ConfigEditInput.SetValue("100mb")
		_, cmd := m.handleRedisConfigScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("editing esc cancels", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.EditingConfigParam = "maxmemory"
		result, _ := m.handleRedisConfigScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).EditingConfigParam != "" {
			t.Error("expected editing cleared")
		}
	})
	t.Run("editing default input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.EditingConfigParam = "maxmemory"
		_, _ = m.handleRedisConfigScreen(keyMsg('a'))
	})
	t.Run("up down navigation", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RedisConfigParams = []types.RedisConfigParam{{Name: "a"}, {Name: "b"}}
		m.SelectedConfigIdx = 1
		result, _ := m.handleRedisConfigScreen(keyMsg('k'))
		if result.(Model).SelectedConfigIdx != 0 {
			t.Errorf("expected 0, got %d", result.(Model).SelectedConfigIdx)
		}
		_, _ = m.handleRedisConfigScreen(keyMsg('j'))
	})
	t.Run("e edit enter mode", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RedisConfigParams = []types.RedisConfigParam{{Name: "a", Value: "1"}}
		result, _ := m.handleRedisConfigScreen(keyMsg('e'))
		if result.(Model).EditingConfigParam != "a" {
			t.Errorf("expected editing a, got %q", result.(Model).EditingConfigParam)
		}
	})
	t.Run("e with empty params", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRedisConfigScreen(keyMsg('e'))
	})
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleRedisConfigScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleRedisConfigScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", result.(Model).Screen)
		}
	})
}

func TestHandleLiveMetricsScreen(t *testing.T) {
	t.Run("c clears", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LiveMetrics = &types.LiveMetrics{DataPoints: []types.LiveMetricsData{{}}}
		result, _ := m.handleLiveMetricsScreen(keyMsg('c'))
		if len(result.(Model).LiveMetrics.DataPoints) != 0 {
			t.Error("expected data points cleared")
		}
	})
	t.Run("c clears nil", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleLiveMetricsScreen(keyMsg('c'))
	})
	t.Run("q exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LiveMetricsActive = true
		result, _ := m.handleLiveMetricsScreen(keyMsg('q'))
		if result.(Model).LiveMetricsActive {
			t.Error("expected inactive")
		}
	})
	t.Run("esc exits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleLiveMetricsScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}
