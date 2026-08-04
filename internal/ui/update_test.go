package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestUpdate_WindowSizeMsg(t *testing.T) {
	t.Run("sets dimensions", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		model := result.(Model)
		if model.Width != 100 || model.Height != 50 {
			t.Errorf("expected 100x50, got %dx%d", model.Width, model.Height)
		}
	})
	t.Run("resizes vim editor when editing", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenEditValue
		m.VimEditor = createVimEditor("x", 80, 24, "")
		_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	})
}

func TestUpdate_KeyMsg(t *testing.T) {
	m, _, _ := newTestModel(t)
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
}

func TestUpdate_SearchDebounceMsg(t *testing.T) {
	t.Run("matching seq reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SearchSeq = 5
		m.Inputs.PatternInput.SetValue("foo")
		_, cmd := m.Update(types.SearchDebounceMsg{Seq: 5})
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("wildcard pattern preserved", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SearchSeq = 1
		m.Inputs.PatternInput.SetValue("user:*")
		result, _ := m.Update(types.SearchDebounceMsg{Seq: 1})
		if result.(Model).KeyPattern != "user:*" {
			t.Errorf("expected user:*, got %q", result.(Model).KeyPattern)
		}
	})
	t.Run("plain pattern wrapped", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SearchSeq = 1
		m.Inputs.PatternInput.SetValue("user")
		result, _ := m.Update(types.SearchDebounceMsg{Seq: 1})
		if result.(Model).KeyPattern != "*user*" {
			t.Errorf("expected *user*, got %q", result.(Model).KeyPattern)
		}
	})
	t.Run("seq mismatch ignored", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SearchSeq = 5
		_, cmd := m.Update(types.SearchDebounceMsg{Seq: 3})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
}

func TestUpdate_PreviewDebounceMsg(t *testing.T) {
	t.Run("matching seq and key loads preview", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo"}, {Key: "bar"}}
		m.SelectedKeyIdx = 1
		m.PreviewSeq = 3
		_, cmd := m.Update(types.PreviewDebounceMsg{Seq: 3, Key: "bar"})
		if cmd == nil {
			t.Fatal("expected LoadKeyPreview cmd")
		}
		msg := cmd()
		loaded, ok := msg.(types.KeyPreviewLoadedMsg)
		if !ok {
			t.Fatalf("expected KeyPreviewLoadedMsg, got %T", msg)
		}
		if loaded.Key != "bar" {
			t.Errorf("expected key bar, got %q", loaded.Key)
		}
	})
	t.Run("seq mismatch ignored", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo"}}
		m.SelectedKeyIdx = 0
		m.PreviewSeq = 5
		_, cmd := m.Update(types.PreviewDebounceMsg{Seq: 3, Key: "foo"})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("key mismatch ignored", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo"}}
		m.SelectedKeyIdx = 0
		m.PreviewSeq = 1
		_, cmd := m.Update(types.PreviewDebounceMsg{Seq: 1, Key: "bar"})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("empty keys ignored", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.PreviewSeq = 1
		_, cmd := m.Update(types.PreviewDebounceMsg{Seq: 1, Key: "foo"})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
}

func TestUpdate_TickMsg(t *testing.T) {
	m, _, _ := newTestModel(t)
	_, cmd := m.Update(types.TickMsg{})
	if cmd == nil {
		t.Error("expected tick cmd")
	}
}

func TestUpdate_AllMessageDispatchBranches(t *testing.T) {
	// Dispatch smoke test — each case should not panic.
	cases := []tea.Msg{
		types.ConnectionsLoadedMsg{},
		types.ConnectionAddedMsg{},
		types.ConnectionUpdatedMsg{},
		types.ConnectionDeletedMsg{},
		types.AutoConnectMsg{},
		types.ConnectedMsg{},
		types.DisconnectedMsg{},
		types.ConnectionTestMsg{},
		types.GroupsLoadedMsg{},
		types.KeysLoadedMsg{},
		types.KeyValueLoadedMsg{},
		types.KeyPreviewLoadedMsg{},
		types.PreviewDebounceMsg{},
		types.KeyDeletedMsg{},
		types.KeySetMsg{},
		types.KeyRenamedMsg{},
		types.KeyCopiedMsg{},
		types.ValueEditedMsg{},
		types.ItemAddedToCollectionMsg{},
		types.ItemRemovedFromCollectionMsg{},
		types.TTLSetMsg{},
		types.BatchTTLSetMsg{},
		types.ServerInfoLoadedMsg{},
		types.DBSwitchedMsg{},
		types.FlushDBMsg{},
		types.SlowLogLoadedMsg{},
		types.ClientListLoadedMsg{},
		types.MemoryStatsLoadedMsg{},
		types.ClusterInfoLoadedMsg{},
		types.ClusterNodesLoadedMsg{},
		types.MemoryUsageMsg{},
		types.LuaScriptResultMsg{},
		types.PublishResultMsg{},
		types.PubSubChannelsLoadedMsg{},
		types.KeyspaceEventMsg{},
		types.ExportCompleteMsg{},
		types.ImportCompleteMsg{},
		types.BulkDeleteMsg{},
		types.FavoritesLoadedMsg{},
		types.FavoriteAddedMsg{},
		types.FavoriteRemovedMsg{},
		types.RecentKeysLoadedMsg{},
		types.TemplatesLoadedMsg{},
		types.ValueHistoryMsg{},
		types.RegexSearchResultMsg{},
		types.FuzzySearchResultMsg{},
		types.CompareKeysResultMsg{},
		types.JSONPathResultMsg{},
		types.ConfigLoadedMsg{Params: map[string]string{}},
		types.ConfigSetMsg{},
		types.LiveMetricsMsg{},
		types.LiveMetricsTickMsg{},
		types.ClipboardCopiedMsg{},
		types.UpdateAvailableMsg{LatestVersion: "v2.0.0", UpgradeCmd: "brew upgrade"},
		types.EditorQuitMsg{},
	}
	for i, msg := range cases {
		m, _, _ := newTestModel(t)
		defer func(i int, msg tea.Msg) {
			if r := recover(); r != nil {
				t.Errorf("case %d %T panicked: %v", i, msg, r)
			}
		}(i, msg)
		_, _ = m.Update(msg)
	}
}

func TestUpdate_EditorSaveMsg(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeJSON}
		_, cmd := m.Update(types.EditorSaveMsg{Content: `{"a":1}`})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("string", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
		_, cmd := m.Update(types.EditorSaveMsg{Content: "value"})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("nil current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.Update(types.EditorSaveMsg{Content: "x"})
	})
}

func TestUpdate_DefaultVimteaFallback(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenEditValue
	m.VimEditor = createVimEditor("content", 80, 24, "")
	// Send an unknown message type — should delegate to vim editor
	type customMsg struct{}
	_, _ = m.Update(customMsg{})
}

func TestUpdate_UpdateAvailableNoError(t *testing.T) {
	m, _, _ := newTestModel(t)
	result, _ := m.Update(types.UpdateAvailableMsg{LatestVersion: "v2.0.0", UpgradeCmd: "cmd"})
	if result.(Model).UpdateAvailable != "v2.0.0" {
		t.Errorf("expected v2.0.0, got %q", result.(Model).UpdateAvailable)
	}
}

func TestHandleTickMsg(t *testing.T) {
	t.Run("first tick initializes", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleTickMsg()
		if cmd == nil {
			t.Error("expected tick cmd")
		}
	})
	t.Run("decrements ttls", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LastTickTime = time.Now().Add(-2 * time.Second)
		m.CurrentKey = &types.RedisKey{Key: "foo", TTL: 10 * time.Second}
		m.Keys = []types.RedisKey{
			{Key: "foo", TTL: 10 * time.Second},
			{Key: "bar", TTL: 20 * time.Second},
		}
		result, _ := m.handleTickMsg()
		model := result.(Model)
		if model.CurrentKey.TTL >= 10*time.Second {
			t.Errorf("expected decremented, got %v", model.CurrentKey.TTL)
		}
	})
	t.Run("expires keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LastTickTime = time.Now().Add(-1 * time.Hour)
		m.Keys = []types.RedisKey{
			{Key: "expired", TTL: 1 * time.Second},
			{Key: "alive", TTL: 2 * time.Hour},
		}
		m.SelectedKeyIdx = 1
		result, _ := m.handleTickMsg()
		model := result.(Model)
		if len(model.Keys) != 1 {
			t.Errorf("expected 1 key remaining, got %d", len(model.Keys))
		}
	})
	t.Run("current key expires", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LastTickTime = time.Now().Add(-1 * time.Hour)
		m.CurrentKey = &types.RedisKey{Key: "foo", TTL: 1 * time.Second}
		m.Keys = []types.RedisKey{{Key: "foo", TTL: 1 * time.Second}}
		m.Screen = types.ScreenKeyDetail
		result, _ := m.handleTickMsg()
		model := result.(Model)
		if model.CurrentKey != nil {
			t.Error("expected CurrentKey cleared")
		}
		if model.StatusMsg != "Key expired" {
			t.Errorf("expected 'Key expired', got %q", model.StatusMsg)
		}
	})
	t.Run("selected idx clamp after expiry", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LastTickTime = time.Now().Add(-1 * time.Hour)
		m.Keys = []types.RedisKey{
			{Key: "expired", TTL: 1 * time.Second},
		}
		m.SelectedKeyIdx = 5
		_, _ = m.handleTickMsg()
	})
	t.Run("watch active", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.WatchActive = true
		_, cmd := m.handleTickMsg()
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("live metrics active", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LiveMetricsActive = true
		_, cmd := m.handleTickMsg()
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
}

func TestTickCmd(t *testing.T) {
	// tickCmd returns a tea.Tick. Invoking the returned cmd blocks for 1s — skip closure invocation
	// and just ensure construction works. The Tick closure is covered below via direct invocation.
	cmd := tickCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
}

func TestTickCmd_InnerClosure(t *testing.T) {
	// Invoke the outer tea.Tick cmd. This blocks for 1 second then calls the inner closure,
	// which is what we need to cover. The test runs for ~1s.
	cmd := tickCmd()
	msg := cmd()
	if _, ok := msg.(types.TickMsg); !ok {
		t.Errorf("expected TickMsg, got %T", msg)
	}
}

func TestHandleKeyPress_GlobalKeys(t *testing.T) {
	t.Run("ctrl+c quits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeyPress(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Error("expected quit cmd")
		}
	})
	t.Run("q on connections quits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenConnections
		_, cmd := m.handleKeyPress(keyMsg('q'))
		if cmd == nil {
			t.Error("expected quit cmd")
		}
	})
	t.Run("q on keys quits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		_, cmd := m.handleKeyPress(keyMsg('q'))
		if cmd == nil {
			t.Error("expected quit cmd")
		}
	})
	t.Run("q elsewhere no quit", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenServerInfo
		_, _ = m.handleKeyPress(keyMsg('q'))
	})
	t.Run("? opens help", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		result, _ := m.handleKeyPress(keyMsg('?'))
		if result.(Model).Screen != types.ScreenHelp {
			t.Errorf("expected ScreenHelp, got %v", result.(Model).Screen)
		}
	})
	t.Run("? excluded on add connection", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenAddConnection
		result, _ := m.handleKeyPress(keyMsg('?'))
		if result.(Model).Screen != types.ScreenAddConnection {
			t.Error("expected screen unchanged")
		}
	})
	t.Run("q typed into key filter does not quit", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		m.Inputs.PatternInput.Focus()
		m.Inputs.PatternInput.SetValue("sideki")
		result, _ := m.handleKeyPress(keyMsg('q'))
		if got := result.(Model).Inputs.PatternInput.Value(); got != "sidekiq" {
			t.Errorf("expected sidekiq in filter, got %q", got)
		}
	})
	t.Run("? typed into key filter does not open help", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		m.Inputs.PatternInput.Focus()
		result, _ := m.handleKeyPress(keyMsg('?'))
		updated := result.(Model)
		if updated.Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", updated.Screen)
		}
		if got := updated.Inputs.PatternInput.Value(); got != "?" {
			t.Errorf("expected ? in filter, got %q", got)
		}
	})
}

func TestTextEntryActive(t *testing.T) {
	t.Run("vim editor active", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenEditValue
		m.VimEditor = createVimEditor("value", 80, 24, "key")
		if !m.textEntryActive() {
			t.Error("expected text entry active with vim editor")
		}
	})
	t.Run("focused input active", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		m.Inputs.PatternInput.Focus()
		if !m.textEntryActive() {
			t.Error("expected text entry active with focused filter")
		}
	})
	t.Run("no input inactive", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		m.Inputs.PatternInput.Blur()
		if m.textEntryActive() {
			t.Error("expected text entry inactive")
		}
	})
}

func TestHandleKeyPress_AllScreens(t *testing.T) {
	// Route every screen constant through handleKeyPress to cover every dispatch branch.
	screens := []types.Screen{
		types.ScreenConnections, types.ScreenAddConnection, types.ScreenEditConnection,
		types.ScreenKeys, types.ScreenKeyDetail, types.ScreenAddKey,
		types.ScreenConfirmDelete, types.ScreenTTLEditor, types.ScreenHelp,
		types.ScreenServerInfo, types.ScreenEditValue, types.ScreenAddToCollection,
		types.ScreenRemoveFromCollection, types.ScreenRenameKey, types.ScreenCopyKey,
		types.ScreenPubSub, types.ScreenPublishMessage, types.ScreenSwitchDB,
		types.ScreenSearchValues, types.ScreenExport, types.ScreenImport,
		types.ScreenSlowLog, types.ScreenLuaScript, types.ScreenTestConnection,
		types.ScreenLogs, types.ScreenBulkDelete, types.ScreenBatchTTL,
		types.ScreenFavorites, types.ScreenRecentKeys, types.ScreenTreeView,
		types.ScreenRegexSearch, types.ScreenFuzzySearch, types.ScreenClientList,
		types.ScreenMemoryStats, types.ScreenClusterInfo, types.ScreenCompareKeys,
		types.ScreenTemplates, types.ScreenValueHistory, types.ScreenKeyspaceEvents,
		types.ScreenJSONPath, types.ScreenWatchKey, types.ScreenConnectionGroups,
		types.ScreenExpiringKeys, types.ScreenLiveMetrics, types.ScreenPubSubChannels,
		types.ScreenRedisConfig,
	}
	for _, s := range screens {
		t.Run(s.String(), func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.Screen = s
			_, _ = m.handleKeyPress(tea.KeyPressMsg{Code: tea.KeyEsc})
		})
	}
}

func TestInit(t *testing.T) {
	t.Run("without CLI connection", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		cmd := m.Init()
		if cmd == nil {
			t.Error("expected non-nil init cmd")
		}
	})
	t.Run("with CLI connection", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CLIConnection = &types.Connection{Name: "auto"}
		cmd := m.Init()
		if cmd == nil {
			t.Error("expected non-nil init cmd")
		}
	})
}
