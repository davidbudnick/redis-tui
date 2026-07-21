package redis

import (
	"fmt"
	"strings"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/klauspost/compress/s2"
)

// ---------------------------------------------------------------------------
// GetValuePreview — happy paths per type via miniredis
// ---------------------------------------------------------------------------

func TestGetValuePreview_NonexistentKey(t *testing.T) {
	client, _ := setupTestClient(t)

	v, err := client.GetValuePreview("missing")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.Type != "none" {
		t.Errorf("Type = %q, want none", v.Type)
	}
	if v.Truncated {
		t.Error("expected Truncated = false")
	}
}

func TestGetValuePreview_StringSmall(t *testing.T) {
	client, mr := setupTestClient(t)
	mr.Set("s", "hello")

	v, err := client.GetValuePreview("s")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.Type != types.KeyTypeString {
		t.Errorf("Type = %q, want string", v.Type)
	}
	if v.StringValue != "hello" {
		t.Errorf("StringValue = %q, want hello", v.StringValue)
	}
	if v.Truncated {
		t.Error("expected Truncated = false")
	}
}

func TestGetValuePreview_StringTruncated(t *testing.T) {
	client, mr := setupTestClient(t)
	total := previewMaxStringBytes + 1024
	mr.Set("big", strings.Repeat("a", total))

	v, err := client.GetValuePreview("big")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if len(v.StringValue) != previewMaxStringBytes {
		t.Errorf("StringValue length = %d, want %d", len(v.StringValue), previewMaxStringBytes)
	}
	if !v.Truncated {
		t.Error("expected Truncated = true")
	}
	if v.TotalCount != int64(total) {
		t.Errorf("TotalCount = %d, want %d", v.TotalCount, total)
	}
}

func TestGetValuePreview_HyperLogLog(t *testing.T) {
	client, mr := setupTestClient(t)
	mr.Set("hll", "HYLL"+string(make([]byte, 12)))

	v, err := client.GetValuePreview("hll")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.Type != types.KeyTypeHyperLogLog {
		t.Errorf("Type = %q, want hyperloglog", v.Type)
	}
}

func TestGetValuePreview_Bitmap(t *testing.T) {
	client, _ := setupTestClient(t)
	for _, off := range []int64{0, 1, 7} {
		if err := client.SetBit("bm", off, 1); err != nil {
			t.Fatalf("SetBit(%d) error: %v", off, err)
		}
	}

	v, err := client.GetValuePreview("bm")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.Type != types.KeyTypeBitmap {
		t.Errorf("Type = %q, want bitmap", v.Type)
	}
	if v.BitCount != 3 {
		t.Errorf("BitCount = %d, want 3", v.BitCount)
	}
	if len(v.BitPositions) != 3 {
		t.Errorf("BitPositions length = %d, want 3", len(v.BitPositions))
	}
}

func TestGetValuePreview_BitmapTruncated(t *testing.T) {
	client, mr := setupTestClient(t)
	// Binary value larger than the preview byte cap. Incomplete binary must
	// stay typed as string so large compressed protobuf is not mislabeled.
	bin := make([]byte, previewMaxStringBytes+512)
	for i := range bin {
		bin[i] = 0xff
	}
	mr.Set("bigbm", string(bin))

	v, err := client.GetValuePreview("bigbm")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.Type != types.KeyTypeString {
		t.Errorf("Type = %q, want string for truncated non-protobuf binary", v.Type)
	}
	if !v.Truncated {
		t.Error("expected Truncated = true")
	}
}

func TestGetValuePreview_List(t *testing.T) {
	t.Run("small list", func(t *testing.T) {
		client, mr := setupTestClient(t)
		mr.RPush("l", "a", "b", "c")

		v, err := client.GetValuePreview("l")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if len(v.ListValue) != 3 {
			t.Errorf("ListValue length = %d, want 3", len(v.ListValue))
		}
		if v.Truncated {
			t.Error("expected Truncated = false")
		}
	})

	t.Run("large list is truncated", func(t *testing.T) {
		client, mr := setupTestClient(t)
		for i := 0; i < previewMaxItems+50; i++ {
			mr.RPush("biglist", "item")
		}

		v, err := client.GetValuePreview("biglist")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if len(v.ListValue) != previewMaxItems {
			t.Errorf("ListValue length = %d, want %d", len(v.ListValue), previewMaxItems)
		}
		if !v.Truncated {
			t.Error("expected Truncated = true")
		}
		if v.TotalCount != int64(previewMaxItems+50) {
			t.Errorf("TotalCount = %d, want %d", v.TotalCount, previewMaxItems+50)
		}
	})
}

func TestGetValuePreview_Set(t *testing.T) {
	t.Run("small set", func(t *testing.T) {
		client, mr := setupTestClient(t)
		mr.SAdd("st", "a", "b", "c")

		v, err := client.GetValuePreview("st")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if len(v.SetValue) != 3 {
			t.Errorf("SetValue length = %d, want 3", len(v.SetValue))
		}
		if v.Truncated {
			t.Error("expected Truncated = false")
		}
	})

	t.Run("large set is truncated", func(t *testing.T) {
		client, mr := setupTestClient(t)
		for i := 0; i < previewMaxItems+50; i++ {
			mr.SAdd("bigset", strings.Repeat("m", 3)+string(rune('0'+i%10))+strings.Repeat("x", i/10))
		}

		v, err := client.GetValuePreview("bigset")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if len(v.SetValue) >= previewMaxItems+50 {
			t.Errorf("SetValue length = %d, want fewer than the full set", len(v.SetValue))
		}
		if !v.Truncated {
			t.Error("expected Truncated = true")
		}
	})
}

func TestGetValuePreview_ZSet(t *testing.T) {
	t.Run("small zset", func(t *testing.T) {
		client, _ := setupTestClient(t)
		if err := client.ZAdd("z", 1.5, "one"); err != nil {
			t.Fatalf("ZAdd error: %v", err)
		}

		v, err := client.GetValuePreview("z")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if len(v.ZSetValue) != 1 || v.ZSetValue[0].Member != "one" {
			t.Errorf("ZSetValue = %v, want [one]", v.ZSetValue)
		}
		if v.Truncated {
			t.Error("expected Truncated = false")
		}
	})

	t.Run("large zset is truncated", func(t *testing.T) {
		client, mr := setupTestClient(t)
		for i := 0; i < previewMaxItems+50; i++ {
			mr.ZAdd("bigz", float64(i), strings.Repeat("m", i%7+1)+string(rune('a'+i%26)))
		}

		v, err := client.GetValuePreview("bigz")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if len(v.ZSetValue) != previewMaxItems {
			t.Errorf("ZSetValue length = %d, want %d", len(v.ZSetValue), previewMaxItems)
		}
		if !v.Truncated {
			t.Error("expected Truncated = true")
		}
	})
}

func TestGetValuePreview_Geo(t *testing.T) {
	client, _ := setupTestClient(t)

	// 52-bit integer in the geohash range, as produced by GEOADD.
	if err := client.ZAdd("geo", 3.4e15, "Palermo"); err != nil {
		t.Fatalf("ZAdd error: %v", err)
	}

	v, err := client.GetValuePreview("geo")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.Type != types.KeyTypeGeo {
		t.Errorf("Type = %q, want geo", v.Type)
	}
	if len(v.GeoValue) != 1 {
		t.Errorf("GeoValue length = %d, want 1", len(v.GeoValue))
	}
}

func TestGetValuePreview_HashCapped(t *testing.T) {
	client, mr := setupTestClient(t)
	// miniredis returns every field from a single HSCAN iteration regardless
	// of COUNT, which exercises the defensive insert cap.
	for i := 0; i < previewMaxItems+50; i++ {
		mr.HSet("bighash", fmt.Sprintf("f%03d", i), "v")
	}

	v, err := client.GetValuePreview("bighash")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if len(v.HashValue) != previewMaxItems {
		t.Errorf("HashValue length = %d, want cap %d", len(v.HashValue), previewMaxItems)
	}
	if !v.Truncated {
		t.Error("expected Truncated = true")
	}
	if v.TotalCount != int64(previewMaxItems+50) {
		t.Errorf("TotalCount = %d, want %d", v.TotalCount, previewMaxItems+50)
	}
}

func TestGetValuePreview_Hash(t *testing.T) {
	client, mr := setupTestClient(t)
	mr.HSet("h", "f1", "v1")
	mr.HSet("h", "f2", "v2")

	v, err := client.GetValuePreview("h")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if len(v.HashValue) != 2 {
		t.Errorf("HashValue length = %d, want 2", len(v.HashValue))
	}
	if v.HashValue["f1"] != "v1" {
		t.Errorf("HashValue[f1] = %q, want v1", v.HashValue["f1"])
	}
	if v.Truncated {
		t.Error("expected Truncated = false")
	}
}

func TestGetValuePreview_Stream(t *testing.T) {
	t.Run("small stream", func(t *testing.T) {
		client, _ := setupTestClient(t)
		if _, err := client.XAdd("str", map[string]any{"k": "v"}); err != nil {
			t.Fatalf("XAdd error: %v", err)
		}

		v, err := client.GetValuePreview("str")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if len(v.StreamValue) != 1 {
			t.Errorf("StreamValue length = %d, want 1", len(v.StreamValue))
		}
		if v.Truncated {
			t.Error("expected Truncated = false")
		}
	})

	t.Run("large stream is truncated", func(t *testing.T) {
		client, _ := setupTestClient(t)
		for i := 0; i < previewMaxItems+50; i++ {
			if _, err := client.XAdd("bigstr", map[string]any{"k": "v"}); err != nil {
				t.Fatalf("XAdd error: %v", err)
			}
		}

		v, err := client.GetValuePreview("bigstr")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if len(v.StreamValue) != previewMaxItems {
			t.Errorf("StreamValue length = %d, want %d", len(v.StreamValue), previewMaxItems)
		}
		if !v.Truncated {
			t.Error("expected Truncated = true")
		}
	})
}

// ---------------------------------------------------------------------------
// GetValuePreview — HLL cardinality via the fake server (PFCOUNT succeeds)
// ---------------------------------------------------------------------------

func TestGetValuePreview_HLLCount(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "TYPE":
			return "+string\r\n"
		case "STRLEN":
			return ":16\r\n"
		case "GETRANGE":
			return "$16\r\nHYLL000000000000\r\n"
		case "PFCOUNT":
			return ":42\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	v, err := c.GetValuePreview("hll")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.Type != types.KeyTypeHyperLogLog {
		t.Errorf("Type = %q, want hyperloglog", v.Type)
	}
	if v.HLLCount != 42 {
		t.Errorf("HLLCount = %d, want 42", v.HLLCount)
	}
}

// ---------------------------------------------------------------------------
// GetValuePreview — ReJSON paths via the fake server (miniredis lacks JSON.GET)
// ---------------------------------------------------------------------------

func TestGetValuePreview_JSON(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "TYPE":
			return "+ReJSON-RL\r\n"
		case "JSON.GET":
			return "$9\r\n[{\"a\":1}]\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	v, err := c.GetValuePreview("doc")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.JSONValue != `[{"a":1}]` {
		t.Errorf("JSONValue = %q, want [{\"a\":1}]", v.JSONValue)
	}
}

// ---------------------------------------------------------------------------
// GetValuePreview — error paths, one per failing command, via the fake server
// ---------------------------------------------------------------------------

func TestGetValuePreview_Errors(t *testing.T) {
	tests := []struct {
		name     string
		failCmd  string
		keyType  string
		prefetch map[string]string // canned success replies for commands before the failure
	}{
		{"type error", "TYPE", "", nil},
		{"strlen error", "STRLEN", "+string\r\n", nil},
		{"getrange error", "GETRANGE", "+string\r\n", map[string]string{"STRLEN": ":5\r\n"}},
		{"llen error", "LLEN", "+list\r\n", nil},
		{"lrange error", "LRANGE", "+list\r\n", map[string]string{"LLEN": ":5\r\n"}},
		{"scard error", "SCARD", "+set\r\n", nil},
		{"sscan error", "SSCAN", "+set\r\n", map[string]string{"SCARD": ":5\r\n"}},
		{"zcard error", "ZCARD", "+zset\r\n", nil},
		{"zrange error", "ZRANGE", "+zset\r\n", map[string]string{"ZCARD": ":5\r\n"}},
		{"hlen error", "HLEN", "+hash\r\n", nil},
		{"hscan error", "HSCAN", "+hash\r\n", map[string]string{"HLEN": ":5\r\n"}},
		{"xlen error", "XLEN", "+stream\r\n", nil},
		{"xrange error", "XRANGE", "+stream\r\n", map[string]string{"XLEN": ":5\r\n"}},
		{"json get error", "JSON.GET", "+ReJSON-RL\r\n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeRedisServer(t)
			srv.setHandler(func(argv []string) string {
				if argv[0] == tt.failCmd {
					return "-ERR injected\r\n"
				}
				if argv[0] == "TYPE" {
					return tt.keyType
				}
				if resp, ok := tt.prefetch[argv[0]]; ok {
					return resp
				}
				return ""
			})
			host, port := srv.addr()
			c := NewClient()
			if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port}); err != nil {
				t.Fatalf("Connect: %v", err)
			}
			t.Cleanup(func() { _ = c.Disconnect() })

			if _, err := c.GetValuePreview("k"); err == nil {
				t.Error("expected error from GetValuePreview")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// previewGeoDetect — GEOPOS error leaves the value as a plain zset
// ---------------------------------------------------------------------------

func TestGetValuePreview_GeoPosError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "TYPE":
			return "+zset\r\n"
		case "ZCARD":
			return ":1\r\n"
		case "ZRANGE":
			return "*2\r\n$7\r\nPalermo\r\n$16\r\n3400000000000000\r\n"
		case "GEOPOS":
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	v, err := c.GetValuePreview("geo")
	if err != nil {
		t.Fatalf("GetValuePreview error: %v", err)
	}
	if v.Type != types.KeyTypeZSet {
		t.Errorf("Type = %q, want zset when GEOPOS fails", v.Type)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks: the fetch performed when a key is selected in the keys list.
// Before GetValuePreview this ran GetValue, pulling entire collections
// (LRANGE 0 -1, HGETALL, SMEMBERS, ZRANGE 0 -1) for a screenful of preview.
// ---------------------------------------------------------------------------

func BenchmarkPreviewFetch(b *testing.B) {
	b.Run("hash_50k_fields", func(b *testing.B) {
		client, mr := setupBenchClient(b)
		for i := 0; i < 50_000; i++ {
			mr.HSet("bighash", fmt.Sprintf("field:%06d", i), strings.Repeat("v", 64))
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := client.GetValuePreview("bighash"); err != nil {
				b.Fatalf("preview fetch: %v", err)
			}
		}
	})

	b.Run("list_100k_items", func(b *testing.B) {
		client, mr := setupBenchClient(b)
		item := strings.Repeat("v", 64)
		for i := 0; i < 100_000; i++ {
			mr.RPush("biglist", item)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := client.GetValuePreview("biglist"); err != nil {
				b.Fatalf("preview fetch: %v", err)
			}
		}
	})

	b.Run("zset_50k_members", func(b *testing.B) {
		client, mr := setupBenchClient(b)
		for i := 0; i < 50_000; i++ {
			mr.ZAdd("bigzset", float64(i), fmt.Sprintf("member:%06d", i))
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := client.GetValuePreview("bigzset"); err != nil {
				b.Fatalf("preview fetch: %v", err)
			}
		}
	})

	b.Run("string_8mb", func(b *testing.B) {
		client, mr := setupBenchClient(b)
		mr.Set("bigstring", strings.Repeat("x", 8*1024*1024))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := client.GetValuePreview("bigstring"); err != nil {
				b.Fatalf("preview fetch: %v", err)
			}
		}
	})
}

func BenchmarkCompareListFetch(b *testing.B) {
	client, mr := setupBenchClient(b)
	item := strings.Repeat("v", 64)
	for i := 0; i < 100_000; i++ {
		mr.RPush("biglist", item)
	}
	b.Run("GetValue_full", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := client.GetValue("biglist"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("GetValuePreview_bounded", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := client.GetValuePreview("biglist"); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCompareStringFetch(b *testing.B) {
	client, mr := setupBenchClient(b)
	mr.Set("bigstring", strings.Repeat("x", 8*1024*1024))
	b.Run("GetValue_full", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := client.GetValue("bigstring"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("GetValuePreview_bounded", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := client.GetValuePreview("bigstring"); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestGetValuePreview_ProtobufS2(t *testing.T) {
	client, mr := setupTestClient(t)
	var raw []byte
	raw = append(raw, 0x0a, 0x04)
	raw = append(raw, []byte("menu")...)
	compressed := s2.Encode(nil, raw)
	mr.Set("proto:s2", string(compressed))
	v, err := client.GetValuePreview("proto:s2")
	if err != nil {
		t.Fatalf("GetValuePreview: %v", err)
	}
	if v.Type != types.KeyTypeProtobuf {
		t.Fatalf("Type = %q, want protobuf", v.Type)
	}
	if v.DecodedFormat != "s2+protobuf" {
		t.Errorf("DecodedFormat = %q", v.DecodedFormat)
	}
	if !strings.Contains(v.DecodedValue, "menu") {
		t.Errorf("DecodedValue = %q", v.DecodedValue)
	}
}

func TestGetValuePreview_TruncatedBinaryStaysString(t *testing.T) {
	client, mr := setupTestClient(t)
	// Binary blob larger than previewMaxStringBytes that is not valid protobuf.
	bin := make([]byte, previewMaxStringBytes+1024)
	for i := range bin {
		bin[i] = 0xff
	}
	mr.Set("bigbin", string(bin))
	v, err := client.GetValuePreview("bigbin")
	if err != nil {
		t.Fatalf("GetValuePreview: %v", err)
	}
	if v.Type != types.KeyTypeString {
		t.Fatalf("Type = %q, want string (not bitmap) for truncated binary", v.Type)
	}
	if !v.Truncated {
		t.Error("expected Truncated")
	}
}
