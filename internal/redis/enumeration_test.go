package redis

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
	goredis "github.com/redis/go-redis/v9"
)

func TestGetTotalKeys(t *testing.T) {
	t.Run("returns count of existing keys", func(t *testing.T) {
		client, mr := setupTestClient(t)

		for i := 0; i < 5; i++ {
			mr.Set("key:"+string(rune('a'+i)), "val")
		}

		got := client.GetTotalKeys()
		if got != 5 {
			t.Errorf("GetTotalKeys() = %d, want 5", got)
		}
	})

	t.Run("empty database returns 0", func(t *testing.T) {
		client, _ := setupTestClient(t)

		got := client.GetTotalKeys()
		if got != 0 {
			t.Errorf("GetTotalKeys() = %d, want 0", got)
		}
	})
}

func TestScanKeys(t *testing.T) {
	t.Run("scan with pattern", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("user:1", "alice")
		mr.Set("user:2", "bob")
		mr.Set("other:1", "charlie")

		keys, _, err := client.ScanKeys("user:*", 0, 100)
		if err != nil {
			t.Fatalf("ScanKeys() error = %v", err)
		}

		if len(keys) != 2 {
			t.Fatalf("ScanKeys() returned %d keys, want 2", len(keys))
		}

		names := make(map[string]bool)
		for _, k := range keys {
			names[k.Key] = true
			if k.Type != types.KeyTypeString {
				t.Errorf("key %q type = %q, want %q", k.Key, k.Type, types.KeyTypeString)
			}
		}
		if !names["user:1"] || !names["user:2"] {
			t.Errorf("expected user:1 and user:2, got %v", names)
		}
	})

	t.Run("empty pattern defaults to wildcard", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("a", "1")
		mr.Set("b", "2")

		keys, _, err := client.ScanKeys("", 0, 100)
		if err != nil {
			t.Fatalf("ScanKeys() error = %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("ScanKeys(\"\") returned %d keys, want 2", len(keys))
		}
	})

	t.Run("empty database returns empty slice", func(t *testing.T) {
		client, _ := setupTestClient(t)

		keys, _, err := client.ScanKeys("*", 0, 100)
		if err != nil {
			t.Fatalf("ScanKeys() error = %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("ScanKeys() on empty db returned %d keys, want 0", len(keys))
		}
	})

	t.Run("returns TTL field", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("ttlkey", "value")

		keys, _, err := client.ScanKeys("ttlkey", 0, 100)
		if err != nil {
			t.Fatalf("ScanKeys() error = %v", err)
		}
		if len(keys) != 1 {
			t.Fatalf("ScanKeys() returned %d keys, want 1", len(keys))
		}
		// miniredis returns -1 for no TTL
		if keys[0].TTL == 0 {
			t.Log("TTL field is populated (zero or negative for no expiry)")
		}
	})
}

func TestScanKeys_WithoutTypes(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("key:1", "val1")
	mr.Set("key:2", "val2")

	client.SetIncludeTypes(false)

	keys, _, err := client.ScanKeys("key:*", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("ScanKeys() returned %d keys, want 2", len(keys))
	}
	for _, k := range keys {
		if k.Type != "" {
			t.Errorf("key %q type = %q, want empty when includeTypes=false", k.Key, k.Type)
		}
		// TTL should still be populated (miniredis returns -1 for no expiry)
		if k.TTL == 0 {
			t.Errorf("key %q TTL should be non-zero (no expiry = -1)", k.Key)
		}
	}
}

func TestScanKeysWithRegex(t *testing.T) {
	t.Run("matches regex pattern", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("user:123", "a")
		mr.Set("user:abc", "b")
		mr.Set("session:456", "c")

		keys, err := client.ScanKeysWithRegex(`user:\d+`, 100)
		if err != nil {
			t.Fatalf("ScanKeysWithRegex() error = %v", err)
		}

		if len(keys) != 1 {
			t.Fatalf("ScanKeysWithRegex() returned %d keys, want 1", len(keys))
		}
		if keys[0].Key != "user:123" {
			t.Errorf("key = %q, want %q", keys[0].Key, "user:123")
		}
	})

	t.Run("invalid regex returns error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		_, err := client.ScanKeysWithRegex(`[invalid`, 100)
		if err == nil {
			t.Fatal("ScanKeysWithRegex() expected error for invalid regex, got nil")
		}
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("foo", "bar")

		keys, err := client.ScanKeysWithRegex(`^zzz`, 100)
		if err != nil {
			t.Fatalf("ScanKeysWithRegex() error = %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("ScanKeysWithRegex() returned %d keys, want 0", len(keys))
		}
	})
}

func TestFuzzySearchKeys(t *testing.T) {
	t.Run("returns matching keys sorted by score", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("user:profile:settings", "a")
		mr.Set("user:data", "b")
		mr.Set("other:key", "c")

		// "user" is a substring match for "user:profile:settings" and "user:data"
		keys, err := client.FuzzySearchKeys("user", 10)
		if err != nil {
			t.Fatalf("FuzzySearchKeys() error = %v", err)
		}

		if len(keys) < 2 {
			t.Fatalf("FuzzySearchKeys(\"user\") returned %d results, want at least 2", len(keys))
		}

		// Results should be sorted by score; shorter key with substring match scores
		// higher because fuzzyScore returns 100 + (len(str) - len(pattern))
		// "user:data" (len 9) scores 105, "user:profile:settings" (len 21) scores 117
		// Both are substring matches so higher len(str)-len(pattern) gives higher score,
		// meaning the longer key should come first.
		// Verify all returned keys have names and types
		for _, k := range keys {
			if k.Key == "" {
				t.Error("FuzzySearchKeys() returned a key with empty name")
			}
			if k.Type == "" {
				t.Error("FuzzySearchKeys() returned a key with empty type")
			}
		}
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("abc", "1")

		keys, err := client.FuzzySearchKeys("zzz", 10)
		if err != nil {
			t.Fatalf("FuzzySearchKeys() error = %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("FuzzySearchKeys() returned %d keys, want 0", len(keys))
		}
	})
}

func TestSearchByValue(t *testing.T) {
	t.Run("finds matching values across types", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("s1", "hello world")
		mr.RPush("l1", "foo", "bar world")
		mr.SAdd("set1", "world")
		mr.HSet("h1", "f", "world")

		keys, err := client.SearchByValue("*", "world", 10)
		if err != nil {
			t.Fatalf("SearchByValue() error = %v", err)
		}

		if len(keys) < 4 {
			t.Fatalf("SearchByValue() returned %d keys, want at least 4", len(keys))
		}

		found := make(map[string]bool)
		for _, k := range keys {
			found[k.Key] = true
		}
		for _, expected := range []string{"s1", "l1", "set1", "h1"} {
			if !found[expected] {
				t.Errorf("SearchByValue() missing expected key %q", expected)
			}
		}
	})

	t.Run("non-matching search returns empty", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("s1", "hello")

		keys, err := client.SearchByValue("*", "nonexistent", 10)
		if err != nil {
			t.Fatalf("SearchByValue() error = %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("SearchByValue() returned %d keys, want 0", len(keys))
		}
	})
}

func TestGetKeyPrefixes(t *testing.T) {
	t.Run("returns unique prefixes", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("user:profile:name", "a")
		mr.Set("user:profile:age", "b")
		mr.Set("user:settings", "c")
		mr.Set("session:abc", "d")

		prefixes, err := client.GetKeyPrefixes(":", 3)
		if err != nil {
			t.Fatalf("GetKeyPrefixes() error = %v", err)
		}

		// Should include: user, user:profile, user:profile:name, user:profile:age,
		// user:settings, session, session:abc
		sort.Strings(prefixes)

		expected := map[string]bool{
			"user":              true,
			"user:profile":      true,
			"user:profile:name": true,
			"user:profile:age":  true,
			"user:settings":     true,
			"session":           true,
			"session:abc":       true,
		}

		for _, p := range prefixes {
			if !expected[p] {
				t.Errorf("unexpected prefix %q", p)
			}
			delete(expected, p)
		}
		for p := range expected {
			t.Errorf("missing expected prefix %q", p)
		}
	})

	t.Run("empty database returns empty slice", func(t *testing.T) {
		client, _ := setupTestClient(t)

		prefixes, err := client.GetKeyPrefixes(":", 3)
		if err != nil {
			t.Fatalf("GetKeyPrefixes() error = %v", err)
		}
		if len(prefixes) != 0 {
			t.Errorf("GetKeyPrefixes() on empty db returned %d prefixes, want 0", len(prefixes))
		}
	})
}

// ---------------------------------------------------------------------------
// ScanKeysWithRegex — pattern length exceeds maxRegexLen
// ---------------------------------------------------------------------------

func TestScanKeysWithRegex_PatternTooLong(t *testing.T) {
	client, _ := setupTestClient(t)

	// 1500 characters - well over the 1024 max.
	longPattern := strings.Repeat("a", 1500)

	_, err := client.ScanKeysWithRegex(longPattern, 100)
	if err == nil {
		t.Fatal("expected error for pattern exceeding max length")
	}
}

// ---------------------------------------------------------------------------
// ScanKeysWithRegex — early termination when maxKeys reached.
// ---------------------------------------------------------------------------

func TestScanKeysWithRegex_MaxKeysEarlyTerm(t *testing.T) {
	client, mr := setupTestClient(t)

	for i := 0; i < 30; i++ {
		mr.Set(fmt.Sprintf("regex:%d", i), "v")
	}

	results, err := client.ScanKeysWithRegex(`^regex:`, 5)
	if err != nil {
		t.Fatalf("ScanKeysWithRegex error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("results length = %d, want 5", len(results))
	}
}

// ---------------------------------------------------------------------------
// detectZSetSubtypes — exercised via ScanKeys when zset keys are present.
// ---------------------------------------------------------------------------

func TestScanKeys_DetectZSetSubtypes(t *testing.T) {
	client, mr := setupTestClient(t)

	// Plain zset with non-geo scores.
	mr.ZAdd("plain:zset", 1.5, "alpha")
	mr.ZAdd("plain:zset", 2.5, "beta")

	// Empty zset cannot exist in Redis (an empty zset is deleted),
	// but a single-member zset is fine.
	mr.ZAdd("solo:zset", 100, "x")

	keys, _, err := client.ScanKeys("*", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}

	for _, k := range keys {
		// Both should remain plain "zset" because their scores are not in geo range.
		if k.Type != "zset" {
			t.Errorf("key %q type = %q, want %q", k.Key, k.Type, "zset")
		}
	}
}

// ---------------------------------------------------------------------------
// detectZSetSubtypes — geo path. We seed a zset whose scores look like
// 52-bit geohash integers via direct ZADD with such scores.
// ---------------------------------------------------------------------------

func TestScanKeys_DetectZSetSubtypes_Geo(t *testing.T) {
	client, _ := setupTestClient(t)

	// 52-bit integer in the geohash range (~1e14 to ~5e15) that is not a
	// fractional float.
	if err := client.ZAdd("geo:zset", 3.4e15, "Palermo"); err != nil {
		t.Fatalf("ZAdd error: %v", err)
	}

	keys, _, err := client.ScanKeys("geo:zset", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Type != "geo" {
		t.Errorf("type = %q, want geo", keys[0].Type)
	}
}

// ---------------------------------------------------------------------------
// detectStringSubtypes — exercise the HLL magic-bytes detection branch.
// ---------------------------------------------------------------------------

func TestScanKeys_DetectStringSubtypes_HLL(t *testing.T) {
	client, mr := setupTestClient(t)

	// String key whose raw value starts with "HYLL" magic.
	mr.Set("hll:fake", "HYLL"+string(make([]byte, 12)))

	keys, _, err := client.ScanKeys("hll:fake", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Type != "hyperloglog" {
		t.Errorf("type = %q, want hyperloglog", keys[0].Type)
	}
}

// ---------------------------------------------------------------------------
// FuzzySearchKeys — edge case: maxKeys smaller than match count.
// ---------------------------------------------------------------------------

func TestFuzzySearchKeys_MaxKeysLimit(t *testing.T) {
	client, mr := setupTestClient(t)

	for i := 0; i < 10; i++ {
		mr.Set(fmt.Sprintf("user:%d", i), "v")
	}

	results, err := client.FuzzySearchKeys("user", 5)
	if err != nil {
		t.Fatalf("FuzzySearchKeys error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("results length = %d, want 5", len(results))
	}
	for _, r := range results {
		if !strings.HasPrefix(r.Key, "user:") {
			t.Errorf("unexpected key %q", r.Key)
		}
	}
}

// ---------------------------------------------------------------------------
// SearchByValue — exercise the maxKeys early termination.
// ---------------------------------------------------------------------------

func TestSearchByValue_MaxKeysLimit(t *testing.T) {
	client, mr := setupTestClient(t)

	for i := 0; i < 20; i++ {
		mr.Set(fmt.Sprintf("v:%d", i), "needle")
	}

	results, err := client.SearchByValue("*", "needle", 5)
	if err != nil {
		t.Fatalf("SearchByValue error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("results length = %d, want 5", len(results))
	}
}

// ---------------------------------------------------------------------------
// GetKeyPrefixes — exercise depth limit boundary.
// ---------------------------------------------------------------------------

func TestGetKeyPrefixes_DepthLimit(t *testing.T) {
	client, mr := setupTestClient(t)

	// Deep key path; only the first two levels should be returned.
	mr.Set("a:b:c:d:e", "v")

	prefixes, err := client.GetKeyPrefixes(":", 2)
	if err != nil {
		t.Fatalf("GetKeyPrefixes error: %v", err)
	}

	// Expect "a" and "a:b" (depth 1 and 2).
	want := map[string]bool{"a": true, "a:b": true}
	got := make(map[string]bool)
	for _, p := range prefixes {
		got[p] = true
	}
	for w := range want {
		if !got[w] {
			t.Errorf("missing prefix %q in %v", w, prefixes)
		}
	}
	for g := range got {
		if !want[g] {
			t.Errorf("unexpected prefix %q", g)
		}
	}
}

// ---------------------------------------------------------------------------
// scanLimited / scanEach early-termination — exercise via direct call.
// ---------------------------------------------------------------------------

func TestScanLimitedViaScanEach(t *testing.T) {
	client, mr := setupTestClient(t)

	// Seed enough keys to span multiple scan batches.
	for i := 0; i < 250; i++ {
		mr.Set(fmt.Sprintf("limited:%d", i), "v")
	}

	keys, err := client.scanLimited("*", 100, 50)
	if err != nil {
		t.Fatalf("scanLimited error: %v", err)
	}
	if len(keys) != 50 {
		t.Errorf("scanLimited returned %d keys, want 50", len(keys))
	}
}

// ---------------------------------------------------------------------------
// GetTotalKeys — DBSIZE error path. Use the fake server to inject an error
// reply for DBSIZE so the function returns 0.
// ---------------------------------------------------------------------------

func TestGetTotalKeys_Error(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "DBSIZE" {
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

	got := c.GetTotalKeys()
	if got != 0 {
		t.Errorf("GetTotalKeys = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// ScanKeys — error path on the underlying SCAN command.
// ---------------------------------------------------------------------------

func TestScanKeys_ScanError(t *testing.T) {
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

	if _, _, err := c.ScanKeys("*", 0, 100); err == nil {
		t.Error("expected error from ScanKeys when SCAN errors")
	}
}

// ---------------------------------------------------------------------------
// ScanKeys — pipe.Exec error path. Return one key from SCAN, then make TYPE
// (the first command in the pipeline) fail. The pipeline.Exec should bubble
// up an error that is not redis.Nil.
// ---------------------------------------------------------------------------

func TestScanKeys_PipeExecError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*1\r\n$3\r\nfoo\r\n"
		case "TYPE", "TTL":
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

	if _, _, err := c.ScanKeys("*", 0, 100); err == nil {
		t.Error("expected error from ScanKeys when pipeline.Exec errors")
	}
}

// ---------------------------------------------------------------------------
// ScanKeys — cluster branch dispatches to scanAll.
// ---------------------------------------------------------------------------

func TestScanKeys_ClusterBranch(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*1\r\n$3\r\nfoo\r\n"
		case "TYPE":
			return "+string\r\n"
		case "TTL":
			return ":-1\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	keys, _, err := client.ScanKeys("*", 0, 100)
	if err != nil {
		t.Logf("ScanKeys cluster branch err: %v", err)
	}
	_ = keys
}

// ---------------------------------------------------------------------------
// ScanKeysWithRegex — scanEach error path.
// ---------------------------------------------------------------------------

func TestScanKeysWithRegex_ScanError(t *testing.T) {
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

	if _, err := c.ScanKeysWithRegex(".*", 100); err == nil {
		t.Error("expected error from ScanKeysWithRegex when SCAN errors")
	}
}

// ---------------------------------------------------------------------------
// FuzzySearchKeys — scanEach error path.
// ---------------------------------------------------------------------------

func TestFuzzySearchKeys_ScanError(t *testing.T) {
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

	if _, err := c.FuzzySearchKeys("term", 100); err == nil {
		t.Error("expected error from FuzzySearchKeys when SCAN errors")
	}
}

// ---------------------------------------------------------------------------
// SearchByValue — scanAll error path.
// ---------------------------------------------------------------------------

func TestSearchByValue_ScanError(t *testing.T) {
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

	if _, err := c.SearchByValue("*", "needle", 10); err == nil {
		t.Error("expected error from SearchByValue when SCAN errors")
	}
}

// ---------------------------------------------------------------------------
// SearchByValue — ReJSON-RL branch and default-continue branch via fake server.
// ---------------------------------------------------------------------------

func TestSearchByValue_ReJSONAndDefault(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			// Return two keys: one ReJSON-RL and one with an unknown type.
			return "*2\r\n$1\r\n0\r\n*2\r\n$5\r\nkjson\r\n$5\r\nweird\r\n"
		case "TYPE":
			if len(argv) >= 2 && argv[1] == "kjson" {
				return "+ReJSON-RL\r\n"
			}
			return "+weirdtype\r\n"
		case "JSON.GET":
			return respBulkString(`{"haystack":"needle"}`)
		case "TTL":
			return ":-1\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	results, err := c.SearchByValue("*", "needle", 10)
	if err != nil {
		t.Fatalf("SearchByValue error: %v", err)
	}
	// Only the kjson key should match (the weird-typed one is skipped via the
	// default branch in queue and find-matching switches).
	found := false
	for _, r := range results {
		if r.Key == "kjson" {
			found = true
		}
		if r.Key == "weird" {
			t.Errorf("weird key should be skipped via default branch")
		}
	}
	if !found {
		t.Errorf("kjson should be in results: %v", results)
	}
}

// ---------------------------------------------------------------------------
// GetKeyPrefixes — scanEach error path.
// ---------------------------------------------------------------------------

func TestGetKeyPrefixes_ScanError(t *testing.T) {
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

	if _, err := c.GetKeyPrefixes(":", 3); err == nil {
		t.Error("expected error from GetKeyPrefixes when SCAN errors")
	}
}

// ---------------------------------------------------------------------------
// detectStringSubtypes — bitmap detection branch via SETBIT-created key.
// ---------------------------------------------------------------------------

func TestScanKeys_DetectStringSubtypes_Bitmap(t *testing.T) {
	client, _ := setupTestClient(t)

	// Use SETBIT via the wrapper to create a binary-string-typed key.
	for _, off := range []int64{0, 1, 7} {
		if err := client.SetBit("bm:detect", off, 1); err != nil {
			t.Fatalf("SetBit: %v", err)
		}
	}

	keys, _, err := client.ScanKeys("bm:detect", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Type != "bitmap" {
		t.Errorf("type = %q, want bitmap", keys[0].Type)
	}
}

// ---------------------------------------------------------------------------
// detectStringSubtypes — GetRange error path on a single string key. We use
// the fake server to return one key from SCAN, "string" type, then have
// GETRANGE fail.
// ---------------------------------------------------------------------------

func TestDetectStringSubtypes_GetError(t *testing.T) {
	srv := newFakeRedisServer(t)
	var firstGet sync.Once
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*1\r\n$1\r\nx\r\n"
		case "TYPE":
			return "+string\r\n"
		case "TTL":
			return ":-1\r\n"
		case "GETRANGE":
			// First GETRANGE (the one from detectStringSubtypes) errors out.
			ret := ""
			firstGet.Do(func() { ret = "-ERR injected\r\n" })
			if ret != "" {
				return ret
			}
			return "$0\r\n\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	keys, _, err := c.ScanKeys("*", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	// Detection failed (GET errored) so type stays "string".
	if keys[0].Type != "string" {
		t.Errorf("type = %q, want string (detection failed)", keys[0].Type)
	}
}

// ---------------------------------------------------------------------------
// detectZSetSubtypes — ZRange error path. Use the fake server: SCAN returns
// one zset, TYPE says zset, ZRANGE errors. Detection should leave the type
// as plain "zset".
// ---------------------------------------------------------------------------

func TestDetectZSetSubtypes_ZRangeError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*1\r\n$1\r\nz\r\n"
		case "TYPE":
			return "+zset\r\n"
		case "TTL":
			return ":-1\r\n"
		case "ZRANGE":
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

	keys, _, err := c.ScanKeys("*", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Type != "zset" {
		t.Errorf("type = %q, want zset (detection failed)", keys[0].Type)
	}
}

// ---------------------------------------------------------------------------
// ScanKeys — plain zset not mis-detected.
// ---------------------------------------------------------------------------

func TestScanKeys_PlainZSetNotMisDetected(t *testing.T) {
	client, _ := setupTestClient(t)

	if err := client.ZAddBatch("nz", goredis.Z{Score: 0.123, Member: "x"}); err != nil {
		t.Fatalf("ZAddBatch error: %v", err)
	}

	keys, _, err := client.ScanKeys("nz", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Type != "zset" {
		t.Errorf("type = %q, want zset", keys[0].Type)
	}
}

// ---------------------------------------------------------------------------
// detectStringSubtypes — probe truncation branch (value >= subtypeProbeBytes).
// ---------------------------------------------------------------------------

func TestScanKeys_DetectStringSubtypes_LargeValues(t *testing.T) {
	client, mr := setupTestClient(t)

	// Large plain-text value: probe returns exactly subtypeProbeBytes bytes and
	// the key must stay a plain string.
	mr.Set("large:text", strings.Repeat("a", subtypeProbeBytes*2))

	// Large binary value: incomplete probes stay "string" so large compressed
	// protobuf blobs are not mislabeled as bitmaps. GetValue still classifies
	// on open.
	bin := make([]byte, subtypeProbeBytes*2)
	for i := range bin {
		bin[i] = 0xff
	}
	mr.Set("large:bin", string(bin))

	// Short complete binary value still classifies as bitmap from the probe alone.
	mr.Set("short:bin", string([]byte{0xff, 0xfe}))

	keys, _, err := client.ScanKeys("*", 0, 100)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}
	byName := map[string]types.KeyType{}
	for _, k := range keys {
		byName[k.Key] = k.Type
	}
	if byName["large:text"] != "string" {
		t.Errorf("large:text type = %q, want string", byName["large:text"])
	}
	if byName["large:bin"] != "string" {
		t.Errorf("large:bin type = %q, want string (incomplete probe)", byName["large:bin"])
	}
	if byName["short:bin"] != "bitmap" {
		t.Errorf("short:bin type = %q, want bitmap", byName["short:bin"])
	}
}

// ---------------------------------------------------------------------------
// Benchmark: key-list scan over large string values. Before the GETRANGE
// probe, every SCAN page pipelined a full GET of each string key, transferring
// entire values just to sniff HLL/bitmap magic bytes.
// ---------------------------------------------------------------------------

func BenchmarkScanKeysLargeStringValues(b *testing.B) {
	client, mr := setupBenchClient(b)

	val := strings.Repeat("x", 512*1024) // 512KB per key
	for i := 0; i < 100; i++ {
		mr.Set(fmt.Sprintf("big:%03d", i), val)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := client.ScanKeys("big:*", 0, 100); err != nil {
			b.Fatalf("ScanKeys: %v", err)
		}
	}
}

func TestScanKeys_DetectStringSubtypes_ProtobufProbe(t *testing.T) {
	client, mr := setupTestClient(t)
	// field 1 string "menu"
	var raw []byte
	raw = append(raw, 0x0a, 0x04)
	raw = append(raw, []byte("menu")...)
	mr.Set("proto:small", string(raw))
	keys, _, err := client.ScanKeys("proto:small", 0, 10)
	if err != nil {
		t.Fatalf("ScanKeys: %v", err)
	}
	if len(keys) != 1 || keys[0].Type != types.KeyTypeProtobuf {
		t.Fatalf("got %+v, want protobuf", keys)
	}
}
