package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

// assertNonEmpty runs a view func and fails if it returns empty.
func assertNonEmpty(t *testing.T, name, s string) {
	t.Helper()
	if s == "" {
		t.Errorf("%s: expected non-empty output", name)
	}
}

func TestViewConnections(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		out := m.viewConnections()
		assertNonEmpty(t, "empty", out)
		if !strings.Contains(out, "No connections") {
			t.Error("expected empty-list message")
		}
	})
	t.Run("with connections", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = []types.Connection{
			{Name: "c1", Host: "localhost", Port: 6379, DB: 0},
			{Name: "c2", Host: "redis.prod", Port: 6380, UseCluster: true},
			{Name: "c3", Host: "redis.tls", Port: 6381, UseTLS: true},
		}
		assertNonEmpty(t, "with conns", m.viewConnections())
	})
	t.Run("scroll past max visible", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Height = 25
		for range 20 {
			m.Connections = append(m.Connections, types.Connection{Name: "c", Host: "h", Port: 6379})
		}
		m.SelectedConnIdx = 19
		assertNonEmpty(t, "scroll", m.viewConnections())
	})
	t.Run("selected out of range clamps", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = []types.Connection{{Name: "c"}}
		m.SelectedConnIdx = 99
		assertNonEmpty(t, "clamp", m.viewConnections())
	})
	t.Run("selected negative clamps", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = []types.Connection{{Name: "c"}}
		m.SelectedConnIdx = -1
		assertNonEmpty(t, "clamp", m.viewConnections())
	})
	t.Run("connection error", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnectionError = "boom"
		out := m.viewConnections()
		if !strings.Contains(out, "Connection Failed") {
			t.Error("expected error box")
		}
	})
}

func TestBuildStatsBar(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Connections = []types.Connection{{Name: "a"}}
	assertNonEmpty(t, "stats", m.buildStatsBar())
}

func TestViewAddConnection(t *testing.T) {
	m, _, _ := newTestModel(t)
	out := m.viewAddConnection()
	if !strings.Contains(out, "Add Connection") {
		t.Error("expected title")
	}
}

func TestViewEditConnection(t *testing.T) {
	m, _, _ := newTestModel(t)
	out := m.viewEditConnection()
	if !strings.Contains(out, "Edit Connection") {
		t.Error("expected title")
	}
}

func TestRenderConnForm(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		assertNonEmpty(t, "form", m.renderConnForm())
	})
	t.Run("cluster mode hides db", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnClusterMode = true
		out := m.renderConnForm()
		if strings.Contains(out, "Database:") {
			t.Error("expected Database hidden in cluster mode")
		}
	})
	t.Run("focus on each field", func(t *testing.T) {
		for i := 0; i <= 9; i++ {
			m, _, _ := newTestModel(t)
			m.ConnFocusIdx = i
			assertNonEmpty(t, "focus", m.renderConnForm())
		}
	})
}

func TestViewKeys(t *testing.T) {
	t.Run("narrow terminal", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 80
		assertNonEmpty(t, "narrow", m.viewKeys())
	})
	t.Run("wide with keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 150
		m.Keys = []types.RedisKey{{Key: "foo", Type: types.KeyTypeString}}
		assertNonEmpty(t, "wide", m.viewKeys())
	})
}

func TestViewKeysListOnly(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		out := m.viewKeysListOnly()
		if !strings.Contains(out, "No keys") {
			t.Error("expected empty message")
		}
	})
	t.Run("with conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{Name: "c", Host: "h", Port: 6379, DB: 0}
		m.TotalKeys = 100
		assertNonEmpty(t, "with conn", m.viewKeysListOnly())
	})
	t.Run("cluster conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{Name: "c", Host: "h", Port: 6379, UseCluster: true}
		m.ClusterNodes = []types.ClusterNode{{ID: "n1"}}
		assertNonEmpty(t, "cluster", m.viewKeysListOnly())
	})
	t.Run("pattern focused", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.PatternInput.Focus()
		assertNonEmpty(t, "pattern focused", m.viewKeysListOnly())
	})
	t.Run("with keys various TTLs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{
			{Key: "short", Type: types.KeyTypeString, TTL: 5 * time.Second},
			{Key: "medium", Type: types.KeyTypeString, TTL: 30 * time.Second},
			{Key: "long", Type: types.KeyTypeString, TTL: 3600 * time.Second},
			{Key: "none", Type: types.KeyTypeList, TTL: 0},
			{Key: "expired", Type: types.KeyTypeSet, TTL: -2},
			{Key: strings.Repeat("x", 50), Type: types.KeyTypeHash, TTL: -1}, // test truncation
		}
		assertNonEmpty(t, "keys", m.viewKeysListOnly())
	})
	t.Run("many keys scrolled past window", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Height = 20
		for range 30 {
			m.Keys = append(m.Keys, types.RedisKey{Key: "k", Type: types.KeyTypeString})
		}
		m.SelectedKeyIdx = 29
		m.KeyCursor = 100
		assertNonEmpty(t, "scroll", m.viewKeysListOnly())
	})
	t.Run("selected out of bounds", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a"}}
		m.SelectedKeyIdx = 99
		assertNonEmpty(t, "oob", m.viewKeysListOnly())
	})
	t.Run("selected negative", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a"}}
		m.SelectedKeyIdx = -5
		assertNonEmpty(t, "neg", m.viewKeysListOnly())
	})
	t.Run("small height clamps", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Height = 5
		m.Keys = []types.RedisKey{{Key: "a"}}
		assertNonEmpty(t, "small", m.viewKeysListOnly())
	})
}

func TestBuildKeysListPanel(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		out := m.buildKeysListPanel(80)
		if !strings.Contains(out, "No keys") {
			t.Error("expected empty message")
		}
	})
	t.Run("with conn cluster", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{Name: "c", UseCluster: true}
		m.ClusterNodes = []types.ClusterNode{{ID: "n1"}}
		m.TotalKeys = 50
		assertNonEmpty(t, "cluster", m.buildKeysListPanel(80))
	})
	t.Run("with conn normal", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{Name: "c"}
		assertNonEmpty(t, "normal", m.buildKeysListPanel(80))
	})
	t.Run("pattern focused", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.PatternInput.Focus()
		assertNonEmpty(t, "focus", m.buildKeysListPanel(80))
	})
	t.Run("with keys various formats", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{
			{Key: "short", Type: types.KeyTypeString, TTL: 30 * time.Second},
			{Key: "medium", Type: types.KeyTypeList, TTL: 120 * time.Second},
			{Key: "long", Type: types.KeyTypeSet, TTL: 7200 * time.Second},
			{Key: "expired", Type: types.KeyTypeHash, TTL: -2},
			{Key: strings.Repeat("a", 40), Type: types.KeyTypeZSet},
		}
		assertNonEmpty(t, "keys", m.buildKeysListPanel(60))
	})
	t.Run("narrow width", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "k"}}
		assertNonEmpty(t, "narrow", m.buildKeysListPanel(30))
	})
	t.Run("many keys with pagination", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Height = 20
		for range 30 {
			m.Keys = append(m.Keys, types.RedisKey{Key: "k"})
		}
		m.SelectedKeyIdx = 25
		m.KeyCursor = 100
		assertNonEmpty(t, "paginate", m.buildKeysListPanel(80))
	})
	t.Run("selected clamp", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a"}}
		m.SelectedKeyIdx = 99
		assertNonEmpty(t, "clamp", m.buildKeysListPanel(80))
	})
	t.Run("selected negative", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a"}}
		m.SelectedKeyIdx = -1
		assertNonEmpty(t, "neg", m.buildKeysListPanel(80))
	})
	t.Run("small height", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Height = 5
		m.Keys = []types.RedisKey{{Key: "a"}}
		assertNonEmpty(t, "small", m.buildKeysListPanel(80))
	})
}

func TestBuildPreviewPanel(t *testing.T) {
	t.Run("no keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		out := m.buildPreviewPanel(60)
		if !strings.Contains(out, "No key selected") {
			t.Error("expected no key message")
		}
	})
	t.Run("no preview loaded", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "foo", Type: types.KeyTypeString}}
		out := m.buildPreviewPanel(60)
		if !strings.Contains(out, "Loading") {
			t.Error("expected loading message")
		}
	})
	t.Run("ttl variants", func(t *testing.T) {
		cases := []time.Duration{5 * time.Second, 30 * time.Second, 120 * time.Second, 7200 * time.Second, 86400 * time.Second * 2, -1}
		for _, d := range cases {
			m, _, _ := newTestModel(t)
			m.Keys = []types.RedisKey{{Key: "foo", Type: types.KeyTypeString, TTL: d}}
			m.PreviewKey = "foo"
			m.PreviewValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "v"}
			assertNonEmpty(t, "ttl", m.buildPreviewPanel(60))
		}
	})
	t.Run("long key name truncated", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		longKey := strings.Repeat("k", 100)
		m.Keys = []types.RedisKey{{Key: longKey, Type: types.KeyTypeString}}
		m.PreviewKey = longKey
		m.PreviewValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "v"}
		assertNonEmpty(t, "long key", m.buildPreviewPanel(30))
	})
}

func TestFormatPreviewValue(t *testing.T) {
	tests := []struct {
		name  string
		value types.RedisValue
	}{
		{"string", types.RedisValue{Type: types.KeyTypeString, StringValue: "hello"}},
		{"string multiline", types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("line\n", 20)}},
		{"string long line", types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("x", 200)}},
		{"empty list", types.RedisValue{Type: types.KeyTypeList, ListValue: nil}},
		{"list", types.RedisValue{Type: types.KeyTypeList, ListValue: []string{"a", "b", "c"}}},
		{"list long values", types.RedisValue{Type: types.KeyTypeList, ListValue: []string{strings.Repeat("x", 100)}}},
		{"list overflow", types.RedisValue{Type: types.KeyTypeList, ListValue: make([]string, 30)}},
		{"empty set", types.RedisValue{Type: types.KeyTypeSet}},
		{"set", types.RedisValue{Type: types.KeyTypeSet, SetValue: []string{"x", "y"}}},
		{"set long", types.RedisValue{Type: types.KeyTypeSet, SetValue: []string{strings.Repeat("x", 100)}}},
		{"set overflow", types.RedisValue{Type: types.KeyTypeSet, SetValue: make([]string, 30)}},
		{"empty zset", types.RedisValue{Type: types.KeyTypeZSet}},
		{"zset", types.RedisValue{Type: types.KeyTypeZSet, ZSetValue: []types.ZSetMember{{Member: "a", Score: 1.0}}}},
		{"zset long", types.RedisValue{Type: types.KeyTypeZSet, ZSetValue: []types.ZSetMember{{Member: strings.Repeat("x", 100)}}}},
		{"zset overflow", types.RedisValue{Type: types.KeyTypeZSet, ZSetValue: make([]types.ZSetMember, 30)}},
		{"empty hash", types.RedisValue{Type: types.KeyTypeHash}},
		{"hash", types.RedisValue{Type: types.KeyTypeHash, HashValue: map[string]string{"k": "v"}}},
		{"hash long key", types.RedisValue{Type: types.KeyTypeHash, HashValue: map[string]string{strings.Repeat("k", 30): "v"}}},
		{"hash long value", types.RedisValue{Type: types.KeyTypeHash, HashValue: map[string]string{"k": strings.Repeat("v", 200)}}},
		{"hash overflow", types.RedisValue{Type: types.KeyTypeHash, HashValue: func() map[string]string {
			h := map[string]string{}
			for i := range 30 {
				h[string(rune('a'+i))] = "v"
			}
			return h
		}()}},
		{"empty stream", types.RedisValue{Type: types.KeyTypeStream}},
		{"stream", types.RedisValue{Type: types.KeyTypeStream, StreamValue: []types.StreamEntry{{ID: "1-0", Fields: map[string]any{"a": 1}}}}},
		{"stream long", types.RedisValue{Type: types.KeyTypeStream, StreamValue: []types.StreamEntry{{ID: "1-0", Fields: map[string]any{strings.Repeat("k", 50): strings.Repeat("v", 50)}}}}},
		{"stream overflow", types.RedisValue{Type: types.KeyTypeStream, StreamValue: make([]types.StreamEntry, 30)}},
		{"json", types.RedisValue{Type: types.KeyTypeJSON, JSONValue: `{"a":1}`}},
		{"json multiline", types.RedisValue{Type: types.KeyTypeJSON, JSONValue: strings.Repeat("{\"a\":1}\n", 20)}},
		{"json long line", types.RedisValue{Type: types.KeyTypeJSON, JSONValue: strings.Repeat("x", 200)}},
		{"hll", types.RedisValue{Type: types.KeyTypeHyperLogLog, HLLCount: 42}},
		{"bitmap with positions", types.RedisValue{Type: types.KeyTypeBitmap, BitCount: 3, BitPositions: []int64{1, 2, 3}}},
		{"bitmap overflow positions", types.RedisValue{Type: types.KeyTypeBitmap, BitPositions: make([]int64, 30)}},
		{"bitmap empty", types.RedisValue{Type: types.KeyTypeBitmap}},
		{"protobuf", types.RedisValue{Type: types.KeyTypeProtobuf, DecodedFormat: "s2+protobuf", DecodedValue: "1: \"menu\"\n2: 42\n", RawSize: 100, DecodedSize: 400}},
		{"protobuf empty decode", types.RedisValue{Type: types.KeyTypeProtobuf, RawSize: 10}},
		{"protobuf long lines", types.RedisValue{Type: types.KeyTypeProtobuf, DecodedFormat: "protobuf", DecodedValue: strings.Repeat("1: \"x\"\n", 40), RawSize: 50, DecodedSize: 50}},
		{"protobuf wide line", types.RedisValue{Type: types.KeyTypeProtobuf, DecodedFormat: "protobuf", DecodedValue: "1: \"" + strings.Repeat("w", 200) + "\"\n", RawSize: 20, DecodedSize: 20}},
		{"protobuf same size preview", types.RedisValue{Type: types.KeyTypeProtobuf, DecodedFormat: "", DecodedValue: "1: 1\n", RawSize: 8, DecodedSize: 8}},
		{"empty geo", types.RedisValue{Type: types.KeyTypeGeo}},
		{"geo", types.RedisValue{Type: types.KeyTypeGeo, GeoValue: []types.GeoMember{{Name: "sf", Longitude: -122.4, Latitude: 37.7}}}},
		{"geo long name", types.RedisValue{Type: types.KeyTypeGeo, GeoValue: []types.GeoMember{{Name: strings.Repeat("x", 100)}}}},
		{"geo overflow", types.RedisValue{Type: types.KeyTypeGeo, GeoValue: make([]types.GeoMember, 30)}},
		{"unknown", types.RedisValue{Type: "unknown"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.PreviewValue = tt.value
			_ = m.formatPreviewValue(80, 10)
		})
	}
	t.Run("too narrow", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.PreviewValue = types.RedisValue{Type: types.KeyTypeString}
		out := m.formatPreviewValue(10, 10)
		if !strings.Contains(out, "narrow") {
			t.Error("expected narrow message")
		}
	})
}

func TestViewKeyDetail(t *testing.T) {
	types_ := []types.KeyType{
		types.KeyTypeString, types.KeyTypeList, types.KeyTypeSet,
		types.KeyTypeZSet, types.KeyTypeHash, types.KeyTypeStream,
		types.KeyTypeJSON, types.KeyTypeHyperLogLog, types.KeyTypeBitmap, types.KeyTypeGeo, types.KeyTypeProtobuf,
	}
	for _, kt := range types_ {
		t.Run(string(kt), func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.CurrentKey = &types.RedisKey{Key: "foo", Type: kt, TTL: 5 * time.Second, MemorySize: 100}
			m.MemoryUsage = 200
			m.CurrentValue = types.RedisValue{
				Type:         kt,
				StringValue:  "hello",
				ListValue:    []string{"a", "b"},
				SetValue:     []string{"x"},
				ZSetValue:    []types.ZSetMember{{Member: "a", Score: 1}},
				HashValue:    map[string]string{"k": "v"},
				StreamValue:  []types.StreamEntry{{ID: "1-0", Fields: map[string]any{"a": 1}}},
				JSONValue:    `{"a":1}`,
				HLLCount:     100,
				BitCount:     5,
				BitPositions: []int64{1, 2, 3},
				GeoValue:     []types.GeoMember{{Name: "sf", Longitude: -122.4, Latitude: 37.7}},
			}
			assertNonEmpty(t, string(kt), m.viewKeyDetail())
		})
	}
	t.Run("nil current key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		out := m.viewKeyDetail()
		if !strings.Contains(out, "No key") {
			t.Error("expected no key message")
		}
	})
	t.Run("ttl short", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString, TTL: 5 * time.Second}
		assertNonEmpty(t, "short ttl", m.viewKeyDetail())
	})
	t.Run("ttl medium", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString, TTL: 30 * time.Second}
		assertNonEmpty(t, "med ttl", m.viewKeyDetail())
	})
	t.Run("ttl long", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString, TTL: 3600 * time.Second}
		assertNonEmpty(t, "long ttl", m.viewKeyDetail())
	})
	t.Run("empty collections", func(t *testing.T) {
		for _, kt := range []types.KeyType{types.KeyTypeList, types.KeyTypeSet, types.KeyTypeZSet, types.KeyTypeHash, types.KeyTypeStream, types.KeyTypeGeo} {
			m, _, _ := newTestModel(t)
			m.CurrentKey = &types.RedisKey{Key: "foo", Type: kt}
			m.CurrentValue = types.RedisValue{Type: kt}
			assertNonEmpty(t, string(kt), m.viewKeyDetail())
		}
	})
	t.Run("hash multiline value", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeHash}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeHash, HashValue: map[string]string{"k": `{"nested":"json"}`}}
		assertNonEmpty(t, "multiline", m.viewKeyDetail())
	})
	t.Run("bitmap empty positions", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeBitmap}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeBitmap, BitCount: 0}
		assertNonEmpty(t, "empty bits", m.viewKeyDetail())
	})
	t.Run("bitmap capped positions", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeBitmap}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeBitmap, BitCount: 10, BitPositions: []int64{0, 1, 2}}
		out := m.viewKeyDetail()
		if !strings.Contains(out, "more") {
			t.Error("expected truncated bit positions message")
		}
	})
	t.Run("protobuf decoded", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "menu", Type: types.KeyTypeProtobuf}
		m.CurrentValue = types.RedisValue{
			Type:          types.KeyTypeProtobuf,
			DecodedFormat: "s2+protobuf",
			DecodedValue:  "1: \"menu-id\"\n2: 7\n",
			RawSize:       100,
			DecodedSize:   400,
		}
		out := m.viewKeyDetail()
		if !strings.Contains(out, "s2+protobuf") {
			t.Error("expected format label in detail view")
		}
		if !strings.Contains(out, "menu-id") {
			t.Error("expected decoded body in detail view")
		}
	})
	t.Run("protobuf same size", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "menu", Type: types.KeyTypeProtobuf}
		m.CurrentValue = types.RedisValue{
			Type:          types.KeyTypeProtobuf,
			DecodedFormat: "protobuf",
			DecodedValue:  "1: 1\n",
			RawSize:       20,
			DecodedSize:   20,
		}
		assertNonEmpty(t, "proto same size", m.viewKeyDetail())
	})
	t.Run("protobuf empty body", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "menu", Type: types.KeyTypeProtobuf}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeProtobuf}
		out := m.viewKeyDetail()
		if !strings.Contains(out, "unable to decode") {
			t.Error("expected unable-to-decode message")
		}
	})
	t.Run("protobuf selected line band", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 120
		m.Height = 40
		m.CurrentKey = &types.RedisKey{Key: "menu", Type: types.KeyTypeProtobuf}
		m.CurrentValue = types.RedisValue{
			Type:          types.KeyTypeProtobuf,
			DecodedFormat: "protobuf",
			DecodedValue:  strings.Repeat("1: \"item\"\n", 80),
			RawSize:       10,
			DecodedSize:   10,
		}
		m.DetailCursor = 40
		m.DetailScroll = 30
		out := m.viewKeyDetail()
		if !strings.Contains(out, "item") {
			t.Error("expected content with selection")
		}
		if !strings.Contains(out, "more lines") {
			t.Error("expected scroll hints when cursor mid-body")
		}
		_ = out
	})
	t.Run("string selected line band", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 120
		m.Height = 40
		m.CurrentKey = &types.RedisKey{Key: "s", Type: types.KeyTypeString}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "line1\nline2\nline3"}
		m.DetailCursor = 1
		assertNonEmpty(t, "string select", m.viewKeyDetail())
	})
	t.Run("long value scrolls", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("line\n", 100)}
		m.DetailScroll = 10
		assertNonEmpty(t, "scroll", m.viewKeyDetail())
	})
	t.Run("scroll past bounds", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("line\n", 100)}
		m.DetailScroll = 9999
		assertNonEmpty(t, "scroll max", m.viewKeyDetail())
	})
	t.Run("negative scroll", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("line\n", 100)}
		m.DetailScroll = -10
		assertNonEmpty(t, "scroll neg", m.viewKeyDetail())
	})
	t.Run("small height", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Height = 5
		m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
		assertNonEmpty(t, "small", m.viewKeyDetail())
	})
}

func TestDetailContentLines(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Width = 120
	m.Height = 40
	m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "a\nb\nc"}
	if n := m.detailContentLines(); n < 1 {
		t.Errorf("expected >= 1, got %d", n)
	}
	m.CurrentValue = types.RedisValue{
		Type:          types.KeyTypeProtobuf,
		DecodedFormat: "s2+protobuf",
		DecodedValue:  strings.Repeat("1: \"x\"\n", 100),
		RawSize:       50,
		DecodedSize:   200,
	}
	if n := m.detailContentLines(); n < 10 {
		t.Errorf("protobuf lines = %d, want many", n)
	}
	if max := m.detailMaxScroll(); max <= 0 {
		t.Errorf("expected positive max scroll for long protobuf, got %d", max)
	}
	m.Height = 8
	if n := m.detailMaxScroll(); n < 0 {
		t.Errorf("tiny height scroll = %d", n)
	}
	// Force avail clamp path (detailMaxVisible floors at 5, so maxVisible-1 is always >=4
	// for normal sizes; exercise via huge content anyway).
	m.Height = detailChromeLines + 1
	_ = m.detailMaxScroll()
	// short value → zero scroll
	m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "hi"}
	m.Height = 40
	if n := m.detailMaxScroll(); n != 0 {
		t.Errorf("short value max scroll = %d", n)
	}
}

func TestDetailValueString(t *testing.T) {
	types_ := []types.KeyType{
		types.KeyTypeString, types.KeyTypeList, types.KeyTypeSet,
		types.KeyTypeZSet, types.KeyTypeHash, types.KeyTypeStream,
		types.KeyTypeJSON, types.KeyTypeHyperLogLog, types.KeyTypeBitmap, types.KeyTypeGeo, types.KeyTypeProtobuf,
	}
	for _, kt := range types_ {
		t.Run(string(kt), func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.CurrentValue = types.RedisValue{
				Type:         kt,
				StringValue:  "s",
				ListValue:    []string{"a"},
				SetValue:     []string{"a"},
				ZSetValue:    []types.ZSetMember{{Member: "m"}},
				HashValue:    map[string]string{"k": "v"},
				StreamValue:  []types.StreamEntry{{ID: "1"}},
				JSONValue:    `{}`,
				HLLCount:     1,
				BitCount:     1,
				BitPositions: []int64{1},
				GeoValue:     []types.GeoMember{{Name: "a"}},
			}
			_ = m.detailValueString()
		})
	}
}

func TestDetailMaxScroll(t *testing.T) {
	t.Run("no overflow", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		if n := m.detailMaxScroll(); n != 0 {
			t.Errorf("expected 0, got %d", n)
		}
	})
	t.Run("with overflow", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Height = 10
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("line\n", 100)}
		if n := m.detailMaxScroll(); n <= 0 {
			t.Errorf("expected > 0, got %d", n)
		}
	})
}

func TestViewAddKey(t *testing.T) {
	for _, kt := range []types.KeyType{
		types.KeyTypeString, types.KeyTypeList, types.KeyTypeSet,
		types.KeyTypeZSet, types.KeyTypeHash, types.KeyTypeStream,
		types.KeyTypeJSON, types.KeyTypeHyperLogLog, types.KeyTypeBitmap, types.KeyTypeGeo,
	} {
		t.Run(string(kt), func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.AddKeyType = kt
			assertNonEmpty(t, string(kt), m.viewAddKey())
		})
	}
	t.Run("narrow width", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Width = 30
		assertNonEmpty(t, "narrow", m.viewAddKey())
	})
}

func TestDetailValueContentCache(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "hi"}
	m.DetailRendered = ""
	if got := m.detailValueContent(); got == "" {
		t.Error("expected uncached content")
	}
	_ = m.detailValuePlainString()
	m.DetailRendered = "CACHED"
	if m.detailValueContent() != "CACHED" {
		t.Error("expected cache hit")
	}
}

func TestPreviewFormattedAndTruncated(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Keys = []types.RedisKey{{Key: "k", Type: types.KeyTypeList}}
	m.SelectedKeyIdx = 0
	m.PreviewKey = "k"
	m.PreviewValue = types.RedisValue{Type: types.KeyTypeList, ListValue: []string{"a"}, Truncated: true, TotalCount: 99}
	m.PreviewRendered = ""
	if m.previewFormatted("x") == "" {
		t.Error("fallback format")
	}
	m.PreviewRendered = "PRE"
	if m.previewFormatted("x") != "PRE" {
		t.Error("cache hit")
	}
	out := m.buildPreviewPanel(40)
	if !strings.Contains(out, "99") {
		t.Errorf("expected truncated total, got %q", out)
	}
}

func TestBuildDetailValueContent_Truncated(t *testing.T) {
	out := buildDetailValueContent(types.RedisValue{
		Type:       types.KeyTypeList,
		ListValue:  []string{"a", "b"},
		Truncated:  true,
		TotalCount: 1500,
	})
	if !strings.Contains(out, "truncated") || !strings.Contains(out, "1500") {
		t.Errorf("expected truncated marker, got %q", out)
	}
	if !strings.Contains(out, "a") {
		t.Errorf("expected list body, got %q", out)
	}

	empty := buildDetailValueContent(types.RedisValue{
		Type:       types.KeyTypeString,
		Truncated:  true,
		TotalCount: 42,
	})
	if !strings.Contains(empty, "truncated") || !strings.Contains(empty, "42") {
		t.Errorf("expected truncated-only body, got %q", empty)
	}
}

// BenchmarkViewKeyDetailLargeJSON measures one cached detail render frame.
func BenchmarkViewKeyDetailLargeJSON(b *testing.B) {
	var sb strings.Builder
	sb.WriteString(`{"items": [`)
	for i := 0; i < 5000; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"id": 12345, "name": "some-item-name", "active": true, "score": 98.6, "tag": null}`)
	}
	sb.WriteString(`]}`)

	m := NewModel()
	m.Width = 120
	m.Height = 40
	m.CurrentKey = &types.RedisKey{Key: "big:json", Type: types.KeyTypeString}
	m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: sb.String()}
	m.DetailRendered = buildDetailValueContent(m.CurrentValue)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.viewKeyDetail()
	}
}

// BenchmarkViewKeyDetailProtobuf measures a cached detail frame for a large
// decoded protobuf body (typical after s2 decompress + decode_raw).
func BenchmarkViewKeyDetailProtobuf(b *testing.B) {
	var body strings.Builder
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&body, "5: {\n  1: \"cat:%d\"\n  2: \"Category Name %d\"\n  4: 1\n  5: 1\n  6: 1\n  7: %d\n}\n", i, i, i)
	}
	m := NewModel()
	m.Width = 120
	m.Height = 40
	m.CurrentKey = &types.RedisKey{Key: "menu:v4:demo", Type: types.KeyTypeProtobuf}
	m.CurrentValue = types.RedisValue{
		Type:          types.KeyTypeProtobuf,
		DecodedFormat: "s2+protobuf",
		DecodedValue:  body.String(),
		RawSize:       200_000,
		DecodedSize:   800_000,
	}
	m.DetailRendered = buildDetailValueContent(m.CurrentValue)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.viewKeyDetail()
	}
}
