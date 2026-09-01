package ui

import (
	"strings"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 500, "500 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"exactly 1 MB", 1024 * 1024, "1.0 MB"},
		{"exactly 1 GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"1023 bytes", 1023, "1023 B"},
		{"large MB", 5 * 1024 * 1024, "5.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestSanitizeBinaryString(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedBinary bool
		checkContains  string // substring the result should contain
	}{
		{"plain ASCII", "hello world", false, "hello world"},
		{"empty string", "", false, ""},
		{"HyperLogLog prefix", "HYLL\x00\x01\x02data", true, "HyperLogLog"},
		{"tabs and newlines preserved", "line1\nline2\ttab", false, "line1\nline2\ttab"},
		{"high non-printable ratio", string([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x10, 0x11}), true, "binary data"},
		{"java serialization excerpt", "\xac\xed\x00\x05sr\x00&com.garmin.engq.model.AdminSettingsDTO", true, `Preview: \xac\xed\x00\x05sr\x00&com.garmin`},
		{"low non-printable below threshold", "abcdefghijklmnopqrst\x01", false, "\\x01"},
		{"low non-printable above threshold", "abc\x01def", true, "binary data"},
		{"multibyte text uses rune ratio", "界界界界界\x01", true, "binary data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, isBinary := sanitizeBinaryString(tt.input)
			if isBinary != tt.expectedBinary {
				t.Errorf("sanitizeBinaryString(%q) isBinary = %v, want %v", tt.input, isBinary, tt.expectedBinary)
			}
			if tt.checkContains != "" && !strings.Contains(result, tt.checkContains) {
				t.Errorf("sanitizeBinaryString(%q) = %q, want to contain %q", tt.input, result, tt.checkContains)
			}
		})
	}
}

func TestSanitizeBinaryStringExcerpt(t *testing.T) {
	input := "\xac\xed\x00\x05\x1b[31m\\\n\r\t" + strings.Repeat("a", 100)
	result, isBinary := sanitizeBinaryString(input)
	if !isBinary {
		t.Fatal("expected binary data")
	}
	if !strings.Contains(result, "(binary data, 113 bytes)\nPreview: \\xac\\xed\\x00\\x05\\x1b[31m\\\\\\n\\r\\t") {
		t.Errorf("unexpected binary summary: %q", result)
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("expected truncated excerpt, got %q", result)
	}
	if strings.ContainsRune(result, '\x1b') {
		t.Errorf("result contains a terminal escape byte: %q", result)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"equal to max", "hello", 5, "hello"},
		{"longer than max", "hello world", 8, "hello..."},
		{"exactly at boundary", "abcdef", 6, "abcdef"},
		{"min truncation", "abcdefgh", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestColorizeProtobuf(t *testing.T) {
	if colorizeProtobuf("") != "" {
		t.Error("empty should stay empty")
	}
	in := "Format: s2+protobuf\nSize: 1 KB\n\n1: \"menu\"\n2: 7\n3: {\n  1: \"cat\"\n  4: 0xab\n  5: <3 bytes>\n}\n… truncated\n(unable to decode)\nnot-a-field\n"
	out := colorizeProtobuf(in)
	if !strings.Contains(out, "menu") || !strings.Contains(out, "Format:") {
		t.Errorf("colorize lost content: %q", out)
	}
	// field line with no space after colon
	if got := colorizeProtobufLine("1:\"x\""); !strings.Contains(got, "x") {
		t.Errorf("no-space colon: %q", got)
	}
	if got := colorizeProtobufLine("  }"); !strings.Contains(got, "}") {
		t.Errorf("brace: %q", got)
	}
	if got := colorizeProtobufLine("↑ 3 more"); !strings.Contains(got, "↑") {
		t.Errorf("hint: %q", got)
	}
	if got := colorizeProtobufLine("12"); got != "12" {
		t.Errorf("non-field passthrough: %q", got)
	}
	if got := colorizeProtobufLine("ab: 1"); got != "ab: 1" {
		t.Errorf("non-numeric field: %q", got)
	}
	if got := colorizeProtobufLine(""); got != "" {
		t.Errorf("empty line: %q", got)
	}
	if got := colorizeProtobufLine("1: "); !strings.Contains(got, "1") {
		t.Errorf("empty value: %q", got)
	}
}

func TestWrapPlainLines(t *testing.T) {
	got := wrapPlainLines([]string{"", "hello-world-extra"}, 8)
	if len(got) < 3 {
		t.Fatalf("expected wraps, got %v", got)
	}
	if got[0] != "" || got[1] != "hello-wo" {
		t.Errorf("unexpected wrap: %v", got)
	}
	// width below floor clamps to 8
	got = wrapPlainLines([]string{"abcdefghij"}, 2)
	if len(got) != 2 || got[0] != "abcdefgh" {
		t.Errorf("narrow clamp wrap got=%v", got)
	}
}

func TestDetailLayoutHelpers(t *testing.T) {
	if w := detailBoxWidth(200); w < 50 || w > 194 {
		t.Errorf("detailBoxWidth(200)=%d", w)
	}
	if w := detailContentWidth(40); w != 36 {
		t.Errorf("content width=%d", w)
	}
	if w := detailContentWidth(10); w != 20 {
		t.Errorf("min content width=%d", w)
	}
	if n := detailMaxVisible(10); n != 5 {
		t.Errorf("min visible=%d", n)
	}
	if n := detailMaxVisible(40); n != 24 {
		t.Errorf("visible=%d", n)
	}
}

func TestPadRightTruncateRunes(t *testing.T) {
	if got := padRight("ab", 5); got != "ab   " {
		t.Errorf("padRight = %q", got)
	}
	if got := padRight("abcdef", 3); got != "abcdef" {
		t.Errorf("padRight no-op = %q", got)
	}
	if got := truncateRunes("hello", 3); got != "he…" {
		t.Errorf("truncateRunes = %q", got)
	}
	if got := truncateRunes("hi", 5); got != "hi" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncateRunes("hello", 0); got != "" {
		t.Errorf("truncate 0 = %q", got)
	}
	if got := truncateRunes("hello", 1); got != "h" {
		t.Errorf("truncate 1 = %q", got)
	}
}

func TestEnsureDetailCursorVisible(t *testing.T) {
	c, s := ensureDetailCursorVisible(0, 0, 0, 10)
	if c != 0 || s != 0 {
		t.Errorf("empty: %d %d", c, s)
	}
	c, s = ensureDetailCursorVisible(-1, 0, 10, 5)
	if c != 0 {
		t.Errorf("neg cursor: %d", c)
	}
	c, s = ensureDetailCursorVisible(99, 0, 10, 5)
	if c != 9 {
		t.Errorf("clamp high: %d", c)
	}
	c, s = ensureDetailCursorVisible(2, 5, 20, 5)
	if s != 2 {
		t.Errorf("scroll up to cursor: s=%d", s)
	}
	c, s = ensureDetailCursorVisible(15, 0, 20, 5)
	if s != 13 {
		t.Errorf("scroll down to cursor: s=%d want 13", s)
	}
	// tiny maxVisible → window floors at 1; negative scroll clamps
	c, s = ensureDetailCursorVisible(0, -5, 3, 1)
	if s != 0 || c != 0 {
		t.Errorf("tiny window: c=%d s=%d", c, s)
	}
	_ = c
}

func TestScrollValueLines(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = string(rune('a' + i%26))
	}
	// fits
	vis, top, bot, sc := scrollValueLines(lines[:3], 0, 10)
	if len(vis) != 3 || top != "" || bot != "" || sc != 0 {
		t.Errorf("fit: vis=%d top=%q bot=%q sc=%d", len(vis), top, bot, sc)
	}
	// overflow from top
	vis, top, bot, sc = scrollValueLines(lines, 0, 5)
	if sc != 0 || top != "" || bot == "" || len(vis) == 0 {
		t.Errorf("top: vis=%d top=%q bot=%q sc=%d", len(vis), top, bot, sc)
	}
	// overflow mid
	vis, top, bot, sc = scrollValueLines(lines, 5, 5)
	if sc != 5 || top == "" || bot == "" {
		t.Errorf("mid: vis=%d top=%q bot=%q sc=%d", len(vis), top, bot, sc)
	}
	// clamp high scroll — should land near EOF with no bottom hint
	vis, top, bot, sc = scrollValueLines(lines, 999, 5)
	if sc >= 20 {
		t.Errorf("scroll not clamped: %d", sc)
	}
	if bot != "" {
		t.Errorf("expected no bottom hint at EOF, got %q (sc=%d vis=%d)", bot, sc, len(vis))
	}
	if top == "" {
		t.Error("expected top hint when scrolled to end")
	}
	// negative scroll
	_, _, _, sc = scrollValueLines(lines, -3, 5)
	if sc != 0 {
		t.Errorf("neg scroll=%d", sc)
	}
	// tiny / zero maxVisible
	vis, _, bot, _ = scrollValueLines(lines, 0, 1)
	if len(vis) < 1 || bot == "" {
		t.Errorf("tiny window vis=%d bot=%q", len(vis), bot)
	}
	vis, _, _, _ = scrollValueLines(lines, 2, 0)
	if len(vis) < 1 {
		t.Error("zero maxVisible should still show a line")
	}
	// single-line reclaim at EOF with top hint
	_, top, bot, sc = scrollValueLines(lines, 19, 2)
	if sc > 19 {
		t.Errorf("eof scroll=%d", sc)
	}
	_ = top
	_ = bot
	_ = vis
}

func TestParseLogEntry(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedLevel string
		expectedMsg   string
		hasTime       bool
	}{
		{
			"valid JSON with all fields",
			`{"time":"2024-01-15T10:30:00.123456789Z","level":"INFO","msg":"server started"}`,
			"INFO",
			"server started",
			true,
		},
		{
			"missing time field",
			`{"level":"ERROR","msg":"connection failed"}`,
			"ERROR",
			"connection failed",
			false,
		},
		{
			"missing level field",
			`{"time":"2024-01-15T10:30:00Z","msg":"hello"}`,
			"",
			"hello",
			true,
		},
		{
			"non-JSON fallback",
			"plain text log line",
			"",
			"plain text log line",
			false,
		},
		{
			"empty JSON object",
			`{}`,
			"",
			"",
			false,
		},
		{
			"level lowercased in source",
			`{"level":"warn","msg":"low disk"}`,
			"WARN",
			"low disk",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := parseLogEntry(tt.input)
			if entry.Level != tt.expectedLevel {
				t.Errorf("Level = %q, want %q", entry.Level, tt.expectedLevel)
			}
			if entry.Msg != tt.expectedMsg {
				t.Errorf("Msg = %q, want %q", entry.Msg, tt.expectedMsg)
			}
			if tt.hasTime && entry.Time == "" {
				t.Error("expected Time to be set")
			}
			if !tt.hasTime && entry.Time != "" {
				t.Errorf("expected empty Time, got %q", entry.Time)
			}
		})
	}
}

func TestParseLogEntry_RFC3339Nano(t *testing.T) {
	input := `{"time":"2024-01-15T10:30:45.123456789Z","level":"INFO","msg":"test"}`
	entry := parseLogEntry(input)
	expected := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC).Format("15:04:05")
	if entry.Time != expected {
		t.Errorf("Time = %q, want %q", entry.Time, expected)
	}
}

func TestFindStringEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		expected int
	}{
		{"simple string", `"hello"`, 1, 6},
		{"escaped quote", `"he\"llo"`, 1, 8},
		{"escaped backslash", `"he\\llo"`, 1, 8},
		{"no closing quote", `"hello`, 1, -1},
		{"empty string", `""`, 1, 1},
		{"escaped backslash before quote", `"he\\\\"`, 1, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findStringEnd(tt.input, tt.start)
			if got != tt.expected {
				t.Errorf("findStringEnd(%q, %d) = %d, want %d", tt.input, tt.start, got, tt.expected)
			}
		})
	}
}

func TestFormatPossibleJSON(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantEmpty  bool
		wantSame   bool // output should match input (passthrough)
		wantBinary bool // expect binary data message
	}{
		{"empty string", "", true, false, false},
		{"whitespace only", "   ", true, false, false},
		{"plain text passthrough", "hello world", false, true, false},
		{"non-JSON passthrough", "not json at all", false, true, false},
		{"valid JSON object", `{"key":"value"}`, false, false, false},
		{"valid JSON array", `[1,2,3]`, false, false, false},
		{"binary data", "HYLL\x00\x01\x02data", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPossibleJSON(tt.input)
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			if tt.wantSame && got != tt.input {
				t.Errorf("expected passthrough %q, got %q", tt.input, got)
			}
			if tt.wantBinary && !strings.Contains(got, "HyperLogLog") {
				t.Errorf("expected binary data message, got %q", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Error("expected non-empty output")
			}
		})
	}
}

func TestFormatPossibleJSON_NoPanic(t *testing.T) {
	// Ensure no panic on various inputs
	inputs := []string{
		"",
		"plain",
		`{"valid": true}`,
		`[1, 2, 3]`,
		`{"nested": {"deep": [1, 2]}}`,
		`invalid json {`,
		`{`,
		`[`,
		string([]byte{0x00, 0x01, 0x02}),
	}

	for _, input := range inputs {
		t.Run("no panic", func(t *testing.T) {
			// Should not panic
			_ = formatPossibleJSON(input)
		})
	}
}

func TestColorizeJSON_LinearPaths(t *testing.T) {
	cases := []string{
		`{"a":1,"b":true,"c":null,"d":false}`,
		`[1,2,"x"]`,
		`{"k":"unterminated`,
		`{"n":-1.5e+2}`,
	}
	for _, c := range cases {
		out := colorizeJSON(c)
		if out == "" {
			t.Errorf("empty colorize for %q", c)
		}
	}
}

// BenchmarkColorizeJSONLarge guards against colorizeJSON regressing to
// super-linear behavior on large JSON arrays of strings.
func BenchmarkColorizeJSONLarge(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("[\n")
	for i := 0; i < 20000; i++ {
		sb.WriteString("  \"element-number-with-some-padding-0123456789\",\n")
	}
	sb.WriteString("  \"last\"\n]")
	s := sb.String()
	b.SetBytes(int64(len(s)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = colorizeJSON(s)
	}
}
