package redis

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

// ---------------------------------------------------------------------------
// ExportKeys tests
// ---------------------------------------------------------------------------

func TestExportKeys_AllTypes(t *testing.T) {
	client, mr := setupTestClient(t)

	// Set up all five key types.
	mr.Set("str", "hello")
	mr.RPush("lst", "a", "b", "c")
	mr.SAdd("st", "x", "y", "z")
	mr.ZAdd("zs", 1.0, "alpha")
	mr.ZAdd("zs", 2.5, "beta")
	mr.HSet("hs", "f1", "v1")
	mr.HSet("hs", "f2", "v2")

	result, err := client.ExportKeys("*")
	if err != nil {
		t.Fatalf("ExportKeys returned error: %v", err)
	}

	// All five keys must be present.
	for _, key := range []string{"str", "lst", "st", "zs", "hs"} {
		if _, ok := result[key]; !ok {
			t.Errorf("expected key %q in export result", key)
		}
	}

	// -- string --
	strData := result["str"].(map[string]any)
	if strData["type"] != "string" {
		t.Errorf("str type = %v, want string", strData["type"])
	}
	if strData["value"] != "hello" {
		t.Errorf("str value = %v, want hello", strData["value"])
	}

	// -- list --
	lstData := result["lst"].(map[string]any)
	if lstData["type"] != "list" {
		t.Errorf("lst type = %v, want list", lstData["type"])
	}
	listVals, ok := lstData["value"].([]string)
	if !ok {
		t.Fatalf("lst value is not []string, got %T", lstData["value"])
	}
	if len(listVals) != 3 || listVals[0] != "a" || listVals[1] != "b" || listVals[2] != "c" {
		t.Errorf("lst value = %v, want [a b c]", listVals)
	}

	// -- set --
	stData := result["st"].(map[string]any)
	if stData["type"] != "set" {
		t.Errorf("st type = %v, want set", stData["type"])
	}
	setVals, ok := stData["value"].([]string)
	if !ok {
		t.Fatalf("st value is not []string, got %T", stData["value"])
	}
	sort.Strings(setVals)
	if len(setVals) != 3 || setVals[0] != "x" || setVals[1] != "y" || setVals[2] != "z" {
		t.Errorf("st value = %v, want [x y z]", setVals)
	}

	// -- zset --
	zsData := result["zs"].(map[string]any)
	if zsData["type"] != "zset" {
		t.Errorf("zs type = %v, want zset", zsData["type"])
	}
	zsVals, ok := zsData["value"].([]map[string]any)
	if !ok {
		t.Fatalf("zs value is not []map[string]any, got %T", zsData["value"])
	}
	if len(zsVals) != 2 {
		t.Fatalf("zs value length = %d, want 2", len(zsVals))
	}
	// ZRangeWithScores returns sorted by score ascending.
	if zsVals[0]["member"] != "alpha" || zsVals[0]["score"] != 1.0 {
		t.Errorf("zs[0] = %v, want {member:alpha score:1}", zsVals[0])
	}
	if zsVals[1]["member"] != "beta" || zsVals[1]["score"] != 2.5 {
		t.Errorf("zs[1] = %v, want {member:beta score:2.5}", zsVals[1])
	}

	// -- hash --
	hsData := result["hs"].(map[string]any)
	if hsData["type"] != "hash" {
		t.Errorf("hs type = %v, want hash", hsData["type"])
	}
	hashVals, ok := hsData["value"].(map[string]string)
	if !ok {
		t.Fatalf("hs value is not map[string]string, got %T", hsData["value"])
	}
	if hashVals["f1"] != "v1" || hashVals["f2"] != "v2" {
		t.Errorf("hs value = %v, want {f1:v1 f2:v2}", hashVals)
	}
}

func TestExportKeys_WithTTL(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("ttlkey", "val")
	mr.SetTTL("ttlkey", 60*time.Second)

	result, err := client.ExportKeys("*")
	if err != nil {
		t.Fatalf("ExportKeys returned error: %v", err)
	}

	keyData, ok := result["ttlkey"].(map[string]any)
	if !ok {
		t.Fatalf("ttlkey not found or wrong type in export")
	}

	ttl, ok := keyData["ttl"].(float64)
	if !ok {
		t.Fatalf("ttl is not float64, got %T", keyData["ttl"])
	}
	// Allow some tolerance: should be ~60 seconds.
	if ttl < 55 || ttl > 65 {
		t.Errorf("ttl = %v, want ~60", ttl)
	}
}

func TestExportKeys_EmptyDB(t *testing.T) {
	client, _ := setupTestClient(t)

	result, err := client.ExportKeys("*")
	if err != nil {
		t.Fatalf("ExportKeys returned error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys", len(result))
	}
}

func TestExportKeys_Pattern(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("user:1", "alice")
	mr.Set("user:2", "bob")
	mr.Set("session:abc", "data")

	result, err := client.ExportKeys("user:*")
	if err != nil {
		t.Fatalf("ExportKeys returned error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 keys, got %d", len(result))
	}
	if _, ok := result["user:1"]; !ok {
		t.Error("expected user:1 in result")
	}
	if _, ok := result["user:2"]; !ok {
		t.Error("expected user:2 in result")
	}
	if _, ok := result["session:abc"]; ok {
		t.Error("session:abc should not be in result")
	}
}

// ---------------------------------------------------------------------------
// ImportKeys tests
// ---------------------------------------------------------------------------

func TestImportKeys_String(t *testing.T) {
	client, mr := setupTestClient(t)

	data := map[string]any{
		"mystr": map[string]any{
			"type":  "string",
			"value": "hello world",
			"ttl":   float64(30),
		},
	}

	count, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	got, err := mr.Get("mystr")
	if err != nil {
		t.Fatalf("miniredis Get error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("mystr = %q, want %q", got, "hello world")
	}

	ttl := mr.TTL("mystr")
	if ttl < 25*time.Second || ttl > 35*time.Second {
		t.Errorf("mystr TTL = %v, want ~30s", ttl)
	}
}

func TestImportKeys_List(t *testing.T) {
	client, mr := setupTestClient(t)

	data := map[string]any{
		"mylist": map[string]any{
			"type":  "list",
			"value": []any{"a", "b", "c"},
			"ttl":   float64(0),
		},
	}

	count, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	vals, err := mr.List("mylist")
	if err != nil {
		t.Fatalf("miniredis List error: %v", err)
	}
	if len(vals) != 3 || vals[0] != "a" || vals[1] != "b" || vals[2] != "c" {
		t.Errorf("mylist = %v, want [a b c]", vals)
	}
}

func TestImportKeys_Set(t *testing.T) {
	client, mr := setupTestClient(t)

	data := map[string]any{
		"myset": map[string]any{
			"type":  "set",
			"value": []any{"x", "y", "z"},
			"ttl":   float64(0),
		},
	}

	count, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	members, err := mr.Members("myset")
	if err != nil {
		t.Fatalf("miniredis Members error: %v", err)
	}
	sort.Strings(members)
	if len(members) != 3 || members[0] != "x" || members[1] != "y" || members[2] != "z" {
		t.Errorf("myset = %v, want [x y z]", members)
	}
}

func TestImportKeys_ZSet(t *testing.T) {
	client, mr := setupTestClient(t)

	data := map[string]any{
		"myzset": map[string]any{
			"type": "zset",
			"value": []any{
				map[string]any{"member": "alpha", "score": float64(1.0)},
				map[string]any{"member": "beta", "score": float64(2.5)},
			},
			"ttl": float64(0),
		},
	}

	count, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	score1, err := mr.ZScore("myzset", "alpha")
	if err != nil {
		t.Fatalf("ZScore alpha error: %v", err)
	}
	if score1 != 1.0 {
		t.Errorf("alpha score = %v, want 1.0", score1)
	}

	score2, err := mr.ZScore("myzset", "beta")
	if err != nil {
		t.Fatalf("ZScore beta error: %v", err)
	}
	if score2 != 2.5 {
		t.Errorf("beta score = %v, want 2.5", score2)
	}
}

func TestImportKeys_Hash(t *testing.T) {
	client, mr := setupTestClient(t)

	data := map[string]any{
		"myhash": map[string]any{
			"type": "hash",
			"value": map[string]any{
				"field1": "val1",
				"field2": "val2",
			},
			"ttl": float64(0),
		},
	}

	count, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	v1 := mr.HGet("myhash", "field1")
	if v1 != "val1" {
		t.Errorf("myhash field1 = %q, want %q", v1, "val1")
	}
	v2 := mr.HGet("myhash", "field2")
	if v2 != "val2" {
		t.Errorf("myhash field2 = %q, want %q", v2, "val2")
	}
}

func TestImportKeys_WithTTL(t *testing.T) {
	client, mr := setupTestClient(t)

	data := map[string]any{
		"ttlstr": map[string]any{
			"type":  "string",
			"value": "expiring",
			"ttl":   float64(120),
		},
	}

	_, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys returned error: %v", err)
	}

	ttl := mr.TTL("ttlstr")
	if ttl < 115*time.Second || ttl > 125*time.Second {
		t.Errorf("ttlstr TTL = %v, want ~120s", ttl)
	}
}

func TestImportKeys_InvalidData(t *testing.T) {
	client, _ := setupTestClient(t)

	tests := []struct {
		name string
		data map[string]any
	}{
		{
			name: "value is not a map",
			data: map[string]any{
				"bad": "not a map",
			},
		},
		{
			name: "missing type field",
			data: map[string]any{
				"bad": map[string]any{
					"value": "hello",
				},
			},
		},
		{
			name: "wrong value type for string",
			data: map[string]any{
				"bad": map[string]any{
					"type":  "string",
					"value": 12345, // not a string
				},
			},
		},
		{
			name: "wrong value type for list",
			data: map[string]any{
				"bad": map[string]any{
					"type":  "list",
					"value": "not a slice",
				},
			},
		},
		{
			name: "wrong value type for set",
			data: map[string]any{
				"bad": map[string]any{
					"type":  "set",
					"value": 42,
				},
			},
		},
		{
			name: "wrong value type for zset",
			data: map[string]any{
				"bad": map[string]any{
					"type":  "zset",
					"value": "not a slice",
				},
			},
		},
		{
			name: "wrong value type for hash",
			data: map[string]any{
				"bad": map[string]any{
					"type":  "hash",
					"value": []any{"not", "a", "map"},
				},
			},
		},
		{
			name: "unknown type",
			data: map[string]any{
				"bad": map[string]any{
					"type":  "unknown",
					"value": "whatever",
				},
			},
		},
		{
			name: "empty data",
			data: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := client.ImportKeys(tt.data)
			if err != nil {
				t.Errorf("ImportKeys returned unexpected error: %v", err)
			}
			if count != 0 {
				t.Errorf("count = %d, want 0 for invalid data", count)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Round-trip test
// ---------------------------------------------------------------------------

func TestExportImportRoundTrip(t *testing.T) {
	client, mr := setupTestClient(t)

	// Create keys of multiple types.
	mr.Set("rt:str", "round-trip")
	mr.RPush("rt:lst", "one", "two", "three")
	mr.SAdd("rt:set", "m1", "m2")
	mr.ZAdd("rt:zs", 10, "first")
	mr.ZAdd("rt:zs", 20, "second")
	mr.HSet("rt:hs", "k1", "v1")
	mr.HSet("rt:hs", "k2", "v2")

	// Export.
	exported, err := client.ExportKeys("rt:*")
	if err != nil {
		t.Fatalf("ExportKeys error: %v", err)
	}
	if len(exported) != 5 {
		t.Fatalf("exported %d keys, want 5", len(exported))
	}

	// Clear.
	mr.FlushAll()

	// Convert exported data to the format ImportKeys expects.
	// ExportKeys returns typed Go values ([]string, map[string]string, etc.)
	// but ImportKeys expects JSON-like any types ([]any, map[string]any).
	importData := make(map[string]any)
	for key, raw := range exported {
		keyData := raw.(map[string]any)
		entry := map[string]any{
			"type": keyData["type"],
			"ttl":  keyData["ttl"],
		}

		switch keyData["type"] {
		case "string":
			entry["value"] = keyData["value"]
		case "list":
			vals := keyData["value"].([]string)
			iface := make([]any, len(vals))
			for i, v := range vals {
				iface[i] = v
			}
			entry["value"] = iface
		case "set":
			vals := keyData["value"].([]string)
			iface := make([]any, len(vals))
			for i, v := range vals {
				iface[i] = v
			}
			entry["value"] = iface
		case "zset":
			vals := keyData["value"].([]map[string]any)
			iface := make([]any, len(vals))
			for i, v := range vals {
				iface[i] = v
			}
			entry["value"] = iface
		case "hash":
			vals := keyData["value"].(map[string]string)
			iface := make(map[string]any, len(vals))
			for k, v := range vals {
				iface[k] = v
			}
			entry["value"] = iface
		}

		importData[key] = entry
	}

	// Import.
	count, err := client.ImportKeys(importData)
	if err != nil {
		t.Fatalf("ImportKeys error: %v", err)
	}
	if count != 5 {
		t.Errorf("imported %d keys, want 5", count)
	}

	// Verify all keys restored correctly.

	// String
	strVal, err := mr.Get("rt:str")
	if err != nil {
		t.Fatalf("Get rt:str error: %v", err)
	}
	if strVal != "round-trip" {
		t.Errorf("rt:str = %q, want %q", strVal, "round-trip")
	}

	// List
	lstVal, err := mr.List("rt:lst")
	if err != nil {
		t.Fatalf("List rt:lst error: %v", err)
	}
	if len(lstVal) != 3 || lstVal[0] != "one" || lstVal[1] != "two" || lstVal[2] != "three" {
		t.Errorf("rt:lst = %v, want [one two three]", lstVal)
	}

	// Set
	setVal, err := mr.Members("rt:set")
	if err != nil {
		t.Fatalf("Members rt:set error: %v", err)
	}
	sort.Strings(setVal)
	if len(setVal) != 2 || setVal[0] != "m1" || setVal[1] != "m2" {
		t.Errorf("rt:set = %v, want [m1 m2]", setVal)
	}

	// ZSet
	score1, err := mr.ZScore("rt:zs", "first")
	if err != nil {
		t.Fatalf("ZScore rt:zs first error: %v", err)
	}
	if score1 != 10 {
		t.Errorf("rt:zs first score = %v, want 10", score1)
	}
	score2, err := mr.ZScore("rt:zs", "second")
	if err != nil {
		t.Fatalf("ZScore rt:zs second error: %v", err)
	}
	if score2 != 20 {
		t.Errorf("rt:zs second score = %v, want 20", score2)
	}

	// Hash
	h1 := mr.HGet("rt:hs", "k1")
	if h1 != "v1" {
		t.Errorf("rt:hs k1 = %q, want %q", h1, "v1")
	}
	h2 := mr.HGet("rt:hs", "k2")
	if h2 != "v2" {
		t.Errorf("rt:hs k2 = %q, want %q", h2, "v2")
	}
}

// ---------------------------------------------------------------------------
// ExportKeys — error path: scanAll fails on the underlying SCAN.
// ---------------------------------------------------------------------------

func TestExportKeys_ScanAllError(t *testing.T) {
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

	if _, err := c.ExportKeys("*"); err == nil {
		t.Error("expected error from ExportKeys when SCAN fails")
	}
}

// ---------------------------------------------------------------------------
// ExportKeys — exercise every "continue" branch in the value-fetch switch by
// returning a specific TYPE for each key and an error reply for the matching
// value command. Also exercises the ReJSON-RL queue branch.
// ---------------------------------------------------------------------------

func TestExportKeys_AllValueFetchErrors(t *testing.T) {
	keyTypes := map[string]string{
		"kstr":    "string",
		"klist":   "list",
		"kset":    "set",
		"kzset":   "zset",
		"khash":   "hash",
		"kstream": "stream",
		"kjson":   "ReJSON-RL",
	}
	keys := []string{"kstr", "klist", "kset", "kzset", "khash", "kstream", "kjson"}

	srv := newFakeRedisServer(t)
	var mu sync.Mutex
	scanCalls := 0
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			mu.Lock()
			scanCalls++
			mu.Unlock()
			// Cursor 0, 7 keys.
			out := fmt.Sprintf("*2\r\n$1\r\n0\r\n*%d\r\n", len(keys))
			for _, k := range keys {
				out += fmt.Sprintf("$%d\r\n%s\r\n", len(k), k)
			}
			return out
		case "TYPE":
			if len(argv) < 2 {
				return "+none\r\n"
			}
			if kt, ok := keyTypes[argv[1]]; ok {
				return "+" + kt + "\r\n"
			}
			return "+none\r\n"
		case "TTL":
			return ":-1\r\n"
		case "GET", "LRANGE", "SMEMBERS", "ZRANGE", "HGETALL", "XRANGE", "JSON.GET":
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

	result, err := c.ExportKeys("*")
	if err != nil {
		t.Fatalf("ExportKeys error: %v", err)
	}
	// Every value fetch returned an error so every key should have been
	// skipped via the "continue" branch — result should be empty.
	if len(result) != 0 {
		t.Errorf("ExportKeys returned %d entries, want 0 (all errors)", len(result))
	}
}

// ---------------------------------------------------------------------------
// ExportKeys — unknown type triggers the default-continue branch in the
// value-fetch queue switch (and the same in the result-collect switch).
// ---------------------------------------------------------------------------

func TestExportKeys_UnknownType_DefaultContinue(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*1\r\n$5\r\nweird\r\n"
		case "TYPE":
			return "+weirdtype\r\n"
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

	result, err := c.ExportKeys("*")
	if err != nil {
		t.Fatalf("ExportKeys error: %v", err)
	}
	// Unknown type is skipped via default branch.
	if _, ok := result["weird"]; ok {
		t.Errorf("expected weird key to be skipped, got %v", result["weird"])
	}
}

// ---------------------------------------------------------------------------
// ExportKeys — ReJSON-RL where the JSON.GET reply succeeds at the pipeline
// layer (returns *redis.Cmd, no top-level error) but the inner cmd.Text()
// fails because the reply payload is not a string. We return an integer
// reply to drive that branch.
// ---------------------------------------------------------------------------

func TestExportKeys_ReJSON_TextError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*1\r\n$5\r\nkjson\r\n"
		case "TYPE":
			return "+ReJSON-RL\r\n"
		case "TTL":
			return ":-1\r\n"
		case "JSON.GET":
			// Integer reply — Text() will fail with redis.Nil or type error.
			return ":42\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	result, err := c.ExportKeys("*")
	if err != nil {
		t.Fatalf("ExportKeys error: %v", err)
	}
	// Text() error => key skipped via continue.
	if _, ok := result["kjson"]; ok {
		t.Errorf("expected kjson skipped on Text() error, got %v", result["kjson"])
	}
}

// ---------------------------------------------------------------------------
// ExportKeys — ReJSON-RL success path: exercises the JSON.GET extraction
// branch including the cmd.Text() success leg (line 135-137).
// ---------------------------------------------------------------------------

func TestExportKeys_ReJSON_Success(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*1\r\n$5\r\nkjson\r\n"
		case "TYPE":
			return "+ReJSON-RL\r\n"
		case "TTL":
			return ":-1\r\n"
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

	result, err := c.ExportKeys("*")
	if err != nil {
		t.Fatalf("ExportKeys error: %v", err)
	}
	entry, ok := result["kjson"].(map[string]any)
	if !ok {
		t.Fatalf("kjson missing or wrong shape: %T", result["kjson"])
	}
	if entry["type"] != "ReJSON-RL" {
		t.Errorf("type = %v, want ReJSON-RL", entry["type"])
	}
	if entry["value"] != `{"a":1}` {
		t.Errorf("value = %v, want %q", entry["value"], `{"a":1}`)
	}
}

// ---------------------------------------------------------------------------
// CompareKeys tests
// ---------------------------------------------------------------------------

func TestCompareKeys(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("cmp1", "value-one")
	mr.Set("cmp2", "value-two")

	val1, val2, err := client.CompareKeys("cmp1", "cmp2")
	if err != nil {
		t.Fatalf("CompareKeys returned error: %v", err)
	}

	if val1.StringValue != "value-one" {
		t.Errorf("val1 = %q, want %q", val1.StringValue, "value-one")
	}
	if val2.StringValue != "value-two" {
		t.Errorf("val2 = %q, want %q", val2.StringValue, "value-two")
	}
}

func TestCompareKeys_MissingKey(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("exists", "present")

	// When a key does not exist, Redis TYPE returns "none" and GetValue
	// returns a RedisValue with Type "none" (no error). CompareKeys passes
	// this through, so we verify the type field reflects the missing key.

	// First key missing.
	val1, val2, err := client.CompareKeys("nonexistent", "exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val1.Type != "none" {
		t.Errorf("val1.Type = %q, want %q for missing key", val1.Type, "none")
	}
	if val2.StringValue != "present" {
		t.Errorf("val2.StringValue = %q, want %q", val2.StringValue, "present")
	}

	// Second key missing.
	val1, val2, err = client.CompareKeys("exists", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val1.StringValue != "present" {
		t.Errorf("val1.StringValue = %q, want %q", val1.StringValue, "present")
	}
	if val2.Type != "none" {
		t.Errorf("val2.Type = %q, want %q for missing key", val2.Type, "none")
	}
}

// ---------------------------------------------------------------------------
// ExportKeys — exercise the stream extraction path
// ---------------------------------------------------------------------------

func TestExportKeys_Stream(t *testing.T) {
	client, _ := setupTestClient(t)

	id, err := client.XAdd("estream", map[string]any{"field": "value"})
	if err != nil {
		t.Fatalf("XAdd error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty stream ID")
	}

	result, err := client.ExportKeys("estream")
	if err != nil {
		t.Fatalf("ExportKeys error: %v", err)
	}

	streamData, ok := result["estream"].(map[string]any)
	if !ok {
		t.Fatalf("estream not found or wrong type in export, got %T", result["estream"])
	}
	if streamData["type"] != "stream" {
		t.Errorf("estream type = %v, want stream", streamData["type"])
	}

	entries, ok := streamData["value"].([]map[string]any)
	if !ok {
		t.Fatalf("estream value is not []map[string]any, got %T", streamData["value"])
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["id"] != id {
		t.Errorf("entry id = %v, want %v", entries[0]["id"], id)
	}
}

// ---------------------------------------------------------------------------
// ImportKeys — TTL branches for collection types
// ---------------------------------------------------------------------------

func TestImportKeys_CollectionTTLBranches(t *testing.T) {
	client, mr := setupTestClient(t)

	data := map[string]any{
		"il": map[string]any{
			"type":  "list",
			"value": []any{"a", "b"},
			"ttl":   float64(60),
		},
		"is": map[string]any{
			"type":  "set",
			"value": []any{"x", "y"},
			"ttl":   float64(60),
		},
		"iz": map[string]any{
			"type": "zset",
			"value": []any{
				map[string]any{"member": "alpha", "score": float64(1.0)},
			},
			"ttl": float64(60),
		},
		"ih": map[string]any{
			"type": "hash",
			"value": map[string]any{
				"field": "val",
			},
			"ttl": float64(60),
		},
	}

	count, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys error: %v", err)
	}
	if count != 4 {
		t.Errorf("count = %d, want 4", count)
	}

	for _, k := range []string{"il", "is", "iz", "ih"} {
		ttl := mr.TTL(k)
		if ttl < 50*time.Second || ttl > 70*time.Second {
			t.Errorf("%s TTL = %v, want ~60s", k, ttl)
		}
	}
}

// ---------------------------------------------------------------------------
// ImportKeys — ReJSON-RL branch (JSONSet will fail in miniredis but the
// branch should still be exercised and counted).
// ---------------------------------------------------------------------------

func TestImportKeys_ReJSON(t *testing.T) {
	client, _ := setupTestClient(t)

	data := map[string]any{
		"jsonkey": map[string]any{
			"type":  "ReJSON-RL",
			"value": `{"a":1}`,
			"ttl":   float64(0),
		},
	}

	count, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys error: %v", err)
	}
	// Even though JSON.SET fails in miniredis, the count is still incremented
	// because the code intentionally swallows the error.
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestImportKeys_ReJSONWithTTL(t *testing.T) {
	client, _ := setupTestClient(t)

	data := map[string]any{
		"jsonttl": map[string]any{
			"type":  "ReJSON-RL",
			"value": `{"a":1}`,
			"ttl":   float64(60),
		},
	}
	_, err := client.ImportKeys(data)
	if err != nil {
		t.Fatalf("ImportKeys error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// queueValueFetch / extractValue — direct call with ReJSON-RL keyType
// ---------------------------------------------------------------------------

func TestQueueValueFetch_ReJSON(t *testing.T) {
	client, _ := setupTestClient(t)

	pipe := client.pipeline()
	cmds := queueValueFetch(pipe, client.ctx, "jsonkey", "ReJSON-RL")
	_, _ = pipe.Exec(client.ctx)

	if cmds.jsonCmd == nil {
		t.Error("expected jsonCmd to be set for ReJSON-RL keyType")
	}

	// extractValue should walk the ReJSON-RL branch (Text() will return an
	// error from miniredis but the branch is still exercised).
	val := extractValue("ReJSON-RL", cmds)
	if val.Type != "ReJSON-RL" {
		t.Errorf("Type = %q, want ReJSON-RL", val.Type)
	}
}

func TestCompareKeys_StringSubtypes(t *testing.T) {
	client, mr := setupTestClient(t)

	var protoRaw []byte
	protoRaw = append(protoRaw, 0x0a, 0x03)
	protoRaw = append(protoRaw, []byte("abc")...)
	mr.Set("proto", string(protoRaw))
	mr.Set("hll", "HYLL"+string(make([]byte, 12)))

	v1, v2, err := client.CompareKeys("proto", "hll")
	if err != nil {
		t.Fatalf("CompareKeys error: %v", err)
	}
	if v1.Type != types.KeyTypeProtobuf {
		t.Errorf("proto type = %q, want protobuf", v1.Type)
	}
	if v1.DecodedValue == "" || !containsStr(v1.DecodedValue, `1: "abc"`) {
		t.Errorf("proto decoded = %q", v1.DecodedValue)
	}
	if v2.Type != types.KeyTypeHyperLogLog {
		t.Errorf("hll type = %q, want hyperloglog", v2.Type)
	}

	// Bitmap path: invalid UTF-8 that is not protobuf.
	if err := client.SetBit("bm", 0, 1); err != nil {
		t.Fatalf("SetBit: %v", err)
	}
	if err := client.SetBit("bm", 1, 1); err != nil {
		t.Fatalf("SetBit: %v", err)
	}
	vb, _, err := client.CompareKeys("bm", "proto")
	if err != nil {
		t.Fatalf("CompareKeys bitmap error: %v", err)
	}
	if vb.Type != types.KeyTypeBitmap {
		t.Errorf("bitmap type = %q, want bitmap", vb.Type)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOfStr(s, sub) >= 0)
}

func indexOfStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestExtractValue_AllTypes(t *testing.T) {
	// Empty fetch cmds — verifies the nil-check branches in each case arm.
	for _, kt := range []string{"string", "list", "set", "zset", "hash", "stream", "ReJSON-RL"} {
		val := extractValue(kt, valueFetchCmds{})
		if string(val.Type) != kt {
			t.Errorf("Type = %q, want %q", val.Type, kt)
		}
	}

	// Default branch — unknown type just sets Type and returns.
	val := extractValue("unknown", valueFetchCmds{})
	if string(val.Type) != "unknown" {
		t.Errorf("Type = %q, want unknown", val.Type)
	}
}

// ---------------------------------------------------------------------------
// CompareKeys — TYPE pipeline error path. Use the fake server to make TYPE
// return an error so the pipeline Exec returns a non-nil, non-redis.Nil err.
// ---------------------------------------------------------------------------

func TestCompareKeys_TypeExecError(t *testing.T) {
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

	if _, _, err := c.CompareKeys("a", "b"); err == nil {
		t.Error("expected error from CompareKeys when TYPE pipeline errors")
	}
}

// ---------------------------------------------------------------------------
// CompareKeys — exercise non-string types so queueValueFetch and extractValue
// take their list/set/zset/hash/stream branches.
// ---------------------------------------------------------------------------

func TestCompareKeys_AllTypes(t *testing.T) {
	client, mr := setupTestClient(t)

	// list vs list
	mr.RPush("l1", "a", "b")
	mr.RPush("l2", "c", "d", "e")

	v1, v2, err := client.CompareKeys("l1", "l2")
	if err != nil {
		t.Fatalf("CompareKeys list error: %v", err)
	}
	if len(v1.ListValue) != 2 || len(v2.ListValue) != 3 {
		t.Errorf("list compare lengths = %d/%d, want 2/3", len(v1.ListValue), len(v2.ListValue))
	}

	// set vs set
	mr.SAdd("s1", "x", "y")
	mr.SAdd("s2", "p")
	v1, v2, err = client.CompareKeys("s1", "s2")
	if err != nil {
		t.Fatalf("CompareKeys set error: %v", err)
	}
	if len(v1.SetValue) != 2 || len(v2.SetValue) != 1 {
		t.Errorf("set compare lengths = %d/%d, want 2/1", len(v1.SetValue), len(v2.SetValue))
	}

	// zset vs zset
	mr.ZAdd("z1", 1.0, "a")
	mr.ZAdd("z2", 2.0, "b")
	v1, v2, err = client.CompareKeys("z1", "z2")
	if err != nil {
		t.Fatalf("CompareKeys zset error: %v", err)
	}
	if len(v1.ZSetValue) != 1 || len(v2.ZSetValue) != 1 {
		t.Errorf("zset compare lengths = %d/%d, want 1/1", len(v1.ZSetValue), len(v2.ZSetValue))
	}

	// hash vs hash
	mr.HSet("h1", "k", "v")
	mr.HSet("h2", "k1", "v1")
	mr.HSet("h2", "k2", "v2")
	v1, v2, err = client.CompareKeys("h1", "h2")
	if err != nil {
		t.Fatalf("CompareKeys hash error: %v", err)
	}
	if len(v1.HashValue) != 1 || len(v2.HashValue) != 2 {
		t.Errorf("hash compare lengths = %d/%d, want 1/2", len(v1.HashValue), len(v2.HashValue))
	}

	// stream vs stream
	if _, err := client.XAdd("st1", map[string]any{"a": "1"}); err != nil {
		t.Fatalf("XAdd st1 error: %v", err)
	}
	if _, err := client.XAdd("st2", map[string]any{"b": "2"}); err != nil {
		t.Fatalf("XAdd st2 error: %v", err)
	}
	v1, v2, err = client.CompareKeys("st1", "st2")
	if err != nil {
		t.Fatalf("CompareKeys stream error: %v", err)
	}
	if len(v1.StreamValue) != 1 || len(v2.StreamValue) != 1 {
		t.Errorf("stream compare lengths = %d/%d, want 1/1", len(v1.StreamValue), len(v2.StreamValue))
	}
}
