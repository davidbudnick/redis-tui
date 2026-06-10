package ui

import (
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// ---- view_features_batch.go ----

func TestViewBulkDelete(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "default", m.viewBulkDelete())
	})
	t.Run("with preview", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.BulkDeletePreview = []string{"a", "b", "c", "d", "e", "f", "g"}
		assertNonEmpty(t, "preview", m.viewBulkDelete())
	})
}

func TestViewBatchTTL(t *testing.T) {
	m, _, _ := newTestModel(t)
	assertNonEmpty(t, "batch ttl", m.viewBatchTTL())
}

// ---- view_features_logs.go ----

func TestViewLogs(t *testing.T) {
	t.Run("nil logs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "nil", m.viewLogs())
	})
	t.Run("empty logs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Logs = types.NewLogWriter()
		assertNonEmpty(t, "empty", m.viewLogs())
	})
	t.Run("with log entries", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Logs = types.NewLogWriter()
		entries := []string{
			`{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"info msg"}`,
			`{"time":"2024-01-15T10:30:00Z","level":"ERROR","msg":"error msg"}`,
			`{"time":"2024-01-15T10:30:00Z","level":"WARN","msg":"warn msg"}`,
			`{"time":"2024-01-15T10:30:00Z","level":"DEBUG","msg":"debug msg"}`,
			`plain text log line that is very long ` + string(make([]byte, 100)),
		}
		for _, e := range entries {
			if _, err := m.Logs.Write([]byte(e + "\n")); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		}
		assertNonEmpty(t, "entries", m.viewLogs())
	})
	t.Run("showing detail", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Logs = types.NewLogWriter()
		if _, err := m.Logs.Write([]byte(`{"level":"INFO","msg":"hello"}` + "\n")); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		m.ShowingLogDetail = true
		assertNonEmpty(t, "detail", m.viewLogs())
	})
	t.Run("scrolled", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Logs = types.NewLogWriter()
		for range 30 {
			if _, err := m.Logs.Write([]byte(`{"level":"INFO","msg":"m"}` + "\n")); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		}
		m.Height = 15
		m.LogCursor = 25
		assertNonEmpty(t, "scroll", m.viewLogs())
	})
}

func TestViewLogDetail(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "json", m.viewLogDetail(`{"key":"value"}`))
	})
	t.Run("invalid json", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "invalid", m.viewLogDetail("plain text"))
	})
}

// ---- view_features_navigation.go ----

func TestViewFavorites(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewFavorites())
	})
	t.Run("with favorites", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Favorites = []types.Favorite{
			{Key: "a", Connection: "prod"},
			{Key: "b", Label: "my label"}, // Connection empty -> label fallback
		}
		assertNonEmpty(t, "with", m.viewFavorites())
	})
}

func TestViewRecentKeys(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewRecentKeys())
	})
	t.Run("with keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RecentKeys = []types.RecentKey{{Key: "a", AccessedAt: time.Now()}}
		assertNonEmpty(t, "with", m.viewRecentKeys())
	})
}

func TestViewTreeView(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewTreeView())
	})
	t.Run("with nodes", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TreeNodes = []types.TreeNode{
			{Name: "user", FullPath: "user", IsKey: false, ChildCount: 2},
			{Name: "1", FullPath: "user:1", IsKey: true},
		}
		m.TreeExpanded["user"] = true
		assertNonEmpty(t, "expanded", m.viewTreeView())
	})
	t.Run("collapsed node", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TreeNodes = []types.TreeNode{
			{Name: "user", FullPath: "user", IsKey: false},
		}
		assertNonEmpty(t, "collapsed", m.viewTreeView())
	})
}

func TestViewTemplates(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewTemplates())
	})
	t.Run("with templates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Templates = []types.KeyTemplate{
			{Name: "tpl1", KeyType: types.KeyTypeString, Pattern: "user:{id}"},
			{Name: "tpl2", KeyType: types.KeyTypeHash, Pattern: "cart:{id}"},
		}
		assertNonEmpty(t, "with", m.viewTemplates())
	})
}

func TestViewValueHistory(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewValueHistory())
	})
	t.Run("with history and current", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.ValueHistory = []types.ValueHistoryEntry{
			{Key: "foo", Value: types.RedisValue{StringValue: "old"}, Timestamp: time.Now()},
			{Key: "foo", Value: types.RedisValue{StringValue: "newer"}, Timestamp: time.Now()},
		}
		assertNonEmpty(t, "with", m.viewValueHistory())
	})
}

func TestViewExpiringKeys(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewExpiringKeys())
	})
	t.Run("with keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ExpiringKeys = []types.RedisKey{
			{Key: "critical", TTL: 5 * time.Second},
			{Key: "warning", TTL: 30 * time.Second},
			{Key: "normal", TTL: 300 * time.Second},
		}
		assertNonEmpty(t, "with", m.viewExpiringKeys())
	})
}

// ---- view_features_search.go ----

func TestViewSearchViews(t *testing.T) {
	m, _, _ := newTestModel(t)
	assertNonEmpty(t, "regex", m.viewRegexSearch())
	assertNonEmpty(t, "fuzzy", m.viewFuzzySearch())
}

func TestViewCompareKeys(t *testing.T) {
	t.Run("no result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "no result", m.viewCompareKeys())
	})
	t.Run("equal result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CompareResult = &types.KeyComparison{Equal: true}
		assertNonEmpty(t, "equal", m.viewCompareKeys())
	})
	t.Run("unequal with diffs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CompareResult = &types.KeyComparison{Equal: false, Differences: []string{"field x differs"}}
		assertNonEmpty(t, "diffs", m.viewCompareKeys())
	})
}

func TestViewJSONPath(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "no key", m.viewJSONPath())
	})
	t.Run("with key and result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.JSONPathResult = "some result"
		assertNonEmpty(t, "with", m.viewJSONPath())
	})
}

// ---- view_features_server.go ----

func TestViewClientList(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewClientList())
	})
	t.Run("with clients", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ClientList = []types.ClientInfo{
			{ID: 1, Addr: "127.0.0.1:1", Name: "c1", Age: time.Minute},
			{ID: 2, Addr: "127.0.0.1:2", Age: time.Hour}, // no name
		}
		assertNonEmpty(t, "with", m.viewClientList())
	})
}

func TestViewMemoryStats(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "nil", m.viewMemoryStats())
	})
	t.Run("with stats", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.MemoryStats = &types.MemoryStats{
			UsedMemory: 1024 * 1024,
			PeakMemory: 2048 * 1024,
			FragRatio:  1.5,
			RSS:        "2M",
			LuaMemory:  "10K",
			TopKeys:    []types.KeyMemory{{Key: "big", Memory: 500000}},
		}
		assertNonEmpty(t, "with", m.viewMemoryStats())
	})
}

func TestViewClusterInfo(t *testing.T) {
	t.Run("not enabled", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "disabled", m.viewClusterInfo())
	})
	t.Run("enabled no nodes", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ClusterEnabled = true
		assertNonEmpty(t, "empty", m.viewClusterInfo())
	})
	t.Run("with nodes", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ClusterEnabled = true
		m.ClusterNodes = []types.ClusterNode{
			{ID: "abcdef1234", Role: "master", Slots: "0-5460"},
			{ID: "short", Role: "slave"}, // no slots
		}
		assertNonEmpty(t, "with", m.viewClusterInfo())
	})
}

func TestViewKeyspaceEvents(t *testing.T) {
	t.Run("inactive no events", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "none", m.viewKeyspaceEvents())
	})
	t.Run("active with events", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.KeyspaceSubActive = true
		m.KeyspaceEvents = []types.KeyspaceEvent{
			{Event: "set", Key: "a", Timestamp: time.Now()},
			{Event: "del", Key: "b", Timestamp: time.Now()},
		}
		assertNonEmpty(t, "with", m.viewKeyspaceEvents())
	})
	t.Run("many events truncates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		for range 20 {
			m.KeyspaceEvents = append(m.KeyspaceEvents, types.KeyspaceEvent{Event: "set", Key: "k"})
		}
		assertNonEmpty(t, "many", m.viewKeyspaceEvents())
	})
}

// ---- view_metrics.go ----

func TestViewLiveMetrics(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "nil", m.viewLiveMetrics())
	})
	t.Run("empty metrics with conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{Name: "c", Host: "h", Port: 6379}
		m.LiveMetrics = &types.LiveMetrics{}
		assertNonEmpty(t, "empty metrics", m.viewLiveMetrics())
	})
	t.Run("with data", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{Name: "c", Host: "h", Port: 6379}
		m.LiveMetrics = &types.LiveMetrics{
			MaxDataPoints: 60,
			DataPoints: []types.LiveMetricsData{
				{OpsPerSec: 100, UsedMemoryBytes: 1024, ConnectedClients: 5, KeyspaceHits: 1000, KeyspaceMisses: 100, InputKbps: 1.5, OutputKbps: 2.5, UsedCPUSys: 0.1, UsedCPUUser: 0.2, TotalConnections: 50},
				{OpsPerSec: 200, UsedMemoryBytes: 2048, ConnectedClients: 10, KeyspaceHits: 2000, KeyspaceMisses: 200, InputKbps: 3.0, OutputKbps: 5.0, UsedCPUSys: 0.2, UsedCPUUser: 0.3, TotalConnections: 100},
			},
		}
		assertNonEmpty(t, "with data", m.viewLiveMetrics())
	})
	t.Run("cluster mode", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{Name: "c", Host: "h", Port: 6379, UseCluster: true}
		m.ClusterNodes = []types.ClusterNode{{ID: "n1"}, {ID: "n2"}}
		m.LiveMetrics = &types.LiveMetrics{
			MaxDataPoints: 60,
			DataPoints:    []types.LiveMetricsData{{OpsPerSec: 100}},
		}
		assertNonEmpty(t, "cluster", m.viewLiveMetrics())
	})
	t.Run("small terminal", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 40
		m.Height = 20
		m.LiveMetrics = &types.LiveMetrics{
			MaxDataPoints: 60,
			DataPoints:    []types.LiveMetricsData{{OpsPerSec: 100}},
		}
		assertNonEmpty(t, "small", m.viewLiveMetrics())
	})
	t.Run("huge terminal", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 300
		m.Height = 200
		m.LiveMetrics = &types.LiveMetrics{
			MaxDataPoints: 60,
			DataPoints:    []types.LiveMetricsData{{OpsPerSec: 100}},
		}
		assertNonEmpty(t, "huge", m.viewLiveMetrics())
	})
	t.Run("zero keyspace hits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LiveMetrics = &types.LiveMetrics{
			MaxDataPoints: 60,
			DataPoints:    []types.LiveMetricsData{{}},
		}
		assertNonEmpty(t, "zero", m.viewLiveMetrics())
	})
}

func TestRenderLineChart(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		out := renderLineChart("test", nil, 50, 5, lipgloss.Color("39"))
		if out != "" {
			t.Error("expected empty output for empty data")
		}
	})
	t.Run("same min max", func(t *testing.T) {
		out := renderLineChart("test", []float64{5.0, 5.0, 5.0}, 30, 5, lipgloss.Color("39"))
		if out == "" {
			t.Error("expected non-empty")
		}
	})
	t.Run("varied data", func(t *testing.T) {
		out := renderLineChart("test", []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 30, 5, lipgloss.Color("39"))
		if out == "" {
			t.Error("expected non-empty")
		}
	})
}

func TestResampleData(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out := resampleData(nil, 10)
		if len(out) != 0 {
			t.Errorf("expected 0, got %d", len(out))
		}
	})
	t.Run("upsample", func(t *testing.T) {
		out := resampleData([]float64{1, 2, 3}, 10)
		if len(out) != 10 {
			t.Errorf("expected 10, got %d", len(out))
		}
	})
	t.Run("downsample", func(t *testing.T) {
		out := resampleData([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 3)
		if len(out) != 3 {
			t.Errorf("expected 3, got %d", len(out))
		}
	})
	t.Run("same size", func(t *testing.T) {
		out := resampleData([]float64{1, 2, 3}, 3)
		if len(out) != 3 {
			t.Errorf("expected 3, got %d", len(out))
		}
	})
}

// ---- view_modals_help.go ----

func TestViewHelp(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "help", m.viewHelp())
	})
	t.Run("narrow", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 60
		assertNonEmpty(t, "narrow", m.viewHelp())
	})
}

func TestViewTestConnection(t *testing.T) {
	t.Run("loading", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Loading = true
		assertNonEmpty(t, "loading", m.viewTestConnection())
	})
	t.Run("success", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TestConnResult = "Connected in 2ms"
		assertNonEmpty(t, "ok", m.viewTestConnection())
	})
	t.Run("failure", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TestConnResult = "Failed: refused"
		assertNonEmpty(t, "fail", m.viewTestConnection())
	})
}

// ---- view_modals_io.go ----

func TestViewIOModals(t *testing.T) {
	m, _, _ := newTestModel(t)
	assertNonEmpty(t, "search values", m.viewSearchValues())
	assertNonEmpty(t, "export", m.viewExport())
	m.KeyPattern = "user:*"
	assertNonEmpty(t, "export pattern", m.viewExport())
	assertNonEmpty(t, "import", m.viewImport())
}

// ---- view_modals_key_ops.go ----

func TestViewTTLEditor(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "no key", m.viewTTLEditor())
	})
	t.Run("with key narrow", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 40
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		assertNonEmpty(t, "narrow", m.viewTTLEditor())
	})
}

func TestViewEditValue(t *testing.T) {
	t.Run("no editor", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "no editor", m.viewEditValue())
	})
	t.Run("with editor and key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.VimEditor = createVimEditor("content", 80, 24, "")
		assertNonEmpty(t, "with", m.viewEditValue())
	})
}

func TestViewAddToCollection(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "no key", m.viewAddToCollection())
	})
	for _, kt := range []types.KeyType{
		types.KeyTypeList, types.KeyTypeSet, types.KeyTypeZSet,
		types.KeyTypeHash, types.KeyTypeStream, types.KeyTypeHyperLogLog,
		types.KeyTypeBitmap, types.KeyTypeGeo,
	} {
		t.Run(string(kt), func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.CurrentKey = &types.RedisKey{Key: "foo", Type: kt}
			assertNonEmpty(t, string(kt), m.viewAddToCollection())
		})
	}
}

func TestViewRemoveFromCollection(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "no key", m.viewRemoveFromCollection())
	})
	t.Run("list", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeList}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeList, ListValue: []string{"a", "b", "c"}}
		m.SelectedItemIdx = 1
		assertNonEmpty(t, "list", m.viewRemoveFromCollection())
	})
	t.Run("set", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeSet}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeSet, SetValue: []string{"x", "y"}}
		m.SelectedItemIdx = 0
		assertNonEmpty(t, "set", m.viewRemoveFromCollection())
	})
	t.Run("zset", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeZSet}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeZSet, ZSetValue: []types.ZSetMember{{Member: "a", Score: 1}, {Member: "b", Score: 2}}}
		m.SelectedItemIdx = 1
		assertNonEmpty(t, "zset", m.viewRemoveFromCollection())
	})
	t.Run("hash", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeHash}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeHash, HashValue: map[string]string{"k1": "v1", "k2": "v2"}}
		m.SelectedItemIdx = 0
		assertNonEmpty(t, "hash", m.viewRemoveFromCollection())
	})
	t.Run("stream", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeStream}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeStream, StreamValue: []types.StreamEntry{{ID: "1"}, {ID: "2"}}}
		m.SelectedItemIdx = 0
		assertNonEmpty(t, "stream", m.viewRemoveFromCollection())
	})
	t.Run("geo", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeGeo}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeGeo, GeoValue: []types.GeoMember{{Name: "a"}, {Name: "b"}}}
		m.SelectedItemIdx = 0
		assertNonEmpty(t, "geo", m.viewRemoveFromCollection())
	})
}

func TestViewRenameKey(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "no key", m.viewRenameKey())
	})
	t.Run("with key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		assertNonEmpty(t, "with", m.viewRenameKey())
	})
}

func TestViewCopyKey(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "no key", m.viewCopyKey())
	})
	t.Run("with key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		assertNonEmpty(t, "with", m.viewCopyKey())
	})
}

// ---- view_modals_server.go ----

func TestViewConfirmDelete(t *testing.T) {
	t.Run("connection", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "connection"
		m.ConfirmData = types.Connection{Name: "c"}
		assertNonEmpty(t, "conn", m.viewConfirmDelete())
	})
	t.Run("connection bad type", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "connection"
		m.ConfirmData = "not a conn"
		assertNonEmpty(t, "bad conn", m.viewConfirmDelete())
	})
	t.Run("key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "key"
		m.ConfirmData = types.RedisKey{Key: "foo"}
		assertNonEmpty(t, "key", m.viewConfirmDelete())
	})
	t.Run("key bad type", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "key"
		m.ConfirmData = "bad"
		assertNonEmpty(t, "bad key", m.viewConfirmDelete())
	})
	t.Run("flushdb", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConfirmType = "flushdb"
		assertNonEmpty(t, "flush", m.viewConfirmDelete())
	})
}

func TestViewServerInfo(t *testing.T) {
	t.Run("with info", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ServerInfo = types.ServerInfo{
			Version: "7.0.0", Mode: "standalone", OS: "linux",
			UsedMemory: "1M", Clients: "10", TotalKeys: "100", Uptime: "1h",
		}
		assertNonEmpty(t, "info", m.viewServerInfo())
	})
	t.Run("narrow width", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 40
		assertNonEmpty(t, "narrow", m.viewServerInfo())
	})
}

func TestViewPubSubChannels(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewPubSubChannels())
	})
	t.Run("with channels", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.PubSubChannels = []types.PubSubChannel{
			{Name: "a", Subscribers: 5},
			{Name: "b"},
		}
		assertNonEmpty(t, "with", m.viewPubSubChannels())
	})
}

func TestViewPubSub(t *testing.T) {
	m, _, _ := newTestModel(t)
	assertNonEmpty(t, "pubsub", m.viewPubSub())
}

func TestViewRedisConfig(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewRedisConfig())
	})
	t.Run("with params", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RedisConfigParams = []types.RedisConfigParam{
			{Name: "maxmemory", Value: "100mb"},
			{Name: "timeout", Value: "0"},
		}
		assertNonEmpty(t, "params", m.viewRedisConfig())
	})
	t.Run("editing", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RedisConfigParams = []types.RedisConfigParam{{Name: "maxmemory", Value: "100mb"}}
		m.EditingConfigParam = "maxmemory"
		m.Inputs.ConfigEditInput.SetValue("200mb")
		assertNonEmpty(t, "edit", m.viewRedisConfig())
	})
	t.Run("many params scroll", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		for i := range 30 {
			m.RedisConfigParams = append(m.RedisConfigParams, types.RedisConfigParam{Name: string(rune('a' + i))})
		}
		m.SelectedConfigIdx = 25
		m.Height = 15
		assertNonEmpty(t, "scroll", m.viewRedisConfig())
	})
	t.Run("small height clamps", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Height = 5
		m.RedisConfigParams = []types.RedisConfigParam{{Name: "a"}}
		assertNonEmpty(t, "small", m.viewRedisConfig())
	})
}

func TestViewSwitchDB(t *testing.T) {
	t.Run("no conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "none", m.viewSwitchDB())
	})
	t.Run("with conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{DB: 3}
		assertNonEmpty(t, "with", m.viewSwitchDB())
	})
}

func TestViewSlowLog(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "empty", m.viewSlowLog())
	})
	t.Run("with entries", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.SlowLogEntries = []types.SlowLogEntry{
			{ID: 1, Duration: 10 * time.Millisecond, Command: "GET foo"},
		}
		assertNonEmpty(t, "entries", m.viewSlowLog())
	})
}

func TestViewLuaScript(t *testing.T) {
	t.Run("no result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "none", m.viewLuaScript())
	})
	t.Run("with result", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.LuaResult = "42"
		assertNonEmpty(t, "result", m.viewLuaScript())
	})
}
