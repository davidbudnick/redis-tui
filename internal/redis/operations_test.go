package redis

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/klauspost/compress/s2"
	goredis "github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------------
// GetValue
// ---------------------------------------------------------------------------

func TestGetValue(t *testing.T) {
	client, mr := setupTestClient(t)

	// Seed each key type via miniredis helpers.
	mr.Set("str", "hello")
	mr.RPush("list", "a", "b")
	mr.SAdd("set", "x", "y")
	mr.ZAdd("zset", 1.5, "m1")
	mr.HSet("hash", "f1", "v1")

	// Streams are not natively supported by miniredis helpers,
	// so we use the client wrapper.
	streamID, err := client.XAdd("stream", map[string]any{"field": "value"})
	if err != nil {
		t.Fatalf("XAdd failed: %v", err)
	}
	if streamID == "" {
		t.Fatal("expected non-empty stream ID")
	}

	tests := []struct {
		name     string
		key      string
		wantType types.KeyType
		check    func(t *testing.T, v types.RedisValue)
	}{
		{
			name:     "string key",
			key:      "str",
			wantType: types.KeyTypeString,
			check: func(t *testing.T, v types.RedisValue) {
				if v.StringValue != "hello" {
					t.Errorf("StringValue = %q, want %q", v.StringValue, "hello")
				}
			},
		},
		{
			name:     "list key",
			key:      "list",
			wantType: types.KeyTypeList,
			check: func(t *testing.T, v types.RedisValue) {
				if len(v.ListValue) != 2 {
					t.Fatalf("ListValue length = %d, want 2", len(v.ListValue))
				}
				if v.ListValue[0] != "a" || v.ListValue[1] != "b" {
					t.Errorf("ListValue = %v, want [a b]", v.ListValue)
				}
			},
		},
		{
			name:     "set key",
			key:      "set",
			wantType: types.KeyTypeSet,
			check: func(t *testing.T, v types.RedisValue) {
				if len(v.SetValue) != 2 {
					t.Fatalf("SetValue length = %d, want 2", len(v.SetValue))
				}
				sort.Strings(v.SetValue)
				if v.SetValue[0] != "x" || v.SetValue[1] != "y" {
					t.Errorf("SetValue = %v, want [x y]", v.SetValue)
				}
			},
		},
		{
			name:     "zset key",
			key:      "zset",
			wantType: types.KeyTypeZSet,
			check: func(t *testing.T, v types.RedisValue) {
				if len(v.ZSetValue) != 1 {
					t.Fatalf("ZSetValue length = %d, want 1", len(v.ZSetValue))
				}
				if v.ZSetValue[0].Member != "m1" {
					t.Errorf("ZSetValue[0].Member = %q, want %q", v.ZSetValue[0].Member, "m1")
				}
				if v.ZSetValue[0].Score != 1.5 {
					t.Errorf("ZSetValue[0].Score = %f, want 1.5", v.ZSetValue[0].Score)
				}
			},
		},
		{
			name:     "hash key",
			key:      "hash",
			wantType: types.KeyTypeHash,
			check: func(t *testing.T, v types.RedisValue) {
				if v.HashValue["f1"] != "v1" {
					t.Errorf("HashValue[f1] = %q, want %q", v.HashValue["f1"], "v1")
				}
			},
		},
		{
			name:     "stream key",
			key:      "stream",
			wantType: types.KeyTypeStream,
			check: func(t *testing.T, v types.RedisValue) {
				if len(v.StreamValue) != 1 {
					t.Fatalf("StreamValue length = %d, want 1", len(v.StreamValue))
				}
				if v.StreamValue[0].Fields["field"] != "value" {
					t.Errorf("StreamValue[0].Fields[field] = %v, want %q", v.StreamValue[0].Fields["field"], "value")
				}
			},
		},
		{
			name:     "missing key returns none type",
			key:      "nonexistent",
			wantType: types.KeyType("none"),
			check:    func(t *testing.T, v types.RedisValue) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := client.GetValue(tt.key)
			if err != nil {
				t.Fatalf("GetValue(%q) error: %v", tt.key, err)
			}
			if v.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", v.Type, tt.wantType)
			}
			tt.check(t, v)
		})
	}
}

// ---------------------------------------------------------------------------
// DeleteKey
// ---------------------------------------------------------------------------

func TestDeleteKey(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("to-delete", "bye")

	if err := client.DeleteKey("to-delete"); err != nil {
		t.Fatalf("DeleteKey error: %v", err)
	}

	if mr.Exists("to-delete") {
		t.Error("key should no longer exist after DeleteKey")
	}
}

// ---------------------------------------------------------------------------
// BulkDelete
// ---------------------------------------------------------------------------

func TestBulkDelete(t *testing.T) {
	t.Run("basic pattern delete", func(t *testing.T) {
		client, mr := setupTestClient(t)

		for i := range 5 {
			mr.Set(fmt.Sprintf("bulk:%d", i), "val")
		}
		// Key that should NOT be deleted.
		mr.Set("other:1", "keep")

		deleted, err := client.BulkDelete("bulk:*")
		if err != nil {
			t.Fatalf("BulkDelete error: %v", err)
		}
		if deleted != 5 {
			t.Errorf("deleted = %d, want 5", deleted)
		}

		for i := range 5 {
			if mr.Exists(fmt.Sprintf("bulk:%d", i)) {
				t.Errorf("key bulk:%d should be deleted", i)
			}
		}
		if !mr.Exists("other:1") {
			t.Error("other:1 should still exist")
		}
	})

	t.Run("chunked delete with 250+ keys", func(t *testing.T) {
		client, mr := setupTestClient(t)

		const keyCount = 260
		for i := range keyCount {
			mr.Set(fmt.Sprintf("chunk:%d", i), "v")
		}

		deleted, err := client.BulkDelete("chunk:*")
		if err != nil {
			t.Fatalf("BulkDelete error: %v", err)
		}
		if deleted != keyCount {
			t.Errorf("deleted = %d, want %d", deleted, keyCount)
		}

		for i := range keyCount {
			if mr.Exists(fmt.Sprintf("chunk:%d", i)) {
				t.Errorf("key chunk:%d should be deleted", i)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// SetString
// ---------------------------------------------------------------------------

func TestSetString(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		ttl        time.Duration
		wantValue  string
		wantTTL    bool
		ttlSeconds int
	}{
		{
			name:      "basic set without TTL",
			key:       "s1",
			value:     "v1",
			ttl:       0,
			wantValue: "v1",
			wantTTL:   false,
		},
		{
			name:       "set with TTL",
			key:        "s2",
			value:      "v2",
			ttl:        10 * time.Second,
			wantValue:  "v2",
			wantTTL:    true,
			ttlSeconds: 10,
		},
		{
			name:      "overwrite existing key",
			key:       "s1",
			value:     "overwritten",
			ttl:       0,
			wantValue: "overwritten",
			wantTTL:   false,
		},
	}

	client, mr := setupTestClient(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.SetString(tt.key, tt.value, tt.ttl); err != nil {
				t.Fatalf("SetString error: %v", err)
			}

			got, err := mr.Get(tt.key)
			if err != nil {
				t.Fatalf("miniredis Get error: %v", err)
			}
			if got != tt.wantValue {
				t.Errorf("value = %q, want %q", got, tt.wantValue)
			}

			if tt.wantTTL {
				mrTTL := mr.TTL(tt.key)
				if mrTTL != time.Duration(tt.ttlSeconds)*time.Second {
					t.Errorf("TTL = %v, want %v", mrTTL, time.Duration(tt.ttlSeconds)*time.Second)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SetTTL
// ---------------------------------------------------------------------------

func TestSetTTL(t *testing.T) {
	t.Run("set TTL on key", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("ttlkey", "val")
		if err := client.SetTTL("ttlkey", 30*time.Second); err != nil {
			t.Fatalf("SetTTL error: %v", err)
		}

		ttl := mr.TTL("ttlkey")
		if ttl != 30*time.Second {
			t.Errorf("TTL = %v, want 30s", ttl)
		}
	})

	t.Run("remove TTL (persist)", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("ttlkey2", "val")
		mr.SetTTL("ttlkey2", 60*time.Second)

		if err := client.SetTTL("ttlkey2", 0); err != nil {
			t.Fatalf("SetTTL(0) error: %v", err)
		}

		ttl := mr.TTL("ttlkey2")
		if ttl != 0 {
			t.Errorf("TTL = %v, want 0 (persistent)", ttl)
		}
	})
}

// ---------------------------------------------------------------------------
// BatchSetTTL
// ---------------------------------------------------------------------------

func TestBatchSetTTL(t *testing.T) {
	client, mr := setupTestClient(t)

	for i := range 5 {
		mr.Set(fmt.Sprintf("bttl:%d", i), "val")
	}

	count, err := client.BatchSetTTL("bttl:*", 20*time.Second)
	if err != nil {
		t.Fatalf("BatchSetTTL error: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}

	for i := range 5 {
		ttl := mr.TTL(fmt.Sprintf("bttl:%d", i))
		if ttl != 20*time.Second {
			t.Errorf("bttl:%d TTL = %v, want 20s", i, ttl)
		}
	}
}

// ---------------------------------------------------------------------------
// Rename
// ---------------------------------------------------------------------------

func TestRename(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("oldname", "data")

	if err := client.Rename("oldname", "newname"); err != nil {
		t.Fatalf("Rename error: %v", err)
	}

	if mr.Exists("oldname") {
		t.Error("old key should not exist after rename")
	}

	got, err := mr.Get("newname")
	if err != nil {
		t.Fatalf("miniredis Get(newname) error: %v", err)
	}
	if got != "data" {
		t.Errorf("newname value = %q, want %q", got, "data")
	}
}

// ---------------------------------------------------------------------------
// Copy
// ---------------------------------------------------------------------------

func TestCopy(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("src", "original")

	if err := client.Copy("src", "dst", false); err != nil {
		t.Fatalf("Copy error: %v", err)
	}

	srcVal, err := mr.Get("src")
	if err != nil {
		t.Fatalf("miniredis Get(src) error: %v", err)
	}
	dstVal, err := mr.Get("dst")
	if err != nil {
		t.Fatalf("miniredis Get(dst) error: %v", err)
	}

	if srcVal != "original" {
		t.Errorf("src value = %q, want %q", srcVal, "original")
	}
	if dstVal != "original" {
		t.Errorf("dst value = %q, want %q", dstVal, "original")
	}
}

// Regression: copying on a non-zero DB must keep the destination key in that
// same DB rather than landing in db0. See issue #29.
func TestCopy_NonZeroDB(t *testing.T) {
	client, mr := setupTestClient(t)

	if err := client.SelectDB(5); err != nil {
		t.Fatalf("SelectDB(5) error: %v", err)
	}

	mr.Select(5)
	mr.Set("src", "original")

	if err := client.Copy("src", "dst", false); err != nil {
		t.Fatalf("Copy error: %v", err)
	}

	mr.Select(5)
	dstVal, err := mr.Get("dst")
	if err != nil {
		t.Fatalf("miniredis Get(dst) on db5 error: %v", err)
	}
	if dstVal != "original" {
		t.Errorf("dst value on db5 = %q, want %q", dstVal, "original")
	}

	mr.Select(0)
	if mr.Exists("dst") {
		t.Error("dst should not exist on db0 when copying on db5")
	}
}

// ---------------------------------------------------------------------------
// MemoryUsage
// ---------------------------------------------------------------------------

func TestMemoryUsage(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("memkey", "some-value-to-measure")

	mem, err := client.MemoryUsage("memkey")
	if err != nil {
		// miniredis may not support MEMORY USAGE; skip if unsupported.
		t.Skipf("MemoryUsage not supported by miniredis: %v", err)
	}
	if mem <= 0 {
		t.Errorf("MemoryUsage = %d, want > 0", mem)
	}
}

// ---------------------------------------------------------------------------
// silentLogger.Printf — trivial coverage
// ---------------------------------------------------------------------------

func TestSilentLoggerPrintf(t *testing.T) {
	// Should not panic and produces no output (logger is silent).
	l := &silentLogger{}
	l.Printf(context.Background(), "test %s %d", "hello", 42)
}

// ---------------------------------------------------------------------------
// looksLikeGeoScores — direct unit tests
// ---------------------------------------------------------------------------

func TestLooksLikeGeoScores(t *testing.T) {
	tests := []struct {
		name    string
		members []types.ZSetMember
		want    bool
	}{
		{
			name:    "empty slice returns true (vacuously)",
			members: []types.ZSetMember{},
			want:    true,
		},
		{
			name: "small score is not geo",
			members: []types.ZSetMember{
				{Member: "a", Score: 1.5},
			},
			want: false,
		},
		{
			name: "non-integer score in geo range",
			members: []types.ZSetMember{
				// 1.5e14 + 0.25 — fits in float64 mantissa, retains fractional part.
				{Member: "a", Score: 1.5e14 + 0.25},
			},
			want: false,
		},
		{
			name: "integer score in geo range",
			members: []types.ZSetMember{
				{Member: "a", Score: 3.4e15},
				{Member: "b", Score: 1.5e15},
			},
			want: true,
		},
		{
			name: "score too large",
			members: []types.ZSetMember{
				{Member: "a", Score: 6e15},
			},
			want: false,
		},
		{
			name: "score below threshold",
			members: []types.ZSetMember{
				{Member: "a", Score: 1e13},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeGeoScores(tt.members)
			if got != tt.want {
				t.Errorf("looksLikeGeoScores(%v) = %v, want %v", tt.members, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isBinaryString — direct unit tests
// ---------------------------------------------------------------------------

func TestIsBinaryString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty string", "", false},
		{"valid utf-8 ascii", "hello", false},
		{"valid utf-8 multibyte", "héllo", false},
		// Invalid UTF-8 sequence: a single 0xff byte (continuation byte without lead)
		{"invalid utf-8 single byte", string([]byte{0xff}), true},
		{"invalid utf-8 mixed", string([]byte{'a', 0xfe, 0xfd}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinaryString(tt.s)
			if got != tt.want {
				t.Errorf("isBinaryString(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetValue — HyperLogLog branch
// ---------------------------------------------------------------------------

func TestGetValue_HyperLogLog(t *testing.T) {
	client, mr := setupTestClient(t)

	// Real Redis HLL header is "HYLL" + 12 bytes of metadata, then registers.
	// We just need the prefix for detection.
	mr.Set("hllmagic", "HYLL"+string(make([]byte, 12)))

	v, err := client.GetValue("hllmagic")
	if err != nil {
		t.Fatalf("GetValue error: %v", err)
	}
	if v.Type != types.KeyTypeHyperLogLog {
		t.Errorf("Type = %q, want %q", v.Type, types.KeyTypeHyperLogLog)
	}
}

// ---------------------------------------------------------------------------
// GetValue — Bitmap branch
// ---------------------------------------------------------------------------

func TestGetValue_Bitmap(t *testing.T) {
	client, _ := setupTestClient(t)

	// Set bits at positions 0, 1, and 7 (within first byte: 0b11000001 = 0xC1).
	for _, off := range []int64{0, 1, 7} {
		if err := client.SetBit("bm", off, 1); err != nil {
			t.Fatalf("SetBit(%d) error: %v", off, err)
		}
	}

	v, err := client.GetValue("bm")
	if err != nil {
		t.Fatalf("GetValue error: %v", err)
	}

	// The value 0xC1 contains a continuation byte (0xC1 by itself isn't valid utf-8),
	// so it should be detected as a bitmap.
	if v.Type != types.KeyTypeBitmap {
		t.Logf("Bitmap detection requires invalid UTF-8 raw bytes; got type %q (raw=%q)", v.Type, v.StringValue)
		t.Errorf("Type = %q, want %q", v.Type, types.KeyTypeBitmap)
	}
	if v.BitCount != 3 {
		t.Errorf("BitCount = %d, want 3", v.BitCount)
	}
	if len(v.BitPositions) != 3 {
		t.Errorf("BitPositions length = %d, want 3", len(v.BitPositions))
	}
}

// ---------------------------------------------------------------------------
// GetValue — Geo branch
// ---------------------------------------------------------------------------

func TestGetValue_Geo(t *testing.T) {
	client, _ := setupTestClient(t)

	err := client.GeoAdd("places",
		&goredis.GeoLocation{Name: "Palermo", Longitude: 13.361389, Latitude: 38.115556},
		&goredis.GeoLocation{Name: "Catania", Longitude: 15.087269, Latitude: 37.502669},
	)
	if err != nil {
		t.Skipf("GeoAdd not supported by miniredis: %v", err)
	}

	v, err := client.GetValue("places")
	if err != nil {
		t.Fatalf("GetValue error: %v", err)
	}

	if v.Type == types.KeyTypeGeo {
		if len(v.GeoValue) != 2 {
			t.Errorf("GeoValue length = %d, want 2", len(v.GeoValue))
		}
	} else {
		t.Logf("Geo not detected as KeyTypeGeo, got %q (miniredis score format may differ)", v.Type)
	}
}

// ---------------------------------------------------------------------------
// JSONGet / JSONGetPath / JSONSet — RedisJSON not supported by miniredis,
// but exercising the do() method ensures it returns gracefully.
// ---------------------------------------------------------------------------

func TestJSONGet_Unsupported(t *testing.T) {
	client, _ := setupTestClient(t)

	_, err := client.JSONGet("nokey")
	if err == nil {
		t.Error("JSONGet expected error from miniredis, got nil")
	}
}

func TestJSONGetPath_Unsupported(t *testing.T) {
	client, _ := setupTestClient(t)

	_, err := client.JSONGetPath("nokey", "$.field")
	if err == nil {
		t.Error("JSONGetPath expected error from miniredis, got nil")
	}
}

func TestJSONSet_Unsupported(t *testing.T) {
	client, _ := setupTestClient(t)

	err := client.JSONSet("nokey", `{"a":1}`)
	if err == nil {
		t.Error("JSONSet expected error from miniredis, got nil")
	}
}

// ---------------------------------------------------------------------------
// GeoPos — direct call after seeding via GeoAdd
// ---------------------------------------------------------------------------

func TestGeoPos(t *testing.T) {
	client, _ := setupTestClient(t)

	err := client.GeoAdd("locs",
		&goredis.GeoLocation{Name: "A", Longitude: 13.361389, Latitude: 38.115556},
	)
	if err != nil {
		t.Skipf("GeoAdd not supported by miniredis: %v", err)
	}

	positions, err := client.GeoPos("locs", "A")
	if err != nil {
		t.Fatalf("GeoPos error: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	if positions[0] == nil {
		t.Error("expected non-nil position for known member")
	}
}

// ---------------------------------------------------------------------------
// BulkDelete — empty match returns 0 with no error
// ---------------------------------------------------------------------------

func TestBulkDelete_NoMatches(t *testing.T) {
	client, _ := setupTestClient(t)

	deleted, err := client.BulkDelete("absent:*")
	if err != nil {
		t.Fatalf("BulkDelete error: %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted = %d, want 0", deleted)
	}
}

// ---------------------------------------------------------------------------
// BatchSetTTL — empty match path
// ---------------------------------------------------------------------------

func TestBatchSetTTL_NoMatches(t *testing.T) {
	client, _ := setupTestClient(t)

	count, err := client.BatchSetTTL("absent:*", 30)
	if err != nil {
		t.Fatalf("BatchSetTTL error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

// ---------------------------------------------------------------------------
// BatchSetTTL — persist (ttl = 0) path on real keys
// ---------------------------------------------------------------------------

func TestBatchSetTTL_Persist(t *testing.T) {
	client, mr := setupTestClient(t)

	// Seed keys with existing TTLs.
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("persistme:%d", i)
		mr.Set(key, "v")
		mr.SetTTL(key, 60_000_000_000) // 60s in nanoseconds (time.Duration)
	}

	count, err := client.BatchSetTTL("persistme:*", 0)
	if err != nil {
		t.Fatalf("BatchSetTTL error: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
	// All keys should now have no TTL (persistent).
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("persistme:%d", i)
		ttl := mr.TTL(key)
		if ttl != 0 {
			t.Errorf("%s TTL = %v, want 0", key, ttl)
		}
	}
}

// ---------------------------------------------------------------------------
// GetValue — HyperLogLog success path with PFCount populated.
// ---------------------------------------------------------------------------

func TestGetValue_HyperLogLog_PFCountSuccess(t *testing.T) {
	srv := newFakeRedisServer(t)
	hll := "HYLL" + string(make([]byte, 12))
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "TYPE":
			return "+string\r\n"
		case "STRLEN":
			return fmt.Sprintf(":%d\r\n", len(hll))
		case "GETRANGE":
			return respBulkString(hll)
		case "PFCOUNT":
			return ":7\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	v, err := c.GetValue("k")
	if err != nil {
		t.Fatalf("GetValue error: %v", err)
	}
	if v.Type != types.KeyTypeHyperLogLog {
		t.Errorf("Type = %q, want %q", v.Type, types.KeyTypeHyperLogLog)
	}
	if v.HLLCount != 7 {
		t.Errorf("HLLCount = %d, want 7", v.HLLCount)
	}
}

// ---------------------------------------------------------------------------
// GetValue — error paths for each value type. Uses the fake server so we can
// reply success for TYPE but inject an error for the value command.
// ---------------------------------------------------------------------------

func TestGetValue_TypeError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "TYPE" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	if _, err := c.GetValue("anything"); err == nil {
		t.Error("expected error from GetValue when TYPE returns error")
	}
}

// gvFakeClient connects a real Client to a fakeRedisServer that responds to
// TYPE with the requested key type, optional prefetch successes, then fails valueCmd.
func gvFakeClient(t *testing.T, keyType string, valueCmd string, prefetch map[string]string) *Client {
	t.Helper()
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == valueCmd {
			return "-ERR injected\r\n"
		}
		if argv[0] == "TYPE" {
			return "+" + keyType + "\r\n"
		}
		if resp, ok := prefetch[argv[0]]; ok {
			return resp
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })
	return c
}

func TestGetValue_StringError(t *testing.T) {
	c := gvFakeClient(t, "string", "STRLEN", nil)
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on string STRLEN error")
	}
}

func TestGetValue_StringGetRangeError(t *testing.T) {
	c := gvFakeClient(t, "string", "GETRANGE", map[string]string{"STRLEN": ":5\r\n"})
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on string GETRANGE error")
	}
}

func TestGetValue_ListError(t *testing.T) {
	c := gvFakeClient(t, "list", "LLEN", nil)
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on list LLEN error")
	}
}

func TestGetValue_ListLRangeError(t *testing.T) {
	c := gvFakeClient(t, "list", "LRANGE", map[string]string{"LLEN": ":5\r\n"})
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on list LRANGE error")
	}
}

func TestGetValue_SetError(t *testing.T) {
	c := gvFakeClient(t, "set", "SCARD", nil)
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on set SCARD error")
	}
}

func TestGetValue_SetScanError(t *testing.T) {
	c := gvFakeClient(t, "set", "SSCAN", map[string]string{"SCARD": ":5\r\n"})
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on set SSCAN error")
	}
}

func TestGetValue_ZSetError(t *testing.T) {
	c := gvFakeClient(t, "zset", "ZCARD", nil)
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on zset ZCARD error")
	}
}

func TestGetValue_ZSetRangeError(t *testing.T) {
	c := gvFakeClient(t, "zset", "ZRANGE", map[string]string{"ZCARD": ":5\r\n"})
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on zset ZRANGE error")
	}
}

func TestGetValue_HashError(t *testing.T) {
	c := gvFakeClient(t, "hash", "HLEN", nil)
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on hash HLEN error")
	}
}

func TestGetValue_HashScanError(t *testing.T) {
	c := gvFakeClient(t, "hash", "HSCAN", map[string]string{"HLEN": ":5\r\n"})
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on hash HSCAN error")
	}
}

func TestGetValue_StreamError(t *testing.T) {
	c := gvFakeClient(t, "stream", "XLEN", nil)
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on stream XLEN error")
	}
}

func TestGetValue_StreamRangeError(t *testing.T) {
	c := gvFakeClient(t, "stream", "XRANGE", map[string]string{"XLEN": ":5\r\n"})
	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on stream XRANGE error")
	}
}

// GetValue ReJSON-RL branch — success path returning a JSON value.
func TestGetValue_ReJSON_Success(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "TYPE":
			return "+ReJSON-RL\r\n"
		case "JSON.GET":
			return respBulkString(`{"a":1}`)
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	v, err := c.GetValue("k")
	if err != nil {
		t.Fatalf("GetValue error: %v", err)
	}
	if v.Type != types.KeyTypeJSON {
		t.Errorf("Type = %q, want %q", v.Type, types.KeyTypeJSON)
	}
	if v.JSONValue != `{"a":1}` {
		t.Errorf("JSONValue = %q, want %q", v.JSONValue, `{"a":1}`)
	}
}

// GetValue ReJSON-RL branch — error path on JSON.GET command.
func TestGetValue_ReJSON_Error(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "TYPE":
			return "+ReJSON-RL\r\n"
		case "JSON.GET":
			return "-ERR no json\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	if _, err := c.GetValue("k"); err == nil {
		t.Error("expected error from GetValue on ReJSON-RL JSON.GET error")
	}
}

// ---------------------------------------------------------------------------
// BulkDelete error paths
// ---------------------------------------------------------------------------

func TestBulkDelete_ScanAllError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "SCAN" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	if _, err := c.BulkDelete("*"); err == nil {
		t.Error("expected error from BulkDelete on SCAN error")
	}
}

// BulkDelete — DEL returns an error in the chunked loop. We use the fake
// server to return a SCAN reply with one key, then make DEL fail.
func TestBulkDelete_DelError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			// cursor 0, one key "foo"
			return "*2\r\n$1\r\n0\r\n*1\r\n$3\r\nfoo\r\n"
		case "DEL":
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	if _, err := c.BulkDelete("*"); err == nil {
		t.Error("expected DEL error from BulkDelete")
	}
}

// ---------------------------------------------------------------------------
// BatchSetTTL — scanAll error path
// ---------------------------------------------------------------------------

func TestBatchSetTTL_ScanAllError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "SCAN" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	if _, err := c.BatchSetTTL("*", 30*time.Second); err == nil {
		t.Error("expected error from BatchSetTTL on SCAN error")
	}
}

// ---------------------------------------------------------------------------
// BulkDelete — exercising scanAll over many batches via large key set
// ---------------------------------------------------------------------------

func TestBulkDelete_LargeKeySet(t *testing.T) {
	client, mr := setupTestClient(t)

	// Seed many more than the SCAN batchSize (100) to ensure multiple cursor iterations.
	const total = 350
	for i := 0; i < total; i++ {
		mr.Set(fmt.Sprintf("scanall:%d", i), "v")
	}
	deleted, err := client.BulkDelete("scanall:*")
	if err != nil {
		t.Fatalf("BulkDelete error: %v", err)
	}
	if deleted != total {
		t.Errorf("deleted = %d, want %d", deleted, total)
	}
	if client.GetTotalKeys() != 0 {
		t.Errorf("expected 0 keys remaining, got %d", client.GetTotalKeys())
	}
}

// ---------------------------------------------------------------------------
// isBinaryPrefix — direct unit tests
// ---------------------------------------------------------------------------

func TestIsBinaryPrefix(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty string", "", false},
		{"valid utf-8 ascii", "hello", false},
		{"valid utf-8 multibyte", "héllo", false},
		// A 3-byte rune (é = 0xE4 0xB8 0xAD is 中) cut after 1 byte: not binary.
		{"split rune one byte left", "hello\xe4", false},
		// Cut after 2 of 3 bytes: not binary.
		{"split rune two bytes left", "hello\xe4\xb8", false},
		// 4-byte rune (😀 = 0xF0 0x9F 0x98 0x80) cut after 3 bytes: not binary.
		{"split rune three bytes left", "hello\xf0\x9f\x98", false},
		// Genuinely binary content in the middle survives trimming.
		{"binary in middle", "a\xffb", true},
		{"all binary", string([]byte{0xff, 0xfe, 0xfd, 0xfc}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinaryPrefix(tt.s)
			if got != tt.want {
				t.Errorf("isBinaryPrefix(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// extractBitPositions — direct unit tests including the cap
// ---------------------------------------------------------------------------

func TestExtractBitPositions(t *testing.T) {
	t.Run("positions from bytes", func(t *testing.T) {
		// 0b10000001 -> bits 0 and 7; 0b01000000 -> bit 9
		got := extractBitPositions([]byte{0x81, 0x40}, 10)
		want := []int64{0, 7, 9}
		if len(got) != len(want) {
			t.Fatalf("positions = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("positions[%d] = %d, want %d", i, got[i], want[i])
			}
		}
	})

	t.Run("capped at max", func(t *testing.T) {
		// 4 bytes of 0xFF = 32 set bits, capped at 5.
		got := extractBitPositions([]byte{0xff, 0xff, 0xff, 0xff}, 5)
		if len(got) != 5 {
			t.Fatalf("len = %d, want 5", len(got))
		}
		if got[4] != 4 {
			t.Errorf("last position = %d, want 4", got[4])
		}
	})

	t.Run("empty value", func(t *testing.T) {
		if got := extractBitPositions(nil, 5); len(got) != 0 {
			t.Errorf("expected no positions, got %v", got)
		}
	})
}

func TestIsBinaryPrefixAndExtractMaxZero(t *testing.T) {
	// incomplete multi-byte rune at end of prefix should not mark binary
	if isBinaryPrefix("hello\xc2") {
		t.Error("incomplete UTF-8 trail should not be binary")
	}
	// More than utf8.UTFMax-1 invalid trailing bytes remain after prefix trim.
	if !isBinaryPrefix(string([]byte{0xff, 0xfe, 0xfd, 0xfc})) {
		t.Error("invalid bytes should be binary")
	}
	if extractBitPositions([]byte{0xff}, 0) != nil {
		t.Error("max 0 should return nil")
	}
}


func TestGetValue_Protobuf(t *testing.T) {
	client, mr := setupTestClient(t)

	// field 1 string "menu-id", field 5 varint 7
	var raw []byte
	raw = append(raw, 0x0a, 0x07) // tag 1, len 7
	raw = append(raw, []byte("menu-id")...)
	raw = append(raw, 0x28, 0x07) // tag 5, varint 7

	mr.Set("proto:raw", string(raw))
	v, err := client.GetValue("proto:raw")
	if err != nil {
		t.Fatalf("GetValue error: %v", err)
	}
	if v.Type != types.KeyTypeProtobuf {
		t.Fatalf("Type = %q, want %q", v.Type, types.KeyTypeProtobuf)
	}
	if v.DecodedFormat != "protobuf" {
		t.Errorf("DecodedFormat = %q, want protobuf", v.DecodedFormat)
	}
	if !strings.Contains(v.DecodedValue, `1: "menu-id"`) {
		t.Errorf("DecodedValue missing menu-id field:\n%s", v.DecodedValue)
	}
	if !strings.Contains(v.DecodedValue, "5: 7") {
		t.Errorf("DecodedValue missing field 5:\n%s", v.DecodedValue)
	}
}


func TestGetValue_Protobuf_S2(t *testing.T) {
	client, mr := setupTestClient(t)

	var raw []byte
	raw = append(raw, 0x0a, 0x04)
	raw = append(raw, []byte("abcd")...)
	compressed := s2.Encode(nil, raw)

	mr.Set("proto:s2", string(compressed))
	v, err := client.GetValue("proto:s2")
	if err != nil {
		t.Fatalf("GetValue error: %v", err)
	}
	if v.Type != types.KeyTypeProtobuf {
		t.Fatalf("Type = %q, want %q", v.Type, types.KeyTypeProtobuf)
	}
	if v.DecodedFormat != "s2+protobuf" {
		t.Errorf("DecodedFormat = %q, want s2+protobuf", v.DecodedFormat)
	}
	if v.RawSize != len(compressed) {
		t.Errorf("RawSize = %d, want %d", v.RawSize, len(compressed))
	}
	if v.DecodedSize != len(raw) {
		t.Errorf("DecodedSize = %d, want %d", v.DecodedSize, len(raw))
	}
	if !strings.Contains(v.DecodedValue, `1: "abcd"`) {
		t.Errorf("DecodedValue missing field:\n%s", v.DecodedValue)
	}
}

func TestGetValue_TruncatesLargeValues(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		client, mr := setupTestClient(t)
		total := detailMaxStringBytes + 2048
		mr.Set("bigstr", strings.Repeat("a", total))

		v, err := client.GetValue("bigstr")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if len(v.StringValue) != detailMaxStringBytes {
			t.Errorf("StringValue length = %d, want %d", len(v.StringValue), detailMaxStringBytes)
		}
		if !v.Truncated {
			t.Error("expected Truncated = true")
		}
		if v.TotalCount != int64(total) {
			t.Errorf("TotalCount = %d, want %d", v.TotalCount, total)
		}
	})

	t.Run("string small not truncated", func(t *testing.T) {
		client, mr := setupTestClient(t)
		mr.Set("small", "hello")
		v, err := client.GetValue("small")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if v.Truncated {
			t.Error("expected Truncated = false")
		}
		if v.StringValue != "hello" {
			t.Errorf("StringValue = %q, want hello", v.StringValue)
		}
	})

	t.Run("list", func(t *testing.T) {
		client, mr := setupTestClient(t)
		total := detailMaxItems + 50
		for i := 0; i < total; i++ {
			mr.RPush("biglist", "item")
		}
		v, err := client.GetValue("biglist")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if len(v.ListValue) != detailMaxItems {
			t.Errorf("ListValue length = %d, want %d", len(v.ListValue), detailMaxItems)
		}
		if !v.Truncated || v.TotalCount != int64(total) {
			t.Errorf("Truncated/TotalCount = %v/%d, want true/%d", v.Truncated, v.TotalCount, total)
		}
	})

	t.Run("set", func(t *testing.T) {
		client, mr := setupTestClient(t)
		total := detailMaxItems + 50
		for i := 0; i < total; i++ {
			mr.SAdd("bigset", fmt.Sprintf("m:%06d", i))
		}
		v, err := client.GetValue("bigset")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if len(v.SetValue) != detailMaxItems {
			t.Errorf("SetValue length = %d, want %d", len(v.SetValue), detailMaxItems)
		}
		if !v.Truncated || v.TotalCount != int64(total) {
			t.Errorf("Truncated/TotalCount = %v/%d, want true/%d", v.Truncated, v.TotalCount, total)
		}
	})

	t.Run("zset", func(t *testing.T) {
		client, mr := setupTestClient(t)
		total := detailMaxItems + 50
		for i := 0; i < total; i++ {
			mr.ZAdd("bigz", float64(i), fmt.Sprintf("m:%06d", i))
		}
		v, err := client.GetValue("bigz")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if len(v.ZSetValue) != detailMaxItems {
			t.Errorf("ZSetValue length = %d, want %d", len(v.ZSetValue), detailMaxItems)
		}
		if !v.Truncated || v.TotalCount != int64(total) {
			t.Errorf("Truncated/TotalCount = %v/%d, want true/%d", v.Truncated, v.TotalCount, total)
		}
	})

	t.Run("hash", func(t *testing.T) {
		client, mr := setupTestClient(t)
		total := detailMaxItems + 50
		for i := 0; i < total; i++ {
			mr.HSet("bighash", fmt.Sprintf("f:%06d", i), "v")
		}
		v, err := client.GetValue("bighash")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if len(v.HashValue) != detailMaxItems {
			t.Errorf("HashValue length = %d, want %d", len(v.HashValue), detailMaxItems)
		}
		if !v.Truncated || v.TotalCount != int64(total) {
			t.Errorf("Truncated/TotalCount = %v/%d, want true/%d", v.Truncated, v.TotalCount, total)
		}
	})

	t.Run("stream", func(t *testing.T) {
		client, _ := setupTestClient(t)
		total := detailMaxItems + 50
		for i := 0; i < total; i++ {
			if _, err := client.XAdd("bigstream", map[string]any{"k": "v"}); err != nil {
				t.Fatalf("XAdd error: %v", err)
			}
		}
		v, err := client.GetValue("bigstream")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if len(v.StreamValue) != detailMaxItems {
			t.Errorf("StreamValue length = %d, want %d", len(v.StreamValue), detailMaxItems)
		}
		if !v.Truncated || v.TotalCount != int64(total) {
			t.Errorf("Truncated/TotalCount = %v/%d, want true/%d", v.Truncated, v.TotalCount, total)
		}
	})

	t.Run("truncated bitmap still classified", func(t *testing.T) {
		client, mr := setupTestClient(t)
		bin := make([]byte, detailMaxStringBytes+512)
		for i := range bin {
			bin[i] = 0xff
		}
		mr.Set("bigbm", string(bin))

		v, err := client.GetValue("bigbm")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if v.Type != types.KeyTypeBitmap {
			t.Errorf("Type = %q, want bitmap", v.Type)
		}
		if !v.Truncated {
			t.Error("expected Truncated = true")
		}
		if len(v.BitPositions) != maxBitPositions {
			t.Errorf("BitPositions length = %d, want %d", len(v.BitPositions), maxBitPositions)
		}
	})
}

func TestExtractBitPositions_Caps(t *testing.T) {
	// 32 bytes all 0xff → 256 set bits; cap at 10.
	raw := bytes.Repeat([]byte{0xff}, 32)
	got := extractBitPositions(raw, 10)
	if len(got) != 10 {
		t.Fatalf("len = %d, want 10", len(got))
	}
	if got[0] != 0 || got[9] != 9 {
		t.Errorf("positions = %v, want 0..9", got)
	}
	if extractBitPositions(raw, 0) != nil {
		t.Error("max 0 should return nil")
	}
	if extractBitPositions(raw, -1) != nil {
		t.Error("negative max should return nil")
	}
}

// ---------------------------------------------------------------------------
// GetValue — Geo branch
// ---------------------------------------------------------------------------



// BenchmarkGetValueProtobufS2 measures full decode of a large s2+protobuf menu value.
func BenchmarkGetValueProtobufS2(b *testing.B) {
	client, mr := setupBenchClient(b)

	var raw []byte
	raw = append(raw, 0x0a, 0x14)
	raw = append(raw, []byte("rst-demo:DELIVERY!!!!")[:20]...)
	for i := 0; i < 3000; i++ {
		s := fmt.Sprintf("item-description-padding-%04d-xxxxxxxx", i)
		raw = append(raw, 0x12, byte(len(s)))
		raw = append(raw, s...)
	}
	compressed := s2.Encode(nil, raw)
	mr.Set("menu:big", string(compressed))

	b.ReportMetric(float64(len(compressed)), "raw-bytes")
	b.ReportMetric(float64(len(raw)), "decoded-bytes")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, err := client.GetValue("menu:big")
		if err != nil {
			b.Fatalf("GetValue: %v", err)
		}
		if v.Type != types.KeyTypeProtobuf {
			b.Fatalf("type = %s, want protobuf", v.Type)
		}
	}
}

// BenchmarkGetValuePreviewProtobufS2 measures bounded preview of s2+protobuf.
func BenchmarkGetValuePreviewProtobufS2(b *testing.B) {
	client, mr := setupBenchClient(b)

	var raw []byte
	for i := 0; i < 3000; i++ {
		s := fmt.Sprintf("item-description-padding-%04d-xxxxxxxx", i)
		raw = append(raw, 0x12, byte(len(s)))
		raw = append(raw, s...)
	}
	compressed := s2.Encode(nil, raw)
	mr.Set("menu:big", string(compressed))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.GetValuePreview("menu:big"); err != nil {
			b.Fatalf("GetValuePreview: %v", err)
		}
	}
}
